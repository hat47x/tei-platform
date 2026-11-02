#!/usr/bin/env python3
"""Isolated reference: explicit per-Function access map; no actual TEI/Provider access."""
from dataclasses import dataclass
import copy
import json
import unittest

MODEL = {
    "Order": ({"id", "customer", "private_note"}, {"lines": "Line", "payments": "Payment"}),
    "Line": ({"id", "amount", "sku"}, {"notes": "Note"}),
    "Note": ({"id", "text", "internal_flag"}, {}),
    "Payment": ({"id", "value"}, {}),
}
DATA = {
    "id": "A", "customer": "C1", "private_note": "not selected",
    "lines": [
        {"id": "L1", "amount": 100, "sku": "S1", "notes": [
            {"id": "N1", "text": "first", "internal_flag": 1},
            {"id": "N2", "text": "second", "internal_flag": 2},
        ]},
        {"id": "L2", "amount": 200, "sku": "S2", "notes": []},
    ],
    "payments": [{"id": "P1", "value": 300}],
}


@dataclass(frozen=True)
class Mapping:
    ident: str
    path: str
    fields: tuple[str, ...]
    scope: str


def to_mapping(rows):
    return tuple(Mapping(r["id"], r["path"], tuple(r["fields"]), r["scope"]) for r in rows)


def markdown_map(text):
    """Tiny, strict fixture grammar. Not a generic Markdown parser or product DSL."""
    lines = [x.strip() for x in text.splitlines() if x.strip()]
    expected = ("Context", "Path", "Fields", "Scope")
    parsed = []
    for index, line in enumerate(lines):
        if not (line.startswith("|") and line.endswith("|")):
            raise ValueError("malformed row")
        cells = tuple(x.strip() for x in line[1:-1].split("|"))
        if len(cells) != 4:
            raise ValueError("ambiguous column count")
        if index == 0:
            if cells != expected:
                raise ValueError("unexpected headings")
            continue
        if all(set(x) <= set(":- ") for x in cells):
            continue
        parsed.append({"id": cells[0], "path": cells[1],
                       "fields": [x.strip() for x in cells[2].split(",")],
                       "scope": cells[3]})
    return to_mapping(parsed)


def validate(mapping):
    if not mapping:
        raise ValueError("empty explicit map")
    ids = set()
    paths = set()
    for row in mapping:
        if row.ident in ids or row.path in paths:
            raise ValueError("duplicate context or path")
        ids.add(row.ident)
        paths.add(row.path)
        segments = row.path.split(".")
        kind = segments[0]
        if kind != "Order":
            raise ValueError("explicit root missing")
        for relation in segments[1:]:
            if relation not in MODEL[kind][1]:
                raise ValueError("relation path unresolvable")
            kind = MODEL[kind][1][relation]
        if not row.fields or not set(row.fields) <= MODEL[kind][0]:
            raise ValueError("field not in target semantic type")
        if row.scope not in {"selected", "all_for_selected", "page_1"}:
            raise ValueError("scope must be explicitly named")
        if len(segments) > 1 and ".".join(segments[:-1]) not in paths:
            # Parent mapping must precede child in this deliberately small fixture.
            raise ValueError("parent context mapping absent")
    if mapping[0].path != "Order" or mapping[0].scope != "selected":
        raise ValueError("root selection must be explicit")
    return mapping


def read_explicit(mapping, root=DATA, observed_complete=None):
    """Explicit projection; trusted fixture observations are separate from read intent.

    This is NOT a real provider, a security boundary, or a complete-row proof.
    """
    validate(mapping)
    result = {}
    observed_complete = observed_complete or {}
    for entry in mapping:
        records = [root]
        for relation in entry.path.split(".")[1:]:
            records = [child for parent in records for child in parent.get(relation, [])]
        if entry.scope == "page_1":
            records = records[:1]
        values = [{field: record[field] for field in entry.fields} for record in records]
        completeness = ("partial" if entry.scope == "page_1" else
                        "complete" if observed_complete.get(entry.ident) is True else "unknown")
        result[entry.ident] = {"values": values, "scope": entry.scope,
                               "completeness": completeness,
                               "grain": "entity", "semantic_path": entry.path}
    return result


def summary(notes_slice):
    if notes_slice["completeness"] != "complete":
        return {"value": None, "status": "not_established", "editable": False}
    return {"value": ";".join(r["text"] for r in notes_slice["values"]),
            "status": "synthetic_projection", "editable": False}


STANDARD = to_mapping([
    {"id": "order", "path": "Order", "fields": ["id", "customer"], "scope": "selected"},
    {"id": "lines", "path": "Order.lines", "fields": ["id", "amount"], "scope": "all_for_selected"},
    {"id": "notes", "path": "Order.lines.notes", "fields": ["id", "text"], "scope": "all_for_selected"},
])
MARKDOWN = """| Context | Path | Fields | Scope |
|---|---|---|---|
|order|Order|id,customer|selected|
|lines|Order.lines|id,amount|all_for_selected|
|notes|Order.lines.notes|id,text|all_for_selected|
"""


class ExplicitAccessMapTests(unittest.TestCase):
    def test_01_table_and_record_same_structure(self):
        self.assertEqual(markdown_map(MARKDOWN), STANDARD)

    def test_02_unmapped_relation_not_fetched(self):
        self.assertNotIn("payments", json.dumps(read_explicit(STANDARD)))

    def test_03_unselected_fields_not_fetched(self):
        x = json.dumps(read_explicit(STANDARD))
        self.assertNotIn("private_note", x)
        self.assertNotIn("internal_flag", x)
        self.assertNotIn("sku", x)

    def test_04_multi_level_fetch_only_declared_path(self):
        x = read_explicit(STANDARD)
        self.assertEqual(len(x["lines"]["values"]), 2)
        self.assertEqual(len(x["notes"]["values"]), 2)

    def test_05_function_specific_scope(self):
        limited = STANDARD[:2]
        self.assertNotIn("notes", read_explicit(limited))

    def test_06_unknown_relation_rejected(self):
        bad = STANDARD + (Mapping("extra", "Order.lines.unknown", ("id",), "all_for_selected"),)
        with self.assertRaisesRegex(ValueError, "relation"):
            read_explicit(bad)

    def test_07_unknown_field_rejected(self):
        bad = STANDARD[:2] + (Mapping("notes", "Order.lines.notes", ("private_note",), "all_for_selected"),)
        with self.assertRaisesRegex(ValueError, "field"):
            read_explicit(bad)

    def test_08_duplicate_context_rejected(self):
        with self.assertRaisesRegex(ValueError, "duplicate"):
            read_explicit(STANDARD + (STANDARD[-1],))

    def test_09_missing_parent_map_rejected(self):
        with self.assertRaisesRegex(ValueError, "parent"):
            read_explicit((STANDARD[0], STANDARD[2]))

    def test_10_partial_cannot_claim_complete(self):
        page = STANDARD[:2] + (Mapping("notes", "Order.lines.notes", ("id", "text"), "page_1"),)
        x = read_explicit(page)
        self.assertEqual(len(x["notes"]["values"]), 1)
        self.assertEqual(x["notes"]["completeness"], "partial")
        self.assertEqual(summary(x["notes"])["status"], "not_established")

    def test_11_derived_3_to_2_is_not_invertible(self):
        x = summary(read_explicit(STANDARD, observed_complete={"notes": True})["notes"])
        self.assertEqual(x["value"], "first;second")
        self.assertFalse(x["editable"])

    def test_12_no_implicit_grant_from_valid_map(self):
        self.assertEqual(validate(STANDARD), STANDARD)
        self.assertNotIn("authorized", json.dumps(read_explicit(STANDARD)))

    def test_13_path_is_semantic_not_provider(self):
        self.assertEqual(read_explicit(STANDARD)["lines"]["semantic_path"], "Order.lines")
        self.assertNotIn("provider", json.dumps(read_explicit(STANDARD)))

    def test_14_markdown_column_ambiguity_rejected(self):
        with self.assertRaisesRegex(ValueError, "column"):
            markdown_map("|Context|Path|Fields|Scope|\n|n|Order|id|selected|bad|")

    def test_16_declared_all_not_automatically_observed_complete(self):
        x = read_explicit(STANDARD)
        self.assertEqual(x["notes"]["scope"], "all_for_selected")
        self.assertEqual(x["notes"]["completeness"], "unknown")
        self.assertEqual(summary(x["notes"])["status"], "not_established")

    def test_17_completeness_is_per_context_evidence(self):
        x = read_explicit(STANDARD, observed_complete={"lines": True})
        self.assertEqual(x["lines"]["completeness"], "complete")
        self.assertEqual(x["notes"]["completeness"], "unknown")

    def test_18_hidden_fields_change_does_not_change_projection(self):
        a = copy.deepcopy(DATA)
        b = copy.deepcopy(DATA)
        b["private_note"] = "other undisclosed value"
        b["lines"][0]["notes"][0]["internal_flag"] = 999
        self.assertEqual(read_explicit(STANDARD, root=a), read_explicit(STANDARD, root=b))

    def test_19_unmapped_relation_change_does_not_change_projection(self):
        a = copy.deepcopy(DATA)
        b = copy.deepcopy(DATA)
        b["payments"].append({"id": "P2", "value": 900})
        self.assertEqual(read_explicit(STANDARD, root=a), read_explicit(STANDARD, root=b))

    def test_20_selected_field_change_does_change_projection(self):
        a = copy.deepcopy(DATA)
        b = copy.deepcopy(DATA)
        b["lines"][0]["amount"] = 101
        self.assertNotEqual(read_explicit(STANDARD, root=a), read_explicit(STANDARD, root=b))

    def test_15_scope_not_inferred(self):
        bad = (Mapping("order", "Order", ("id",), "implicit"),)
        with self.assertRaisesRegex(ValueError, "scope"):
            read_explicit(bad)


if __name__ == "__main__":
    unittest.main(verbosity=2)
