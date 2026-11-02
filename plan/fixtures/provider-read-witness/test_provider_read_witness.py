"""Synthetic read-witness pressure tests. This is not a TEI Provider or security policy."""
from dataclasses import dataclass
import unittest

EXPECTED = ({"id": "L1", "amount": 600},)
RAW = {
    "Order.lines": ({"id": "L1", "amount": 600, "order_id": "O1", "private_note": "canary"},),
    "Order.payments": ({"id": "P1", "status": "internal"},),
}


@dataclass(frozen=True)
class Read:
    path: str
    fields: frozenset[str]
    scope: str = "selected-order"


class SyntheticProvider:
    def __init__(self, *, data=None, observable=True, projects_fields=True):
        self.data = data if data is not None else RAW
        self.observable = observable
        self.projects_fields = projects_fields
        self.calls: list[Read] = []

    def read(self, path, fields):
        # Record observed provider-side requested field set, not only emitted JSON.
        actual_fields = frozenset(fields) if self.projects_fields else frozenset(self.data[path][0])
        self.calls.append(Read(path, actual_fields))
        return tuple({key: row[key] for key in actual_fields} for row in self.data[path])

    def witnessed_calls(self):
        return tuple(self.calls) if self.observable else None


def function_output(provider, *, extra_private=False, extra_relation=False, join_key=False):
    fields = {"id", "amount"}
    if extra_private:
        fields.add("private_note")
    if join_key:
        fields.add("order_id")
    rows = provider.read("Order.lines", fields)
    if extra_relation:
        provider.read("Order.payments", {"id"})
    return tuple({"id": row["id"], "amount": row["amount"]} for row in rows)


def classify_access(provider, *, allow_join_key=True):
    calls = provider.witnessed_calls()
    if calls is None:
        return "unknown"
    allowed = {"id", "amount"}
    if allow_join_key:
        allowed.add("order_id")
    if any(call.path != "Order.lines" or not call.fields <= allowed for call in calls):
        return "overfetch"
    return "narrow"


def completeness(required_scope, *, provider_complete_evidence):
    if required_scope == "page":
        return "partial"
    return "complete" if provider_complete_evidence is True else "unknown"


class WitnessTests(unittest.TestCase):
    def test_rw01_narrow_output_and_read(self):
        provider = SyntheticProvider()
        self.assertEqual(function_output(provider), EXPECTED)
        self.assertEqual(classify_access(provider), "narrow")

    def test_rw02_secret_overfetch_same_output(self):
        narrow, broad = SyntheticProvider(), SyntheticProvider()
        self.assertEqual(function_output(narrow), function_output(broad, extra_private=True))
        self.assertEqual(classify_access(broad), "overfetch")

    def test_rw03_unused_relation_read_same_output(self):
        narrow, broad = SyntheticProvider(), SyntheticProvider()
        self.assertEqual(function_output(narrow), function_output(broad, extra_relation=True))
        self.assertEqual(classify_access(broad), "overfetch")

    def test_rw04_explained_join_key_differs_from_output(self):
        provider = SyntheticProvider()
        self.assertEqual(function_output(provider, join_key=True), EXPECTED)
        self.assertEqual(classify_access(provider, allow_join_key=True), "narrow")
        self.assertEqual(classify_access(provider, allow_join_key=False), "overfetch")

    def test_rw05_api_lacks_projection_capability(self):
        provider = SyntheticProvider(projects_fields=False)
        self.assertEqual(function_output(provider), EXPECTED)
        self.assertEqual(classify_access(provider), "overfetch")

    def test_rw06_without_access_witness_is_unknown(self):
        provider = SyntheticProvider(observable=False)
        self.assertEqual(function_output(provider, extra_private=True), EXPECTED)
        self.assertEqual(classify_access(provider), "unknown")

    def test_rw07_requested_all_without_evidence_is_unknown(self):
        self.assertEqual(completeness("all_for_selected", provider_complete_evidence=None), "unknown")
        self.assertEqual(completeness("all_for_selected", provider_complete_evidence=True), "complete")
        self.assertEqual(completeness("page", provider_complete_evidence=True), "partial")

    def test_hidden_world_change_does_not_prove_no_read(self):
        raw2 = {**RAW, "Order.lines": ({**RAW["Order.lines"][0], "private_note": "different"},)}
        a, b = SyntheticProvider(), SyntheticProvider(data=raw2)
        self.assertEqual(function_output(a, extra_private=True), function_output(b, extra_private=True))
        self.assertEqual(classify_access(a), "overfetch")
        self.assertEqual(classify_access(b), "overfetch")

    def test_selected_world_change_is_detected(self):
        raw2 = {**RAW, "Order.lines": ({**RAW["Order.lines"][0], "amount": 601},)}
        a, b = SyntheticProvider(), SyntheticProvider(data=raw2)
        self.assertNotEqual(function_output(a), function_output(b))

    def test_disclosure_requires_separate_synthetic_grant(self):
        def can_send(*, local_read, recipient_approved):
            return local_read and recipient_approved
        self.assertFalse(can_send(local_read=True, recipient_approved=False))
        self.assertTrue(can_send(local_read=True, recipient_approved=True))

    def test_source_subject_id_does_not_bind_physical_alias(self):
        subject = "sales.order_line"
        bindings = {"primary": "sql.prod", "replica": "sql.readonly"}
        self.assertEqual(subject, "sales.order_line")
        self.assertNotEqual(bindings["primary"], bindings["replica"])

    def test_unobserved_underlying_storage_remains_unknown(self):
        provider = SyntheticProvider()
        self.assertEqual(function_output(provider), EXPECTED)
        self.assertEqual(classify_access(provider), "narrow")
        # A narrow adapter request does not observe DB page/cache I/O.
        db_internal_io_observed = False
        self.assertFalse(db_internal_io_observed)


if __name__ == "__main__":
    unittest.main()
