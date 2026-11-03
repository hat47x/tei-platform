#!/usr/bin/env python3
"""Tests for check.py. Temporary repositories keep the public tree clean."""

from __future__ import annotations

import pathlib
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import check  # noqa: E402


class ScanTextTest(unittest.TestCase):
    def test_clean_text_has_no_marker(self):
        self.assertEqual(check.scan_text("nothing to see here"), [])

    def test_repository_name_marker_is_detected(self):
        marker = "tei-platform-" + "private" + "-resources"
        self.assertEqual(check.scan_text(f"see {marker} for detail"), ["非公開リポジトリ名"])

    def test_archive_marker_is_detected(self):
        marker = "archives/" + "x-tea-2026-10-14"
        self.assertEqual(check.scan_text(f"stored in {marker}"), ["アーカイブ内パス"])

    def test_research_marker_is_detected(self):
        marker = "research/" + "low-code-management" + "-tools"
        self.assertEqual(check.scan_text(f"see {marker}/note.md"), ["調査記録のパス"])

    def test_checker_source_does_not_match_itself(self):
        source = pathlib.Path(check.__file__).read_text(encoding="utf-8")
        self.assertEqual(check.scan_text(source), [])


class CliTest(unittest.TestCase):
    def run_checker(self, files: dict[str, str]) -> subprocess.CompletedProcess:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = pathlib.Path(tmp.name)
        subprocess.run(["git", "-C", str(root), "init", "-q"], check=True)
        for name, text in files.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8")
        subprocess.run(["git", "-C", str(root), "add", "-A"], check=True)
        return subprocess.run(
            [sys.executable, str(pathlib.Path(check.__file__)), "--root", str(root)],
            capture_output=True,
            text=True,
        )

    def test_clean_repository_passes(self):
        result = self.run_checker({"doc/ok.md": "clean\n"})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("public boundary: ok", result.stdout)

    def test_injected_marker_fails(self):
        marker = "archives/" + "x-tea-2026-10-14"
        result = self.run_checker({"doc/leak.md": f"stored in {marker}\n"})
        self.assertEqual(result.returncode, 1)
        self.assertIn("doc/leak.md", result.stderr)

    def test_git_ignored_file_is_not_scanned(self):
        # git ls-files only sees tracked files, so an untracked leak does not fail here.
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = pathlib.Path(tmp.name)
        subprocess.run(["git", "-C", str(root), "init", "-q"], check=True)
        (root / "tracked.md").write_text("clean\n", encoding="utf-8")
        subprocess.run(["git", "-C", str(root), "add", "tracked.md"], check=True)
        marker = "research/" + "low-code-management" + "-tools"
        (root / "untracked.md").write_text(f"{marker}\n", encoding="utf-8")
        result = subprocess.run(
            [sys.executable, str(pathlib.Path(check.__file__)), "--root", str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
