#!/usr/bin/env python3
"""Actual in-memory SQLite SQL READ witness; not TEI or Host conformance.

sqlite3.set_authorizer observes compiled SQL column access, not physical page I/O,
actual row-level reads, host grants, a general SQL parser, or RLS enforcement.
"""
from dataclasses import dataclass
import sqlite3
import unittest


@dataclass(frozen=True)
class Observation:
    rows: tuple
    read_columns: frozenset


class LocalDB:
    def __init__(self):
        self.db = sqlite3.connect(":memory:", cached_statements=0)
        self.db.executescript("""
            CREATE TABLE orders (
                id TEXT PRIMARY KEY, customer_id TEXT NOT NULL,
                amount INTEGER NOT NULL, private_note TEXT
            );
            CREATE TABLE lines (
                id TEXT PRIMARY KEY, order_id TEXT NOT NULL,
                amount INTEGER NOT NULL, sku TEXT NOT NULL
            );
            CREATE TABLE notes (
                id TEXT PRIMARY KEY, line_id TEXT NOT NULL,
                text TEXT NOT NULL, internal_flag INTEGER NOT NULL
            );
            CREATE TABLE payments (
                id TEXT PRIMARY KEY, order_id TEXT NOT NULL, value INTEGER
            );
            CREATE TABLE accounts (
                id TEXT PRIMARY KEY, credit_limit INTEGER NOT NULL,
                internal_risk TEXT
            );
            CREATE TABLE holds (
                id TEXT PRIMARY KEY, order_id TEXT NOT NULL,
                active INTEGER NOT NULL
            );
            INSERT INTO orders VALUES ('A','C1',600,'private-o');
            INSERT INTO lines VALUES ('L1','A',100,'SKU-z'),('L2','A',200,'SKU-a');
            INSERT INTO notes VALUES ('N1','L1','one',7),('N2','L1','two',9);
            INSERT INTO payments VALUES ('P1','A',300);
            INSERT INTO accounts VALUES ('C1',700,'secret-risk');
        """)

    def close(self):
        self.db.close()

    def read(self, sql, args=(), denied=frozenset()):
        reads = set()

        def auth(action, arg1, arg2, db_name, trigger):
            if action == sqlite3.SQLITE_READ:
                reads.add((arg1, arg2))
                if (arg1, arg2) in denied:
                    return sqlite3.SQLITE_DENY
            return sqlite3.SQLITE_OK

        self.db.set_authorizer(auth)
        try:
            rows = tuple(self.db.execute(sql, args).fetchall())
        finally:
            self.db.set_authorizer(None)
        return Observation(rows, frozenset(reads))


DISPLAY_ORDER = "SELECT id,amount FROM orders WHERE id=?"
DISPLAY_LINES = "SELECT id,amount FROM lines WHERE order_id=? ORDER BY id"
DISPLAY_NOTES = ("SELECT id,text FROM notes "
                 "WHERE line_id IN (SELECT id FROM lines WHERE order_id=?) "
                 "ORDER BY id")
CREDIT = ("SELECT o.amount,a.credit_limit FROM orders o "
          "JOIN accounts a ON a.id=o.customer_id WHERE o.id=?")
NO_ACTIVE_HOLD = ("SELECT NOT EXISTS (SELECT 1 FROM holds h "
                  "WHERE h.order_id=? AND h.active=1)")


def display(db):
    parts = [db.read(DISPLAY_ORDER, ("A",)), db.read(DISPLAY_LINES, ("A",)),
             db.read(DISPLAY_NOTES, ("A",))]
    return tuple(part.rows for part in parts), frozenset().union(
        *(part.read_columns for part in parts))


def rule_preview(db, release_result):
    obs = db.read(CREDIT, ("A",))
    assert len(obs.rows) == 1
    amount, limit = obs.rows[0]
    public = ("satisfied" if amount <= limit else "not_satisfied") if release_result else "unavailable"
    return public, obs


class SQLiteReadWitnessTests(unittest.TestCase):
    def setUp(self):
        self.local = LocalDB()

    def tearDown(self):
        self.local.close()

    def test_rw01_explicit_display_does_not_reference_secret_columns(self):
        rows, reads = display(self.local)
        self.assertEqual(rows[0], (("A", 600),))
        self.assertEqual(len(rows[1]), 2)
        self.assertEqual(len(rows[2]), 2)
        self.assertFalse({("orders", "private_note"), ("lines", "sku"),
                          ("notes", "internal_flag"), ("accounts", "credit_limit")} & reads)

    def test_rw02_unmapped_payment_relation_is_not_sql_referenced(self):
        _, reads = display(self.local)
        self.assertFalse(any(table == "payments" for table, _ in reads))

    def test_rw03_post_query_redaction_does_not_undo_sql_read(self):
        selected = self.local.read(DISPLAY_ORDER, ("A",))
        excessive = self.local.read("SELECT * FROM orders WHERE id=?", ("A",))
        sanitized = tuple((r[0], r[2]) for r in excessive.rows)
        self.assertEqual(selected.rows, sanitized)
        self.assertNotIn(("orders", "private_note"), selected.read_columns)
        self.assertIn(("orders", "private_note"), excessive.read_columns)

    def test_rw04_where_on_secret_column_is_a_non_output_dependency(self):
        obs = self.local.read("SELECT id,amount FROM orders WHERE private_note=?",
                              ("private-o",))
        self.assertEqual(obs.rows, (("A", 600),))
        self.assertIn(("orders", "private_note"), obs.read_columns)
        self.assertEqual(len(obs.rows[0]), 2)

    def test_rw05_order_by_on_non_output_column_is_a_dependency(self):
        obs = self.local.read("SELECT id FROM lines WHERE order_id=? ORDER BY sku", ("A",))
        self.assertEqual(obs.rows, (("L2",), ("L1",)))
        self.assertIn(("lines", "sku"), obs.read_columns)

    def test_rw06_hidden_rule_input_has_a_distinct_sql_read_footprint(self):
        public, observed = rule_preview(self.local, release_result=True)
        self.assertEqual(public, "satisfied")
        self.assertIn(("accounts", "credit_limit"), observed.read_columns)
        _, display_reads = display(self.local)
        self.assertNotIn(("accounts", "credit_limit"), display_reads)

    def test_rw07_same_display_different_rule_result(self):
        before, _ = display(self.local)
        self.local.db.execute("UPDATE accounts SET credit_limit=500 WHERE id='C1'")
        after, _ = display(self.local)
        self.assertEqual(before, after)
        self.assertEqual(rule_preview(self.local, True)[0], "not_satisfied")

    def test_rw08_rule_verdict_itself_can_be_withheld(self):
        a = rule_preview(self.local, release_result=False)[0]
        self.local.db.execute("UPDATE accounts SET credit_limit=500 WHERE id='C1'")
        b = rule_preview(self.local, release_result=False)[0]
        self.assertEqual((a, b), ("unavailable", "unavailable"))

    def test_rw09_denied_column_read_aborts_even_if_output_would_hide_it(self):
        with self.assertRaises(sqlite3.DatabaseError):
            self.local.read("SELECT * FROM orders WHERE id=?", ("A",),
                            frozenset({("orders", "private_note")}))
        self.assertEqual(self.local.read(DISPLAY_ORDER, ("A",)).rows, (("A", 600),))

    def test_rw10_denied_rule_input_is_not_a_business_validation_failure(self):
        with self.assertRaises(sqlite3.DatabaseError):
            self.local.read(CREDIT, ("A",), frozenset({("accounts", "credit_limit")}))
        self.assertEqual(rule_preview(self.local, True)[0], "satisfied")

    def test_rw11_negative_dependency_is_read_even_when_set_empty(self):
        before = self.local.read(NO_ACTIVE_HOLD, ("A",))
        self.assertEqual(before.rows, ((1,),))
        self.assertIn(("holds", "active"), before.read_columns)
        self.assertIn(("holds", "order_id"), before.read_columns)
        self.local.db.execute("INSERT INTO holds VALUES ('H1','A',1)")
        after = self.local.read(NO_ACTIVE_HOLD, ("A",))
        self.assertEqual(after.rows, ((0,),))

    def test_rw12_limit_one_is_not_full_collection_evidence(self):
        page = self.local.read("SELECT id FROM lines WHERE order_id=? ORDER BY id LIMIT 1", ("A",))
        all_rows = self.local.read("SELECT id FROM lines WHERE order_id=? ORDER BY id", ("A",))
        self.assertEqual(len(page.rows), 1)
        self.assertEqual(len(all_rows.rows), 2)
        self.assertNotEqual(page.rows, all_rows.rows)

    def test_rw13_all_row_ids_do_not_prove_required_value_observed(self):
        ids = self.local.read("SELECT id FROM lines WHERE order_id=? ORDER BY id", ("A",))
        self.assertEqual(len(ids.rows), 2)
        self.assertNotIn(("lines", "amount"), ids.read_columns)
        valued = self.local.read(DISPLAY_LINES, ("A",))
        self.assertIn(("lines", "amount"), valued.read_columns)

    def test_rw14_same_output_cannot_prove_same_provider_read_plan(self):
        a = self.local.read("SELECT id FROM orders WHERE id=?", ("A",))
        b = self.local.read("SELECT id FROM orders WHERE id=? AND private_note IS NOT NULL", ("A",))
        self.assertEqual(a.rows, b.rows)
        self.assertNotEqual(a.read_columns, b.read_columns)


if __name__ == "__main__":
    unittest.main(verbosity=2)
