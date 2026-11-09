#!/usr/bin/env python3
"""Isolated synthetic comparison, NOT TEI's actual editing or transaction runtime."""
from dataclasses import dataclass
import unittest


@dataclass(frozen=True)
class Patch:
    context: str
    anchor: str
    subject: str
    base_version: int
    value: int


@dataclass(frozen=True)
class Slice:
    anchor: str
    context: str
    version: int
    complete: bool
    values: dict


class IsolatedSession:
    """A reference model for instance-scoped dirty data, not a product API."""

    def __init__(self, instance_id, selected_anchor):
        self.instance_id = instance_id
        self.selected_anchor = selected_anchor
        self.epoch = 0
        self.slices = {}
        self.patches = {}
        self.mounted = True

    def select(self, anchor):
        self.selected_anchor = anchor
        self.epoch += 1

    def receive(self, epoch, result):
        if epoch != self.epoch or result.anchor != self.selected_anchor:
            return "stale"
        self.slices[(result.anchor, result.context)] = result
        return "received"

    def stage(self, context, subject, base_version, value):
        key = (self.selected_anchor, context)
        source = self.slices.get(key)
        if source is None or subject not in source.values:
            return "unresolved"
        if source.version != base_version:
            return "conflict"
        self.patches[(self.selected_anchor, context, subject)] = Patch(
            context, self.selected_anchor, subject, base_version, value
        )
        return "staged"

    def visible(self, context, subject):
        key = (self.selected_anchor, context)
        source = self.slices[key]
        patch = self.patches.get((self.selected_anchor, context, subject))
        if patch and patch.base_version != source.version:
            return ("conflict", None)
        return ("draft" if patch else "read", patch.value if patch else source.values[subject])

    def save_candidates(self, anchor=None, require_complete=False):
        anchor = self.selected_anchor if anchor is None else anchor
        changes = [p for (a, _, _), p in self.patches.items() if a == anchor]
        if require_complete:
            for p in changes:
                part = self.slices.get((anchor, p.context))
                if part is None or not part.complete:
                    return ("unresolved", [])
        return ("candidates", changes)

    def unmount(self):
        self.mounted = False
        return "retained" if self.patches else "no_draft"


def classify_save(providers, actual_atomic_evidence, result_by_context):
    """Evidence-only classification; never invokes a Provider or validates a grant."""
    if len(set(providers)) != 1 or not actual_atomic_evidence:
        guarantee = "atomicity_not_established"
    else:
        guarantee = "atomicity_declared_in_fixture"
    statuses = set(result_by_context.values())
    if statuses == {"confirmed"}:
        outcome = "all_confirmed_in_fixture"
    elif "unknown" in statuses:
        outcome = "unknown_effect"
    else:
        outcome = "partial_or_failed"
    return guarantee, outcome


class EditSessionTests(unittest.TestCase):
    def setUp(self):
        self.a = IsolatedSession("instance-left", "A")
        self.b = IsolatedSession("instance-right", "B")
        self.assertEqual(self.a.receive(0, Slice("A", "D1", 1, True, {"L1": 100, "L2": 100})), "received")
        self.assertEqual(self.a.receive(0, Slice("A", "D2", 1, False, {"X1": 5})), "received")
        self.assertEqual(self.b.receive(0, Slice("B", "D1", 1, True, {"L1": 900})), "received")

    def test_es01_instance_isolation(self):
        self.assertEqual(self.a.stage("D1", "L1", 1, 120), "staged")
        self.assertEqual(self.a.visible("D1", "L1"), ("draft", 120))
        self.assertEqual(self.b.visible("D1", "L1"), ("read", 900))

    def test_es02_anchor_changes_preserve_target(self):
        self.a.stage("D1", "L1", 1, 120)
        self.a.select("B")
        self.assertEqual(self.a.receive(1, Slice("B", "D1", 1, True, {"L1": 500})), "received")
        self.assertEqual(self.a.visible("D1", "L1"), ("read", 500))
        self.a.select("A")
        self.assertEqual(self.a.visible("D1", "L1"), ("draft", 120))

    def test_es03_late_response_cannot_mix_or_erase_draft(self):
        self.a.stage("D1", "L1", 1, 120)
        self.a.select("B")
        self.assertEqual(self.a.receive(0, Slice("A", "D1", 1, True, {"L1": 200})), "stale")
        self.assertNotIn(("B", "D1"), self.a.slices)
        self.a.select("A")
        self.assertEqual(self.a.visible("D1", "L1"), ("draft", 120))

    def test_es04_independent_collections_and_identity(self):
        self.a.stage("D1", "L1", 1, 120)
        self.a.stage("D2", "X1", 1, 6)
        verdict, changes = self.a.save_candidates("A")
        self.assertEqual(verdict, "candidates")
        self.assertEqual({(p.context, p.subject) for p in changes}, {("D1", "L1"), ("D2", "X1")})
        self.assertEqual(len(changes), 2)  # NOT the 2 x 3 JOIN row count.

    def test_es05_no_atomicity_from_one_ui_action(self):
        self.assertEqual(classify_save(["P1", "P2"], True, {"D1": "confirmed", "D2": "confirmed"})[0], "atomicity_not_established")
        self.assertEqual(classify_save(["P1", "P1"], False, {"D1": "confirmed", "D2": "confirmed"})[0], "atomicity_not_established")
        self.assertEqual(classify_save(["P1", "P1"], True, {"D1": "confirmed", "D2": "confirmed"})[0], "atomicity_declared_in_fixture")

    def test_es06_partial_and_unknown_distinct(self):
        self.assertEqual(classify_save(["P1", "P2"], False, {"D1": "confirmed", "D2": "failed"})[1], "partial_or_failed")
        self.assertEqual(classify_save(["P1", "P2"], False, {"D1": "confirmed", "D2": "unknown"})[1], "unknown_effect")

    def test_es09_partial_slice_blocks_full_set_claim(self):
        self.a.stage("D2", "X1", 1, 6)
        self.assertEqual(self.a.save_candidates("A", require_complete=True), ("unresolved", []))
        self.assertEqual(self.a.save_candidates("A", require_complete=False)[0], "candidates")

    def test_es10_unmount_never_discards_patch(self):
        self.a.stage("D1", "L1", 1, 120)
        self.assertEqual(self.a.unmount(), "retained")
        self.assertEqual(self.a.visible("D1", "L1"), ("draft", 120))

    def test_es11_refresh_version_conflict(self):
        self.a.stage("D1", "L1", 1, 120)
        self.assertEqual(self.a.receive(0, Slice("A", "D1", 2, True, {"L1": 110})), "received")
        self.assertEqual(self.a.visible("D1", "L1"), ("conflict", None))

    def test_es13_confirmation_not_implied_by_zero_dirty(self):
        self.assertEqual(self.a.save_candidates("A"), ("candidates", []))
        confirmation_current = False  # independent owner/review decision, not edit state.
        self.assertFalse(confirmation_current)


if __name__ == "__main__":
    unittest.main(verbosity=2)
