#!/usr/bin/env python3
"""Finite, synthetic Function-connection checks; not TEI product conformance.

All capability/meaning facts below are fixture declarations, not accepted TEI
contracts, live grants, runtime effects, or universal logical proofs.
"""
from dataclasses import dataclass, replace
import unittest


@dataclass(frozen=True)
class Offer:
    subject: str | None = "order-H"
    grain: str | None = "line_entity"
    coverage: str | None = "complete"  # complete, partial, unknown
    scope: str | None = "order-H"
    unit: str | None = "JPY"
    metric_components: frozenset[str] = frozenset({"sum", "count"})
    purpose: str | None = "submit"
    version: int | None = 2
    clock: tuple[str, int] | None = ("discrete", 1)
    possible_results: frozenset[str] | None = frozenset({"confirmed"})


@dataclass(frozen=True)
class Need:
    subject: str = "order-H"
    grain: str = "line_entity"
    coverage: str = "complete"
    scope: str = "order-H"
    unit: str = "JPY"
    metric_components: frozenset[str] = frozenset({"sum", "count"})
    purpose: str = "submit"
    version: int = 2
    clock: tuple[str, int] = ("discrete", 1)
    accepted_results: frozenset[str] = frozenset({"confirmed"})
    require_live_grant: bool = False


@dataclass(frozen=True)
class Verdict:
    status: str  # contradicted, not_established, fixture_consistent
    reasons: tuple[str, ...]


def compare(offer: Offer, need: Need, *, clock_bridge: str | None = None) -> Verdict:
    """Only checks *declared* fixture claims; never upgrades to actual execution."""
    contradictions, unknowns = [], []
    for attr in ("subject", "grain", "scope", "unit", "purpose", "version"):
        available = getattr(offer, attr)
        required = getattr(need, attr)
        if available is None:
            unknowns.append(f"{attr}:unobserved")
        elif available != required:
            contradictions.append(f"{attr}:mismatch")
    if offer.coverage is None or offer.coverage == "unknown":
        unknowns.append("coverage:unobserved")
    elif need.coverage == "complete" and offer.coverage != "complete":
        contradictions.append("coverage:partial_cannot_guarantee_all")
    if not need.metric_components.issubset(offer.metric_components):
        unknowns.append("metric:required_intermediate_state_not_supplied")
    if offer.clock is None:
        unknowns.append("clock:unobserved")
    elif offer.clock != need.clock:
        if clock_bridge == "sample_hold_10_to_60" and offer.clock == ("fixed_hz", 10) and need.clock == ("fixed_hz", 60):
            unknowns.append("clock:explicit_bridge_requires_conformance")
        else:
            contradictions.append("clock:incompatible_without_validated_bridge")
    if offer.possible_results is None:
        unknowns.append("results:unobserved")
    elif not offer.possible_results.issubset(need.accepted_results):
        contradictions.append("results:unaccepted_possible_outcome")
    if need.require_live_grant:
        unknowns.append("grant:must_be_checked_by_actual_host")
    if contradictions:
        return Verdict("contradicted", tuple(contradictions + unknowns))
    if unknowns:
        return Verdict("not_established", tuple(unknowns))
    return Verdict("fixture_consistent", ("declared_witness_only",))


def check_trace(events: tuple[str, ...], claim: str) -> Verdict:
    """Find a counterexample in one trace, not a proof over all executions."""
    effects = {"inner_commit", "external_effect"}
    if claim == "cancel_guarantees_no_effect":
        if "outer_cancel" in events and effects.intersection(events):
            return Verdict("contradicted", ("cancel_does_not_undo_prior_or_concurrent_effect",))
    elif claim == "complete_requires_all_children_known_committed":
        if "outer_complete" in events and ("child_unknown" in events or "child_failed" in events):
            return Verdict("contradicted", ("partial_child_result_masked",))
    elif claim == "nested_preserves_unknown":
        if "child_unknown" in events and "outer_success_only" in events:
            return Verdict("contradicted", ("unknown_observation_hidden",))
    else:
        raise ValueError("unsupported local claim")
    return Verdict("not_established", ("one_trace_is_not_general_proof",))


def instance_isolation(placements: tuple[tuple[str, str], ...]) -> Verdict:
    """Pairs of (placement ID, state namespace). Sharing a namespace needs an explicit contract."""
    state_namespaces = [ns for _, ns in placements]
    ids = [i for i, _ in placements]
    if len(ids) != len(set(ids)):
        return Verdict("contradicted", ("placement_id_collision",))
    if len(state_namespaces) != len(set(state_namespaces)):
        return Verdict("contradicted", ("implicit_mutable_state_alias",))
    return Verdict("fixture_consistent", ("state_namespaces_disjoint_in_fixture",))


class CompositeContractFixtureTests(unittest.TestCase):
    def setUp(self):
        self.offer = Offer()
        self.need = Need()

    def test_ag01_partial_cannot_satisfy_full(self):
        x = compare(replace(self.offer, coverage="partial"), self.need)
        self.assertEqual(x.status, "contradicted")
        self.assertIn("coverage:partial_cannot_guarantee_all", x.reasons)

    def test_ag02_complete_matches_only_fixture(self):
        self.assertEqual(compare(self.offer, self.need).status, "fixture_consistent")

    def test_ag03_unobserved_coverage_is_not_proof(self):
        self.assertEqual(compare(replace(self.offer, coverage="unknown"), self.need).status, "not_established")

    def test_ag04_group_grain_is_not_line_grain(self):
        self.assertIn("grain:mismatch", compare(replace(self.offer, grain="aggregate_group"), self.need).reasons)

    def test_ag05_average_missing_sum_count(self):
        x = compare(replace(self.offer, metric_components=frozenset({"avg"})), self.need)
        self.assertEqual(x.status, "not_established")
        self.assertIn("metric:required_intermediate_state_not_supplied", x.reasons)

    def test_ag06_sum_count_is_only_declared_state(self):
        self.assertEqual(compare(self.offer, self.need).reasons, ("declared_witness_only",))

    def test_ag07_submit_and_cancel_are_not_interchangeable(self):
        x = compare(replace(self.offer, purpose="cancel"), self.need)
        self.assertIn("purpose:mismatch", x.reasons)

    def test_ag08_revision_upgrade_needs_requalification(self):
        self.assertIn("version:mismatch", compare(replace(self.offer, version=1), self.need).reasons)

    def test_ag09_ten_hz_not_sixty_hz_without_bridge(self):
        x = compare(replace(self.offer, clock=("fixed_hz", 10)), replace(self.need, clock=("fixed_hz", 60)))
        self.assertEqual(x.status, "contradicted")

    def test_ag10_explicit_bridge_still_requires_evidence(self):
        x = compare(replace(self.offer, clock=("fixed_hz", 10)), replace(self.need, clock=("fixed_hz", 60)), clock_bridge="sample_hold_10_to_60")
        self.assertEqual(x.status, "not_established")
        self.assertIn("clock:explicit_bridge_requires_conformance", x.reasons)

    def test_ag11_new_provider_failure_outcome_breaks_substitution(self):
        x = compare(replace(self.offer, possible_results=frozenset({"confirmed", "unknown"})), self.need)
        self.assertIn("results:unaccepted_possible_outcome", x.reasons)

    def test_ag12_no_runtime_grant_inferred(self):
        x = compare(self.offer, replace(self.need, require_live_grant=True))
        self.assertIn("grant:must_be_checked_by_actual_host", x.reasons)

    def test_ag13_same_definition_two_instances_not_same_state(self):
        self.assertEqual(instance_isolation((("I1", "state-A"), ("I2", "state-A"))).status, "contradicted")
        self.assertEqual(instance_isolation((("I1", "state-A"), ("I2", "state-B"))).status, "fixture_consistent")

    def test_ag14_cancellation_after_child_commit(self):
        x = check_trace(("inner_commit", "outer_cancel"), "cancel_guarantees_no_effect")
        self.assertIn("cancel_does_not_undo_prior_or_concurrent_effect", x.reasons)

    def test_ag15_partial_child_must_not_be_full_success(self):
        x = check_trace(("inner_commit", "child_unknown", "outer_complete"), "complete_requires_all_children_known_committed")
        self.assertEqual(x.status, "contradicted")

    def test_ag16_nested_projection_cannot_hide_unknown(self):
        x = check_trace(("child_unknown", "outer_success_only"), "nested_preserves_unknown")
        self.assertEqual(x.status, "contradicted")

    def test_ag17_one_successful_trace_not_universal_proof(self):
        x = check_trace(("inner_commit", "outer_complete"), "complete_requires_all_children_known_committed")
        self.assertEqual(x.status, "not_established")

    def test_ag18_unit_mismatch(self):
        x = compare(replace(self.offer, unit="USD"), self.need)
        self.assertIn("unit:mismatch", x.reasons)

    def test_ag19_scope_mismatch(self):
        x = compare(replace(self.offer, scope="order-B"), self.need)
        self.assertIn("scope:mismatch", x.reasons)

    def test_ag20_output_outcome_missing_is_unknown(self):
        self.assertEqual(compare(replace(self.offer, possible_results=None), self.need).status, "not_established")


if __name__ == "__main__":
    unittest.main(verbosity=2)
