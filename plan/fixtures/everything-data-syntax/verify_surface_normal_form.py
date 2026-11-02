"""Bounded, synthetic tests for desugaring four authoring surfaces to plain data.

No general Markdown/Goa/Graphviz/TEI parser; no target execution, acceptance,
or permission authority. The grammar is deliberately limited to this fixture.
"""
import hashlib
import json
import re
import unittest
from dataclasses import dataclass

ROLE = 'function'
FUNC = 'sales.confirmed_total'
BINDINGS = {'orders': 'sales.orders@R1', 'eligible': 'eligible'}
OPERATORS = {'where': 'tei.where@v1', 'sum': 'tei.sum@v1'}


class Unresolved(ValueError):
    pass


def candidate(source, steps):
    """A plain-data dict, *not* a second authoritative TEI semantic model."""
    if source != FUNC:
        raise Unresolved('unknown-function')
    if len(steps) != 2 or [s[0] for s in steps] != ['eligible', 'total']:
        raise Unresolved('unresolved-step-identity-or-order')
    a, b = steps
    if a[1:] != ('where', 'orders', 'status', 'confirmed'):
        raise Unresolved('unknown-predicate-or-binding')
    if b[1:] != ('sum', 'eligible', 'amount', ''):
        raise Unresolved('unknown-aggregation-or-binding')
    return {
        'role': ROLE, 'id': FUNC, 'result': {'ref': 'total'},
        'steps': [
            {'id': 'eligible', 'op': OPERATORS['where'], 'args': [
                {'ref': BINDINGS['orders']}, {'path': 'status'}, {'literal': 'confirmed'}]},
            {'id': 'total', 'op': OPERATORS['sum'], 'args': [
                {'ref': BINDINGS['eligible']}, {'path': 'amount'}]},
        ],
    }


def parse_markdown(text):
    lines = [x.strip() for x in text.strip().splitlines()]
    if not lines or lines[0] != '# function: sales.confirmed_total':
        raise Unresolved('no-explicit-role')
    # Header and delimiter are exact; explanation tables do not match.
    if lines[1:3] != ['| step | op | source | parameters |',
                       '| --- | --- | --- | --- |']:
        raise Unresolved('not-declared-function-table')
    rows = []
    for line in lines[3:]:
        cols = [c.strip() for c in line.strip('|').split('|')]
        if len(cols) != 4:
            raise Unresolved('unsupported-markdown-cell')
        p = cols[3].split('=') if cols[1] == 'where' else [cols[3]]
        if cols[1] == 'where' and len(p) == 2:
            rows.append((cols[0], cols[1], cols[2], p[0], p[1]))
        else:
            rows.append((cols[0], cols[1], cols[2], cols[3], ''))
    return candidate(FUNC, rows)


def parse_indent(text):
    lines = [line.rstrip() for line in text.strip().splitlines()]
    if lines[0] != 'function sales.confirmed_total':
        raise Unresolved('no-explicit-role')
    steps = []
    for line in lines[1:]:
        m = re.fullmatch(r'  (\w+): (where|sum) (\w+) (\w+)(?: (\w+))?', line)
        if not m:
            raise Unresolved('invalid-indent-or-token')
        name, op, source, arg, value = m.groups()
        steps.append((name, op, source, arg, value or ''))
    return candidate(FUNC, steps)


def parse_chain(text):
    m = re.fullmatch(r'function ([\w.]+) = (\w+)\.where\((\w+)==(\w+)\)\.sum\((\w+)\)', text.strip())
    if not m:
        raise Unresolved('unsupported-chain-or-effect-order')
    func, source, key, val, field = m.groups()
    return candidate(func, [
        ('eligible', 'where', source, key, val),
        ('total', 'sum', 'eligible', field, ''),
    ])


def parse_json(text):
    try:
        obj = json.loads(text)
    except json.JSONDecodeError as exc:
        raise Unresolved('invalid-json') from exc
    if not isinstance(obj, dict) or set(obj) != {'role', 'id', 'steps', 'result'}:
        raise Unresolved('invalid-plain-data-keys')
    if obj.get('role') != ROLE or obj.get('id') != FUNC:
        raise Unresolved('unresolved-role-or-identity')
    if obj.get('result') != {'ref': 'total'}:
        raise Unresolved('unresolved-result')
    # Strict comparisons bind operator identities/arguments; no invented coercion.
    if obj != candidate(FUNC, [
        ('eligible', 'where', 'orders', 'status', 'confirmed'),
        ('total', 'sum', 'eligible', 'amount', ''),
    ]):
        raise Unresolved('not-equivalent-under-fixture-contract')
    return obj


def parse_dot(text):
    """VERY restricted typed-DOT specimen, not a general Graphviz parser.

    Require semantic graph identity, node-wise typed attributes and edge
    direction; visual properties are ignored. Reject labels-only diagrams.
    """
    if not re.search(r'graph\s*\[semantic_id="sales\.confirmed_total"\]', text):
        raise Unresolved('diagram-lacks-semantic-contract')
    node_attrs = {}
    for node in ('eligible', 'total'):
        matches = re.findall(r'\b' + node + r'\s*\[([^\]]+)\]', text)
        if len(matches) != 1:
            raise Unresolved('diagram-lacks-semantic-contract')
        fields = dict(re.findall(r'(\w+)="([^"]*)"', matches[0]))
        node_attrs[node] = fields
    required = {
        'eligible': {'op': 'tei.where@v1', 'ref': 'sales.orders@R1',
                     'predicate': 'status=confirmed'},
        'total': {'op': 'tei.sum@v1', 'field': 'amount'},
    }
    for node, expected in required.items():
        if not all(node_attrs[node].get(k) == v for k, v in expected.items()):
            raise Unresolved('diagram-lacks-semantic-contract')
        # Neither an undeclared semantic key nor a substitute effect is harmless.
        if set(node_attrs[node]) - set(expected) - {'color', 'shape', 'pos', 'label'}:
            raise Unresolved('unsupported-dot-attribute')
    edges = re.findall(r'\b(\w+)\s*->\s*(\w+)\s*;', text)
    if edges != [('eligible', 'total')]:
        raise Unresolved('diagram-edge-binding-mismatch')
    return candidate(FUNC, [
        ('eligible', 'where', 'orders', 'status', 'confirmed'),
        ('total', 'sum', 'eligible', 'amount', ''),
    ])


def meaning_digest(data):
    """Canonical candidate bytes exclude comments/layout/provenance by design."""
    return hashlib.sha256(json.dumps(data, ensure_ascii=False, sort_keys=True,
                                     separators=(',', ':')).encode('utf-8')).hexdigest()


MD = '''# function: sales.confirmed_total
| step | op | source | parameters |
| --- | --- | --- | --- |
| eligible | where | orders | status=confirmed |
| total | sum | eligible | amount |'''
IND = '''function sales.confirmed_total
  eligible: where orders status confirmed
  total: sum eligible amount'''
CHAIN = 'function sales.confirmed_total = orders.where(status==confirmed).sum(amount)'
DOT = '''digraph {
 graph [semantic_id="sales.confirmed_total"];
 eligible [op="tei.where@v1", ref="sales.orders@R1", predicate="status=confirmed"];
 total [op="tei.sum@v1", field="amount"];
 eligible -> total;
}'''


class SurfaceNormalizationTests(unittest.TestCase):
    def setUp(self):
        self.truth = candidate(FUNC, [
            ('eligible', 'where', 'orders', 'status', 'confirmed'),
            ('total', 'sum', 'eligible', 'amount', ''),
        ])

    def test_01_three_readable_surfaces_same_plain_data(self):
        self.assertEqual(parse_markdown(MD), parse_indent(IND))
        self.assertEqual(parse_indent(IND), parse_chain(CHAIN))

    def test_02_json_structured_surface_same_data(self):
        self.assertEqual(parse_json(json.dumps(self.truth)), self.truth)

    def test_03_meaning_digest_stable_across_four_surfaces(self):
        values = [parse_markdown(MD), parse_indent(IND), parse_chain(CHAIN),
                  parse_json(json.dumps(self.truth, indent=2))]
        self.assertEqual(len({meaning_digest(v) for v in values}), 1)

    def test_04_same_meaning_not_same_source(self):
        self.assertNotEqual(hashlib.sha256(MD.encode()).hexdigest(),
                            hashlib.sha256(CHAIN.encode()).hexdigest())

    def test_05_explanatory_markdown_table_not_executable(self):
        with self.assertRaisesRegex(Unresolved, 'no-explicit-role'):
            parse_markdown(MD.replace('# function: sales.confirmed_total', '# Example'))

    def test_06_missing_predicate_rejected(self):
        with self.assertRaisesRegex(Unresolved, 'unknown-predicate'):
            parse_markdown(MD.replace('status=confirmed', 'status='))

    def test_07_unknown_operator_rejected(self):
        with self.assertRaises(Unresolved):
            parse_indent(IND.replace('sum eligible', 'distinct eligible'))

    def test_08_order_change_not_normalized_away(self):
        lines = IND.splitlines()
        with self.assertRaisesRegex(Unresolved, 'unresolved-step'):
            parse_indent('\n'.join([lines[0], lines[2], lines[1]]))

    def test_09_new_binding_not_same_meaning(self):
        with self.assertRaisesRegex(Unresolved, 'unknown-predicate'):
            parse_chain(CHAIN.replace('status==confirmed', 'kind==confirmed'))

    def test_10_label_only_dot_rejected(self):
        with self.assertRaisesRegex(Unresolved, 'diagram-lacks-semantic-contract'):
            parse_dot('digraph { orders -> filter -> total; }')

    def test_11_typed_dot_is_explicit_only_in_toy_contract(self):
        self.assertEqual(parse_dot(DOT), self.truth)

    def test_12_dot_layout_change_not_semantic_change(self):
        self.assertEqual(meaning_digest(parse_dot(DOT)),
                         meaning_digest(parse_dot(DOT.replace('eligible -> total;',
                                     'eligible -> total;\n rankdir=LR;'))))

    def test_13_indent_error_is_not_auto_fixed(self):
        with self.assertRaisesRegex(Unresolved, 'invalid-indent'):
            parse_indent(IND.replace('  total:', '    total:'))

    def test_14_json_operand_change_rejected(self):
        tampered = json.loads(json.dumps(self.truth))
        tampered['steps'][0]['args'][2]['literal'] = 'pending'
        with self.assertRaisesRegex(Unresolved, 'not-equivalent'):
            parse_json(json.dumps(tampered))

    def test_15_json_operator_version_change_rejected(self):
        tampered = json.loads(json.dumps(self.truth))
        tampered['steps'][1]['op'] = 'tei.sum@v2'
        with self.assertRaisesRegex(Unresolved, 'not-equivalent'):
            parse_json(json.dumps(tampered))

    def test_16_duplicate_step_identity_rejected(self):
        with self.assertRaisesRegex(Unresolved, 'unresolved-step'):
            parse_indent(IND.replace('  total:', '  eligible:'))

    def test_17_same_label_different_semantic_ref_rejected(self):
        tampered = json.loads(json.dumps(self.truth))
        tampered['steps'][0]['args'][0]['ref'] = 'external.orders@R1'
        with self.assertRaisesRegex(Unresolved, 'not-equivalent'):
            parse_json(json.dumps(tampered))

    def test_18_method_chain_short_circuit_effect_unmodelled(self):
        with self.assertRaisesRegex(Unresolved, 'unsupported-chain-or-effect-order'):
            parse_chain(CHAIN.replace('.sum(amount)', '.send().sum(amount)'))

    def test_19_utf8_authored_source_differs_from_normal_form(self):
        source = '# 説明：受注集計\n' + MD
        self.assertNotEqual(hashlib.sha256(source.encode()).hexdigest(), meaning_digest(self.truth))

    def test_20_dot_swapped_node_attributes_rejected(self):
        swapped = DOT.replace('eligible [op="tei.where@v1"', 'eligible [op="tei.sum@v1"')
        with self.assertRaisesRegex(Unresolved, 'diagram-lacks-semantic-contract'):
            parse_dot(swapped)

    def test_21_dot_reversed_edge_rejected(self):
        with self.assertRaisesRegex(Unresolved, 'diagram-edge-binding-mismatch'):
            parse_dot(DOT.replace('eligible -> total;', 'total -> eligible;'))

    def test_22_dot_identity_required(self):
        with self.assertRaisesRegex(Unresolved, 'diagram-lacks-semantic-contract'):
            parse_dot(DOT.replace('sales.confirmed_total', 'sales.other_total'))

    def test_23_dot_unknown_semantic_attribute_rejected(self):
        with self.assertRaisesRegex(Unresolved, 'unsupported-dot-attribute'):
            parse_dot(DOT.replace('field="amount"', 'field="amount", effect="send"'))


if __name__ == '__main__':
    unittest.main()
