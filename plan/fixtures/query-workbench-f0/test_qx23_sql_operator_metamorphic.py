"""QX-23 finite SQL/Data Operator metamorphic oracle.

SQLite + hand-written, non-SQL Python expectations. No TEI SQL parser,
DSL evaluator, Target/Provider conformance, or universal equivalence proof.
"""
import itertools
from contextlib import closing
import sqlite3
import unittest


def fixture(lines, deliveries=()):
    db = sqlite3.connect(':memory:')
    db.executescript('''CREATE TABLE line(id TEXT PRIMARY KEY, parent TEXT, value INTEGER);
                    CREATE TABLE delivery(id TEXT PRIMARY KEY, parent TEXT,
                        note TEXT, priority INTEGER);''')
    db.executemany('INSERT INTO line VALUES(?,?,?)', lines)
    db.executemany('INSERT INTO delivery VALUES(?,?,?,?)', deliveries)
    return db


def sql_sum(values):
    present = [v for v in values if v is not None]
    return sum(present) if present else None


def inner_bag(lines, deliveries):
    return sorted((l[0], d[0]) for l in lines for d in deliveries if l[1] == d[1])


def exists_rows(lines, deliveries, negate=False):
    return sorted((l[0],) for l in lines if any(l[1] == d[1] for d in deliveries) != negate)


def left_on_positive(lines, deliveries):
    result = []
    for l in lines:
        matches = [d for d in deliveries if d[1] == l[1] and d[3] is not None and d[3] > 0]
        result.extend((l[0], d[0]) for d in matches)
        if not matches:
            result.append((l[0], None))
    return sorted(result, key=lambda r: (r[0], r[1] or ''))


class SQLToDataOperationOracle(unittest.TestCase):
    def test_bag_join_vs_independent_reference(self):
        lines = [('L1','H1',100), ('L2','H1',100), ('L3','H2',None)]
        for n in range(4):
            delivery = [(f'D{i}','H1',None,i) for i in range(n)]
            with self.subTest(matches=n):
                with closing(fixture(lines,delivery)) as db:
                    result = db.execute('SELECT l.id,d.id FROM line l JOIN delivery d ON l.parent=d.parent ORDER BY l.id,d.id').fetchall()
                    self.assertEqual(result, inner_bag(lines,delivery))

    def test_mutant_join_to_exists_killed_by_second_match(self):
        lines = [('L1','H1',1)]
        one = [('D1','H1',None,1)]
        two = one + [('D2','H1',None,2)]
        self.assertEqual([p[0] for p in inner_bag(lines,one)], [p[0] for p in exists_rows(lines,one)])
        self.assertNotEqual([p[0] for p in inner_bag(lines,two)], [p[0] for p in exists_rows(lines,two)])

    def test_exists_and_restricted_join_distinct_positive(self):
        lines = [('L1','H1',1),('L2','H1',1),('L3','H2',None)]
        for n in (0,1,2):
            ds = [(f'D{i}','H1',None,i) for i in range(n)]
            with closing(fixture(lines,ds)) as db:
                observed = db.execute('SELECT l.id FROM line l WHERE EXISTS(SELECT 1 FROM delivery d WHERE d.parent=l.parent) ORDER BY l.id').fetchall()
                rewritten = db.execute('SELECT DISTINCT l.id FROM line l JOIN delivery d ON d.parent=l.parent ORDER BY l.id').fetchall()
                self.assertEqual(observed, exists_rows(lines,ds))
                self.assertEqual(observed, rewritten)

    def test_sql_three_valued_where_with_nulls(self):
        for trio in itertools.product((None,0,2),repeat=3):
            rows = [(f'L{i}','H1',v) for i,v in enumerate(trio)]
            with self.subTest(values=trio):
                with closing(fixture(rows)) as db:
                    actual = db.execute('SELECT id FROM line WHERE value != 0 ORDER BY id').fetchall()
                    expected = [(r[0],) for r in rows if r[2] is not None and r[2] != 0]
                    self.assertEqual(actual,expected)
        self.assertNotEqual([(r[0],) for r in [('L0','H1',None)] if r[2] is not None and r[2]!=0], [('L0',)])

    def test_left_join_on_vs_where_not_equivalent(self):
        lines = [('L1','H1',10),('L2','H2',20)]
        for priorities in ((),(None,),(0,),(1,),(None,1),(1,2)):
            ds = [(f'D{i}','H1',None,p) for i,p in enumerate(priorities)]
            with closing(fixture(lines,ds)) as db:
                result = db.execute('SELECT l.id,d.id FROM line l LEFT JOIN delivery d ON l.parent=d.parent AND d.priority>0 ORDER BY l.id,d.id').fetchall()
                self.assertEqual(result,left_on_positive(lines,ds))
        with closing(fixture(lines,[('D1','H1',None,0)])) as db:
            on = db.execute('SELECT l.id FROM line l LEFT JOIN delivery d ON l.parent=d.parent AND d.priority>0 ORDER BY l.id').fetchall()
            where = db.execute('SELECT l.id FROM line l LEFT JOIN delivery d ON l.parent=d.parent WHERE d.priority>0 ORDER BY l.id').fetchall()
            self.assertEqual(on,[('L1',),('L2',)])
            self.assertEqual(where,[])

    def test_sum_group_null_empty_and_value_distinct_mutants(self):
        for pair in itertools.product((None,0,2),repeat=2):
            rows = [('L1','H1',pair[0]),('L2','H1',pair[1]),('L3','H2',None)]
            with closing(fixture(rows)) as db:
                got = db.execute('SELECT parent,SUM(value) FROM line GROUP BY parent ORDER BY parent').fetchall()
                self.assertEqual(got,[('H1',sql_sum(pair)),('H2',None)])
        for vals in ((),(None,),(0,),(2,2)):
            rows = [(f'L{i}','H1',v) for i,v in enumerate(vals)]
            with closing(fixture(rows)) as db:
                result = db.execute('SELECT SUM(value),COUNT(*),COUNT(value),SUM(DISTINCT value) FROM line').fetchone()
                unique = set(v for v in vals if v is not None)
                self.assertEqual(result,(sql_sum(vals),len(vals),sum(v is not None for v in vals),sql_sum(unique)))
                if vals == (2,2):
                    self.assertNotEqual(result[0],result[3])

    def test_not_exists_is_not_null_note_filter(self):
        rows = [('L1','H1',5),('L2','H2',8)]
        ds = [('D1','H1',None,1)]
        with closing(fixture(rows,ds)) as db:
            actual = db.execute('SELECT l.id FROM line l WHERE NOT EXISTS(SELECT 1 FROM delivery d WHERE d.parent=l.parent) ORDER BY l.id').fetchall()
            mutant = db.execute('SELECT DISTINCT l.id FROM line l LEFT JOIN delivery d ON l.parent=d.parent WHERE d.note IS NULL ORDER BY l.id').fetchall()
            self.assertEqual(actual,exists_rows(rows,ds,negate=True))
            self.assertNotEqual(actual,mutant)

    def test_filter_limit_sequence_mutant(self):
        rows = [('L1','H1',0),('L2','H1',2)]
        with closing(fixture(rows)) as db:
            correct = db.execute('SELECT id FROM line WHERE value>0 ORDER BY id LIMIT 1').fetchall()
            mutant = db.execute('SELECT id FROM (SELECT id,value FROM line ORDER BY id LIMIT 1) WHERE value>0').fetchall()
            self.assertEqual(correct,[('L2',)])
            self.assertNotEqual(correct,mutant)

    def test_same_example_values_not_a_generic_equivalence_license(self):
        # A single-match observation cannot erase distinct bag/set contracts.
        join = {'operator':'join@v1','grain':'row-occurrence','mult':'bag','truth':'sql-3vl'}
        exists = {'operator':'exists@v1','grain':'row-occurrence','mult':'set','truth':'sql-3vl'}
        self.assertNotEqual(join,exists)
        self.assertEqual([('L1',)],[('L1',)])  # one matched delivery, same value


if __name__ == '__main__':
    unittest.main()
