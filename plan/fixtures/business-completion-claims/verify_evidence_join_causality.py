#!/usr/bin/env python3
"""Self-contained synthetic evidence-splicing counterexamples for TEI design.

This is NOT an implementation of TEI EffectCommit/Owner acceptance or a
business completion validator. Names and fields are independent fixtures.
"""
from __future__ import annotations

from dataclasses import dataclass, replace
import unittest


@dataclass(frozen=True)
class Intent:
    action: str
    meaning_rev: str
    subject: str


@dataclass(frozen=True)
class Invocation:
    attempt: str
    action: str
    meaning_rev: str
    subject: str
    scope: str
    operation: str
    request_hash: str
    effect_ref: str


@dataclass(frozen=True)
class Receipt:
    scope: str
    operation: str
    request_hash: str
    effect_ref: str


@dataclass(frozen=True)
class Observation:
    # This is a trusted synthetic effect correlation witness, not a plain read.
    scope: str
    subject: str
    effect_ref: str
    observed_value: str


def candidate_join(intent: Intent, invocation: Invocation,
                   receipt: Receipt | None,
                   observation: Observation | None) -> tuple[str, tuple[str, ...]]:
    """Return a correlation assessment, never a Business completion verdict."""
    if receipt is None or observation is None:
        return "unresolved", ("missing corroborating receipt or observation",)
    checks = {
        "accepted action": intent.action == invocation.action,
        "meaning revision": intent.meaning_rev == invocation.meaning_rev,
        "intended subject": intent.subject == invocation.subject,
        "receipt scope": invocation.scope == receipt.scope,
        "receipt operation": invocation.operation == receipt.operation,
        "submitted request bytes": invocation.request_hash == receipt.request_hash,
        "receipt effect link": invocation.effect_ref == receipt.effect_ref,
        "observation scope": receipt.scope == observation.scope,
        "observation subject": intent.subject == observation.subject,
        "observation effect link": receipt.effect_ref == observation.effect_ref,
    }
    mismatches = tuple(key for key, matches in checks.items() if not matches)
    return ("contradicted", mismatches) if mismatches else ("correlation_candidate", ())


def fixture() -> tuple[Intent, Invocation, Receipt, Observation]:
    intent = Intent("submit", "m1", "order-1001")
    invocation = Invocation("attempt-A", "submit", "m1", "order-1001",
                            "scope-A", "operation-7", "raw-request-hash-1", "effect-E1")
    receipt = Receipt("scope-A", "operation-7", "raw-request-hash-1", "effect-E1")
    observation = Observation("scope-A", "order-1001", "effect-E1", "submitted")
    return intent, invocation, receipt, observation


class SyntheticCorrelationPressure(unittest.TestCase):
    def test_matched_causal_chain_is_a_candidate_not_completion(self) -> None:
        self.assertEqual(candidate_join(*fixture()), ("correlation_candidate", ()))

    def test_replay_attempt_may_differ_without_changing_operation_identity(self) -> None:
        intent, invocation, receipt, observation = fixture()
        replay = replace(invocation, attempt="attempt-B")
        self.assertEqual(candidate_join(intent, replay, receipt, observation)[0],
                         "correlation_candidate")

    def test_individually_plausible_evidence_may_be_spliced(self) -> None:
        intent, invocation, receipt, observation = fixture()
        variants = {
            "wrong business action": (intent, replace(invocation, action="cancel"), receipt, observation),
            "wrong meaning revision": (intent, replace(invocation, meaning_rev="m2"), receipt, observation),
            "wrong intended subject": (replace(intent, subject="order-1002"), invocation, receipt, observation),
            "same operation label, wrong scope": (intent, invocation, replace(receipt, scope="scope-B"), observation),
            "wrong operation": (intent, invocation, replace(receipt, operation="operation-8"), observation),
            "different submitted request byte": (intent, invocation, replace(receipt, request_hash="raw-request-hash-2"), observation),
            "receipt for different effect": (intent, invocation, replace(receipt, effect_ref="effect-E2"), observation),
            "observation from other scope": (intent, invocation, receipt, replace(observation, scope="scope-B")),
            "observation of other subject": (intent, invocation, receipt, replace(observation, subject="order-1002")),
            "same current value, other effect": (intent, invocation, receipt, replace(observation, effect_ref="effect-E2")),
        }
        for name, evidence in variants.items():
            with self.subTest(name=name):
                status, diagnostics = candidate_join(*evidence)
                self.assertEqual(status, "contradicted")
                self.assertTrue(diagnostics)

    def test_missing_correlating_witness_is_unknown_not_no_effect(self) -> None:
        intent, invocation, receipt, observation = fixture()
        for name, r, o in (("missing receipt", None, observation),
                            ("missing observer", receipt, None)):
            with self.subTest(name=name):
                status, _ = candidate_join(intent, invocation, r, o)
                self.assertEqual(status, "unresolved")


if __name__ == "__main__":
    unittest.main(verbosity=2)
