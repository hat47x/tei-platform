/**
 * Read-only cross-consumer semantic projection reference for TEI organization data.
 * These functions are fixture-local, not production TEI Modeler/UI/Function contracts.
 * Run: node --test reference/org-semantic-views/verify-projection.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';

const source = {
  sourceVersion: 'org-synthetic-r7',
  persons: ['P1', 'P2', 'P3'],
  departments: ['D1', 'D2'],
  memberships: [
    { id: 'A1', person: 'P1', dept: 'D1', provenance: 'direct' },
    { id: 'A2', person: 'P2', dept: 'D1', provenance: 'direct' },
    { id: 'A3', person: 'P2', dept: 'D2', provenance: 'direct' },
    { id: 'A4', person: 'P3', dept: 'D2', provenance: 'direct' },
  ],
  allocationBasisPoints: { A1: 100, A2: 60, A3: 40, A4: 100 },
  position: { id: 'M1', dept: 'D1' },
  occupancy: [
    { id: 'O1', person: 'P2', position: 'M1', from: '2027-04-01', to: '2027-07-01' },
    { id: 'O2', person: 'P3', position: 'M1', from: '2027-07-01', to: null },
  ],
};
const evidence = (grain, claim) => ({
  source: source.sourceVersion, scope: 'synthetic-all', complete: true,
  grain, claim, status: 'read-only-projection',
});
function tableView() {
  return {
    evidence: evidence('Membership occurrence', 'direct memberships'),
    rows: source.memberships.map(m => ({
      viewId: 'table:' + m.id, subject: m.person, relation: m.id,
      department: m.dept, provenance: m.provenance,
    })),
  };
}
function graphView() {
  return {
    evidence: evidence('Subject and Relation occurrence', 'relationship network'),
    nodes: [
      ...source.persons.map(p => ({ viewId: 'person:' + p, subject: p, kind: 'Person' })),
      ...source.departments.map(d => ({ viewId: 'department:' + d, subject: d, kind: 'OrgUnit' })),
    ],
    edges: source.memberships.map(m => ({
      viewId: 'graph:' + m.id, relation: m.id, from: m.person, to: m.dept,
      role: 'Membership', provenance: m.provenance,
    })),
  };
}
// A forest-style view; a projection can occur multiple times for one Person.
function treeView({ collapseDuplicatePerson = false } = {}) {
  const seen = new Set(), hiddenRelations = [], leaves = [];
  for (const m of source.memberships) {
    if (collapseDuplicatePerson && seen.has(m.person)) {
      hiddenRelations.push(m.id);
      continue;
    }
    seen.add(m.person);
    leaves.push({
      viewId: 'tree:' + m.dept + ':' + m.id,
      subject: m.person, relation: m.id, parent: m.dept,
    });
  }
  return {
    evidence: evidence('Membership occurrence', 'department-member forest'),
    leaves, hiddenRelations,
    projectionIsLossy: hiddenRelations.length > 0,
    editMode: hiddenRelations.length ? 'read-only' : 'candidate-only',
  };
}
function chartView() {
  const groups = source.departments.map(dept => {
    const edges = source.memberships.filter(x => x.dept === dept);
    return {
      dimension: dept, grain: 'department',
      membershipOccurrences: edges.length,
      distinctPersons: new Set(edges.map(x => x.person)).size,
      allocationBasisPoints: edges.reduce((sum, m) =>
        sum + source.allocationBasisPoints[m.id], 0),
      provenance: edges.map(x => x.id),
    };
  });
  return {
    evidence: evidence('group and distinct Person', 'membership/people/FTE measure comparison'),
    groups,
    globalDistinctPersons: new Set(source.memberships.map(x => x.person)).size,
  };
}
function occupantsAt(date) {
  return source.occupancy
    .filter(o => o.from <= date && (o.to === null || date < o.to))
    .map(o => ({ position: o.position, person: o.person, relation: o.id }));
}
// Interpreting a *selection* is read-only and requires the original edge ID.
function selectionCandidate(viewOccurrence) {
  const kind = viewOccurrence.relation ? 'Membership' :
    viewOccurrence.subject ? 'Subject' : 'Unknown';
  return kind === 'Membership'
    ? { status: 'candidate', operation: 'select-relation', target: viewOccurrence.relation }
    : kind === 'Subject'
      ? { status: 'candidate', operation: 'select-subject', target: viewOccurrence.subject }
      : { status: 'unresolved', reason: 'missing-identity' };
}

test('table and graph preserve four distinct relationship occurrences', () => {
  assert.deepEqual(tableView().rows.map(x => x.relation), ['A1','A2','A3','A4']);
  assert.deepEqual(graphView().edges.map(x => x.relation), ['A1','A2','A3','A4']);
});
test('graph nodes preserve Person identity while P2 has two independent edges', () => {
  assert.equal(graphView().nodes.filter(x => x.kind === 'Person').length, 3);
  assert.deepEqual(graphView().edges.filter(x => x.from === 'P2').map(x => x.relation),
    ['A2','A3']);
});
test('table rows are display occurrences and are not Person IDs', () => {
  const rows = tableView().rows.filter(x => x.subject === 'P2');
  assert.deepEqual(rows.map(x => x.viewId), ['table:A2','table:A3']);
  assert.equal(rows[0].subject, rows[1].subject);
});
test('tree displays P2 twice without creating another Person', () => {
  const tree = treeView();
  assert.deepEqual(tree.leaves.filter(x => x.subject === 'P2').map(x => x.relation),
    ['A2','A3']);
  assert.equal(new Set(tree.leaves.map(x => x.subject)).size, 3);
});
test('collapsed tree advertises lost A3 and becomes read-only', () => {
  const tree = treeView({ collapseDuplicatePerson: true });
  assert.deepEqual(tree.hiddenRelations, ['A3']);
  assert.equal(tree.projectionIsLossy, true);
  assert.equal(tree.editMode, 'read-only');
  assert.equal(source.memberships.length, 4);
});
test('chart keeps membership count separate from global distinct Persons', () => {
  const chart = chartView();
  assert.deepEqual(chart.groups.map(x => x.membershipOccurrences), [2,2]);
  assert.equal(chart.groups.reduce((n,x)=>n+x.distinctPersons,0), 4);
  assert.equal(chart.globalDistinctPersons,3);
});
test('explicit FTE allocation is separate from structure-derived headcount', () => {
  const groups = chartView().groups;
  assert.deepEqual(groups.map(x => x.allocationBasisPoints), [160,140]);
  assert.equal(groups.reduce((n,x)=>n+x.allocationBasisPoints,0), 300);
});
test('chart carries supporting relation identifiers, not an aggregate update target', () => {
  const group = chartView().groups[1];
  assert.deepEqual(group.provenance, ['A3','A4']);
  assert.equal('relation' in group,false);
});
test('table and graph A3 selections identify the same relation not the Person', () => {
  const tableA3 = tableView().rows.find(x => x.relation === 'A3');
  const graphA3 = graphView().edges.find(x => x.relation === 'A3');
  assert.deepEqual(selectionCandidate(tableA3),selectionCandidate(graphA3));
  assert.deepEqual(selectionCandidate(tableA3),
    {status:'candidate',operation:'select-relation',target:'A3'});
});
test('graph Person P2 selection is not the same as A3 relation selection', () => {
  const node = graphView().nodes.find(x => x.subject === 'P2');
  const edge = graphView().edges.find(x => x.relation === 'A3');
  assert.notDeepEqual(selectionCandidate(node),selectionCandidate(edge));
});
test('M1 occupancy switches without adding a direct D1 membership for P3', () => {
  assert.deepEqual(occupantsAt('2027-06-30').map(x => x.person), ['P2']);
  assert.deepEqual(occupantsAt('2027-07-01').map(x => x.person), ['P3']);
  assert.equal(source.memberships.some(m => m.person === 'P3' && m.dept === 'D1'),false);
});
test('all views tie their interpretation to the same Source revision', () => {
  assert.deepEqual(
    [tableView(),graphView(),treeView(),chartView()].map(v => v.evidence.source),
    Array(4).fill('org-synthetic-r7'));
});
test('view creation leaves the underlying source facts unmodified', () => {
  const old = JSON.stringify(source);
  tableView();graphView();treeView({collapseDuplicatePerson:true});chartView();
  assert.equal(JSON.stringify(source),old);
});
