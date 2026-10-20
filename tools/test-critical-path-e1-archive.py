#!/usr/bin/env python3
"""Tampering regression tests for the retained synthetic E1 witness set."""
import importlib.util
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location(
    "critical_path_e1_archive", ROOT / "tools/verify-critical-path-e1-archive.py")
archive = importlib.util.module_from_spec(spec)
spec.loader.exec_module(archive)


class ArchivedEvidenceIntegrity(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="tei-e1-archive-")
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.manifest = root / "manifest.json"
        self.receipts = root / "receipts"
        self.receipts.mkdir()
        shutil.copyfile(archive.MANIFEST, self.manifest)
        for name in archive.NAMES:
            shutil.copyfile(archive.RECEIPTS / name, self.receipts / name)

    def verify(self):
        return archive.verify_archive(self.manifest, self.receipts)

    def test_original_eight_receipts(self):
        result = self.verify()
        self.assertEqual(8, result["receiptCount"])
        self.assertEqual("UNQUALIFIED", result["productGate"])

    def test_autocrlf_checkout_preserves_original_witness_bytes(self):
        # Exercise Git's checkout conversion, not a normalized verifier input.
        root = Path(self.temp.name)
        source, checkout = root / "source", root / "checkout"
        source.mkdir()
        paths = [archive.MANIFEST, *[archive.RECEIPTS / name for name in archive.NAMES],
                 ROOT / "plan/evidence/TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json"]
        for path in [ROOT / ".gitattributes", *paths]:
            target = source / path.relative_to(ROOT)
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(path.read_bytes())

        def git(*args, cwd=source):
            subprocess.run(["git", *args], cwd=cwd, check=True,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        git("init")
        git("config", "core.autocrlf", "true")
        git("add", ".")
        git("-c", "user.name=Archive test", "-c", "user.email=archive-test@example.invalid",
            "-c", "commit.gpgsign=false", "commit", "-m", "Archive checkout fixture")
        git("clone", "--no-local", "--no-checkout", str(source), str(checkout))
        git("config", "core.autocrlf", "true", cwd=checkout)
        git("checkout", "--force", "HEAD", cwd=checkout)
        for path in paths:
            self.assertEqual(path.read_bytes(), (checkout / path.relative_to(ROOT)).read_bytes(),
                             str(path.relative_to(ROOT)))
        result = archive.verify_archive(checkout / archive.MANIFEST.relative_to(ROOT),
                                        checkout / archive.RECEIPTS.relative_to(ROOT))
        self.assertEqual("UNQUALIFIED", result["productGate"])

    def test_mutated_raw_receipt_refused(self):
        path = self.receipts / "web-positive.json"
        path.write_bytes(path.read_bytes() + b"\n")
        with self.assertRaisesRegex(ValueError, "evidence bytes"):
            self.verify()

    def test_deleted_receipt_refused(self):
        (self.receipts / "stale-source-control.json").unlink()
        with self.assertRaisesRegex(ValueError, "missing/extra"):
            self.verify()

    def test_extra_unlisted_receipt_refused(self):
        (self.receipts / "new-e2.json").write_text('{"productGate":"PASS"}')
        with self.assertRaisesRegex(ValueError, "missing/extra"):
            self.verify()

    def test_manifest_qualification_forgery_refused(self):
        text = self.manifest.read_text()
        self.manifest.write_text(text.replace('"UNQUALIFIED"', '"PASS"'))
        with self.assertRaisesRegex(ValueError, "manifest digest mismatch"):
            self.verify()

    def test_manifest_digest_replacement_refused(self):
        text = self.manifest.read_text()
        self.manifest.write_text(text.replace('"runtimeVersionBindingVerified": false',
                                              '"runtimeVersionBindingVerified": true'))
        with self.assertRaisesRegex(ValueError, "manifest digest mismatch"):
            self.verify()

    def test_symlink_instead_of_receipt_refused(self):
        path = self.receipts / "web-positive.json"
        target = self.receipts / "web-negative-unit-price.json"
        path.unlink()
        try:
            path.symlink_to(target.name)
        except OSError as exc:
            if getattr(exc, "winerror", None) == 1314:
                self.skipTest("Windows symlink privilege is unavailable")
            raise
        with self.assertRaisesRegex(ValueError, "not a regular file"):
            self.verify()


if __name__ == "__main__":
    unittest.main()
