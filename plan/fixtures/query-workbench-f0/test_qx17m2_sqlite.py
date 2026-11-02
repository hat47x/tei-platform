"""QX-17 M2 synthetic SQLite contrast: value lineage != result-row dependency.

Only independent local reference evidence. Not a TEI parser, worker, provider,
Data Context, semantic acceptance, or security/authority implementation.
"""
import sqlite3
import unittest


def sql_db(lines=(("L1", "H", 100), ("L2", "H", 100)), deliveries=()):
    db = sqlite3.connect(":memory:")
    db.executescript("""
      CREATE TABLE header (id TEXT PRIMARY KEY);
      CREATE TABLE line (id TEXT PRIMARY KEY, header_id TEXT NOT NULL, amount INTEGER NOT NULL);
      CREATE TABLE delivery (id TEXT PRIMARY KEY, header_id TEXT NOT NULL, rank INTEGER NOT NULL);
      INSERT INTO header VALUES ('H');
    """)
    for _id, h, _ in lines:
        if h != "H":
            db.execute("INSERT OR IGNORE INTO header(id) VALUES (?)", (h,))
    db.executemany("INSERT INTO line VALUES (?, ?, ?)", lines)
    db.executemany("INSERT INTO delivery VALUES (?, ?, ?)", deliveries)
    return db


class QueryOutputDependencyReference(unittest.TestCase):
    def test_a_value_lineage_same_but_inner_join_controls_rows(self):
        db = sql_db(deliveries=(("D1", "H", 1), ("D2", "H", 2), ("D3", "H", 3)))
        try:
            alone = db.execute("SELECT amount FROM line ORDER BY id").fetchall()
            joined = db.execute("SELECT l.amount FROM line l JOIN delivery d ON d.header_id=l.header_id ORDER BY l.id,d.id").fetchall()
            self.assertEqual(alone, [(100,), (100,)])
            self.assertEqual(joined, [(100,)] * 6)
        finally:
            db.close()
        empty = sql_db()
        try:
            self.assertEqual(empty.execute("SELECT l.amount FROM line l JOIN delivery d ON d.header_id=l.header_id").fetchall(), [])
            self.assertEqual(empty.execute("SELECT amount FROM line").fetchall(), [(100,), (100,)])
        finally:
            empty.close()

    def test_b_left_join_unmatched_row_is_replaced_not_augmented(self):
        db = sql_db()
        try:
            q = "SELECT h.id,d.id FROM header h LEFT JOIN delivery d ON d.header_id=h.id ORDER BY d.id"
            before = db.execute(q).fetchall()
            db.execute("INSERT INTO delivery VALUES ('D1','H',1)")
            after = db.execute(q).fetchall()
            self.assertEqual(before, [("H", None)])
            self.assertEqual(after, [("H", "D1")])
            self.assertNotIn(("H", None), after)
        finally:
            db.close()

    def test_c_anti_join_retracts_header_when_matching_row_arrives(self):
        db = sql_db()
        try:
            q = "SELECT h.id FROM header h LEFT JOIN delivery d ON d.header_id=h.id WHERE d.id IS NULL"
            self.assertEqual(db.execute(q).fetchall(), [("H",)])
            db.execute("INSERT INTO delivery VALUES ('D1','H',1)")
            self.assertEqual(db.execute(q).fetchall(), [])
        finally:
            db.close()

    def test_d_join_sum_distinct_and_post_join_rollup_disagree(self):
        db = sql_db(deliveries=(("D1", "H", 1), ("D2", "H", 2), ("D3", "H", 3)))
        try:
            independent = db.execute("SELECT SUM(amount) FROM line").fetchone()[0]
            fanned = db.execute("SELECT SUM(l.amount) FROM line l JOIN delivery d ON d.header_id=l.header_id").fetchone()[0]
            value_distinct = db.execute("SELECT SUM(DISTINCT l.amount) FROM line l JOIN delivery d ON d.header_id=l.header_id").fetchone()[0]
            preaggregated = db.execute("""
              WITH line_total AS (SELECT header_id,SUM(amount) AS amount FROM line GROUP BY header_id)
              SELECT SUM(lt.amount) FROM line_total lt JOIN delivery d ON d.header_id=lt.header_id
            """).fetchone()[0]
            self.assertEqual((independent, fanned, value_distinct, preaggregated), (200, 600, 100, 600))
        finally:
            db.close()

    def test_e_order_limit_selects_different_line_with_new_delivery(self):
        db = sql_db(lines=(("L1", "H", 100), ("L2", "J", 200)),
                    deliveries=(("D1", "H", 30), ("D2", "J", 20)))
        try:
            q = """SELECT l.id,l.amount FROM line l
                   JOIN delivery d ON d.header_id=l.header_id
                   ORDER BY d.rank ASC,l.id ASC LIMIT 1"""
            self.assertEqual(db.execute(q).fetchall(), [("L2", 200)])
            db.execute("INSERT INTO delivery VALUES ('D0','H',10)")
            self.assertEqual(db.execute(q).fetchall(), [("L1", 100)])
        finally:
            db.close()

    def test_f_join_weights_change_avg_without_changing_line_value_sources(self):
        db = sql_db(lines=(("L1", "H", 100), ("L2", "J", 200)),
                    deliveries=(("D1", "H", 1), ("D2", "H", 2), ("D3", "H", 3), ("D4", "J", 4)))
        try:
            plain = db.execute("SELECT AVG(amount) FROM line").fetchone()[0]
            joined = db.execute("SELECT AVG(l.amount) FROM line l JOIN delivery d ON d.header_id=l.header_id").fetchone()[0]
            self.assertEqual((plain, joined), (150.0, 125.0))
        finally:
            db.close()


if __name__ == "__main__":
    unittest.main()
