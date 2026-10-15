#!/usr/bin/env python3
"""Offline regressions for #1445 E1 archive reconciliation; no GitHub token."""
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import types
import unittest
import zipfile


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = Path(__file__).with_name("e1_archive_reconcile.py")
spec = importlib.util.spec_from_file_location("tei_e1_archive_reconcile", SCRIPT)
bridge = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bridge)
canonical = bridge.canonical_archive()


class E1ArchiveBridgeTest(unittest.TestCase):
    def test_original_repository_archive_uses_existing_immutable_verifier(self):
        result = bridge.inspect()
        self.assertEqual(result["archive"]["receiptCount"], 8)
        self.assertEqual(result["archive"]["workflowRunID"], "38267656696")
        self.assertEqual(result["archive"]["checkout"], canonical.INTEGRATION_HEAD)
        self.assertEqual(result["artifactZip"]["status"], "not-supplied")
        self.assertEqual(result["sourceSnapshots"]["status"], "not-supplied")
        self.assertEqual(result["qualification"], "unresolved")
        self.assertIs(result["executionAllowed"], False)
        self.assertEqual(result["productGate"], "UNQUALIFIED")

    def make_zip(self, directory, *, missing=None, extra=None, mutate=None):
        path = Path(directory) / "archive.zip"
        members = {n: (canonical.RECEIPTS / n).read_bytes() for n in canonical.NAMES}
        members["evidence-manifest.json"] = canonical.MANIFEST.read_bytes()
        if missing:
            del members[missing]
        if extra:
            members[extra] = b"forged"
        if mutate:
            members[mutate] += b"\n"
        with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED) as z:
            for name, body in sorted(members.items()):
                z.writestr(name, body)
        return path, "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()

    def test_supplied_zip_matches_preserved_archive(self):
        with tempfile.TemporaryDirectory() as tmp:
            path, digest = self.make_zip(tmp)
            report = bridge.inspect(path, digest)
            self.assertEqual(report["artifactZip"]["status"], "bytes-match")
            self.assertEqual(report["artifactZip"]["sha256"], digest)
            self.assertEqual(report["sourceSnapshots"]["status"], "not-supplied")
            self.assertEqual(report["productGate"], "UNQUALIFIED")

    def test_zip_requires_independently_supplied_digest(self):
        with tempfile.TemporaryDirectory() as tmp:
            path, digest = self.make_zip(tmp)
            for args in [(path, None), (None, digest), (path, "sha256:wrong")]:
                with self.subTest(args=args), self.assertRaises(ValueError):
                    bridge.inspect(*args)

    def test_matching_self_supplied_zip_digest_cannot_hide_tampering(self):
        with tempfile.TemporaryDirectory() as tmp:
            path, digest = self.make_zip(tmp, mutate="web-positive.json")
            with self.assertRaisesRegex(ValueError, "differs from the archived original"):
                bridge.inspect(path, digest)

    def test_missing_extra_or_path_traversal_zip_is_rejected(self):
        for name in ["missing", "extra", "traversal"]:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as tmp:
                kwargs = {
                    "missing": {"missing": "stale-source-control.json"},
                    "extra": {"extra": "unclaimed.json"},
                    "traversal": {"extra": "../escape"},
                }[name]
                path, digest = self.make_zip(tmp, **kwargs)
                with self.assertRaisesRegex(ValueError, "missing, duplicated or unexpected"):
                    bridge.inspect(path, digest)

    def test_zip_digest_mismatch_is_rejected_before_contents(self):
        with tempfile.TemporaryDirectory() as tmp:
            path, digest = self.make_zip(tmp)
            with zipfile.ZipFile(path, "a") as z:
                z.writestr("extra.json", b"forged")
            with self.assertRaisesRegex(ValueError, "metadata digest"):
                bridge.inspect(path, digest)

    def mock_source_runner(self, before, after, head, *, status="declared-links-consistent",
                           qualification="unresolved", allowed=False):
        def run(argv, **kwargs):
            self.assertEqual(argv[0:3], ["go", "run", "./tools/reference-change-assurance"])
            self.assertEqual(kwargs["cwd"], ROOT)
            self.assertEqual(kwargs["check"], True)
            self.assertIn("-archived-witness-dir", argv)
            return types.SimpleNamespace(stdout=json.dumps({
                "integrationHead": head,
                "linkStatus": status,
                "qualification": qualification,
                "executionAllowed": allowed,
                "diff": {"beforeSourceRevision": before, "afterSourceRevision": after},
            }))
        return run

    def test_snapshot_review_only_matches_declared_versions_without_granting_gate(self):
        manifest = canonical.decode_unique(canonical.MANIFEST.read_bytes(), "manifest")
        before, after, head = (manifest["sourceBefore"],
                               manifest["sourceAfter"], manifest["exactCheckoutHead"])
        ok = bridge.inspect(before="/historical/r0", after="/historical/r1",
                            runner=self.mock_source_runner(before, after, head))
        self.assertEqual(ok["sourceSnapshots"]["status"],
                         "supplied-snapshot-revisions-match-archived-claims-only")
        self.assertIs(ok["sourceSnapshots"]["independentlyAuthenticated"], False)
        self.assertIs(ok["executionAllowed"], False)
        self.assertEqual(ok["qualification"], "unresolved")
        self.assertEqual(ok["productGate"], "UNQUALIFIED")

        for runner in [
            self.mock_source_runner(after, before, head),
            self.mock_source_runner(before, after, "b" * 40),
            self.mock_source_runner(before, after, head, status="unresolved"),
            self.mock_source_runner(before, after, head, qualification="qualified"),
            self.mock_source_runner(before, after, head, allowed=True),
        ]:
            with self.subTest(runner=runner), self.assertRaisesRegex(
                    ValueError, "do not match"):
                bridge.inspect(before="/historical/r0", after="/historical/r1",
                               runner=runner)

    def test_source_pair_is_required(self):
        with self.assertRaisesRegex(ValueError, "together"):
            bridge.inspect(before="/historical/r0")
        with self.assertRaisesRegex(ValueError, "together"):
            bridge.inspect(after="/historical/r1")


if __name__ == "__main__":
    unittest.main()
