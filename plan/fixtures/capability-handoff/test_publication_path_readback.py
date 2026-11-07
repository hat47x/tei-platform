"""AP-12/13 and the bounded ledger path-to-member evidence boundary."""
import collections
from pathlib import Path
import tempfile
import unittest

from publication_path_readback import verify_ledger_publication_paths


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

    def test_modified_member_byte_rejected(self):
        (self.root / PATHS[0]).write_bytes(b"tampered")
        with self.assertRaisesRegex(AssertionError, "path-to-byte mismatch"):
            verify_ledger_publication_paths(MEMBERS, self.root)


if __name__ == "__main__":
    unittest.main()
