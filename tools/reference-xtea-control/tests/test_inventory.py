import json
from pathlib import Path
import sys
import tempfile
import unittest
import xml.etree.ElementTree as ET
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from inventory_xtea import inspect_model, inventory


class InventoryTest(unittest.TestCase):
    def test_versions_counts_and_no_model_text(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "synthetic.zip"
            data = b'<System><Function ID="PRIVATE" Name="PRIVATE" Type="XF200">' \
                   b'<Button Action="OUTPUT"/><Button Action="COPY"/><Button Action="CALL(PRIVATE)"/></Function>' \
                   b'<Function Type="XF000" Script="session.getInputDialog(); d.addField(&quot;PRIVATE&quot;,&quot;DATE&quot;);"/></System>'
            with zipfile.ZipFile(path, "w") as archive:
                archive.writestr("current.xeaf", data)
                archive.writestr("backup/old.xeaf", b'<System><Function Type="XF110"/></System>')
                archive.writestr("unused.txt", "PRIVATE")
            no_driver = Path(folder) / "design-only.zip"
            with zipfile.ZipFile(no_driver, "w") as archive:
                archive.writestr("model.xead", "PRIVATE")
            result = inventory([path, no_driver])
            self.assertEqual(len(result["models"]), 2)
            self.assertEqual(result["usedFunctionTypes"], ["XF000", "XF110", "XF200"])
            self.assertEqual(result["models"][0]["scriptUiCallCandidates"], {"getInputDialog": 1})
            self.assertEqual(result["models"][0]["inputFieldTypeCandidates"], {"DATE": 1})
            self.assertEqual(result["models"][0]["buttonActions"], {"XF200": {"COPY": 1, "OTHER": 1, "OUTPUT": 1}})
            self.assertEqual(result["uninspectedInputs"], [{"archive": "design-only.zip", "reason": "NO_XEAF"}])
            self.assertIn("XF390", result["absentFunctionTypes"])
            self.assertNotIn("PRIVATE", json.dumps(result))

    def test_reject_entities_and_malformed_input(self):
        with self.assertRaises(ValueError):
            inspect_model(b'<!DOCTYPE System [<!ENTITY x "secret">]><System/>')
        with self.assertRaises(ET.ParseError):
            inspect_model(b'<System>')


if __name__ == "__main__":
    unittest.main(verbosity=2)
