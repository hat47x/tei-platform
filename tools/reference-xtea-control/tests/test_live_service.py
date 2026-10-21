#!/usr/bin/env python3
"""Live XeadServer Service test with a disposable Skeleton database.

Requires:
    XTEA_SERVER_SOURCE   cloned XeadServer repository
    XTEA_RUNTIME_HOME    X-TEA Driver runtime (contains xteadrv.jar, xercesImpl.jar, ...)
    XTEA_SKELETON_HOME   Skeleton directory containing Skeleton.xeaf and db/

The harness applies two explicit build-time compatibility patches:
- Session: check the `user` argument instead of the unset `userID` field.
- Service: redirect the JDK-internal XML serializer import to Xerces.
The upstream files are never modified.
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
SKELETON = os.environ.get("XTEA_SKELETON_HOME")


@unittest.skipUnless(SOURCE and RUNTIME and SKELETON,
                     "XTEA_SERVER_SOURCE, XTEA_RUNTIME_HOME and XTEA_SKELETON_HOME are required")
class LiveServiceTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.tmp = tempfile.TemporaryDirectory(prefix="xtea-service-live-")
        cls.work = Path(cls.tmp.name)
        build = ROOT / "server-harness" / "build.ps1"
        result = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(build),
             "-XteaSource", str(Path(SOURCE).resolve()), "-RuntimeHome", str(Path(RUNTIME).resolve())],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        fixture = ROOT / "driver-fixture.ps1"
        target = cls.work / "skeleton"
        result = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", str(fixture),
             "-SkeletonHome", str(Path(SKELETON).resolve()), "-Target", str(target)],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        cls.classpath = (ROOT / "build" / "server-classpath.txt").read_text(encoding="utf-8").strip()
        cls.endpoint = cls.work / "endpoint.json"
        cls.process = subprocess.Popen(
            ["java", "-cp", cls.classpath, "XteaServerHarness",
             "--xeaf", str(target / "Skeleton.xeaf"), "--endpoint", str(cls.endpoint), "--port", "0"],
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

    def test_service_status(self) -> None:
        value = self.client.service_status()
        self.assertEqual(value["status"], 200)
        self.assertIn("Session Status", value["body"])

    def test_service_function(self) -> None:
        value = self.client.service("AF200J", {"REQUEST_TYPE": "ListSections"})
        self.assertEqual(value["status"], 200)
        payload = json.loads(value["body"])
        self.assertEqual(payload["Status"], "OK")
        self.assertEqual(len(payload["Section"]), 2)
        self.assertEqual(payload["Section"][0]["CDBUMON"], "00001")


if __name__ == "__main__":
    unittest.main(verbosity=2)
