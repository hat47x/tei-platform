/**
 * Synthetic, read-only counterexample to unconditional org-view cache reuse.
 * Run: node --test reference/org-semantic-views/verify-change.mjs
 * No TEI adapter, database, production Rule, authorization or actual refresh.
 */
import { fixture, scenario } from './source-fixture.mjs';
import test from 'node:test';
import assert from 'node:assert/strict';

const seed = {
  people: fixture.persons,
  departments: Object.fromEntries(fixture.departments.map(d => [d.id, d.label])),
  versions: fixture.membershipVersions,
  fteBasisPoints: Object.fromEntries(fixture.allocations.map(a => [a.relation, a.basisPoints])),
  occupancy: fixture.occupancy,
};

const within = (point, from, to) => from <= point && (to === null || point < to);
const relationsAt = (s, validAt, knownAt) => s.versions.filter(x =>
  within(validAt, x.validFrom, x.validTo) && within(knownAt, x.knownFrom, x.knownTo));
const distinctCount = (xs, key) => new Set(xs.map(key)).size;
function claims(s, validAt, knownAt) {
  const r = relationsAt(s, validAt, knownAt);
  const count = dept => r.filter(x => x.dept === dept).length;
  const fte = dept => r.filter(x => x.dept === dept)
    .reduce((total, x) => total + s.fteBasisPoints[x.id], 0);
  const incumbent = s.occupancy.filter(x => x.position === 'M1' && within(validAt, x.from, x.to))
    .map(x => x.person).sort();
  return { members: r.length, people: distinctCount(r, x => x.person),
    d1Members: count('D1'), d2Members: count('D2'),
    d1Fte: fte('D1'), d2Fte: fte('D2'), incumbent };
}
const query = (s, at = scenario.baseline.validAt, known = scenario.baseline.knownAt) => claims(s, at, known);

test('baseline: three people but four membership occurrences', () => {
  assert.deepEqual(query(seed), { members: 4, people: 3, d1Members: 2, d2Members: 2,
    d1Fte: 160, d2Fte: 140, incumbent: ['P2'] });
});
test('display rename leaves identity, membership, counts and FTE unchanged', () => {
  const renamed = { ...seed, departments: { ...seed.departments, D2: 'Beta renamed' } };
  assert.deepEqual(query(renamed), query(seed));
  assert.notEqual(renamed.departments.D2, seed.departments.D2);
});
test('membership source order does not alter claims', () => {
  assert.deepEqual(query({ ...seed, versions: [...seed.versions].reverse() }), query(seed));
});
test('retroactive correction remains invisible at previous known-at cutoff', () => {
  assert.equal(query(seed, '2027-06-15', '2027-07-01').members, 4);
  assert.equal(query(seed, '2027-06-15', '2027-07-03').members, 3);
});
test('retroactive correction changes D1 membership and FTE, not global Person identity', () => {
  const before = query(seed, '2027-06-15', '2027-07-01');
  const after = query(seed, '2027-06-15', '2027-07-03');
  assert.equal(before.people, after.people);
  assert.equal(after.d1Members, 1);
  assert.equal(after.d1Fte, 100);
  assert.equal(after.d2Members, 2);
  assert.equal(after.d2Fte, 140);
});
test('valid-time earlier than corrected end retains membership', () => {
  assert.equal(query(seed, '2027-05-31', '2027-07-03').members, 4);
});
test('occupancy switches on half-open boundary but does not create direct membership', () => {
  assert.deepEqual(query(seed, '2027-06-30', '2027-07-03').incumbent, ['P2']);
  assert.deepEqual(query(seed, '2027-07-01', '2027-07-03').incumbent, ['P3']);
  assert.equal(relationsAt(seed, '2027-07-01', '2027-07-03')
    .some(x => x.person === 'P3' && x.dept === 'D1'), false);
});
test('allocation-only update changes FTE but not person and membership counts', () => {
  const changed = { ...seed, fteBasisPoints: { ...seed.fteBasisPoints, A2: 40, A3: 60 } };
  assert.equal(query(changed).d1Fte, 140);
  assert.equal(query(changed).d2Fte, 160);
  assert.equal(query(changed).members, query(seed).members);
  assert.equal(query(changed).people, query(seed).people);
});
test('incomplete slice supports observed existence but not universal absence or full count', () => {
  const slice = { rows: relationsAt(seed, '2027-06-15', '2027-07-01').slice(0, 1), complete: false };
  assert.equal(slice.rows.some(x => x.person === 'P1' && x.dept === 'D1'), true);
  assert.equal(slice.rows.some(x => x.person === 'P2' && x.dept === 'D1'), false);
  assert.equal(slice.complete, false);
  assert.notEqual(distinctCount(slice.rows, x => x.person), query(seed).people);
});
test('non-monotone negative claim is revoked by later matching input', () => {
  const first = [{ person: 'P1', dept: 'D1' }];
  const later = [...first, { person: 'P2', dept: 'D1' }];
  assert.equal(first.some(x => x.person === 'P2'), false);
  assert.equal(later.some(x => x.person === 'P2'), true);
});
test('a changed view epoch prevents using a prior selected edge as current action target', () => {
  const oldSelection = { relationId: 'A3', epoch: 7 };
  const current = { epoch: 8, selectedRelation: 'A2' };
  assert.notEqual(oldSelection.epoch, current.epoch);
  assert.notEqual(oldSelection.relationId, current.selectedRelation);
});
