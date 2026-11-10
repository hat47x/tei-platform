import importlib.util
import json
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

MODULE = Path(__file__).with_name('audit.py')
spec = importlib.util.spec_from_file_location('rea_t0_audit', MODULE)
audit_mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit_mod)


class ReportAudit(unittest.TestCase):
    def setUp(self):
        self.tmp = TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / 'analysis.stderr').write_text('progress\n')

    def saved(self, exit_code, content):
        (self.root / 'exit-code.txt').write_text(str(exit_code) + '\n')
        (self.root / 'analysis.json').write_bytes(content)
        return audit_mod.audit(self.root)

    def test_zero_exit_is_not_full_coverage(self):
        out = self.saved(0, b'{"ok":true,"data":{"normalized_result":{"statistics":{"parse_failures":0}}}}')
        self.assertEqual(out['analysisQualification'], 'unknown')
        self.assertEqual(out['providerInvocation'], 'not-observed')
        self.assertEqual(out['reportedParseFailures'], 0)

    def test_exit_zero_partial(self):
        out = self.saved(0, b'{"ok":true,"data":{"normalized_result":{"statistics":{"parse_failures":2}}}}')
        self.assertEqual(out['analysisQualification'], 'partial')
        self.assertEqual(out['processOutcome'], 'exit-zero')

    def test_nonzero_empty_stdout_is_not_success(self):
        out = self.saved(1, b'')
        self.assertEqual(out['processOutcome'], 'exit-nonzero')
        self.assertEqual(out['reportShape'], 'empty')

    def test_nonzero_report_preserves_failure(self):
        out = self.saved(2, b'{"ok":false,"result":{"category":"unsupported_target"}}')
        self.assertEqual(out['analysisQualification'], 'reported-failure')
        self.assertEqual(out['processOutcome'], 'exit-nonzero')

    def test_truncated_json_does_not_promote(self):
        out = self.saved(0, b'{"ok":true,')
        self.assertEqual(out['reportShape'], 'invalid-json')
        self.assertEqual(out['analysisQualification'], 'unknown')

    def test_missing_exit_blocks_claim(self):
        with self.assertRaises(ValueError):
            audit_mod.audit(self.root)

    def test_symlink_report_rejected(self):
        outside = self.root / 'outside'
        outside.write_text('0')
        (self.root / 'exit-code.txt').symlink_to(outside)
        with self.assertRaises(ValueError):
            audit_mod.audit(self.root)

    def test_jsonl_object_metadata(self):
        path = self.root / 'view.jsonl'
        lines = [json.dumps({'id': 1}), json.dumps({'$rea': 'jsonl/v1', 'type': 'metadata', 'records': {'emitted': 1, 'source': 100}})]
        path.write_text('\n'.join(lines) + '\n')
        out = audit_mod.jsonl_audit(path, object_collection=True)
        self.assertEqual(out['framing'], 'object-collection-framed')
        self.assertEqual(out['sourceCoverage'], 'not-proven')

    def test_jsonl_missing_terminal(self):
        path = self.root / 'view.jsonl'
        path.write_text('{"id":1}\n')
        self.assertEqual(audit_mod.jsonl_audit(path, object_collection=True)['framing'], 'missing-metadata')

    def test_jsonl_bare_array_no_terminal_required(self):
        path = self.root / 'view.jsonl'
        path.write_text('{"id":1}\n')
        self.assertEqual(audit_mod.jsonl_audit(path, object_collection=False)['framing'], 'bare-array-without-completeness-proof')

    def test_jsonl_count_mismatch(self):
        path = self.root / 'view.jsonl'
        path.write_text('{"id":1}\n{"$rea":"jsonl/v1","type":"metadata","records":{"emitted":2}}\n')
        self.assertEqual(audit_mod.jsonl_audit(path, object_collection=True)['framing'], 'count-mismatch')


    def test_exit_report_conflict_on_zero_exit(self):
        out = self.saved(0, b'{"ok":false,"result":{"category":"invalid_request"}}')
        self.assertEqual(out['reportedOK'], False)
        self.assertEqual(out['processReportConsistency'], 'conflict')
        self.assertEqual(out['analysisQualification'], 'reported-failure')

    def test_exit_report_conflict_on_nonzero_exit(self):
        out = self.saved(2, b'{"ok":true,"data":{}}')
        self.assertEqual(out['reportedOK'], True)
        self.assertEqual(out['processReportConsistency'], 'conflict')
        self.assertEqual(out['analysisQualification'], 'unknown')

    def test_jsonl_error_control_record_in_object_view(self):
        path = self.root / 'view.jsonl'
        path.write_text('{"$rea":"jsonl/v1","type":"error","ok":false,"result":{"code":"invalid_input"}}\n')
        out = audit_mod.jsonl_audit(path, object_collection=True)
        self.assertEqual(out['framing'], 'reported-error')
        self.assertEqual(out['rowsSeen'], 0)

    def test_jsonl_error_control_record_in_bare_view(self):
        path = self.root / 'view.jsonl'
        path.write_text('{"$rea":"jsonl/v1","type":"error","ok":false}\n')
        out = audit_mod.jsonl_audit(path, object_collection=False)
        self.assertEqual(out['framing'], 'reported-error')
        self.assertEqual(out['rowsSeen'], 0)

    def test_jsonl_duplicate_control_rejected(self):
        path = self.root / 'view.jsonl'
        line = '{"$rea":"jsonl/v1","type":"metadata","records":{"emitted":1}}'
        path.write_text(line + '\n' + line + '\n')
        out = audit_mod.jsonl_audit(path, object_collection=True)
        self.assertEqual(out['framing'], 'extra-control-record')
        self.assertEqual(out['rowsSeen'], 0)

    def test_jsonl_invalid_utf8_is_not_a_valid_view(self):
        path = self.root / 'view.jsonl'
        path.write_bytes(b'\xff')
        self.assertEqual(audit_mod.jsonl_audit(path, object_collection=True)['framing'], 'invalid-jsonl')

    def test_symlink_report_directory_rejected(self):
        (self.root / 'exit-code.txt').write_text('0\n')
        alias = self.root / 'alias'
        alias.symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(ValueError):
            audit_mod.audit(alias)


if __name__ == '__main__':
    unittest.main()
