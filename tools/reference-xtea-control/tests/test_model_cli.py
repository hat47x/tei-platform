#!/usr/bin/env python3
"""Tests for the read-only .xead inspection CLI."""

from __future__ import annotations

import io
import json
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import xtea_cli  # noqa: E402

FIXTURE = ROOT / "tests" / "fixtures" / "minimal.xead"


class ModelCliTest(unittest.TestCase):
    def setUp(self) -> None:
        self.model = xtea_cli.load_model(FIXTURE)

    def test_summary(self) -> None:
        summary = xtea_cli.model_summary(self.model)
        self.assertEqual(summary["system"]["name"], "テストシステム")
        self.assertEqual(summary["counts"]["tables"], 1)
        self.assertEqual(summary["counts"]["functions"], 2)
        self.assertEqual(summary["counts"]["functionsByType"]["XF100"], 1)
        self.assertEqual(summary["counts"]["functionsByType"]["Menu"], 1)

    def test_table_fields_and_key(self) -> None:
        table = self.model["tables"][0]
        self.assertEqual(table["id"], "10")
        self.assertEqual(table["alias"], "AT010")
        self.assertEqual(len(table["fields"]), 2)
        self.assertEqual(table["fields"][0]["alias"], "CDBUMON")
        self.assertTrue(table["fields"][0]["notNull"])
        self.assertEqual(table["fields"][0]["dataType"]["sql"], "CHAR(10)")
        self.assertEqual(table["keys"][0]["type"], "PK")
        self.assertEqual(table["keys"][0]["fields"][0]["fieldId"], "1")

    def test_function_io_and_calls(self) -> None:
        functions = {item["id"]: item for item in self.model["functions"]}
        listed = functions["20"]
        self.assertEqual(listed["type"]["code"], "XF100")
        self.assertEqual(listed["ioTables"][0]["table"]["name"], "部門")
        self.assertTrue(listed["ioTables"][0]["operations"]["create"])
        self.assertTrue(listed["ioTables"][0]["operations"]["read"])
        self.assertFalse(listed["ioTables"][0]["operations"]["update"])

        menu = functions["21"]
        self.assertEqual(menu["type"]["code"], "Menu")
        self.assertEqual(menu["calls"][0]["id"], "20")
        self.assertEqual(menu["ioPanels"][0]["background"], "WINDOW")

    def test_schema_contract(self) -> None:
        schemas = xtea_cli.model_schemas()
        self.assertIn("system", schemas["model"]["required"])
        self.assertIn("functions", schemas["model"]["required"])
        self.assertIn("tables", schemas["diff"]["required"])
        output = io.StringIO()
        with redirect_stdout(output):
            code = xtea_cli.main(["schema", "diff"])
        self.assertEqual(code, 0)
        self.assertEqual(json.loads(output.getvalue())["title"], "X-TEA model diff")

    def test_validate_references(self) -> None:
        self.assertTrue(xtea_cli.validate_model(self.model)["valid"])
        text = FIXTURE.read_text(encoding="utf-8")
        text = text.replace('SubsystemID="1"', 'SubsystemID="999"')
        text = text.replace('TableID="10"', 'TableID="999"')
        with tempfile.NamedTemporaryFile("w", suffix=".xead", encoding="utf-8",
                                         delete=False) as handle:
            handle.write(text)
            modified_path = Path(handle.name)
        try:
            result = xtea_cli.validate_model(xtea_cli.load_model(modified_path))
        finally:
            modified_path.unlink()
        self.assertFalse(result["valid"])
        self.assertTrue(any("unknown subsystem" in item for item in result["errors"]))
        self.assertTrue(any("unknown IO table" in item for item in result["errors"]))

    def test_type_attribute_fallback(self) -> None:
        text = FIXTURE.read_text(encoding="utf-8")
        text = text.replace('FunctionTypeID="1"', 'Type="XF100"')
        with tempfile.NamedTemporaryFile("w", suffix=".xead", encoding="utf-8",
                                         delete=False) as handle:
            handle.write(text)
            modified_path = Path(handle.name)
        try:
            model = xtea_cli.load_model(modified_path)
        finally:
            modified_path.unlink()
        functions = {item["id"]: item for item in model["functions"]}
        self.assertEqual(functions["20"]["type"]["code"], "XF100")

    def test_diff_changed_field_and_removed_function(self) -> None:
        self.assertEqual(
            xtea_cli.diff_models(self.model, self.model)["tables"]["changed"], [])
        text = FIXTURE.read_text(encoding="utf-8")
        text = text.replace("部門名", "部門名称")
        text = text.replace('<Function ID="21"', '<Function ID="22"')
        with tempfile.NamedTemporaryFile("w", suffix=".xead", encoding="utf-8",
                                         delete=False) as handle:
            handle.write(text)
            modified_path = Path(handle.name)
        try:
            diff = xtea_cli.diff_models(self.model, xtea_cli.load_model(modified_path))
            with redirect_stdout(io.StringIO()):
                same_code = xtea_cli.main(
                    ["diff", str(FIXTURE), str(FIXTURE), "--exit-code"])
                changed_code = xtea_cli.main(
                    ["diff", str(FIXTURE), str(modified_path), "--exit-code"])
        finally:
            modified_path.unlink()
        self.assertEqual(same_code, 0)
        self.assertEqual(changed_code, 1)
        self.assertEqual([item["id"] for item in diff["tables"]["changed"]], ["10"])
        fields = diff["tables"]["changed"][0]["after"]["fields"]
        self.assertIn("部門名称", [field["name"] for field in fields])
        self.assertEqual([item["id"] for item in diff["functions"]["removed"]], ["21"])
        self.assertEqual([item["id"] for item in diff["functions"]["added"]], ["22"])

    def test_export_cli(self) -> None:
        output = io.StringIO()
        with redirect_stdout(output):
            code = xtea_cli.main(["export", str(FIXTURE)])
        self.assertEqual(code, 0)
        exported = json.loads(output.getvalue())
        self.assertEqual(exported["schemaVersion"], 1)
        self.assertEqual(len(exported["tables"]), 1)
        self.assertEqual(len(exported["functions"]), 2)


if __name__ == "__main__":
    unittest.main(verbosity=2)
