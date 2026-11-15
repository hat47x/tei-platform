"""Contract tests for the REA real-tool verifier itself.

These tests mock the CLI transport only. They neither run REA nor qualify
captured graph data as an actual candidate or an accepted business meaning.
"""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock


MODULE_PATH = Path(__file__).with_name("verify_real_rea_intake.py")
spec = importlib.util.spec_from_file_location("verify_real_rea_intake", MODULE_PATH)
verifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verifier)


def evidence_bytes(with_graph):
    data = {"evidence_id": "ev-test", "normalized_result": {}}
    if with_graph:
        data["normalized_result"]["semantic_graph"] = {
            "nodes": [{"node_id": "node-1"}],
            "relations": [{"relation_id": "relation-1"}],
        }
    return json.dumps({"ok": True, "data": data}).encode("utf-8")


class RealREAVerifierBoundaryTests(unittest.TestCase):
    def save(self, directory, raw):
        path = Path(directory) / "evidence.json"
        path.write_bytes(raw)
        return path

    def fake_cli(self, evidence_path, calls):
        received = "sha256:" + verifier.digest(evidence_path)

        def invoke(binary, args, expect=0):
            self.assertEqual(expect, 0)
            calls.append(list(args))
            if args[0] == "rea-evidence":
                if "-request" in args:
                    request_path = Path(args[args.index("-request") + 1])
                    request = json.loads(request_path.read_text(encoding="utf-8"))
                    self.assertEqual(request["source"]["kind"], "inline")
                    self.assertEqual(request["source"]["evidence"]["evidence_id"], "ev-test")
                return {
                    "ok": True,
                    "result": {"receipt": {
                        "state": "complete",
                        "evidence": {"id": "ev-test"},
                        "raw": {"fingerprint": received},
                        "promotion": {
                            "acceptedMeaning": False,
                            "canonicalWorld": False,
                            "databaseObservation": False,
                        },
                    }},
                }
            self.assertEqual(args[0], "external-candidates")
            return {
                "ok": True,
                "schemaVersion": "tei.cli.external-candidates/v0",
                "result": {
                    "schemaVersion": "tei.operations.external-candidates/v0",
                    "effect": "read-only",
                    "attempt": {"receivedDigest": received},
                    "observation": {"nodeId": "node-1"},
                    "relation": {"relationId": "relation-1"},
                    "authorityLimits": [
                        "external-candidate-is-not-accepted-meaning",
                        "external-identity-is-not-a-tei-semantic-subject",
                    ],
                },
            }
        return invoke

    def test_no_graph_still_receives_and_validates_without_selecting(self):
        with tempfile.TemporaryDirectory() as directory:
            path = self.save(directory, evidence_bytes(False))
            self.assertIsNone(verifier.selection_from_evidence(path))
            report = {"checks": [], "notRun": []}
            calls = []
            with mock.patch.object(verifier, "tei_json", side_effect=self.fake_cli(path, calls)):
                verifier.verify_common("tei", path, "6.8.0", None, report)
            self.assertEqual(len(calls), 2)
            self.assertTrue(all(c[0] == "rea-evidence" for c in calls))
            self.assertEqual(len(report["checks"]), 1)
            self.assertEqual(report["checks"][0]["receipt"], verifier.digest(path))
            self.assertTrue(any("NOT RUN" in text for text in report["notRun"]))

    def test_node_relation_selection_uses_existing_external_candidates_cli(self):
        with tempfile.TemporaryDirectory() as directory:
            path = self.save(directory, evidence_bytes(True))
            selection = verifier.selection_from_evidence(path)
            self.assertEqual(selection, {"nodeID": "node-1", "relationID": "relation-1"})
            report = {"checks": [], "notRun": []}
            calls = []
            with mock.patch.object(verifier, "tei_json", side_effect=self.fake_cli(path, calls)):
                verifier.verify_common("tei", path, "6.8.0", selection, report)
            self.assertEqual(len(calls), 3)
            self.assertEqual([c[0] for c in calls], ["rea-evidence", "rea-evidence", "external-candidates"])
            self.assertEqual(calls[2], [
                "external-candidates", "-evidence-path", str(path),
                "-tool-version", "6.8.0", "-observation-id", "node-1",
                "-relation-id", "relation-1",
            ])
            self.assertEqual(report["notRun"], [])
            self.assertEqual(len(report["checks"]), 2)

    def test_graph_without_producer_relation_id_is_not_projected(self):
        with tempfile.TemporaryDirectory() as directory:
            document = json.loads(evidence_bytes(True))
            document["data"]["normalized_result"]["semantic_graph"]["relations"] = [
                {"relation": "imports"}]
            path = self.save(directory, json.dumps(document).encode())
            self.assertIsNone(verifier.selection_from_evidence(path))

    def test_malformed_graph_is_not_silently_downgraded(self):
        with tempfile.TemporaryDirectory() as directory:
            document = json.loads(evidence_bytes(False))
            document["data"]["normalized_result"]["semantic_graph"] = ["wrong"]
            path = self.save(directory, json.dumps(document).encode())
            with self.assertRaisesRegex(AssertionError, "semantic_graph must be an object"):
                verifier.selection_from_evidence(path)

    def test_failed_real_verification_still_writes_a_failure_record(self):
        with tempfile.TemporaryDirectory() as directory:
            report_path = Path(directory) / "failed-intake.json"
            with mock.patch.object(verifier, "verify_real_mode",
                                   side_effect=AssertionError("selection rejected")):
                with mock.patch.object(verifier.sys, "argv", [
                    "verify_real_rea_intake.py", "--binary", "tei",
                    "--version", "6.8.0", "--report", str(report_path),
                ]):
                    with self.assertRaisesRegex(AssertionError, "selection rejected"):
                        verifier.main()
            report = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(report["status"], "failed")
            self.assertEqual(report["failure"]["type"], "AssertionError")
            self.assertEqual(report["checks"], [])
            self.assertEqual(report["notRun"], [])
            self.assertEqual(report["version"], "6.8.0")

    def test_captured_receipt_does_not_invent_a_semantic_graph(self):
        path = (Path(__file__).resolve().parents[3] /
                "internal/externalanalysis/testdata/rea-6.8.0/analysis.json")
        captured = path.read_bytes()
        self.assertIsNone(verifier.selection_from_evidence(path))
        self.assertEqual(path.read_bytes(), captured)


if __name__ == "__main__":
    unittest.main()
