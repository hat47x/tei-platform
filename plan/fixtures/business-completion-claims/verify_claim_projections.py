#!/usr/bin/env python3
"""Synthetic two-world witnesses for claim-relative business completion.

This only proves that a particular read-only projection is insufficient to
settle a particular claim. Owner policies below are hypothetical. No TEI
runtime, Provider, original X-TEA model, or real business effect is exercised.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Callable
import unittest

World = dict[str, Any]


@dataclass(frozen=True)
class Pair:
    case_id: str
    claim: str
    a: World
    b: World
    visible: Callable[[World], object]
    asserted: Callable[[World], bool]


CASES = (
    Pair(
        "BC-01", "Every required line has a positive quantity",
        {"rows": {"L1": 2, "L2": 1}},
        {"rows": {"L1": 2, "L2": -1}},
        lambda w: {"fetched": {"L1": w["rows"]["L1"]}, "validation": "pass"},
        lambda w: all(q > 0 for q in w["rows"].values()),
    ),
    Pair(
        "BC-05", "External recipient has accepted the shipment",
        {"order": "H", "request_committed": True, "external_accepted": True},
        {"order": "H", "request_committed": True, "external_accepted": False},
        lambda w: {"order": w["order"], "request_committed": w["request_committed"]},
        lambda w: w["external_accepted"],
    ),
    Pair(
        "BC-07", "Original business process is complete by authorized recovery",
        {"parent_ok": True, "child_ok": True, "initial_partial": True, "recovery_for": "B1"},
        {"parent_ok": True, "child_ok": True, "initial_partial": True, "recovery_for": "OTHER"},
        lambda w: {"parent_ok": w["parent_ok"], "child_ok": w["child_ok"]},
        lambda w: w["parent_ok"] and w["child_ok"] and w["recovery_for"] == "B1",
    ),
    Pair(
        "BC-08", "Hypothetical P3 confirmation remains usable (no relevant edits)",
        {"quantity": 2, "confirmation": True, "relevant_edits": []},
        {"quantity": 2, "confirmation": True, "relevant_edits": [3, 2]},
        lambda w: {"quantity": w["quantity"], "confirmation": w["confirmation"]},
        lambda w: w["confirmation"] and not w["relevant_edits"],
    ),
    Pair(
        "BC-09", "Hypothetical P3 confirmation remains usable (no cohort edits)",
        {"current_cohort": ("L1", "L2"), "membership_edits": []},
        {"current_cohort": ("L1", "L2"), "membership_edits": ["+L3", "-L3"]},
        lambda w: {"current_cohort": w["current_cohort"]},
        lambda w: not w["membership_edits"],
    ),
    Pair(
        "BC-10", "Intended target is the target of the committed effect",
        {"intent": "1001", "effect_target": "1001", "receipt": "K1"},
        {"intent": "1001", "effect_target": "1002", "receipt": "K1"},
        lambda w: {"intent": w["intent"], "receipt": w["receipt"]},
        lambda w: w["effect_target"] == w["intent"],
    ),
    Pair(
        "BC-11", "Original historical output was P1 (effect receipt alone is insufficient)",
        {"receipt": "K1", "replayed": True, "historical_output": "P1"},
        {"receipt": "K1", "replayed": True, "historical_output": "P2"},
        lambda w: {"receipt": w["receipt"], "replayed": w["replayed"]},
        lambda w: w["historical_output"] == "P1",
    ),
    Pair(
        "BC-12", "The first underlying detail amount is 100",
        {"amounts": (100, 100)},
        {"amounts": (50, 150)},
        lambda w: {"chart_sum": sum(w["amounts"])},
        lambda w: w["amounts"][0] == 100,
    ),
    Pair(
        "BC-15", "Consumer actually loaded the complete current artifact generation",
        {"published_revision": "R2", "complete": True, "expected_digest": "d2", "loaded_digest": "d2"},
        {"published_revision": "R2", "complete": True, "expected_digest": "d2", "loaded_digest": "d1"},
        lambda w: {"published_revision": w["published_revision"], "complete": w["complete"], "expected_digest": w["expected_digest"]},
        lambda w: w["loaded_digest"] == w["expected_digest"],
    ),
    Pair(
        "BC-16", "Edited value is durably committed, not merely staged in a session",
        {"editing_status": "submitted", "persisted_status": "submitted"},
        {"editing_status": "submitted", "persisted_status": "draft"},
        lambda w: {"editing_status": w["editing_status"]},
        lambda w: w["persisted_status"] == w["editing_status"],
    ),

    Pair(
        "BC-13", "Executed Business Action purpose equals the intended purpose",
        {"intent_action": "sales.order.submit", "executed_action": "sales.order.submit",
         "order": "1001", "receipt": "K1", "updated": True},
        {"intent_action": "sales.order.submit", "executed_action": "sales.order.cancel",
         "order": "1001", "receipt": "K1", "updated": True},
        lambda w: {"intent_action": w["intent_action"], "order": w["order"],
                   "receipt": w["receipt"], "updated": w["updated"]},
        lambda w: w["executed_action"] == w["intent_action"],
    ),
    Pair(
        "BC-14", "Cancelling the parent leaves its pending write uncommitted",
        {"child_return": "01", "parent_action": "rollback", "parent_write_committed": False},
        {"child_return": "01", "parent_action": "rollback", "parent_write_committed": True},
        lambda w: {"child_return": w["child_return"], "parent_action": w["parent_action"]},
        lambda w: not w["parent_write_committed"],
    ),
)


class ProjectionIndistinguishability(unittest.TestCase):
    def test_same_visible_evidence_can_support_opposite_claim_truths(self) -> None:
        for case in CASES:
            with self.subTest(case=case.case_id, claim=case.claim):
                self.assertEqual(case.visible(case.a), case.visible(case.b))
                self.assertTrue(case.asserted(case.a))
                self.assertFalse(case.asserted(case.b))


if __name__ == "__main__":
    unittest.main(verbosity=2)
