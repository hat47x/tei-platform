"""AP-12/13 and the bounded ledger path-to-member evidence boundary."""
import collections
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from publication_path_readback import (verify_ledger_publication_paths,
                                       verify_pinned_ledger_publication_paths)


MEMBERS = [
    {"kind": "table-definition", "content": "table public.notes\nnotes\n"},
    {"kind": "table-definition", "content": "table public.orders\norders\n"},
    {"kind": "table-definition", "content": "table public.order_archive\narchive\n"},
    {"kind": "function-definition", "content": "application ledger\nfunctions\n"},
    {"kind": "operation-surface", "content": '{"operations":[]}\n'},
]
PATHS = ["table-public-notes.md", "table-public-orders.md",
         "table-public-order_archive.md", "function-ledger.md", "operation-surface.json"]


class PublicationPathReadbackTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for member, path in zip(MEMBERS, PATHS):
            (self.root / path).write_bytes(member["content"].encode("utf-8"))
        (self.root / "manifest.json").write_bytes(b'{}\n')

    def test_exact_paths_and_bytes_qualify_only_fixture_binding(self):
        r = verify_ledger_publication_paths(MEMBERS, self.root)
        self.assertEqual(r["file_count"], 5)
        self.assertEqual(r["member_binding"], "ledger-fixture-path-to-bytes-only")
        self.assertEqual(set(r["member_body_sha256_by_path"]), set(PATHS))

    def test_swapped_body_paths_rejected_despite_equal_body_multiset(self):
        a, b = self.root / PATHS[0], self.root / PATHS[1]
        x, y = a.read_bytes(), b.read_bytes()
        a.write_bytes(y)
        b.write_bytes(x)
        self.assertEqual(collections.Counter(p.read_bytes() for p in self.root.iterdir()
                                              if p.name != "manifest.json"),
                         collections.Counter(m["content"].encode("utf-8") for m in MEMBERS))
        with self.assertRaisesRegex(AssertionError, "path-to-byte mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)

    def test_renamed_member_rejected_even_with_same_body(self):
        (self.root / PATHS[0]).rename(self.root / "table-misnamed.md")
        with self.assertRaisesRegex(AssertionError, "path set mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)

    def test_extra_unexpected_member_rejected(self):
        (self.root / "table-stale.md").write_bytes(b"stale")
        with self.assertRaisesRegex(AssertionError, "path set mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)

    def test_unexpected_subdir_rejected(self):
        (self.root / "hidden").mkdir()
        with self.assertRaisesRegex(AssertionError, "directory or symlink"):
            verify_ledger_publication_paths(MEMBERS, self.root)

    def test_member_role_change_rejected_even_if_bytes_match(self):
        changed = [dict(m) for m in MEMBERS]
        changed[0]["kind"] = "function-definition"
        with self.assertRaisesRegex(AssertionError, "unsupported ledger member role"):
            verify_ledger_publication_paths(changed, self.root)

    def test_missing_logical_member_rejected(self):
        with self.assertRaisesRegex(AssertionError, "required-member set is incomplete"):
            verify_ledger_publication_paths(MEMBERS[:-1], self.root)

    def test_duplicate_logical_member_rejected(self):
        with self.assertRaisesRegex(AssertionError, "duplicate ledger member"):
            verify_ledger_publication_paths(MEMBERS + [dict(MEMBERS[0])], self.root)

    def test_same_body_with_other_envelope_does_not_prove_artifact_identity(self):
        # AP-10/11: changing the rule/generator/gap details without changing
        # Content must NOT turn a successful body read into an envelope claim.
        reference = verify_ledger_publication_paths(MEMBERS, self.root)
        changed = [dict(m) for m in MEMBERS]
        changed[0]["rule"] = {"id": "different-rule", "version": "R2"}
        changed[0]["generator"] = "different-producer"
        changed[0]["unresolved"] = [{"subject": "x", "reason": "different"}]
        observed = verify_ledger_publication_paths(changed, self.root)
        self.assertEqual(observed, reference)
        self.assertEqual(observed["member_binding"], "ledger-fixture-path-to-bytes-only")
        self.assertNotIn("artifact_digest_verified", observed)

    def test_partial_r2_member_under_r1_manifest_is_rejected_as_r1_read(self):
        # AP-01/AP-06: a new member from R2 alongside old R1 files/manifest.
        # This proves only detection by the bounded Reader, NOT atomic publish.
        old_manifest = (self.root / "manifest.json").read_bytes()
        (self.root / PATHS[0]).write_bytes(b"table public.notes\nR2 only\n")
        self.assertEqual((self.root / "manifest.json").read_bytes(), old_manifest)
        with self.assertRaisesRegex(AssertionError, "path-to-byte mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)

    def test_pinned_manifest_same_bytes_qualifies_only_body_binding(self):
        pinned = (self.root / "manifest.json").read_bytes()
        evidence = verify_pinned_ledger_publication_paths(MEMBERS, self.root, pinned)
        self.assertEqual(evidence["file_count"], len(PATHS))
        self.assertNotIn("atomic_publication", evidence)

    def test_pinned_r1_rejected_on_r2_manifest_even_if_bodies_unchanged(self):
        pinned = (self.root / "manifest.json").read_bytes()
        self.assertEqual(verify_ledger_publication_paths(MEMBERS, self.root)["file_count"], 5)
        (self.root / "manifest.json").write_bytes(b'{"generation":"R2"}\n')
        with self.assertRaisesRegex(AssertionError, "changed before member read"):
            verify_pinned_ledger_publication_paths(MEMBERS, self.root, pinned)
        # The same physical bodies are insufficient to qualify the R1 pin.
        self.assertEqual(verify_ledger_publication_paths(MEMBERS, self.root)["file_count"], 5)

    def test_pinned_manifest_changed_during_member_read(self):
        pinned = (self.root / "manifest.json").read_bytes()
        original_read = Path.read_bytes
        path_first = self.root / "function-ledger.md"
        changed = [False]

        def intercept_read(path):
            data = original_read(path)
            if path == path_first and not changed[0]:
                changed[0] = True
                (self.root / "manifest.json").write_bytes(b'{"generation":"R2"}\n')
            return data

        with mock.patch.object(Path, "read_bytes", autospec=True,
                               side_effect=intercept_read):
            with self.assertRaisesRegex(AssertionError, "changed during member read"):
                verify_pinned_ledger_publication_paths(MEMBERS, self.root, pinned)
        self.assertTrue(changed[0])

    def test_pinned_manifest_requires_explicit_bytes(self):
        with self.assertRaisesRegex(AssertionError, "bytes are required"):
            verify_pinned_ledger_publication_paths(MEMBERS, self.root, None)

    def test_manifest_aba_returns_only_byte_witness_not_continuous_generation(self):
        # Characterization: read R1, briefly expose R2, restore R1 before
        # the final manifest read. The two-point test cannot observe ABA.
        # This is NOT a successful claim of continuous R1 generation pinning.
        manifest = self.root / "manifest.json"
        pinned = manifest.read_bytes()
        first_member = self.root / "function-ledger.md"
        original_read = Path.read_bytes
        saw_r2 = []

        def intercept_read(path):
            data = original_read(path)
            if path == first_member and not saw_r2:
                manifest.write_bytes(b'{"generation":"R2"}\n')
                saw_r2.append(original_read(manifest))
                manifest.write_bytes(pinned)
            return data

        with mock.patch.object(Path, "read_bytes", autospec=True,
                               side_effect=intercept_read):
            evidence = verify_pinned_ledger_publication_paths(MEMBERS, self.root, pinned)
        self.assertEqual(saw_r2, [b'{"generation":"R2"}\n'])
        self.assertEqual(manifest.read_bytes(), pinned)
        self.assertEqual(evidence["file_count"], 5)
        self.assertEqual(evidence["member_binding"], "ledger-fixture-path-to-bytes-only")
        self.assertNotIn("continuous_generation_pin", evidence)
        self.assertNotIn("atomic_publication", evidence)

    def test_symlink_swap_after_directory_enumeration_is_not_identity_proof(self):
        # This bounded Reader checks a directory entry before it opens the
        # member. A subsequent physical path replacement can escape the dir
        # while still returning expected bytes. This characterizes a TOCTOU
        # gap; it does not invoke the real Publisher or access private data.
        pinned = (self.root / "manifest.json").read_bytes()
        first_member = self.root / "function-ledger.md"
        changed_member = self.root / "table-public-orders.md"
        original_read = Path.read_bytes
        performed = []

        with tempfile.TemporaryDirectory(prefix="tei-test-external-member-") as external:
            external_body = Path(external) / "public-test-member.md"
            external_body.write_bytes(MEMBERS[1]["content"].encode("utf-8"))

            def intercept_read(path):
                data = original_read(path)
                if path == first_member and not performed:
                    changed_member.unlink()
                    try:
                        changed_member.symlink_to(external_body)
                    except (OSError, NotImplementedError) as exc:
                        self.skipTest(f"symlink creation unavailable: {exc}")
                    performed.append(True)
                return data

            with mock.patch.object(Path, "read_bytes", autospec=True,
                                   side_effect=intercept_read):
                evidence = verify_pinned_ledger_publication_paths(
                    MEMBERS, self.root, pinned)
            self.assertEqual(performed, [True])
            self.assertTrue(changed_member.is_symlink())
            self.assertEqual(evidence["file_count"], 5)
            self.assertNotIn("path_object_identity_verified", evidence)
            self.assertNotIn("scope_stability_verified", evidence)

    def test_modified_member_byte_rejected(self):
        (self.root / PATHS[0]).write_bytes(b"tampered")
        with self.assertRaisesRegex(AssertionError, "path-to-byte mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)


if __name__ == "__main__":
    unittest.main()
