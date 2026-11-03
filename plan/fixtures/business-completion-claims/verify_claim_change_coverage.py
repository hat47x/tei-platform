#!/usr/bin/env python3
"""Synthetic coverage test for claim-relative selective requalification.

No TEI Provider, event stream or authorization system is used here. Coverage,
source binding, and event completeness are trusted *fixture inputs*; this is
not a proof that any real change feed is gapless or any claim is currently true.
"""
from dataclasses import dataclass
import unittest

REUSE = 'reuse-candidate-for-this-change-window'
RECHECK = 'recheck-affected-claim'
UNKNOWN = 'unresolved-change-window'


@dataclass(frozen=True)
class StreamNeed:
    stream: str
    binding: str
    checkpoint: int
    decision_position: int


@dataclass(frozen=True)
class StreamCoverage:
    stream: str
    binding: str
    first: int
    last: int
    gapless: bool
    event_set_complete: bool


@dataclass(frozen=True)
class Change:
    stream: str
    position: int
    category: str | None
    relation: str  # same | proven-disjoint | not-established


@dataclass(frozen=True)
class Claim:
    dependencies_closed: bool
    dependency_categories: frozenset[str]
    needs: tuple[StreamNeed, ...]


def classify_window(claim: Claim, coverages: tuple[StreamCoverage, ...],
                    changes: tuple[Change, ...]) -> str:
    """Conservative *synthetic* impact classification, not a permission decision."""
    if not claim.dependencies_closed:
        return UNKNOWN
    coverage_by_stream = {c.stream: c for c in coverages}
    if len(coverage_by_stream) != len(coverages):
        return UNKNOWN
    windows = {}
    for need in claim.needs:
        if need.stream in windows or need.checkpoint > need.decision_position:
            return UNKNOWN
        coverage = coverage_by_stream.get(need.stream)
        if (coverage is None or coverage.binding != need.binding
                or not coverage.gapless or not coverage.event_set_complete
                or coverage.first > need.checkpoint + 1
                or coverage.last < need.decision_position):
            return UNKNOWN
        windows[need.stream] = (need.checkpoint, need.decision_position)

    affected = False
    for event in changes:
        if event.stream not in windows:
            # Events on irrelevant streams do not establish completeness.
            continue
        start, end = windows[event.stream]
        if not (start < event.position <= end):
            continue
        if event.category is None:
            return UNKNOWN
        if event.category not in claim.dependency_categories:
            continue
        if event.relation == 'same':
            affected = True
        elif event.relation != 'proven-disjoint':
            return UNKNOWN
    return RECHECK if affected else REUSE


class TestClaimChangeCoverage(unittest.TestCase):
    NEED = StreamNeed('order-primary', 'tenant-A/primary/v2', 10, 15)
    CLOSED = Claim(True, frozenset({'cohort-history', 'owner-rule'}), (NEED,))
    COMPLETE = StreamCoverage('order-primary', 'tenant-A/primary/v2', 11, 15, True, True)

    def test_cwt01_complete_disjoint_window_is_only_reuse_candidate(self):
        event = Change('order-primary', 13, 'cohort-history', 'proven-disjoint')
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), (event,)), REUSE)

    def test_cwt02_gap_in_feed_forbids_reuse(self):
        gap = StreamCoverage('order-primary', self.NEED.binding, 11, 15, False, True)
        self.assertEqual(classify_window(self.CLOSED, (gap,), ()), UNKNOWN)

    def test_cwt03_watermark_lag_forbids_reuse(self):
        lag = StreamCoverage('order-primary', self.NEED.binding, 11, 14, True, True)
        self.assertEqual(classify_window(self.CLOSED, (lag,), ()), UNKNOWN)

    def test_cwt04_aba_seen_with_complete_feed_requires_recheck(self):
        events = (Change('order-primary', 12, 'cohort-history', 'same'),
                  Change('order-primary', 14, 'cohort-history', 'same'))
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), events), RECHECK)

    def test_cwt05_unclosed_query_predicate_is_unknown(self):
        open_claim = Claim(False, self.CLOSED.dependency_categories, (self.NEED,))
        self.assertEqual(classify_window(open_claim, (self.COMPLETE,), ()), UNKNOWN)

    def test_cwt06_second_provider_missing_coverage_is_unknown(self):
        other = StreamNeed('replica', 'tenant-A/replica/v1', 8, 12)
        claim = Claim(True, self.CLOSED.dependency_categories, (self.NEED, other))
        self.assertEqual(classify_window(claim, (self.COMPLETE,), ()), UNKNOWN)

    def test_cwt07_same_stream_name_different_binding_is_unknown(self):
        wrong = StreamCoverage('order-primary', 'tenant-B/primary/v2', 11, 15, True, True)
        self.assertEqual(classify_window(self.CLOSED, (wrong,), ()), UNKNOWN)

    def test_cwt08_unknown_relation_is_not_disjoint(self):
        event = Change('order-primary', 13, 'owner-rule', 'not-established')
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), (event,)), UNKNOWN)

    def test_cwt09_unknown_event_category_is_not_no_change(self):
        event = Change('order-primary', 13, None, 'same')
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), (event,)), UNKNOWN)

    def test_cwt10_classified_irrelevant_change_is_only_candidate(self):
        event = Change('order-primary', 13, 'commentary', 'same')
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), (event,)), REUSE)

    def test_cwt11_event_set_unverified_is_unknown(self):
        incomplete = StreamCoverage('order-primary', self.NEED.binding, 11, 15, True, False)
        self.assertEqual(classify_window(self.CLOSED, (incomplete,), ()), UNKNOWN)

    def test_cwt12_known_impact_in_a_complete_window_requires_recheck(self):
        event = Change('order-primary', 13, 'owner-rule', 'same')
        self.assertEqual(classify_window(self.CLOSED, (self.COMPLETE,), (event,)), RECHECK)


if __name__ == '__main__':
    unittest.main(verbosity=2)
