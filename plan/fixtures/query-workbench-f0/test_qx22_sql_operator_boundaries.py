"""QX-22 SQL operator-boundary pressure; synthetic SQLite independent oracle.

This suite does not parse/import SQL, define TEI DSL semantics, implement a
SQL-to-DSL transpiler, or prove equivalence on other Providers/dialects.
"""
import sqlite3
import unittest


def fixture():
    db = sqlite3.connect(":memory:")
    db.executescript("""
        CREATE TABLE header (id TEXT PRIMARY KEY);
        CREATE TABLE line (id TEXT PRIMARY KEY, header_id TEXT NOT NULL,
                           amount INTEGER NOT NULL);
        CREATE TABLE delivery (id TEXT PRIMARY KEY, header_id TEXT NOT NULL,
                               rank INTEGER NOT NULL, note TEXT);
        INSERT INTO header VALUES ('H1'),('H2');
        INSERT INTO line VALUES ('L1','H1',100),('L2','H1',100),('L3','H2',200);
        INSERT INTO delivery VALUES ('D1','H1',1,NULL),('D2','H1',2,'ok');
    """)
    return db


class SQLBoundaryOracle(unittest.TestCase):
    def setUp(self):
        self.db = fixture()

    def tearDown(self):
        self.db.close()

    def sql(self, source):
        return self.db.execute(source).fetchall()

    def test_qx22_01_left_join_filter_in_on_vs_where(self):
        on = self.sql("""SELECT h.id,d.id FROM header h LEFT JOIN delivery d
            ON h.id=d.header_id AND d.rank=99 ORDER BY h.id""")
        where = self.sql("""SELECT h.id,d.id FROM header h LEFT JOIN delivery d
            ON h.id=d.header_id WHERE d.rank=99 ORDER BY h.id""")
        self.assertEqual(on, [('H1', None), ('H2', None)])
        self.assertEqual(where, [])

    def test_qx22_02_exists_is_not_join_bag_multiplicity(self):
        joined = self.sql("""SELECT l.id FROM line l JOIN delivery d
            ON l.header_id=d.header_id ORDER BY l.id,d.id""")
        exists = self.sql("""SELECT l.id FROM line l WHERE EXISTS
            (SELECT 1 FROM delivery d WHERE d.header_id=l.header_id)
            ORDER BY l.id""")
        self.assertEqual(joined, [('L1',),('L1',),('L2',),('L2',)])
        self.assertEqual(exists, [('L1',),('L2',)])

    def test_qx22_03_not_exists_not_nullable_value_is_null(self):
        correct = self.sql("""SELECT h.id FROM header h WHERE NOT EXISTS
            (SELECT 1 FROM delivery d WHERE d.header_id=h.id) ORDER BY h.id""")
        wrong = self.sql("""SELECT DISTINCT h.id FROM header h LEFT JOIN delivery d
            ON d.header_id=h.id WHERE d.note IS NULL ORDER BY h.id""")
        self.assertEqual(correct, [('H2',)])
        self.assertEqual(wrong, [('H1',),('H2',)])

    def test_qx22_04_sql_null_equality_is_unknown(self):
        equals_null = self.sql("SELECT id FROM delivery WHERE note = NULL")
        is_null = self.sql("SELECT id FROM delivery WHERE note IS NULL")
        not_equal = self.sql("SELECT id FROM delivery WHERE NOT(note = 'ok')")
        self.assertEqual(equals_null, [])
        self.assertEqual(is_null, [('D1',)])
        self.assertEqual(not_equal, [])

    def test_qx22_05_limit_and_filter_are_not_commutative(self):
        limit_first = self.sql("""SELECT id,amount FROM
            (SELECT id,amount FROM line ORDER BY id LIMIT 1)
            WHERE amount>100""")
        filter_first = self.sql("""SELECT id,amount FROM line WHERE amount>100
            ORDER BY id LIMIT 1""")
        self.assertEqual(limit_first, [])
        self.assertEqual(filter_first, [('L3',200)])

    def test_qx22_06_where_before_group_vs_having_after_group(self):
        where_first = self.sql("""SELECT header_id,SUM(amount) FROM line
            WHERE amount>100 GROUP BY header_id ORDER BY header_id""")
        having = self.sql("""SELECT header_id,SUM(amount) FROM line
            GROUP BY header_id HAVING SUM(amount)>100 ORDER BY header_id""")
        self.assertEqual(where_first, [('H2',200)])
        self.assertEqual(having, [('H1',200),('H2',200)])

    def test_qx22_07_distinct_is_on_value_not_fact_identity(self):
        total = self.sql("SELECT SUM(amount) FROM line WHERE header_id='H1'")
        distinct = self.sql("SELECT SUM(DISTINCT amount) FROM line WHERE header_id='H1'")
        self.assertEqual(total, [(200,)])
        self.assertEqual(distinct, [(100,)])

    def test_qx22_08_self_join_occurrence_count(self):
        pairs = self.sql("""SELECT a.id,b.id FROM line a JOIN line b
            ON a.header_id=b.header_id WHERE a.id<>b.id ORDER BY a.id,b.id""")
        self.assertEqual(pairs, [('L1','L2'),('L2','L1')])

    def test_qx22_09_json_missing_null_empty_are_distinct(self):
        self.assertEqual(self.sql("SELECT json_type('{\"x\":null}','$.x')"), [('null',)])
        self.assertEqual(self.sql("SELECT json_type('{\"x\":null}','$.y')"), [(None,)])
        self.assertEqual(self.sql("SELECT count(*) FROM json_each('[]')"), [(0,)])
        self.assertEqual(self.sql("SELECT count(*) FROM json_each('null')"), [(1,)])
        self.assertEqual(self.sql("SELECT count(*) FROM json_each('{}')"), [(0,)])

    def test_qx22_10_inner_join_left_predicate_pushdown_local_equal(self):
        on = self.sql("""SELECT l.id,d.id FROM line l JOIN delivery d
            ON l.header_id=d.header_id AND l.id='L1' ORDER BY d.id""")
        where = self.sql("""SELECT l.id,d.id FROM line l JOIN delivery d
            ON l.header_id=d.header_id WHERE l.id='L1' ORDER BY d.id""")
        self.assertEqual(on, [('L1','D1'),('L1','D2')])
        self.assertEqual(on, where)


if __name__ == '__main__':
    unittest.main()
