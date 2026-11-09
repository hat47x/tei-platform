"""QX-31 SQL-only counterexamples: dynamic relation path and negative membership.

SQLite in-memory oracle, no TEI Worker/Function Context/Provider, no live
change feed. SQL authorizer reports prepare-time column references only.
"""
import sqlite3
import unittest


RULE = """SELECT o.id FROM orders o
JOIN customer c ON c.id = o.customer_ref
JOIN account a ON a.id = c.account_ref
WHERE o.id = 'O1' AND o.amount <= a.credit_limit
AND NOT EXISTS (
  SELECT 1 FROM holds h WHERE h.order_id = o.id AND h.active = 1
)"""


def make_db():
    db = sqlite3.connect(':memory:')
    db.executescript('''
    CREATE TABLE orders(id TEXT PRIMARY KEY, customer_ref TEXT, amount INT);
    CREATE TABLE customer(id TEXT PRIMARY KEY, account_ref TEXT);
    CREATE TABLE account(id TEXT PRIMARY KEY, credit_limit INT);
    CREATE TABLE holds(id TEXT PRIMARY KEY, order_id TEXT, active INT);
    INSERT INTO orders VALUES ('O1','C1',600),('O2','C2',30);
    INSERT INTO customer VALUES ('C1','A1'),('C2','A2');
    INSERT INTO account VALUES ('A1',700),('A2',500);
    ''')
    return db


def rows(db, sql=RULE):
    return db.execute(sql).fetchall()


def source_witness(db):
    return rows(db, '''SELECT o.customer_ref,c.account_ref,a.credit_limit
    FROM orders o JOIN customer c ON c.id=o.customer_ref
    JOIN account a ON a.id=c.account_ref WHERE o.id='O1' ''')[0]


class RelationPathNegativeDependency(unittest.TestCase):
    def setUp(self):
        self.db = make_db()

    def tearDown(self):
        self.db.close()

    def test_01_complete_no_hold_and_credit_limit_allows_o1(self):
        self.assertEqual(rows(self.db), [('O1',)])
        self.assertEqual(source_witness(self.db), ('C1', 'A1', 700))

    def test_02_order_to_customer_rebind_without_display_amount_change(self):
        old_amount = rows(self.db, "SELECT amount FROM orders WHERE id='O1'")
        self.db.execute("UPDATE orders SET customer_ref='C2' WHERE id='O1'")
        self.assertEqual(rows(self.db, "SELECT amount FROM orders WHERE id='O1'"), old_amount)
        self.assertEqual(source_witness(self.db), ('C2', 'A2', 500))
        self.assertEqual(rows(self.db), [])

    def test_03_customer_to_account_rebind_without_account_value_change(self):
        self.db.execute("UPDATE customer SET account_ref='A2' WHERE id='C1'")
        self.assertEqual(source_witness(self.db), ('C1', 'A2', 500))
        self.assertEqual(rows(self.db), [])
        self.assertEqual(rows(self.db, "SELECT credit_limit FROM account WHERE id='A1'"), [(700,)])

    def test_04_inserting_first_active_hold_retracts_not_exists(self):
        self.assertEqual(rows(self.db), [('O1',)])
        self.db.execute("INSERT INTO holds VALUES ('H1','O1',1)")
        self.assertEqual(rows(self.db), [])

    def test_05_inactive_hold_is_not_a_match_for_scoped_predicate(self):
        self.db.execute("INSERT INTO holds VALUES ('H1','O1',0)")
        self.assertEqual(rows(self.db), [('O1',)])
        self.db.execute("UPDATE holds SET active=1 WHERE id='H1'")
        self.assertEqual(rows(self.db), [])

    def test_06_unrelated_order_hold_does_not_remove_o1_in_this_world(self):
        self.db.execute("INSERT INTO holds VALUES ('H2','O2',1)")
        self.assertEqual(rows(self.db), [('O1',)])

    def test_07_page_of_unrelated_holds_does_not_prove_o1_absence(self):
        self.db.execute("INSERT INTO holds VALUES ('H0','O2',1)")
        self.db.execute("INSERT INTO holds VALUES ('H1','O1',1)")
        # Deliberately wrong inference from an unscoped first page.
        first_page = rows(self.db, "SELECT id,order_id FROM holds ORDER BY id LIMIT 1")
        self.assertEqual(first_page, [('H0', 'O2')])
        self.assertEqual(rows(self.db), [])
        self.assertEqual(rows(self.db,
            "SELECT id FROM holds WHERE order_id='O1' AND active=1"), [('H1',)])

    def test_08_membership_aba_same_current_result_not_no_changes(self):
        initial = rows(self.db)
        self.db.execute("INSERT INTO holds VALUES ('H1','O1',1)")
        during = rows(self.db)
        self.db.execute("DELETE FROM holds WHERE id='H1'")
        final = rows(self.db)
        self.assertEqual(initial, final)
        self.assertEqual(during, [])
        self.assertEqual(initial, [('O1',)])
        # Only the known two writes witness intervening change; current
        # result equality alone would not establish this fact.

    def test_09_rebind_to_equal_credit_limit_same_result_different_path(self):
        before, witness_before = rows(self.db), source_witness(self.db)
        self.db.execute("UPDATE account SET credit_limit=700 WHERE id='A2'")
        self.db.execute("UPDATE orders SET customer_ref='C2' WHERE id='O1'")
        after, witness_after = rows(self.db), source_witness(self.db)
        self.assertEqual(before, after)
        self.assertNotEqual(witness_before, witness_after)
        self.assertEqual(witness_after, ('C2', 'A2', 700))

    def test_10_null_limit_is_unknown_not_pass(self):
        self.db.execute("UPDATE account SET credit_limit=NULL WHERE id='A1'")
        self.assertEqual(rows(self.db), [])
        self.assertEqual(source_witness(self.db), ('C1', 'A1', None))
        self.assertIsNone(rows(self.db, "SELECT 600<=NULL")[0][0])

    def test_11_ignoring_scope_changes_negative_dependency(self):
        self.db.execute("INSERT INTO holds VALUES ('H2','O2',1)")
        self.assertEqual(rows(self.db), [('O1',)])
        bad = RULE.replace('h.order_id = o.id AND h.active = 1', 'h.active = 1')
        self.assertEqual(rows(self.db, bad), [])

    def test_12_authorizer_shows_relation_link_and_negative_input_references(self):
        referenced = set()
        def auth(op, table, column, db_name, origin):
            if op == sqlite3.SQLITE_READ:
                referenced.add((table, column))
            return sqlite3.SQLITE_OK
        self.db.set_authorizer(auth)
        try:
            self.assertEqual(rows(self.db), [('O1',)])
        finally:
            self.db.set_authorizer(None)
        for item in [('orders','customer_ref'), ('orders','amount'),
                     ('customer','account_ref'), ('account','credit_limit'),
                     ('holds','order_id'), ('holds','active')]:
            self.assertIn(item, referenced)


if __name__ == '__main__':
    unittest.main()
