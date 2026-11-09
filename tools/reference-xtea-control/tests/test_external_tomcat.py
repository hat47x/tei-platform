#!/usr/bin/env python3
"""Deploy the XeadServer WAR to an already-running external Tomcat.

Required environment:
    XTEA_SERVER_SOURCE, XTEA_RUNTIME_HOME, XTEA_SKELETON_HOME
    TOMCAT_HOME  external Tomcat base with a running instance
    TOMCAT_URL   e.g. http://127.0.0.1:18080

The test uses a dedicated disposable Tomcat. It copies xteaServer.war into
webapps and removes the deployed application afterwards.
"""

from __future__ import annotations

import json
import os
import shutil
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
TOMCAT_HOME = os.environ.get("TOMCAT_HOME")
TOMCAT_URL = os.environ.get("TOMCAT_URL")


@unittest.skipUnless(all((SOURCE, RUNTIME, SKELETON, TOMCAT_HOME, TOMCAT_URL)),
                     "XTEA_SERVER_SOURCE, XTEA_RUNTIME_HOME, XTEA_SKELETON_HOME, TOMCAT_HOME and TOMCAT_URL are required")
class ExternalTomcatTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.tmp = tempfile.TemporaryDirectory(prefix="xtea-external-tomcat-")
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

        cls.tomcat = Path(TOMCAT_HOME)
        cls.war = cls.tomcat / "webapps" / "xteaServer.war"
        cls.exploded = cls.tomcat / "webapps" / "xteaServer"
        for path in (cls.war, cls.exploded):
            if path.is_dir():
                shutil.rmtree(path)
            elif path.exists():
                path.unlink()
        result = subprocess.run(
            ["jar", "cf", str(cls.war), "-C", str(cls.work / "war"), "."],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise RuntimeError(result.stderr or result.stdout)

        cls.client = ServerClient(TOMCAT_URL.rstrip("/") + "/xteaServer", timeout=30)
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            try:
                if cls.client.status()["status"] == 200:
                    break
            except Exception:
                pass
            time.sleep(1)
        else:
            raise RuntimeError("external Tomcat did not deploy xteaServer.war")

    @classmethod
    def tearDownClass(cls) -> None:
        for path in (cls.war, cls.exploded):
            try:
                if path.is_dir():
                    shutil.rmtree(path)
                elif path.exists():
                    path.unlink()
            except Exception:
                pass
        cls.tmp.cleanup()

    def test_external_tomcat(self) -> None:
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
