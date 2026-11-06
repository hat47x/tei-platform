#!/usr/bin/env python3
"""M0 checker自体の合成負例。実CLI・実Provider試験とは別に記録する。"""
import copy
import unittest

from verify_cli import CheckError, inspect_observation

HEAD = "a" * 40
OPS = [
    "ledger.list.sales.note", "ledger.list.sales.order",
    "ledger.read.sales.order", "ledger.create.sales.order",
    "ledger.update.sales.order",
]
GAPS = [
    ("sales.note", "create", "missing-semantics"),
    ("sales.note", "read", "missing-semantics"),
    ("sales.order", "list", "conflict"),
    ("sales.order", "retire", "missing-input"),
]


def sample():
    version = {"schemaVersion": "tei.cli.version/v0",
               "source": "build-info", "vcsRevision": HEAD}
    discovery = {
        "schemaVersion": "tei.cli.discovery/v0",
        "scope": "selected-read-only-commands",
        "commands": [{"name": n, "effect": "read-only"} for n in
                     ("version", "model catalog", "inspect", "trace", "validate")],
    }
    validation = {
        "schemaVersion": "tei.cli.validate/v0",
        "ok": True,
        "diagnostics": [{"severity": "warning"}],
        "worldFingerprint": "sha256:world",
        "sourceIndexFingerprint": "sha256:source",
    }
    surface = {
        "schemaVersion": "tei.operations.operation-surface/v0",
        "effect": "read-only", "application": "ledger",
        "worldFingerprint": "sha256:world",
        "modelRevision": "sha256:revision",
        "rule": {"id": "tei.operation-surface/standard-crud",
                 "version": "v1", "fingerprint": "sha256:rule"},
        "authorityLimits": ["not a runtime authorization"],
        "operations": [{"id": op,
                        "binding": ("sales.order.register" if "create" in op
                                    else "sales.order.cancel" if "update" in op
                                    else "")} for op in OPS],
        "gaps": [{"target": target, "kind": kind, "reason": reason,
                  "detail": "expected synthetic gap"} for target, kind, reason in GAPS],
        "conflicts": [],
    }
    artifacts = {
        "inputRevision": "sha256:revision",
        "artifacts": [{"kind": "table-definition"}],
        "manifest": {"inputRevision": "sha256:revision",
                     "entries": [{"kind": "table-definition"}],
                     "digest": "sha256:manifest", "complete": True},
        "findings": [],
    }
    return [version, discovery, validation, surface, artifacts]


class VerifyCLIContractTests(unittest.TestCase):
    def run_check(self, case):
        return inspect_observation(*case, HEAD)

    def rejects(self, case):
        with self.assertRaises(CheckError):
            self.run_check(case)

    def test_reference_fixture_passes_without_grant(self):
        result = self.run_check(sample())
        self.assertEqual(result["operationCount"], 5)
        self.assertEqual(result["gapCount"], 4)
        self.assertEqual(result["actualHostAuthority"], "NOT_RUN")
        self.assertEqual(result["consumerArtifactDelivery"], "NOT_RUN")

    def test_world_drift_rejected(self):
        case = sample()
        case[3]["worldFingerprint"] = "sha256:other"
        self.rejects(case)

    def test_artifact_revision_drift_rejected(self):
        case = sample()
        case[4]["inputRevision"] = "sha256:other"
        self.rejects(case)

    def test_manifest_revision_drift_rejected(self):
        case = sample()
        case[4]["manifest"]["inputRevision"] = "sha256:other"
        self.rejects(case)

    def test_missing_gap_rejected(self):
        case = sample()
        case[3]["gaps"].pop()
        self.rejects(case)

    def test_write_binding_changed_rejected(self):
        case = sample()
        case[3]["operations"][-1]["binding"] = "unrelated.capability"
        self.rejects(case)

    def test_warning_is_retained_not_silent_success(self):
        result = self.run_check(sample())
        self.assertEqual(result["validationDiagnosticCount"], 1)

    def test_manifest_complete_is_not_delivery(self):
        case = sample()
        case[4]["manifest"]["complete"] = False
        case[4]["findings"] = [{"code": "INCOMPLETE"}]
        result = self.run_check(case)
        self.assertEqual(result["artifactFindingCount"], 1)
        self.assertFalse(result["manifestComplete"])
        self.assertEqual(result["consumerArtifactDelivery"], "NOT_RUN")

    def test_missing_binary_provenance_rejected(self):
        case = sample()
        del case[0]["vcsRevision"]
        self.rejects(case)

    def test_discovery_not_full_capability_list(self):
        case = sample()
        # An expanded discovery catalog is allowed only if its entries are
        # explicit; no entry is interpreted as a Host authorization.
        case[1]["commands"].append({"name": "operations", "effect": "read-only"})
        result = self.run_check(case)
        self.assertTrue(result["operationsAdvertised"])
        self.assertEqual(result["actualHostAuthority"], "NOT_RUN")

    def test_wrong_fixture_semantics_rejected(self):
        case = sample()
        case[3]["operations"][2]["id"] = "ledger.read.sales.note"
        self.rejects(case)


if __name__ == "__main__":
    unittest.main()
