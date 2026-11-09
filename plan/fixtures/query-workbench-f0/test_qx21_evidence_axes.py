"""QX-21 reference: compare evidence axes before reusing SQL explanations.

Synthetic read-only comparison. This is NOT TEI Query/Context code, a semantic
proof, a cache, a policy engine, a query evaluator, or a grant decision. Input
labels are fixture-only, not proposed Core API/enum names.
"""
from copy import deepcopy
import json
import unittest


# A change triggers checks of *different* qualifications. A changed display
# epoch must not retroactively erase a historic claim or move an unsaved patch.
# Conversely a display-only revision must not invalidate a stored fact.
RECHECK = {
    "query_id": ("claim", "disclosure", "presentation", "editing"),
    "sql_digest": ("claim", "disclosure", "presentation", "editing"),
    "dialect": ("claim", "disclosure", "presentation"),
    "schema_revision": ("claim", "disclosure", "presentation"),
    "analysis_invocation": ("claim", "disclosure", "presentation"),
    "target_binding": ("claim", "disclosure", "presentation", "editing"),
    "search_path_context": ("claim", "disclosure", "presentation"),
    "view_function_revision": ("claim", "disclosure", "presentation"),
    "effective_policy_revision": ("claim", "disclosure", "presentation"),
    "visible_population_scope": ("claim", "disclosure", "presentation"),
    "data_snapshot": ("claim", "disclosure", "presentation", "editing"),
    "context_instance": ("disclosure", "presentation", "editing"),
    "selection_epoch": ("disclosure", "presentation", "editing"),
    "measure_contract_revision": ("claim", "disclosure", "presentation"),
    "measure_rounding_stage": ("claim", "disclosure", "presentation"),
    "semantic_model_revision": ("claim", "disclosure", "presentation", "editing"),
    "owner_acceptance_revision": ("claim", "disclosure", "presentation", "editing"),
    "disclosure_grant_revision": ("disclosure", "presentation"),
    "render_revision": ("presentation",),
}


def evidence_diff(previous, current):
    """Report differences without asserting correctness/currentness.

    A missing/None axis is unobserved, not 'same'. Equal labels are not an
    independently proven end-to-end binding; no Source/Provider facts are read.
    """
    unknown = sorted(k for k in RECHECK if k not in previous or k not in current
                     or previous[k] is None or current[k] is None)
    changed = sorted(k for k in RECHECK if k not in unknown
                     and previous[k] != current[k])
    # Unknown axes need rechecking too; a consumer inspecting only the verdict
    # must not mistake missing metadata for a safe, unchanged dependency.
    unknown_checks = sorted({kind for k in unknown for kind in RECHECK[k]})
    checks = sorted({kind for k in changed + unknown for kind in RECHECK[k]})
    return {
        "comparison": "evidence-changed" if changed else
                      "missing-evidence" if unknown else
                      "declared-axes-equal-only",
        "changedAxes": changed,
        "unknownAxes": unknown,
        "recheckQualifications": checks,
        "unknownQualifications": unknown_checks,
        "provesCurrent": False,
        "provesDisclosureAllowed": False,
        "provesEffectCommitted": False,
    }


def specimen():
    return {
        "query_id": "Q:1", "sql_digest": "sql:v1", "dialect": "postgresql",
        "schema_revision": "schema:v1", "analysis_invocation": "worker:a1",
        "target_binding": "table:tenant_a.orders", "search_path_context": "path:a",
        "view_function_revision": "view:v1", "effective_policy_revision": "policy:v1",
        "visible_population_scope": "principal:alice/tenant:a",
        "data_snapshot": "dbsnap:t1", "context_instance": "I1",
        "selection_epoch": "selection:A:1", "measure_contract_revision": "sum:v1",
        "measure_rounding_stage": "round-after-sum", "semantic_model_revision": "model:v1",
        "owner_acceptance_revision": "owner:accepted:v1",
        "disclosure_grant_revision": "grant:report:v1", "render_revision": "renderer:v1",
    }


class QX21ReadOnlyEvidenceAxisTests(unittest.TestCase):
    def setUp(self):
        self.old = specimen()

    def compare_change(self, key, value):
        new = deepcopy(self.old)
        new[key] = value
        return evidence_diff(self.old, new)

    def test_a_same_query_id_different_sql_digest(self):
        diff = self.compare_change("sql_digest", "sql:v2")
        self.assertEqual(diff["changedAxes"], ["sql_digest"])
        self.assertIn("claim", diff["recheckQualifications"])

    def test_b_same_sql_different_effective_target_binding(self):
        diff = self.compare_change("target_binding", "table:tenant_b.orders")
        self.assertIn("claim", diff["recheckQualifications"])
        self.assertIn("disclosure", diff["recheckQualifications"])

    def test_c_same_target_different_row_policy(self):
        diff = self.compare_change("effective_policy_revision", "policy:v2")
        self.assertIn("claim", diff["recheckQualifications"])
        self.assertIn("disclosure", diff["recheckQualifications"])

    def test_d_same_displayed_value_different_snapshot(self):
        # No numeric value is stored in the evidence vector, deliberately.
        diff = self.compare_change("data_snapshot", "dbsnap:t2")
        self.assertEqual(diff["comparison"], "evidence-changed")
        self.assertIn("claim", diff["recheckQualifications"])

    def test_e_rounding_stage_affects_measure_without_sql_revision(self):
        diff = self.compare_change("measure_rounding_stage", "round-each-row")
        self.assertIn("claim", diff["recheckQualifications"])
        self.assertNotIn("editing", diff["recheckQualifications"])

    def test_f_parent_selection_epoch_only_current_display_and_edit(self):
        diff = self.compare_change("selection_epoch", "selection:B:2")
        self.assertEqual(diff["recheckQualifications"], ["disclosure", "editing", "presentation"])
        # The old patch's own target remains A even when a view selects B.
        patch = {"anchor": "A", "instance": "I1", "base": "dbsnap:t1"}
        current_view = {"anchor": "B", "instance": "I1", "epoch": 2}
        self.assertNotEqual(patch["anchor"], current_view["anchor"])
        self.assertEqual(patch["anchor"], "A")

    def test_g_disclosure_change_not_fact_change(self):
        diff = self.compare_change("disclosure_grant_revision", "grant:report:revoked")
        self.assertEqual(diff["recheckQualifications"], ["disclosure", "presentation"])
        self.assertFalse(diff["provesDisclosureAllowed"])

    def test_h_render_change_only_presentation(self):
        diff = self.compare_change("render_revision", "renderer:v2")
        self.assertEqual(diff["recheckQualifications"], ["presentation"])

    def test_i_missing_axis_does_not_equal_verified_current(self):
        new = deepcopy(self.old)
        del new["view_function_revision"]
        diff = evidence_diff(self.old, new)
        self.assertEqual(diff["comparison"], "missing-evidence")
        self.assertEqual(diff["unknownAxes"], ["view_function_revision"])
        self.assertEqual(diff["unknownQualifications"], ["claim", "disclosure", "presentation"])
        self.assertEqual(diff["recheckQualifications"], diff["unknownQualifications"])
        self.assertFalse(diff["provesCurrent"])
        new["view_function_revision"] = None
        self.assertEqual(evidence_diff(self.old, new)["unknownAxes"],
                         ["view_function_revision"])

    def test_j_all_labels_equal_is_not_reuse_authorization(self):
        diff = evidence_diff(self.old, deepcopy(self.old))
        self.assertEqual(diff["comparison"], "declared-axes-equal-only")
        self.assertFalse(diff["provesCurrent"])
        self.assertFalse(diff["provesDisclosureAllowed"])
        self.assertFalse(diff["provesEffectCommitted"])

    def test_k_owner_review_with_same_bytes_invalidates_claim_qualification(self):
        diff = self.compare_change("owner_acceptance_revision", "owner:review-withdrawn:v2")
        self.assertIn("claim", diff["recheckQualifications"])

    def test_l_combined_change_does_not_collapse_distinct_reasons(self):
        new = deepcopy(self.old)
        new["selection_epoch"] = "selection:B:2"
        new["effective_policy_revision"] = "policy:v2"
        new["disclosure_grant_revision"] = "grant:report:revoked"
        diff = evidence_diff(self.old, new)
        self.assertEqual(len(diff["changedAxes"]), 3)
        self.assertEqual(diff["recheckQualifications"],
                         ["claim", "disclosure", "editing", "presentation"])

    def test_n_missing_and_changed_rechecks_are_both_retained(self):
        new = deepcopy(self.old)
        new["render_revision"] = "renderer:v2"
        del new["target_binding"]
        diff = evidence_diff(self.old, new)
        self.assertEqual(diff["changedAxes"], ["render_revision"])
        self.assertEqual(diff["unknownAxes"], ["target_binding"])
        self.assertEqual(diff["recheckQualifications"],
                         ["claim", "disclosure", "editing", "presentation"])
        self.assertEqual(diff["unknownQualifications"],
                         ["claim", "disclosure", "editing", "presentation"])
        self.assertFalse(diff["provesCurrent"])

    def test_o_search_path_and_plan_invocation_recheck_without_source_edit(self):
        new = deepcopy(self.old)
        new["search_path_context"] = "path:b"
        new["analysis_invocation"] = "worker:a2"
        diff = evidence_diff(self.old, new)
        self.assertEqual(diff["changedAxes"],
                         ["analysis_invocation", "search_path_context"])
        self.assertIn("claim", diff["recheckQualifications"])
        self.assertNotIn("editing", diff["recheckQualifications"])

    def test_m_json_output_contains_only_axis_names_not_secret_values(self):
        new = deepcopy(self.old)
        new["visible_population_scope"] = "highly-sensitive-principal-id"
        response = json.dumps(evidence_diff(self.old, new), sort_keys=True)
        self.assertNotIn("highly-sensitive-principal-id", response)
        self.assertNotIn("principal:alice", response)
        self.assertIn("visible_population_scope", response)


if __name__ == "__main__":
    unittest.main()
