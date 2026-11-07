/**
 * ORG-TM-01..08: closed-world, read-only temporal pressure for #2510/#2386.
 * Uses the existing org-synthetic-r7 fixture. The half-open [from,to)
 * convention and sample business rules are TEST assumptions, not TEI Core
 * semantics, Owner acceptance, real Provider completeness or Host authority.
 * Run: node --test reference/org-semantic-views/verify-temporal.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, membershipVersionsAt } from './source-fixture.mjs';

const active = (date, from, to) => from <= date && (to === null || date < to);
const ids = rows => rows.map(row => row.id).sort();
const at = (valid, known, data = fixture) =>
  membershipVersionsAt(data, valid, known);
const occupantIds = (date, rows = fixture.occupancy) =>
  rows.filter(row => active(date, row.from, row.to)).map(row => row.id).sort();

// Only used for the ISO YYYY-MM-DD fixture, with null = an open end.
// This is not a timezone/calendar or real-DB range type.
function overlaps(a, b) {
  const ae = a.to ?? '9999-12-31';
  const be = b.to ?? '9999-12-31';
  return a.from < be && b.from < ae;
}
function occupancyOverlapPairs(rows) {
  const pairs = [];
  for (let i = 0; i < rows.length; i++) {
    for (let j = i + 1; j < rows.length; j++) {
      if (rows[i].position === rows[j].position && overlaps(rows[i], rows[j])) {
        pairs.push([rows[i].id, rows[j].id].sort().join('/'));
      }
    }
  }
  return pairs.sort();
}
// This is a TEST-only Claim qualification, not an Organization validation engine.
function overlapClaim(rows, complete, acceptedRule) {
  const witnesses = occupancyOverlapPairs(rows);
  if (witnesses.length) {
    if (acceptedRule === 'single-incumbent') return { state: 'conflict', witnesses };
    if (acceptedRule === 'multi-incumbent') return { state: 'allowed', witnesses };
    return { state: 'unresolved', witnesses };
  }
  return complete ? { state: 'no-overlap', witnesses: [] } :
    { state: 'unknown', witnesses: [] };
}
function occurrenceAt(id, valid, known, data = fixture) {
  const rows = at(valid, known, data).filter(row => row.id === id);
  // Explicitly refuse first-wins if two same-ID versions coexist.
  return rows.length > 1 ? { state: 'ambiguous' } :
    rows.length === 1 ? { state: 'one', row: rows[0] } :
    { state: 'absent' };
}

test('ORG-TM-01: a concurrent dual affiliation retains two Membership occurrences', () => {
  const two = at('2027-06-15', '2027-07-01').filter(r => r.person === 'P2');
  assert.deepEqual(ids(two), ['A2', 'A3']);
  assert.equal(new Set(two.map(r => r.person)).size, 1);
  assert.equal(two.length, 2);
});

test('ORG-TM-02: correction versions select one A2, not a second Membership', () => {
  const before = occurrenceAt('A2', '2027-05-15', '2027-07-01');
  const after = occurrenceAt('A2', '2027-05-15', '2027-07-02');
  assert.equal(before.state, 'one');
  assert.equal(after.state, 'one');
  assert.equal(before.row.validTo, null);
  assert.equal(after.row.validTo, '2027-06-01');
  assert.equal(occurrenceAt('A2', '2027-06-15', '2027-07-02').state, 'absent');
});

test('ORG-TM-03: half-open correction ends A2 exactly at valid 2027-06-01', () => {
  assert.equal(occurrenceAt('A2', '2027-05-31', '2027-07-03').state, 'one');
  assert.equal(occurrenceAt('A2', '2027-06-01', '2027-07-03').state, 'absent');
});

test('ORG-TM-04: O1/O2 share a boundary, not a same-instant incumbency', () => {
  assert.deepEqual(occupantIds('2027-06-30'), ['O1']);
  assert.deepEqual(occupantIds('2027-07-01'), ['O2']);
  assert.deepEqual(occupancyOverlapPairs(fixture.occupancy), []);
});

test('ORG-TM-05: an added O3 overlaps both O1 and O2 in interval time', () => {
  const rows = [...fixture.occupancy,
    { id: 'O3', person: 'P1', position: 'M1', from: '2027-06-30', to: '2027-07-02' }];
  const expected = ['O1/O3', 'O2/O3'];
  assert.deepEqual(occupancyOverlapPairs(rows), expected);
  assert.deepEqual(overlapClaim(rows, true, 'single-incumbent'),
    { state: 'conflict', witnesses: expected });
  assert.deepEqual(overlapClaim(rows, true, 'multi-incumbent'),
    { state: 'allowed', witnesses: expected });
  assert.deepEqual(overlapClaim(rows, true, null),
    { state: 'unresolved', witnesses: expected });
});

test('ORG-TM-06: A5 same Person and OrgUnit cannot be deduplicated by keys', () => {
  const data = { ...fixture, membershipVersions: [...fixture.membershipVersions,
    { id: 'A5', person: 'P2', dept: 'D1', provenance: 'direct',
      validFrom: '2027-06-30', validTo: '2027-07-02',
      knownFrom: '2027-04-01', knownTo: null }] };
  const samePair = at('2027-06-30', '2027-07-01', data)
    .filter(r => r.person === 'P2' && r.dept === 'D1');
  assert.deepEqual(ids(samePair), ['A2', 'A5']);
  assert.equal(new Set(samePair.map(r => r.person)).size, 1);
  // Business cardinality/primary/allocation policy remains Owner-unaccepted.
});

test('ORG-TM-07: equal known-time conflicting A2 versions are ambiguous', () => {
  const original = fixture.membershipVersions.find(r => r.id === 'A2');
  const others = fixture.membershipVersions.filter(r => r.id !== 'A2');
  const first = { ...original, knownTo: null };
  const second = { ...first, dept: 'D2' };
  const data = { ...fixture, membershipVersions: [...others, first, second] };
  const flipped = { ...data, membershipVersions: [...data.membershipVersions].reverse() };
  assert.equal(occurrenceAt('A2', '2027-06-15', '2027-07-03', data).state, 'ambiguous');
  assert.equal(occurrenceAt('A2', '2027-06-15', '2027-07-03', flipped).state, 'ambiguous');
});

test('ORG-TM-08: absence of overlaps requires a complete period scope', () => {
  const base = fixture.occupancy;
  assert.deepEqual(overlapClaim(base, true, 'single-incumbent'),
    { state: 'no-overlap', witnesses: [] });
  assert.deepEqual(overlapClaim(base.slice(0, 1), false, 'single-incumbent'),
    { state: 'unknown', witnesses: [] });
  const o3 = { id: 'O3', person: 'P1', position: 'M1',
    from: '2027-06-30', to: '2027-07-02' };
  assert.deepEqual(overlapClaim([base[0], o3], false, 'single-incumbent'),
    { state: 'conflict', witnesses: ['O1/O3'] });
});

test('five literal valid/known expectations stay independent of a display tree', () => {
  const rows = [
    ['2027-05-15', '2027-07-03', ['A1', 'A2'], ['A3', 'A4'], ['O1']],
    ['2027-06-15', '2027-07-01', ['A1', 'A2'], ['A3', 'A4'], ['O1']],
    ['2027-06-15', '2027-07-02', ['A1'], ['A3', 'A4'], ['O1']],
    ['2027-06-15', '2027-07-03', ['A1'], ['A3', 'A4'], ['O1']],
    ['2027-07-01', '2027-07-03', ['A1'], ['A3', 'A4'], ['O2']],
  ];
  for (const [valid, known, d1, d2, occupied] of rows) {
    assert.deepEqual(ids(at(valid, known).filter(x => x.dept === 'D1')), d1);
    assert.deepEqual(ids(at(valid, known).filter(x => x.dept === 'D2')), d2);
    assert.deepEqual(occupantIds(valid), occupied);
  }
});

test('a new Occupancy does not create an implied D1 Membership or Host Grant', () => {
  const present = at('2027-07-01', '2027-07-03');
  assert.deepEqual(occupantIds('2027-07-01'), ['O2']);
  assert.equal(present.some(m => m.person === 'P3' && m.dept === 'D1'), false);
  // This reference has no trusted Principal/Tenant/Grant Provider.
  assert.equal(Object.hasOwn(fixture, 'hostGrant'), false);
});
