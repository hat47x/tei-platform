#!/usr/bin/env python3
"""SQLite-only transaction boundary comparison; NOT TEI Provider/Host evidence."""
import sqlite3
import unittest


def database():
    db = sqlite3.connect(":memory:", isolation_level=None)
    db.executescript("""
        CREATE TABLE parent(id TEXT PRIMARY KEY);
        CREATE TABLE child(id TEXT PRIMARY KEY, parent_id TEXT NOT NULL,
            FOREIGN KEY(parent_id) REFERENCES parent(id));
        PRAGMA foreign_keys=ON;
    """)
    return db


class SQLiteSaveBoundaryTests(unittest.TestCase):
    def test_single_explicit_transaction_rolls_back_parent_and_bad_child(self):
        db = database()
        db.execute("BEGIN")
        db.execute("INSERT INTO parent VALUES ('H')")
        with self.assertRaises(sqlite3.IntegrityError):
            db.execute("INSERT INTO child VALUES ('L','missing-parent')")
        db.execute("ROLLBACK")
        self.assertEqual(db.execute("SELECT COUNT(*) FROM parent").fetchone()[0], 0)
        self.assertEqual(db.execute("SELECT COUNT(*) FROM child").fetchone()[0], 0)
        db.close()

    def test_independent_autocommit_statements_can_partially_succeed(self):
        db = database()
        db.execute("INSERT INTO parent VALUES ('H')")  # autocommitted already
        with self.assertRaises(sqlite3.IntegrityError):
            db.execute("INSERT INTO child VALUES ('L','missing-parent')")
        self.assertEqual(db.execute("SELECT COUNT(*) FROM parent").fetchone()[0], 1)
        self.assertEqual(db.execute("SELECT COUNT(*) FROM child").fetchone()[0], 0)
        db.close()

    def test_savepoint_rollback_is_not_rollback_of_whole_transaction(self):
        db = database()
        db.execute("BEGIN")
        db.execute("INSERT INTO parent VALUES ('H')")
        db.execute("SAVEPOINT later_changes")
        db.execute("INSERT INTO child VALUES ('L','H')")
        db.execute("ROLLBACK TO SAVEPOINT later_changes")
        db.execute("RELEASE SAVEPOINT later_changes")
        db.execute("COMMIT")
        self.assertEqual(db.execute("SELECT COUNT(*) FROM parent").fetchone()[0], 1)
        self.assertEqual(db.execute("SELECT COUNT(*) FROM child").fetchone()[0], 0)
        db.close()


if __name__ == "__main__":
    unittest.main(verbosity=2)
