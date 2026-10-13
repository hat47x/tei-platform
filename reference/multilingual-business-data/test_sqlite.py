"""SQLite D2 reference: real storage and cross-language publication selection."""
import pathlib
import sqlite3
import tempfile
import unittest

SCHEMA = (pathlib.Path(__file__).parent / "testdata" / "schema.sql").read_text(encoding="utf-8")
BASE = ("store-a", "product:P-100", "catalog", "attribute", "catalog.product.description")


class RelationalSQLite(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = pathlib.Path(self.tmp.name) / "values.sqlite"
        self.db = sqlite3.connect(self.path)
        self.addCleanup(self.db.close)
        # SQLite foreign keys must be enabled on each connection.
        self.db.execute("PRAGMA foreign_keys=ON")
        self.assertEqual(self.db.execute("PRAGMA foreign_keys").fetchone(), (1,))
        self.db.executescript(SCHEMA)
        self.db.commit()

    def revision(self, locale, rev, body, origin, review="approved", source=None, tenant="store-a"):
        self.db.execute(
            "INSERT INTO tei_multilingual_revision VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)",
            (tenant, *BASE[1:], locale, "catalog", rev, body, origin, review,
             source[0] if source else None, source[1] if source else None),
        )

    def publication(self, locale, rev, tenant="store-a"):
        self.db.execute(
            "INSERT INTO tei_multilingual_publication VALUES (?,?,?,?,?,?,?,?)",
            (tenant, *BASE[1:], locale, "catalog", rev),
        )

    def current(self, locale, tenant="store-a", term=None):
        query = ("SELECT revision, body FROM tei_multilingual_current "
                 "WHERE tenant_id=? AND record_id=? AND locale=?")
        parameters = [tenant, BASE[1], locale]
        if term is not None:
            # A substring smoke test; not language-specific full-text search.
            query += " AND instr(body, ?) > 0"
            parameters.append(term)
        return self.db.execute(query, parameters).fetchall()

    def populate(self):
        self.revision("ja-JP", "ja-1", "安全に保管してください。", "original")
        self.revision("en-US", "en-1", "Store safely.", "translation", source=("ja-JP", "ja-1"))
        self.publication("ja-JP", "ja-1")
        self.publication("en-US", "en-1")
        self.db.commit()

    def test_durable_reopen_and_search(self):
        self.populate()
        with sqlite3.connect(self.path) as reopened:
            reopened.execute("PRAGMA foreign_keys=ON")
            row = reopened.execute(
                "SELECT body FROM tei_multilingual_current "
                "WHERE tenant_id=? AND record_id=? AND locale=?",
                ("store-a", BASE[1], "en-US"),
            ).fetchone()
            self.assertEqual(row, ("Store safely.",))
        self.assertEqual(self.current("ja-JP", term="保管"), [("ja-1", "安全に保管してください。")])
        self.assertEqual(self.current("en-US", term="保管"), [])
        self.assertEqual(self.current("en-US", term="Store"), [("en-1", "Store safely.")])
        self.assertEqual(self.current("en-GB"), [])  # No implicit fallback.
        self.assertEqual(self.current("en-US", tenant="store-b"), [])

    def test_stale_translation_hidden_without_erasing_history(self):
        self.populate()
        self.revision("ja-JP", "ja-2", "冷暗所に保管してください。", "original")
        self.db.execute(
            "UPDATE tei_multilingual_publication SET revision='ja-2' "
            "WHERE tenant_id='store-a' AND locale='ja-JP'"
        )
        self.db.commit()
        self.assertEqual(self.current("ja-JP"), [("ja-2", "冷暗所に保管してください。")])
        self.assertEqual(self.current("en-US"), [])
        self.assertEqual(
            self.db.execute("SELECT body FROM tei_multilingual_revision WHERE locale='en-US'").fetchall(),
            [("Store safely.",)],
        )
        self.revision("en-US", "en-2", "Keep in a cool, dark place.", "translation", source=("ja-JP", "ja-2"))
        self.db.execute(
            "UPDATE tei_multilingual_publication SET revision='en-2' "
            "WHERE tenant_id='store-a' AND locale='en-US'"
        )
        self.db.commit()
        self.assertEqual(self.current("en-US"), [("en-2", "Keep in a cool, dark place.")])

    def test_candidate_and_independent_text(self):
        self.populate()
        self.revision(
            "en-US", "en-2", "Unapproved AI proposal", "translation",
            review="candidate", source=("ja-JP", "ja-1"),
        )
        self.assertEqual(self.current("en-US"), [("en-1", "Store safely.")])
        # Directly moving the pointer cannot make a candidate visible.
        self.db.execute("UPDATE tei_multilingual_publication SET revision='en-2' WHERE locale='en-US'")
        self.assertEqual(self.current("en-US"), [])
        self.revision("en-US", "en-independent", "Binding English agreement", "independent")
        self.db.execute(
            "UPDATE tei_multilingual_publication SET revision='en-independent' WHERE locale='en-US'"
        )
        self.revision("ja-JP", "ja-2", "日本語改訂", "independent")
        self.db.execute("UPDATE tei_multilingual_publication SET revision='ja-2' WHERE locale='ja-JP'")
        self.assertEqual(self.current("en-US"), [("en-independent", "Binding English agreement")])

    def test_atomic_target_compare_and_swap(self):
        self.populate()
        self.revision("en-US", "en-2", "Revised text", "translation", source=("ja-JP", "ja-1"))
        self.db.commit()
        self.db.execute("BEGIN IMMEDIATE")
        # A SQLite write transaction must also verify the current source
        # revision before making the optimistic target update.
        original = self.db.execute(
            "SELECT revision FROM tei_multilingual_publication WHERE tenant_id=? AND locale=?",
            ("store-a", "ja-JP"),
        ).fetchone()
        self.assertEqual(original, ("ja-1",))
        changed = self.db.execute(
            "UPDATE tei_multilingual_publication SET revision=? "
            "WHERE tenant_id=? AND record_id=? AND locale=? AND revision=?",
            ("en-2", "store-a", BASE[1], "en-US", "en-1"),
        )
        self.assertEqual(changed.rowcount, 1)
        self.db.commit()
        stale = self.db.execute(
            "UPDATE tei_multilingual_publication SET revision=? "
            "WHERE tenant_id=? AND record_id=? AND locale=? AND revision=?",
            ("en-1", "store-a", BASE[1], "en-US", "en-1"),
        )
        self.assertEqual(stale.rowcount, 0)
        self.db.commit()
        self.assertEqual(self.current("en-US"), [("en-2", "Revised text")])

    def test_composite_foreign_keys_and_rejected_translation(self):
        self.populate()
        with self.assertRaises(sqlite3.IntegrityError):
            self.revision("fr-FR", "fr-1", "Bonjour", "translation", source=("ja-JP", "missing"))
        with self.assertRaises(sqlite3.IntegrityError):
            self.revision("fr-FR", "fr-2", "Bonjour", "translation", source=("fr-FR", "fr-2"))
        with self.assertRaises(sqlite3.IntegrityError):
            self.db.execute(
                "INSERT INTO tei_multilingual_publication VALUES (?,?,?,?,?,?,?,?)",
                (*BASE, "fr-FR", "catalog", "missing"),
            )
        self.assertEqual(self.current("en-US"), [("en-1", "Store safely.")])


if __name__ == "__main__":
    unittest.main()
