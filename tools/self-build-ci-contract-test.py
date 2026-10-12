#!/usr/bin/env python3
"""セルフビルド専用CIの起動漏れ・証拠欠落を防ぐ局所回帰試験。

GitHub Actionsは実行しない。標準ライブラリのみを用い、実際の
ワークフロー定義の監視範囲と安全境界を検査する。
"""

from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = ROOT / ".github/workflows/tei-self-build-qualification.yml"

# Goの参照ランナーは変換実装以外にも、core、配備、検証、
# 作業領域の各パッケージへ依存する。Goモジュール定義も構築入力。
# 細かなパッケージ列挙による監視漏れよりも、安全側の範囲を採る。
REQUIRED_INPUTS = (
    "go.mod",
    "go.sum",
    "internal/core/example.go",
    "internal/deployment/executor/example.go",
    "internal/ir/example.go",
    "internal/plugin/runtime/example.go",
    "internal/plugin/transform/example.go",
    "internal/validation/example.go",
    "internal/workspace/example.go",
    "tools/reference-plugin-self-build/example.go",
    "tools/reference-transform-seed-builder/example.go",
    "tools/reference-transform-runtime-engine/example.go",
    "tools/reference-transform-runtime-engine-build/example.go",
    "reference/plugin-self-build/example.json",
    "tools/self-build-ci-contract-test.py",
    ".github/workflows/tei-self-build-qualification.yml",
)


def trigger_paths(source: str, event: str) -> tuple[str, ...]:
    lines = source.splitlines()
    header = f"  {event}:"
    try:
        start = lines.index(header)
    except ValueError as exc:
        raise AssertionError(f"{event} の起動条件がない") from exc
    paths: list[str] = []
    in_paths = False
    for line in lines[start + 1:]:
        if line and not line.startswith("    "):
            break
        if line == "    paths:":
            in_paths = True
            continue
        if in_paths:
            match = re.fullmatch(r"      - ([^\s#]+)", line)
            if match:
                paths.append(match.group(1))
            elif line.strip() and not line.lstrip().startswith("#"):
                break
    if not paths:
        raise AssertionError(f"{event} の監視対象パスがない")
    return tuple(paths)


def covers(path: str, patterns: tuple[str, ...]) -> bool:
    # この契約で使うGitHubパターンは、完全一致または階層全体の
    # 接頭辞（/**）のみ。理解できない構文を勝手に「検証済」にしない。
    for pattern in patterns:
        if pattern.startswith("!"):
            raise AssertionError("除外パターンは明示的な契約検討が必要")
        if pattern.endswith("/**"):
            if path.startswith(pattern[:-2]):
                return True
        elif "*" not in pattern and "?" not in pattern and path == pattern:
            return True
        elif "*" in pattern or "?" in pattern:
            raise AssertionError(f"未検証の監視グロブ: {pattern}")
    return False


class SelfBuildCIContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.source = WORKFLOW.read_text(encoding="utf-8")

    def test_push_and_pr_watch_the_same_source_inputs(self) -> None:
        pr = trigger_paths(self.source, "pull_request")
        push = trigger_paths(self.source, "push")
        self.assertEqual(set(pr), set(push), "PR/main間で資格確認の起動条件が不一致")
        for path in REQUIRED_INPUTS:
            with self.subTest(path=path):
                self.assertTrue(covers(path, pr), f"セルフビルド入力の監視漏れ: {path}")

    def test_main_push_is_not_cancelled_by_newer_main_push(self) -> None:
        self.assertRegex(self.source, r"(?m)^  cancel-in-progress: \$\{\{ github.event_name == 'pull_request' \}\}$")
        self.assertIn("group: tei-self-build-${{ github.event.pull_request.number || github.ref }}", self.source)
        self.assertIn("    branches: [main]", self.source)

    def test_execution_has_verifiable_evidence_and_safe_permissions(self) -> None:
        self.assertRegex(self.source, r"(?m)^permissions:\n  contents: read$")
        self.assertNotIn("pull_request_target:", self.source)
        self.assertNotIn("secrets.", self.source)
        self.assertIn("persist-credentials: false", self.source)
        self.assertIn("fetch-depth: 0", self.source)
        self.assertIn('test "$(git rev-parse HEAD)" = "$GITHUB_SHA"', self.source)
        self.assertIn("python3 tools/self-build-ci-contract-test.py", self.source)
        self.assertIn("go run ./tools/reference-transform-runtime-engine-build", self.source)
        self.assertIn("go run ./tools/reference-plugin-self-build", self.source)
        self.assertIn("tei-self-build-evidence.json", self.source)
        self.assertIn("tei-transform-engine-build-evidence.json", self.source)
        self.assertIn("if-no-files-found: error", self.source)

    def test_unknown_wildcard_is_not_assumed_to_be_safe(self) -> None:
        with self.assertRaises(AssertionError):
            covers("internal/core/a.go", ("internal/*",))


if __name__ == "__main__":
    unittest.main()
