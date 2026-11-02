#!/usr/bin/env python3
"""Finite synthetic classification of action, receipt and post-commit read evidence.

This is NOT a TEI Runner, Provider, acknowledgement protocol, receipt verifier,
Owner Business Rule or current authorization decision. All positive witness
flags are assumptions supplied by the fixture author, not discovered facts.
"""
from __future__ import annotations

from dataclasses import dataclass, replace
import unittest

QUALIFIED_RECEIPT = "qualified-receipt-only"
NOT_ESTABLISHED = "not-established"
JOIN_CANDIDATE = "action-receipt-join-candidate"
VISIBLE = "causal-read-includes-commit-candidate"
LAGGING = "read-before-commit-position"
MATCHED = "postimage-value-matches"
DIFFERENT = "postimage-value-differs"


@dataclass(frozen=True)
class Action:
    accepted_action: str
    meaning_revision: str
    target: str
    scope: str
    request_hash: str
    typed_to_request_verified: bool
    caller_ack: str  # success | unknown; does not define historical K


@dataclass(frozen=True)
class Receipt:
    accepted_action: str
    meaning_revision: str
    target: str
    scope: str
    request_hash: str
    source_log: str
    commit_position: int
    qualified: bool


@dataclass(frozen=True)
class Read:
    target: str
    scope: str
    route: str
    source_log: str | None
    applied_through: int | None
    fence_verified_for_this_read: bool
    result_scope_verified: bool
    value: int
    # A single-row read does not cover an Owner-defined all-rows business claim.
    complete_for_business_claim: bool


@dataclass(frozen=True)
class Comparison:
    historical_receipt: str
    action_join: str
    read_visibility: str
    observed_postimage: str
    business_completed: str
    current_disclosure: str


def compare(action: Action, receipt: Receipt, read: Read,
            expected_postimage: int) -> Comparison:
    """Compare *supplied* witnesses; never grant permission or certify B."""
    receipt_state = QUALIFIED_RECEIPT if receipt.qualified else NOT_ESTABLISHED
    join = (
        JOIN_CANDIDATE
        if receipt.qualified and action.typed_to_request_verified
        and (action.accepted_action, action.meaning_revision,
             action.target, action.scope, action.request_hash)
        == (receipt.accepted_action, receipt.meaning_revision,
            receipt.target, receipt.scope, receipt.request_hash)
        else NOT_ESTABLISHED
    )

    visibility = NOT_ESTABLISHED
    postimage = NOT_ESTABLISHED
    if (receipt.qualified and read.target == receipt.target
            and read.scope == receipt.scope
            and read.source_log is not None
            and read.source_log == receipt.source_log
            and read.applied_through is not None
            and read.fence_verified_for_this_read
            and read.result_scope_verified):
        if read.applied_through < receipt.commit_position:
            visibility = LAGGING
        else:
            visibility = VISIBLE
            postimage = MATCHED if read.value == expected_postimage else DIFFERENT

    # External handoff, complete Owner target-set L, rules, current grant and
    # disclosure are deliberately never inferred from K or one visible field.
    return Comparison(receipt_state, join, visibility, postimage,
                      NOT_ESTABLISHED, NOT_ESTABLISHED)


class TestPostCommitObservationPressure(unittest.TestCase):
    ACTION = Action('order.submit', 'meaning-v3', 'order-1001',
                    'tenant-A', 'raw-request-hash-1', True, 'success')
    RECEIPT = Receipt('order.submit', 'meaning-v3', 'order-1001',
                      'tenant-A', 'raw-request-hash-1',
                      'tenant-A/primary-log-v2', 100, True)
    READ = Read('order-1001', 'tenant-A', 'replica-r1',
                'tenant-A/primary-log-v2', 100, True, True, 3, False)

    def classify(self, action=None, receipt=None, read=None, expected=3):
        return compare(action or self.ACTION, receipt or self.RECEIPT,
                       read or self.READ, expected)

    def test_po01_fenced_read_is_only_value_observation_not_business_completion(self):
        result = self.classify()
        self.assertEqual(result.historical_receipt, QUALIFIED_RECEIPT)
        self.assertEqual(result.action_join, JOIN_CANDIDATE)
        self.assertEqual(result.read_visibility, VISIBLE)
        self.assertEqual(result.observed_postimage, MATCHED)
        self.assertEqual(result.business_completed, NOT_ESTABLISHED)
        self.assertEqual(result.current_disclosure, NOT_ESTABLISHED)

    def test_po02_stale_replica_does_not_undo_receipt(self):
        r = self.classify(read=replace(self.READ, applied_through=99, value=2))
        self.assertEqual(r.historical_receipt, QUALIFIED_RECEIPT)
        self.assertEqual(r.read_visibility, LAGGING)
        self.assertEqual(r.observed_postimage, NOT_ESTABLISHED)

    def test_po03_ack_loss_with_qualified_receipt_keeps_historical_evidence(self):
        # 'ACK lost' is a caller observation; a separately recovered K can remain.
        r = self.classify(action=replace(self.ACTION, caller_ack='unknown'))
        self.assertEqual(r.historical_receipt, QUALIFIED_RECEIPT)
        self.assertEqual(r.business_completed, NOT_ESTABLISHED)

    def test_po04_without_qualified_receipt_equal_value_proves_neither_result(self):
        r = self.classify(action=replace(self.ACTION, caller_ack='unknown'),
                          receipt=replace(self.RECEIPT, qualified=False))
        self.assertEqual(r.historical_receipt, NOT_ESTABLISHED)
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)
        self.assertEqual(r.observed_postimage, NOT_ESTABLISHED)

    def test_po05_same_value_wrong_actual_target(self):
        r = self.classify(read=replace(self.READ, target='order-1002'))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)

    def test_po06_same_numeric_position_on_another_log_is_incomparable(self):
        r = self.classify(read=replace(self.READ, source_log='tenant-A/archive-log'))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)

    def test_po07_position_claim_without_per_read_fence_is_inadequate(self):
        r = self.classify(read=replace(self.READ, fence_verified_for_this_read=False))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)

    def test_po08_unknown_replica_applied_position_is_not_current(self):
        r = self.classify(read=replace(self.READ, applied_through=None))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)

    def test_po09_newer_read_with_different_value_does_not_negate_commit(self):
        r = self.classify(read=replace(self.READ, applied_through=110, value=4))
        self.assertEqual(r.historical_receipt, QUALIFIED_RECEIPT)
        self.assertEqual(r.read_visibility, VISIBLE)
        self.assertEqual(r.observed_postimage, DIFFERENT)
        self.assertEqual(r.business_completed, NOT_ESTABLISHED)

    def test_po10_unverified_typed_to_byte_mapping_has_no_action_join(self):
        r = self.classify(action=replace(self.ACTION, typed_to_request_verified=False))
        self.assertEqual(r.action_join, NOT_ESTABLISHED)
        self.assertEqual(r.read_visibility, VISIBLE)  # receipt/read is a separate claim

    def test_po11_same_target_wrong_request_hash_is_not_same_action(self):
        r = self.classify(action=replace(self.ACTION, request_hash='other-payload'))
        self.assertEqual(r.action_join, NOT_ESTABLISHED)
        self.assertEqual(r.historical_receipt, QUALIFIED_RECEIPT)

    def test_po12_same_hash_wrong_meaning_revision_is_not_owner_action_join(self):
        r = self.classify(action=replace(self.ACTION, meaning_revision='meaning-v4'))
        self.assertEqual(r.action_join, NOT_ESTABLISHED)

    def test_po13_read_scope_unverified_even_with_token_and_matching_value(self):
        r = self.classify(read=replace(self.READ, result_scope_verified=False))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)

    def test_po14_single_row_not_complete_owner_target_set(self):
        r = self.classify(read=replace(self.READ, complete_for_business_claim=True))
        self.assertEqual(r.read_visibility, VISIBLE)
        self.assertEqual(r.business_completed, NOT_ESTABLISHED)

    def test_po15_tenant_scope_mismatch_does_not_qualify_read(self):
        r = self.classify(read=replace(self.READ, scope='tenant-B'))
        self.assertEqual(r.read_visibility, NOT_ESTABLISHED)


if __name__ == '__main__':
    unittest.main(verbosity=2)
