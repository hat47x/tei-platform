"""A skipped or failed runtime check must never be reported as passed."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import test_live_driver as driver_tests
from test_live_driver import write_run_summary


class DriverResultTest(unittest.TestCase):
    def test_failures_are_captured_before_state_changes(self):
        class Client:
            value = "failed state"
            broken = False

            def snapshot(self, **options):
                if self.broken:
                    raise RuntimeError("snapshot unavailable")
                return {"value": self.value}

            def trace(self):
                return {"events": []}

        class Fixture(driver_tests.LiveDriverTest):
            client = Client()

            def setUp(self):
                pass

            def test_flow(self):
                with self.subTest(input_type="ALPHA"):
                    self.fail("input missing")
                self.client.value = "next state"

            def test_parent_failure(self):
                self.fail("input missing")

            def test_broken_snapshot(self):
                self.client.broken = True
                self.fail("original failure")

            def tearDown(self):
                self.client.value = "cleanup state"

        for name in ("test_flow", "test_parent_failure", "test_broken_snapshot"):
            with self.subTest(test_method=name), tempfile.TemporaryDirectory() as tmp:
                Fixture.artifacts = Path(tmp)
                Fixture.client = Client()
                result = unittest.TestResult()
                Fixture(name).run(result)
                self.assertEqual(len(result.failures), 1)
                capture = json.loads(next(Path(tmp).glob("*-failure-*.json")).read_text())
                if name == "test_broken_snapshot":
                    self.assertIn("snapshot unavailable", capture["snapshotError"])
                    self.assertEqual(capture["trace"]["events"], [])
                    self.assertIn("original failure", result.failures[0][1])
                    continue
                self.assertEqual(capture["snapshot"]["value"], "failed state")
                if name == "test_flow":
                    self.assertEqual(capture["context"]["params"]["input_type"], "ALPHA")
                else:
                    self.assertEqual(capture["context"]["phase"], "test")

    def test_actual_unittest_outcomes(self):
        class Fixture(unittest.TestCase):
            def test_pass(self):
                pass

            def test_fail(self):
                with self.subTest(field="ALPHA"):
                    self.fail("input missing")

            def test_error(self):
                raise RuntimeError("input missing")

            @unittest.skip("runtime unavailable")
            def test_skip(self):
                pass

            @unittest.expectedFailure
            def test_expected_failure(self):
                self.fail("known failure")

            @unittest.expectedFailure
            def test_unexpected_success(self):
                pass

        for name, expected in (("pass", "passed"), ("fail", "failed"),
                               ("error", "failed"), ("skip", "skipped"), ("empty", "skipped"),
                               ("expected_failure", "skipped"), ("unexpected_success", "failed")):
            with self.subTest(outcome=name), tempfile.TemporaryDirectory() as tmp:
                result = unittest.TestResult()
                if name != "empty":
                    Fixture("test_" + name).run(result)
                written_status = write_run_summary(result, Path(tmp))
                summary = json.loads((Path(tmp) / "summary.json").read_text())
                self.assertEqual(written_status, expected)
                self.assertEqual(summary["status"], expected)
                self.assertEqual(summary["testsRun"], 0 if name == "empty" else 1)
                if name == "fail":
                    self.assertIn("field='ALPHA'", summary["failures"][0]["test"])

    def test_skipped_runtime_exits_nonzero(self):
        # A missing Driver runtime must not look like a successful verification.
        env = os.environ.copy()
        env.pop("XTEA_RUNTIME_HOME", None)
        script = Path(driver_tests.__file__).resolve()
        completed = subprocess.run(
            [sys.executable, str(script), "LiveDriverTest.test_000_script_launcher"],
            cwd=script.parent.parent, env=env, capture_output=True, text=True,
            check=False,
        )
        self.assertNotEqual(completed.returncode, 0, completed.stdout + completed.stderr)
        marker = "Driver run artifacts:"
        self.assertIn(marker, completed.stdout)
        run_folder = Path(completed.stdout.split(marker, 1)[1].strip().splitlines()[0])
        summary = json.loads((run_folder / "summary.json").read_text(encoding="utf-8"))
        self.assertEqual(summary["status"], "skipped")
        self.assertTrue(summary["skipped"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
