/**
 * Organization split/succession: read-only, synthetic conformance pressure.
 * No TEI parser, Provider, real write, Org identity acceptance or Host authorization.
 * Standalone successor case; does not mutate the earlier org-synthetic-r7 fixture.
 * node --test reference/org-semantic-views/verify-succession.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';

const V = '2027-07-01';
const R1 = '2027-07-01';
const R2 = '2027-07-03';
const SRC = 'org-succession-toy-r1';

const source = {
  revision: SRC,
  persons: ['P1', 'P2', 'P3'],
  // An occurrence is not a Person identity; succession does not re-use the former ID.
  membershipVersions: [
    { id: 'A1', person: 'P1', dept: 'D1', validFrom: '2027-01-01', validTo: V, knownFrom: '2027-06-01', knownTo: null },
    { id: 'A2', person: 'P2', dept: 'D1', validFrom: '2027-01-01', validTo: V, knownFrom: '2027-06-01', knownTo: null },
    { id: 'A1new', person: 'P1', dept: 'D1a', validFrom: V, validTo: null, knownFrom: R1, knownTo: null },
    // R2 correction: A2new never existed at valid=V, according to knowledge at R2.
    { id: 'A2new', person: 'P2', dept: 'D1b', validFrom: V, validTo: null, knownFrom: R1, knownTo: R2 },
    { id: 'A3', person: 'P2', dept: 'D2', validFrom: '2027-01-01', validTo: null, knownFrom: '2027-06-01', knownTo: null },
    { id: 'A4', person: 'P3', dept: 'D2', validFrom: '2027-01-01', validTo: null, knownFrom: '2027-06-01', knownTo: null },
  ],
  allocationVersions: [
    { relation: 'A1', points: 100, knownFrom: '2027-06-01', knownTo: null },
    { relation: 'A2', points: 60, knownFrom: '2027-06-01', knownTo: null },
    { relation: 'A1new', points: 100, knownFrom: R1, knownTo: null },
    { relation: 'A2new', points: 60, knownFrom: R1, knownTo: R2 },
    { relation: 'A3', points: 40, knownFrom: '2027-06-01', knownTo: R2 },
    { relation: 'A3', points: 100, knownFrom: R2, knownTo: null },
    { relation: 'A4', points: 100, knownFrom: '2027-06-01', knownTo: null },
  ],
  successors: [
    { previous: 'D1', next: 'D1a', kind: 'succession-candidate' },
    { previous: 'D1', next: 'D1b', kind: 'succession-candidate' },
  ],
  externalIds: [
    { issuer: 'hr-system', code: 'C42', org: 'D1', validFrom: '2027-01-01', validTo: V },
    { issuer: 'hr-system', code: 'C42', org: 'D1b', validFrom: V, validTo: null },
  ],
};

const inWindow = (at, from, to) => from <= at && (to === null || at < to);
const active = (validAt, knownAt) => source.membershipVersions.filter(
  r => inWindow(validAt, r.validFrom, r.validTo) && inWindow(knownAt, r.knownFrom, r.knownTo)
);
function rowsAt(validAt, knownAt) {
  return active(validAt, knownAt).map(r => {
    const allocation = source.allocationVersions.filter(
      a => a.relation === r.id && inWindow(knownAt, a.knownFrom, a.knownTo)
    );
    if (allocation.length !== 1) return { ...r, state: 'unresolved-allocation' };
    return { ...r, points: allocation[0].points, state: 'resolved' };
  });
}
const ids = rows => rows.map(x => x.id).sort();
const fte = rows => rows.reduce((n, x) => n + x.points, 0);
const distinct = rows => new Set(rows.map(x => x.person)).size;
const select = (rows, departments) => rows.filter(x => departments.includes(x.dept));
function summary(validAt, knownAt) {
  const rows = rowsAt(validAt, knownAt);
  if (rows.some(r => r.state !== 'resolved')) return { state: 'unresolved' };
  const byDept = Object.fromEntries(
    [...new Set(rows.map(r => r.dept))].map(d => [d, fte(select(rows, [d]))])
  );
  return { state: 'resolved', ids: ids(rows), byDept,
    allPoints: fte(rows), persons: distinct(rows), occurrences: rows.length };
}

// A Bridge is an explicit TEST-ONLY owner decision, not permission, identity or authorization.
const bridge = {
  kind: 'accepted-comparison-only', revision: 'bridge-r1', sourceRevision: SRC,
  from: ['D1'], to: ['D1a', 'D1b'], measure: 'fte-points',
  grain: 'membership-allocation', scope: 'all-memberships',
};
function compareFte(from, to, b, observations) {
  if (!b || b.kind !== 'accepted-comparison-only' || b.sourceRevision !== SRC ||
    b.measure !== 'fte-points' || b.grain !== 'membership-allocation' ||
    b.scope !== 'all-memberships' || b.from.join(',') !== 'D1' ||
    b.to.join(',') !== 'D1a,D1b') return { state: 'unresolved', reason: 'unaccepted-or-incompatible-bridge' };
  if (!observations || !observations.every(o => o.complete === true && o.cohort === 'witnessed'))
    return { state: 'unverified', reason: 'physical-read-completeness-or-cohort' };
  if ([...from, ...to].some(r => r.state !== 'resolved'))
    return { state: 'unresolved', reason: 'source-values' };
  return { state: 'comparable', old: fte(select(from, b.from)), successor: fte(select(to, b.to)) };
}

const old = rowsAt('2027-06-30', R1);
const newR1 = rowsAt(V, R1);
const newR2 = rowsAt(V, R2);
const witnessed = [{ complete: true, cohort: 'witnessed' }];

test('OC-01 former/new org are distinct despite a recorded successor relationship', () => {
  assert.deepEqual(source.successors.map(x => x.next), ['D1a', 'D1b']);
  assert.notEqual('D1', source.successors[0].next);
  assert.equal(source.successors[0].kind, 'succession-candidate');
});
test('OC-02 old-time cut has four occurrences and distinct three Persons', () => {
  assert.deepEqual(ids(old), ['A1', 'A2', 'A3', 'A4']);
  assert.equal(distinct(old), 3); assert.equal(fte(select(old, ['D1'])), 160);
  assert.equal(fte(select(old, ['D2'])), 140);
});
test('OC-03 R1 published split has consistent new org ID occurrences', () => {
  const s = summary(V, R1);
  assert.deepEqual(s.ids, ['A1new', 'A2new', 'A3', 'A4']);
  assert.deepEqual(s.byDept, { D1a: 100, D1b: 60, D2: 140 });
  assert.equal(s.persons, 3); assert.equal(s.occurrences, 4);
  assert.equal(s.allPoints, 300);
});
test('OC-04 same validAt with later knownAt retroactively removes A2new', () => {
  const s = summary(V, R2);
  assert.deepEqual(s.ids, ['A1new', 'A3', 'A4']);
  assert.deepEqual(s.byDept, { D1a: 100, D2: 200 });
  assert.equal(s.persons, 3); assert.equal(s.occurrences, 3);
  assert.equal(s.allPoints, 300);
});
test('OC-05 published R1 reproduction differs from current R2 restatement', () => {
  const publication = { validAt: V, knownAt: R1, revision: 'report-r1',
    report: { successorFte: 160, allFte: 300 } };
  assert.equal(publication.report.successorFte,
    fte(select(rowsAt(publication.validAt, publication.knownAt), ['D1a', 'D1b'])));
  assert.equal(fte(select(rowsAt(publication.validAt, R2), ['D1a', 'D1b'])), 100);
  assert.notEqual(publication.report.successorFte, 100);
});
test('OC-06 explicit Owner Bridge absent => comparison unresolved', () => {
  assert.equal(compareFte(old, newR1, null, witnessed).state, 'unresolved');
});
test('OC-07 accepted Bridge compares only declared FTE grain', () => {
  assert.deepEqual(compareFte(old, newR1, bridge, witnessed),
    { state: 'comparable', old: 160, successor: 160 });
  assert.deepEqual(compareFte(old, newR2, bridge, witnessed),
    { state: 'comparable', old: 160, successor: 100 });
});
test('OC-08 Bridge for headcount cannot be silently reused for FTE', () => {
  assert.equal(compareFte(old, newR1, { ...bridge, measure: 'persons' }, witnessed).state, 'unresolved');
  assert.equal(compareFte(old, newR1, { ...bridge, sourceRevision: 'other' }, witnessed).state, 'unresolved');
});
test('OC-09 incomplete/unknown acquisition cannot attest comparison', () => {
  assert.equal(compareFte(old, newR1, bridge, [{ complete: false, cohort: 'witnessed' }]).state, 'unverified');
  assert.equal(compareFte(old, newR1, bridge, [{ complete: true, cohort: 'unknown' }]).state, 'unverified');
});
test('OC-10 the same P2 in two departments does not make four people', () => {
  assert.equal(select(old, ['D1']).length + select(old, ['D2']).length, 4);
  assert.equal(distinct(old), 3);
  assert.equal(distinct(select(newR1, ['D1a', 'D1b'])), 2);
  assert.equal(distinct(select(newR2, ['D1a', 'D1b'])), 1);
});
test('OC-11 JOIN fanout is not a change to member occurrence or allocated FTE', () => {
  const member = select(newR1, ['D1a']);
  const joined = member.flatMap(row => ['J1', 'J2'].map(project => ({ member: row.id, project })));
  assert.equal(joined.length, 2);
  assert.equal(member.length, 1); assert.equal(fte(member), 100);
});
test('OC-12 reused external HR code never proves identical TEI Org ID', () => {
  const early = source.externalIds.find(r => inWindow('2027-06-30', r.validFrom, r.validTo));
  const later = source.externalIds.find(r => inWindow(V, r.validFrom, r.validTo));
  assert.equal(early.code, later.code);
  assert.notEqual(early.org, later.org);
});
test('OC-13 successor relation never supplies grant or decision-right', () => {
  const rights = source.successors.flatMap(x => Object.keys(x).filter(k => /grant|right|authority/i.test(k)));
  assert.deepEqual(rights, []);
});
test('OC-14 old selection A2 occurrence is not new A2new write target', () => {
  assert.equal(old.find(x => x.id === 'A2').person, 'P2');
  assert.equal(newR1.some(x => x.id === 'A2'), false);
  assert.equal(newR1.some(x => x.id === 'A2new'), true);
  assert.equal(newR2.some(x => x.id === 'A2new'), false);
});
test('OC-15 missing allocation is unresolved, not zero', () => {
  const original = source.allocationVersions;
  source.allocationVersions = original.filter(x => x.relation !== 'A1new');
  try { assert.equal(summary(V, R1).state, 'unresolved'); }
  finally { source.allocationVersions = original; }
});
test('OC-16 validAt cut is not knownAt cut', () => {
  assert.deepEqual(ids(rowsAt('2027-06-30', R2)), ['A1', 'A2', 'A3', 'A4']);
  assert.deepEqual(ids(rowsAt(V, R1)), ['A1new', 'A2new', 'A3', 'A4']);
});


/* ORG-RC: in-flight Case responsibility is NOT an Org successor, Grant or Effect.
 * Test-only policy flags and read witnesses: not accepted TEI business rules,
 * not actual Provider reads, and not a transaction/Host test.
 */
const Q1 = 'Q1';
const caseResponsibility = [
  { id: 'QR-old', caseId: Q1, org: 'D1', validFrom: '2027-06-01', validTo: V, knownFrom: '2027-06-01', knownTo: null },
  { id: 'QR-r1', caseId: Q1, org: 'D1a', validFrom: V, validTo: null, knownFrom: R1, knownTo: R2 },
  { id: 'QR-r2', caseId: Q1, org: 'D1b', validFrom: V, validTo: null, knownFrom: R2, knownTo: null },
];
const caseRowsAt = (validAt, knownAt) => caseResponsibility.filter(
  r => inWindow(validAt, r.validFrom, r.validTo) && inWindow(knownAt, r.knownFrom, r.knownTo)
);
const singleRule = Object.freeze({ revision: 'responsibility-r1', mode: 'single', acceptedInFixture: true });
const jointRule = Object.freeze({ revision: 'responsibility-r2', mode: 'joint', acceptedInFixture: true });
const closedCaseWitness = Object.freeze({ relationComplete: true, cohort: 'witnessed' });

// Demonstration only. Existence of two conflicting assignments is a positive
// witness even if a later Page is partial; a claim of zero requires closure.
function coverage({ rule, rows, witness }) {
  if (!rule?.acceptedInFixture) return { state: 'unresolved', reason: 'rule-not-accepted' };
  if (!Array.isArray(rows) || !rows.every(x => x.caseId === Q1))
    return { state: 'unresolved', reason: 'unqualified-case-scope' };
  const owners = [...new Set(rows.map(x => x.org))];
  if (rule.mode === 'single' && owners.length > 1)
    return { state: 'violated', reason: 'multiple-owners-observed', owners };
  if (!witness || witness.relationComplete !== true || witness.cohort !== 'witnessed')
    return { state: 'unverified', reason: 'relation-collection-not-established' };
  if (owners.length === 0) return { state: 'unassigned-for-scope', owners };
  if (rule.mode === 'single' && owners.length === 1)
    return { state: 'established-for-scope', owners };
  if (rule.mode === 'joint' && owners.length === 2)
    return { state: 'established-for-scope', owners };
  return { state: 'unresolved', reason: 'policy-cardinality', owners };
}
function pendingCaseOwner({ mode, opened, current }) {
  if (mode === 'freeze-on-application') return { state: 'candidate', owner: opened[0]?.org };
  if (mode === 'resolve-on-approval') return { state: 'candidate', owner: current[0]?.org };
  return { state: 'unresolved', reason: 'pending-case-policy-not-selected' };
}
function attemptedCommit({ checkedOwner, postOwner, hasAtomicFence }) {
  if (!hasAtomicFence) return 'guarantee-unsupported';
  return checkedOwner === postOwner ? 'candidate-only' : 'blocked-by-post-image';
}
function approvalCandidate({ subjectIsApplicant, grantCurrent, delegationScopeValid }) {
  return (subjectIsApplicant || !grantCurrent || !delegationScopeValid) ? 'denied-in-fixture' : 'candidate-only';
}
function assessAllCases({ casePopulationComplete, assessedCases, expectedCases }) {
  if (!casePopulationComplete) return 'unverified';
  if (assessedCases !== expectedCases) return 'unverified';
  return 'established-for-fixture-only';
}

test('RC-01 successor relation is not case migration without its own rule', () => {
  assert.equal(source.successors.length, 2);
  assert.equal(coverage({ rule: null, rows: [], witness: closedCaseWitness }).state, 'unresolved');
});
test('RC-02 explicit Q1 to D1a is only a responsibility claim for R1', () => {
  assert.deepEqual(coverage({ rule: singleRule, rows: caseRowsAt(V, R1), witness: closedCaseWitness }),
    { state: 'established-for-scope', owners: ['D1a'] });
});
test('RC-03 two witnessed owners violate the case-specific single-owner rule', () => {
  const rows = [...caseRowsAt(V, R1), { id: 'QR-extra', caseId: Q1, org: 'D1b' }];
  assert.equal(coverage({ rule: singleRule, rows, witness: { relationComplete: false, cohort: 'unknown' } }).state, 'violated');
});
test('RC-04 the same two owners fit the fixture joint-responsibility rule', () => {
  const rows = [...caseRowsAt(V, R1), { id: 'QR-extra', caseId: Q1, org: 'D1b' }];
  assert.deepEqual(coverage({ rule: jointRule, rows, witness: closedCaseWitness }),
    { state: 'established-for-scope', owners: ['D1a', 'D1b'] });
});
test('RC-05 zero rows from partial or unknown cohort does not prove unassigned', () => {
  assert.equal(coverage({ rule: singleRule, rows: [], witness: { relationComplete: false, cohort: 'witnessed' } }).state, 'unverified');
  assert.equal(coverage({ rule: singleRule, rows: [], witness: closedCaseWitness }).state, 'unassigned-for-scope');
});
test('RC-06 pending-case freeze vs current resolution is an explicit policy choice', () => {
  assert.equal(pendingCaseOwner({ mode: null, opened: caseRowsAt('2027-06-30', R1), current: caseRowsAt(V, R1) }).state, 'unresolved');
  assert.equal(pendingCaseOwner({ mode: 'freeze-on-application', opened: caseRowsAt('2027-06-30', R1), current: caseRowsAt(V, R1) }).owner, 'D1');
  assert.equal(pendingCaseOwner({ mode: 'resolve-on-approval', opened: caseRowsAt('2027-06-30', R1), current: caseRowsAt(V, R1) }).owner, 'D1a');
});
test('RC-07 R1 publication preserves its historical owner against R2 correction', () => {
  const published = { revision: 'Q1-published-R1', validAt: V, knownAt: R1, owner: caseRowsAt(V, R1)[0].org };
  assert.equal(published.owner, 'D1a');
  assert.equal(caseRowsAt(V, R2)[0].org, 'D1b');
  assert.notEqual(published.owner, caseRowsAt(V, R2)[0].org);
});
test('RC-08 checked owner is not a final-post-image invariant without fence', () => {
  assert.equal(attemptedCommit({ checkedOwner: 'D1a', postOwner: 'D1b', hasAtomicFence: true }), 'blocked-by-post-image');
  assert.equal(attemptedCommit({ checkedOwner: 'D1a', postOwner: 'D1b', hasAtomicFence: false }), 'guarantee-unsupported');
  assert.equal(attemptedCommit({ checkedOwner: 'D1a', postOwner: 'D1a', hasAtomicFence: true }), 'candidate-only');
});
test('RC-09 a displayed approver candidate is neither SoD nor current Grant', () => {
  assert.equal(approvalCandidate({ subjectIsApplicant: true, grantCurrent: true, delegationScopeValid: true }), 'denied-in-fixture');
  assert.equal(approvalCandidate({ subjectIsApplicant: false, grantCurrent: false, delegationScopeValid: true }), 'denied-in-fixture');
  assert.equal(approvalCandidate({ subjectIsApplicant: false, grantCurrent: true, delegationScopeValid: true }), 'candidate-only');
});
test('RC-10 an incomplete population cannot support an all-cases transferred claim', () => {
  assert.equal(assessAllCases({ casePopulationComplete: false, assessedCases: 3, expectedCases: 3 }), 'unverified');
  assert.equal(assessAllCases({ casePopulationComplete: true, assessedCases: 2, expectedCases: 3 }), 'unverified');
  assert.equal(assessAllCases({ casePopulationComplete: true, assessedCases: 3, expectedCases: 3 }), 'established-for-fixture-only');
});
