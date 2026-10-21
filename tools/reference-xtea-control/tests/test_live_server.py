#!/usr/bin/env python3
"""Live XeadServer DBMethod test against embedded Jetty and a disposable H2 DB.

Requires:
    XTEA_SERVER_SOURCE  path to the cloned XeadServer repository
    XTEA_RUNTIME_HOME   directory containing xteadrv.jar, h2-1.4.187.jar, etc.

The test is skipped when either variable is absent, so it does not make the
regular repository health check depend on a local X-TEA installation.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from xtea_server import ServerClient  # noqa: E402

SOURCE = os.environ.get("XTEA_SERVER_SOURCE")
RUNTIME = os.environ.get("XTEA_RUNTIME_HOME")


@unittest.skipUnless(SOURCE and RUNTIME, "XTEA_SERVER_SOURCE and XTEA_RUNTIME_HOME are required")
class LiveServerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.source = Path(SOURCE).resolve()
        cls.runtime = Path(RUNTIME).resolve()
        cls.tmp = tempfile.TemporaryDirectory(prefix="xtea-server-live-")
        cls.work = Path(cls.tmp.name)
        build = ROOT / "server-harness" / "build.ps1"
        result = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(build),
             "-XteaSource", str(cls.source), "-RuntimeHome", str(cls.runtime)],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)
        cls.classpath = (ROOT / "build" / "server-classpath.txt").read_text(encoding="utf-8").strip()

        cls.db_url = "jdbc:h2:file:" + (cls.work / "xtea").as_posix()
        h2 = cls.runtime / "h2-1.4.187.jar"
        init_cp = f"{ROOT / 'build' / 'server-harness-classes'};{h2}"
        result = subprocess.run(
            ["java", "-cp", init_cp, "H2Init", cls.db_url],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        cls.xeaf = cls.work / "system.xeaf"
        cls.xeaf.write_text(
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            f'<System DatabaseName="{cls.db_url}" DatabaseUser="sa" DatabasePassword="" DBCP="" />\n',
            encoding="utf-8",
        )
        cls.endpoint = cls.work / "endpoint.json"
        cls.process = subprocess.Popen(
            ["java", "-cp", cls.classpath, "XteaServerHarness",
             "--xeaf", str(cls.xeaf), "--endpoint", str(cls.endpoint), "--port", "0"],
            cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if cls.endpoint.exists():
                try:
                    cls.url = json.loads(cls.endpoint.read_text(encoding="utf-8"))["url"]
                    break
                except (json.JSONDecodeError, OSError):
                    pass
            if cls.process.poll() is not None:
                raise RuntimeError(f"server harness exited with {cls.process.returncode}")
            time.sleep(0.1)
        else:
            raise RuntimeError("server harness endpoint was not written")
        cls.client = ServerClient(cls.url, timeout=15)

    @classmethod
    def tearDownClass(cls) -> None:
        try:
            cls.process.terminate()
            cls.process.wait(timeout=10)
        except Exception:
            try:
                cls.process.kill()
            except Exception:
                pass
        cls.tmp.cleanup()

    def test_status(self) -> None:
        value = self.client.status()
        self.assertEqual(value["status"], 200)
        self.assertIn("DB-Method", value["body"])

    def test_all_db_branches(self) -> None:
        q = chr(39)
        self.assertEqual(self.client.insert(f"INSERT INTO T VALUES (1, {q}alpha{q})")["body"], "1")
        self.assertEqual(self.client.count("SELECT COUNT(*) FROM T")["body"], "1")
        value = self.client.query("SELECT * FROM T")
        self.assertTrue(value["serialized"])
        self.assertGreater(value["byteLength"], 0)
        self.assertEqual(self.client.update(f"UPDATE T SET NAME={q}beta{q} WHERE ID=1")["body"], "1")
        self.assertIn("commited", self.client.commit("")["body"])
        self.assertEqual(self.client.delete("DELETE FROM T WHERE ID=1")["body"], "1")
        self.assertIn("rollbacked", self.client.rollback("")["body"])
        self.assertEqual(self.client.ip()["body"], "127.0.0.1")


if __name__ == "__main__":
    unittest.main(verbosity=2)
