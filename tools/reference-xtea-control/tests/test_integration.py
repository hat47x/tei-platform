#!/usr/bin/env python3
"""End-to-end test for the X-TEA control agent against a synthetic Swing app.

Build first:
    powershell -ExecutionPolicy Bypass -File build.ps1
    python tests/test_integration.py
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import tempfile
import time
import unittest
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from xtea_control import ControlClient, ControlError  # noqa: E402


class IntegrationTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.java = os.environ.get("JAVA", "java")
        cls.jar = ROOT / "build" / "xtea-control-agent.jar"
        cls.fixture = ROOT / "build" / "fixture-classes"
        if not cls.jar.exists() or not cls.fixture.exists():
            raise unittest.SkipTest("run build.ps1 or build.sh first")
        cls.tmp = tempfile.TemporaryDirectory(prefix="xtea-control-test-")
        cls.endpoint_file = Path(cls.tmp.name) / "endpoint.json"
        cls.process = subprocess.Popen(
            [
                cls.java,
                f"-javaagent:{cls.jar}=port=0,tokenFile={cls.endpoint_file}",
                "-cp",
                str(cls.fixture),
                "FakeXteaApp",
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if cls.endpoint_file.exists():
                try:
                    endpoint = json.loads(cls.endpoint_file.read_text(encoding="utf-8"))
                    if endpoint.get("url"):
                        cls.endpoint = endpoint
                        break
                except (json.JSONDecodeError, OSError):
                    pass
            if cls.process.poll() is not None:
                raise RuntimeError(f"fixture exited with {cls.process.returncode}")
            time.sleep(0.1)
        else:
            raise RuntimeError("agent endpoint was not written")
        cls.client = ControlClient(cls.endpoint["url"], cls.endpoint["token"])
        cls.client.wait_window("Fake X-TEA Control Fixture", timeout_ms=15000)

    @classmethod
    def tearDownClass(cls) -> None:
        try:
            cls.client.shutdown(exit_process=True)
        except Exception:
            pass
        try:
            cls.process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            cls.process.kill()
        cls.tmp.cleanup()

    def test_01_health_and_windows(self) -> None:
        health = self.client.health()
        self.assertEqual(health["protocol"], "xtea-control/1")
        self.assertEqual(health["pid"], self.endpoint["pid"])
        windows = self.client.windows()["windows"]
        self.assertTrue(any(w["title"] == "Fake X-TEA Control Fixture" for w in windows))

    def test_02_menu_catalog_and_path(self) -> None:
        catalog = self.client.menus()
        texts = {entry["text"] for entry in catalog["actions"]}
        self.assertIn("Open", texts)
        self.assertIn("Copy", texts)
        open_entry = next(entry for entry in catalog["actions"] if entry["text"] == "Open")
        self.assertTrue(open_entry["path"].startswith("m."))
        self.client.invoke({"path": open_entry["path"], "window": "Fake X-TEA"})
        self.client.wait_text({"name": "status"}, expected="opened", timeout_ms=5000)

    def test_03_text_button_and_key(self) -> None:
        self.client.set_value({"name": "input"}, "hello")
        self.client.set_value({"name": "wrappedInput"}, "wrapped")
        self.client.set_value({"name": "wrappedInput"}, "-value", mode="append")
        self.assertEqual(self.client.find({"name": "wrappedInput"})["textValue"], "wrapped-value")
        self.client.invoke({"role": "button", "text": "Run"}, settle_ms=100)
        self.client.wait_text({"name": "status"}, expected="ran:hello", timeout_ms=5000)
        self.client.wait_text({"name": "wrapperValue"}, expected="wrapped-value", timeout_ms=5000)
        self.client.key({"name": "input"}, key="ENTER")
        self.client.wait_text({"name": "status"}, expected="enter:hello", timeout_ms=5000)
        self.client.select({"name": "table"}, row=0, column=0)
        self.client.key({"name": "table"}, key="ENTER")
        self.client.wait_text({"name": "status"}, expected="table-enter", timeout_ms=5000)
        long_text = "日本語の長文" * 100 + "END"
        self.client.set_value({"name": "input"}, long_text)
        self.client.wait_text({"name": "input"}, contains="END", timeout_ms=1000)
        result = self.client.set_value({"name": "input"}, "-appended", mode="append")
        self.assertEqual(result["oldValue"], long_text)
        self.assertEqual(result["value"], long_text + "-appended")
        self.client.set_value({"name": "input"}, "hello")

    def test_04_choices_tabs_and_table(self) -> None:
        enabled = self.client.set_value({"name": "enabled"}, False)
        self.assertEqual(enabled["value"], False)
        mode = self.client.set_value({"name": "mode"}, "Safe")
        self.assertEqual(mode["value"], "Safe")
        table = self.client.table_read({"name": "table"})
        self.assertEqual(table["rowCount"], 2)
        info = self.client.table_info({"name": "table"})
        self.assertEqual(info["tableRowCount"], 2)
        self.assertEqual(info["modelRowCount"], 2)
        self.assertIn("TableModel", info["modelClass"])
        self.client.table_set({"name": "table"}, 0, "Name", "Changed")
        self.assertEqual(self.client.find({"name": "table"})["columnCount"], 2)
        self.assertEqual(self.client.table_read({"name": "table"})["rows"][0][1], "Changed")
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=1)
        self.assertEqual(self.client.find({"role": "tab", "window": "Fake X-TEA"})["selectedIndex"], 1)
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=0)

    def test_04b_custom_row_table(self) -> None:
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=2)
        value = self.client.table_read({"name": "customTable"})
        self.assertEqual(value["columns"], ["Code", "Name"])
        self.assertEqual(value["rows"][0], ["A", "Alpha"])
        self.assertEqual(value["rowType"], "FakeXteaApp$CustomRow")
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=0)

    def test_04c_real_cell_editor_commit_cancel(self) -> None:
        table = {"windowClass": "javax.swing.JFrame$", "name": "table"}
        original = self.client.table_read(table)["rows"][0][1]
        self.client.table_edit(table, row=0, column="Name")
        node = self.client.find({"window": "Fake X-TEA", "role": "text", "text": "^" + re.escape(original) + "$"})
        self.assertIsNotNone(node)
        editor = {"window": "Fake X-TEA", "path": node["path"]}
        self.client.set_value(editor, "Committed")
        self.client.table_edit(table, "commit")
        self.assertEqual(self.client.table_read(table)["rows"][0][1], "Committed")
        self.client.table_edit(table, row=0, column="Name")
        self.client.set_value(editor, "Discarded")
        self.client.table_edit(table, "cancel")
        self.assertEqual(self.client.table_read(table)["rows"][0][1], "Committed")
        with self.assertRaises(ControlError) as caught:
            self.client.table_edit(table, row=999)
        self.assertEqual(caught.exception.code, "INVALID_ROW")

    def test_04c_tree_read(self) -> None:
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=3)
        value = self.client.tree_read({"name": "tree"})
        self.assertEqual([row["text"] for row in value["rows"]],
                         ["root", "tables", "T1", "T2", "functions"])
        self.assertEqual(value["rows"][2]["indices"], [0, 0, 0])
        self.client.select({"name": "tree"}, path=[0, 0, 0])
        tree = self.client.find({"name": "tree"})
        self.assertIn("T1", tree["selectedPath"])
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=0)

    def test_04d_table_double_click(self) -> None:
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=0)
        self.client.click({"role": "table"}, count=2, row=0, column=0)
        self.client.wait_text({"name": "status"}, expected="table-double", timeout_ms=5000)

    def test_04e_list_items_and_check_state(self) -> None:
        self.client.select({"name": "tabs"}, title="List")
        try:
            node = self.client.find({"name": "observedList"})
            self.assertEqual(node["itemCount"], 201)
            self.assertTrue(node["itemsTruncated"])
            self.assertEqual(len(node["items"]), 200)
            self.assertEqual(node["items"][0], {"index": 0, "text": "Checked item",
                                               "checked": True, "enabled": False})
            self.assertEqual(node["items"][1], {"index": 1, "text": "Item 1"})
            self.client.select({"name": "observedList"}, item_index=1)
            self.assertEqual(self.client.find({"name": "observedList"})["selectedIndex"], 1)
        finally:
            self.client.select({"name": "tabs"}, title="Main")

    def test_05_modal_dialog(self) -> None:
        self.client.select({"role": "tab", "window": "Fake X-TEA"}, item_index=0)
        cursor = self.client.trace()["lastId"]
        operation = self.client.invoke({"role": "button", "text": "Open Dialog"})
        self.client.wait_window("Fake Dialog", timeout_ms=10000)
        windows = self.client.windows()["windows"]
        dialog = next(w for w in windows if w["title"] == "Fake Dialog")
        self.assertTrue(dialog["modal"])
        window_call = self.client.invoke_method({"role": "window", "window": "^Fake Dialog$"}, "toFront")
        self.assertTrue(window_call["accepted"])
        events = [event["type"] for event in self.client.trace(since=cursor)["events"]
                  if event["target"] == operation["target"]]
        self.assertEqual(events, ["invoke", "invoke.start"])
        self.client.set_value({"name": "dialogInput"}, "changed")
        self.client.invoke({"role": "button", "text": "OK"})
        self.client.wait_window("Fake Dialog", state="absent", timeout_ms=10000)
        self.client.wait_text({"name": "status"}, expected="dialog:hello", timeout_ms=5000)
        events = [event["type"] for event in self.client.trace(since=cursor)["events"]
                  if event["target"] == operation["target"]]
        self.assertEqual(events, ["invoke", "invoke.start", "invoke.complete"])

    def test_06_screenshot_and_trace(self) -> None:
        path = Path(self.tmp.name) / "fixture.png"
        value = self.client.screenshot(path=path)
        self.assertGreater(value["width"], 0)
        self.assertTrue(path.exists() and path.stat().st_size > 100)
        trace = self.client.trace()
        self.assertGreater(trace["lastId"], 0)
        self.assertTrue(any(event["type"] == "invoke" for event in trace["events"]))

    def test_07_unknown_and_unauthorized(self) -> None:
        with self.assertRaises(ControlError) as caught:
            self.client.invoke({"role": "button", "text": "does-not-exist"})
        self.assertEqual(caught.exception.code, "NOT_FOUND")
        request = urllib.request.Request(self.client.url + "/windows", method="GET")
        with self.assertRaises(urllib.error.HTTPError) as caught_http:
            urllib.request.urlopen(request, timeout=5)
        self.assertEqual(caught_http.exception.code, 401)


if __name__ == "__main__":
    unittest.main(verbosity=2)
