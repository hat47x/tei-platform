"""Local negative cases for the read-only E1 Acceptance lineage observation."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("observe_frozen_source_lineage.py")
spec = importlib.util.spec_from_file_location("frozen_lineage", SCRIPT)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)
SHA = "sha256:" + "a" * 64
REV = "sha256:" + "b" * 64
DECISION_PATH = ".tei/decisions/" + "c" * 64 + ".json"


class FrozenLineageTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.frozen = self.root / "frozen.json"
        self.metadata = self.root / "metadata.json"
        self.index = self.root / "index.json"
        self.frozen_payload = {
            "schema": "tei.reference.first-web-frozen-world/v1",
            "applicationId": "order_entry",
            "world": {"applications": [], "databases": [], "environments": [], "queries": []},
        }
        self.observation = {"derivedWorldSha256": SHA}
        self.decisions = {
            "schema": "tei.reference.decision-index/v0",
            "modelRevision": REV,
            "records": [],
        }
        self.save()

    def save(self):
        data = json.dumps(self.frozen_payload).encode()
        self.frozen.write_bytes(data)
        self.observation["frozenWorldSha256"] = hashlib.sha256(data).hexdigest()
        self.metadata.write_text(json.dumps(self.observation), encoding="utf-8")
        self.index.write_text(json.dumps(self.decisions), encoding="utf-8")

    def observe(self):
        return mod.observe(self.frozen, self.metadata, self.index)

    def add_accepted(self, status="current", derived=SHA):
        self.decisions["records"].append({
            "path": DECISION_PATH,
            "status": status,
            "sourceEdits": "intact",
            "evidenceStatus": "current",
            "reviewBindingStatus": "verified",
            "decisionAuthorityStatus": "unresolved",
            "decision": {
                "schema": "tei.reference.change-decision/v0",
                "decision": "accepted",
                "sourceChanged": True,
                "modelRevisionAfter": REV,
                "derivedWorldSha256": derived,
                "origin": {"actor": "test-fixture", "kind": "cli"},
            }
        })
        self.save()

    def test_no_current_decision_keeps_authority_unproven(self):
        observed = self.observe()
        self.assertFalse(observed["currentAcceptedDecisionMatched"])
        self.assertFalse(observed["sourceAcceptanceAuthorityVerified"])
        self.assertFalse(observed["buildArtifactLockVerified"])

    def test_matching_current_decision_is_read_only_evidence_not_authority(self):
        self.add_accepted()
        observed = self.observe()
        self.assertEqual(observed["currentAcceptedDecisionRecords"], [DECISION_PATH])
        self.assertTrue(observed["currentAcceptedDecisionMatched"])
        self.assertFalse(observed["sourceAcceptanceAuthorityVerified"])
        self.assertFalse(observed["artifactAuthority"])

    def test_superseded_acceptance_does_not_claim_current_meaning(self):
        self.add_accepted(status="superseded")
        self.assertFalse(self.observe()["currentAcceptedDecisionMatched"])

    def test_current_world_mismatch_fails_closed(self):
        self.add_accepted(derived="sha256:" + "d" * 64)
        with self.assertRaisesRegex(ValueError, "generated World"):
            self.observe()

    def test_changed_frozen_bytes_are_rejected(self):
        self.save()
        self.frozen.write_bytes(self.frozen.read_bytes() + b" ")
        with self.assertRaisesRegex(ValueError, "differs"):
            self.observe()

    def test_diverged_or_forged_current_record_is_rejected(self):
        self.add_accepted()
        self.decisions["records"][0]["sourceEdits"] = "diverged"
        self.save()
        with self.assertRaisesRegex(ValueError, "source edits"):
            self.observe()
        self.decisions["records"][0]["sourceEdits"] = "intact"
        self.decisions["records"][0]["path"] = "../fake.json"
        self.save()
        with self.assertRaisesRegex(ValueError, "Decision path"):
            self.observe()

    def test_index_revision_and_duplicate_fields_are_rejected(self):
        self.decisions["modelRevision"] = "not-a-source-index-revision"
        self.save()
        with self.assertRaisesRegex(ValueError, "model revision"):
            self.observe()
        self.decisions["modelRevision"] = REV
        self.save()
        self.metadata.write_text('{"derivedWorldSha256":"%s","derivedWorldSha256":"%s"}' % (SHA, SHA))
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.observe()

    def test_rejected_decision_cannot_prove_acceptance(self):
        self.add_accepted()
        self.decisions["records"][0]["decision"]["decision"] = "rejected"
        self.save()
        self.assertFalse(self.observe()["currentAcceptedDecisionMatched"])


if __name__ == "__main__":
    unittest.main()
