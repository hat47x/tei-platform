"""SM-15: SQLite-only evidence that SQL's implicit schema dependencies matter.

These isolated tests use hand-authored artificial data; they do not test TEI's
parser, Model/Meaning acceptance, a PostgreSQL/OData Provider, or migration.
"""
import sqlite3
import unittest


class ImplicitSQLDependencies(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.db.execute("PRAGMA foreign_keys=OFF")
        self.db.executescript("""
            CREATE TABLE orders (
                tenant_id TEXT NOT NULL,
                order_id TEXT NOT NULL,
                status TEXT CHECK(status IN ('draft','submitted')),
                PRIMARY KEY(tenant_id,order_id)
            );
            CREATE TABLE payments (
                tenant_id TEXT NOT NULL,
                order_id TEXT NOT NULL,
                amount INTEGER NOT NULL,
                PRIMARY KEY(tenant_id,order_id)
            );
            INSERT INTO orders VALUES ('A','O1','draft'),('A','O2','submitted');
            INSERT INTO payments VALUES ('A','O1',600),('A','O2',700);
        """)

    def tearDown(self):
        self.db.close()

    def rows(self, sql):
        return self.db.execute(sql).fetchall()

    def test_01_declared_composite_key_order_is_not_sort_contract(self):
        columns = self.rows("PRAGMA table_info(orders)")
        self.assertEqual([(r[1],r[5]) for r in columns if r[5]],
                         [('tenant_id',1),('order_id',2)])
        # An ORDER BY business sort and TEI Key Member SortKey are not declared.

    def test_02_natural_join_without_shared_status_has_two_rows(self):
        self.assertEqual(self.rows("SELECT order_id FROM orders NATURAL JOIN payments ORDER BY order_id"),
                         [('O1',),('O2',)])

    def test_03_unchanged_natural_join_changes_with_schema(self):
        sql = "SELECT order_id FROM orders NATURAL JOIN payments ORDER BY order_id"
        before = self.rows(sql)
        self.db.execute("ALTER TABLE payments ADD COLUMN status TEXT DEFAULT 'draft'")
        after = self.rows(sql)
        self.assertEqual(before, [('O1',),('O2',)])
        self.assertEqual(after, [('O1',)])
        # SQL bytes remain the same, but an implicit join predicate was added.

    def test_04_explicit_on_join_is_stable_under_same_change(self):
        sql = ("SELECT o.order_id FROM orders o JOIN payments p "
               "ON o.tenant_id=p.tenant_id AND o.order_id=p.order_id ORDER BY o.order_id")
        before = self.rows(sql)
        self.db.execute("ALTER TABLE payments ADD COLUMN status TEXT DEFAULT 'draft'")
        self.assertEqual(self.rows(sql), before)
        self.assertEqual(before, [('O1',),('O2',)])

    def test_05_select_star_schema_changes_output_shape(self):
        sql = "SELECT * FROM payments ORDER BY order_id"
        before = self.rows(sql)
        self.db.execute("ALTER TABLE payments ADD COLUMN status TEXT DEFAULT 'draft'")
        after = self.rows(sql)
        self.assertEqual(len(before[0]), 3)
        self.assertEqual(len(after[0]), 4)

    def test_06_foreign_key_definition_not_enforcement(self):
        self.db.execute("""CREATE TABLE child(
            tenant_id TEXT, order_id TEXT,
            FOREIGN KEY(tenant_id,order_id) REFERENCES orders(tenant_id,order_id)
        )""")
        self.assertEqual(len(self.rows('PRAGMA foreign_key_list(child)')), 2)
        self.assertEqual(self.rows('PRAGMA foreign_keys'), [(0,)])
        self.db.execute("INSERT INTO child VALUES ('A','missing')")
        self.assertEqual(self.rows('SELECT count(*) FROM child'), [(1,)])

    def test_07_foreign_key_enforcement_enabled_on_connection(self):
        self.db.execute('PRAGMA foreign_keys=ON')
        self.db.execute("""CREATE TABLE child(
            tenant_id TEXT, order_id TEXT,
            FOREIGN KEY(tenant_id,order_id) REFERENCES orders(tenant_id,order_id)
        )""")
        with self.assertRaises(sqlite3.IntegrityError):
            self.db.execute("INSERT INTO child VALUES ('A','missing')")
        self.assertEqual(self.rows('SELECT count(*) FROM child'), [(0,)])

    def test_08_check_allows_null_without_not_null(self):
        self.db.execute("INSERT INTO orders VALUES ('A','O3',NULL)")
        self.assertEqual(self.rows("SELECT status FROM orders WHERE order_id='O3'"), [(None,)])
        with self.assertRaises(sqlite3.IntegrityError):
            self.db.execute("INSERT INTO orders VALUES ('A','O4','cancelled')")

    def test_09_check_range_does_not_define_transition(self):
        self.db.execute("UPDATE orders SET status='draft' WHERE order_id='O2'")
        self.assertEqual(self.rows("SELECT status FROM orders WHERE order_id='O2'"), [('draft',)])
        # Whether submitted->draft is business-permitted requires an Owner Rule.

    def test_10_default_omission_differs_from_explicit_null(self):
        self.db.execute("CREATE TABLE inputs(id INTEGER PRIMARY KEY, status TEXT DEFAULT 'draft')")
        self.db.execute("INSERT INTO inputs(id) VALUES (1)")
        self.db.execute("INSERT INTO inputs(id,status) VALUES (2,NULL)")
        self.assertEqual(self.rows('SELECT status FROM inputs ORDER BY id'),
                         [('draft',),(None,)])

    def test_11_history_columns_do_not_exclude_overlaps(self):
        self.db.execute("CREATE TABLE membership(person TEXT, org TEXT, valid_from INT, valid_to INT)")
        self.db.execute("INSERT INTO membership VALUES ('P','X',1,10),('P','Y',5,15)")
        self.assertEqual(self.rows("SELECT org FROM membership WHERE person='P' "
                                   "AND valid_from <= 7 AND (valid_to IS NULL OR 7 < valid_to) "
                                   "ORDER BY org"), [('X',),('Y',)])

    def test_12_complete_zero_becomes_nonzero_without_sql_change(self):
        sql = "SELECT NOT EXISTS(SELECT 1 FROM holds WHERE active=1)"
        self.db.execute("CREATE TABLE holds(active INT)")
        self.assertEqual(self.rows(sql), [(1,)])
        self.db.execute("INSERT INTO holds VALUES (1)")
        self.assertEqual(self.rows(sql), [(0,)])

    def test_13_missing_table_is_error_not_empty_set(self):
        with self.assertRaises(sqlite3.OperationalError):
            self.db.execute("SELECT * FROM not_fetched").fetchall()

    def test_14_unique_nullable_columns_do_not_imply_not_null(self):
        self.db.execute("CREATE TABLE token(value TEXT UNIQUE)")
        self.db.executemany("INSERT INTO token VALUES (?)", [(None,), (None,)])
        self.assertEqual(self.rows("SELECT count(*) FROM token WHERE value IS NULL"), [(2,)])


if __name__ == '__main__':
    unittest.main()
