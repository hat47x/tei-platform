"""Negative tests for the diagnostic E1 Go build-input observation."""
import importlib.util
from pathlib import Path
import tempfile
import unittest


NAME = "inspect_frozen_api_build"
SOURCE = Path(__file__).with_name(NAME + ".py")
spec = importlib.util.spec_from_file_location(NAME, SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
COMMIT = "a" * 40
TREE = "b" * 40
BASE_PACKAGES = (
    "encoding/json\n"
    "tei/internal/ir\n"
    "tei/internal/driver\n"
    "tei/tools/reference-first-web-target\n"
)
VERSION = (
    "/tmp/order-entry: go1.23.2\n"
    "\tpath\ttei/tools/reference-first-web-target\n"
    "\tmod\ttei\t(devel)\n"
    "\tbuild\t-buildmode=exe\n"
    "\tbuild\t-tags=tei_order_only,tei_api_only,tei_frozen_world\n"
    "\tbuild\t-compiler=gc\n"
    "\tbuild\t-trimpath=true\n"
    "\tbuild\tCGO_ENABLED=0\n"
    "\tbuild\tGOARCH=amd64\n"
    "\tbuild\tGOOS=linux\n"
    "\tbuild\tGOAMD64=v1\n"
)


class GoBuildObservationTests(unittest.TestCase):
    def setUp(self):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        self.root = Path(d.name)
        self.binary = self.root / "order-entry"
        self.binary.write_bytes(b"diagnostic-binary-fixture")
        self.packages = self.root / "closure.txt"
        self.packages.write_text(BASE_PACKAGES, encoding="utf-8")

    def get(self, output=VERSION):
        return module.observe(self.binary, self.packages, TREE, COMMIT, output)

    def test_records_built_binary_and_source_inputs_without_authority(self):
        result = self.get()
        self.assertEqual(result["goPackageCount"], 4)
        self.assertEqual(result["go"]["buildTags"], sorted(module.EXPECTED_TAGS))
        self.assertEqual(result["sourceGitTree"], TREE)
        self.assertEqual(result["sourceCommit"], COMMIT)
        self.assertEqual(result["apiBytes"], len(self.binary.read_bytes()))
        self.assertFalse(result["builderArtifactAuthorityVerified"])
        self.assertFalse(result["buildArtifactLockVerified"])
        self.assertFalse(result["reproducibleBinaryVerified"])

    def test_missing_or_added_tag_rejected(self):
        with self.assertRaisesRegex(ValueError, "tags"):
            self.get(VERSION.replace(",tei_frozen_world", ""))
        with self.assertRaisesRegex(ValueError, "tags"):
            self.get(VERSION.replace("tei_frozen_world", "tei_repair_only"))

    def test_unsafe_architecture_or_cgo_rejected(self):
        for old, new in (
            ("GOOS=linux", "GOOS=darwin"),
            ("GOARCH=amd64", "GOARCH=arm64"),
            ("GOAMD64=v1", "GOAMD64=v3"),
            ("CGO_ENABLED=0", "CGO_ENABLED=1"),
            ("-trimpath=true", "-trimpath=false"),
        ):
            with self.subTest(setting=old), self.assertRaisesRegex(ValueError, "setting mismatch"):
                self.get(VERSION.replace(old, new))

    def test_duplicate_or_missing_build_info_rejected(self):
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.get(VERSION + "\tbuild\tGOOS=linux\n")
        with self.assertRaisesRegex(ValueError, "setting mismatch"):
            self.get(VERSION.replace("\tbuild\tGOOS=linux\n", ""))
        with self.assertRaisesRegex(ValueError, "version"):
            self.get("not a Go executable")

    def test_source_parser_cannot_appear_in_runtime_dependency_closure(self):
        for forbidden in ("tei/internal/workspace", "tei/internal/syntax"):
            self.packages.write_text(forbidden + "\n" + BASE_PACKAGES, encoding="utf-8")
            with self.subTest(package=forbidden), self.assertRaisesRegex(ValueError, "build-only"):
                self.get()

    def test_duplicate_or_missing_runtime_package_rejected(self):
        self.packages.write_text(BASE_PACKAGES + "tei/internal/ir\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.get()
        self.packages.write_text(BASE_PACKAGES.replace("tei/internal/driver\n", ""), encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "missing"):
            self.get()
        self.packages.write_text(BASE_PACKAGES.replace("tei/tools/reference-first-web-target\n", ""), encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "incomplete"):
            self.get()

    def test_rejects_dirty_closure_and_invalid_git_identity(self):
        self.packages.write_bytes(BASE_PACKAGES.encode() + b"\n")
        with self.assertRaisesRegex(ValueError, "canonical"):
            self.get()
        self.packages.write_text(BASE_PACKAGES, encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Git"):
            module.observe(self.binary, self.packages, TREE, "../rev", VERSION)

    def test_rejects_missing_or_symlinked_executable(self):
        self.binary.unlink()
        with self.assertRaisesRegex(ValueError, "real file"):
            self.get()
        self.binary.symlink_to(self.packages)
        with self.assertRaisesRegex(ValueError, "real file"):
            self.get()


if __name__ == "__main__":
    unittest.main()
