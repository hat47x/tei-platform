#!/usr/bin/env python3
"""Synthetic, read-only Action applicability: no TEI binding or actual permission checks."""
from dataclasses import dataclass, replace
import json
import unittest


@dataclass(frozen=True)
class DataContext:
    instance: str = "I1"
    grain: str = "entity"  # entity, join_occurrence, relation_occurrence, aggregate
    subject_id: str | None = "line-L1"
    relation_id: str | None = None
    parent_id: str | None = "order-A"
    base_version: int | None = 3
    action_scope_complete: bool = False
    visible_slice_complete: bool = True
    scope: str = "order-A"
    measure_parts: frozenset[str] = frozenset({"sum", "count"})
    accepted: frozenset[str] = frozenset({"patch_amount", "unlink", "delete_all", "rollup", "create_child"})
    declared_nullable: frozenset[str] = frozenset({"memo"})
    hidden_record_count: int = 0  # Must never appear in consumer output.


@dataclass(frozen=True)
class Operation:
    action: str
    field: str | None = None
    value_kind: str = "set"  # set, clear, omitted
    binding_available: bool = True
    ui_surface: str = "table"


@dataclass(frozen=True)
class Decision:
    status: str
    reasons: tuple[str, ...]
    candidate: tuple | None = None  # Not an invocation and not a grant.


def decide(ctx: DataContext, op: Operation) -> Decision:
    """Pure fixture matcher, *never* returns execution authorization."""
    if op.action not in ctx.accepted:
        return Decision("unresolved", ("action_meaning_not_accepted",))
    if not op.binding_available:
        return Decision("unsupported_target", ("no_confirmed_target_binding",))
    if op.action == "patch_amount":
        if ctx.grain not in ("entity", "join_occurrence") or not ctx.subject_id:
            return Decision("unsupported", ("not_a_single_identified_entity",))
        if ctx.base_version is None:
            return Decision("unresolved", ("base_version_missing",))
        if op.field not in ("amount", "memo"):
            return Decision("unsupported", ("field_not_supported_by_fixture",))
        if op.value_kind == "omitted":
            return Decision("unresolved", ("omission_is_not_clear",))
        if op.value_kind == "clear" and op.field not in ctx.declared_nullable:
            return Decision("unsupported", ("field_non_nullable",))
        return Decision("candidate_only", ("execution_grant_unverified",),
                        (op.action, ctx.instance, ctx.subject_id, ctx.scope,
                         ctx.base_version, op.field, op.value_kind))
    if op.action == "unlink":
        if ctx.grain != "relation_occurrence" or not ctx.relation_id:
            return Decision("unsupported", ("relation_occurrence_not_identified",))
        return Decision("candidate_only", ("execution_grant_unverified",),
                        (op.action, ctx.instance, ctx.relation_id, ctx.scope, ctx.base_version))
    if op.action == "delete_all":
        if not ctx.action_scope_complete:
            return Decision("unresolved", ("entire_action_scope_not_established",))
        if not ctx.parent_id or not ctx.scope:
            return Decision("unresolved", ("scope_or_parent_missing",))
        return Decision("candidate_only", ("execution_grant_unverified",),
                        (op.action, ctx.instance, ctx.parent_id, ctx.scope))
    if op.action == "rollup":
        if ctx.grain != "aggregate":
            return Decision("unsupported", ("aggregate_grain_required",))
        if not {"sum", "count"}.issubset(ctx.measure_parts):
            return Decision("unresolved", ("intermediate_state_missing",))
        return Decision("candidate_only", ("read_only_derivation",),
                        (op.action, ctx.instance, ctx.scope, "sum_count"))
    if op.action == "create_child":
        if not ctx.parent_id:
            return Decision("unresolved", ("parent_not_identified",))
        return Decision("candidate_only", ("execution_grant_unverified",),
                        (op.action, ctx.instance, ctx.parent_id, ctx.scope))
    return Decision("unsupported", ("action_not_implemented_in_fixture",))


def public_projection(ctx: DataContext, operations: tuple[Operation, ...], *, visible: bool,
                      released_actions: frozenset[str]) -> str:
    """Two-world test projection: never serialize hidden counts/reason/diagnostic detail.

    `visible` is a trusted caller-supplied *fixture assumption*, not a security check.
    """
    if not visible:
        return json.dumps({"actions": [], "status": "unavailable"}, sort_keys=True)
    labels = sorted(op.action for op in operations
                    if op.action in released_actions and decide(ctx, op).status == "candidate_only")
    return json.dumps({"actions": labels, "status": "candidate_only_not_authorized"}, sort_keys=True)


class SemanticActionTests(unittest.TestCase):
    def setUp(self):
        self.ctx = DataContext()
        self.patch = Operation("patch_amount", "amount")

    def test_sa01_entity_action_is_ui_independent(self):
        a = decide(self.ctx, self.patch)
        b = decide(self.ctx, replace(self.patch, ui_surface="graph"))
        self.assertEqual(a, b)
        self.assertEqual(a.status, "candidate_only")

    def test_sa02_join_repetitions_refer_to_one_entity(self):
        j = replace(self.ctx, grain="join_occurrence")
        self.assertEqual(decide(j, self.patch).candidate[2], "line-L1")
        self.assertEqual(decide(j, self.patch), decide(j, self.patch))

    def test_sa03_aggregate_point_is_not_single_row_edit(self):
        self.assertEqual(decide(replace(self.ctx, grain="aggregate", subject_id=None), self.patch).status, "unsupported")

    def test_sa04_partial_collection_still_allows_identified_patch_candidate(self):
        self.assertEqual(decide(replace(self.ctx, visible_slice_complete=False), self.patch).status, "candidate_only")

    def test_sa05_partial_collection_blocks_all_scope_delete(self):
        self.assertEqual(decide(self.ctx, Operation("delete_all")).status, "unresolved")

    def test_sa06_visible_scope_complete_does_not_establish_action_scope_complete(self):
        c = replace(self.ctx, visible_slice_complete=True, action_scope_complete=False, hidden_record_count=4)
        self.assertEqual(decide(c, Operation("delete_all")).status, "unresolved")
        self.assertEqual(decide(replace(c, action_scope_complete=True), Operation("delete_all")).status, "candidate_only")

    def test_sa07_unlink_is_about_relation_occurrence(self):
        c = replace(self.ctx, grain="relation_occurrence", subject_id="person-P2", relation_id="edge-A3")
        x = decide(c, Operation("unlink"))
        self.assertEqual(x.status, "candidate_only")
        self.assertEqual(x.candidate[2], "edge-A3")

    def test_sa08_selected_node_cannot_be_inferred_as_unlink_edge(self):
        self.assertEqual(decide(self.ctx, Operation("unlink")).status, "unsupported")

    def test_sa09_unaccepted_purpose_cannot_be_auto_selected(self):
        self.assertEqual(decide(self.ctx, Operation("submit")), Decision("unresolved", ("action_meaning_not_accepted",)))

    def test_sa10_omitted_field_is_not_implicit_clear(self):
        self.assertEqual(decide(self.ctx, Operation("patch_amount", "memo", "omitted")).status, "unresolved")

    def test_sa11_explicit_clear_requires_nullable_field(self):
        self.assertEqual(decide(self.ctx, Operation("patch_amount", "amount", "clear")).status, "unsupported")
        self.assertEqual(decide(self.ctx, Operation("patch_amount", "memo", "clear")).status, "candidate_only")

    def test_sa12_missing_base_version_not_writable(self):
        self.assertEqual(decide(replace(self.ctx, base_version=None), self.patch).status, "unresolved")

    def test_sa13_unbound_provider_is_not_silent_fallback(self):
        self.assertEqual(decide(self.ctx, replace(self.patch, binding_available=False)).status, "unsupported_target")

    def test_sa14_candidate_is_not_host_authorization(self):
        x = decide(self.ctx, self.patch)
        self.assertEqual(x.reasons, ("execution_grant_unverified",))

    def test_sa15_instance_and_key_stable_not_row_index(self):
        a = decide(self.ctx, self.patch).candidate
        b = decide(replace(self.ctx, instance="I2", subject_id="line-L2"), self.patch).candidate
        self.assertNotEqual(a, b)

    def test_sa16_rollup_requires_intermediate_state(self):
        c = replace(self.ctx, grain="aggregate", measure_parts=frozenset({"avg"}))
        self.assertEqual(decide(c, Operation("rollup")).status, "unresolved")
        self.assertEqual(decide(replace(c, measure_parts=frozenset({"sum", "count"})), Operation("rollup")).status, "candidate_only")

    def test_sa17_hidden_world_variation_cannot_change_public_projection(self):
        ops = (self.patch, Operation("delete_all"), Operation("create_child"))
        a = public_projection(replace(self.ctx, hidden_record_count=0), ops, visible=True, released_actions=frozenset({"patch_amount", "create_child"}))
        b = public_projection(replace(self.ctx, hidden_record_count=999), ops, visible=True, released_actions=frozenset({"patch_amount", "create_child"}))
        self.assertEqual(a, b)
        self.assertNotIn("999", a)

    def test_sa18_denied_world_produces_no_distinguishing_diagnostics(self):
        ops = (self.patch, Operation("delete_all"))
        a = public_projection(replace(self.ctx, hidden_record_count=0), ops, visible=False, released_actions=frozenset({"patch_amount", "create_child"}))
        b = public_projection(replace(self.ctx, hidden_record_count=6), ops, visible=False, released_actions=frozenset({"patch_amount", "create_child"}))
        self.assertEqual(a, b)
        self.assertNotIn("reason", a)

    def test_sa21_secret_action_identity_must_not_leak_through_candidate_list(self):
        ops = (self.patch, Operation("create_child"), Operation("unlink"))
        released = frozenset({"patch_amount", "create_child"})
        a = public_projection(replace(self.ctx, accepted=frozenset({"patch_amount", "create_child"})),
                              ops, visible=True, released_actions=released)
        b = public_projection(replace(self.ctx, accepted=frozenset({"patch_amount", "create_child", "unlink"})),
                              ops, visible=True, released_actions=released)
        self.assertEqual(a, b)
        self.assertNotIn("unlink", a)

    def test_sa22_visible_context_needs_per_action_release(self):
        ops = (self.patch, Operation("create_child"))
        a = public_projection(self.ctx, ops, visible=True, released_actions=frozenset())
        self.assertEqual(json.loads(a)["actions"], [])
        self.assertNotIn("create_child", a)

    def test_sa19_ui_widget_choice_has_no_effect_on_candidate_semantics(self):
        widgets = ("form", "table", "chart", "cui", "native")
        results = {decide(self.ctx, replace(self.patch, ui_surface=w)).candidate for w in widgets}
        self.assertEqual(len(results), 1)

    def test_sa20_parent_binding_is_required_for_create_child(self):
        self.assertEqual(decide(replace(self.ctx, parent_id=None), Operation("create_child")).status, "unresolved")
        self.assertEqual(decide(self.ctx, Operation("create_child")).status, "candidate_only")


if __name__ == "__main__":
    unittest.main(verbosity=2)
