"""Negative cases for E1 Browser + frozen API byte/build preflight."""
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from verify_frozen_unit_pair import preflight


SOURCE = "a" * 40
BASELINE = "b" * 40
TREE = "c" * 40
DERIVED = "sha256:" + "d" * 64
MODEL = "sha256:" + "e" * 64
GO = {
    "goVersion": "go1.26.9",
    "goos": "linux",
    "goarch": "amd64",
    "goamd64": "v1",
    "cgoEnabled": False,
    "trimpath": True,
    "buildTags": sorted(["tei_order_only", "tei_api_only", "tei_frozen_world"]),
}
INFO = (
    "/tmp/order-entry: go1.26.9\n"
    "\tpath\ttei/tools/reference-first-web-target\n"
    "\tmod\ttei\t(devel)\n"
    "\tbuild\t-tags=tei_order_only,tei_api_only,tei_frozen_world\n"
    "\tbuild\t-trimpath=true\n"
    "\tbuild\tCGO_ENABLED=0\n"
    "\tbuild\tGOARCH=amd64\n"
    "\tbuild\tGOOS=linux\n"
    "\tbuild\tGOAMD64=v1\n"
)


def digest(data):
    return hashlib.sha256(data).hexdigest()


class UnitPairPreflightTests(unittest.TestCase):
    def setUp(self):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        self.root = Path(d.name)
        (self.root / "browser").mkdir()
        (self.root / "api").mkdir()
        self.files = {
            "browser/index.html": b'<form id="json-command" method="post" action="/scenario/submit">hello</form>',
            "browser/browser-command.js": b"void 0;",
            "api/order-entry": b"Go binary fixture",
            "api/frozen-world.json": json.dumps({
                "schema": "tei.reference.first-web-frozen-world/v1",
                "applicationId": "order_entry",
                "handoverBaselineId": "B",
                "baselineRepositoryRevision": BASELINE,
                "targetBuildRevision": SOURCE,
                "world": {"applications": []},
            }).encode(),
        }
        for name, data in self.files.items():
            (self.root / name).write_bytes(data)
        self.lineage = {
            "sourceModelRevision": MODEL,
            "derivedWorldSHA256": DERIVED,
            "frozenWorldSHA256": digest(self.files["api/frozen-world.json"]),
            "currentAcceptedDecisionMatched": False,
            "currentAcceptedDecisionRecords": [],
            "sourceAcceptanceAuthorityVerified": False,
            "buildArtifactLockVerified": False,
            "runtimeModelAcceptanceAuthority": False,
            "artifactAuthority": False,
        }
        self.m = {
            "applicationId": "order_entry",
            "handoverBaselineId": "B",
            "baselineRepositoryRevision": BASELINE,
            "browserAndApiSourceRevision": SOURCE,
            "sourceModelRevision": MODEL,
            "sourceWorldDerivedSHA256": DERIVED,
            "browserTransport": "same-origin-json",
            "apiWorldMode": "digest-bound-frozen-world",
            "apiBuildTags": ["tei_order_only", "tei_api_only", "tei_frozen_world"],
            "goVersion": "go1.26.9",
            "currentAcceptedDecisionMatched": False,
            "sourceAcceptanceAuthorityVerified": False,
            "buildArtifactLockVerified": False,
            "runtimeModelAcceptanceAuthority": False,
            "productionAuthorizationVerified": False,
            "artifactAuthority": False,
            "fullE1BrowserAPIPackagingVerified": False,
        }
        for key, digest_key, size_key in (
            ("browser/index.html", "browserSHA256", "browserBytes"),
            ("browser/browser-command.js", "browserScriptSHA256", "browserScriptBytes"),
            ("api/order-entry", "apiSHA256", "apiBytes"),
            ("api/frozen-world.json", "frozenWorldSHA256", "frozenWorldBytes"),
        ):
            self.m[digest_key] = digest(self.files[key])
            self.m[size_key] = len(self.files[key])
        self.m["apiBuildInputObservation"] = {
            "kind": "diagnostic-e1-api-build-input-observation",
            "entrypoint": "tei/tools/reference-first-web-target",
            "sourceCommit": SOURCE,
            "sourceGitTree": TREE,
            "apiSHA256": self.m["apiSHA256"],
            "apiBytes": self.m["apiBytes"],
            "go": GO.copy(),
            "goPackageCount": 227,
            "goPackageClosureSHA256": "f" * 64,
            "builderArtifactAuthorityVerified": False,
            "buildArtifactLockVerified": False,
            "hermeticBuildVerified": False,
            "reproducibleBinaryVerified": False,
        }
        self.save()

    def save(self):
        (self.root / "measurement.json").write_text(json.dumps(self.m), encoding="utf-8")
        (self.root / "source-lineage.json").write_text(json.dumps(self.lineage), encoding="utf-8")

    def check(self):
        return preflight(self.root, build_info=INFO, world_digest=DERIVED)

    def test_exact_pair_passes_without_claiming_authority(self):
        result = self.check()
        self.assertTrue(result["byteAndBuildConsistencyVerified"])
        self.assertEqual(result["sourceRevision"], SOURCE)
        self.assertEqual(result["packageCount"], 227)
        self.assertFalse(result["buildArtifactLockVerified"])
        self.assertFalse(result["productionAuthorizationVerified"])

    def test_any_modified_unit_byte_rejected(self):
        for name in self.files:
            with self.subTest(file=name):
                original = self.files[name]
                (self.root / name).write_bytes(original + b"modified")
                with self.assertRaises(ValueError):
                    self.check()
                (self.root / name).write_bytes(original)

    def test_recomputed_frozen_world_rejects_re_stamped_semantic_change(self):
        # The full check uses the Go typed-IR reserializer, not a hash
        # recomputed from an attacker-controlled observation JSON.
        snapshot = json.loads((self.root / "api/frozen-world.json").read_text())
        snapshot["world"]["applications"].append({"id": "forged_extra"})
        payload = json.dumps(snapshot).encode()
        (self.root / "api/frozen-world.json").write_bytes(payload)
        self.m["frozenWorldSHA256"] = digest(payload)
        self.m["frozenWorldBytes"] = len(payload)
        self.lineage["frozenWorldSHA256"] = digest(payload)
        self.save()
        with self.assertRaisesRegex(ValueError, "frozen payload versus accepted derived World"):
            preflight(self.root, build_info=INFO, world_digest="sha256:" + "0" * 64)

    def test_go_reserialization_matches_actual_frozen_world_fixture(self):
        # Run the same tool as the staged publication, using a minimal
        # correctly ordered JSON representation of the actual IR World.
        from verify_frozen_unit_pair import observed_typed_world_digest
        snapshot = json.loads((self.root / "api/frozen-world.json").read_text())
        canonical = (json.dumps(snapshot["world"], indent=2, ensure_ascii=True) + "\n").encode()
        expected = "sha256:" + digest(canonical)
        observed = observed_typed_world_digest(self.root / "api/frozen-world.json")
        self.assertEqual(observed, expected)

    def test_mixed_api_from_another_build_rejected(self):
        (self.root / "api/order-entry").write_bytes(b"other API built from different Source")
        with self.assertRaisesRegex(ValueError, "api/order-entry SHA-256"):
            self.check()

    def test_frozen_binding_mismatch_fails_even_if_digests_recomputed(self):
        snapshot = json.loads((self.root / "api/frozen-world.json").read_text())
        snapshot["targetBuildRevision"] = BASELINE
        changed = json.dumps(snapshot).encode()
        (self.root / "api/frozen-world.json").write_bytes(changed)
        self.m["frozenWorldSHA256"] = digest(changed)
        self.m["frozenWorldBytes"] = len(changed)
        self.lineage["frozenWorldSHA256"] = digest(changed)
        self.save()
        with self.assertRaisesRegex(ValueError, "targetBuildRevision"):
            self.check()

    def test_source_lineage_mismatch_rejected(self):
        self.lineage["derivedWorldSHA256"] = "sha256:" + "0" * 64
        self.save()
        with self.assertRaisesRegex(ValueError, "derived World"):
            self.check()

    def test_forged_builder_identity_or_tag_rejected(self):
        self.m["apiBuildInputObservation"]["sourceCommit"] = BASELINE
        self.save()
        with self.assertRaisesRegex(ValueError, "Go build Source commit"):
            self.check()
        self.m["apiBuildInputObservation"]["sourceCommit"] = SOURCE
        self.m["apiBuildInputObservation"]["go"]["goamd64"] = "v3"
        self.save()
        with self.assertRaisesRegex(ValueError, "actual embedded Go build options"):
            self.check()

    def test_true_permission_or_build_authority_claim_fails(self):
        for section, key in (
            (self.m, "artifactAuthority"),
            (self.m, "sourceAcceptanceAuthorityVerified"),
            (self.m["apiBuildInputObservation"], "buildArtifactLockVerified"),
            (self.lineage, "artifactAuthority"),
        ):
            section[key] = True
            self.save()
            with self.subTest(key=key), self.assertRaisesRegex(ValueError, "authority was asserted"):
                self.check()
            section[key] = False
        self.save()

    def test_unknown_file_source_map_and_missing_unit_rejected(self):
        extra = self.root / "browser/browser-command.js.map"
        extra.write_text("source-map and secret")
        with self.assertRaisesRegex(ValueError, "packaged unit member"):
            self.check()
        extra.unlink()
        (self.root / "browser/browser-command.js").unlink()
        with self.assertRaisesRegex(ValueError, "packaged unit member"):
            self.check()

    def test_symlink_to_neighbor_rejected(self):
        target = self.root / "browser/index.html"
        target.unlink()
        target.symlink_to("browser-command.js")
        with self.assertRaisesRegex(ValueError, "symlink"):
            self.check()

    def test_duplicate_json_keys_and_wrong_model_rejected(self):
        meta = self.root / "measurement.json"
        meta.write_text('{"applicationId":"order_entry","applicationId":"repair_intake"}')
        with self.assertRaisesRegex(ValueError, "duplicate JSON key"):
            self.check()
        self.save()
        self.m["sourceModelRevision"] = "not-a-sha"
        self.save()
        with self.assertRaisesRegex(ValueError, "sourceModelRevision"):
            self.check()

    def test_compatible_optional_smoke_evidence(self):
        (self.root / "json-api-smoke.json").write_text('{"synthetic":true}')
        self.assertTrue(self.check()["byteAndBuildConsistencyVerified"])


if __name__ == "__main__":
    unittest.main()
