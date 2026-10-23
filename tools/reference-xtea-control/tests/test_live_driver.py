#!/usr/bin/env python3
"""Real, unmodified Driver functions + original synthetic model + disposable H2.

XTEA_RUNTIME_HOME supplies xteadrv.jar and its distribution dependencies.
JAVA selects JDK 11 (Nashorn); JAVAC selects a compiler supporting --release 8.
"""
from __future__ import annotations

import hashlib
from contextlib import contextmanager
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
import xml.etree.ElementTree as ET
import zipfile

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / "tests"))
from driver_fixture import write_fixture
from xtea_control import ControlClient, ControlError


def new_run_folder():
    parent = ROOT / "artifacts/live-driver"
    parent.mkdir(parents=True, exist_ok=True)
    return Path(tempfile.mkdtemp(prefix="run-", dir=parent))


def write_run_summary(result, folder):
    status = ("failed" if not result.wasSuccessful() else
              "skipped" if result.skipped or result.expectedFailures or not result.testsRun else "passed")
    (folder / "summary.json").write_text(json.dumps({
        "status": status, "testsRun": result.testsRun,
        "failures": [{"test": test.id(), "traceback": trace} for test, trace in result.failures],
        "errors": [{"test": test.id(), "traceback": trace} for test, trace in result.errors],
        "skipped": [{"test": test.id(), "reason": reason} for test, reason in result.skipped],
        "expectedFailures": [{"test": test.id(), "traceback": trace} for test, trace in result.expectedFailures],
        "unexpectedSuccesses": [test.id() for test in result.unexpectedSuccesses],
    }, ensure_ascii=False, indent=2), encoding="utf-8")


class LiveDriverTest(unittest.TestCase):
    STARTUP_TIMEOUT_SECONDS = 120
    OUTPUT_TIMEOUT_SECONDS = 120

    @contextmanager
    def failure_diagnostic(self, context):
        try:
            yield
        except unittest.SkipTest:
            raise
        except Exception:
            try:
                capture = {"test": self.id(), "context": context}
                for name, read in (("snapshot", lambda: self.client.snapshot(includeTableRows=True)),
                                   ("trace", self.client.trace)):
                    try:
                        capture[name] = read()
                    except Exception as error:
                        capture[name + "Error"] = f"{type(error).__name__}: {error}"
                path = self.artifacts / f"{self._testMethodName}-failure-{time.time_ns()}.json"
                path.write_text(json.dumps(capture, ensure_ascii=False, indent=2, default=str), encoding="utf-8")
            except Exception as error:
                print(f"Failure diagnostic unavailable: {type(error).__name__}", file=sys.stderr)
            raise

    def _callTestMethod(self, method):
        # Capture before unittest runs tearDown/cleanups, including on Python 3.10.
        with self.failure_diagnostic({"phase": "test"}):
            return super()._callTestMethod(method)

    @contextmanager
    def subTest(self, *args, **params):
        with super().subTest(*args, **params):
            with self.failure_diagnostic({"args": args, "params": params}):
                yield

    @classmethod
    def setUpClass(cls):
        cls.artifacts = new_run_folder()
        runtime = os.environ.get("XTEA_RUNTIME_HOME")
        if not runtime:
            raise unittest.SkipTest("XTEA_RUNTIME_HOME is not set; real Driver was not tested")
        cls.runtime = Path(runtime).resolve()
        for name in ("xteadrv.jar", "h2-1.4.187.jar"):
            if not (cls.runtime / name).is_file():
                raise RuntimeError(f"missing external runtime dependency: {name}")
        cls.java = os.environ.get("JAVA", "java")
        cls.agent = ROOT / "build/xtea-control-agent.jar"
        if not cls.agent.is_file():
            raise RuntimeError("build the agent first with build.sh or build.cmd")
        cls.classes = ROOT / "build/driver-classes"
        cls.classes.mkdir(parents=True, exist_ok=True)
        cls.classpath = os.pathsep.join([str(cls.runtime / "*"), str(cls.classes)])
        subprocess.run([os.environ.get("JAVAC", "javac"), "--release", "8", "-encoding", "UTF-8",
                        "-cp", cls.classpath, "-d", str(cls.classes),
                        str(ROOT / "tests/fixtures/DriverFixture.java")], check=True, timeout=60)
        fingerprint = {p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                       for p in sorted(cls.runtime.glob("*.jar"))}
        (cls.artifacts / "runtime.json").write_text(json.dumps({
            "startedAtUtc": datetime.now(timezone.utc).isoformat(),
            "display": os.environ.get("DISPLAY"),
            "startupTimeoutSeconds": cls.STARTUP_TIMEOUT_SECONDS,
            "outputTimeoutSeconds": cls.OUTPUT_TIMEOUT_SECONDS,
            "java": subprocess.run([cls.java, "-version"], capture_output=True, text=True,
                                   check=True).stderr,
            "jars": fingerprint,
            "agentSha256": hashlib.sha256(cls.agent.read_bytes()).hexdigest(),
            "sourceSha256": {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest()
                             for p in [Path(__file__).resolve(), ROOT / "tests/driver_fixture.py",
                                       ROOT / "tests/fixtures/DriverFixture.java", ROOT / "xtea_control.py",
                                       *sorted((ROOT / "agent/src").rglob("*.java"))]},
            "scope": "synthetic model; original Driver FunctionLauncher; no external viewer",
        }, indent=2), encoding="utf-8")

    def setUp(self):
        # Repeating a test ID in one run must not overwrite an earlier attempt's logs.
        self.artifacts = Path(tempfile.mkdtemp(prefix="case-", dir=type(self).artifacts))
        self.tmp = tempfile.TemporaryDirectory(prefix="driver-", dir=ROOT / "build")
        self.addCleanup(self.tmp.cleanup)
        self.folder = Path(self.tmp.name)
        write_fixture(self.folder)
        endpoint = self.folder / "endpoint.json"
        self.log = open(self.artifacts / (self._testMethodName + ".log"), "w", encoding="utf-8")
        self.addCleanup(self.log.close)
        self.process = subprocess.Popen([
            self.java, f"-javaagent:{self.agent}=tokenFile={endpoint}",
            "-cp", self.classpath, "DriverFixture", str(self.folder)],
            stdout=self.log, stderr=self.log)
        self.addCleanup(self.stop_process)
        started = time.monotonic()
        deadline = started + self.STARTUP_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                self.fail(f"Driver exited with {self.process.returncode}; see {self.log.name}")
            if endpoint.exists() and (self.folder / "ready").exists():
                try:
                    self.client = ControlClient.from_endpoint_file(endpoint)
                    if self.client.find({"name": "result"}):
                        break
                except (ValueError, OSError):
                    pass
            time.sleep(0.1)
        else:
            self.fail(f"Driver launcher did not appear within {self.STARTUP_TIMEOUT_SECONDS}s "
                      f"(endpoint={endpoint.exists()}, ready={(self.folder / 'ready').exists()}); "
                      f"see {self.log.name}")
        self.startup_seconds = time.monotonic() - started
        self.observations = []
        self.outputs = []
        self.addCleanup(self.record_evidence)
        self.observe()

    def stop_process(self):
        try:
            if hasattr(self, "client"):
                self.client.shutdown(exit_process=True)
            self.process.wait(timeout=5)
        except Exception:
            self.process.kill()
            self.process.wait(timeout=5)

    def record_evidence(self):
        evidence = {"health": self.client.health(), "observations": self.observations, "outputs": self.outputs,
                    "startupSeconds": self.startup_seconds,
                    "snapshot": self.client.snapshot(),
                    "modelSha256": hashlib.sha256((self.folder / "fixture.xeaf").read_bytes()).hexdigest(),
                    "trace": self.client.trace()}
        (self.artifacts / (self._testMethodName + ".json")).write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2), encoding="utf-8")

    def observe(self):
        self.client.invoke({"window": "^Driver fixture launcher$", "role": "button", "text": "^Observe DB$"})
        text = self.client.find({"window": "^Driver fixture launcher$", "name": "result"})["text"]
        self.observations.append(text)
        self.assertNotIn("ERROR", text)
        return text

    def open_function(self, typ, mode="DISPLAY"):
        self.client.invoke({"window": "^Driver fixture launcher$", "role": "button",
                            "text": f"^F{typ} {mode}$"})

    def button(self, window_class, caption):
        self.client.invoke({"windowClass": window_class + "$", "role": "button", "text": caption})

    def wait_class(self, cls, state="present"):
        self.assertTrue(self.client.wait_selector({"windowClass": cls + "$", "role": "text"},
                                                 state=state)["matched"])

    def result(self, contains):
        self.client.wait_text({"window": "^Driver fixture launcher$", "name": "result"}, contains=contains)

    def test_000_script_launcher(self):
        self.open_function("000")
        self.result("CONTROL_RESULT=ran")
        self.assertEqual(self.observe(), self.observations[0])

    def script_transaction(self, answer, expected):
        self.script("TRANSACTION")
        confirmation = {"window": "^Script transaction$", "role": "button", "text": "^" + answer + "$"}
        self.assertTrue(self.client.wait_selector(confirmation)["matched"])
        self.client.invoke(confirmation)
        # The same transaction must see the UPDATE before either decision.
        self.result("UI_RESULT=1:Pending:" + expected)

    def test_000_script_commit(self):
        self.script_transaction("Yes", "Pending")
        observed = self.observe()  # Independent JDBC connection after commit.
        self.assertIn("001:Pending:1;", observed)
        self.assertIn("002:Beta:0;", observed)

    def test_000_script_rollback(self):
        self.script_transaction("No", "Alpha")
        self.assertEqual(self.observe(), self.observations[0])

    def test_000_console_run_and_cancel(self):
        console = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextArea$"}
        self.open_function("001")
        self.client.wait_text(console, contains="console-ready")
        locked = {"windowClass": "xeadDriver.XF000$", "role": "checkbox", "text": "^Run right now$"}
        with self.assertRaises(ControlError) as caught:
            self.client.set_value(locked, False)
        self.assertEqual(caught.exception.code, "COMPONENT_NOT_EDITABLE")
        self.assertTrue(self.client.find(locked)["selected"])
        self.button("xeadDriver.XF000", "^Exit$")
        self.result("RETURN_CODE=01")
        self.assertEqual(self.observe(), self.observations[0])
        self.open_function("001")
        self.client.wait_text(console, contains="console-ready")
        self.assertFalse(self.client.find({"windowClass": "xeadDriver.XF000$", "role": "text",
                                           "class": "JTextField$"})["enabled"])
        self.button("xeadDriver.XF000", "^Start$")
        self.client.wait_text(console, contains="console-finished")
        self.assertFalse(self.client.find({"windowClass": "xeadDriver.XF000$", "role": "button", "text": "^Start$"})["enabled"])
        self.button("xeadDriver.XF000", "^Exit$")
        self.result("CONTROL_RESULT=console")
        self.assertIn("001:Console:1;002:Beta:0;", self.observe())

    def test_000_clock_start_stop(self):
        self.open_function("002")
        console = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextArea$"}
        clock = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextField$"}
        self.client.wait_text(console, contains="timer-ready")
        self.client.set_value(clock, "invalid")
        self.button("xeadDriver.XF000", "^Start$")
        error = {"windowClass": "^javax.swing.JDialog$", "role": "button", "text": "^OK$"}
        self.assertTrue(self.client.wait_selector(error)["matched"])
        self.client.invoke(error)
        # Read a future time prepared with the same JVM clock used by the original scheduler.
        future = self.client.find({"window": "^Driver fixture launcher$", "name": "futureTime"})["text"]
        self.client.set_value(clock, future)
        self.client.invoke({"windowClass": "xeadDriver.XF000$", "role": "checkbox", "text": "^Run on day off$"})
        self.button("xeadDriver.XF000", "^Start$")
        self.client.wait_text(console, contains="is the time to execute")
        self.assertFalse(self.client.find(clock)["enabled"])
        self.assertTrue(self.client.find({"windowClass": "xeadDriver.XF000$", "role": "button", "text": "^Stop$"})["enabled"])
        self.button("xeadDriver.XF000", "^Stop$")
        self.client.wait_text(console, contains="The timer is canceled")
        self.assertTrue(self.client.find(clock)["enabled"])
        self.button("xeadDriver.XF000", "^Exit$")
        self.wait_class("xeadDriver.XF000", "absent")
        self.assertEqual(self.observe(), self.observations[0])

    def clock_execution(self, repeat):
        launcher = {"window": "^Driver fixture launcher$"}
        self.client.invoke(dict(launcher, role="button", text="^Prepare next clock$"))
        requested_time = self.client.find(dict(launcher, name="nextTime"))["text"]
        self.open_function("002")
        console = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextArea$"}
        clock = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextField$"}
        self.client.wait_text(console, contains="timer-ready")
        self.client.set_value(clock, requested_time)
        if not repeat:
            self.client.invoke({"windowClass": "xeadDriver.XF000$", "role": "checkbox", "text": "^Repeat in time$"})
        self.client.invoke({"windowClass": "xeadDriver.XF000$", "role": "checkbox", "text": "^Run on day off$"})
        started = time.monotonic()
        self.button("xeadDriver.XF000", "^Start$")
        self.client.wait_text(console, contains="is the time to execute")
        self.assertEqual(self.observe(), self.observations[0])
        self.client.wait_text(console, contains="timer-finished", timeout_ms=120000)
        if repeat:
            deadline = time.monotonic() + 10
            while time.monotonic() < deadline:
                if self.client.find(console)["textValue"].count("is the time to execute") == 2:
                    break
                time.sleep(0.05)
            self.assertEqual(self.client.find(console)["textValue"].count("is the time to execute"), 2)
            self.button("xeadDriver.XF000", "^Stop$")
        self.client.wait_text(console, contains="The timer is canceled")
        self.outputs.append({"kind": "clockExecution", "requestedTime": requested_time, "repeat": repeat,
                             "elapsedSeconds": time.monotonic() - started,
                             "console": self.client.find(console)["textValue"]})
        self.assertTrue(self.client.find(clock)["enabled"])
        self.assertFalse(self.client.find({"windowClass": "xeadDriver.XF000$", "role": "button", "text": "^Stop$"})["enabled"])
        self.button("xeadDriver.XF000", "^Exit$")
        self.result("CONTROL_RESULT=timer")
        self.assertIn("001:Timer:1;002:Beta:0;", self.observe())

    def test_000_clock_fires_once(self):
        self.clock_execution(repeat=False)

    def test_000_clock_fires_and_rearms(self):
        self.clock_execution(repeat=True)

    def test_000_clock_run_now(self):
        self.open_function("002")
        console = {"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextArea$"}
        self.client.wait_text(console, contains="timer-ready")
        self.client.invoke({"windowClass": "xeadDriver.XF000$", "role": "checkbox", "text": "^Run right now$"})
        self.assertFalse(self.client.find({"windowClass": "xeadDriver.XF000$", "role": "text", "class": "JTextField$"})["enabled"])
        self.button("xeadDriver.XF000", "^Start$")
        self.client.wait_text(console, contains="timer-finished")
        self.button("xeadDriver.XF000", "^Exit$")
        self.result("CONTROL_RESULT=timer")
        self.assertIn("001:Timer:1;002:Beta:0;", self.observe())

    def test_100_search_select_detail(self):
        self.open_function("100")
        self.wait_class("xeadDriver.XF100")
        table = {"windowClass": "xeadDriver.XF100$", "role": "table"}
        self.assertEqual(self.client.table_read(table)["rowCount"], 2)
        self.client.set_value({"windowClass": "xeadDriver.XF100$", "role": "text",
                               "class": "JTextField$"}, "Alpha")
        self.button("xeadDriver.XF100", "List|Search")
        self.assertEqual(self.client.table_read(table)["rowCount"], 1)
        self.client.select(table, row=0)
        self.client.key(table, key="ENTER")
        self.wait_class("xeadDriver.XF200")
        self.assertIsNotNone(self.client.find({"windowClass": "xeadDriver.XF200$", "text": "^Alpha$"}))
        self.button("xeadDriver.XF200", "Close")
        self.button("xeadDriver.XF100", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_110_select_edit_commit(self):
        self.open_function("110")
        self.wait_class("xeadDriver.XF110")
        table = {"windowClass": "xeadDriver.XF110$", "role": "table"}
        self.client.select(table, row=0)
        self.client.key(table, key="SPACE")
        self.button("xeadDriver.XF110", "Next")
        self.wait_class("xeadDriver.XF110_SubList")
        editor = {"windowClass": "xeadDriver.XF110_SubList$", "role": "table"}
        self.client.table_edit(editor, row=0)
        self.client.set_value({"windowClass": "xeadDriver.XF110_SubList$", "role": "text", "text": "^Alpha$"}, "Bulk")
        self.client.table_edit(editor, "commit")
        self.assertIn("Bulk", self.client.table_read(editor)["rows"][0])
        self.button("xeadDriver.XF110_SubList", "Save")
        self.wait_class("xeadDriver.XF110", "absent")
        observed = self.observe()
        self.assertIn("001:Bulk:1;", observed)
        self.assertIn("002:Beta:0;", observed)

    def test_200_update_cancel_insert_delete(self):
        self.open_function("200", "EDIT")
        self.wait_class("xeadDriver.XF200")
        field = {"windowClass": "xeadDriver.XF200$", "role": "text", "text": "^Alpha$"}
        self.client.set_value(field, "Edited")
        self.button("xeadDriver.XF200", "Save")
        self.wait_class("xeadDriver.XF200", "absent")
        self.assertIn("001:Edited:1;", self.observe())
        self.open_function("200", "EDIT")
        self.wait_class("xeadDriver.XF200")
        self.client.set_value(dict(field, text="^Edited$"), "Discarded")
        self.button("xeadDriver.XF200", "Close")
        self.wait_class("xeadDriver.XF200", "absent")
        self.assertIn("001:Edited:1;", self.observe())
        self.open_function("200", "ADD")
        self.wait_class("xeadDriver.XF200")
        self.client.set_value({"windowClass": "xeadDriver.XF200$", "role": "text", "class": "JTextField$"}, "003")
        self.client.set_value({"windowClass": "xeadDriver.XF200$", "role": "text", "class": "JTextField$", "index": 1}, "New")
        self.button("xeadDriver.XF200", "Add|Save")
        self.wait_class("xeadDriver.XF200", "absent")
        self.assertIn("003:New:0;", self.observe())
        self.open_function("200")
        self.wait_class("xeadDriver.XF200")
        self.button("xeadDriver.XF200", "Delete")
        self.assertTrue(self.client.wait_selector({"role": "button", "text": "^Yes$"})["matched"])
        self.client.invoke({"role": "button", "text": "^Yes$"})
        self.wait_class("xeadDriver.XF200", "absent")
        self.assertNotIn("001:", self.observe().split("L:")[0])

    def test_200_copy(self):
        self.open_function("200")
        self.wait_class("xeadDriver.XF200")
        self.button("xeadDriver.XF200", "Copy")
        self.client.set_value({"windowClass": "xeadDriver.XF200$", "role": "text", "text": "^001$"}, "003")
        self.button("xeadDriver.XF200", "Add")
        self.wait_class("xeadDriver.XF200", "absent")
        observed = self.observe()
        self.assertIn("001:Alpha:0;", observed)
        self.assertIn("003:Alpha:0;", observed)

    def test_110_cancel_update(self):
        self.open_function("110")
        self.wait_class("xeadDriver.XF110")
        selector = {"windowClass": "xeadDriver.XF110$", "role": "table"}
        self.client.select(selector, row=0)
        self.client.key(selector, key="SPACE")
        self.button("xeadDriver.XF110", "Next")
        self.wait_class("xeadDriver.XF110_SubList")
        editor = {"windowClass": "xeadDriver.XF110_SubList$", "role": "table"}
        self.client.table_edit(editor, row=0)
        self.client.set_value({"windowClass": "xeadDriver.XF110_SubList$", "role": "text", "text": "^Alpha$"}, "Discarded")
        self.client.table_edit(editor, "commit")
        self.button("xeadDriver.XF110_SubList", "Close")
        self.wait_class("xeadDriver.XF110_SubList", "absent")
        if self.client.find({"windowClass": "xeadDriver.XF110$", "role": "table"}):
            self.button("xeadDriver.XF110", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_100_excel(self):
        self.open_function("100")
        self.wait_class("xeadDriver.XF100")
        self.check_excel("xeadDriver.XF100", ["001", "Alpha", "002", "Beta"])
        self.button("xeadDriver.XF100", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_110_excel_both_lists(self):
        self.open_function("110")
        self.wait_class("xeadDriver.XF110")
        self.check_excel("xeadDriver.XF110", ["001", "Alpha", "002", "Beta"])
        table = {"windowClass": "xeadDriver.XF110$", "role": "table"}
        self.client.select(table, row=0)
        self.client.key(table, key="SPACE")
        self.button("xeadDriver.XF110", "Next")
        self.wait_class("xeadDriver.XF110_SubList")
        self.check_excel("xeadDriver.XF110_SubList", ["001", "Alpha"], absent=["Beta"])
        self.button("xeadDriver.XF110_SubList", "Close")
        self.wait_class("xeadDriver.XF110_SubList", "absent")
        if self.client.find({"windowClass": "xeadDriver.XF110$", "role": "table"}):
            self.button("xeadDriver.XF110", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_200_excel(self):
        self.open_function("200")
        self.wait_class("xeadDriver.XF200")
        self.check_excel("xeadDriver.XF200", ["001", "Alpha"], absent=["Beta"])
        self.button("xeadDriver.XF200", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_300_excel(self):
        self.open_function("300")
        self.wait_class("xeadDriver.XF300")
        self.check_excel("xeadDriver.XF300", ["Alpha", "First", "Second"])
        self.button("xeadDriver.XF300", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_310_excel(self):
        self.open_function("310")
        self.wait_class("xeadDriver.XF310")
        self.check_excel("xeadDriver.XF310", ["Alpha", "First", "Second"])
        self.button("xeadDriver.XF310", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def check_excel(self, window_class, expected, absent=()):
        ready = self.folder / "output-ready"
        ready.unlink(missing_ok=True)
        started = time.monotonic()
        self.button(window_class, "Excel")
        deadline = started + self.OUTPUT_TIMEOUT_SECONDS
        while not ready.exists() and time.monotonic() < deadline:
            time.sleep(0.05)
        self.assertTrue(ready.exists(), f"Driver did not finish the output callback within {self.OUTPUT_TIMEOUT_SECONDS}s")
        completion_seconds = time.monotonic() - started
        output = self.folder / ready.read_text(encoding="utf-8")
        self.assertEqual(output.suffix, ".xlsx")
        with zipfile.ZipFile(output) as archive:
            ns = {"s": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
            strings = ET.fromstring(archive.read("xl/sharedStrings.xml"))
            shared = ["".join(si.itertext()) for si in strings.findall("s:si", ns)]
            sheet = ET.fromstring(archive.read("xl/worksheets/sheet1.xml"))
            values = []
            for cell in sheet.findall(".//s:c", ns):
                value = cell.find("s:v", ns)
                if value is not None and value.text is not None:
                    values.append(shared[int(value.text)] if cell.get("t") == "s" else value.text)
            for value in expected:
                self.assertIn(value, values)
            for value in absent:
                self.assertNotIn(value, values)
        data = output.read_bytes()
        name = self._testMethodName + "-" + window_class.rsplit(".", 1)[-1] + ".xlsx"
        (self.artifacts / name).write_bytes(data)
        self.outputs.append({"file": name, "sha256": hashlib.sha256(data).hexdigest(),
                             "bytes": len(data), "cellValues": values, "completionSeconds": completion_seconds})

    def test_290_single_record_pdf(self):
        self.check_pdf("290")

    def test_300_header_detail(self):
        self.open_function("300")
        self.wait_class("xeadDriver.XF300")
        self.assertIsNotNone(self.client.find({"windowClass": "xeadDriver.XF300$", "text": "^Alpha$"}))
        table = self.client.table_read({"windowClass": "xeadDriver.XF300$", "role": "table"})
        self.assertEqual(table["rowCount"], 2)
        self.assertIn("First", table["rows"][0])
        self.button("xeadDriver.XF300", "Close")
        self.assertEqual(self.observe(), self.observations[0])

    def test_310_detail_edit_commit(self):
        self.open_function("310")
        self.wait_class("xeadDriver.XF310")
        table = {"windowClass": "xeadDriver.XF310$", "role": "table"}
        with self.assertRaises(ControlError) as caught:
            self.client.table_set(table, 0, 0, "would replace the row object")
        self.assertEqual(caught.exception.code, "INVALID_ARGUMENT")
        self.client.table_edit(table, row=0)
        self.client.set_value({"windowClass": "xeadDriver.XF310$", "role": "text", "text": "^First$"}, "Changed")
        self.client.table_edit(table, "commit")
        self.assertIn("Changed", self.client.table_read(table)["rows"][0])
        self.button("xeadDriver.XF310", "Save")
        self.wait_class("xeadDriver.XF310", "absent")
        self.assertIn("L:001:1:Changed;", self.observe())

    def test_390_header_detail_pdf(self):
        self.check_pdf("390")

    def test_310_long_text_dialog(self):
        self.open_function("311")
        self.wait_class("xeadDriver.XF310")
        table = {"windowClass": "xeadDriver.XF310$", "role": "table"}
        self.client.table_edit(table, row=0)
        with self.assertRaises(ControlError) as caught:
            self.client.set_value({"windowClass": "xeadDriver.XF310$", "role": "text", "text": "^Note$"}, "Bypass")
        self.assertEqual(caught.exception.code, "COMPONENT_NOT_EDITABLE")
        self.client.invoke({"windowClass": "xeadDriver.XF310$", "role": "button",
                            "class": "^javax.swing.JButton$", "text": "^$"})
        editor = {"windowClass": "xeadDriver.XFLongTextEditor$", "role": "text"}
        self.assertTrue(self.client.wait_selector(editor)["matched"])
        self.client.set_value(editor, "Long edited")
        self.client.close(editor)
        self.client.table_edit(table, "commit")
        self.button("xeadDriver.XF310", "Save")
        self.wait_class("xeadDriver.XF310", "absent")
        self.assertIn("N:1:Long edited;", self.observe())

    def test_310_add_remove_cancel(self):
        self.open_function("310")
        self.wait_class("xeadDriver.XF310")
        table = {"windowClass": "xeadDriver.XF310$", "role": "table"}
        self.button("xeadDriver.XF310", "Add row")
        prompt = {"windowClass": "javax.swing.JDialog$", "role": "text"}
        self.assertTrue(self.client.wait_selector(prompt)["matched"])
        self.client.set_value(prompt, "3")
        self.button("javax.swing.JDialog", "^OK$")
        self.assertEqual(self.client.table_read(table)["rowCount"], 3)
        self.client.table_edit(table, row=2)
        self.client.set_value({"windowClass": "xeadDriver.XF310$", "role": "text", "text": "^$"}, "Third")
        self.client.table_edit(table, "commit")
        self.button("xeadDriver.XF310", "Save")
        self.wait_class("xeadDriver.XF310", "absent")
        self.assertIn("L:001:3:Third;", self.observe())
        self.open_function("310")
        self.wait_class("xeadDriver.XF310")
        self.client.table_edit(table, row=1)
        self.button("xeadDriver.XF310", "Remove row")
        self.assertTrue(self.client.wait_selector({"role": "button", "text": "^Yes$"})["matched"])
        self.client.invoke({"role": "button", "text": "^Yes$"})
        self.button("xeadDriver.XF310", "Save")
        self.wait_class("xeadDriver.XF310", "absent")
        before_cancel = self.observe()
        self.assertNotIn("L:001:2:", before_cancel)
        self.assertIn("L:001:3:Third;", before_cancel)
        self.open_function("310")
        self.wait_class("xeadDriver.XF310")
        self.client.table_edit(table, row=0)
        self.client.set_value({"windowClass": "xeadDriver.XF310$", "role": "text", "text": "^First$"}, "Discarded")
        self.client.table_edit(table, "commit")
        self.button("xeadDriver.XF310", "Close")
        self.wait_class("xeadDriver.XF310", "absent")
        self.assertEqual(self.observe(), before_cancel)

    def check_pdf(self, typ):
        self.open_function(typ)
        self.result("FILE_NAME=")
        self.assertIn("RETURN_CODE=00", self.client.find({"name": "result"})["text"])
        files = list(self.folder.glob(f"*F{typ}*.pdf"))
        self.assertEqual(len(files), 1)
        data = files[0].read_bytes()
        self.assertTrue(data.startswith(b"%PDF-"))
        self.assertIn(b"%%EOF", data)
        self.assertGreater(len(data), 500)
        output = self.artifacts / (self._testMethodName + ".pdf")
        output.write_bytes(data)
        self.outputs.append({"file": output.name, "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)})
        self.assertEqual(self.observe(), self.observations[0])

    def script(self, name):
        self.client.invoke({"window": "^Driver fixture launcher$", "role": "button",
                            "text": f"^S{name} DISPLAY$"})

    def test_script_input_accept_and_cancel(self):
        self.script("INPUT")
        selector = {"windowClass": "xeadDriver.XFInputDialog$", "role": "text", "text": "^initial$"}
        self.assertTrue(self.client.wait_selector(selector)["matched"])
        self.client.set_value(selector, "Entered")
        self.button("xeadDriver.XFInputDialog", "OK")
        self.result("UI_RESULT=0:Entered")
        self.script("INPUT")
        self.assertTrue(self.client.wait_selector(selector)["matched"])
        self.button("xeadDriver.XFInputDialog", "Close")
        self.result("UI_RESULT=-1:")
        self.assertEqual(self.observe(), self.observations[0])

    def test_script_option_and_checklist(self):
        self.script("OPTION")
        selector = {"windowClass": "xeadDriver.XFOptionDialog$", "role": "list"}
        self.assertTrue(self.client.wait_selector(selector)["matched"])
        self.client.select(selector, item_index=1)
        self.client.key(selector, key="ENTER")
        self.result("UI_RESULT=1")
        self.script("CHECK")
        selector = {"windowClass": "xeadDriver.XFCheckListDialog$", "role": "list"}
        self.assertTrue(self.client.wait_selector(selector)["matched"])
        self.client.select(selector, item_index=1)
        self.client.key(selector, key="SPACE")
        self.button("xeadDriver.XFCheckListDialog", "OK")
        self.result("UI_RESULT=0:B")
        self.assertEqual(self.observe(), self.observations[0])

    def test_script_input_types_used_by_reference_models(self):
        for typ, value, expected in (("ALPHA", "Typed", "Typed"), ("TEXTAREA", "Lines", "Lines"),
                                      ("NUMERIC", "123", "123"), ("ZEROFILL", "12", "00012"),
                                      ("CHECKBOX", True, "true"), ("LISTBOX", "Second", "B"),
                                      ("DATE", None, "2026-10-22")):
            with self.subTest(input_type=typ):
                self.script("TYPE_" + typ)
                self.wait_class("xeadDriver.XFInputDialog")
                selector = {"windowClass": "xeadDriver.XFInputDialog$"}
                if typ == "DATE":
                    self.client.invoke(dict(selector, role="button", text="^$", **{"class": "^javax.swing.JButton$"}))
                    day = {"windowClass": "xeadDriver.XFCalendar$", "role": "button", "text": "^22$"}
                    self.assertTrue(self.client.wait_selector(day)["matched"])
                    self.client.invoke(day)
                elif typ == "CHECKBOX":
                    self.client.set_value(dict(selector, role="checkbox"), value)
                elif typ == "LISTBOX":
                    self.client.set_value(dict(selector, role="combobox"), value)
                else:
                    field = dict(selector, role="text", **{"class": "JTextArea$" if typ == "TEXTAREA" else "JTextField$"})
                    if typ == "TEXTAREA":
                        field["text"] = "^initi$"
                    self.client.set_value(field, value)
                self.button("xeadDriver.XFInputDialog", "OK")
                self.result("UI_RESULT=0:" + expected)
        self.assertEqual(self.observe(), self.observations[0])

    def test_script_swing_message_confirm_prompt_choice(self):
        for name, caption, value in (("MESSAGE", "^OK$", "seen"), ("CONFIRM", "^Yes$", "0"),
                                      ("PROMPT", "^OK$", "Typed"), ("CHOICE", "^Second$", "1")):
            with self.subTest(name=name):
                self.script(name)
                self.assertTrue(self.client.wait_selector({"windowClass": "javax.swing.JDialog$",
                                                           "role": "button", "text": caption})["matched"])
                if name == "PROMPT":
                    self.client.set_value({"windowClass": "javax.swing.JDialog$", "role": "text"}, "Typed")
                self.button("javax.swing.JDialog", caption)
                self.result("UI_RESULT=" + value)
        self.assertEqual(self.observe(), self.observations[0])


if __name__ == "__main__":
    result = unittest.main(verbosity=2, exit=False).result
    folder = getattr(LiveDriverTest, "artifacts", None) or new_run_folder()
    write_run_summary(result, folder)
    print("Driver run artifacts:", folder)
    sys.exit(not result.wasSuccessful())
