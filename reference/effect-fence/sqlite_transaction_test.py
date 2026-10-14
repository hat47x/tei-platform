"""実SQLiteの確定境界を検証する、標準ライブラリだけの独立した回帰試験。

実行: python3 -m unittest -v reference/effect-fence/sqlite_transaction_test.py
TEIの認可Adapterや本番DB実装を代替するものではない。
"""

import os
import sqlite3
import tempfile
import threading
import unittest

SCHEMA = """
CREATE TABLE authority (scope TEXT PRIMARY KEY, epoch INTEGER NOT NULL, allowed INTEGER NOT NULL);
CREATE TABLE resource (id TEXT PRIMARY KEY, version INTEGER NOT NULL, value TEXT NOT NULL);
CREATE TABLE receipt (operation TEXT PRIMARY KEY, request TEXT NOT NULL, version INTEGER NOT NULL);
"""


def connect(path):
    db = sqlite3.connect(path, timeout=5, isolation_level=None, check_same_thread=False)
    db.execute("PRAGMA journal_mode=WAL")
    db.execute("PRAGMA synchronous=FULL")
    db.execute("PRAGMA busy_timeout=5000")
    return db


def change_authority(db, scope, epoch, allowed):
    db.execute("BEGIN IMMEDIATE")
    try:
        previous = db.execute("SELECT epoch FROM authority WHERE scope=?", (scope,)).fetchone()
        if epoch <= 0 or (previous and epoch <= previous[0]):
            raise ValueError("権限版は単調増加でなければなりません")
        db.execute(
            "INSERT INTO authority VALUES(?,?,?) "
            "ON CONFLICT(scope) DO UPDATE SET epoch=excluded.epoch, allowed=excluded.allowed",
            (scope, epoch, int(allowed)),
        )
        db.execute("COMMIT")
    except BaseException:
        db.execute("ROLLBACK")
        raise


def commit(db, *, scope, expected_epoch, resource, expected_version, operation, value):
    """実際の認可・保存先・確定記録が同一DBにある場合だけ成立する契約。"""
    request = repr((scope, expected_epoch, resource, expected_version, value))
    db.execute("BEGIN IMMEDIATE")
    try:
        # 既存確定記録の再照会であっても、先に現在の権限を確認する。
        authority = db.execute("SELECT epoch,allowed FROM authority WHERE scope=?", (scope,)).fetchone()
        if authority != (expected_epoch, 1):
            result = ("denied-authority", None)
        else:
            prior = db.execute("SELECT request, version FROM receipt WHERE operation=?", (operation,)).fetchone()
            if prior:
                result = ("replayed", prior[1]) if prior[0] == request else ("denied-id-conflict", None)
            else:
                row = db.execute("SELECT version FROM resource WHERE id=?", (resource,)).fetchone()
                current = row[0] if row else 0
                if current != expected_version:
                    result = ("denied-version", None)
                else:
                    version = current + 1
                    db.execute(
                        "INSERT INTO resource VALUES(?,?,?) "
                        "ON CONFLICT(id) DO UPDATE SET version=excluded.version,value=excluded.value",
                        (resource, version, value),
                    )
                    db.execute("INSERT INTO receipt VALUES(?,?,?)", (operation, request, version))
                    result = ("committed", version)
        db.execute("COMMIT")
        return result
    except BaseException:
        db.execute("ROLLBACK")
        raise


class SQLiteTransactionProof(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.temp.name, "proof.db")
        self.a = connect(self.path)
        self.a.executescript(SCHEMA)
        self.b = connect(self.path)
        change_authority(self.a, "editor:write:record-1", 1, True)
        self.args = dict(scope="editor:write:record-1", expected_epoch=1, resource="record-1",
                         expected_version=0, operation="operation-1", value="更新後")

    def tearDown(self):
        self.a.close()
        self.b.close()
        self.temp.cleanup()

    def test_revocation_between_authorization_and_commit(self):
        change_authority(self.b, self.args["scope"], 2, False)
        self.assertEqual(commit(self.a, **self.args)[0], "denied-authority")
        self.assertIsNone(self.b.execute("SELECT * FROM resource").fetchone())

    def test_epoch_change_even_when_still_allowed(self):
        change_authority(self.b, self.args["scope"], 2, True)
        self.assertEqual(commit(self.a, **self.args)[0], "denied-authority")
        self.assertIsNone(self.a.execute("SELECT * FROM receipt").fetchone())

    def test_resource_cas_and_retry_after_reconnect(self):
        self.b.execute("INSERT INTO resource VALUES(?,?,?)", ("record-1", 1, "別の書き手"))
        self.assertEqual(commit(self.a, **self.args)[0], "denied-version")
        self.assertEqual(self.a.execute("SELECT value FROM resource").fetchone()[0], "別の書き手")
        self.b.execute("DELETE FROM resource")
        self.assertEqual(commit(self.a, **self.args), ("committed", 1))
        self.a.close()
        self.a = connect(self.path)
        self.assertEqual(commit(self.a, **self.args), ("replayed", 1))
        self.assertEqual(self.a.execute("SELECT COUNT(*) FROM receipt").fetchone()[0], 1)

    def test_receipt_failure_rolls_back_resource_update(self):
        self.a.execute("CREATE TRIGGER deny_receipt BEFORE INSERT ON receipt "
                       "BEGIN SELECT RAISE(ABORT, 'injected failure'); END")
        with self.assertRaises(sqlite3.IntegrityError):
            commit(self.a, **self.args)
        self.assertIsNone(self.b.execute("SELECT * FROM resource").fetchone())
        self.assertIsNone(self.b.execute("SELECT * FROM receipt").fetchone())

    def test_parallel_connections_commit_once(self):
        barrier = threading.Barrier(2)
        results = []
        errors = []
        def worker(db):
            try:
                barrier.wait(timeout=5)
                results.append(commit(db, **self.args))
            except BaseException as exc:
                errors.append(exc)
        a = threading.Thread(target=worker, args=(self.a,))
        b = threading.Thread(target=worker, args=(self.b,))
        a.start(); b.start(); a.join(timeout=10); b.join(timeout=10)
        self.assertFalse(a.is_alive() or b.is_alive())
        self.assertEqual(errors, [])
        self.assertCountEqual(results, [("committed", 1), ("replayed", 1)])
        self.assertEqual(self.a.execute("SELECT COUNT(*) FROM receipt").fetchone()[0], 1)

    def test_revoked_replay_does_not_disclose_receipt(self):
        self.assertEqual(commit(self.a, **self.args), ("committed", 1))
        change_authority(self.b, self.args["scope"], 2, False)
        self.assertEqual(commit(self.a, **self.args), ("denied-authority", None))
        self.assertEqual(self.b.execute("SELECT COUNT(*) FROM receipt").fetchone()[0], 1)

    def test_operation_id_collision(self):
        self.assertEqual(commit(self.a, **self.args), ("committed", 1))
        other = dict(self.args, value="不正な再送")
        self.assertEqual(commit(self.b, **other)[0], "denied-id-conflict")
        self.assertEqual(self.a.execute("SELECT value FROM resource").fetchone()[0], "更新後")


if __name__ == "__main__":
    unittest.main()
