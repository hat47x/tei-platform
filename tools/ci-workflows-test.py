#!/usr/bin/env python3
"""CI workflow contract regression with isolated local Git repositories.

Covers the manual workflows, the automatic workflow's public-PR boundary
and its main push path.
Does not execute GitHub Actions or the TEI repository health check.
Run from the repository root: python3 tools/ci-workflows-test.py
"""

import os
import re
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
HEALTH = ROOT / '.github/workflows/tei-repository-health-manual.yml'
SMOKE = ROOT / '.github/workflows/tei-runner-smoke-manual.yml'
AUTO_HEALTH = ROOT / '.github/workflows/tei-repository-health-ci.yml'
REMOTE_URL = 'https://github.com/hat47x/tei-platform.git'
PER_SHA_GROUP = ('tei-health-${{ github.event_name }}-'
                 '${{ github.event.pull_request.number || github.sha }}')
CANCEL_EXPRESSION = "${{ github.event_name == 'pull_request' }}"
CANCEL_ONLY_PULL_REQUEST = 'cancel-in-progress: ' + CANCEL_EXPRESSION


def command(*argv, cwd=None, env=None, check=True):
    return subprocess.run(argv, cwd=cwd, env=env, text=True,
                          capture_output=True, check=check)


# The workflow bodies are bash, and the cases below depend on the environment
# variables this process passes down. On Windows the `bash` on PATH is the WSL
# launcher (C:\WINDOWS\system32\bash.exe), which does not forward Windows
# environment variables: GITHUB_EVENT_NAME arrives empty and every trigger case
# lands in the workflow's "unsupported trigger" branch. That is a property of
# the host, not of the workflow (#4355).
#
# So resolve a bash that forwards them, preferring the one on PATH (which is
# /bin/bash in CI), then Git for Windows bash. TEI_CI_TEST_BASH overrides the
# choice. When nothing forwards them, the bash cases skip with the reason rather
# than failing as if the repository were broken.
def _bash_forwards_environment(candidate):
    probe = 'echo "$TEI_BASH_PROBE"'
    env = dict(os.environ, TEI_BASH_PROBE='forwarded')
    try:
        result = subprocess.run([candidate, '-c', probe], env=env, text=True,
                                capture_output=True, check=False, timeout=60)
    except (OSError, subprocess.SubprocessError):
        return False
    return result.stdout.strip() == 'forwarded'


def resolve_bash():
    override = os.environ.get('TEI_CI_TEST_BASH')
    candidates = [override] if override else []
    candidates.append('bash')
    if os.name == 'nt':
        candidates.append(r'C:\Program Files\Git\bin\bash.exe')
    for candidate in candidates:
        if candidate and _bash_forwards_environment(candidate):
            return candidate
    return None


BASH = resolve_bash()


def _any_bash():
    for candidate in [os.environ.get('TEI_CI_TEST_BASH'), 'bash']:
        if not candidate:
            continue
        try:
            result = subprocess.run([candidate, '-n', '-c', 'true'], text=True,
                                    capture_output=True, check=False, timeout=60)
        except (OSError, subprocess.SubprocessError):
            continue
        if result.returncode == 0:
            return candidate
    return None


# Parsing a script does not read the environment, so the syntax checks accept any
# bash. Running one does read it, which is why only those cases need BASH below.
SYNTAX_BASH = BASH or _any_bash()
SKIP_NO_BASH = unittest.skipIf(
    BASH is None,
    'no bash forwards the environment variables these workflow steps read; '
    'set TEI_CI_TEST_BASH to one that does (Git for Windows bash works) [#4355]',
)
SKIP_NO_SYNTAX_BASH = unittest.skipIf(
    SYNTAX_BASH is None, 'no bash is available to parse the workflow bodies [#4355]',
)


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


def meaningful_shell_lines(script):
    """Compare executable shell lines, not command names in comments."""
    return [line.strip() for line in script.splitlines()
            if line.strip() and not line.lstrip().startswith('#')]


def required_health_commands():
    return ['set -euo pipefail', 'go version',
            'bash tools/repository-health-check.sh']



def public_pr_boundary_errors(source):
    """Static public-fork safety guard; does not prove a live fork run."""
    failures = []
    if not re.search(r'(?m)^on:\n  pull_request:\n    branches: \[main\]', source):
        failures.append('expected main-targeted pull_request event')
    if 'pull_request_target' in source:
        failures.append('no elevated pull_request_target execution')
    if not re.search(r'(?m)^permissions:\n  contents: read(?:\n|$)', source):
        failures.append('token must be read-only')
    if re.search(r'(?m)^[ \t]+permissions:', source):
        failures.append('job must not override token permissions')
    if 'secrets.' in source or 'secrets[' in source:
        failures.append('no secrets may be referenced')
    if 'continue-on-error:' in source:
        failures.append('do not ignore checks')
    if 'persist-credentials: false' not in source:
        failures.append('checkout credentials must not persist')
    if 'fetch-depth: 0' not in source:
        failures.append('full Git history is required')
    if 'ref: ${{ github.event.pull_request.head.sha }}' in source:
        failures.append('primary checkout must use PR merge commit')
    uses = re.findall(r'(?m)^\s*uses:\s*(\S+)', source)
    if uses != [
            'actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1',
            'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e']:
        failures.append('use only audited pinned actions')
    if 'PR_HEAD_SHA: ${{ github.event.pull_request.head.sha }}' not in source:
        failures.append('head must come from PR event')
    if source.count('test "$head" = "$PR_HEAD_SHA"') < 2:
        failures.append('verify PR merge commit against head')
    if 'test "$head_parent" = "$PR_HEAD_SHA"' not in source:
        failures.append('verify isolated candidate head')
    if 'bash tools/repository-health-check.sh' not in source:
        failures.append('do not skip repository health')
    if 'timeout-minutes:' not in source or CANCEL_ONLY_PULL_REQUEST not in source:
        failures.append('bound runner resource use')
    return failures


def push_path_errors(source):
    """Static guard for the main push path; does not run GitHub Actions."""
    failures = []
    group = re.findall(r'(?m)^  group:\s*(.+)$', source)
    if group != [PER_SHA_GROUP]:
        failures.append('push runs must be grouped per commit SHA')
    cancel = re.findall(r'(?m)^  cancel-in-progress:\s*(.+)$', source)
    if cancel != [CANCEL_EXPRESSION]:
        failures.append('only pull_request runs may be cancelled')
    if 'ls-remote' in source:
        failures.append('push checks must not depend on the live remote main')
    if '\n  push:\n    branches: [main]\n' not in source:
        failures.append('push trigger must stay main-only')
    if source.count('concurrency:') != 1:
        failures.append('concurrency must be declared once at the workflow level')
    return failures


def git(directory, *args):
    return command('git', '-C', str(directory), *args).stdout.strip()


@SKIP_NO_BASH
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
        self.assertEqual(5, len(blocks))
        self.pre, self.contract, self.health, self.e1, self.post = [x.replace(REMOTE_URL, self.remote_url) for x in blocks]
        self.env = os.environ.copy()
        self.env.update(EXPECTED_HEAD_SHA=self.head_sha, EXPECTED_MAIN_SHA=self.main_sha,
                        GITHUB_SHA=self.head_sha, GITHUB_REF='refs/heads/work/issue-1189-ci')

    def check(self, script, success, env=None):
        result = command(BASH, '-c', script, cwd=self.checkout,
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

    def test_e1_script_rejects_stale_head_before_execution(self):
        env = dict(self.env, EXPECTED_HEAD_SHA=self.main_sha)
        self.check(self.e1, False, env)

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

    def test_assume_unchanged_flag_rejected(self):
        # git status can remain empty even if this index flag hides changes.
        git(self.checkout, 'update-index', '--assume-unchanged', 'fixture.txt')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_skip_worktree_flag_rejected(self):
        git(self.checkout, 'update-index', '--skip-worktree', 'fixture.txt')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_main_advance_rejected(self):
        git(self.sender, 'switch', '-q', 'main')
        self.commit_and_push(self.sender, 'refs/heads/main')
        self.check(self.post, False)

    def test_work_branch_advance_rejected(self):
        git(self.sender, 'fetch', '-q', 'origin', 'work/issue-1189-ci')
        git(self.sender, 'switch', '-qc', 'work/issue-1189-ci', 'FETCH_HEAD')
        self.commit_and_push(self.sender, 'refs/heads/work/issue-1189-ci')
        self.check(self.post, False)


def auto_push_scripts(path=AUTO_HEALTH, replace_url=None):
    """Pick the pre and post steps of the automatic workflow by their content."""
    _, blocks = scripts(path)
    if replace_url is not None:
        blocks = [x.replace(REMOTE_URL, replace_url) for x in blocks]
    pre = [x for x in blocks if '実行前の作業領域の変更' in x]
    post = [x for x in blocks if '実行後の作業領域の変更' in x]
    if len(pre) != 1 or len(post) != 1:
        raise AssertionError('expected exactly one pre and one post step')
    return pre[0], post[0]


@SKIP_NO_BASH
class AutoWorkflowPushPath(unittest.TestCase):
    """Run the push branch of the automatic workflow's pre and post steps.

    Uses isolated local Git repositories only. It does not show how real
    GitHub Actions push events behave.
    """

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='tei-ci-push-')
        self.addCleanup(self.temp.cleanup)
        top = Path(self.temp.name)
        upstream = top / 'upstream'
        self.remote = top / 'remote.git'
        self.checkout = top / 'checkout'
        self.sender = top / 'sender'
        command('git', 'init', '-q', '-b', 'main', str(upstream))
        git(upstream, 'config', 'user.name', 'TEI CI Fixture')
        git(upstream, 'config', 'user.email', 'test@example.invalid')
        (upstream / '.gitignore').write_text('*.ignored\n', encoding='utf-8')
        (upstream / 'fixture.txt').write_text('baseline\n', encoding='utf-8')
        git(upstream, 'add', '.')
        git(upstream, 'commit', '-qm', 'initial')
        (upstream / 'fixture.txt').write_text('pushed commit\n', encoding='utf-8')
        git(upstream, 'commit', '-qam', 'pushed commit')
        command('git', 'clone', '-q', '--bare', str(upstream), str(self.remote))
        command('git', 'clone', '-q', str(self.remote), str(self.checkout))
        command('git', 'clone', '-q', str(self.remote), str(self.sender))
        for directory in (self.checkout, self.sender):
            git(directory, 'config', 'user.name', 'TEI CI Fixture')
            git(directory, 'config', 'user.email', 'test@example.invalid')
        self.pushed_sha = git(self.checkout, 'rev-parse', 'HEAD')
        self.older_sha = git(self.checkout, 'rev-parse', 'HEAD~1')
        # If a check reintroduces a live remote lookup, it reads this local
        # remote instead of the network, so the main-advance case still shows it.
        self.pre, self.post = auto_push_scripts(replace_url=self.remote.as_uri())
        self.env = os.environ.copy()
        self.env.update(GITHUB_EVENT_NAME='push', GITHUB_REF='refs/heads/main',
                        GITHUB_SHA=self.pushed_sha,
                        PR_HEAD_SHA='', PR_BASE_SHA='')

    def check(self, script, success, env=None):
        result = command(BASH, '-c', script, cwd=self.checkout,
                         env=env or self.env, check=False)
        self.assertEqual(result.returncode == 0, success,
                         f'exit={result.returncode}\nstdout={result.stdout}\nstderr={result.stderr}')

    def advance_main(self):
        git(self.sender, 'switch', '-q', 'main')
        (self.sender / 'fixture.txt').write_text('later main update\n', encoding='utf-8')
        git(self.sender, 'commit', '-qam', 'later main update')
        git(self.sender, 'push', '-q', 'origin', 'HEAD:refs/heads/main')

    def test_pushed_commit_passes(self):
        self.check(self.pre, True)
        self.check(self.post, True)

    def test_main_advanced_after_checkout_still_passes(self):
        self.advance_main()
        self.assertNotEqual(
            self.pushed_sha,
            git(self.checkout, 'ls-remote', '--exit-code', self.remote.as_uri(),
                'refs/heads/main').split()[0])
        self.check(self.pre, True)
        self.check(self.post, True)

    def test_non_main_ref_rejected(self):
        for ref in ('refs/heads/work/issue-1189-ci', 'refs/tags/version-1', ''):
            with self.subTest(ref=ref):
                env = dict(self.env, GITHUB_REF=ref)
                self.check(self.pre, False, env)
                self.check(self.post, False, env)

    def test_head_differs_from_event_sha_rejected(self):
        env = dict(self.env, GITHUB_SHA=self.older_sha)
        self.check(self.pre, False, env)
        self.check(self.post, False, env)

    def test_unsupported_event_rejected(self):
        env = dict(self.env, GITHUB_EVENT_NAME='workflow_dispatch')
        self.check(self.pre, False, env)
        self.check(self.post, False, env)

    def test_assume_unchanged_flag_rejected(self):
        git(self.checkout, 'update-index', '--assume-unchanged', 'fixture.txt')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_skip_worktree_flag_rejected(self):
        git(self.checkout, 'update-index', '--skip-worktree', 'fixture.txt')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_untracked_file_rejected(self):
        (self.checkout / 'out.tmp').write_text('temporary\n', encoding='utf-8')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_ignored_file_rejected(self):
        (self.checkout / 'out.ignored').write_text('temporary\n', encoding='utf-8')
        self.check(self.pre, False)
        self.check(self.post, False)

    def test_modified_tracked_file_rejected(self):
        (self.checkout / 'fixture.txt').write_text('changed\n', encoding='utf-8')
        self.check(self.pre, False)
        self.check(self.post, False)


class StaticWorkflowContract(unittest.TestCase):
    def test_health_script_syntax_and_permissions(self):
        text, blocks = scripts(HEALTH)
        self.assertEqual(5, len(blocks))
        self.assertRegex(text, r'(?m)^on:\n  workflow_dispatch:')
        self.assertEqual(1, text.count('  workflow_dispatch:\n'))
        self.assertEqual(1, text.count('  repository-health:\n'))
        self.assertNotRegex(text, r'(?m)^  (push|pull_request|pull_request_target|schedule):')
        self.assertNotRegex(text, r'(?m)^\s*if:\s*(?:false|\$\{\{\s*false\s*\}\})\s*$')
        self.assertNotIn('continue-on-error:', text)
        self.assertNotIn('secrets.', text)
        self.assertIn('permissions:\n  contents: read', text)
        self.assertIn('persist-credentials: false', text)
        self.assertIn('fetch-depth: 0', text)
        self.assertEqual(2, text.count('git ls-files -v'))
        self.assertIn('if: always()', text)
        self.assertLess(text.index('bash tools/repository-health-check.sh'),
                        text.index('if: always()'))
        uses = re.findall(r'(?m)^\s*uses:\s*(\S+)', text)
        self.assertEqual([
            'actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1',
            'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e',
        ], uses)
        # A comment or echo mentioning the command is not a health check.
        self.assertEqual(
            ['set -euo pipefail', 'python3 tools/ci-workflows-test.py'],
            meaningful_shell_lines(blocks[1]))
        self.assertEqual(required_health_commands(),
                         meaningful_shell_lines(blocks[2]))
        # E1 remains independently observable when full health fails;
        # it does not weaken the combined job's pass/fail authority.
        self.assertIn('if: ${{ !cancelled() }}', text)
        e1 = meaningful_shell_lines(blocks[3])
        self.assertIn('test "$(git rev-parse HEAD)" = "$EXPECTED_HEAD_SHA"', e1)
        self.assertIn('bash -n tools/reference-external-coordination/run.sh', e1)
        self.assertIn('bash -n tools/reference-external-coordination/clean-check-test.sh', e1)
        self.assertEqual(1, e1.count(
            'bash tools/reference-external-coordination/clean-check-test.sh'))
        self.assertEqual(1, e1.count(
            'bash tools/reference-external-coordination/run.sh'))
        self.assertLess(text.index('bash tools/repository-health-check.sh'),
                        text.index('bash tools/reference-external-coordination/clean-check-test.sh'))
        self.assertLess(text.index('bash tools/reference-external-coordination/run.sh'),
                        text.index('if: always()'))
        for script in blocks:
            self.assertEqual(0, command(SYNTAX_BASH, '-n', '-c', script, check=False).returncode)

    def test_echo_instead_of_health_fails_contract(self):
        _, blocks = scripts(HEALTH)
        bypass = blocks[2].replace(
            'bash tools/repository-health-check.sh',
            "echo 'skipped bash tools/repository-health-check.sh'")
        self.assertNotEqual(required_health_commands(),
                            meaningful_shell_lines(bypass))
        self.assertIn('bash tools/repository-health-check.sh', bypass)

    def test_public_pr_ci_read_only_boundary(self):
        text, blocks = scripts(AUTO_HEALTH)
        self.assertEqual([], public_pr_boundary_errors(text))
        self.assertGreaterEqual(len(blocks), 3)
        for script in blocks:
            self.assertEqual(
                0, command(SYNTAX_BASH, '-n', '-c', script, check=False).returncode)

    def test_public_pr_ci_rejects_unsafe_mutations(self):
        source = AUTO_HEALTH.read_text(encoding='utf-8')
        mutations = [
            source.replace('  pull_request:', '  pull_request_target:', 1),
            source.replace('permissions:\n  contents: read',
                           'permissions:\n  contents: write', 1),
            source.replace('    runs-on: ubuntu-24.04',
                           '    permissions:\n      contents: write\n'
                           '    runs-on: ubuntu-24.04', 1),
            source.replace('      GOENV: off',
                           '      UNSAFE_TOKEN: ${{ secrets.ACCESS_TOKEN }}\n'
                           '      GOENV: off', 1),
            source.replace('persist-credentials: false',
                           'persist-credentials: true', 1),
            source.replace('actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1',
                           'actions/checkout@v7', 1),
            source.replace('PR_HEAD_SHA: ${{ github.event.pull_request.head.sha }}',
                           'PR_HEAD_SHA: attacker-chosen', 1),
            source.replace('test "$head_parent" = "$PR_HEAD_SHA"',
                           'true # head not verified', 1),
            source.replace('          fetch-depth: 0',
                           '          ref: ${{ github.event.pull_request.head.sha }}\n'
                           '          fetch-depth: 0', 1),
            source.replace('bash tools/repository-health-check.sh',
                           'echo full-health-skipped', 1),
            source + '\n    continue-on-error: true\n',
            source.replace(CANCEL_ONLY_PULL_REQUEST, 'cancel-in-progress: false', 1),
        ]
        self.assertEqual(12, len(mutations))
        for index, changed in enumerate(mutations):
            with self.subTest(index=index):
                self.assertNotEqual(source, changed)
                self.assertNotEqual([], public_pr_boundary_errors(changed))

    def test_push_concurrency_is_per_sha_and_cancels_only_pull_requests(self):
        text, _ = scripts(AUTO_HEALTH)
        self.assertEqual([], push_path_errors(text))
        self.assertIn(CANCEL_ONLY_PULL_REQUEST, text)

    def test_push_path_rejects_unsafe_mutations(self):
        source = AUTO_HEALTH.read_text(encoding='utf-8')
        mutations = [
            source.replace('github.event.pull_request.number || github.sha',
                           'github.event.pull_request.number || github.ref', 1),
            source.replace('github.event.pull_request.number || github.sha',
                           'github.event.pull_request.number', 1),
            source.replace(CANCEL_ONLY_PULL_REQUEST, 'cancel-in-progress: true', 1),
            source.replace(CANCEL_ONLY_PULL_REQUEST, 'cancel-in-progress: false', 1),
            source.replace('              test "$GITHUB_REF" = refs/heads/main\n',
                           '              test "$GITHUB_REF" = refs/heads/main\n'
                           '              git ls-remote ' + REMOTE_URL + ' refs/heads/main\n', 1),
            source.replace('  push:\n    branches: [main]\n',
                           '  push:\n    branches: [main, "work/**"]\n', 1),
            source.replace('\njobs:\n',
                           '\nconcurrency:\n  group: extra\n\njobs:\n', 1),
        ]
        self.assertEqual(7, len(mutations))
        for index, changed in enumerate(mutations):
            with self.subTest(index=index):
                self.assertNotEqual(source, changed)
                self.assertNotEqual([], push_path_errors(changed))

    @SKIP_NO_BASH
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
            result = command(BASH, '-c', script, env=env, check=False)
            self.assertEqual(result.returncode == 0, expected)
        del base_env['GITHUB_RUN_ID']
        result = command(BASH, '-c', script, env=base_env, check=False)
        self.assertNotEqual(result.returncode, 0)

    def test_setup_go_follows_go_mod(self):
        """Go版の直接指定はgo.modの要求版と食い違うため禁止する(#2948)。

        例外は、固定の理由をファイルに書いた場合だけにする。Goの確認は
        tools/repository-architecture-check の TestWorkflowsFollowGoModVersion が
        同じ規則で見ており、こちらは「注記なしの直接指定が無いこと」を見る。
        """
        drift = []
        for path in sorted((ROOT / '.github/workflows').glob('*.yml')):
            text = path.read_text(encoding='utf-8')
            if not re.search(r'(?m)^\s*go-version:', text):
                continue
            if '# allow-literal-go-version:' in text:
                continue
            drift.append(path.name)
        self.assertEqual([], drift)


    # Jobs that deliberately allocate no hosted runner (`if: ${{ false }}`).
    TIMEOUT_EXEMPT = {
        'knowledge-portability-reference-evidence.yml': {'verify'},
    }

    def test_every_workflow_pins_actions_and_bounds_jobs(self):
        root = Path(__file__).resolve().parent.parent / '.github' / 'workflows'
        failures = []
        for path in sorted(root.glob('*.yml')):
            text = path.read_text(encoding='utf-8')
            for ref in re.findall(r'(?m)^\s*(?:-\s+)?uses:\s*(\S+)', text):
                if ref.startswith('./'):
                    continue
                if not re.fullmatch(r'[^@\s]+@[0-9a-f]{40}', ref):
                    failures.append('%s: action is not pinned to a commit SHA: %s' % (path.name, ref))
            jobs = text.split('\njobs:\n', 1)
            if len(jobs) != 2:
                failures.append('%s: no jobs block' % path.name)
                continue
            for name, body in re.findall(r'(?ms)^  ([A-Za-z0-9_-]+):\n(.*?)(?=^  [A-Za-z0-9_-]+:\n|\Z)', jobs[1]):
                if re.search(r'(?m)^    uses:', body):
                    continue  # reusable workflow call; the callee bounds itself
                if name in self.TIMEOUT_EXEMPT.get(path.name, ()):
                    continue
                if not re.search(r'(?m)^    timeout-minutes:\s*\d+\s*$', body):
                    failures.append('%s: job %s has no timeout-minutes' % (path.name, name))
        self.assertEqual([], failures)

    # CONTRIBUTING.md asks every automatic workflow to set `concurrency`, so a
    # new push to the same pull request replaces the run in flight instead of
    # queueing another one (#3682). This checks presence, group and the
    # cancel-in-progress key; the value is not pinned here because 30 older
    # workflows cancel push runs too, which is a separate behaviour change.
    def test_every_automatic_workflow_declares_concurrency(self):
        root = Path(__file__).resolve().parent.parent / '.github' / 'workflows'
        failures = []
        for path in sorted(root.glob('*.yml')):
            text = path.read_text(encoding='utf-8')
            on = re.search(r'(?m)^on:\n((?:[ #].*\n|\n)+)', text)
            if not on:
                continue
            triggers = on.group(1)
            if not re.search(r'(?m)^  (?:pull_request|push):', triggers):
                continue
            block = re.search(r'(?ms)^concurrency:\n(.*?)(?=^\S|\Z)', text)
            if not block:
                failures.append('%s: no top-level concurrency' % path.name)
                continue
            body = block.group(1)
            if not re.search(r'(?m)^  group:\s*\S', body):
                failures.append('%s: concurrency has no group' % path.name)
            if not re.search(r'(?m)^  cancel-in-progress:\s*\S', body):
                failures.append('%s: concurrency has no cancel-in-progress' % path.name)
        self.assertEqual([], failures)

    # Workflows added or changed from 2026-11-09 must not cancel a main push:
    # one result per commit is kept, and only pull-request runs are replaced.
    def test_new_concurrency_cancels_only_pull_requests(self):
        root = Path(__file__).resolve().parent.parent / '.github' / 'workflows'
        failures = []
        pr_only = re.compile(
            r'(?m)^  cancel-in-progress:\s*'
            r"\$\{\{\s*github\.event_name == 'pull_request'\s*\}\}\s*$")
        for name in ('tei-postgres-unrun-suites-ci.yml',
                     'tei-windows-go-test-survey.yml'):
            text = (root / name).read_text(encoding='utf-8')
            block = re.search(r'(?ms)^concurrency:\n(.*?)(?=^\S|\Z)', text)
            if not block or not pr_only.search(block.group(1)):
                failures.append('%s: concurrency does not cancel only pull requests' % name)
        self.assertEqual([], failures)

    def test_every_checkout_drops_persisted_credentials(self):
        root = Path(__file__).resolve().parent.parent / '.github' / 'workflows'
        failures = []
        for path in sorted(root.glob('*.yml')):
            text = path.read_text(encoding='utf-8')
            if 'uses: actions/checkout@' not in text:
                continue
            if 'persist-credentials: false' not in text:
                failures.append('%s: checkout keeps credentials' % path.name)
        self.assertEqual([], failures)


if __name__ == '__main__':
    unittest.main(verbosity=2)
