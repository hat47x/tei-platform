"""QX-28: SQLite and plain Python differences in Data Operator value semantics.

Finite independent SQLite/stdlib observations, not TEI Worker/Go/Provider/Host
conformance. No implicit collation, precision or rounding rule is adopted.
"""

from contextlib import closing
from decimal import Decimal, ROUND_HALF_EVEN
import sqlite3
import unittest
import unicodedata


class ValueSemanticsAcrossOperatorPlacement(unittest.TestCase):
    def test_01_binary_vs_nocase_order_changes_top_n(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE words (id INTEGER PRIMARY KEY, val TEXT)")
            db.executemany("INSERT INTO words VALUES (?,?)", [(1, "a"), (2, "B"), (3, "c"), (4, "A")])
            binary = [r[0] for r in db.execute("SELECT id FROM words ORDER BY val COLLATE BINARY, id LIMIT 2")]
            nocase = [r[0] for r in db.execute("SELECT id FROM words ORDER BY val COLLATE NOCASE, id LIMIT 2")]
            self.assertEqual(binary, [4, 2])
            self.assertEqual(nocase, [1, 4])
            self.assertNotEqual(binary, nocase)

    def test_02_sqlite_nocase_does_not_implement_unicode_casefold(self):
        with closing(sqlite3.connect(":memory:")) as db:
            same = db.execute("SELECT ? = ? COLLATE NOCASE", ("Ä", "ä")).fetchone()[0]
            self.assertEqual(same, 0)
            self.assertTrue("Ä".casefold() == "ä".casefold())

    def test_03_sqlite_binary_does_not_implicitly_normalize_unicode(self):
        composed = "é"
        decomposed = "e\u0301"
        with closing(sqlite3.connect(":memory:")) as db:
            self.assertEqual(db.execute("SELECT ? = ? COLLATE BINARY", (composed, decomposed)).fetchone()[0], 0)
        self.assertEqual(unicodedata.normalize("NFC", composed), unicodedata.normalize("NFC", decomposed))

    def test_04_null_sort_position_is_part_of_operator_contract(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE x (id TEXT PRIMARY KEY, v TEXT)")
            db.executemany("INSERT INTO x VALUES (?,?)", [("N", None), ("A", "a"), ("B", "b")])
            first = [x[0] for x in db.execute("SELECT id FROM x ORDER BY v ASC")]
            last = [x[0] for x in db.execute("SELECT id FROM x ORDER BY v ASC NULLS LAST")]
            self.assertEqual(first, ["N", "A", "B"])
            self.assertEqual(last, ["A", "B", "N"])

    def test_05_join_key_collation_changes_membership(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE left_side (k TEXT)")
            db.execute("CREATE TABLE right_side (k TEXT)")
            db.execute("INSERT INTO left_side VALUES ('a')")
            db.execute("INSERT INTO right_side VALUES ('A')")
            strict = db.execute("SELECT COUNT(*) FROM left_side l JOIN right_side r ON l.k = r.k").fetchone()[0]
            insensitive = db.execute("SELECT COUNT(*) FROM left_side l JOIN right_side r ON l.k = r.k COLLATE NOCASE").fetchone()[0]
            self.assertEqual((strict, insensitive), (0, 1))

    def test_06_text_vs_numeric_sort_is_not_a_presentation_detail(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE x (v TEXT)")
            db.executemany("INSERT INTO x VALUES (?)", [("2",), ("10",)])
            text_order = [x[0] for x in db.execute("SELECT v FROM x ORDER BY v")]
            numeric_order = [x[0] for x in db.execute("SELECT v FROM x ORDER BY CAST(v AS INTEGER)")]
            self.assertEqual(text_order, ["10", "2"])
            self.assertEqual(numeric_order, ["2", "10"])

    def test_07_decimal_vs_binary_float_addition(self):
        with closing(sqlite3.connect(":memory:")) as db:
            real_value = db.execute("SELECT 0.1 + 0.2").fetchone()[0]
        self.assertNotEqual(real_value, 0.3)
        self.assertEqual(Decimal("0.1") + Decimal("0.2"), Decimal("0.3"))

    def test_08_real_conversion_loses_integer_identity(self):
        integer = 9007199254740993
        with closing(sqlite3.connect(":memory:")) as db:
            stored, as_real = db.execute("SELECT ?, CAST(? AS REAL)", (integer, integer)).fetchone()
        self.assertEqual(stored, integer)
        self.assertEqual(as_real, 9007199254740992.0)
        self.assertNotEqual(stored, int(as_real))

    def test_09_aggregate_cast_changes_money_precision(self):
        amounts = ["0.1", "0.2"]
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE payments (amount TEXT)")
            db.executemany("INSERT INTO payments VALUES (?)", [(x,) for x in amounts])
            amount = db.execute("SELECT SUM(CAST(amount AS REAL)) FROM payments").fetchone()[0]
        self.assertNotEqual(amount, 0.3)
        self.assertEqual(sum((Decimal(x) for x in amounts), Decimal(0)), Decimal("0.3"))

    def test_10_rounding_rule_must_be_fixed_explicitly(self):
        with closing(sqlite3.connect(":memory:")) as db:
            pos, neg = db.execute("SELECT ROUND(2.5,0), ROUND(-2.5,0)").fetchone()
        self.assertEqual((pos, neg), (3.0, -3.0))
        self.assertEqual(Decimal("2.5").quantize(Decimal("1"), rounding=ROUND_HALF_EVEN), Decimal("2"))
        self.assertEqual(Decimal("-2.5").quantize(Decimal("1"), rounding=ROUND_HALF_EVEN), Decimal("-2"))

    def test_11_integer_division_depends_on_operand_types(self):
        with closing(sqlite3.connect(":memory:")) as db:
            integer, real = db.execute("SELECT 5 / 2, 5 / 2.0").fetchone()
        self.assertEqual(integer, 2)
        self.assertEqual(real, 2.5)
        self.assertEqual(Decimal(5) / Decimal(2), Decimal("2.5"))

    def test_12_integer_sum_overflow_vs_arbitrary_precision(self):
        max_int64 = 9223372036854775807
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE value (n INTEGER)")
            db.executemany("INSERT INTO value VALUES (?)", [(max_int64,), (1,)])
            with self.assertRaisesRegex(sqlite3.OperationalError, "integer overflow"):
                db.execute("SELECT SUM(n) FROM value").fetchone()
        self.assertEqual(sum([max_int64, 1]), 9223372036854775808)

    def test_13_same_cursor_label_with_changed_collation_loses_coverage(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE words (id TEXT PRIMARY KEY, word TEXT)")
            db.executemany("INSERT INTO words VALUES (?,?)", [("a", "a"), ("b", "B"), ("c", "c"), ("d", "D")])
            first = db.execute("SELECT id,word FROM words ORDER BY word COLLATE BINARY,id LIMIT 2").fetchall()
            continuation = db.execute("SELECT id,word FROM words WHERE word > ? COLLATE NOCASE ORDER BY word COLLATE NOCASE,id", (first[-1][1],)).fetchall()
            self.assertEqual([x[0] for x in first], ["b", "d"])
            self.assertEqual(continuation, [])
            self.assertEqual([x[0] for x in db.execute("SELECT id FROM words ORDER BY word COLLATE NOCASE,id")], ["a", "b", "c", "d"])
            self.assertEqual(len(first + continuation), 2)

    def test_14_empty_sum_vs_total_and_python_sum(self):
        with closing(sqlite3.connect(":memory:")) as db:
            db.execute("CREATE TABLE x(v INTEGER)")
            null_sum, zero_total = db.execute("SELECT SUM(v), TOTAL(v) FROM x").fetchone()
        self.assertIsNone(null_sum)
        self.assertEqual(zero_total, 0.0)
        self.assertEqual(sum([]), 0)


if __name__ == "__main__":
    unittest.main()
