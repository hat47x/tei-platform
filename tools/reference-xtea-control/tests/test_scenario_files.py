#!/usr/bin/env python3
"""Structural checks for the JSON control scenarios.

These run without X-TEA, Java, or a display. They catch malformed JSON,
unknown operations, and selector typos before a Windows E2E run.
"""

from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCENARIOS = ROOT / "scenarios"

OPERATIONS = {
    "assert_text",
    "click",
    "close",
    "drag",
    "file_contains",
    "file_not_contains",
    "invoke",
    "invoke_method",
    "key",
    "model_assert_field",
    "model_assert_function",
    "model_assert_table",
    "model_assert_table_absent",
    "screenshot",
    "select",
    "set",
    "sleep",
    "snapshot",
    "table_assert",
    "table_edit",
    "table_row_count",
    "table_set",
    "wait_selector",
    "wait_text",
    "wait_window",
}

SELECTOR_KEYS = {"index", "name", "path", "role", "text", "window"}


class ScenarioFilesTest(unittest.TestCase):
    def test_scenarios_are_well_formed(self) -> None:
        files = sorted(SCENARIOS.glob("*.json"))
        self.assertTrue(files, "no scenario files found")
        for path in files:
            with self.subTest(path=path.name):
                document = json.loads(path.read_text(encoding="utf-8"))
                self.assertTrue(document.get("name"), f"{path.name}: missing name")
                steps = document.get("steps")
                self.assertIsInstance(steps, list, f"{path.name}: steps must be a list")
                self.assertTrue(steps, f"{path.name}: steps must not be empty")
                for number, step in enumerate(steps, start=1):
                    where = f"{path.name} step {number}"
                    self.assertIsInstance(step, dict, f"{where}: step must be an object")
                    operation = step.get("op")
                    self.assertIn(operation, OPERATIONS, f"{where}: unknown op {operation!r}")
                    selector = step.get("selector")
                    if selector is not None:
                        self.assertIsInstance(selector, dict, f"{where}: selector must be an object")
                        self.assertTrue(selector, f"{where}: selector must not be empty")
                        unknown = set(selector) - SELECTOR_KEYS
                        self.assertFalse(unknown, f"{where}: unknown selector keys {sorted(unknown)}")


if __name__ == "__main__":
    unittest.main(verbosity=2)
