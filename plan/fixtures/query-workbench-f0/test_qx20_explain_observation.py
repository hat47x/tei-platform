"""QX-20 fixture-only oracle for structural Explain disclosure boundaries.

This module does NOT implement a TEI disclosure policy or invoke TEI/SQL.
The response examples are synthetic. It tests whether a DIFFERENTIAL ORACLE
would reject observable differences from secrets and allow explicitly declared
release values. Scope: deterministic JSON payloads, not timing/network effects.
"""

import copy
import json
import unittest


def encode(payload):
    """Treat JSON keys and list length/order as observable."""
    return json.dumps(payload, sort_keys=True, ensure_ascii=False, separators=(",", ":"))


def erase_explicit_release(payload, released_keys):
    """Only declared top-level release fields may differ across worlds."""
    cleaned = copy.deepcopy(payload)
    for key in released_keys:
        if key not in cleaned:
            raise AssertionError(f"declared release missing: {key}")
        cleaned[key] = "<approved-release-value>"
    return cleaned


def same_public_observation(left, right, released_keys=()):
    """A necessary differential noninterference oracle for one world pair."""
    return encode(erase_explicit_release(left, released_keys)) == encode(
        erase_explicit_release(right, released_keys)
    )


def full_ir(private_name, private_literal, hidden_estimate):
    """Synthetic payload shaped like selected TEI DerivedIR/plan fields."""
    return {
        "query": "app.public_report",
        "sources": [
            {"id": "source:public", "name": "public.orders"},
            {"id": "source:secret", "name": private_name},
        ],
        "outputFields": [
            {"id": "output:0", "name": "amount", "expression": "SUM(o.amount)"}
        ],
        "relational": {
            "nodes": [
                {"id": "source:public", "kind": "source", "source": "public.orders"},
                {"id": "filter:hidden", "kind": "filter", "expressions": [
                    f"private_key = '{private_literal}'"
                ]},
            ]
        },
        "diagnostics": [f"Unable to open {private_name}"],
        "physicalExplain": {"Plan Rows": hidden_estimate},
    }


def fixture_project_with_explicit_source_grant(raw, granted_source_ids):
    """Model a caller-supplied source grant; NOT a TEI authorization decision."""
    return {
        "query": raw["query"],
        "explainStatus": "partial",
        "lineageStatus": "not-disclosed",
        "publishedSources": [
            s["name"] for s in raw["sources"] if s["id"] in granted_source_ids
        ],
    }


class ExplainObservationOracleTest(unittest.TestCase):
    def setUp(self):
        # Same public world, different private relation name, value, and size.
        self.world_a = full_ir("secret.payments_a", "alpha", 2)
        self.world_b = full_ir("secret.payments_b", "beta", 400)
        self.public_projection = {
            "query": "app.public_report",
            "explainStatus": "partial",
            "lineageStatus": "not-disclosed",
            "publishedSources": ["public.orders"],
        }

    def test_01_safe_projection_is_identical_across_private_worlds(self):
        a = fixture_project_with_explicit_source_grant(self.world_a, {"source:public"})
        b = fixture_project_with_explicit_source_grant(self.world_b, {"source:public"})
        self.assertEqual(a, self.public_projection)
        self.assertTrue(same_public_observation(a, b))

    def test_02_raw_derived_ir_exposes_hidden_source_and_expression(self):
        self.assertFalse(same_public_observation(self.world_a, self.world_b))

    def test_03_source_list_length_is_observable(self):
        a = copy.deepcopy(self.public_projection)
        b = copy.deepcopy(a)
        b["publishedSources"].append("<redacted>")
        self.assertFalse(same_public_observation(a, b))

    def test_04_diagnostic_text_is_observable(self):
        a = {"status": "not-disclosed", "message": "secret.payments_a denied"}
        b = {"status": "not-disclosed", "message": "secret.payments_b denied"}
        self.assertFalse(same_public_observation(a, b))

    def test_05_plan_estimate_is_observable(self):
        a = {"status": "partial", "planRows": 2}
        b = {"status": "partial", "planRows": 400}
        self.assertFalse(same_public_observation(a, b))

    def test_06_authorized_release_is_explicitly_excepted(self):
        a = {**self.public_projection, "authorizedAggregate": 20}
        b = {**self.public_projection, "authorizedAggregate": 40}
        self.assertFalse(same_public_observation(a, b))
        self.assertTrue(same_public_observation(a, b, ("authorizedAggregate",)))

    def test_07_whitelisting_release_cannot_hide_other_leaks(self):
        a = {**self.public_projection, "authorizedAggregate": 20, "debug": "alpha"}
        b = {**self.public_projection, "authorizedAggregate": 40, "debug": "beta"}
        self.assertFalse(same_public_observation(a, b, ("authorizedAggregate",)))

    def test_08_declared_release_must_exist(self):
        with self.assertRaisesRegex(AssertionError, "declared release missing"):
            same_public_observation(self.public_projection, self.public_projection, ("missing",))

    def test_09_status_change_can_leak(self):
        a = {**self.public_projection, "explainStatus": "partial"}
        b = {**self.public_projection, "explainStatus": "denied"}
        self.assertFalse(same_public_observation(a, b))

    def test_10_projection_order_is_observable(self):
        a = {**self.public_projection, "publishedSources": ["public.orders", "public.lines"]}
        b = {**self.public_projection, "publishedSources": ["public.lines", "public.orders"]}
        self.assertFalse(same_public_observation(a, b))


if __name__ == "__main__":
    unittest.main()
