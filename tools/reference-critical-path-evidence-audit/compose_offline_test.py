#!/usr/bin/env python3
"""Isolated Git/shell contract tests; stub Go/Node do NOT run TEI product tests."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
RUNNER = ROOT / "tools/reference-critical-path-evidence-audit/compose-smoke.sh"


class OfflinePinnedCompose(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="tei-cp-offline-")
        self.addCleanup(self.temp.cleanup)
        top = Path(self.temp.name)
        self.repo = top / "checkout"
        self.repo.mkdir()
        self.bin = top / "bin"
        self.bin.mkdir()
        self.real_git = shutil.which("git")
        if not self.real_git:
            self.skipTest("git not installed")

        def write(relpath, data):
            path = self.repo / relpath
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(data, encoding="utf-8")
            return path

        write("tools/reference-critical-path-evidence-audit/compose-smoke.sh",
              RUNNER.read_text(encoding="utf-8"))
        # The fixture checks the shell control flow, not TEI tests.
        write("tools/repository-health-check.sh",
              '#!/bin/bash\nset -euo pipefail\n'
              'test "$GOPROXY" = off\n'
              'test "$GOSUMDB" = off\n'
              'test "$GOTOOLCHAIN" = local\n')
        (self.bin / "git").write_text(
            '#!/bin/sh\n'
            'if [ "$1" = fetch ]; then\n'
            '  echo "unrequested network fetch" >&2; exit 99\n'
            'fi\n'
            f'exec "{self.real_git}" "$@"\n', encoding="utf-8")
        (self.bin / "go").write_text(
            '#!/bin/sh\n'
            'if [ "$1" = version ]; then echo "fixture go"; exit 0; fi\n'
            'test "$GOPROXY" = off && test "$GOTOOLCHAIN" = local\n',
            encoding="utf-8")
        (self.bin / "node").write_text(
            '#!/bin/sh\n'
            'if [ "$1" = --version ]; then echo "fixture node"; exit 0; fi\n'
            'test "$GOPROXY" = off\n', encoding="utf-8")
        for name in ("git", "go", "node"):
            (self.bin / name).chmod(0o755)
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Test Fixture")
        self.git("config", "user.email", "test@example.invalid")
        self.git("remote", "add", "origin", "https://github.com/hat47x/tei-platform.git")
        self.git("add", ".")
        self.git("commit", "-qm", "offline fixture")
        self.sha = self.git("rev-parse", "HEAD").stdout.strip()

    def git(self, *args):
        return subprocess.run([self.real_git, "-C", str(self.repo), *args],
                              text=True, capture_output=True, check=True)

    def run_smoke(self, **overrides):
        env = os.environ.copy()
        for name in list(env):
            if name.startswith("TEI_CP_COMPOSE_"):
                env.pop(name)
        env.update(TEI_CP_COMPOSE_MODE="integrated-pinned-offline",
                   TEI_CP_COMPOSE_MAIN=self.sha)
        env.update(overrides)
        env["PATH"] = str(self.bin) + os.pathsep + os.environ["PATH"]
        return subprocess.run(["bash", "tools/reference-critical-path-evidence-audit/compose-smoke.sh"],
                              cwd=self.repo, env=env, text=True, capture_output=True)

    def test_exact_local_head_passes_without_remote_fetch(self):
        outcome = self.run_smoke()
        self.assertEqual(0, outcome.returncode, outcome.stdout + outcome.stderr)
        self.assertIn("PINNED_OFFLINE_CROSS_LANE_SMOKE_ONLY_NO_REMOTE_FRESHNESS_NO_PRODUCT_GATE",
                      outcome.stdout)
        self.assertIn("remoteFreshness=NOT_CHECKED_offline_operator_pinned",
                      outcome.stdout)
        self.assertIn("integrationHead=" + self.sha, outcome.stdout)
        self.assertNotIn("unrequested network fetch", outcome.stderr)
        self.assertEqual(self.sha, self.git("rev-parse", "HEAD").stdout.strip())
        self.assertEqual("", self.git("status", "--porcelain").stdout.strip())

    def test_wrong_pinned_commit_refused(self):
        outcome = self.run_smoke(TEI_CP_COMPOSE_MAIN="0" * 40)
        self.assertNotEqual(0, outcome.returncode)
        self.assertIn("requires the checkout HEAD", outcome.stderr)

    def test_dirty_checkout_refused(self):
        (self.repo / "tools/repository-health-check.sh").write_text("modified\n")
        outcome = self.run_smoke()
        self.assertNotEqual(0, outcome.returncode)
        self.assertIn("worktree is not clean", outcome.stderr)

    def test_unreviewed_draft_tips_refused(self):
        outcome = self.run_smoke(TEI_CP_COMPOSE_1121=self.sha)
        self.assertNotEqual(0, outcome.returncode)
        self.assertIn("refuse stale Draft branch pins", outcome.stderr)

    def test_online_mode_still_requires_fetch(self):
        outcome = self.run_smoke(TEI_CP_COMPOSE_MODE="integrated-main")
        self.assertNotEqual(0, outcome.returncode)
        self.assertIn("unrequested network fetch", outcome.stderr)
        self.assertIn("fetch failed", outcome.stderr)

    def test_missing_trusted_remote_refused(self):
        self.git("remote", "set-url", "origin", "https://example.invalid/fake.git")
        outcome = self.run_smoke()
        self.assertNotEqual(0, outcome.returncode)
        self.assertIn("remote is not canonical", outcome.stderr)


if __name__ == "__main__":
    unittest.main()
