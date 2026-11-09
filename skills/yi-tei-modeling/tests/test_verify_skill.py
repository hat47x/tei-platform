"""Isolated fixture tests for the Product Skill static consistency checker."""
from __future__ import annotations

import importlib.util
from pathlib import Path
import tempfile
import unittest

MODULE = Path(__file__).resolve().parents[1] / "verify_skill.py"
spec = importlib.util.spec_from_file_location("verify_skill", MODULE)
assert spec and spec.loader
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


class SkillStaticCheckTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.put("cmd/tei/cli_contract.go", '''
SchemaVersion: "tei.cli.discovery/v0",
Scope: "selected-read-only-commands",
{Name: "version", Effect: "read-only"},
{Name: "model catalog", Effect: "read-only"},
{Name: "inspect", Effect: "read-only"},
{Name: "trace", Effect: "read-only"},
{Name: "validate", Effect: "read-only"},
''')
        self.put("cmd/tei/main.go", 'if command == "commands" {}')
        self.put("skills/yi-tei-modeling/SKILL.md", '''---
name: yi-tei-modeling
description: >-
  Read-only modeling.
---

NOT RUN. [Reference](references/read-only-model-investigation.md)
''')
        self.put("skills/yi-tei-modeling/references/read-only-model-investigation.md", '''[Doc](../../../docs/getting-started.md)

result.snapshotFingerprint result.trace.snapshotFingerprint result.trace.graphFingerprint result.trace.continuation.fingerprint -expected-fingerprint

```sh
go run ./cmd/tei commands
go run ./cmd/tei model catalog -workspace reference/workspace
```
''')
        self.put("docs/getting-started.md", "Getting started")
        self.put("plan/design/TEI_PRODUCT_SKILL_HARNESS_BINDING_2026-11.md", "Design")

    def put(self, path: str, text: str) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(text, encoding="utf-8")

    def has_error(self, term: str) -> None:
        errors = checker.check_repository(self.root)
        self.assertTrue(any(term in item for item in errors), errors)

    def test_complete_fixture_passes(self) -> None:
        self.assertEqual(checker.check_repository(self.root), [])

    def test_wrong_skill_name_is_rejected(self) -> None:
        p = self.root / "skills/yi-tei-modeling/SKILL.md"
        self.put(str(p.relative_to(self.root)), p.read_text().replace("name: yi-tei-modeling", "name: other"))
        self.has_error("name must match")

    def test_missing_cli_read_only_declaration_is_rejected(self) -> None:
        p = self.root / "cmd/tei/cli_contract.go"
        self.put(str(p.relative_to(self.root)), p.read_text().replace('"trace", Effect: "read-only"', '"trace", Effect: "write"'))
        self.has_error("missing or not read-only: trace")

    def test_broken_link_is_rejected(self) -> None:
        p = self.root / "skills/yi-tei-modeling/SKILL.md"
        self.put(str(p.relative_to(self.root)), p.read_text().replace("references/read-only-model-investigation.md", "references/missing.md"))
        self.has_error("broken link")

    def test_undeclared_tei_example_is_rejected(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() + "\n```sh\ngo run ./cmd/tei database-migrate -apply\n```\n")
        self.has_error("unverified/read-write TEI example")

    def test_chained_tei_write_cannot_hide_after_read_only_command(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ngo run ./cmd/tei validate -json;go run ./cmd/tei artifacts -out /tmp/out\n```\n")
        self.has_error("composite TEI shell example")

    def test_shell_and_operator_cannot_hide_write_command(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ntei validate -json && tei artifacts -out /tmp/out\n```\n")
        self.has_error("composite TEI shell example")

    def test_pipe_from_tei_is_not_automatically_qualified_read_only(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ntei validate -json | tee /tmp/validate.json\n```\n")
        self.has_error("composite TEI shell example")

    def test_wrapped_tei_invocation_is_not_silently_ignored(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\nenv TEI_ENV=1 tei artifacts -out /tmp/out\n```\n")
        self.has_error("unverified TEI invocation context")

    def test_known_read_only_command_with_unverified_flag_is_rejected(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ngo run ./cmd/tei validate -workspace reference/workspace -out /tmp/result\n```\n")
        self.has_error("unverified CLI flag/argument: -out")

    def test_known_command_with_shell_chaining_is_rejected(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ngo run ./cmd/tei validate -workspace reference/workspace && echo accepted\n```\n")
        self.has_error("composite TEI shell example")

    def test_documented_trace_flags_are_accepted(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text() +
                 "\n```sh\ngo run ./cmd/tei trace -workspace reference/workspace "
                 "-scope ref -subject-kind semantic-relation -subject-id sales.order.customer "
                 "-direction both -mode full -limit 10 -offset 0 -expected-fingerprint abc\n```\n")
        self.assertEqual(checker.check_repository(self.root), [])

    def test_trace_token_roles_are_required(self) -> None:
        p = self.root / "skills/yi-tei-modeling/references/read-only-model-investigation.md"
        self.put(str(p.relative_to(self.root)), p.read_text().replace("result.trace.snapshotFingerprint", "missing"))
        self.has_error("reference missing trace pagination binding")

    def test_catalog_scope_drift_is_rejected(self) -> None:
        p = self.root / "cmd/tei/cli_contract.go"
        self.put(str(p.relative_to(self.root)), p.read_text().replace("selected-read-only-commands", "all-capabilities"))
        self.has_error("discovery scope changed")


if __name__ == "__main__":
    unittest.main()
