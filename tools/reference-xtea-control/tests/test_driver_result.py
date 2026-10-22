"""A skipped or failed runtime check must never be reported as passed."""
import json
from pathlib import Path
import tempfile
import unittest

from test_live_driver import write_run_summary


class DriverResultTest(unittest.TestCase):
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
                write_run_summary(result, Path(tmp))
                summary = json.loads((Path(tmp) / "summary.json").read_text())
                self.assertEqual(summary["status"], expected)
                self.assertEqual(summary["testsRun"], 0 if name == "empty" else 1)
                if name == "fail":
                    self.assertIn("field='ALPHA'", summary["failures"][0]["test"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
