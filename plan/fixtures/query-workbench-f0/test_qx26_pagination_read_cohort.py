"""QX-26 isolated SQLite oracle: pagination and cross-page read cohorts.

Tests compare actual SQLite WAL transactions, offset/keyset pages and explicit
source/scope checks. No TEI, REST provider, Host authorization or other RDB
transaction-isolation behavior is exercised. Fixture labels are not authority.
"""
import os
import sqlite3
import tempfile
import unittest

BASE = [('A1', 'A', 1, 10), ('A2', 'A', 2, 20),
        ('A3', 'A', 3, 30), ('A4', 'A', 4, 40),
        ('B1', 'B', 0, 1000)]


def page(conn, tenant='A', limit=2, offset=0):
    return conn.execute('''SELECT id,priority,amount FROM item WHERE tenant=?
                           ORDER BY priority,id LIMIT ? OFFSET ?''',
                        (tenant, limit, offset)).fetchall()


def keyset_page(conn, after=None, tenant='A', limit=2):
    if after is None:
        return page(conn, tenant, limit)
    return conn.execute('''SELECT id,priority,amount FROM item
                           WHERE tenant=? AND (priority,id)>(?,?)
                           ORDER BY priority,id LIMIT ?''',
                        (tenant, after[0], after[1], limit)).fetchall()


def cursor_of(row):
    return row[1], row[0]


def ids(rows):
    return [r[0] for r in rows]


class ReadCohortOracle(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='qx26_')
        path = os.path.join(self.tmp.name, 'read.sqlite')
        self.reader = sqlite3.connect(path, isolation_level=None, timeout=5)
        self.writer = sqlite3.connect(path, isolation_level=None, timeout=5)
        self.reader.execute('PRAGMA journal_mode=WAL')
        self.writer.execute('PRAGMA journal_mode=WAL')
        self.reader.execute('CREATE TABLE item (id TEXT PRIMARY KEY, tenant TEXT, '
                            'priority INTEGER NOT NULL, amount INTEGER)')
        self.reader.executemany('INSERT INTO item VALUES (?,?,?,?)', BASE)

    def tearDown(self):
        self.reader.close()
        self.writer.close()
        self.tmp.cleanup()

    def test_01_offset_stable_population_positive(self):
        self.assertEqual(ids(page(self.reader, offset=0) + page(self.reader, offset=2)),
                         ['A1', 'A2', 'A3', 'A4'])

    def test_02_offset_insert_before_cursor_duplicates_and_misses(self):
        first = page(self.reader)
        self.writer.execute("INSERT INTO item VALUES ('A0','A',0,5)")
        second = page(self.reader, offset=2)
        self.assertEqual(ids(first + second), ['A1', 'A2', 'A2', 'A3'])
        self.assertNotEqual(ids(first + second), ids(page(self.reader, limit=20)))

    def test_03_offset_delete_before_cursor_skips_unread_row(self):
        first = page(self.reader)
        self.writer.execute("DELETE FROM item WHERE id='A1'")
        second = page(self.reader, offset=2)
        self.assertEqual(ids(first + second), ['A1', 'A2', 'A4'])
        self.assertNotIn('A3', ids(first + second))

    def test_04_keyset_insert_before_cursor_is_not_current_complete(self):
        first = keyset_page(self.reader)
        self.writer.execute("INSERT INTO item VALUES ('A0','A',0,5)")
        second = keyset_page(self.reader, cursor_of(first[-1]))
        self.assertEqual(ids(first + second), ['A1', 'A2', 'A3', 'A4'])
        self.assertEqual(ids(page(self.reader, limit=20)),
                         ['A0', 'A1', 'A2', 'A3', 'A4'])

    def test_05_keyset_torn_values_and_new_rows_not_single_snapshot(self):
        first = keyset_page(self.reader)
        self.writer.execute("UPDATE item SET amount=100 WHERE id='A1'")
        self.writer.execute("INSERT INTO item VALUES ('A25','A',2,25)")
        second = keyset_page(self.reader, cursor_of(first[-1]), limit=10)
        self.assertEqual(ids(first + second), ['A1', 'A2', 'A25', 'A3', 'A4'])
        self.assertEqual(first[0][2], 10)
        self.assertEqual(page(self.reader, limit=10)[0][2], 100)
        # Its row membership corresponds to the new world, but its A1 value
        # comes from the old world: this cannot be one SQLite snapshot.
        self.assertNotEqual(first + second, page(self.reader, limit=20))

    def test_06_sqlite_wal_read_transaction_pins_old_snapshot(self):
        self.reader.execute('BEGIN')
        try:
            first = page(self.reader)
            self.writer.execute("INSERT INTO item VALUES ('A0','A',0,5)")
            self.writer.execute("UPDATE item SET amount=100 WHERE id='A1'")
            second = page(self.reader, offset=2)
            self.assertEqual(ids(first + second), ['A1', 'A2', 'A3', 'A4'])
            self.assertEqual(first[0][2], 10)
            self.assertEqual(self.reader.execute("SELECT COUNT(*) FROM item WHERE tenant='A'").fetchone(), (4,))
        finally:
            self.reader.execute('ROLLBACK')
        self.assertEqual(ids(page(self.reader, limit=10)), ['A0','A1','A2','A3','A4'])

    def test_07_keyset_requires_total_order_tie_breaker(self):
        self.reader.execute("UPDATE item SET priority=1 WHERE id='A2'")
        first = page(self.reader, limit=1)[0]
        bad = self.reader.execute('''SELECT id FROM item WHERE tenant='A'
                       AND priority>? ORDER BY priority,id LIMIT 10''',
                       (first[1],)).fetchall()
        good = keyset_page(self.reader, cursor_of(first), limit=10)
        self.assertEqual(ids(good), ['A2', 'A3', 'A4'])
        self.assertEqual([x[0] for x in bad], ['A3', 'A4'])

    def test_08_cursor_cannot_be_reused_across_scope_without_requalification(self):
        first = keyset_page(self.reader, tenant='A')
        token = {'scope': 'A', 'sort': ('priority','id'),
                 'after': cursor_of(first[-1]), 'version': 'fixture-rev-1'}
        self.assertEqual(keyset_page(self.reader, after=token['after'], tenant='B'), [])
        self.assertEqual(ids(page(self.reader, tenant='B')), ['B1'])
        self.assertNotEqual(token['scope'], 'B')
        # Deliberately not a real token validator: issuer/consumer must bind
        # the effective scope and selected source; a label alone cannot do so.

    def test_09_changed_filter_invalidates_resume_qualification(self):
        first = self.reader.execute('''SELECT id,priority FROM item
            WHERE tenant='A' AND amount>=20 ORDER BY priority,id LIMIT 1''').fetchall()
        self.assertEqual(first, [('A2',2)])
        resumed = self.reader.execute('''SELECT id FROM item WHERE tenant='A'
            AND amount>=0 AND (priority,id)>(?,?) ORDER BY priority,id''',
            (first[0][1], first[0][0])).fetchall()
        actual = self.reader.execute('''SELECT id FROM item WHERE tenant='A'
            AND amount>=0 ORDER BY priority,id''').fetchall()
        self.assertNotEqual([('A2',)] + resumed, actual)
        self.assertEqual(actual[0], ('A1',))

    def test_10_page_count_equal_old_total_does_not_prove_new_complete(self):
        first = page(self.reader)
        old_total = self.reader.execute("SELECT COUNT(*) FROM item WHERE tenant='A'").fetchone()[0]
        self.writer.execute("INSERT INTO item VALUES ('A0','A',0,5)")
        second = keyset_page(self.reader, cursor_of(first[-1]), limit=10)
        self.assertEqual(len(first + second), old_total)
        self.assertNotEqual(len(first + second), self.reader.execute(
            "SELECT COUNT(*) FROM item WHERE tenant='A'").fetchone()[0])

    def test_11_declared_epoch_string_is_not_effective_transaction_snapshot(self):
        cursor = {'source_revision': 'same-label', 'snapshot': 's1', 'scope': 'A'}
        before = ids(page(self.reader, limit=20))
        self.writer.execute("INSERT INTO item VALUES ('A0','A',0,5)")
        after = ids(page(self.reader, limit=20))
        self.assertEqual(cursor, {'source_revision': 'same-label', 'snapshot': 's1', 'scope': 'A'})
        self.assertNotEqual(before, after)


if __name__ == '__main__':
    unittest.main()
