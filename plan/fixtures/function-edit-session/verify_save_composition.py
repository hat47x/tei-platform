#!/usr/bin/env python3
"""Synthetic save-plan conformance only. No TEI Host, provider, grant, or commit."""
from dataclasses import dataclass
import unittest


@dataclass(frozen=True)
class Intent:
    instance: str
    request: str
    kind: str
    target: str
    provider: str
    depends_on: tuple = ()  # explicitly scoped (instance, request) identifiers
    base_version: int = 1


def order_candidates(intents):
    """Order dependencies; ordering does not assert atomicity or execution."""
    by_key = {(x.instance, x.request): x for x in intents}
    if len(by_key) != len(intents):
        raise ValueError("duplicate request identity")
    for x in intents:
        for d in x.depends_on:
            if d not in by_key:
                raise ValueError("unknown dependency")
    visiting, complete, result = set(), set(), []

    def visit(key):
        if key in visiting:
            raise ValueError("dependency cycle")
        if key in complete:
            return
        visiting.add(key)
        for d in by_key[key].depends_on:
            visit(d)
        visiting.remove(key)
        complete.add(key)
        result.append(by_key[key])

    for key in by_key:
        visit(key)
    return result


def atomicity_claim(intents, declared_group, provider_evidence):
    """A fixture declaration, not evidence of a real transaction."""
    if not intents or not declared_group or not provider_evidence:
        return "not_established"
    if len({x.provider for x in intents}) != 1:
        return "not_established"
    return "fixture_declared_only"


def resolve_parent_temp(instance, provisional, parent_status, confirmed_ids):
    """Local temp IDs are not physical keys or accepted business identities."""
    if parent_status != "confirmed_in_fixture":
        return "parent_unresolved", None
    value = confirmed_ids.get((instance, provisional))
    if value is None:
        return "parent_unresolved", None
    return "resolved_from_fixture_receipt", value


def summarize_results(intents, responses):
    """Response omission means unknown; a positive ACK is not business completion."""
    keys = {(x.instance, x.request) for x in intents}
    if not set(responses).issubset(keys):
        raise ValueError("unrecognized response identity")
    statuses = {k: responses.get(k, "unknown") for k in keys}
    if "unknown" in statuses.values():
        verdict = "unknown_effect"
    elif all(v == "confirmed_in_fixture" for v in statuses.values()):
        verdict = "locally_confirmed_not_business_completion"
    else:
        verdict = "partial_or_failed"
    return verdict, statuses


def typed_change(consumer, instance, context, target, kind, field, value, version):
    """UI surface is intentionally excluded from a normalized candidate."""
    if consumer not in ("cui", "web"):
        raise ValueError("unrecognized synthetic consumer")
    return (instance, context, target, kind, field, value, version)


class SaveCompositionTests(unittest.TestCase):
    def test_es07_temp_parent_mapping_is_instance_scoped(self):
        ids = {("I1", "tmp-H"): "DB-101", ("I2", "tmp-H"): "DB-909"}
        self.assertEqual(resolve_parent_temp("I1", "tmp-H", "confirmed_in_fixture", ids),
                         ("resolved_from_fixture_receipt", "DB-101"))
        self.assertEqual(resolve_parent_temp("I2", "tmp-H", "confirmed_in_fixture", ids),
                         ("resolved_from_fixture_receipt", "DB-909"))
        self.assertEqual(resolve_parent_temp("I1", "tmp-H", "unknown", ids),
                         ("parent_unresolved", None))
        self.assertEqual(resolve_parent_temp("I3", "tmp-H", "confirmed_in_fixture", ids),
                         ("parent_unresolved", None))

    def test_es08_unlink_targets_edge_not_node(self):
        edge = Intent("I1", "r1", "unlink_relation", "membership-edge-A3", "P1")
        self.assertNotEqual(edge.kind, "delete_entity")
        self.assertNotEqual(edge.target, "person-P2")

    def test_es12_consumer_neutral_typed_change(self):
        a = typed_change("cui", "I1", "D1", "line-L1", "patch", "amount", 120, 1)
        b = typed_change("web", "I1", "D1", "line-L1", "patch", "amount", 120, 1)
        self.assertEqual(a, b)
        self.assertNotEqual(a, typed_change("web", "I2", "D1", "line-L1", "patch", "amount", 120, 1))

    def test_es14_parent_child_dependency_before_child(self):
        child = Intent("I1", "create-line", "create_entity", "line-local", "P1", (("I1", "create-header"),))
        parent = Intent("I1", "create-header", "create_entity", "header-local", "P1")
        self.assertEqual([x.request for x in order_candidates([child, parent])],
                         ["create-header", "create-line"])
        self.assertEqual(atomicity_claim([parent, child], True, False), "not_established")

    def test_es15_cycle_and_missing_dependency_rejected(self):
        a = Intent("I1", "a", "update", "A", "P1", (("I1", "b"),))
        b = Intent("I1", "b", "update", "B", "P1", (("I1", "a"),))
        with self.assertRaisesRegex(ValueError, "cycle"):
            order_candidates([a, b])
        with self.assertRaisesRegex(ValueError, "unknown dependency"):
            order_candidates([a])

    def test_es16_serial_batch_does_not_imply_transaction(self):
        a = Intent("I1", "a", "update", "A", "P1")
        b = Intent("I1", "b", "update", "B", "P2", (("I1", "a"),))
        self.assertEqual([x.request for x in order_candidates([b, a])], ["a", "b"])
        self.assertEqual(atomicity_claim([a, b], True, True), "not_established")
        self.assertEqual(atomicity_claim([a, b], False, True), "not_established")
        b1 = Intent("I1", "b", "update", "B", "P1", (("I1", "a"),))
        self.assertEqual(atomicity_claim([a, b1], True, True), "fixture_declared_only")

    def test_es17_missing_response_is_unknown_not_failure_or_success(self):
        a = Intent("I1", "a", "update", "A", "P1")
        b = Intent("I1", "b", "update", "B", "P2")
        verdict, details = summarize_results([a, b], {("I1", "a"): "confirmed_in_fixture"})
        self.assertEqual(verdict, "unknown_effect")
        self.assertEqual(details[("I1", "b")], "unknown")
        with self.assertRaisesRegex(ValueError, "unrecognized response identity"):
            summarize_results([a], {("I2", "a"): "confirmed_in_fixture"})

    def test_es18_all_local_confirmations_not_business_completion(self):
        a = Intent("I1", "a", "update", "A", "P1")
        self.assertEqual(summarize_results([a], {("I1", "a"): "confirmed_in_fixture"})[0],
                         "locally_confirmed_not_business_completion")


if __name__ == "__main__":
    unittest.main(verbosity=2)
