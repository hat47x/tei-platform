/**
 * ORG-CAP-01..05/07: read-only organization quantity/scenario pressure test.
 * Uses the EXISTING org-synthetic-r7 source and scenario fixtures; it is NOT
 * the TEI Query, Provider, Scenario UI, Host, Grant or business acceptance.
 * Run: node --test reference/org-semantic-views/verify-capacity-scenarios.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, scenario, membershipVersionsAt } from './source-fixture.mjs';

// Independent literal expectations are authored here. Do not compute the
// expected values with the evaluator that computes Observed.
const BASELINE_EXPECTED = Object.freeze({
  membershipCount: 4,
  uniquePeople: ['P1', 'P2', 'P3'],
  byDepartment: {
    D1: { memberships: 2, people: 2, allocatedBasisPoints: 160, usableBasisPoints: 160 },
    D2: { memberships: 2, people: 2, allocatedBasisPoints: 140, usableBasisPoints: 140 },
  },
  allocatedTotalBasisPoints: 300,
});

// 'basisPoints' is a legacy fixture FIELD NAME, not a standardized financial bp.
// Its synthetic meaning is hundredths of one FTE: 100 raw units = 1 FTE.
// Financial basis points would instead require 10,000 units per whole ratio.
// Do not infer a Quantity scale from the raw field name or an integer value.
const SYNTHETIC_FTE_ENCODING = Object.freeze({
  field: 'basisPoints', unit: 'FTE', encoding: 'fte-hundredths', rawUnitsPerFTE: 100,
});

// Fixture-local pure comparison; no Source write, accepted Rule or real read.
function calculate({
  data = fixture,
  validAt = scenario.baseline.validAt,
  knownAt = scenario.baseline.knownAt,
  sourceRevisionRef = scenario.sourceRevisionRef,
  complete = true,
  allocationOverrides = {},
  unavailablePeople = [],
  allocationEncoding = SYNTHETIC_FTE_ENCODING,
} = {}) {
  if (data.sourceRevision !== sourceRevisionRef) {
    return { state: 'unresolved', reason: 'source-revision-mismatch' };
  }
  if (!complete) return { state: 'unknown', reason: 'incomplete-membership-or-allocation-scope' };
  if (allocationEncoding?.field !== 'basisPoints' ||
      allocationEncoding?.unit !== 'FTE' ||
      allocationEncoding?.encoding !== 'fte-hundredths' ||
      allocationEncoding?.rawUnitsPerFTE !== 100) {
    return { state: 'unresolved', reason: 'unaccepted-allocation-unit-scale' };
  }
  const rows = membershipVersionsAt(data, validAt, knownAt);
  const allocationByRelation = new Map();
  for (const allocation of data.allocations) {
    if (allocationByRelation.has(allocation.relation)) {
      return { state: 'unresolved', reason: 'ambiguous-allocation-occurrence' };
    }
    allocationByRelation.set(allocation.relation, allocation.basisPoints);
  }
  const rowIds = new Set(rows.map(r => r.id));
  if (rowIds.size !== rows.length) return { state: 'unresolved', reason: 'ambiguous-occurrence' };
  if (Object.keys(allocationOverrides).some(id => !rowIds.has(id))) {
    return { state: 'unresolved', reason: 'unknown-allocation-occurrence' };
  }
  if (unavailablePeople.some(id => !data.persons.includes(id))) {
    return { state: 'unresolved', reason: 'unknown-person' };
  }
  const byDepartment = {};
  for (const { id } of data.departments) {
    const members = rows.filter(r => r.dept === id);
    let allocatedBasisPoints = 0, usableBasisPoints = 0;
    for (const member of members) {
      const amount = Object.hasOwn(allocationOverrides, member.id)
        ? allocationOverrides[member.id] : allocationByRelation.get(member.id);
      if (!Number.isInteger(amount) || amount < 0) {
        return { state: 'unknown', reason: 'missing-or-invalid-allocation-basis' };
      }
      allocatedBasisPoints += amount;
      if (!unavailablePeople.includes(member.person)) usableBasisPoints += amount;
    }
    byDepartment[id] = {
      memberships: members.length,
      people: new Set(members.map(m => m.person)).size,
      allocatedBasisPoints,
      usableBasisPoints,
    };
  }
  return {
    state: 'synthetic-complete',
    sourceRevision: data.sourceRevision,
    validAt,
    knownAt,
    membershipCount: rows.length,
    uniquePeople: [...new Set(rows.map(r => r.person))].sort(),
    byDepartment,
    allocatedTotalBasisPoints: Object.values(byDepartment)
      .reduce((acc, department) => acc + department.allocatedBasisPoints, 0),
  };
}

test('ORG-CAP-01: r7 baseline is three distinct people / four Memberships / 3.00 FTE', () => {
  const observed = calculate();
  assert.equal(observed.state, 'synthetic-complete');
  for (const [key, value] of Object.entries(BASELINE_EXPECTED)) {
    assert.deepEqual(observed[key], value, 'expected mismatch: ' + key);
  }
  // D1(2) + D2(2) counts P2 twice; global Person cardinality is three.
  assert.equal(observed.byDepartment.D1.people + observed.byDepartment.D2.people, 4);
});

test('ORG-CAP-02: a two-project JOIN does not double count Membership allocation', () => {
  const rows = membershipVersionsAt(fixture, scenario.baseline.validAt, scenario.baseline.knownAt)
    .filter(row => row.dept === 'D1');
  const joined = rows.flatMap(row => fixture.projects.filter(p => p.dept === 'D1')
    .map(p => ({ relation: row.id, project: p.id })));
  const amountById = new Map(fixture.allocations.map(a => [a.relation, a.basisPoints]));
  assert.equal(joined.length, 4);
  assert.equal(joined.reduce((sum, r) => sum + amountById.get(r.relation), 0), 320);
  assert.equal(calculate().byDepartment.D1.allocatedBasisPoints, 160);
});

test('ORG-CAP-03: P1 absence affects usable allocation, not actual Membership/FTE', () => {
  const result = calculate({ unavailablePeople: ['P1'] });
  assert.equal(result.byDepartment.D1.allocatedBasisPoints, 160);
  assert.equal(result.byDepartment.D1.usableBasisPoints, 60);
  assert.equal(result.byDepartment.D2.allocatedBasisPoints, 140);
  assert.equal(result.membershipCount, 4);
});

test('ORG-CAP-04: hypothetical P2 allocation shift changes no Membership', () => {
  // Not a transfer of A2 or of Case responsibility: only A2/A3 allocation assumptions.
  const shifted = calculate({ allocationOverrides: { A2: 0, A3: 100 } });
  assert.equal(shifted.byDepartment.D1.allocatedBasisPoints, 100);
  assert.equal(shifted.byDepartment.D2.allocatedBasisPoints, 200);
  assert.equal(shifted.allocatedTotalBasisPoints, 300);
  assert.deepEqual(shifted.uniquePeople, BASELINE_EXPECTED.uniquePeople);
  assert.equal(shifted.membershipCount, 4);
  assert.deepEqual(calculate().byDepartment, BASELINE_EXPECTED.byDepartment);
});

test('ORG-CAP-05: incomplete input and missing allocation are unknown, never zero', () => {
  assert.deepEqual(calculate({ complete: false }), {
    state: 'unknown', reason: 'incomplete-membership-or-allocation-scope',
  });
  const missing = { ...fixture, allocations: fixture.allocations.filter(x => x.relation !== 'A2') };
  assert.deepEqual(calculate({ data: missing }), {
    state: 'unknown', reason: 'missing-or-invalid-allocation-basis',
  });
});

test('ORG-CAP-UNIT-01: synthetic FTE-hundredths must not be silently cast as financial bp', () => {
  // These are literal fixture expectations; this reference cannot adopt a new Core Unit.
  assert.equal(fixture.allocations.find(a => a.relation === 'A1').basisPoints, 100);
  assert.equal(fixture.allocations.find(a => a.relation === 'A2').basisPoints, 60);
  assert.equal(100 / SYNTHETIC_FTE_ENCODING.rawUnitsPerFTE, 1);
  assert.equal(60 / SYNTHETIC_FTE_ENCODING.rawUnitsPerFTE, 0.6);
  // Generic financial basis points denote 1/10,000 of a ratio, not 1/100.
  assert.equal(100 / 10000, 0.01);
  assert.deepEqual(calculate({ allocationEncoding: undefined }), calculate());
  assert.deepEqual(calculate({ allocationEncoding: null }), {
    state: 'unresolved', reason: 'unaccepted-allocation-unit-scale',
  });
  assert.deepEqual(calculate({ allocationEncoding: { field: 'basisPoints', unit: 'FTE',
    encoding: 'financial-basis-points', rawUnitsPerFTE: 10000 } }), {
    state: 'unresolved', reason: 'unaccepted-allocation-unit-scale',
  });
});

test('ORG-CAP-UNIT-02: duplicate Allocation occurrence is a conflict, not last-value-wins', () => {
  const data = { ...fixture, allocations: [...fixture.allocations,
    { relation: 'A2', basisPoints: 800 }] };
  assert.deepEqual(calculate({ data }), {
    state: 'unresolved', reason: 'ambiguous-allocation-occurrence',
  });
  assert.equal(calculate().byDepartment.D1.allocatedBasisPoints, 160);
});

test('ORG-CAP-07: same valid date with later known date changes baseline eligibility', () => {
  const late = calculate({ knownAt: '2027-07-03' });
  assert.equal(late.membershipCount, 3);
  assert.deepEqual(late.uniquePeople, ['P1', 'P2', 'P3']);
  assert.equal(late.byDepartment.D1.allocatedBasisPoints, 100);
  assert.equal(late.byDepartment.D2.allocatedBasisPoints, 140);
  assert.equal(late.allocatedTotalBasisPoints, 240);
  assert.notDeepEqual(late.byDepartment, calculate().byDepartment);
  assert.deepEqual(calculate({ sourceRevisionRef: 'org-synthetic-r8' }), {
    state: 'unresolved', reason: 'source-revision-mismatch',
  });
});

test('ORG-CAP-07b: known-time correction affects D1/global, not D2 in complete synthetic scope', () => {
  const earlier = calculate({ knownAt: '2027-07-01' });
  const later = calculate({ knownAt: '2027-07-03' });
  assert.equal(earlier.state, 'synthetic-complete');
  assert.equal(later.state, 'synthetic-complete');
  assert.deepEqual(later.byDepartment.D2, earlier.byDepartment.D2);
  assert.notDeepEqual(later.byDepartment.D1, earlier.byDepartment.D1);
  assert.notEqual(later.allocatedTotalBasisPoints, earlier.allocatedTotalBasisPoints);
  assert.deepEqual(calculate({ knownAt: '2027-07-03', complete: false }), {
    state: 'unknown', reason: 'incomplete-membership-or-allocation-scope',
  });
  // A matching D2 computed value is not evidence of a common real Provider cohort.
});

test('scenario rejects unrecognized occurrence and invalid basis, without mutating Source', () => {
  const original = JSON.stringify(fixture);
  assert.deepEqual(calculate({ allocationOverrides: { A99: 50 } }), {
    state: 'unresolved', reason: 'unknown-allocation-occurrence',
  });
  assert.deepEqual(calculate({ allocationOverrides: { A2: -1 } }), {
    state: 'unknown', reason: 'missing-or-invalid-allocation-basis',
  });
  calculate({ allocationOverrides: { A2: 0, A3: 100 }, unavailablePeople: ['P1'] });
  assert.equal(JSON.stringify(fixture), original);
});

// Not covered by this reference: ORG-CAP-06 actual qualification/Grant/Host;
// ORG-CAP-08 Case/Directive/Effect/ACK or real Studio/SHO/Harness consumer.
