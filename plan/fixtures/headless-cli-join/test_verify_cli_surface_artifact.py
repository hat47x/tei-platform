import copy
import json
import sys
import unittest
import tempfile
from unittest.mock import patch
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))
from verify_cli_surface_artifact import compare, artifact_digest, manifest_digest, WitnessError, main


def sample():
    operations = {
        'schemaVersion': 'tei.operations.operation-surface/v0', 'effect': 'read-only',
        'application': 'order_entry', 'modelRevision': 'sha256:revision',
        'worldFingerprint': 'sha256:world',
        'rule': {'id': 'tei.operation-surface/standard-crud', 'version': 'v1', 'fingerprint': 'sha256:rule'},
        'operations': [{'id': 'order_entry.read.sales.order', 'basis': 'sha256:basis', 'autonomy': 'unattended'}],
        'gaps': [], 'summary': {'operations': 1, 'gaps': 0}, 'authorityLimits': ['not-a-grant'],
    }
    embedded = {k: operations[k] for k in ('schemaVersion', 'application', 'rule', 'operations', 'gaps', 'summary')}
    entry = {'kind': 'operation-surface', 'inputRevision': operations['modelRevision'],
             'rule': {'id': 'tei.artifact.operation-surface', 'version': 'v1', 'fingerprint': 'sha256:artifact-rule'},
             'generator': 'tei-artifact-surface/v1', 'content': json.dumps(embedded, sort_keys=True),
             'unresolved': []}
    entry['digest'] = artifact_digest(entry)
    manifest = {'inputRevision': operations['modelRevision'], 'entries': [
        {'kind': entry['kind'], 'rule': entry['rule']['id'],
         'version': entry['rule']['version'], 'fingerprint': entry['rule']['fingerprint'],
         'digest': entry['digest'], 'unresolved': 0}], 'complete': True}
    manifest['digest'] = manifest_digest(manifest)
    return operations, {'inputRevision': operations['modelRevision'], 'manifest': manifest,
                        'artifacts': [entry], 'findings': []}


def refingerprint(artifact_set):
    item = artifact_set['artifacts'][0]
    item['digest'] = artifact_digest(item)
    manifest = artifact_set['manifest']
    manifest['entries'][0]['digest'] = item['digest']
    manifest['digest'] = manifest_digest(manifest)


class EvidenceJoinCases(unittest.TestCase):
    def setUp(self):
        self.a, self.b = sample()

    def err(self, reason):
        with self.assertRaisesRegex(WitnessError, reason):
            compare(self.a, self.b, 'order_entry')

    def test_01_consistent_not_executable(self):
        out = compare(self.a, self.b, 'order_entry')
        self.assertEqual(out['status'], 'read_only_cross_projection_consistent')
        self.assertFalse(out['actionReady'])
        self.assertFalse(out['sameAtomicReadProven'])

    def test_02_cross_load_revision_mismatch(self):
        self.b['inputRevision'] = 'sha256:later'
        self.err('cross_load_revision_mismatch')

    def test_03_stale_artifact_body(self):
        self.b['artifacts'][0]['content'] += '\n'
        self.err('artifact_envelope_digest_mismatch')

    def test_04_validly_rehashed_old_operation(self):
        item = self.b['artifacts'][0]
        payload = json.loads(item['content'])
        payload['operations'][0]['basis'] = 'sha256:other'
        item['content'] = json.dumps(payload, sort_keys=True)
        refingerprint(self.b)
        self.err('surface_operations_mismatch')

    def test_05_manifest_digest_tampered(self):
        self.b['manifest']['digest'] = 'sha256:broken'
        self.err('manifest_digest_mismatch')

    def test_06_missing_member(self):
        self.b['manifest']['entries'].clear()
        self.b['manifest']['digest'] = manifest_digest(self.b['manifest'])
        self.err('manifest_member_mismatch')

    def test_07_findings_are_not_silent(self):
        self.b['findings'] = [{'code': 'ARTIFACT_CONTENT_INCOMPLETE'}]
        self.err('artifact_consistency_findings')

    def test_08_gap_does_not_grant_action(self):
        self.a['gaps'] = [{'target': 'sales.order', 'kind': 'update', 'reason': 'unbound', 'detail': 'test'}]
        self.b['artifacts'][0]['unresolved'] = [{'subject': 'sales.order', 'reason': 'unbound', 'detail': 'update: test'}]
        payload = json.loads(self.b['artifacts'][0]['content'])
        payload['gaps'] = self.a['gaps']
        self.b['artifacts'][0]['content'] = json.dumps(payload, sort_keys=True)
        self.b['manifest']['entries'][0]['unresolved'] = 1
        self.b['manifest']['complete'] = False
        refingerprint(self.b)
        result = compare(self.a, self.b, 'order_entry')
        self.assertEqual(result['unresolvedGapCount'], 1)
        self.assertFalse(result['actionReady'])

    def test_09_other_application_not_interchangeable(self):
        self.err_app = 'inventory'
        with self.assertRaisesRegex(WitnessError, 'surface_application_mismatch'):
            compare(self.a, self.b, self.err_app)

    def test_10_rule_fingerprint_drift(self):
        self.b['artifacts'][0]['content'] = self.b['artifacts'][0]['content'].replace('sha256:rule', 'sha256:changed-rule')
        refingerprint(self.b)
        self.err('surface_rule_mismatch')

    def test_11_manifest_complete_claim_tampered(self):
        self.b['manifest']['complete'] = False
        self.b['manifest']['digest'] = manifest_digest(self.b['manifest'])
        self.err('manifest_completeness_mismatch')

    def test_12_missing_world_identity(self):
        self.a['worldFingerprint'] = ''
        self.err('world_fingerprint_absent')

    def test_13_duplicate_member_is_not_ignored(self):
        self.b['manifest']['entries'].append(copy.deepcopy(self.b['manifest']['entries'][0]))
        self.b['manifest']['digest'] = manifest_digest(self.b['manifest'])
        self.err('manifest_member_mismatch')

    def test_14_wrong_surface_schema(self):
        self.a['schemaVersion'] = 'unknown'
        self.err('surface_schema_unknown')


class RealCommandHandoffShape(unittest.TestCase):
    def test_15_calls_actual_command_names_without_writes(self):
        a, b = sample()
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            binary = root / 'tei'
            binary.touch()
            ws = root / 'workspace'
            ws.mkdir()
            (ws / 'tei.workspace.yaml').write_text('fixture: true', encoding='utf-8')
            revision = 'a' * 40
            with patch('verify_cli_surface_artifact.run_cli', side_effect=[{'vcsRevision': revision}, a, b]) as run:
                with patch('builtins.print') as emit:
                    main(['--binary', str(binary), '--workspace', str(ws),
                          '--application', 'order_entry', '--expected-head', revision])
                self.assertEqual(run.call_count, 3)
                self.assertEqual(run.call_args_list[0].args[1:], ('version', '-json'))
                self.assertEqual(run.call_args_list[1].args[1:3], ('operations', '-workspace'))
                self.assertEqual(run.call_args_list[2].args[1:3], ('artifacts', '-workspace'))
                self.assertTrue(json.loads(emit.call_args.args[0])['actionReady'] is False)

    def test_16_wrong_binary_revision_stops_before_workspace_read(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            binary = root / 'tei'
            binary.touch()
            ws = root / 'workspace'
            ws.mkdir()
            (ws / 'tei.workspace.yaml').touch()
            with patch('verify_cli_surface_artifact.run_cli', return_value={'vcsRevision': 'b' * 40}) as run:
                with self.assertRaisesRegex(WitnessError, 'binary_head_mismatch'):
                    main(['--binary', str(binary), '--workspace', str(ws),
                          '--application', 'order_entry', '--expected-head', 'a' * 40])
                self.assertEqual(run.call_count, 1)


if __name__ == '__main__':
    unittest.main(verbosity=2)
