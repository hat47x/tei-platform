#!/usr/bin/env python3
"""Synthetic, read-only Construction Context conformance examples (#2516).

Run: python3 verify_synthetic.py

This is not the TEI CLI, an Oracle of accepted business meaning, a real
authorization layer, or a production Claim/Context engine.
"""

import json
import unittest
from dataclasses import dataclass
from decimal import Decimal


@dataclass(frozen=True)
class Observation:
    key: str
    value: object
    provider: str
    snapshot: str
    source_revision: str
    verified_read: bool = True
    complete: bool = True
    snapshot_guaranteed: bool = True


def qualify_inputs(observations, required_keys, *, require_same_snapshot=True):
    """Minimal synthetic claim-relative witness; not a provider guarantee."""
    selected = {item.key: item for item in observations if item.key in required_keys}
    if len(selected) != len(required_keys):
        return "unknown"  # missing is not a NULL value or a negative claim
    selected = [selected[key] for key in required_keys]
    if not all(item.verified_read and item.complete for item in selected):
        return "unknown"
    if require_same_snapshot:
        if not all(item.snapshot_guaranteed for item in selected):
            return "unknown"
        if len({(item.provider, item.snapshot, item.source_revision) for item in selected}) != 1:
            return "incompatible"
    return "qualified"


def comparable(a, b):
    """Only compare values AFTER checking their claim identity and conditions."""
    dimensions = (
        "claim", "subject", "scope", "grain", "operation", "snapshot",
        "rule_revision", "unit", "effective_at", "read_qualified",
    )
    if not a.get("read_qualified") or not b.get("read_qualified"):
        return "unverified"
    if any(a.get(dim) != b.get(dim) for dim in dimensions):
        return "not_comparable"
    return "agree" if a["value"] == b["value"] else "disagreement"


def derive_subset_claim(*, subset_verified, observed, all_pass):
    """One-way existential witness and universal counterexample only."""
    if not subset_verified:
        return "unknown"
    if observed:
        return "exists_in_full"
    # A complete scan of a subset does not prove absence in the full population.
    return "unknown" if all_pass else "unknown"


def derive_universal_claim(*, subset_verified, counterexample_in_subset):
    if not subset_verified:
        return "unknown"
    return "false_in_full" if counterexample_in_subset else "unknown"


def public_measure(*, visible, hidden, disclose_total=False):
    """Fixture-only disclosure contrast, not a policy implementation.

    The public result must not expose hidden data when the total is not
    explicitly released. The policy decision is an input, not inferred here.
    """
    response = {"scope": "visible", "average": str(sum(visible, Decimal(0)) / len(visible))}
    if disclose_total:
        values = visible + hidden
        response["released_total_average"] = str(sum(values, Decimal(0)) / len(values))
    return json.dumps(response, sort_keys=True, ensure_ascii=False)


def publication_coherent(manifest_generation, file_generations):
    return all(gen == manifest_generation for gen in file_generations)


def actual_tool_bound(discovered, invoked):
    """Intended command discovery is not evidence of actual invocation."""
    fields = ("executable", "workspace", "checkout", "source_revision")
    return all(discovered.get(field) == invoked.get(field) for field in fields)


class SyntheticContextTest(unittest.TestCase):
    def test_j01_consistent_r0(self):
        reads = [Observation("stock", 120, "db", "R0", "S0"), Observation("reserved", 100, "db", "R0", "S0")]
        self.assertEqual(qualify_inputs(reads, ["stock", "reserved"]), "qualified")
        self.assertEqual(reads[0].value - reads[1].value, 20)

    def test_j01_consistent_r1(self):
        reads = [Observation("stock", 110, "db", "R1", "S1"), Observation("reserved", 90, "db", "R1", "S1")]
        self.assertEqual(qualify_inputs(reads, ["stock", "reserved"]), "qualified")
        self.assertEqual(reads[0].value - reads[1].value, 20)

    def test_j01_mixed_snapshot_rejected_even_if_arithmetic_succeeds(self):
        reads = [Observation("stock", 120, "db", "R0", "S0"), Observation("reserved", 90, "db", "R1", "S1")]
        self.assertEqual(reads[0].value - reads[1].value, 30)  # invalid as a jointly evidenced state
        self.assertEqual(qualify_inputs(reads, ["stock", "reserved"]), "incompatible")

    def test_j04_only_required_slice_controls_claim(self):
        reads = [Observation("D1", (100, 100), "db", "R1", "S1"), Observation("D2", (50,), "db", "R2", "S2", complete=False)]
        self.assertEqual(qualify_inputs(reads, ["D1"]), "qualified")
        self.assertEqual(sum(reads[0].value), 200)
        self.assertEqual(qualify_inputs(reads, ["D1", "D2"]), "unknown")

    def test_j01_unobserved_provider_guarantee_not_inferred(self):
        reads = [Observation("stock", 120, "db", "R0", "S0", snapshot_guaranteed=False), Observation("reserved", 100, "db", "R0", "S0", snapshot_guaranteed=False)]
        self.assertEqual(qualify_inputs(reads, ["stock", "reserved"]), "unknown")

    def test_w8_same_number_not_same_claim(self):
        a = dict(claim="line-total", subject="order-H", scope="D1-2/2", grain="line", operation="SUM", snapshot="R0", rule_revision="v1", unit="JPY", effective_at="T0", read_qualified=True, value=200)
        b = dict(a, claim="joined-total", scope="D1-D2-1/3")
        self.assertEqual(comparable(a, b), "not_comparable")

    def test_w8_different_number_not_a_contradiction_across_scope(self):
        a = dict(claim="average", subject="cohort", scope="visible", grain="row", operation="AVG", snapshot="R0", rule_revision="v1", unit="unit", effective_at="T0", read_qualified=True, value=15)
        b = dict(a, scope="all", value=Decimal("7.5"))
        self.assertEqual(comparable(a, b), "not_comparable")

    def test_w8_same_claim_disagreement_requires_oracle(self):
        a = dict(claim="line-total", subject="order-H", scope="D1-all", grain="line", operation="SUM", snapshot="R0", rule_revision="v1", unit="JPY", effective_at="T0", read_qualified=True, value=200)
        self.assertEqual(comparable(a, dict(a, value=201)), "disagreement")
        self.assertEqual(comparable(a, dict(a, read_qualified=False)), "unverified")

    def test_w9_subset_positive_witness_and_universal_counterexample(self):
        self.assertEqual(derive_subset_claim(subset_verified=True, observed=True, all_pass=True), "exists_in_full")
        self.assertEqual(derive_subset_claim(subset_verified=True, observed=False, all_pass=True), "unknown")
        self.assertEqual(derive_universal_claim(subset_verified=True, counterexample_in_subset=True), "false_in_full")
        self.assertEqual(derive_universal_claim(subset_verified=True, counterexample_in_subset=False), "unknown")

    def test_w9_near_equality_not_transitive(self):
        a, b, c, epsilon = map(Decimal, ("0", "0.006", "0.012", "0.01"))
        self.assertLessEqual(abs(a - b), epsilon)
        self.assertLessEqual(abs(b - c), epsilon)
        self.assertGreater(abs(a - c), epsilon)

    def test_w10_tool_discovery_not_actual_invocation(self):
        a = {"executable": "tei-A", "workspace": "W", "checkout": "C1", "source_revision": "S1"}
        self.assertFalse(actual_tool_bound(a, dict(a, executable="tei-B")))
        self.assertFalse(actual_tool_bound(a, dict(a, workspace="W2")))
        self.assertTrue(actual_tool_bound(a, dict(a)))

    def test_w10_manifest_and_file_mixed_generation(self):
        self.assertTrue(publication_coherent("R1", ["R1", "R1"]))
        self.assertFalse(publication_coherent("R1", ["R1", "R2"]))

    def test_w2_two_worlds_public_observation_same(self):
        visible = (Decimal(10), Decimal(20))
        x = (Decimal(0), Decimal(0))
        y = (Decimal(100), Decimal(100))
        public_x = public_measure(visible=visible, hidden=x)
        public_y = public_measure(visible=visible, hidden=y)
        self.assertEqual(public_x, public_y)
        self.assertNotIn("hidden", public_x)
        # Explicit synthetic declassification changes the output; real policy is out of scope.
        self.assertNotEqual(public_measure(visible=visible, hidden=x, disclose_total=True), public_measure(visible=visible, hidden=y, disclose_total=True))

    def test_w10_missing_required_read_not_a_null(self):
        d1 = Observation("D1", (100,), "db", "R0", "S0")
        self.assertEqual(qualify_inputs([d1], ["D1", "D2"]), "unknown")


if __name__ == "__main__":
    unittest.main(verbosity=2)
