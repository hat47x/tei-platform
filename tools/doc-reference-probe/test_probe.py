import tempfile
import unittest
import json
import subprocess
import sys
from datetime import date
from pathlib import Path

from probe import apply_allowlist, check, compare_allowlists


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

    def test_unsupported_declaration_is_not_silently_ignored(self):
        with tempfile.TemporaryDirectory() as tmp:
            report = check(Path(tmp), "docs/example.md", "`ref:unknown/file` `planned:docs/日本語` `ordinary`")
            self.assertEqual(report["denominator"], 2)
            self.assertEqual([r["result"] for r in report["records"]],
                             ["unsupported-declaration", "unsupported-declaration"])
            self.assertTrue(all(r["severity"] == "error" for r in report["records"]))

    def test_allowlist_exact_expired_stale_and_growth(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            entry = {"document": "docs/example.md", "name": "docs/missing", "reason": "tracking", "issue": 2309}
            report = check(root, "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(report, {"baseline_count": 1, "entries": [entry]}, date(2026, 11, 1))
            self.assertEqual(report["records"][0]["result"], "allowlisted-unresolved")
            expired = dict(entry, expires="2026-10-31")
            report = check(root, "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(report, {"baseline_count": 1, "entries": [expired]}, date(2026, 11, 1))
            self.assertEqual(report["allowlist"]["issues"][0]["result"], "expired-allowlist")
            (root / "docs" / "missing").write_text("", encoding="utf-8")
            report = check(root, "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(report, {"baseline_count": 1, "entries": [entry]}, date(2026, 11, 1))
            self.assertEqual(report["allowlist"]["issues"][0]["result"], "stale-allowlist")
            for data in ({"baseline_count": 0, "entries": [entry]},
                         {"baseline_count": 1, "entries": [dict(entry, name="docs/*")]},
                         {"baseline_count": 2, "entries": [entry, entry]}):
                with self.assertRaises(ValueError):
                    apply_allowlist(report, data, date(2026, 11, 1))

    def test_allowlist_comparison_does_not_trust_current_cap(self):
        with tempfile.TemporaryDirectory() as tmp:
            entry = {"document": "docs/example.md", "name": "docs/missing", "reason": "tracking", "issue": 2309}
            baseline = {"baseline_count": 1, "entries": [entry]}
            current = {"baseline_count": 2, "entries": [entry, dict(entry, name="docs/other")]}
            report = check(Path(tmp), "docs/example.md", "`ref:docs/missing` `ref:docs/other`")
            apply_allowlist(report, current, date(2026, 11, 1))
            compare_allowlists(report, baseline, date(2026, 11, 1))
            self.assertEqual([i["result"] for i in report["allowlist"]["issues"]],
                             ["allowlist-increased", "allowlist-limit-increased", "allowlist-entry-added"])
            same = check(Path(tmp), "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(same, baseline, date(2026, 11, 1))
            compare_allowlists(same, baseline, date(2026, 11, 1))
            self.assertEqual(same["allowlist"]["issues"], [])

    def test_cli_rejects_increase_even_when_local_limit_is_raised(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "example.md").write_text("`ref:docs/missing`", encoding="utf-8")
            entry = {"document": "docs/example.md", "name": "docs/missing", "reason": "test", "issue": 2309}
            baseline = root / "base.json"
            current = root / "current.json"
            baseline.write_text(json.dumps({"baseline_count": 0, "entries": []}), encoding="utf-8")
            current.write_text(json.dumps({"baseline_count": 1, "entries": [entry]}), encoding="utf-8")
            run = subprocess.run([sys.executable, str(Path(__file__).with_name("probe.py")),
                                  "--root", str(root), "--allowlist", str(current),
                                  "--baseline-allowlist", str(baseline), "docs/example.md"],
                                 capture_output=True, text=True, check=False)
            self.assertEqual(run.returncode, 1, run.stderr)
            issues = json.loads(run.stdout)["allowlist"]["issues"]
            self.assertEqual({item["result"] for item in issues},
                             {"allowlist-increased", "allowlist-limit-increased", "allowlist-entry-added"})

    def test_allowlist_replacement_is_not_hidden_by_equal_count(self):
        with tempfile.TemporaryDirectory() as tmp:
            old = {"document": "docs/example.md", "name": "docs/old", "reason": "tracking", "issue": 2309}
            new = dict(old, name="docs/new")
            report = check(Path(tmp), "docs/example.md", "`ref:docs/new`")
            apply_allowlist(report, {"baseline_count": 1, "entries": [new]}, date(2026, 11, 1))
            compare_allowlists(report, {"baseline_count": 1, "entries": [old]}, date(2026, 11, 1))
            self.assertEqual(report["allowlist"]["issues"], [{"document": "docs/example.md", "name": "docs/new",
                                                            "result": "allowlist-entry-added", "severity": "error"}])
            removed = check(Path(tmp), "docs/example.md", "")
            apply_allowlist(removed, {"baseline_count": 1, "entries": []}, date(2026, 11, 1))
            compare_allowlists(removed, {"baseline_count": 1, "entries": [old]}, date(2026, 11, 1))
            self.assertEqual(removed["allowlist"]["issues"], [])


if __name__ == "__main__":
    unittest.main()
