#!/usr/bin/env python3
"""Deploy the XeadServer webapp to embedded Tomcat and exercise it over HTTP."""

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
SKELETON = os.environ.get("XTEA_SKELETON_HOME")


@unittest.skipUnless(SOURCE and RUNTIME and SKELETON,
                     "XTEA_SERVER_SOURCE, XTEA_RUNTIME_HOME and XTEA_SKELETON_HOME are required")
class TomcatWarTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.tmp = tempfile.TemporaryDirectory(prefix="xtea-tomcat-")
        cls.work = Path(cls.tmp.name)
        fixture = ROOT / "driver-fixture.ps1"
        target = cls.work / "skeleton"
        result = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(fixture),
             "-SkeletonHome", str(Path(SKELETON).resolve()), "-Target", str(target)],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        build = ROOT / "server-harness" / "build-war.ps1"
        result = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(build),
             "-XteaSource", str(Path(SOURCE).resolve()),
             "-RuntimeHome", str(Path(RUNTIME).resolve()),
             "-Xeaf", str(target / "Skeleton.xeaf"),
             "-Target", str(cls.work / "war")],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        classpath = (ROOT / "build" / "tomcat-classpath.txt").read_text(encoding="utf-8").strip()
        cls.endpoint = cls.work / "endpoint.json"
        cls.process = subprocess.Popen(
            ["java", "-cp", classpath, "TomcatRunner",
             "--webapp", str(cls.work / "war"), "--endpoint", str(cls.endpoint), "--port", "0"],
            cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            if cls.endpoint.exists():
                try:
                    cls.url = json.loads(cls.endpoint.read_text(encoding="utf-8"))["url"]
                    break
                except (json.JSONDecodeError, OSError):
                    pass
            if cls.process.poll() is not None:
                raise RuntimeError(f"Tomcat exited with {cls.process.returncode}")
            time.sleep(0.1)
        else:
            raise RuntimeError("Tomcat endpoint was not written")
        cls.client = ServerClient(cls.url, timeout=30)

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

    def test_war_status_and_service(self) -> None:
        status = self.client.status()
        self.assertEqual(status["status"], 200)
        self.assertIn("DB-Method", status["body"])

        value = self.client.service("AF200J", {"REQUEST_TYPE": "ListSections"})
        self.assertEqual(value["status"], 200)
        payload = json.loads(value["body"])
        self.assertEqual(payload["Status"], "OK")
        self.assertEqual(len(payload["Section"]), 2)


if __name__ == "__main__":
    unittest.main(verbosity=2)
