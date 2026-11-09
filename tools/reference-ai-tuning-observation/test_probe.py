"""Independent synthetic negative controls for a read-only observation join.

No TEI Parser, DB, OpenTelemetry SDK, Host, AI service, or actual Provider runs.
Expected outcomes are fixed before invoking the subject under test.
"""
from __future__ import annotations

import copy
import io
import json
import unittest
from unittest.mock import patch

from probe import evaluate, main


def fixture() -> dict:
    design = {
        "targetRef": "function:demo.search",
        "queryRef": "query:demo.items",
        "revision": "R-A",
        "worldGeneration": 3,
        "scope": "scope-A",
        "selectedProvider": "B",
    }
    return {
        "consumer": {"mayDisclose": True, "grantEpoch": "G-7"},
        "design": design,
        "delivery": {
            "expectedParts": 2,
            "parts": [
                {"index": 1, "sha256": "a" * 64},
                {"index": 2, "sha256": "b" * 64},
            ],
            "endMarker": True,
        },
        "statistics": {
            "windowId": "W-1",
            "revision": "R-A",
            "worldGeneration": 3,
            "totalCalls": 4,
            "sampledCalls": 4,
            "droppedCalls": 0,
        },
        "observations": [{
            "kind": "db-read",
            "attemptId": "A-1",
            "queryRef": design["queryRef"],
            "targetRef": design["targetRef"],
            "selectedProvider": "B",
            "invokedProvider": "B",
            "revision": "R-A",
            "worldGeneration": 3,
            "scope": "scope-A",
            "grantEpoch": "G-7",
            "windowId": "W-1",
            "queryReadCalls": 4,
        }],
    }


class TestObservationJoin(unittest.TestCase):
    def test_consistent_but_unverified(self):
        observed = evaluate(fixture())
        self.assertEqual("input-consistent", observed["status"])
        self.assertEqual("investigate-repeated-reads", observed["candidate"])
        self.assertEqual("reference-only-not-actual-invocation", observed["qualification"])
        self.assertFalse(observed["candidateIsVerified"])
        self.assertFalse(observed["optimizationApplied"])
        self.assertFalse(observed["performanceImproved"])

    def test_no_repetition_not_absence_proof(self):
        p = fixture()
        p["observations"][0]["queryReadCalls"] = 1
        observed = evaluate(p)
        self.assertEqual("none-observed", observed["candidate"])
        self.assertEqual("input-consistent", observed["status"])

    def test_sampled_statistics_are_not_complete(self):
        p = fixture()
        p["statistics"]["sampledCalls"] = 3
        p["statistics"]["droppedCalls"] = 1
        self.assertEqual("window-sampled", evaluate(p)["coverage"])

    def test_no_permission_leaks_no_identity_or_counts(self):
        p = fixture()
        p["consumer"]["mayDisclose"] = False
        p["design"]["targetRef"] = "TOP-SECRET"
        self.assertEqual({"status": "withheld"}, evaluate(p))

    def test_missing_grant(self):
        p = fixture()
        del p["consumer"]["grantEpoch"]
        self.assertEqual("missing-current-grant", evaluate(p)["reason"])

    def test_provider_b_selected_but_a_called(self):
        p = fixture()
        p["observations"][0]["invokedProvider"] = "A"
        observed = evaluate(p)
        self.assertEqual("unresolved", observed["status"])
        self.assertEqual("incompatible-call-binding", observed["reason"])

    def test_source_revision_mismatch(self):
        p = fixture()
        p["observations"][0]["revision"] = "R-B"
        self.assertEqual("incompatible-call-binding", evaluate(p)["reason"])

    def test_a_b_a_generation_rejects_old_data(self):
        p = fixture()
        p["statistics"]["worldGeneration"] = 1
        self.assertEqual("stale-statistics", evaluate(p)["reason"])

    def test_changed_tenant_scope(self):
        p = fixture()
        p["observations"][0]["scope"] = "scope-B"
        self.assertEqual("incompatible-call-binding", evaluate(p)["reason"])

    def test_current_grant_epoch_change(self):
        p = fixture()
        p["consumer"]["grantEpoch"] = "G-8"
        self.assertEqual("incompatible-call-binding", evaluate(p)["reason"])

    def test_missing_reply_part(self):
        p = fixture()
        p["delivery"]["parts"].pop()
        self.assertEqual("incomplete-delivery", evaluate(p)["reason"])

    def test_missing_end_marker(self):
        p = fixture()
        p["delivery"]["endMarker"] = False
        self.assertEqual("incomplete-delivery", evaluate(p)["reason"])

    def test_duplicate_page(self):
        p = fixture()
        p["delivery"]["parts"][1] = copy.deepcopy(p["delivery"]["parts"][0])
        self.assertEqual("invalid-delivery", evaluate(p)["reason"])

    def test_malformed_digest(self):
        p = fixture()
        p["delivery"]["parts"][1]["sha256"] = "not-a-digest"
        self.assertEqual("invalid-delivery", evaluate(p)["reason"])

    def test_impossible_sampling(self):
        p = fixture()
        p["statistics"]["sampledCalls"] = 5
        self.assertEqual("invalid-sampling", evaluate(p)["reason"])

    def test_zero_calls_is_observation_not_false_absence(self):
        p = fixture()
        p["observations"][0]["queryReadCalls"] = 0
        observed = evaluate(p)
        self.assertEqual("none-observed", observed["candidate"])
        self.assertNotIn("not-found", json.dumps(observed))

    def test_no_raw_inputs_in_result(self):
        p = fixture()
        p["password"] = "TOP-SECRET"
        p["sql"] = "SELECT * FROM private_table"
        text = json.dumps(evaluate(p))
        for sensitive in ("TOP-SECRET", "private_table", "scope-A",
                          "query:demo.items", "G-7", "A-1"):
            self.assertNotIn(sensitive, text)

    def test_stdin_duplicate_fields_are_refused(self):
        fake_input = io.StringIO('{"consumer": {}, "consumer": {}}\n')
        output = io.StringIO()
        with patch("sys.stdin", fake_input), patch("sys.stdout", output):
            self.assertEqual(0, main())
        self.assertEqual("invalid-input", json.loads(output.getvalue())["reason"])

    def test_stdin_nonfinite_is_refused(self):
        fake_input = io.StringIO('{"consumer": {"mayDisclose":true}, "x": NaN}\n')
        output = io.StringIO()
        with patch("sys.stdin", fake_input), patch("sys.stdout", output):
            self.assertEqual(0, main())
        self.assertEqual("invalid-input", json.loads(output.getvalue())["reason"])


if __name__ == "__main__":
    unittest.main()
