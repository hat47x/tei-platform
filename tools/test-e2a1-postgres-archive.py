#!/usr/bin/env python3
"""Focused, self-contained tamper cases for the historical E2a-1 archive."""
import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location(
    "tei_e2a1_archive", ROOT / "tools/verify-e2a1-postgres-archive.py")
verify_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_module)


class RealPostgresArchive(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(prefix="tei-e2a1-archive-")
        self.addCleanup(temp.cleanup)
        self.receipt = Path(temp.name) / "receipt.json"
        self.receipt.write_bytes(verify_module.RECEIPT.read_bytes())

    def test_byte_exact_original_remains_unqualified(self):
        result = verify_module.verify(self.receipt)
        self.assertEqual("UNQUALIFIED", result["productGate"])
        self.assertEqual("real-postgres-ephemeral-fixture", result["evidenceNature"])

    def test_whitespace_mutation_is_refused(self):
        self.receipt.write_bytes(self.receipt.read_bytes() + b"\n")
        with self.assertRaisesRegex(ValueError, "byte SHA256 mismatch"):
            verify_module.verify(self.receipt)

    def test_g2_forgery_is_refused(self):
        source = self.receipt.read_bytes()
        self.assertIn(b'"productE2": false', source)
        self.receipt.write_bytes(source.replace(b'"productE2": false',
                                                b'"productE2": true'))
        with self.assertRaisesRegex(ValueError, "byte SHA256 mismatch"):
            verify_module.verify(self.receipt)

    def test_deleted_receipt_is_refused(self):
        self.receipt.unlink()
        with self.assertRaisesRegex(ValueError, "regular file"):
            verify_module.verify(self.receipt)

    def test_receipt_symlink_is_refused(self):
        actual = self.receipt.with_name("original.json")
        self.receipt.rename(actual)
        self.receipt.symlink_to(actual.name)
        with self.assertRaisesRegex(ValueError, "regular file"):
            verify_module.verify(self.receipt)


if __name__ == "__main__":
    unittest.main()
