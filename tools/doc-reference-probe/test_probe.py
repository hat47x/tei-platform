import tempfile
import unittest
from pathlib import Path

from probe import check


class ProbeTest(unittest.TestCase):
    def test_current_and_planned_references(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "present.md").write_text("ok", encoding="utf-8")
            text = "`ref:docs/present.md` `ref:docs/missing.md` `docs/missing.md` `ordinary`"
            report = check(root, "docs/example.md", text)
            self.assertEqual(report["denominator"], 3)
            self.assertEqual([r["severity"] for r in report["records"]], ["info", "error", "warning"])
            text = "`planned:docs/missing.md` `planned:docs/present.md` `ref:docs/missing.md`"
            plan = check(root, "plan/example.md", text)
            self.assertEqual([r["result"] for r in plan["records"]],
                             ["planned", "planned-now-exists", "unresolved"])
            self.assertEqual([r["severity"] for r in plan["records"]], ["info", "warning", "warning"])
            current = check(root, "docs/example.md", "`planned:docs/missing.md`")
            self.assertEqual(current["records"][0]["result"], "planned-outside-plan")

    def test_invalid_path_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            report = check(Path(tmp), "docs/example.md", "`ref:docs/../outside`")
            self.assertEqual(report["records"][0]["result"], "invalid-path")

    def test_directory_marker_does_not_resolve_a_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "file").write_text("", encoding="utf-8")
            report = check(root, "docs/example.md", "`ref:docs/file/`")
            self.assertEqual(report["records"][0]["result"], "unresolved")


if __name__ == "__main__":
    unittest.main()
