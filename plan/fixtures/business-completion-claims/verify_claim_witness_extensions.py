#!/usr/bin/env python3
"""Candidate evidence-extension index for synthetic BC two-world counterexamples.

Only identifies *one observed difference* between the existing two worlds.
It never certifies the witness source, completeness, authorization, action
meaning, target execution, or the truth of a production Business claim.
"""
from __future__ import annotations

from dataclasses import dataclass
import unittest

from verify_claim_projections import CASES


@dataclass(frozen=True)
class CandidateWitness:
    field: str
    origin_to_verify: str
    qualification_needed: str


WITNESSES = {
    "BC-01": CandidateWitness("rows", "complete detail Read", "scope/grain, full set and snapshot"),
    "BC-05": CandidateWitness("external_accepted", "recipient acknowledgement", "recipient identity and acknowledgement binding"),
    "BC-07": CandidateWitness("recovery_for", "authorized recovery association", "operation lineage and accepted recovery policy"),
    "BC-08": CandidateWitness("relevant_edits", "change history", "target scope and entire confirmation interval"),
    "BC-09": CandidateWitness("membership_edits", "relation membership history", "cohort scope, interval and late-arriving edits"),
    "BC-10": CandidateWitness("effect_target", "provider effect evidence", "actual invocation/request/effect identity join"),
    "BC-11": CandidateWitness("historical_output", "historical result or reproducible output", "original result version and trustworthy source"),
    "BC-12": CandidateWitness("amounts", "underlying detail projection", "underlying grain, completeness and allowed disclosure"),
    "BC-13": CandidateWitness("executed_action", "actual capability invocation", "accepted action meaning and versioned binding"),
    "BC-14": CandidateWitness("parent_write_committed", "independent transaction observation", "parent pending, child commit and rollback causal trace"),
    "BC-15": CandidateWitness("loaded_digest", "consumer loaded artifact", "generation manifest, actual bytes and consumer runtime"),
    "BC-16": CandidateWitness("persisted_status", "independent durable read", "request/commit identity and observation time"),
}


def candidate_evidence(world: dict, spec: CandidateWitness, *, source_qualified: bool,
                       scope_covered: bool, temporal_coverage: bool,
                       allowed_to_disclose: bool) -> object | None:
    """Illustrative Consumer read; None says *unqualified*, never absent in DB."""
    if not (source_qualified and scope_covered and temporal_coverage and allowed_to_disclose):
        return None
    return world[spec.field]


class WitnessExtensionPressure(unittest.TestCase):
    def test_every_two_world_case_has_a_distinguishing_candidate(self) -> None:
        case_ids = [case.case_id for case in CASES]
        self.assertEqual(len(case_ids), len(set(case_ids)))
        self.assertEqual(set(case_ids), set(WITNESSES))
        for case in CASES:
            with self.subTest(case=case.case_id):
                spec = WITNESSES[case.case_id]
                self.assertEqual(case.visible(case.a), case.visible(case.b))
                self.assertIn(spec.field, case.a)
                self.assertIn(spec.field, case.b)
                self.assertNotEqual(case.a[spec.field], case.b[spec.field])
                self.assertTrue(spec.origin_to_verify and spec.qualification_needed)

    def test_partial_or_non_disclosable_evidence_stays_unqualified(self) -> None:
        for case in CASES:
            with self.subTest(case=case.case_id):
                spec = WITNESSES[case.case_id]
                for missing in ("source_qualified", "scope_covered", "temporal_coverage", "allowed_to_disclose"):
                    flags = dict.fromkeys(("source_qualified", "scope_covered", "temporal_coverage", "allowed_to_disclose"), True)
                    flags[missing] = False
                    self.assertIsNone(candidate_evidence(case.a, spec, **flags))
                    self.assertIsNone(candidate_evidence(case.b, spec, **flags))
                positive = dict(source_qualified=True, scope_covered=True,
                                temporal_coverage=True, allowed_to_disclose=True)
                self.assertNotEqual(candidate_evidence(case.a, spec, **positive),
                                    candidate_evidence(case.b, spec, **positive))


if __name__ == "__main__":
    unittest.main(verbosity=2)
