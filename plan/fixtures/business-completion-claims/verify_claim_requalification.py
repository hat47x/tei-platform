#!/usr/bin/env python3
"""Synthetic claim-relative requalification of past evidence after a change.

This is a decision-table fixture, NOT a TEI revalidation engine. A 'reuse'
label means only that this *observed change* does not require re-examining that
claim only when independent Required-set closure AND actual consumption
coverage are both assumed complete in this synthetic fixture. Neither is
discovered here. Business completion, execution authority or disclosure
is never accepted by this fixture.
"""
from __future__ import annotations

from dataclasses import dataclass
import unittest

REUSE_CANDIDATE = "reuse-candidate-for-this-change"
RECHECK = "recheck-affected-claim"
UNRESOLVED = "unresolved-impact"


@dataclass(frozen=True)
class Claim:
    name: str
    subject: str
    depends_on: frozenset[str]
    # Synthetic assumptions, not evidence from actual Workspace readers.
    dependencies_closed: bool  # Independent Required-set closure
    consumption_closed: bool = False  # Actual consumed-input closure


@dataclass(frozen=True)
class Change:
    category: str | None
    subject: str | None
    # A disjoint proof is stronger than a different human-facing string key.
    relationship: str  # same | proven-disjoint | not-established


HISTORICAL_EFFECT = Claim(
    "historical-effect-K1", "order-1001",
    frozenset({"receipt-integrity", "effect-causality"}), True, True,
)
CURRENT_CONFIRMATION = Claim(
    "currently-usable-confirmation", "order-1001",
    frozenset({"cohort-history", "owner-rule", "provider-route"}), True, True,
)
CURRENT_DISCLOSURE = Claim(
    "current-consumer-disclosure", "order-1001",
    frozenset({"disclosure-grant", "consumer-artifact", "source-role"}), True, True,
)
SUBMIT_ELIGIBILITY = Claim(
    "current-submit-eligibility", "order-1001",
    frozenset({"cohort-history", "owner-rule", "provider-route", "action-binding", "host-grant"}), True, True,
)


def requalification(claim: Claim, change: Change) -> str:
    """Return only the review impact of one synthetic change, not truth/allow."""
    if not claim.dependencies_closed or not claim.consumption_closed or change.category is None:
        return UNRESOLVED
    if change.category not in claim.depends_on:
        return REUSE_CANDIDATE
    if change.relationship == "proven-disjoint":
        return REUSE_CANDIDATE
    if change.relationship == "same" and change.subject == claim.subject:
        return RECHECK
    return UNRESOLVED


class TestClaimRelativeRequalification(unittest.TestCase):
    def test_r01_unrelated_commentary_only_with_closed_dependency_set(self):
        change = Change("commentary", "order-1001", "same")
        for claim in (HISTORICAL_EFFECT, CURRENT_CONFIRMATION, CURRENT_DISCLOSURE, SUBMIT_ELIGIBILITY):
            with self.subTest(claim=claim.name):
                self.assertEqual(requalification(claim, change), REUSE_CANDIDATE)

    def test_r02_disclosure_revoke_does_not_undo_past_effect(self):
        change = Change("disclosure-grant", "order-1001", "same")
        self.assertEqual(requalification(CURRENT_DISCLOSURE, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), REUSE_CANDIDATE)

    def test_r03_cohort_add_remove_can_invalidate_current_confirmation_even_if_values_return(self):
        change = Change("cohort-history", "order-1001", "same")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), RECHECK)
        self.assertEqual(requalification(SUBMIT_ELIGIBILITY, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)

    def test_r04_role_plugin_change_affects_explanation_not_receipt(self):
        change = Change("source-role", "order-1001", "same")
        self.assertEqual(requalification(CURRENT_DISCLOSURE, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)

    def test_r05_provider_route_change_affects_current_witness_not_historical_effect(self):
        change = Change("provider-route", "order-1001", "same")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), RECHECK)
        self.assertEqual(requalification(SUBMIT_ELIGIBILITY, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)

    def test_r06_consumer_artifact_replaced_does_not_rewrite_historical_receipt(self):
        change = Change("consumer-artifact", "order-1001", "same")
        self.assertEqual(requalification(CURRENT_DISCLOSURE, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)

    def test_r07_open_dependency_inventory_does_not_justify_skip(self):
        open_claim = Claim("unbounded-rule-claim", "order-1001", frozenset(), False)
        self.assertEqual(requalification(open_claim, Change("commentary", "order-1001", "same")), UNRESOLVED)

    def test_r08_unknown_scope_cannot_justify_reuse(self):
        change = Change("cohort-history", None, "not-established")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), UNRESOLVED)

    def test_r09_proven_disjoint_scope_may_preserve_review_candidate(self):
        change = Change("cohort-history", "order-9999", "proven-disjoint")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), REUSE_CANDIDATE)

    def test_r10_different_key_without_disjoint_proof_is_not_enough(self):
        change = Change("cohort-history", "order-9999", "not-established")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), UNRESOLVED)

    def test_r11_unclassified_change_does_not_mean_no_dependency(self):
        change = Change(None, "order-1001", "same")
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), UNRESOLVED)

    def test_r12_host_grant_changed_and_historical_receipt_persists(self):
        change = Change("host-grant", "order-1001", "same")
        self.assertEqual(requalification(SUBMIT_ELIGIBILITY, change), RECHECK)
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), REUSE_CANDIDATE)

    def test_r13_historical_effect_evidence_tampering_requires_historical_review(self):
        change = Change("receipt-integrity", "order-1001", "same")
        self.assertEqual(requalification(HISTORICAL_EFFECT, change), RECHECK)
        self.assertEqual(requalification(CURRENT_CONFIRMATION, change), REUSE_CANDIDATE)

    def test_r14_matching_tax_type_does_not_prove_required_closure(self):
        # Historical X-TEA A/B preserves tax-value/std.decimal locally, not
        # the independent Required inventory of the entire Tax function.
        tax = Claim("tax-meaning", "master.tax_rate",
                    frozenset({"tax-rate", "calendar-rule"}), False, True)
        changed = Change("numbering-rule", "master.tax_rate", "same")
        self.assertEqual(requalification(tax, changed), UNRESOLVED)

    def test_r15_closed_requirements_without_consumption_cannot_skip(self):
        # A declared Required closure cannot replace evidence that the actual
        # Reader/Consumer consumed precisely the claimed inputs.
        tax = Claim("tax-meaning", "master.tax_rate",
                    frozenset({"tax-rate", "calendar-rule"}), True, False)
        changed = Change("numbering-rule", "master.tax_rate", "same")
        self.assertEqual(requalification(tax, changed), UNRESOLVED)

    def test_r16_both_closed_assumptions_permit_only_a_reuse_candidate(self):
        tax = Claim("tax-meaning", "master.tax_rate",
                    frozenset({"tax-rate", "calendar-rule"}), True, True)
        changed = Change("numbering-rule", "master.tax_rate", "same")
        self.assertEqual(requalification(tax, changed), REUSE_CANDIDATE)
        # This is not a Business Meaning or execution-eligibility decision.
        self.assertNotEqual(REUSE_CANDIDATE, "business-meaning-accepted")

    def test_r17_changed_required_tax_source_still_needs_recheck(self):
        tax = Claim("tax-meaning", "master.tax_rate",
                    frozenset({"tax-rate", "calendar-rule"}), True, True)
        changed = Change("tax-rate", "master.tax_rate", "same")
        self.assertEqual(requalification(tax, changed), RECHECK)


if __name__ == "__main__":
    unittest.main(verbosity=2)
