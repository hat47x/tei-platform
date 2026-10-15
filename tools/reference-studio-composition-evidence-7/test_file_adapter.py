"""Probe contract regression tests; not a claim of PoC B builder qualification."""
import copy
import hashlib
import importlib.util
import json
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).with_name('file_adapter.py')
spec = importlib.util.spec_from_file_location('studio_b_file_probe', SOURCE)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)

GOOD = {
    'fixtureVersion': 'tei-studio-poc-b-reference-v0',
    'scope': 'studio.local',
    'services': [{'id': 'studio.validate', 'version': '0.1.0'},
                 {'id': 'studio.catalog', 'version': '0.1.0'}],
    'uiKit': {'id': 'studio.readonly', 'version': '0.1.0',
              'framework': {'major': 1},
              'slots': ['graph', 'commands'],
              'changes': [{'slot': 'graph', 'implementation': 'studio.graph'},
                          {'slot': 'commands', 'implementation': 'studio.commands'}]},
}


def observed(kind, raw):
    if kind == 'go':
        source = (b'package staticservices\n'
                  b'type CatalogService struct{}; type ValidateService struct{}\n'
                  b'// operations.CatalogWorkspace operations.ValidateWorkspace\n'
                  b'func Services(root string) []any {return nil}\n')
        name = probe.GO_FILES
        meta = {'serviceOrder': ['studio.catalog', 'studio.validate']}
    else:
        source = (b'import { mountCommandPanel as ui0 } from "../../../reference/tei-studio-command/command.ts";\n'
                  b'import { mountReadonlyGraph as ui1 } from "../../../reference/tei-studio-graph/graph.ts";\n'
                  b'export const studioUIContributions = [];\n')
        name = probe.UI_FILES
        meta = {'uiContributions': [
            {'slot': 'commands', 'implementation': 'studio.commands', 'kit': 'studio.readonly',
             'version': '0.1.0', 'sourcePath': '../../../reference/tei-studio-command/command.ts'},
            {'slot': 'graph', 'implementation': 'studio.graph', 'kit': 'studio.readonly',
             'version': '0.1.0', 'sourcePath': '../../../reference/tei-studio-graph/graph.ts'}]}
    meta.update(reference='tei-studio-poc-b-reference-v0', fixtureSha256=probe.sha(raw),
                generatedSha256=probe.sha(source), authorityLimitations=['not-runtime'])
    return {'code': 0, 'stdout': b'', 'stderr': b'', 'files': list(name),
            'data': {name[0]: source, name[1]: json.dumps(meta).encode()}, 'imports': []}


class FileProbeTests(unittest.TestCase):
    def setUp(self):
        self.raw = json.dumps(GOOD).encode()

    def test_generation_evidence_for_both_real_file_shapes(self):
        for kind in ('go', 'ui'):
            with self.subTest(kind=kind):
                result = probe.assert_success(observed(kind, self.raw), kind, self.raw)
                self.assertEqual(result['fixtureSha256'], hashlib.sha256(self.raw).hexdigest())

    def test_connected_head_requires_declaration_provenance_and_exports(self):
        result = observed('ui', self.raw)
        source = (result['data']['ui.generated.mjs'] +
                  b'export const mountCommandPanel = ui0;\n'
                  b'export const mountReadonlyGraph = ui1;\n')
        declaration = (b'export { mountCommandPanel } from "command.ts";\n'
                       b'export { mountReadonlyGraph } from "graph.ts";\n')
        result['files'] = list(probe.CONNECTED_UI_FILES)
        result['data']['ui.generated.mjs'] = source
        result['data']['ui.generated.d.mts'] = declaration
        meta = json.loads(result['data']['ui.provenance.json'])
        meta['generatedSha256'] = probe.sha(source)
        meta['declarationsSha256'] = probe.sha(declaration)
        result['data']['ui.provenance.json'] = json.dumps(meta).encode()
        evidence = probe.assert_success(result, 'ui', self.raw, probe.CONNECTED_HEAD)
        self.assertEqual(evidence['sourceSha256'], probe.sha(source))
        result['data']['ui.generated.d.mts'] = b'tampered'
        with self.assertRaisesRegex(probe.Failed, 'declaration provenance fingerprint differs'):
            probe.assert_success(result, 'ui', self.raw, probe.CONNECTED_HEAD)

    def test_connected_head_keeps_exact_three_outputs_and_pins(self):
        self.assertEqual(probe.ui_files_for(probe.CONNECTED_HEAD),
                         ('ui.generated.mjs', 'ui.generated.d.mts', 'ui.provenance.json'))
        self.assertEqual(probe.ui_files_for(probe.NEW_HEAD), probe.UI_FILES)
        self.assertEqual(probe.pins_for(probe.CONNECTED_HEAD)['connect-workbench.mjs'],
                         '29249c44d02af687103eff5124db95e5bea88c00')
        self.assertEqual(probe.pins_for(probe.CONNECTED_HEAD)['compose-ui.mjs'],
                         '6e20b723ff7f0a3abf2ddce9267c879feeff3599')

    def test_stdout_is_not_an_implicit_json_result(self):
        result = observed('ui', self.raw)
        result['stdout'] = b'{}'
        with self.assertRaisesRegex(probe.Failed, 'unexpected stdout'):
            probe.assert_success(result, 'ui', self.raw)

    def test_missing_or_extra_output_fails(self):
        result = observed('ui', self.raw)
        result['files'].append('untrusted.txt')
        with self.assertRaisesRegex(probe.Failed, 'file outputs mismatch'):
            probe.assert_success(result, 'ui', self.raw)

    def test_tampered_source_is_not_confirmed_by_metadata(self):
        result = observed('go', self.raw)
        result['data'][probe.GO_FILES[0]] = b'tamper'
        with self.assertRaisesRegex(probe.Failed, 'generated byte hash mismatch'):
            probe.assert_success(result, 'go', self.raw)

    def test_tampered_input_is_not_accepted(self):
        result = observed('ui', self.raw)
        with self.assertRaisesRegex(probe.Failed, 'fixture byte hash mismatch'):
            probe.assert_success(result, 'ui', b'other bytes')

    def test_cases_use_real_fixture_mutations(self):
        cases = probe.cases(GOOD)
        self.assertEqual(len(cases), 12)
        self.assertIn('ui.missing_parent', cases)
        self.assertIn('ui.cycle', cases)
        for name, (raw, reason) in cases.items():
            with self.subTest(name=name):
                self.assertNotEqual(json.loads(raw), GOOD)
                self.assertTrue(reason)
        self.assertNotIn('extends', GOOD['uiKit'])

    def test_git_blob_identity(self):
        raw = b'abc'
        self.assertEqual(probe.git_blob(raw), 'f2ba8f84ab5c1bce84a7b441cb1959cfc7093b7f')

    def test_go_preflight_reports_old_toolchain_blocked(self):
        class P:
            returncode = 0
            stdout = 'go version go1.23.2 linux/amd64'
        with patch.object(probe.subprocess, 'run', return_value=P()):
            self.assertFalse(probe.go_toolchain_ready(Path('.')))
        P.stdout = 'go version go1.26.9 linux/amd64'
        with patch.object(probe.subprocess, 'run', return_value=P()):
            self.assertTrue(probe.go_toolchain_ready(Path('.')))


if __name__ == '__main__':
    unittest.main()
