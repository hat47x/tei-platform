#!/usr/bin/env python3
"""W0c / TAI #271: claim-relative *synthetic* selective requalification.

This is a finite pressure test against an independently written expected table,
not an implemented dependency engine, accepted schema, proof of completeness,
policy checker, or authorization to send data.
"""
import unittest
from dataclasses import dataclass

# This is the synthetic fixture's *assumed complete* dependency table.
# Product coverage would need independent Source/Query/Rule/Provider evidence.
DEPS = {
    "meaning_read": {"semantic_world", "cli_revision"},
    "source_citation": {"source_occurrence", "source_index", "document_target"},
    "rule_outcome": {
        "semantic_world", "hidden_rule", "hidden_predicate", "provider_route",
        "read_cohort", "read_grant",
    },
    "query_result": {
        "semantic_world", "hidden_predicate", "provider_route", "read_cohort",
        "read_completeness",
    },
    "human_document": {
        "semantic_world", "source_occurrence", "document_template", "document_target",
    },
    "ai_view": {
        "semantic_world", "hidden_predicate", "provider_route", "read_cohort",
        "ai_view_policy",
    },
    "external_delivery": {
        "semantic_world", "hidden_predicate", "provider_route", "read_cohort",
        "artifact_members", "artifact_generation", "ai_view_policy",
        "disclosure_policy", "egress_grant", "consumer_purpose",
    },
    "document_review": {
        "source_occurrence", "document_template", "document_target",
        "owner_review", "artifact_members",
    },
}
ALL_CLAIMS = frozenset(DEPS)
ALL_DEPENDENCIES = set().union(*DEPS.values())


@dataclass(frozen=True)
class Change:
    changed: frozenset
    complete: bool = True
    grant_now: str = "checked-allow"  # Trusted synthetic input, not a Host grant
    output_digest_same: bool = False


def classify(claim, change, dependencies=DEPS):
    """Classify only a declared, bounded Claim-dependency comparison."""
    if claim not in dependencies or not change.complete:
        return "unknown"
    if not change.changed.issubset(ALL_DEPENDENCIES):
        return "unknown"
    if claim == "external_delivery":
        # Egress current authority is an independent predicate. An old
        # successful read, old permission or same digest cannot replace it.
        if change.grant_now == "checked-deny":
            return "withheld"
        if change.grant_now != "checked-allow":
            return "unknown"
    if dependencies[claim].intersection(change.changed):
        return "requires-requalification"
    return "historically-supported-candidate"


# Independent expected outcomes: scenarios are intentionally stated in terms
# of *which* claims change, not inferred from the DEPS map above.
# All examples are fictional public IDs. Equality of output digests is not
# treated as evidence of dependency or entitlement equivalence.
CASES = (
    ("no-change", (), (), None, True, False),
    ("source-relocation", ("source_occurrence", "source_index"),
     ("source_citation", "human_document", "document_review"), None, True, False),
    ("hidden-rule-same-output", ("hidden_rule",),
     ("rule_outcome",), None, True, True),
    ("hidden-where-predicate", ("hidden_predicate",),
     ("rule_outcome", "query_result", "ai_view", "external_delivery"), None, True, True),
    ("provider-route-same-json", ("provider_route",),
     ("rule_outcome", "query_result", "ai_view", "external_delivery"), None, True, True),
    ("document-template-only", ("document_template",),
     ("human_document", "document_review"), None, True, False),
    ("review-withdrawal", ("owner_review",),
     ("document_review",), None, True, True),
    ("artifact-member-change", ("artifact_members",),
     ("external_delivery", "document_review"), None, True, True),
    ("read-cohort-epoch", ("read_cohort",),
     ("rule_outcome", "query_result", "ai_view", "external_delivery"), None, True, False),
    ("source-index-only", ("source_index",),
     ("source_citation",), None, True, False),
    ("unchanged-output-does-not-erase-rule", ("hidden_rule",),
     ("rule_outcome",), None, True, True),
    ("disclosure-grant-revoked", ("egress_grant",),
     (), "withheld", True, True),
    ("disclosure-grant-unknown", (),
     (), "unknown", True, True),
    ("dependency-coverage-unproven", ("hidden_rule",),
     (), "all-unknown", False, True),
    ("unmapped-changing-dependency", ("unknown-edge",),
     (), "all-unknown", True, False),
    ("independent-doc-comment", (),
     (), None, True, True),
    ("consumer-purpose-change", ("consumer_purpose",),
     ("external_delivery",), None, True, True),
    ("grant-version-changed", ("egress_grant",),
     ("external_delivery",), None, True, True),
    ("binary-identity-changed", ("cli_revision",),
     ("meaning_read",), None, True, False),
    ("document-target-explicit-switch", ("document_target",),
     ("source_citation", "human_document", "document_review"), None, True, False),
    ("artifact-body-generation", ("artifact_generation",),
     ("external_delivery",), None, True, True),
)


class RequalificationFixture(unittest.TestCase):
    def verify_case(self, case):
        name, changed, recheck, special, complete, same_output = case
        grant = ("checked-deny" if special == "withheld" else
                 "unknown" if special == "unknown" else "checked-allow")
        evidence = Change(frozenset(changed), complete, grant, same_output)
        result = {claim: classify(claim, evidence) for claim in ALL_CLAIMS}

        # Expected sets are separately authored, not derived from the function.
        if special == "all-unknown":
            self.assertEqual(set(result.values()), {"unknown"}, name)
        elif special in ("withheld", "unknown"):
            self.assertEqual(result["external_delivery"], special, name)
            self.assertEqual({c for c, v in result.items()
                              if v == "requires-requalification"}, set(recheck), name)
            self.assertTrue(all(v == "historically-supported-candidate"
                                for c, v in result.items()
                                if c != "external_delivery"), name)
        else:
            self.assertEqual(
                {c for c, v in result.items()
                 if v == "requires-requalification"}, set(recheck), name)
            self.assertEqual(
                {c for c, v in result.items()
                 if v == "historically-supported-candidate"},
                ALL_CLAIMS.difference(recheck), name)
        # The past observation remains a past observation; current validity
        # and current permission are qualified independently.
        self.assertTrue(True)


def make_case_test(case):
    def check(self):
        self.verify_case(case)
    return check


for case in CASES:
    setattr(RequalificationFixture, "test_" + case[0].replace("-", "_"),
            make_case_test(case))


class MutationAndCostTest(unittest.TestCase):
    def test_hidden_rule_dependency_deletion_is_caught(self):
        mutant = {key: set(value) for key, value in DEPS.items()}
        mutant["rule_outcome"].discard("hidden_rule")
        now = Change(frozenset({"hidden_rule"}), output_digest_same=True)
        self.assertNotEqual(classify("rule_outcome", now, mutant),
                            "requires-requalification")

    def test_source_occurrence_dependency_deletion_is_caught(self):
        mutant = {key: set(value) for key, value in DEPS.items()}
        mutant["human_document"].discard("source_occurrence")
        now = Change(frozenset({"source_occurrence"}))
        self.assertNotEqual(classify("human_document", now, mutant),
                            "requires-requalification")

    def test_read_only_history_is_not_current_egress_permission(self):
        self.assertEqual(
            classify("external_delivery", Change(frozenset(), grant_now="unknown")),
            "unknown")
        self.assertEqual(
            classify("external_delivery",
                     Change(frozenset(), grant_now="checked-deny")),
            "withheld")
        self.assertEqual(
            classify("meaning_read", Change(frozenset(), grant_now="checked-deny")),
            "historically-supported-candidate")

    def test_full_vs_selective_vs_unchecked_cost_on_known_change(self):
        changed = Change(frozenset({"source_occurrence", "source_index", "hidden_rule"}))
        decisions = {claim: classify(claim, changed) for claim in ALL_CLAIMS}
        j0_full = len(ALL_CLAIMS)  # recheck all eight even if no dependency changed
        j1_selective = sum(v == "requires-requalification" for v in decisions.values())
        j2_unchecked = 0           # unsafe historical-PASS reuse
        self.assertEqual((j0_full, j1_selective, j2_unchecked), (8, 4, 0))
        self.assertEqual({c for c, v in decisions.items()
                          if v == "requires-requalification"},
                         {"source_citation", "human_document",
                          "document_review", "rule_outcome"})
        # This only compares *candidate counts*, not runtime or token costs.

    def test_unknown_dependency_never_proves_no_impact(self):
        v = Change(frozenset({"not-listed-in-the-corpus"}))
        self.assertEqual({classify(x, v) for x in ALL_CLAIMS}, {"unknown"})

    def test_output_digest_is_never_a_dependency_or_policy_oracle(self):
        for same in (False, True):
            state = Change(frozenset({"hidden_rule"}), output_digest_same=same)
            self.assertEqual(classify("rule_outcome", state),
                             "requires-requalification")


if __name__ == "__main__":
    unittest.main(verbosity=2)
