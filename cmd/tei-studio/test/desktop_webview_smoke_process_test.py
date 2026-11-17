"""Focused, platform-independent checks for the Windows smoke process snapshot."""

import json
import subprocess
import sys
import types
import unittest
from unittest.mock import patch

# The product smoke uses winreg only when probing an actual Windows runtime.
# Its process snapshot function can be tested without that runtime.
if sys.platform != "win32":
    sys.modules.setdefault("winreg", types.ModuleType("winreg"))

import desktop_webview_smoke as smoke


class ProcessSnapshotTests(unittest.TestCase):
    def test_process_snapshot_uses_bounded_timeout_and_preserves_identity(self):
        row = {
            "ProcessId": 1234,
            "ParentProcessId": 100,
            "ExecutablePath": "C:\\Example\\msedgewebview2.exe",
            "AppMarker": True,
            "RemoteDebugPort": "9222",
        }
        result = subprocess.CompletedProcess([], 0, stdout=json.dumps([row]))
        with patch.object(smoke.subprocess, "run", return_value=result) as runner:
            self.assertEqual(smoke.find_processes("msedgewebview2.exe"), [row])

        args, kwargs = runner.call_args
        self.assertEqual(args[0][:3], ["powershell.exe", "-NoProfile", "-NonInteractive"])
        self.assertIn("$name = 'msedgewebview2.exe'", args[0][-1])
        self.assertIn("Get-CimInstance Win32_Process", args[0][-1])
        self.assertTrue(kwargs["check"])
        self.assertEqual(kwargs["timeout"], smoke.PROCESS_ENUM_TIMEOUT_SECONDS)
        self.assertEqual(kwargs["timeout"], 60)

    def test_empty_snapshot_and_single_process_shape(self):
        with patch.object(smoke.subprocess, "run",
                          return_value=subprocess.CompletedProcess([], 0, stdout="null")):
            self.assertEqual(smoke.find_processes(smoke.APP_IMAGE), [])
        with patch.object(smoke.subprocess, "run",
                          return_value=subprocess.CompletedProcess([], 0, stdout='{"ProcessId":42}')):
            self.assertEqual(smoke.find_processes(smoke.APP_IMAGE), [{"ProcessId": 42}])

    def test_timeout_is_not_treated_as_no_processes(self):
        timed_out = subprocess.TimeoutExpired(
            cmd="powershell.exe", timeout=smoke.PROCESS_ENUM_TIMEOUT_SECONDS
        )
        with patch.object(smoke.subprocess, "run", side_effect=timed_out):
            with self.assertRaises(subprocess.TimeoutExpired):
                smoke.find_processes(smoke.APP_IMAGE)

    def test_powershell_failure_is_not_treated_as_no_processes(self):
        failed = subprocess.CalledProcessError(returncode=1, cmd="powershell.exe")
        with patch.object(smoke.subprocess, "run", side_effect=failed):
            with self.assertRaises(subprocess.CalledProcessError):
                smoke.find_processes(smoke.APP_IMAGE)


if __name__ == "__main__":
    unittest.main()
