#!/usr/bin/env python3
"""Real, unmodified Driver functions + original synthetic model + disposable H2.

XTEA_RUNTIME_HOME supplies xteadrv.jar and its distribution dependencies.
JAVA selects JDK 11 (Nashorn); JAVAC selects a compiler supporting --release 8.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / "tests"))
from driver_fixture import write_fixture
from xtea_control import ControlClient, ControlError


class LiveDriverTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
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
        cls.artifacts = ROOT / "artifacts/live-driver"
        cls.artifacts.mkdir(parents=True, exist_ok=True)
        fingerprint = {p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                       for p in sorted(cls.runtime.glob("*.jar"))}
        (cls.artifacts / "runtime.json").write_text(json.dumps({
            "java": subprocess.run([cls.java, "-version"], capture_output=True, text=True,
                                   check=True).stderr,
            "jars": fingerprint,
            "agentSha256": hashlib.sha256(cls.agent.read_bytes()).hexdigest(),
            "scope": "synthetic model; original Driver FunctionLauncher; no external PDF viewer",
        }, indent=2), encoding="utf-8")

    def setUp(self):
        for suffix in (".json", ".pdf"):
            (self.artifacts / (self._testMethodName + suffix)).unlink(missing_ok=True)
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
        deadline = time.monotonic() + 30
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
            self.fail(f"Driver launcher did not appear; see {self.log.name}")
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
    unittest.main(verbosity=2)
