#!/usr/bin/env python3
"""Synthetic dependency-footprint fixture, not a TEI rule engine or authorization check."""
from dataclasses import dataclass, replace
import json
import unittest


@dataclass(frozen=True)
class Rule:
    ident: str = "rule.order.credit"
    revision: int = 3
    accepted: bool = True  # Fixture assumption only, not a TEI Owner decision.
    kind: str = "validation"
    requirements: tuple[str, ...] = ("Order.amount", "Account.credit_limit")


@dataclass(frozen=True)
class Evidence:
    path: str
    value: int | None
    scope: str = "order-A"
    revision: int = 3
    epoch: str = "snapshot-1"
    complete: bool = True
    permitted_for_rule: bool = True  # Trusted fixture fact; not current Host Grant.


@dataclass(frozen=True)
class Result:
    state: str
    reasons: tuple[str, ...]
    consumer_value: str | None = None


def footprints(display_fields: tuple[str, ...], rule: Rule) -> dict:
    """Different needs, not a command to fetch every requested input into the client."""
    if not rule.accepted:
        return {"display": display_fields, "evaluation": (), "status": "rule_unaccepted"}
    if rule.kind not in {"pure", "validation"}:
        return {"display": display_fields, "evaluation": (), "status": "effect_requires_owner"}
    return {"display": display_fields, "evaluation": rule.requirements,
            "status": "declaration_only_not_provider_read"}


def evaluate(rule: Rule, data: tuple[Evidence, ...]) -> Result:
    """Classify finite fixture inputs, never authorize/commit/accept an Action."""
    if not rule.accepted:
        return Result("unresolved", ("unaccepted_rule",))
    if rule.kind != "validation":
        return Result("unsupported", ("not_validation_or_effect",))
    if len({x.path for x in data}) != len(data):
        return Result("unresolved", ("ambiguous_duplicate_input",))
    supplied = {x.path: x for x in data}
    required = [supplied.get(path) for path in rule.requirements]
    if any(x is None for x in required):
        return Result("unresolved", ("required_rule_input_missing",))
    if not all(x.permitted_for_rule for x in required):
        return Result("unresolved", ("rule_input_not_qualified",))
    if any(not x.complete for x in required):
        return Result("unresolved", ("required_rule_input_incomplete",))
    if any(x.revision != rule.revision for x in required):
        return Result("unresolved", ("rule_or_input_revision_mismatch",))
    if len({x.scope for x in required}) != 1:
        return Result("unresolved", ("semantic_scope_mismatch",))
    if len({x.epoch for x in required}) != 1:
        return Result("unresolved", ("snapshot_mismatch",))
    if any(x.value is None for x in required):
        return Result("unresolved", ("null_policy_not_declared",))
    if rule.requirements != ("Order.amount", "Account.credit_limit"):
        return Result("unsupported", ("unknown_rule_semantics",))
    amount, limit = (x.value for x in required)
    if amount <= limit:
        return Result("fixture_satisfied", ("preview_not_commit",), "allowed_by_fixture_rule")
    return Result("fixture_not_satisfied", ("preview_not_commit",), "denied_by_fixture_rule")


def public_result(result: Result, allowed_result: bool) -> str:
    """Verdict may itself leak hidden inputs; release is independently qualified."""
    if not allowed_result:
        return json.dumps({"status": "unavailable"}, sort_keys=True)
    if result.state == "fixture_satisfied":
        return json.dumps({"status": "preview", "validation": "satisfied"}, sort_keys=True)
    if result.state == "fixture_not_satisfied":
        return json.dumps({"status": "preview", "validation": "not_satisfied"}, sort_keys=True)
    return json.dumps({"status": "unavailable"}, sort_keys=True)


A = Evidence("Order.amount", 600)
L = Evidence("Account.credit_limit", 700)


class RuleFootprintTests(unittest.TestCase):
    def test_rf01_display_map_does_not_infer_rule_input(self):
        plan = footprints(("Order.id", "Order.amount"), Rule())
        self.assertNotIn("Account.credit_limit", plan["display"])
        self.assertIn("Account.credit_limit", plan["evaluation"])

    def test_rf02_hidden_rule_input_is_not_sent_in_display_projection(self):
        plan = footprints(("Order.amount",), Rule())
        self.assertNotIn("Account.credit_limit", json.dumps(plan["display"]))
        self.assertEqual(plan["status"], "declaration_only_not_provider_read")

    def test_rf03_declared_rule_dependency_can_validate_without_display(self):
        self.assertEqual(evaluate(Rule(), (A, L)).state, "fixture_satisfied")

    def test_rf04_same_display_different_credit_limit_changes_validation(self):
        a = evaluate(Rule(), (A, L))
        b = evaluate(Rule(), (A, replace(L, value=500)))
        self.assertNotEqual(a.state, b.state)
        self.assertEqual(b.state, "fixture_not_satisfied")

    def test_rf05_missing_derived_dependency_is_unknown_not_denial(self):
        self.assertEqual(evaluate(Rule(), (A,)).state, "unresolved")

    def test_rf06_denied_rule_input_is_unavailable_not_null(self):
        r = evaluate(Rule(), (A, replace(L, permitted_for_rule=False)))
        self.assertIn("rule_input_not_qualified", r.reasons)

    def test_rf07_partial_rule_input_cannot_validate_entire_scope(self):
        self.assertEqual(evaluate(Rule(), (A, replace(L, complete=False))).state, "unresolved")

    def test_rf08_unaccepted_rule_cannot_be_inferred(self):
        self.assertEqual(footprints(("Order.amount",), replace(Rule(), accepted=False))["status"], "rule_unaccepted")
        self.assertEqual(evaluate(replace(Rule(), accepted=False), (A, L)).state, "unresolved")

    def test_rf09_effect_rule_is_not_pure_validation(self):
        self.assertEqual(footprints(("Order.id",), replace(Rule(), kind="after_commit"))["status"], "effect_requires_owner")
        self.assertEqual(evaluate(replace(Rule(), kind="after_commit"), (A, L)).state, "unsupported")

    def test_rf10_revision_mismatch_blocks_stale_preview(self):
        self.assertIn("rule_or_input_revision_mismatch", evaluate(Rule(), (A, replace(L, revision=4))).reasons)

    def test_rf11_epoch_mismatch_needs_fresh_evidence(self):
        self.assertIn("snapshot_mismatch", evaluate(Rule(), (A, replace(L, epoch="snapshot-2"))).reasons)

    def test_rf12_scope_mismatch_cannot_compare_different_order(self):
        self.assertIn("semantic_scope_mismatch", evaluate(Rule(), (A, replace(L, scope="order-B"))).reasons)

    def test_rf13_null_policy_is_not_zero(self):
        self.assertIn("null_policy_not_declared", evaluate(Rule(), (A, replace(L, value=None))).reasons)

    def test_rf14_same_hidden_input_withheld_verdict_for_unauthorized_consumer(self):
        x = public_result(evaluate(Rule(), (A, L)), allowed_result=False)
        y = public_result(evaluate(Rule(), (A, replace(L, value=500))), allowed_result=False)
        self.assertEqual(x, y)
        self.assertNotIn("limit", x)

    def test_rf15_explicit_release_can_expose_verdict_without_secret_value(self):
        x = public_result(evaluate(Rule(), (A, replace(L, value=500))), allowed_result=True)
        self.assertIn("not_satisfied", x)
        self.assertNotIn("500", x)

    def test_rf16_duplicate_rule_input_is_ambiguous(self):
        self.assertIn("ambiguous_duplicate_input", evaluate(Rule(), (A, L, replace(L, value=500))).reasons)

    def test_rf17_unknown_rule_semantics_not_silently_evaluated(self):
        r = replace(Rule(), requirements=("Order.amount", "Account.credit_limit", "Rate.multiplier"))
        evidence = (A, L, Evidence("Rate.multiplier", 2))
        self.assertEqual(evaluate(r, evidence).state, "unsupported")

    def test_rf18_local_success_never_equates_execution_permission(self):
        x = evaluate(Rule(), (A, L))
        self.assertIn("preview_not_commit", x.reasons)
        self.assertNotEqual(x.state, "authorized")


if __name__ == "__main__":
    unittest.main(verbosity=2)
