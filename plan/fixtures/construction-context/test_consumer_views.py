#!/usr/bin/env python3
"""Deliberately narrow, pure negative tests for W0b human/AI material equality."""
import copy
import tempfile
import unittest
from pathlib import Path

from consumer_views import (
    LABELS, UNKNOWN, export_views, parse_markdown,
    render_markdown, validate_witness,
)


def sample():
    return {
        "subjectKind": "semantic-relation",
        "subjectId": "sales.order.customer",
        "binaryRevision": "a" * 40,
        "worldFingerprint": "sha256:" + "b" * 64,
        "sourceIndexFingerprint": "sha256:" + "c" * 64,
        "modelRevision": "sha256:" + "d" * 64,
        "sourceArtifacts": [
            "reference/semantic/sales/order.tei",
        ],
        "traceRows": 2,
        "traceUnresolved": 0,
        "validationDiagnostics": 3,
        "evidenceKind": "read-only-reference-cli-observation",
        "disclosureStatus": "reference-fixture-only-no-policy-grant",
        "notEstablished": list(UNKNOWN),
    }


class DualViewReferenceTest(unittest.TestCase):
    def test_roundtrip_one_observation_to_two_file_surfaces(self):
        with tempfile.TemporaryDirectory() as root:
            j, m = export_views(sample(), root)
            self.assertEqual(parse_markdown(m.read_text(encoding="utf-8")), sample())
            self.assertIn("sourceArtifacts", j.read_text(encoding="utf-8"))

    def test_changed_semantic_subject_rejected(self):
        rec = dict(sample(), subjectId="sales.order.other")
        with self.assertRaises(ValueError):
            validate_witness(rec)

    def test_missing_provenance_rejected(self):
        rec = dict(sample())
        del rec["sourceIndexFingerprint"]
        with self.assertRaises(ValueError):
            validate_witness(rec)

    def test_unauthorized_extra_payload_rejected(self):
        rec = dict(sample(), privateCreditLimit=700)
        with self.assertRaises(ValueError):
            validate_witness(rec)

    def test_secret_source_path_rejected(self):
        for path in ("../secrets/credit.txt", "/tmp/secrets.txt", "a|malicious", "a\nprivate"):
            with self.subTest(path=path):
                with self.assertRaises(ValueError):
                    validate_witness(dict(sample(), sourceArtifacts=[path]))

    def test_duplicate_or_reordered_source_paths_rejected(self):
        original = sample()
        for paths in (
            [original["sourceArtifacts"][0]] * 2,
            ["z.tei", "a.tei"],
            [],
        ):
            with self.subTest(paths=paths):
                with self.assertRaises(ValueError):
                    validate_witness(dict(sample(), sourceArtifacts=paths))

    def test_false_disclosure_or_business_authority_rejected(self):
        for key, value in (
            ("disclosureStatus", "authorized-for-external-agent"),
            ("evidenceKind", "business-accepted"),
            ("notEstablished", []),
        ):
            with self.subTest(key=key):
                with self.assertRaises(ValueError):
                    validate_witness(dict(sample(), **{key: value}))

    def test_rewritten_markdown_value_is_detected(self):
        markdown = render_markdown(sample())
        altered = markdown.replace("| 直接参照の観測数 | 2 |", "| 直接参照の観測数 | 3 |")
        parsed = parse_markdown(altered)
        self.assertNotEqual(parsed, sample())

    def test_extra_or_duplicate_markdown_row_rejected(self):
        text = render_markdown(sample())
        row = "| 意味対象ID | \"sales.order.customer\" |"
        self.assertIn(row, text)
        with self.assertRaises(ValueError):
            parse_markdown(text.replace(row, row + "\n" + row))
        with self.assertRaises(ValueError):
            parse_markdown(text.replace(row, row + "\n| 秘密 | 700 |"))

    def test_invalid_markdown_json_or_column_injection_rejected(self):
        text = render_markdown(sample())
        original = "| 直接参照の観測数 | 2 |"
        with self.assertRaises(ValueError):
            parse_markdown(text.replace(original, "| 直接参照の観測数 | two |"))
        with self.assertRaises(ValueError):
            parse_markdown(text.replace(original, "| 直接参照の観測数 | 2 | secret |"))

    def test_numeric_and_list_values_are_not_coerced_to_strings(self):
        result = parse_markdown(render_markdown(sample()))
        self.assertIsInstance(result["traceRows"], int)
        self.assertEqual(result["sourceArtifacts"], ["reference/semantic/sales/order.tei"])

    def test_witness_cannot_claim_more_than_observed_rows(self):
        with self.assertRaises(ValueError):
            validate_witness(dict(sample(), traceUnresolved=3))

    def test_target_directory_must_exist(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                export_views(sample(), Path(tmp) / "nonexistent")


if __name__ == "__main__":
    unittest.main(verbosity=2)
