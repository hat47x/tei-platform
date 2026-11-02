"""QX-25: distinguish displayed columns, SQL-authorized source references,
observed output and disclosure/effect authorization. SQLite-only synthetic data.

sqlite3.set_authorizer records references checked while preparing statements;
it does NOT prove physical row reads, tenant scope, or an external grant.
"""
import sqlite3
import unittest


def db_for(hidden_values=(1, 0, 2)):
    db = sqlite3.connect(":memory:")
    db.executescript("""
        CREATE TABLE item (id TEXT PRIMARY KEY, visible TEXT NOT NULL,
                           hidden_score INTEGER);
        CREATE TABLE link (item_id TEXT NOT NULL, hidden_note TEXT);
        INSERT INTO link VALUES ('I1', 'restricted');
    """)
    db.executemany("INSERT INTO item VALUES (?, ?, ?)",
                   [(f'I{i}', f'public-{i}', value)
                    for i, value in enumerate(hidden_values, 1)])
    return db


def with_reads(db, sql, deny=frozenset()):
    """Record SQLite SQLITE_READ authorizer requests; not a row-access audit."""
    checks = []

    def authorizer(op, table, column, db_name, origin):
        if op == sqlite3.SQLITE_READ:
            checks.append((table, column))
            if (table, column) in deny:
                return sqlite3.SQLITE_DENY
        return sqlite3.SQLITE_OK

    db.set_authorizer(authorizer)
    try:
        rows = db.execute(sql).fetchall()
        return rows, frozenset(checks)
    finally:
        db.set_authorizer(None)


class HiddenReadAndDisclosureTests(unittest.TestCase):
    def setUp(self):
        self.db = db_for()

    def tearDown(self):
        self.db.close()

    def test_01_plain_public_projection_uses_no_hidden_column(self):
        rows, cols = with_reads(self.db, "SELECT id FROM item ORDER BY id")
        self.assertEqual(rows, [('I1',), ('I2',), ('I3',)])
        self.assertIn(('item', 'id'), cols)
        self.assertNotIn(('item', 'hidden_score'), cols)

    def test_02_hidden_predicate_is_read_even_when_output_is_public(self):
        rows, cols = with_reads(self.db,
            "SELECT id FROM item WHERE hidden_score > 0 ORDER BY id")
        self.assertEqual(rows, [('I1',), ('I3',)])
        self.assertIn(('item', 'hidden_score'), cols)
        self.assertNotIn(('item', 'visible'), cols)

    def test_03_hidden_sort_changes_public_row_order(self):
        rows, cols = with_reads(self.db,
            "SELECT id FROM item ORDER BY hidden_score DESC, id")
        self.assertEqual(rows, [('I3',), ('I1',), ('I2',)])
        self.assertIn(('item', 'hidden_score'), cols)

    def test_04_count_filter_has_secret_dependency_without_secret_output(self):
        rows, cols = with_reads(self.db,
            "SELECT COUNT(*) FROM item WHERE hidden_score > 0")
        self.assertEqual(rows, [(2,)])
        self.assertIn(('item', 'hidden_score'), cols)

    def test_05_join_exists_uses_hidden_other_relation(self):
        rows, cols = with_reads(self.db, """SELECT id FROM item i WHERE EXISTS
            (SELECT 1 FROM link l WHERE l.item_id=i.id AND
             l.hidden_note='restricted') ORDER BY id""")
        self.assertEqual(rows, [('I1',)])
        self.assertIn(('link', 'hidden_note'), cols)
        self.assertIn(('link', 'item_id'), cols)

    def test_06_authorizer_denial_on_hidden_where_blocks_public_output(self):
        query = "SELECT id FROM item WHERE hidden_score > 0"
        with self.assertRaises(sqlite3.DatabaseError):
            with_reads(self.db, query, {('item', 'hidden_score')})
        # The same syntactically public projection without hidden predicate works.
        rows, _ = with_reads(self.db, "SELECT id FROM item ORDER BY id")
        self.assertEqual(len(rows), 3)

    def test_07_denial_on_hidden_sort_blocks_public_output(self):
        with self.assertRaises(sqlite3.DatabaseError):
            with_reads(self.db, "SELECT id FROM item ORDER BY hidden_score",
                       {('item', 'hidden_score')})

    def test_08_secret_only_change_alters_membership_of_public_output(self):
        first, used_first = with_reads(self.db,
            "SELECT id FROM item WHERE hidden_score > 0 ORDER BY id")
        other = db_for((0, 0, 2))
        try:
            second, used_second = with_reads(other,
                "SELECT id FROM item WHERE hidden_score > 0 ORDER BY id")
        finally:
            other.close()
        self.assertEqual(first, [('I1',), ('I3',)])
        self.assertEqual(second, [('I3',)])
        self.assertEqual(used_first, used_second)
        self.assertEqual(first[1], second[0])

    def test_09_secret_only_change_alters_count_output(self):
        first, _ = with_reads(self.db,
            "SELECT COUNT(*) FROM item WHERE hidden_score > 0")
        other = db_for((0, 0, 2))
        try:
            second, _ = with_reads(other,
                "SELECT COUNT(*) FROM item WHERE hidden_score > 0")
        finally:
            other.close()
        self.assertEqual(first, [(2,)])
        self.assertEqual(second, [(1,)])

    def test_10_hidden_rank_affects_top_one_without_public_value_change(self):
        first, cols = with_reads(self.db,
            "SELECT id FROM item ORDER BY hidden_score DESC, id LIMIT 1")
        self.assertIn(('item', 'hidden_score'), cols)
        self.db.execute("UPDATE item SET hidden_score=5 WHERE id='I1'")
        second, cols2 = with_reads(self.db,
            "SELECT id FROM item ORDER BY hidden_score DESC, id LIMIT 1")
        self.assertEqual(first, [('I3',)])
        self.assertEqual(second, [('I1',)])
        self.assertEqual(cols, cols2)

    def test_11_hidden_join_filter_is_not_displayed_column(self):
        query = """SELECT i.id FROM item i JOIN link l
            ON l.item_id=i.id WHERE l.hidden_note='restricted'"""
        rows, cols = with_reads(self.db, query)
        self.assertEqual(rows, [('I1',)])
        self.assertIn(('link', 'hidden_note'), cols)
        with self.assertRaises(sqlite3.DatabaseError):
            with_reads(self.db, query, {('link', 'hidden_note')})

    def test_12_public_output_value_equality_does_not_prove_no_hidden_read(self):
        plain, plain_refs = with_reads(self.db,
            "SELECT id FROM item WHERE id='I1'")
        filtered, filtered_refs = with_reads(self.db,
            "SELECT id FROM item WHERE id='I1' AND hidden_score>0")
        self.assertEqual(plain, filtered)
        self.assertNotIn(('item', 'hidden_score'), plain_refs)
        self.assertIn(('item', 'hidden_score'), filtered_refs)


if __name__ == '__main__':
    unittest.main()
