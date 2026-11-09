/**
 * Read-only cross-consumer semantic projection reference for TEI organization data.
 * These functions are fixture-local, not production TEI Modeler/UI/Function contracts.
 * Run: node --test reference/org-semantic-views/verify-projection.mjs
 */
import { fixture, baselineMemberships } from './source-fixture.mjs';
import test from 'node:test';
import assert from 'node:assert/strict';

const source = {
  sourceVersion: fixture.sourceRevision,
  persons: fixture.persons,
  departments: fixture.departments.map(d => d.id),
  memberships: baselineMemberships(fixture).map(({id,person,dept,provenance})=>({id,person,dept,provenance})),
  allocationBasisPoints: Object.fromEntries(fixture.allocations.map(a => [a.relation,a.basisPoints])),
  position: fixture.position,
  occupancy: fixture.occupancy,
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
function treeView({ collapseDuplicatePerson = false, preferredRelationByPerson = {},
                    membershipRows = source.memberships } = {}) {
  const counts = new Map(), hiddenRelations = [], unresolvedSubjects = [], leaves = [];
  for (const m of membershipRows) counts.set(m.person, (counts.get(m.person) || 0) + 1);
  for (const m of membershipRows) {
    if (collapseDuplicatePerson && counts.get(m.person) > 1) {
      const chosen = preferredRelationByPerson[m.person];
      const validChoice = membershipRows.some(x => x.person === m.person && x.id === chosen);
      if (!validChoice) {
        if (!unresolvedSubjects.includes(m.person)) unresolvedSubjects.push(m.person);
        // Ambiguity: show all source edges, rather than choosing a parent by row order.
      } else if (m.id !== chosen) {
        hiddenRelations.push(m.id);
        continue;
      }
    }
    leaves.push({
      viewId: 'tree:' + m.dept + ':' + m.id,
      subject: m.person, relation: m.id, parent: m.dept,
    });
  }
  return {
    evidence: evidence('Membership occurrence', 'department-member forest'),
    leaves, hiddenRelations, unresolvedSubjects,
    projectionIsLossy: hiddenRelations.length > 0,
    editMode: collapseDuplicatePerson ? 'read-only' : 'candidate-only',
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
test('collapsed tree requires an explicit view preference, not source row order', () => {
  const tree = treeView({ collapseDuplicatePerson: true });
  assert.deepEqual(tree.unresolvedSubjects, ['P2']);
  assert.deepEqual(tree.hiddenRelations, []);
  assert.deepEqual(tree.leaves.filter(x => x.subject === 'P2').map(x => x.relation), ['A2','A3']);
  assert.equal(tree.editMode, 'read-only');
});
test('explicitly preferred A2 can hide A3 but reports loss and remains read-only', () => {
  const tree = treeView({
    collapseDuplicatePerson: true, preferredRelationByPerson: { P2: 'A2' },
  });
  assert.deepEqual(tree.hiddenRelations, ['A3']);
  assert.deepEqual(tree.unresolvedSubjects, []);
  assert.equal(tree.projectionIsLossy, true);
  assert.equal(tree.editMode, 'read-only');
  assert.equal(source.memberships.length, 4);
});
test('valid preferred relation remains stable under source row reordering', () => {
  const opts = { collapseDuplicatePerson: true, preferredRelationByPerson: { P2: 'A2' } };
  const original = treeView(opts);
  const reordered = treeView({ ...opts, membershipRows: [...source.memberships].reverse() });
  assert.deepEqual(original.hiddenRelations, reordered.hiddenRelations);
  assert.deepEqual(original.leaves.map(x => x.relation).sort(),
    reordered.leaves.map(x => x.relation).sort());
});
test('unknown preferred relation leaves ambiguity visible', () => {
  const tree = treeView({
    collapseDuplicatePerson: true, preferredRelationByPerson: { P2: 'A99' },
  });
  assert.deepEqual(tree.unresolvedSubjects, ['P2']);
  assert.deepEqual(tree.hiddenRelations, []);
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
