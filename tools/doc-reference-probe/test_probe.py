import tempfile
import unittest
import hashlib
import json
import subprocess
import sys
from datetime import date
from pathlib import Path

from probe import apply_allowlist, audit_allowlist, check, compare_allowlists


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

    def test_fenced_examples_are_not_references(self):
        with tempfile.TemporaryDirectory() as tmp:
            text = ("`ref:docs/before`\n"
                    "```text\n`ref:docs/ignored`\n```\n"
                    "~~~~example\n`ref:unknown/ignored`\n~~~\n`ref:docs/still-ignored`\n~~~~\n"
                    "`ref:docs/after`\n")
            report = check(Path(tmp), "docs/example.md", text)
            self.assertEqual([(r["line"], r["name"]) for r in report["records"]],
                             [(1, "docs/before"), (10, "docs/after")])
            unclosed = check(Path(tmp), "docs/example.md", "```\n`ref:docs/ignored`\n")
            self.assertEqual(unclosed["denominator"], 0)

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
            compare_allowlists(report, baseline, current, date(2026, 11, 1))
            self.assertEqual([i["result"] for i in report["allowlist"]["issues"]],
                             ["allowlist-increased", "allowlist-limit-increased", "allowlist-entry-added"])
            same = check(Path(tmp), "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(same, baseline, date(2026, 11, 1))
            compare_allowlists(same, baseline, baseline, date(2026, 11, 1))
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
            compare_allowlists(report, {"baseline_count": 1, "entries": [old]},
                               {"baseline_count": 1, "entries": [new]}, date(2026, 11, 1))
            self.assertEqual(report["allowlist"]["issues"], [{"document": "docs/example.md", "name": "docs/new",
                                                            "result": "allowlist-entry-added", "severity": "error"}])
            removed = check(Path(tmp), "docs/example.md", "")
            apply_allowlist(removed, {"baseline_count": 1, "entries": []}, date(2026, 11, 1))
            compare_allowlists(removed, {"baseline_count": 1, "entries": [old]},
                               {"baseline_count": 1, "entries": []}, date(2026, 11, 1))
            self.assertEqual(removed["allowlist"]["issues"], [])

    def test_allowlist_metadata_change_is_reported(self):
        with tempfile.TemporaryDirectory() as tmp:
            old = {"document": "docs/example.md", "name": "docs/missing", "reason": "tracking",
                   "issue": 2309, "expires": "2026-12-01"}
            new = dict(old, expires="2027-12-01")
            baseline = {"baseline_count": 1, "entries": [old]}
            current = {"baseline_count": 1, "entries": [new]}
            report = check(Path(tmp), "docs/example.md", "`ref:docs/missing`")
            apply_allowlist(report, current, date(2026, 11, 1))
            compare_allowlists(report, baseline, current, date(2026, 11, 1))
            self.assertEqual(report["allowlist"]["issues"][0]["result"], "allowlist-entry-changed")
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "example.md").write_text("`ref:docs/missing`", encoding="utf-8")
            (root / "baseline.json").write_text(json.dumps(baseline), encoding="utf-8")
            (root / "current.json").write_text(json.dumps(current), encoding="utf-8")
            run = subprocess.run([sys.executable, str(Path(__file__).with_name("probe.py")), "--root", str(root),
                                  "--allowlist", str(root / "current.json"), "--baseline-allowlist",
                                  str(root / "baseline.json"), "docs/example.md"],
                                 capture_output=True, text=True, check=False)
            self.assertEqual(run.returncode, 1, run.stderr)
            self.assertIn("allowlist-entry-changed", [i["result"] for i in json.loads(run.stdout)["allowlist"]["issues"]])

    def test_allowlist_audit_checks_other_and_missing_documents(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "first.md").write_text("`ref:docs/missing`", encoding="utf-8")
            (root / "docs" / "second.md").write_text("`ref:docs/existing`", encoding="utf-8")
            (root / "docs" / "existing").write_text("", encoding="utf-8")
            def entry(doc, name, **extra):
                return {"document": doc, "name": name, "reason": "tracking", "issue": 2309, **extra}
            data = {"baseline_count": 3, "entries": [
                entry("docs/first.md", "docs/missing", expires="2026-10-31"),
                entry("docs/second.md", "docs/existing"),
                entry("docs/deleted.md", "docs/missing"),
            ]}
            audit = audit_allowlist(root, data, date(2026, 11, 1))
            self.assertEqual(audit["document_count"], 3)
            self.assertEqual({i["result"] for i in audit["issues"]},
                             {"expired-allowlist", "stale-allowlist", "allowlist-document-unavailable"})
            allowlist = root / "allowlist.json"
            allowlist.write_text(json.dumps(data), encoding="utf-8")
            run = subprocess.run([sys.executable, str(Path(__file__).with_name("probe.py")),
                                  "--root", str(root), "--allowlist", str(allowlist), "--audit-allowlist",
                                  "docs/second.md"], capture_output=True, text=True, check=False)
            self.assertEqual(run.returncode, 1, run.stderr)
            self.assertEqual(json.loads(run.stdout)["allowlist_audit"]["document_count"], 3)

    def test_cli_unresolved_exception_and_stale_lifecycle(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            document = root / "docs" / "example.md"
            document.write_text("`ref:docs/missing` `docs/implicit-missing`", encoding="utf-8")
            allowlist = root / "allowlist.json"
            entry = {"document": "docs/example.md", "name": "docs/missing", "reason": "tracking", "issue": 2309}
            allowlist.write_text(json.dumps({"baseline_count": 1, "entries": [entry]}), encoding="utf-8")
            command = [sys.executable, str(Path(__file__).with_name("probe.py")), "--root", str(root)]
            def run(*args):
                return subprocess.run(command + list(args) + ["docs/example.md"],
                                      capture_output=True, text=True, check=False)
            initial = run()
            self.assertEqual(initial.returncode, 1, initial.stderr)
            self.assertEqual([r["severity"] for r in json.loads(initial.stdout)["records"]], ["error", "warning"])
            permitted = run("--allowlist", str(allowlist))
            self.assertEqual(permitted.returncode, 0, permitted.stderr)
            self.assertEqual(json.loads(permitted.stdout)["records"][0]["result"], "allowlisted-unresolved")
            (root / "docs" / "missing").write_text("", encoding="utf-8")
            stale = run("--allowlist", str(allowlist), "--audit-allowlist")
            self.assertEqual(stale.returncode, 1, stale.stderr)
            self.assertEqual(json.loads(stale.stdout)["allowlist"]["issues"][0]["result"], "stale-allowlist")

    def test_cli_checks_baseline_digest_against_exact_bytes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs").mkdir()
            (root / "docs" / "example.md").write_text("no references", encoding="utf-8")
            baseline = root / "baseline.json"
            baseline.write_bytes(b'{"baseline_count":0,"entries":[]}\n')
            digest = hashlib.sha256(baseline.read_bytes()).hexdigest()
            command = [sys.executable, str(Path(__file__).with_name("probe.py")), "--root", str(root),
                       "--allowlist", str(baseline), "--baseline-allowlist", str(baseline),
                       "--baseline-sha256", digest, "docs/example.md"]
            good = subprocess.run(command, capture_output=True, text=True, check=False)
            self.assertEqual(good.returncode, 0, good.stderr)
            info = json.loads(good.stdout)["allowlist"]
            self.assertEqual(info["baseline_sha256"], digest)
            self.assertTrue(info["baseline_digest_verified"])
            baseline.write_bytes(b'{"baseline_count":0,"entries":[]}')
            changed = subprocess.run(command, capture_output=True, text=True, check=False)
            self.assertNotEqual(changed.returncode, 0)
            self.assertIn("baseline SHA-256 mismatch", changed.stderr)


if __name__ == "__main__":
    unittest.main()
