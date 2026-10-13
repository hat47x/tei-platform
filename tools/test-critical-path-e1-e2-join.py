#!/usr/bin/env python3
"""Protect the G2 evidence join against evidence substitution and inflation."""
import copy
import importlib.util
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location(
    "historical_join", ROOT / "tools/verify-critical-path-e1-e2-join.py")
join = importlib.util.module_from_spec(spec)
spec.loader.exec_module(join)


class HistoricalG2JoinTests(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory(prefix="tei-e1-e2-audit-")
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        self.manifest = self.root / "e1-manifest.json"
        self.receipts = self.root / "e1-receipts"
        self.receipts.mkdir()
        self.e2 = self.root / "e2-receipt.json"
        shutil.copyfile(join.E1.MANIFEST, self.manifest)
        for name in join.E1.NAMES:
            shutil.copyfile(join.E1.RECEIPTS / name, self.receipts / name)
        shutil.copyfile(join.E2.RECEIPT, self.e2)

    def verify(self):
        return join.verify_join(self.manifest, self.receipts, self.e2)

    def test_real_archives_cannot_be_joined_into_product_g2(self):
        report = self.verify()
        self.assertEqual("UNJOINABLE_HISTORICAL_EVIDENCE", report["status"])
        self.assertEqual("UNQUALIFIED", report["productGate"])
        self.assertFalse(report["canSatisfyG2"])
        self.assertEqual(list(join.OBLIGATIONS), report["blockedObligations"])
        self.assertTrue(report["crossEvidence"]["differentApplication"])
        self.assertTrue(report["crossEvidence"]["distinctGitCheckout"])

    def test_g2_requirement_exits_nonzero(self):
        run = subprocess.run([sys.executable, "-B",
            str(ROOT / "tools/verify-critical-path-e1-e2-join.py"),
            "--require-g2"], text=True, capture_output=True)
        self.assertEqual(3, run.returncode, run.stderr)
        self.assertIn('"productGate": "UNQUALIFIED"', run.stdout)

    def test_byte_change_in_e1_witness_rejected_before_join(self):
        path = self.receipts / "target-launch.json"
        path.write_bytes(path.read_bytes() + b"\n")
        with self.assertRaisesRegex(ValueError, "evidence bytes"):
            self.verify()

    def test_byte_change_in_e2_receipt_rejected_before_join(self):
        self.e2.write_bytes(self.e2.read_bytes() + b"\n")
        with self.assertRaisesRegex(ValueError, "byte SHA256 mismatch"):
            self.verify()

    def test_e1_manifest_forgery_rejected_before_join(self):
        self.manifest.write_bytes(self.manifest.read_bytes() + b"\n")
        with self.assertRaisesRegex(ValueError, "manifest digest mismatch"):
            self.verify()

    def test_claimed_shared_source_without_verified_runtime_stays_blocked(self):
        manifest = join.E1.decode_unique(self.manifest.read_bytes(), "e1")
        launch = join.E1.decode_unique((self.receipts/"target-launch.json").read_bytes(), "launch")
        e2 = join.E2.decode_unique(self.e2.read_bytes())
        e2["acceptedSourceRevision"] = manifest["sourceAfter"]
        report = join.audit_pair(manifest, launch, e2)
        self.assertIn("SAME_ACCEPTED_SOURCE_REVISION", report["blockedObligations"])
        self.assertFalse(report["canSatisfyG2"])

    def test_even_all_forged_checks_cannot_be_product_qualified(self):
        manifest = join.E1.decode_unique(self.manifest.read_bytes(), "e1")
        launch = join.E1.decode_unique((self.receipts/"target-launch.json").read_bytes(), "launch")
        e2 = join.E2.decode_unique(self.e2.read_bytes())
        manifest.update(provider="postgres", runtimeVersionBindingVerified=True,
                        staleSourceRejectedByTarget=True,
                        semanticEquivalentR2RejectedByTarget=True,
                        humanAcceptance=True)
        e2["acceptedSourceRevision"] = manifest["sourceAfter"]
        e2["acceptedSourceVerification"] = "runtime-verified"
        e2["application"] = "order_entry"
        e2["semanticData"] = "sales.order"
        e2["webCommandLinkage"] = "verified"
        launch["commandId"] = e2["positive"]["commandId"]
        e2["authorizationAtCommit"] = True
        e2["humanAcceptedSourceAuthority"] = True
        e2["scope"].update(authority="runtime-host-verified",
            commandToPersistenceLinkage="verified",
            independenceKind="separate-invocation",
            generatedRoundTrip=True, handWrittenOverlay=False,
            handWrittenInsertAndQuerySQL=False)
        report = join.audit_pair(manifest, launch, e2)
        self.assertEqual([], report["blockedObligations"])
        self.assertEqual("UNQUALIFIED", report["productGate"])
        self.assertFalse(report["canSatisfyG2"])

    def test_in_memory_evidence_can_never_bypass_raw_hash_validation(self):
        # The public verifier accepts only immutable paths, not arbitrary maps.
        # The map-only audit is deliberately non-authoritative.
        with self.assertRaises(TypeError):
            join.verify_join(e2_receipt_path={"productGate": "PASS"})


if __name__ == "__main__":
    unittest.main()
