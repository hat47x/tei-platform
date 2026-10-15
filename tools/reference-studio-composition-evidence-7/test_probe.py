#!/usr/bin/env python3
"""Isolated tests of the probe's failure detection, NOT of TEI composition."""
import hashlib
import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("probe.py")
spec = importlib.util.spec_from_file_location("composition_probe", MODULE_PATH)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)

DRIVER = '''import hashlib, json, sys
from pathlib import Path
mode = sys.argv[1]
if mode == "regular":
    print(json.dumps({"result": {"ids": ["two", "one"], "ok": True}}))
elif mode == "composed":
    provenance = {n: hashlib.sha256(Path(n + ".bin").read_bytes()).hexdigest() for n in ("lock", "asset", "build")}
    result = {"ids": ["two", "one"], "ok": True}
    if "mismatch" in sys.argv: result["ok"] = False
    if "bad-binding" in sys.argv: provenance["lock"] = "0" * 64
    print(json.dumps({"result": result, "provenance": provenance}))
else:
    if "accept" in sys.argv: print("{}") ; sys.exit(0)
    code = mode.upper()
    if "wrong-code" in sys.argv: code = "OTHER"
    print(json.dumps({"error": {"code": code}}))
    sys.exit(3)
'''


class ProbeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "driver.py").write_text(DRIVER, encoding="utf-8")
        provenance = {}
        for kind in ("lock", "asset", "build"):
            filename = kind + ".bin"
            data = (kind + "-pinned-evidence").encode()
            (self.root / filename).write_bytes(data)
            provenance[kind] = {"path": filename, "sha256": hashlib.sha256(data).hexdigest(),
                                "composedDigestPointer": "/provenance/" + kind}
        self.probe_spec = {
            "comparison": {
                "regular": {"argv": [sys.executable, "driver.py", "regular"], "resultPointer": "/result"},
                "composed": {"argv": [sys.executable, "driver.py", "composed"], "resultPointer": "/result"},
            },
            "negative": {name: {"argv": [sys.executable, "driver.py", name],
                                "errorPointer": "/error/code", "expectedCode": name.upper()}
                         for name in probe.NEGATIVES},
            "provenance": provenance,
        }

    def test_complete_external_observation(self):
        result = probe.check_spec(self.probe_spec, self.root)
        self.assertEqual(set(result["negative"]), set(probe.NEGATIVES))
        self.assertEqual(set(result["provenance"]), {"lock", "asset", "build"})

    def test_semantic_mismatch_fails(self):
        self.probe_spec["comparison"]["composed"]["argv"].append("mismatch")
        with self.assertRaisesRegex(probe.Failed, "semantically equal"):
            probe.check_spec(self.probe_spec, self.root)

    def test_negative_must_fail_and_expose_exact_code(self):
        self.probe_spec["negative"]["duplicate"]["argv"].append("accept")
        with self.assertRaisesRegex(probe.Failed, "unexpectedly accepted"):
            probe.check_spec(self.probe_spec, self.root)
        self.probe_spec["negative"]["duplicate"]["argv"][-1] = "wrong-code"
        with self.assertRaisesRegex(probe.Failed, "wrong error classification"):
            probe.check_spec(self.probe_spec, self.root)

    def test_missing_negative_blocks(self):
        del self.probe_spec["negative"]["cycle"]
        with self.assertRaises(probe.Blocked):
            probe.check_spec(self.probe_spec, self.root)

    def test_tampered_asset_fails(self):
        (self.root / "asset.bin").write_bytes(b"changed")
        with self.assertRaisesRegex(probe.Failed, "digest differs"):
            probe.check_spec(self.probe_spec, self.root)

    def test_unbound_provenance_fails(self):
        self.probe_spec["comparison"]["composed"]["argv"].append("bad-binding")
        with self.assertRaisesRegex(probe.Failed, "not bound"):
            probe.check_spec(self.probe_spec, self.root)

    def test_missing_provenance_blocks(self):
        del self.probe_spec["provenance"]["build"]
        with self.assertRaises(probe.Blocked):
            probe.check_spec(self.probe_spec, self.root)

    def test_path_escape_blocks(self):
        self.probe_spec["provenance"]["lock"]["path"] = "../outside.bin"
        with self.assertRaises(probe.Blocked):
            probe.check_spec(self.probe_spec, self.root)

    def test_missing_executable_blocks(self):
        self.probe_spec["comparison"]["regular"]["argv"] = ["/nonexistent/studio-composer"]
        with self.assertRaises(probe.Blocked):
            probe.check_spec(self.probe_spec, self.root)

    def test_cli_statuses(self):
        spec_path = self.root / "spec.json"
        spec_path.write_text(json.dumps(self.probe_spec), encoding="utf-8")
        cmd = [sys.executable, str(MODULE_PATH), "--spec", str(spec_path)]
        result = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(json.loads(result.stdout)["status"], "PASS")
        self.probe_spec["comparison"]["composed"]["argv"].append("mismatch")
        spec_path.write_text(json.dumps(self.probe_spec), encoding="utf-8")
        result = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(result.returncode, 1, result.stdout)
        self.probe_spec["comparison"]["composed"]["argv"] = ["/nonexistent/studio-composer"]
        spec_path.write_text(json.dumps(self.probe_spec), encoding="utf-8")
        result = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(result.returncode, 2, result.stdout)


if __name__ == "__main__":
    unittest.main()
