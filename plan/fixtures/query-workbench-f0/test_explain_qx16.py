"""Negative acceptance checks for the fixture-only QX-16 explanation."""
import unittest
from explain_qx16 import build_reference


class QueryExplainQX16Tests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data = build_reference()["scenarios"]

    def claim(self, case, name):
        return self.data[case]["claims"][name]

    def test_observed_partial_sum_has_only_a_conditional_lower_bound(self):
        claim = self.claim("R_partial", "line_sum")
        self.assertEqual(claim["status"], "conditional-lower-bound")
        self.assertEqual(claim["conclusion"], {"gte": 600})
        self.assertNotEqual(claim["status"], "exact-synthetic-scope")
        self.assertEqual(self.claim("R_partial", "line_sum_gte_500")["conclusion"], True)

    def test_same_observation_does_not_determine_whole_sum(self):
        self.assertEqual(self.claim("L1_complete", "line_sum")["conclusion"], 1100)
        self.assertEqual(self.claim("L2_complete", "line_sum")["conclusion"], 1600)
        self.assertIsNone(self.claim("empty_partial", "line_sum")["conclusion"])

    def test_partial_delivery_does_not_invalidate_independent_line_sum(self):
        self.assertEqual(self.claim("line_complete_delivery_partial", "line_sum")["conclusion"], 1100)
        self.assertEqual(self.claim("line_complete_delivery_partial", "join_sum")["status"], "observed-slice-only")
        self.assertIsNone(self.claim("line_complete_delivery_partial", "join_sum")["conclusion"])

    def test_join_query_sum_is_not_business_line_sum(self):
        self.assertEqual(self.claim("L1_complete", "join_sum")["conclusion"], 2200)
        self.assertEqual(self.claim("L1_complete", "line_sum")["conclusion"], 1100)

    def test_signed_model_cannot_use_partial_nonnegative_law(self):
        self.assertEqual(self.claim("signed_model_complete", "line_sum")["conclusion"], 400)
        self.assertFalse(self.claim("signed_model_complete", "line_sum_gte_500")["conclusion"])

    def test_exists_witness_and_unknown_absence(self):
        self.assertEqual(self.claim("R_partial", "exists_500")["status"], "witness-true")
        self.assertFalse(self.claim("R_partial", "not_exists_500")["conclusion"])
        self.assertIsNone(self.claim("empty_partial", "not_exists_500")["conclusion"])
        self.assertTrue(self.claim("empty_complete", "not_exists_500")["conclusion"])

    def test_empty_sum_is_sql_null_not_automatically_zero(self):
        self.assertEqual(self.claim("empty_complete", "line_sum")["status"], "exact-synthetic-scope")
        self.assertIsNone(self.claim("empty_complete", "line_sum")["conclusion"])
        self.assertIsNone(self.claim("empty_complete", "line_sum_gte_500")["conclusion"])

    def test_left_join_unmatched_row_depends_on_right_completeness(self):
        partial = self.claim("left_partial_no_match", "left_unmatched")
        empty = self.claim("left_complete_no_match", "left_unmatched")
        match = self.claim("left_complete_D1", "left_unmatched")
        self.assertEqual(partial["status"], "observed-slice-only")
        self.assertIsNone(partial["conclusion"])
        self.assertTrue(empty["conclusion"]["unmatched"])
        self.assertFalse(match["conclusion"]["unmatched"])

    def test_no_host_or_provider_acceptance(self):
        data = build_reference()
        self.assertIn("Provider completeness or read snapshot", data["notProven"])
        self.assertEqual(data["scenarios"]["R_partial"]["inputContext"]["businessAcceptance"], "not-accepted")


if __name__ == "__main__":
    unittest.main()
