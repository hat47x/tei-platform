#!/usr/bin/env python3
"""Manual CI workflow contract regression with isolated local Git repositories.

Does not execute GitHub Actions or the TEI repository health check.
Run from the repository root: python3 tools/ci-workflows-test.py
"""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
HEALTH = ROOT / '.github/workflows/tei-repository-health-manual.yml'
SMOKE = ROOT / '.github/workflows/tei-runner-smoke-manual.yml'
REMOTE_URL = 'https://github.com/hat47x/tei-platform.git'


def command(*argv, cwd=None, env=None, check=True):
    return subprocess.run(argv, cwd=cwd, env=env, text=True,
                          capture_output=True, check=check)


def scripts(path):
    """Read only literal YAML run: | blocks with standard 10-space body indent."""
    text = path.read_text(encoding='utf-8')
    blocks = []
    lines = text.splitlines()
    for i, line in enumerate(lines):
        if line != '        run: |':
            continue
        body = []
        for following in lines[i + 1:]:
            if following and not following.startswith('          '):
                break
            body.append(following[10:] if following else '')
        blocks.append('\n'.join(body) + '\n')
    return text, blocks


def git(directory, *args):
    return command('git', '-C', str(directory), *args).stdout.strip()


class WorkflowContract(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='tei-ci-regression-')
        self.addCleanup(self.temp.cleanup)
        top = Path(self.temp.name)
        self.upstream = top / 'upstream'
        self.remote = top / 'remote.git'
        self.checkout = top / 'checkout'
        self.sender = top / 'sender'
        command('git', 'init', '-q', '-b', 'main', str(self.upstream))
        git(self.upstream, 'config', 'user.name', 'TEI CI Fixture')
        git(self.upstream, 'config', 'user.email', 'test@example.invalid')
        (self.upstream / '.gitignore').write_text('*.ignored\n', encoding='utf-8')
        (self.upstream / 'fixture.txt').write_text('baseline\n', encoding='utf-8')
        git(self.upstream, 'add', '.')
        git(self.upstream, 'commit', '-qm', 'initial')
        command('git', 'clone', '-q', '--bare', str(self.upstream), str(self.remote))
        command('git', 'clone', '-q', str(self.remote), str(self.checkout))
        command('git', 'clone', '-q', str(self.remote), str(self.sender))
        for directory in (self.checkout, self.sender):
            git(directory, 'config', 'user.name', 'TEI CI Fixture')
            git(directory, 'config', 'user.email', 'test@example.invalid')
        self.main_sha = git(self.checkout, 'rev-parse', 'HEAD')
        git(self.checkout, 'switch', '-qc', 'work/issue-1189-ci')
        (self.checkout / 'fixture.txt').write_text('candidate\n', encoding='utf-8')
        git(self.checkout, 'commit', '-qam', 'candidate')
        git(self.checkout, 'push', '-q', 'origin', 'HEAD:refs/heads/work/issue-1189-ci')
        self.head_sha = git(self.checkout, 'rev-parse', 'HEAD')
        self.remote_url = self.remote.as_uri()
        _, blocks = scripts(HEALTH)
        self.assertEqual(3, len(blocks))
        self.pre, self.health, self.post = [x.replace(REMOTE_URL, self.remote_url) for x in blocks]
        self.env = os.environ.copy()
        self.env.update(EXPECTED_HEAD_SHA=self.head_sha, EXPECTED_MAIN_SHA=self.main_sha,
                        GITHUB_SHA=self.head_sha, GITHUB_REF='refs/heads/work/issue-1189-ci')

    def check(self, script, success, env=None):
        result = command('bash', '-c', script, cwd=self.checkout,
                         env=env or self.env, check=False)
        self.assertEqual(result.returncode == 0, success,
                         f'exit={result.returncode}\nstdout={result.stdout}\nstderr={result.stderr}')

    def commit_and_push(self, directory, ref):
        (directory / 'fixture.txt').write_text('changed again\n', encoding='utf-8')
        git(directory, 'commit', '-qam', 'advance')
        git(directory, 'push', '-q', 'origin', f'HEAD:{ref}')

    def test_pre_and_post_pass(self):
        self.check(self.pre, True)
        self.check(self.post, True)

    def test_stale_main_rejected(self):
        env = dict(self.env, EXPECTED_MAIN_SHA=self.head_sha)
        self.check(self.pre, False, env)

    def test_stale_head_rejected(self):
        env = dict(self.env, EXPECTED_HEAD_SHA=self.main_sha)
        self.check(self.pre, False, env)

    def test_invalid_sha_rejected(self):
        env = dict(self.env, EXPECTED_HEAD_SHA='not-a-sha')
        self.check(self.pre, False, env)

    def test_tag_dispatch_rejected(self):
        env = dict(self.env, GITHUB_REF='refs/tags/version-1')
        self.check(self.pre, False, env)

    def test_event_sha_mismatch_rejected(self):
        env = dict(self.env, GITHUB_SHA=self.main_sha)
        self.check(self.pre, False, env)

    def test_changed_local_head_rejected(self):
        git(self.checkout, 'switch', '-q', 'main')
        self.check(self.post, False)

    def test_ignored_file_rejected(self):
        (self.checkout / 'out.ignored').write_text('temporary\n')
        self.check(self.pre, False)

    def test_untracked_file_rejected(self):
        (self.checkout / 'out.tmp').write_text('temporary\n')
        self.check(self.pre, False)

    def test_main_advance_rejected(self):
        git(self.sender, 'switch', '-q', 'main')
        self.commit_and_push(self.sender, 'refs/heads/main')
        self.check(self.post, False)

    def test_work_branch_advance_rejected(self):
        git(self.sender, 'fetch', '-q', 'origin', 'work/issue-1189-ci')
        git(self.sender, 'switch', '-qc', 'work/issue-1189-ci', 'FETCH_HEAD')
        self.commit_and_push(self.sender, 'refs/heads/work/issue-1189-ci')
        self.check(self.post, False)


class StaticWorkflowContract(unittest.TestCase):
    def test_health_script_syntax_and_permissions(self):
        text, blocks = scripts(HEALTH)
        self.assertEqual(3, len(blocks))
        self.assertRegex(text, r'(?m)^on:\n  workflow_dispatch:')
        self.assertNotRegex(text, r'(?m)^  (push|pull_request|pull_request_target|schedule):')
        self.assertIn('permissions:\n  contents: read', text)
        self.assertIn('fetch-depth: 0', text)
        self.assertIn('bash tools/repository-health-check.sh', text)
        self.assertIn('if: always()', text)
        for script in blocks:
            self.assertEqual(0, command('bash', '-n', '-c', script, check=False).returncode)

    def test_runner_smoke_contract(self):
        text, blocks = scripts(SMOKE)
        self.assertEqual(1, len(blocks))
        self.assertRegex(text, r'(?m)^on:\n  workflow_dispatch:')
        self.assertNotRegex(text, r'(?m)^  (push|pull_request|pull_request_target|schedule):')
        self.assertIn('permissions: {}', text)
        self.assertIn('timeout-minutes: 5', text)
        self.assertNotIn('actions/checkout', text)
        script = blocks[0]
        base_env = dict(os.environ, RUNNER_OS='Linux', RUNNER_ARCH='X64',
                        RUNNER_NAME='fixture-runner', GITHUB_RUN_ID='123', GITHUB_RUN_ATTEMPT='1')
        for osname, expected in [('Linux', True), ('Windows', False)]:
            env = dict(base_env, RUNNER_OS=osname)
            result = command('bash', '-c', script, env=env, check=False)
            self.assertEqual(result.returncode == 0, expected)
        del base_env['GITHUB_RUN_ID']
        result = command('bash', '-c', script, env=base_env, check=False)
        self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main(verbosity=2)
