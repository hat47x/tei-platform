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


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('fixture', type=Path, help='JSON RelationalIR, or a named samples collection')
    parser.add_argument('--sample', help='select one named sample from a collection')
    args = parser.parse_args()
    with args.fixture.open(encoding='utf-8') as f:
        data = json.load(f)
    if 'samples' in data:
        samples = data['samples']
        if args.sample:
            if args.sample not in samples:
                parser.error(f'unknown sample: {args.sample}')
            output = probe(samples[args.sample])
        else:
            output = {name: probe(item) for name, item in samples.items()}
    else:
        if args.sample:
            parser.error('--sample requires a samples collection')
        output = probe(data)
    print(json.dumps(output, ensure_ascii=False, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
