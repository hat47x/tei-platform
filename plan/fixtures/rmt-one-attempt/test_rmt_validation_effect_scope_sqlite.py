"""Isolated SQLite counterexamples for RMT PA-11..13 (not the TEI Provider).

This fixture tests actual SQL results against independently specified outcomes.
It does not claim old X-TEA Driver parity, TEI Runtime execution, Owner
acceptance, authorization or a validated post-commit causal read.
"""

import sqlite3
import unittest


def universal_positive(rows, *, complete, allow_empty):
    """A small, explicit proof-boundary model, NOT production validation."""
    if any(qty is not None and qty <= 0 for (qty,) in rows):
        return "disproved"
    if not complete or any(qty is None for (qty,) in rows):
        return "unknown"
    if not rows and not allow_empty:
        return "disproved"  # independent Owner policy: at least one line
    return "established"


class RMTValidationEffectScopeSQLite(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
            CREATE TABLE lines(
                tenant TEXT NOT NULL,
                order_no TEXT NOT NULL,
                line_no INTEGER NOT NULL,
                quantity INTEGER,
                PRIMARY KEY(tenant, order_no, line_no)
            );
            CREATE TABLE effect_rows(
                tenant TEXT NOT NULL,
                order_no TEXT NOT NULL,
                operation_id TEXT NOT NULL,
                payload TEXT NOT NULL
            );
            CREATE TABLE receipt(
                operation_id TEXT PRIMARY KEY,
                tenant TEXT NOT NULL,
                order_no TEXT NOT NULL,
                result TEXT NOT NULL
            );
            CREATE TABLE outbox(
                operation_id TEXT NOT NULL,
                event_name TEXT NOT NULL
            );
            INSERT INTO lines VALUES
                ('A','O1',10,2), ('A','O1',20,3),
                ('A','O2',10,2), ('A','O2',20,0),
                ('A','O3',10,NULL),
                ('B','O1',10,0);
            INSERT INTO effect_rows VALUES ('A','OTHER','prior-operation','old');
        """)

    def quantities(self, tenant, order, *, limit=None):
        sql = "SELECT quantity FROM lines WHERE tenant=? AND order_no=? ORDER BY line_no"
        if limit is not None:
            sql += " LIMIT ?"
            return self.db.execute(sql, (tenant, order, limit)).fetchall()
        return self.db.execute(sql, (tenant, order)).fetchall()

    def test_pa11_positive_needs_complete_target_set(self):
        whole = self.quantities("A", "O1")
        first_page = self.quantities("A", "O1", limit=1)
        self.assertEqual(whole, [(2,), (3,)])
        self.assertEqual(first_page, [(2,)])
        self.assertEqual(universal_positive(whole, complete=True, allow_empty=False), "established")
        self.assertEqual(universal_positive(first_page, complete=False, allow_empty=False), "unknown")

    def test_pa11_one_qualified_counterexample_disproves_even_partial(self):
        # Row (A,O2,20) has independently established membership in target L.
        invalid = self.db.execute(
            "SELECT quantity FROM lines WHERE tenant='A' AND order_no='O2' AND line_no=20"
        ).fetchall()
        self.assertEqual(invalid, [(0,)])
        self.assertEqual(universal_positive(invalid, complete=False, allow_empty=False), "disproved")

    def test_pa11_wrong_tenant_counterexample_is_not_about_target(self):
        other_tenant = self.quantities("B", "O1")
        target = self.quantities("A", "O1")
        self.assertEqual(other_tenant, [(0,)])
        self.assertEqual(universal_positive(other_tenant, complete=True, allow_empty=False), "disproved")
        self.assertEqual(universal_positive(target, complete=True, allow_empty=False), "established")

    def test_pa11_empty_policy_and_null_are_independent(self):
        empty = self.quantities("A", "EMPTY")
        self.assertEqual(empty, [])
        self.assertEqual(universal_positive(empty, complete=True, allow_empty=False), "disproved")
        self.assertEqual(universal_positive(empty, complete=True, allow_empty=True), "established")
        self.assertEqual(universal_positive(empty, complete=False, allow_empty=True), "unknown")
        self.assertEqual(universal_positive(self.quantities("A", "O3"), complete=True, allow_empty=False), "unknown")

    def test_pa12_unscoped_count_does_not_prove_this_operation(self):
        global_count = self.db.execute("SELECT COUNT(*) FROM effect_rows").fetchone()[0]
        bound_count = self.db.execute(
            "SELECT COUNT(*) FROM effect_rows WHERE tenant=? AND order_no=? AND operation_id=?",
            ("A", "O1", "submit-A-O1"),
        ).fetchone()[0]
        self.assertGreater(global_count, 0)
        self.assertEqual(bound_count, 0)
        self.db.execute("INSERT INTO effect_rows VALUES ('A','O1','submit-A-O1','new')")
        same_operation = self.db.execute(
            "SELECT COUNT(*) FROM effect_rows WHERE tenant=? AND order_no=? AND operation_id=?",
            ("A", "O1", "submit-A-O1"),
        ).fetchone()[0]
        self.assertEqual(same_operation, 1)
        # Even this positive SQL predicate is not a causal read/Owner receipt.

    def test_pa12_operation_id_does_not_override_target_identity(self):
        self.db.execute("INSERT INTO effect_rows VALUES ('B','O1','submit-A-O1','wrong-tenant')")
        by_operation_only = self.db.execute(
            "SELECT COUNT(*) FROM effect_rows WHERE operation_id='submit-A-O1'"
        ).fetchone()[0]
        by_full_target = self.db.execute(
            "SELECT COUNT(*) FROM effect_rows WHERE tenant='A' AND order_no='O1' AND operation_id='submit-A-O1'"
        ).fetchone()[0]
        self.assertEqual(by_operation_only, 1)
        self.assertEqual(by_full_target, 0)

    def test_pa13_preview_change_does_not_emit_committed_event(self):
        old_value, proposed = "draft", "submitted"
        self.assertTrue(old_value != proposed)  # Script preview/oldValue comparison
        before = self.db.execute(
            "SELECT COUNT(*) FROM outbox WHERE operation_id='submit-A-O1'"
        ).fetchone()[0]
        self.assertEqual(before, 0)
        # A test-owned transaction simulates an applied change + outbox.
        with self.db:
            self.db.execute("INSERT INTO receipt VALUES ('submit-A-O1','A','O1','committed')")
            self.db.execute("INSERT INTO outbox VALUES ('submit-A-O1','submitted')")
        after = self.db.execute(
            "SELECT COUNT(*) FROM outbox WHERE operation_id='submit-A-O1'"
        ).fetchone()[0]
        self.assertEqual(after, 1)
        # Outbox row is not proof that an external job actually ran.

    def test_pa13_ack_loss_is_not_negative_receipt(self):
        with self.db:
            self.db.execute("INSERT INTO receipt VALUES ('submit-A-O1','A','O1','committed')")
        client_ack = "unknown"  # independent simulated transport outcome
        actual = self.db.execute(
            "SELECT result FROM receipt WHERE operation_id=? AND tenant=? AND order_no=?",
            ("submit-A-O1", "A", "O1"),
        ).fetchone()
        self.assertEqual(client_ack, "unknown")
        self.assertEqual(actual, ("committed",))
        self.assertIsNone(self.db.execute(
            "SELECT result FROM receipt WHERE operation_id='new-operation'"
        ).fetchone())


if __name__ == "__main__":
    unittest.main(verbosity=2)
