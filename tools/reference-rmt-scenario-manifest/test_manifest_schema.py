#!/usr/bin/env python3
"""Optional RMT manifest schema conformance test (requires jsonschema >= 4.18).

Schema acceptance does not establish provenance, observer conformance,
human acceptance, runtime behavior, or permission to publish.
"""
from __future__ import annotations

import copy
import json
import unittest
from pathlib import Path

from jsonschema import Draft202012Validator

REPO = Path(__file__).resolve().parents[2]
FIXTURES = REPO / "reference" / "reference-model-transfer"
SCHEMA = json.loads((FIXTURES / "scenario-manifest-v0.schema.json").read_text(encoding="utf-8"))
EXAMPLE = json.loads((FIXTURES / "scenario-manifest-v0.example.json").read_text(encoding="utf-8"))
HEX = lambda character: "sha256:" + character * 64


def externally_observed_candidate() -> dict:
    """Structurally valid specimen only; fabricated receipt digests are not evidence."""
    value = copy.deepcopy(EXAMPLE)
    value["origin"] = "external-model-private"
    value["sourceBinding"] = {
        "artifactDigest": HEX("1"),
        "sourceProfile": "xead-system-1.2",
        "transcriptionContractDigest": HEX("2"),
    }
    value["teiRevision"] = "a" * 40
    value["acceptedSourceDigest"] = HEX("3")
    value["acceptanceDecisionReceiptDigest"] = HEX("4")
    value["teiExecution"] = {"status": "observed", "receiptDigest": HEX("5")}
    value["originalExecution"] = {"status": "observed", "receiptDigest": HEX("6")}
    value["preconditionAlignment"] = {"status": "aligned", "receiptDigest": HEX("7")}
    value["comparison"] = {
        "status": "consistent-on-observed-axes",
        "axes": ["business-result"],
        "comparisonEvidenceDigest": HEX("8"),
    }
    return value


class ScenarioManifestConformance(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        Draft202012Validator.check_schema(SCHEMA)
        cls.validator = Draft202012Validator(SCHEMA)

    def valid(self, value: dict) -> None:
        errors = sorted(self.validator.iter_errors(value), key=lambda e: str(e.path))
        self.assertFalse(errors, "\n".join(f"{list(e.path)}: {e.message}" for e in errors))

    def invalid(self, value: dict) -> None:
        self.assertTrue(list(self.validator.iter_errors(value)), value)

    def test_unexecuted_example(self) -> None:
        self.valid(EXAMPLE)

    def test_external_observations_are_structurally_expressible(self) -> None:
        self.valid(externally_observed_candidate())

    def test_tei_only_observation_does_not_claim_equivalence(self) -> None:
        value = copy.deepcopy(EXAMPLE)
        value["teiRevision"] = "b" * 40
        value["acceptedSourceDigest"] = HEX("c")
        value["acceptanceDecisionReceiptDigest"] = HEX("d")
        value["teiExecution"] = {"status": "observed", "receiptDigest": HEX("e")}
        self.valid(value)

    def test_untrusted_publication_pass_is_not_accepted(self) -> None:
        for value in (EXAMPLE, externally_observed_candidate()):
            with self.subTest(origin=value["origin"]):
                bad = copy.deepcopy(value)
                bad["publicationReview"] = {"status": "passed", "receiptDigest": HEX("a")}
                self.invalid(bad)

    def test_negative_controls(self) -> None:
        source = externally_observed_candidate()
        mutations = {
            "original trace missing": lambda v: v.update(originalExecution={"status": "not-run"}),
            "no TEI trace": lambda v: v.update(teiExecution={"status": "not-run"}),
            "precondition not aligned": lambda v: v["preconditionAlignment"].update(status="not-aligned"),
            "no comparable axes": lambda v: v["comparison"].update(axes=[]),
            "comparison receipt missing": lambda v: v["comparison"].pop("comparisonEvidenceDigest"),
            "TEI receipt missing": lambda v: v["teiExecution"].pop("receiptDigest"),
            "original receipt missing": lambda v: v["originalExecution"].pop("receiptDigest"),
            "accepted source missing": lambda v: v.pop("acceptedSourceDigest"),
            "review decision receipt missing": lambda v: v.pop("acceptanceDecisionReceiptDigest"),
            "source binding missing": lambda v: v.pop("sourceBinding"),
            "original source digest malformed": lambda v: v["sourceBinding"].update(artifactDigest="sha256:BAD"),
            "source profile path-like": lambda v: v["sourceBinding"].update(sourceProfile="private/secret"),
            "unknown root field": lambda v: v.update({"privateLocator": "/Model/secret"}),
            "unknown nested field": lambda v: v["originalExecution"].update({"dataSnapshot": "confidential"}),
            "empty scenario ID": lambda v: v.update(scenarioId=""),
            "unbounded class ID": lambda v: v.update(classContractId="x" * 500),
            "missing authority limit": lambda v: v["authorityLimits"].pop(),
        }
        for label, mutate in mutations.items():
            with self.subTest(label=label):
                item = copy.deepcopy(source)
                mutate(item)
                self.invalid(item)

    def test_incomparable_incomplete_evidence_remains_expressible(self) -> None:
        value = externally_observed_candidate()
        value["originalExecution"] = {"status": "not-run"}
        value["preconditionAlignment"] = {"status": "not-assessed"}
        value["comparison"] = {"status": "incomparable", "axes": []}
        self.valid(value)

    def test_unknown_original_result_cannot_be_claimed_consistent(self) -> None:
        value = externally_observed_candidate()
        value["originalExecution"] = {"status": "inconclusive"}
        value["preconditionAlignment"] = {"status": "unknown"}
        value["comparison"] = {"status": "unresolved", "axes": []}
        self.valid(value)
        value["comparison"] = {
            "status": "consistent-on-observed-axes",
            "axes": ["business-result"],
            "comparisonEvidenceDigest": HEX("a"),
        }
        self.invalid(value)

    def test_no_false_evidence_from_synthetic(self) -> None:
        value = externally_observed_candidate()
        value["origin"] = "independent-synthetic"
        self.invalid(value)

    def test_alignment_cannot_precede_original_execution(self) -> None:
        value = copy.deepcopy(EXAMPLE)
        value["preconditionAlignment"] = {"status": "aligned", "receiptDigest": HEX("a")}
        self.invalid(value)

    def test_not_evaluated_cannot_include_comparison_receipt(self) -> None:
        value = copy.deepcopy(EXAMPLE)
        value["comparison"]["comparisonEvidenceDigest"] = HEX("a")
        self.invalid(value)

    def test_pending_publication_does_not_carry_approval_receipt(self) -> None:
        value = copy.deepcopy(EXAMPLE)
        value["publicationReview"]["receiptDigest"] = HEX("b")
        self.invalid(value)


if __name__ == "__main__":
    unittest.main()
