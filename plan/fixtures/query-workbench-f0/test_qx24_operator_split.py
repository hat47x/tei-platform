"""QX-24: SQL pushdown versus local materialization of Data Operations.

Isolated SQLite + hand-written Python/JSON references. This proves only
finite fixture behavior, not TEI Query/Provider/Authority or SQL-to-DSL
conformance. The 'tenant' predicate is a simulated trusted constraint,
not an implementation of actual authentication/authorization.
"""

from contextlib import closing
import json
import sqlite3
import unittest


ROWS = [
    ("B0", "B", 0, 1000),
    ("A2", "A", 2, 20),
    ("A1", "A", 1, 10),
    ("A3", "A", 3, None),
    ("B1", "B", 4, 300),
    ("A4", "A", 4, 50),
]
CHILDREN = [
    ("C1", "A1", 100),
    ("C2", "A1", 100),
    ("C3", "A1", 50),
    ("C4", "A2", 200),
]


def database():
    db = sqlite3.connect(":memory:")
    db.executescript("""
        CREATE TABLE entry(id TEXT PRIMARY KEY, tenant TEXT NOT NULL,
                           priority INTEGER NOT NULL, amount INTEGER);
        CREATE TABLE child(id TEXT PRIMARY KEY, parent_id TEXT NOT NULL,
                           amount INTEGER NOT NULL);
    """)
    db.executemany("INSERT INTO entry VALUES(?,?,?,?)", ROWS)
    db.executemany("INSERT INTO child VALUES(?,?,?)", CHILDREN)
    return db


def stable_top(rows, tenant, min_amount, n):
    """Full authorized input; no unknown rows or partial pages assumed."""
    matched = [r for r in rows if r[1] == tenant and r[3] is not None
               and r[3] >= min_amount]
    return [r[0] for r in sorted(matched, key=lambda r: (r[2], r[0]))[:n]]


def sql_sum(values):
    present = [v for v in values if v is not None]
    return sum(present) if present else None


class OperatorSplitOracle(unittest.TestCase):
    def test_01_full_authorized_materialization_can_match_pushdown(self):
        with closing(database()) as db:
            sql = db.execute("""SELECT id FROM entry
                WHERE tenant=? AND amount>=? ORDER BY priority,id LIMIT ?""",
                ("A", 20, 2)).fetchall()
            self.assertEqual([x[0] for x in sql], stable_top(ROWS, "A", 20, 2))
            self.assertEqual(sql, [("A2",), ("A4",)])

    def test_02_limit_before_tenant_scope_loses_valid_row(self):
        with closing(database()) as db:
            correct = db.execute("""SELECT id FROM entry WHERE tenant='A'
                ORDER BY priority,id LIMIT 2""").fetchall()
            wrong = db.execute("""SELECT id FROM
                (SELECT id,tenant FROM entry ORDER BY priority,id LIMIT 2)
                WHERE tenant='A'""").fetchall()
            self.assertEqual(correct, [("A1",), ("A2",)])
            self.assertEqual(wrong, [("A1",)])

    def test_03_matching_output_cannot_prove_minimal_physical_read(self):
        with closing(database()) as db:
            pushed = db.execute("SELECT id FROM entry WHERE tenant='A' ORDER BY id").fetchall()
            all_rows = db.execute("SELECT id,tenant FROM entry ORDER BY id").fetchall()
            locally_scoped = [(id_,) for id_, tenant in all_rows if tenant == "A"]
            self.assertEqual(locally_scoped, pushed)
            self.assertTrue(any(tenant == "B" for _, tenant in all_rows))
            # This fixture demonstrates a larger SQL-level read footprint,
            # not an actual database grant, RLS or unauthorized access.

    def test_04_filter_of_partial_page_is_not_global_top(self):
        with closing(database()) as db:
            full = db.execute("""SELECT id FROM entry WHERE tenant='A' AND amount>=20
                ORDER BY priority,id LIMIT 2""").fetchall()
            partial = db.execute("""SELECT id,amount FROM entry WHERE tenant='A'
                ORDER BY priority,id LIMIT 2""").fetchall()
            local = [(id_,) for id_, amount in partial if amount is not None and amount >= 20]
            self.assertEqual(full, [("A2",), ("A4",)])
            self.assertEqual(local, [("A2",)])

    def test_05_child_page_can_witness_existence_but_not_group_total(self):
        with closing(database()) as db:
            page = db.execute("""SELECT amount FROM child WHERE parent_id='A1'
                ORDER BY id LIMIT 2""").fetchall()
            full = db.execute("SELECT SUM(amount),COUNT(*) FROM child WHERE parent_id='A1'").fetchone()
            self.assertEqual((sum(x[0] for x in page), len(page)), (200, 2))
            self.assertEqual(full, (250, 3))
            self.assertTrue(len(page) >= 1)  # sufficient existence witness
            self.assertFalse(len(page) == full[1])  # full scope not established

    def test_06_absence_on_page_is_not_absence_in_relation(self):
        with closing(database()) as db:
            page = db.execute("SELECT parent_id FROM child ORDER BY id LIMIT 3").fetchall()
            self.assertNotIn(("A2",), page)
            exists = db.execute("SELECT EXISTS(SELECT 1 FROM child WHERE parent_id='A2')").fetchone()[0]
            self.assertEqual(exists, 1)

    def test_07_avg_of_page_avgs_is_not_global_avg(self):
        with closing(database()) as db:
            values = [x[0] for x in db.execute("""SELECT amount FROM entry
                WHERE tenant='A' AND amount IS NOT NULL ORDER BY priority,id""")]
            self.assertEqual(values, [10, 20, 50])
            pages = [values[:2], values[2:]]
            naive = sum(sum(p) / len(p) for p in pages) / len(pages)
            recombined = sum(map(sum, pages)) / sum(map(len, pages))
            db_avg = db.execute("SELECT AVG(amount) FROM entry WHERE tenant='A'").fetchone()[0]
            self.assertEqual(naive, 32.5)
            self.assertAlmostEqual(recombined, db_avg)
            self.assertNotEqual(naive, db_avg)

    def test_08_tenant_scoped_sum_differs_from_world_sum(self):
        with closing(database()) as db:
            scoped = db.execute("SELECT SUM(amount) FROM entry WHERE tenant='A'").fetchone()[0]
            entire = db.execute("SELECT SUM(amount) FROM entry").fetchone()[0]
            self.assertEqual(scoped, sql_sum([r[3] for r in ROWS if r[1] == "A"]))
            self.assertEqual(scoped, 80)
            self.assertEqual(entire, 1380)

    def test_09_missing_null_empty_array_are_different_doc_states(self):
        docs = [{"id": "J1", "children": []},
                {"id": "J2", "children": None},
                {"id": "J3"},
                {"id": "J4", "children": [{"id": "C1"}]}]
        expected = ["array", "null", None, "array"]
        with closing(database()) as db:
            got = [db.execute("SELECT json_type(?, '$.children')",
                              (json.dumps(doc),)).fetchone()[0] for doc in docs]
            self.assertEqual(got, expected)
        # A value with the same Python len cannot imply the same presence:
        self.assertEqual(len(docs[0]["children"]), 0)
        self.assertIsNone(docs[1]["children"])
        self.assertNotIn("children", docs[2])

    def test_10_json_child_occurrence_is_not_parent_entity(self):
        docs = [{"id": "A1", "lines": [{"id": "C1"}, {"id": "C2"}]},
                {"id": "A2", "lines": []}]
        occurrences = [(doc["id"], i, c["id"]) for doc in docs
                       for i, c in enumerate(doc["lines"])]
        self.assertEqual(occurrences, [("A1", 0, "C1"), ("A1", 1, "C2")])
        self.assertEqual(len({p for p, _, _ in occurrences}), 1)
        self.assertEqual(len(occurrences), 2)

    def test_11_same_values_do_not_prove_reusable_observation(self):
        left = {"value": 80, "source_revision": "rev-1", "scope": "A",
                "observed_complete": True, "snapshot": "snap-1"}
        right = {**left, "source_revision": "rev-2"}
        self.assertEqual(left["value"], right["value"])
        self.assertNotEqual(left, right)
        incomplete = {**left, "observed_complete": False}
        self.assertNotEqual(left, incomplete)

    def test_12_limited_positive_satisfies_join_projection_when_key_is_unique(self):
        # With a one-to-one relation and explicit occurrence projection, a
        # local join can match SQL. This is not a universal pushdown law.
        lines = [("A1", "C1"), ("A2", "C2")]
        names = {"C1": "n1", "C2": "n2"}
        python_result = [(p, names[c]) for p, c in lines]
        with closing(sqlite3.connect(":memory:")) as db:
            db.executescript("CREATE TABLE l(p TEXT,c TEXT); CREATE TABLE n(c TEXT,name TEXT);")
            db.executemany("INSERT INTO l VALUES(?,?)", lines)
            db.executemany("INSERT INTO n VALUES(?,?)", names.items())
            sql = db.execute("""SELECT l.p,n.name FROM l JOIN n ON l.c=n.c
                ORDER BY l.p""").fetchall()
            self.assertEqual(sql, python_result)


if __name__ == "__main__":
    unittest.main()
