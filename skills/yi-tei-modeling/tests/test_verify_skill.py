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

result.snapshotFingerprint result.trace.snapshotFingerprint -expected-fingerprint

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
