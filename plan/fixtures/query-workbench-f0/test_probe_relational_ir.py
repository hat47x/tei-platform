"""Read-only structural negative tests of QX-17 against native RelationalIR JSON fields."""
import json
import unittest
from pathlib import Path
from probe_relational_ir import probe, compare_qx16

SAMPLES = json.loads(Path(__file__).with_name('relational_ir_synthetic.json').read_text(encoding='utf-8'))['samples']


def byid(result, ident):
    return next(n for n in result['nodes'] if n['id'] == ident)


class ProbeTests(unittest.TestCase):
    def test_prejoin_aggregate_only_depends_on_line(self):
        out = probe(SAMPLES['aggregate_before_join'])
        self.assertEqual([s['source'] for s in byid(out, 'agg_l')['sourceOccurrences']], ['line'])
        self.assertEqual([s['source'] for s in byid(out, 'join_groups')['sourceOccurrences']], ['line', 'delivery'])
        self.assertEqual(byid(out, 'agg_l')['grainCandidate'], 'aggregate-group-candidate')
        self.assertEqual(byid(out, 'agg_l')['claimConclusionPresence'], 'not-established')

    def test_aggregate_after_join_depends_on_both(self):
        out = probe(SAMPLES['join_before_aggregate'])
        self.assertEqual([s['source'] for s in byid(out, 'agg_after')['sourceOccurrences']], ['line', 'delivery'])
        self.assertEqual(byid(out, 'join_rows')['grainCandidate'], 'join-pair-candidate')
        self.assertEqual(byid(out, 'agg_after')['grainCandidate'], 'aggregate-group-candidate')
        self.assertEqual(byid(out, 'agg_after')['expressionSemantics'], 'unverified')

    def test_left_join_filter_cannot_prove_absence(self):
        out = probe(SAMPLES['left_join_filter'])
        self.assertEqual([s['source'] for s in byid(out, 'anti_filter')['sourceOccurrences']], ['header', 'delivery'])
        self.assertEqual(byid(out, 'anti_filter')['claimConclusionPresence'], 'not-established')
        self.assertEqual(byid(out, 'anti_filter')['expressionSemantics'], 'unverified')

    def test_same_physical_table_two_occurrences_are_not_deduped(self):
        sample = {'root': 'j', 'nodes': [
            {'id': 'left', 'kind': 'source', 'source': 'line', 'alias': 'a'},
            {'id': 'right', 'kind': 'source', 'source': 'line', 'alias': 'b'},
            {'id': 'j', 'kind': 'join', 'inputs': ['left', 'right']}]}
        o = probe(sample)
        self.assertEqual([s['operationId'] for s in byid(o, 'j')['sourceOccurrences']], ['left', 'right'])

    def test_missing_input_must_not_be_candidate(self):
        o = probe({'root': 'x', 'nodes': [{'id': 'x', 'kind': 'aggregate', 'inputs': ['no']}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertIn('missing-input:no', o['rootIssues'])

    def test_cycle_must_not_be_candidate(self):
        o = probe({'root': 'a', 'nodes': [
            {'id': 'a', 'kind': 'project', 'inputs': ['b']},
            {'id': 'b', 'kind': 'filter', 'inputs': ['a']}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertTrue(any(x.startswith('cycle:') for x in o['rootIssues']))

    def test_duplicate_node_id_must_not_be_candidate(self):
        o = probe({'root': 'a', 'nodes': [
            {'id': 'a', 'kind': 'source', 'source': 'line'},
            {'id': 'a', 'kind': 'source', 'source': 'delivery'}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertIn('duplicate-node:a', o['rootIssues'])

    def test_unknown_operator_and_opaque_not_assumed_complete(self):
        for kind in ['window', 'opaque']:
            with self.subTest(kind=kind):
                o = probe({'root': 'a', 'nodes': [{'id': 'a', 'kind': kind}]})
                self.assertEqual(o['rootStructureTrace'], 'unresolved')

    def test_cross_scope_source_binding_is_unverified(self):
        o = probe({'root': 'root', 'nodes': [
            {'id': 'child', 'kind': 'source', 'scopeId': 'scope:cte', 'source': 'line'},
            {'id': 'root', 'kind': 'project', 'scopeId': 'scope:root', 'inputs': ['child']}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertTrue(any(x.startswith('cross-scope-edge') for x in o['rootIssues']))

    def test_cte_source_requires_binding_verification(self):
        o = probe({'root': 'x', 'nodes': [{'id': 'x', 'kind': 'source',
            'source': 'orders_cte', 'sourceKind': 'cte', 'binding': 'scope:cte'}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertIn('logical-binding-not-expanded:x', o['rootIssues'])

    def test_invalid_operator_arity_is_unresolved(self):
        for node in [
            {'id': 'j', 'kind': 'join', 'inputs': ['single']},
            {'id': 'f', 'kind': 'filter'},
            {'id': 'a', 'kind': 'aggregate', 'inputs': 'src'},
        ]:
            with self.subTest(node=node):
                o = probe({'root': node['id'], 'nodes': [node]})
                self.assertEqual(o['rootStructureTrace'], 'unresolved')

    def test_missing_root_not_candidate(self):
        o = probe({'root': 'missing', 'nodes': [{'id': 'x', 'kind': 'source', 'source': 'line'}]})
        self.assertEqual(o['rootStructureTrace'], 'unresolved')
        self.assertIn('missing-root:missing', o['rootIssues'])

    def test_explicit_qx16_dependency_bridge_matches_but_does_not_prove_claim(self):
        # Independent expectations are hand-labeled QX-16 Claim input collections.
        labels = {'operationDependencies': {
            'line_sum': {'inputCollections': ['line']},
            'join_pairs': {'inputCollections': ['line', 'delivery']},
            'join_sum': {'inputCollections': ['line', 'delivery']},
            'left_unmatched': {'inputCollections': ['header', 'delivery']},
        }}
        matches = compare_qx16(SAMPLES, labels)
        self.assertEqual(len(matches['comparison']), 4)
        self.assertTrue(all(x['status'] == 'fixture-structure-agrees' for x in matches['comparison']))
        self.assertTrue(all(x['claimSemantics'] == 'not-established' for x in matches['comparison']))
        # A bad manually supplied identity of the source population is NOT accepted.
        labels['operationDependencies']['join_sum']['inputCollections'] = ['line']
        failed = compare_qx16(SAMPLES, labels)
        self.assertEqual(failed['comparison'][2]['status'], 'unresolved-or-mismatch')

    def test_candidate_is_not_claim_acceptance(self):
        o = probe(SAMPLES['aggregate_before_join'])
        self.assertEqual(o['rootStructureTrace'], 'candidate')
        self.assertIn('input collection completeness, identity, snapshot or monotonicity', o['notProven'])
        self.assertTrue(all(n['claimConclusionPresence'] == 'not-established' for n in o['nodes']))


if __name__ == '__main__':
    unittest.main()
