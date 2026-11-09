"""QX-32: independently observed SQLite route/policy Provider reads.

Synthetic two-file SQLite experiment. It does not test TEI Provider APIs,
real distributed snapshots, RLS, external Host, business acceptance or egress.
"""

import os
import sqlite3
import tempfile
import unittest


class CrossProviderReadCohort(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qx32_")
        self.route_path = os.path.join(self.temp.name, "route.sqlite")
        self.policy_path = os.path.join(self.temp.name, "policy.sqlite")
        self.route_writer = sqlite3.connect(self.route_path)
        self.policy_writer = sqlite3.connect(self.policy_path)
        for connection in (self.route_writer, self.policy_writer):
            self.assertEqual(connection.execute("PRAGMA journal_mode=WAL").fetchone()[0], "wal")
        self.route_writer.executescript("""
            CREATE TABLE routing(tenant TEXT, order_id TEXT, account_id TEXT,
                PRIMARY KEY(tenant,order_id));
            CREATE TABLE revision(n INTEGER);
            INSERT INTO revision VALUES (1);
            INSERT INTO routing VALUES ('A','O1','A1');
            INSERT INTO routing VALUES ('A','O2','A2');
        """)
        self.policy_writer.executescript("""
            CREATE TABLE accounts(tenant TEXT, account_id TEXT, ceiling INTEGER,
                PRIMARY KEY(tenant,account_id));
            CREATE TABLE holds(tenant TEXT, order_id TEXT, active INTEGER);
            CREATE TABLE revision(n INTEGER);
            INSERT INTO revision VALUES (1);
            INSERT INTO accounts VALUES ('A','A1',500);
            INSERT INTO accounts VALUES ('A','A2',500);
            INSERT INTO accounts VALUES ('B','A1',1000);
        """)
        self.route_writer.commit()
        self.policy_writer.commit()
        self.route_reader = sqlite3.connect(self.route_path)
        self.policy_reader = sqlite3.connect(self.policy_path)

    def tearDown(self):
        for c in (self.route_reader, self.route_writer,
                  self.policy_reader, self.policy_writer):
            c.close()
        self.temp.cleanup()

    def route(self, tenant="A", order="O1", connection=None):
        c = connection or self.route_reader
        return c.execute("SELECT account_id FROM routing WHERE tenant=? AND order_id=?",
                         (tenant, order)).fetchone()[0]

    def ceiling(self, account, tenant="A", connection=None):
        c = connection or self.policy_reader
        row = c.execute("SELECT ceiling FROM accounts WHERE tenant=? AND account_id=?",
                        (tenant, account)).fetchone()
        return None if row is None else row[0]

    def eligible(self, account, tenant="A"):
        ceiling = self.ceiling(account, tenant)
        return ceiling is not None and 600 <= ceiling

    def current(self):
        return self.eligible(self.route()) and self.active_holds() == 0

    def active_holds(self, tenant="A", order="O1", connection=None):
        c = connection or self.policy_reader
        return c.execute("SELECT COUNT(*) FROM holds WHERE tenant=? AND order_id=? AND active=1",
                         (tenant, order)).fetchone()[0]

    def rev(self, connection):
        return connection.execute("SELECT n FROM revision").fetchone()[0]

    def update_route(self, account="A2", order="O1"):
        self.route_writer.execute("UPDATE routing SET account_id=? WHERE tenant='A' AND order_id=?",
                                  (account, order))
        self.route_writer.execute("UPDATE revision SET n=n+1")
        self.route_writer.commit()

    def update_limit(self, account="A1", value=700):
        self.policy_writer.execute("UPDATE accounts SET ceiling=? WHERE tenant='A' AND account_id=?",
                                   (value, account))
        self.policy_writer.execute("UPDATE revision SET n=n+1")
        self.policy_writer.commit()

    def add_hold(self, active=1):
        self.policy_writer.execute("INSERT INTO holds VALUES('A','O1',?)", (active,))
        self.policy_writer.execute("UPDATE revision SET n=n+1")
        self.policy_writer.commit()

    def test_01_baseline_same_world_denial(self):
        self.assertEqual((self.route(), self.ceiling('A1'), self.current()), ('A1', 500, False))

    def test_02_fractured_read_false_positive_has_no_global_world(self):
        old_route = self.route()  # t0: A1/500 -> denied
        self.assertFalse(self.current())
        self.update_route('A2')  # t1: A2/500 -> denied
        self.assertFalse(self.current())
        self.update_limit('A1', 700)  # t2: A2/500; A1/700 -> denied
        self.assertFalse(self.current())
        self.assertTrue(self.eligible(old_route))  # A1/700 never coexisted as route+limit
        self.assertEqual(self.route(), 'A2')

    def test_03_fractured_read_false_negative_has_no_global_world(self):
        self.update_limit('A1', 700)
        self.update_limit('A2', 700)
        old_route = self.route()  # t0: A1/700 -> allowed
        self.assertTrue(self.current())
        self.update_route('A2')  # t1: A2/700 -> allowed
        self.assertTrue(self.current())
        self.update_limit('A1', 500)  # t2: A2/700 -> allowed
        self.assertTrue(self.current())
        self.assertFalse(self.eligible(old_route))

    def test_04_route_revision_fence_detects_change(self):
        old = self.rev(self.route_reader)
        old_route = self.route()
        self.update_route()
        self.update_limit()
        self.assertEqual((old_route, old, self.route(), self.rev(self.route_reader)),
                         ('A1', 1, 'A2', 2))

    def test_05_local_read_transaction_does_not_give_cross_db_snapshot(self):
        self.route_reader.execute('BEGIN')
        try:
            self.assertEqual(self.route(), 'A1')
            self.update_route('A2')
            self.update_limit('A1', 700)
            self.assertEqual(self.route(), 'A1')  # pinned local WAL snapshot
            self.assertTrue(self.eligible(self.route()))  # policy read is at t2
        finally:
            self.route_reader.rollback()
        self.assertEqual(self.route(), 'A2')
        self.assertFalse(self.current())

    def test_06_shared_opaque_label_is_not_a_snapshot_guarantee(self):
        token = 'cohort-1'
        before = (token, self.rev(self.route_reader), self.rev(self.policy_reader))
        self.update_route()
        self.update_limit()
        after = (token, self.rev(self.route_reader), self.rev(self.policy_reader))
        self.assertEqual(before[0], after[0])
        self.assertNotEqual(before[1:], after[1:])

    def test_07_stale_negative_hold_read_yields_false_allow(self):
        self.update_limit('A1', 700)
        zero_holds = self.active_holds()
        self.assertEqual(zero_holds, 0)
        self.add_hold(active=1)
        self.assertTrue(self.eligible(self.route()) and zero_holds == 0)
        self.assertFalse(self.current())

    def test_08_old_positive_hold_observation_cannot_deny_after_removal(self):
        self.update_limit('A1', 700)
        self.add_hold(active=1)
        old_count = self.active_holds()
        self.policy_writer.execute('DELETE FROM holds WHERE tenant="A" AND order_id="O1"')
        self.policy_writer.execute('UPDATE revision SET n=n+1')
        self.policy_writer.commit()
        self.assertEqual(old_count, 1)
        self.assertTrue(self.current())

    def test_09_tenant_qualified_binding_distinguishes_same_local_account_id(self):
        self.assertEqual(self.route('A', 'O1'), 'A1')
        self.assertEqual(self.ceiling('A1', 'A'), 500)
        self.assertEqual(self.ceiling('A1', 'B'), 1000)
        self.assertFalse(self.eligible(self.route('A', 'O1'), 'A'))
        self.assertTrue(self.eligible(self.route('A', 'O1'), 'B'))  # bad binding

    def test_10_unrelated_path_edit_does_not_change_current_claim(self):
        baseline = self.current()
        self.update_route('A1', order='O2')
        self.assertEqual(self.rev(self.route_reader), 2)
        self.assertEqual(self.route('A', 'O1'), 'A1')
        self.assertEqual(self.current(), baseline)

    def test_11_partial_hold_page_zero_is_not_global_not_exists(self):
        self.update_limit('A1', 700)
        self.add_hold(active=1)
        partial = self.policy_reader.execute(
            "SELECT active FROM holds WHERE tenant='A' AND order_id='O1' LIMIT 0").fetchall()
        self.assertEqual(partial, [])
        self.assertEqual(self.active_holds(), 1)
        self.assertTrue(self.eligible(self.route()) and not partial)  # false local conclusion
        self.assertFalse(self.current())

    def test_12_provider_error_is_not_zero_rows(self):
        with self.assertRaises(sqlite3.OperationalError):
            self.policy_reader.execute('SELECT active FROM no_holds_table').fetchall()
        self.add_hold(active=1)
        self.assertGreater(self.active_holds(), 0)

    def test_13_pre_and_post_versions_are_not_atomic_read_evidence(self):
        pre_route = self.rev(self.route_reader)
        pre_policy = self.rev(self.policy_reader)
        old_route = self.route()
        self.update_route()
        self.update_limit()
        after_route = self.rev(self.route_reader)
        after_policy = self.rev(self.policy_reader)
        self.assertEqual((pre_route, pre_policy), (1, 1))
        self.assertEqual((after_route, after_policy), (2, 2))
        self.assertTrue(self.eligible(old_route))
        self.assertFalse(self.current())
        self.assertNotEqual((pre_route, pre_policy), (after_route, after_policy))


if __name__ == '__main__':
    unittest.main()
