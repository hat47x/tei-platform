#!/usr/bin/env python3
"""QX-17: conservative, read-only probe of the existing Go RelationalIR JSON shape.

Only follows RelNode.Inputs. It does NOT parse SQL, establish physical-schema
binding, resolve correlated references, infer row lineage or prove completeness.
Never turn a structural trace into a business/Provider/Authority claim.
"""
import argparse
import json
from pathlib import Path

KNOWN_KINDS = frozenset({
    'source', 'join', 'filter', 'aggregate', 'project',
    'distinct', 'sort', 'limit', 'opaque',
})


def probe(relational):
    """Conservatively walk operation-ID edges of a native-shaped RelationalIR."""
    raw_nodes = relational.get('nodes', [])
    root = relational.get('root', '')
    if not isinstance(raw_nodes, list):
        raise ValueError('nodes must be an array')
    index, duplicates = {}, set()
    for n in raw_nodes:
        if not isinstance(n, dict) or not isinstance(n.get('id'), str) or not n['id']:
            raise ValueError('nodes must have nonempty string IDs')
        if n['id'] in index:
            duplicates.add(n['id'])
        else:
            index[n['id']] = n
    memo, visiting = {}, set()

    def visit(nid):
        if nid in memo:
            return memo[nid]
        if nid in visiting:
            return {'sources': [], 'issues': [f'cycle:{nid}']}
        if nid not in index:
            return {'sources': [], 'issues': [f'missing-input:{nid}']}
        if nid in duplicates:
            return {'sources': [], 'issues': [f'duplicate-node:{nid}']}
        visiting.add(nid)
        n = index[nid]
        kind = n.get('kind')
        scope = n.get('scopeId') or 'scope:root'
        sources, issues = [], []
        if kind not in KNOWN_KINDS:
            issues.append(f'unknown-kind:{nid}:{kind}')
        if kind == 'opaque':
            issues.append(f'opaque-operator:{nid}')
        inputs = n.get('inputs') or []
        if not isinstance(inputs, list):
            issues.append(f'invalid-input-list:{nid}')
            inputs = []
        if kind == 'join' and len(inputs) < 2:
            issues.append(f'join-needs-two-inputs:{nid}')
        if kind in ('aggregate', 'project', 'filter', 'distinct', 'sort', 'limit') and not inputs:
            issues.append(f'operator-needs-input:{nid}')
        if kind == 'source':
            source = n.get('source')
            if not source:
                issues.append(f'source-name-missing:{nid}')
            else:
                sources.append({
                    'operationId': nid, 'scopeId': scope,
                    'source': source, 'sourceKind': n.get('sourceKind') or 'table',
                    'binding': n.get('binding') or '', 'alias': n.get('alias') or '',
                })
            if (n.get('sourceKind') in ('cte', 'subquery')):
                issues.append(f'logical-binding-not-expanded:{nid}')
            if inputs:
                issues.append(f'source-has-operation-inputs:{nid}')
        else:
            for child_id in inputs:
                if not isinstance(child_id, str):
                    issues.append(f'non-string-input:{nid}')
                    continue
                child = visit(child_id)
                sources.extend(child['sources'])
                issues.extend(child['issues'])
                child_node = index.get(child_id, {})
                child_scope = child_node.get('scopeId') or 'scope:root'
                if child_scope != scope:
                    # CTE and subquery cross-scope bindings can be legitimate.
                    # The local operator graph cannot verify their visibility.
                    issues.append(f'cross-scope-edge-unverified:{child_id}->{nid}')
        visiting.remove(nid)
        # Preserve distinct source operation occurrences, including self-joins.
        unique = list({s['operationId']: s for s in sources}.values())
        result = {'sources': unique, 'issues': sorted(set(issues))}
        memo[nid] = result
        return result

    if not root or root not in index:
        root_issues = [f'missing-root:{root or "<empty>"}']
        root_walk = {'sources': [], 'issues': root_issues}
    else:
        root_walk = visit(root)

    results = []
    for n in raw_nodes:
        ident = n['id']
        result = visit(ident)
        kind = n.get('kind')
        grain = {
            'source': 'source-occurrence-candidate',
            'join': 'join-pair-candidate',
            'aggregate': 'aggregate-group-candidate',
            'filter': 'row-filter-candidate',
        }.get(kind, 'unspecified')
        results.append({
            'id': ident, 'kind': kind, 'scopeId': n.get('scopeId') or 'scope:root',
            'inputOperationIds': n.get('inputs') if isinstance(n.get('inputs'), list) else [],
            'sourceOccurrences': result['sources'],
            'grainCandidate': grain,
            'structureTrace': 'candidate' if not result['issues'] else 'unresolved',
            'issues': result['issues'],
            'expressions': n.get('expressions') or [],
            'expressionSemantics': 'unverified' if n.get('expressions') else 'not-supplied',
            'claimConclusionPresence': 'not-established',
        })
    return {
        'kind': 'synthetic-relational-ir-structure-probe',
        'root': root,
        'rootStructureTrace': 'candidate' if not root_walk['issues'] else 'unresolved',
        'rootIssues': root_walk['issues'],
        'nodes': results,
        'notProven': [
            'SQL source->worker->RelationalIR provenance or dialect',
            'column and predicate semantics or expression typing',
            'input collection completeness, identity, snapshot or monotonicity',
            'row/aggregate business grain, resulting values, inverse edit or authority',
            'physical binding and CTE/correlation scope visibility',
            'accepted business semantics and execution/Provider conformance',
        ],
    }


# Explicit fixture mapping, NOT a generated mapping from SQL or a generic operator API.
QX16_LINKS = (
    ('aggregate_before_join', 'agg_l', 'line_sum'),
    ('join_before_aggregate', 'join_rows', 'join_pairs'),
    ('join_before_aggregate', 'agg_after', 'join_sum'),
    ('left_join_filter', 'anti_filter', 'left_unmatched'),
)


def compare_qx16(samples, qx16_report):
    """Compare synthetic IR input dependencies with existing QX-16 hand-labels."""
    labels = qx16_report.get('operationDependencies', {})
    comparisons = []
    for case_id, op_id, claim_id in QX16_LINKS:
        case = samples.get(case_id)
        trace = probe(case) if case is not None else None
        node = next((n for n in trace['nodes'] if n['id'] == op_id), None) if trace else None
        manual = labels.get(claim_id)
        actual_sources = [s['source'] for s in node['sourceOccurrences']] if node else []
        expected_sources = manual.get('inputCollections') if isinstance(manual, dict) else None
        compatible = (node is not None and node['structureTrace'] == 'candidate'
                      and isinstance(expected_sources, list)
                      and set(expected_sources) == set(actual_sources)
                      and len(expected_sources) == len(actual_sources))
        comparisons.append({
            'sample': case_id, 'relNodeId': op_id, 'qx16ManualClaimId': claim_id,
            'qx16ExpectedSources': expected_sources, 'irTracedSources': actual_sources,
            'status': 'fixture-structure-agrees' if compatible else 'unresolved-or-mismatch',
            'claimSemantics': 'not-established',
        })
    return {'kind': 'fixture-only-qx16-to-ir-dependency-comparison',
            'comparison': comparisons,
            'notProven': [
                'actual SQL/SQLGlot/DeriveFromAnalysis correspondence',
                'automatic linkage from OutputField to RelNode',
                'operator meaning, Provider completeness, Owner acceptance or Effect',
            ]}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('fixture', type=Path, help='JSON RelationalIR, or a named samples collection')
    parser.add_argument('--sample', help='select one named sample from a collection')
    parser.add_argument('--compare-qx16', type=Path, help='compare existing explain_qx16.py JSON output with these synthetic IR samples')
    args = parser.parse_args()
    with args.fixture.open(encoding='utf-8') as f:
        data = json.load(f)
    if 'samples' in data:
        samples = data['samples']
        if args.compare_qx16:
            if args.sample:
                parser.error('--compare-qx16 cannot be used with --sample')
            with args.compare_qx16.open(encoding='utf-8') as f:
                old_report = json.load(f)
            output = compare_qx16(samples, old_report)
        elif args.sample:
            if args.sample not in samples:
                parser.error(f'unknown sample: {args.sample}')
            output = probe(samples[args.sample])
        else:
            output = {name: probe(item) for name, item in samples.items()}
    else:
        if args.sample or args.compare_qx16:
            parser.error('--sample/--compare-qx16 requires a samples collection')
        output = probe(data)
    print(json.dumps(output, ensure_ascii=False, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
