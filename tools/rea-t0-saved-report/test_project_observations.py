import hashlib
import json
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

import project_observations as projector

ROOT_SHA = 'a' * 64
FILE_BYTES = b'fixture public source\n'
FILE_SHA = hashlib.sha256(FILE_BYTES).hexdigest()
ROOT = 'build/rea-ci-demo'


def evidence(state='observed', digest=FILE_SHA, loc=None):
    return {'state': state, 'authority': 'artifact-bytes',
            'artifact': {'available': True, 'sha256': digest},
            'location': {'available': True, 'value': loc or {'kind': 'artifact-path', 'path': 'index.js'}},
            'coverage': {'status': 'complete'}}


def doc():
    graph = {
        'schema': 'JavaScriptApplicationGraph', 'coverage': {'status': 'complete'},
        'nodes': [
            {'node_id': 'N1', 'kind': 'javascript-asset', 'observations':
                [{'observation_id': 'O1', 'evidence': evidence()}]},
            {'node_id': 'N2', 'kind': 'package', 'observations':
                [{'observation_id': 'O2', 'evidence': evidence(
                    loc={'kind': 'artifact-path', 'path': 'package.json'})}]}
        ],
        'edges': [{'edge_id': 'E1', 'source_node_id': 'N2', 'target_node_id': 'N1',
                   'relation': 'loads', 'evidence': evidence(
                    loc={'kind': 'artifact-path', 'path': 'package.json'})}]
    }
    return {'ok': True, 'data': {
        'operation': 'analyze_javascript_application', 'evidence_id': 'ev_test',
        'subject': {'digest': {'sha256': ROOT_SHA}},
        'provider': {'id': 'rea-javascript-application', 'version': '1'},
        'limitations': ['static is not runtime'],
        'normalized_result': {
            'root_artifact_sha256': ROOT_SHA, 'graph': graph,
            'semantic_graph': {'unknowns': [
                {'unknown_id': 'U1', 'family': 'call-flow', 'reason': 'dynamic-call'}],
                'coverage': {'status': 'unknown'}}
        }
    }}


class TestExternalProjection(unittest.TestCase):
    def setUp(self):
        self.tmp = TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root/'selected-target.txt').write_text(ROOT+'\n')
        (self.root/'fixture-source-sha256.txt').write_text(FILE_SHA+'  '+ROOT+'/index.js\n'+FILE_SHA+'  '+ROOT+'/package.json\n')
        (self.root/'analysis.stderr').write_text('progress\n')
        self.setdoc(doc())

    def setdoc(self, obj, code=0):
        (self.root/'analysis.json').write_text(json.dumps(obj))
        (self.root/'exit-code.txt').write_text(str(code)+'\n')

    def test_cli_string_report_path_is_supported(self):
        self.snapshots()
        result = projector.project(str(self.root))
        self.assertEqual(result['admission'], 'external-candidates-only')
        self.assertEqual(result['sourceManifestStatus'], 'report-pre-post-byte-match')

    def test_projection_is_not_canonical(self):
        x = projector.project(self.root)
        self.assertEqual(x['admission'],'external-candidates-only')
        self.assertFalse(x['acceptedMeaning'])
        self.assertFalse(x['qualifiedTEISource'])
        self.assertIsNone(x['TEICanonicalSubject'])
        self.assertEqual(len(x['records']),3)

    def test_source_manifest_is_only_self_report(self):
        x=projector.project(self.root)
        self.assertTrue(all(z['sourceBinding']=='saved-manifest-match-only' for z in x['records']))
        self.assertEqual(x['sourceClosure'],'not-qualified')

    def test_unknown_is_preserved_not_absent(self):
        x=projector.project(self.root)
        self.assertEqual(x['unknowns'][0]['externalUnknownID'],'U1')
        self.assertEqual(x['semanticCoverage']['status'],'unknown')

    def test_nonzero_is_refused(self):
        obj=doc();obj['ok']=False
        self.setdoc(obj,1)
        x=projector.project(self.root)
        self.assertEqual(x['admission'],'refused')
        self.assertEqual(x['records'],[])

    def snapshots(self):
        for dirname in ('source-pre', 'source-post'):
            dest = self.root / dirname
            dest.mkdir()
            for name in ('index.js', 'package.json'):
                (dest / name).write_bytes(FILE_BYTES)

    def test_pre_and_post_saved_bytes_match_without_meaning_acceptance(self):
        self.snapshots()
        x = projector.project(self.root)
        self.assertEqual(x['sourceManifestStatus'], 'report-pre-post-byte-match')
        self.assertTrue(all(z['sourceBinding'] == 'report-pre-post-byte-match' for z in x['records']))
        self.assertFalse(x['acceptedMeaning'])
        self.assertEqual(x['sourceClosure'], 'not-qualified')

    def test_pre_source_manifest_conflict_rejected(self):
        self.snapshots()
        (self.root / 'source-pre/index.js').write_bytes(b'changed before analysis')
        with self.assertRaisesRegex(ValueError, 'pre-source byte differs'):
            projector.project(self.root)

    def test_post_source_manifest_conflict_rejected(self):
        self.snapshots()
        (self.root / 'source-post/index.js').write_bytes(b'changed after analysis')
        with self.assertRaisesRegex(ValueError, 'post-source byte differs'):
            projector.project(self.root)

    def test_incomplete_snapshot_pair_rejected(self):
        (self.root / 'source-pre').mkdir()
        with self.assertRaisesRegex(ValueError, 'inconsistent'):
            projector.project(self.root)

    def test_missing_one_source_copy_rejected(self):
        self.snapshots()
        (self.root / 'source-post/package.json').unlink()
        with self.assertRaisesRegex(ValueError, 'missing saved report member'):
            projector.project(self.root)

    def test_symlink_source_copy_rejected(self):
        self.snapshots()
        (self.root / 'source-post/index.js').unlink()
        (self.root / 'source-post/index.js').symlink_to(self.root / 'source-pre/index.js')
        with self.assertRaisesRegex(ValueError, 'symlink'):
            projector.project(self.root)

    def test_bogus_third_manifest_source_refused(self):
        with (self.root / 'fixture-source-sha256.txt').open('a') as f:
            f.write(FILE_SHA + '  ' + ROOT + '/undisclosed.js\n')
        with self.assertRaisesRegex(ValueError, 'exactly the two'):
            projector.project(self.root)

    def test_wrong_root_digest_refused(self):
        obj=doc();obj['data']['normalized_result']['root_artifact_sha256']='f'*64
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')

    def test_source_digest_conflict_rejected(self):
        obj=doc();obj['data']['normalized_result']['graph']['nodes'][0]['observations'][0]['evidence']['artifact']['sha256']='f'*64
        self.setdoc(obj)
        with self.assertRaisesRegex(ValueError,'source digest conflict'):
            projector.project(self.root)

    def test_duplicate_external_observation_refused(self):
        obj=doc();obj['data']['normalized_result']['graph']['nodes'][1]['observations'][0]['observation_id']='O1'
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')

    def test_unknown_endpoint_refused(self):
        obj=doc();obj['data']['normalized_result']['graph']['edges'][0]['target_node_id']='MISSING'
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')

    def test_missing_manifest_rejected(self):
        (self.root/'fixture-source-sha256.txt').unlink()
        with self.assertRaisesRegex(ValueError,'missing saved report member'):
            projector.project(self.root)

    def test_manifest_outside_selection_rejected(self):
        (self.root/'fixture-source-sha256.txt').write_text(FILE_SHA+'  other/index.js\n')
        with self.assertRaisesRegex(ValueError,'outside selected'):
            projector.project(self.root)

    def test_missing_semantic_unknowns_refused(self):
        obj=doc();del obj['data']['normalized_result']['semantic_graph']['unknowns']
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')

    def test_unavailable_occurrence_not_observed(self):
        obj=doc();obj['data']['normalized_result']['graph']['nodes'][0]['observations'][0]['evidence']['state']='unavailable'
        self.setdoc(obj)
        x=projector.project(self.root)
        self.assertEqual(x['records'][0]['observedState'],'unavailable')
        self.assertFalse(x['acceptedMeaning'])

    def test_invalid_range_stops_location_bind(self):
        obj=doc();obs=obj['data']['normalized_result']['graph']['nodes'][0]['observations'][0]
        obs['evidence']['location']['value']={'kind':'source-range','source':'index.js','start':{'line':3,'column':2},'end':{'line':1,'column':1}}
        self.setdoc(obj)
        x=projector.project(self.root)
        self.assertIsNone(x['records'][0]['source'])
        self.assertEqual(x['records'][0]['sourceBinding'],'not-independently-bound')

    def test_parallel_relations_keep_independent_identity(self):
        obj=doc();g=obj['data']['normalized_result']['graph']
        e=dict(g['edges'][0]);e['edge_id']='E2';g['edges'].append(e)
        self.setdoc(obj)
        x=projector.project(self.root)
        ids=[z['externalOccurrenceID'] for z in x['records'] if z['kind']=='graph-relation-observation']
        self.assertEqual(ids,['E1','E2'])

    def test_wrong_provider_refused(self):
        obj=doc();obj['data']['provider']['id']='not-rea'
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')

    def test_duplicated_semantic_unknown_refused(self):
        obj=doc();unknown=obj['data']['normalized_result']['semantic_graph']['unknowns']
        unknown.append(dict(unknown[0]))
        self.setdoc(obj)
        self.assertEqual(projector.project(self.root)['admission'],'refused')


if __name__=='__main__':unittest.main()
