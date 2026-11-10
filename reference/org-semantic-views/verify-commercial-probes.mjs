/**
 * Commercial HCM/ERP/domain-separation synthetic meaning probes.
 * Independent read-only expected values. This file is not a TEI Product,
 * Provider, security Policy, a canonical HR schema or an integration fixture.
 * Run: node --test reference/org-semantic-views/verify-commercial-probes.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';

const active = (x, at) => x.from <= at && (x.to === null || at < x.to);
function positionCandidate(rows, position, at, multi, chosen = null) {
  const people = rows.filter(x => x.position === position && active(x, at)).map(x => x.person);
  if (!multi && people.length > 1) return { status: 'invalid', reason: 'incumbent-cardinality' };
  if (people.length === 0) return { status: 'vacant', people: [] };
  if (chosen !== null && !people.includes(chosen)) return { status: 'unresolved', reason: 'manager-not-incumbent' };
  if (people.length > 1 && chosen === null) return { status: 'unresolved', reason: 'manager-ambiguous', people };
  return { status: 'candidate', manager: chosen ?? people[0], people };
}
function typedParents(edges, role, child, max) {
  const parents = [...new Set(edges.filter(x => x.role === role && x.child === child).map(x => x.parent))];
  return parents.length > max ? { status: 'invalid', reason: 'role-cardinality' } : { status: 'candidate', parents };
}
function sameDay(rows, at) {
  const found = rows.filter(x => x.validAt === at);
  if (!found.length) return { status: 'unresolved', reason: 'no-source-history' };
  if (found.length > 1 && found.some(x => !Number.isInteger(x.sequence)))
    return { status: 'unresolved', reason: 'sequence-not-observed' };
  const sorted = [...found].sort((a,b) => (a.sequence ?? 0) - (b.sequence ?? 0));
  return { status: 'candidate', latest: sorted.at(-1).value, observedCount: sorted.length };
}
function primaryAssignment(employments, assignments) {
  const primary = employments.filter(x => x.primary);
  if (primary.length !== 1) return { status: 'unresolved', reason: 'primary-relationship-ambiguous' };
  const rows = assignments.filter(x => x.workRelationship === primary[0].id && x.primary);
  if (rows.length !== 1) return { status: 'unresolved', reason: 'primary-assignment-ambiguous' };
  return { status: 'candidate', assignment: rows[0].id };
}
const recordRead = (record, principal, grants) => grants.some(g =>
  g.principal === principal.id && g.owningBU === record.owningBU && g.action === 'read')
  ? 'candidate' : 'denied';
const domainAccess = (links, consumer, target, kind) => kind === 'data' &&
  (consumer === target || links.some(x => x.viewer === consumer && x.target === target))
  ? 'candidate' : 'denied';
// Dataverse-specific *synthetic candidate* only. Actual access requires the
// production security role/record/access check; organization paths are not Grants.
function hierarchyAccessCandidate({
  model, enabled, sameOrParentBU, distance, maxDepth, action, tablePrivileges,
}) {
  if (!enabled) return { status: 'not-admitted', reason: 'hierarchy-disabled' };
  if (!tablePrivileges.includes(action))
    return { status: 'not-admitted', reason: 'missing-table-privilege' };
  if (!Number.isInteger(distance) || distance < 1 || distance > maxDepth)
    return { status: 'not-admitted', reason: 'path-outside-approved-depth' };
  if (model === 'manager' && !sameOrParentBU)
    return { status: 'not-admitted', reason: 'manager-bu-scope' };
  if (model !== 'manager' && model !== 'position')
    return { status: 'unresolved', reason: 'hierarchy-model-unknown' };
  if (model === 'position' && distance > 1 && action !== 'read')
    return { status: 'not-admitted', reason: 'indirect-position-read-only' };
  return { status: 'candidate', source: 'synthetic-hierarchy-rule' };
}
// Owning and sharing are two distinct relation roles, even if their team IDs
// are selected by the same person/organization UI.
function teamRecordCandidate({ team, row, sharedToTeam, tableRead }) {
  const owns = team.type === 'owner' && row.ownerTeam === team.id;
  const explicitlyShared = team.type === 'access' && sharedToTeam === team.id;
  return {
    owns,
    canReadCandidate: tableRead && (owns || explicitlyShared),
  };
}

function previewMove(base, move, maxLayers) {
  const proposal = structuredClone(base);
  const target = proposal.positions.find(x => x.id === move.position);
  if (!target) return { status: 'unresolved', reason: 'missing-position' };
  target.dept = move.to;
  return { status: 'preview', proposal,
    warnings: move.layers > maxLayers ? ['layers-guideline'] : [],
    approval: 'not-observed', effect: 'not-observed' };
}

const occupants = [
  { position: 'M1', person: 'P2', from: '2027-06-01', to: null },
  { position: 'M1', person: 'P3', from: '2027-06-01', to: null },
];
test('CM-02 vacant position exists without an incumbent', () => {
  assert.deepEqual(positionCandidate([], 'M1', '2027-06-15', false), { status: 'vacant', people: [] });
});
test('CM-02 single-incumbent invalid; multi-incumbent does not select the first manager silently', () => {
  assert.deepEqual(positionCandidate(occupants, 'M1', '2027-06-15', false),
    { status: 'invalid', reason: 'incumbent-cardinality' });
  assert.deepEqual(positionCandidate(occupants, 'M1', '2027-06-15', true),
    { status: 'unresolved', reason: 'manager-ambiguous', people: ['P2', 'P3'] });
  assert.deepEqual(positionCandidate(occupants, 'M1', '2027-06-15', true, 'P3'),
    { status: 'candidate', manager: 'P3', people: ['P2', 'P3'] });
});
test('CM-01 supervisory parents and matrix parents have different cardinalities', () => {
  const edges = ['supervisory', 'matrix'].flatMap(role => ['D1', 'D2'].map(parent =>
    ({ role, parent, child: 'D3' })));
  assert.deepEqual(typedParents(edges, 'supervisory', 'D3', 1),
    { status: 'invalid', reason: 'role-cardinality' });
  assert.deepEqual(typedParents(edges, 'matrix', 'D3', 2),
    { status: 'candidate', parents: ['D1', 'D2'] });
});
test('CM-03 overall primary assignment uses work relationship, not first row or person alone', () => {
  const jobs = [{ id: 'E2', primary: false }, { id: 'E1', primary: true }];
  const assignments = [{ id: 'A2', workRelationship: 'E2', primary: true },
    { id: 'A1', workRelationship: 'E1', primary: true },
    { id: 'A3', workRelationship: 'E1', primary: false }];
  assert.deepEqual(primaryAssignment(jobs, assignments), { status: 'candidate', assignment: 'A1' });
  assert.deepEqual(primaryAssignment(jobs, [...assignments, { id: 'A4', workRelationship: 'E1', primary: true }]),
    { status: 'unresolved', reason: 'primary-assignment-ambiguous' });
});
test('CM-05 two updates with same valid date require source-provided sequence', () => {
  const rows = [{ validAt: '2027-04-01', sequence: 2, value: 'D2' },
    { validAt: '2027-04-01', sequence: 1, value: 'D1' }];
  assert.deepEqual(sameDay(rows, '2027-04-01'),
    { status: 'candidate', latest: 'D2', observedCount: 2 });
  assert.deepEqual(sameDay(rows.map(({sequence, ...r}) => r), '2027-04-01'),
    { status: 'unresolved', reason: 'sequence-not-observed' });
});
test('CM-05 correction source with only one surviving row does not establish lost intermediates', () => {
  assert.deepEqual(sameDay([{ validAt: '2027-04-01', value: 'D2' }], '2027-04-01'),
    { status: 'candidate', latest: 'D2', observedCount: 1 });
  assert.deepEqual(sameDay([], '2027-04-01'), { status: 'unresolved', reason: 'no-source-history' });
});
test('CM-07 organizational BU is not owning BU or effective read grant', () => {
  const person = { id: 'P2', organizationalBU: 'D1' };
  const row = { id: 'C1', owningBU: 'D2', owner: 'P3' };
  assert.equal(recordRead(row, person, []), 'denied');
  assert.equal(recordRead(row, person, [{ principal: 'P2', owningBU: 'D2', action: 'read' }]), 'candidate');
  assert.equal(row.owningBU, 'D2');
});
test('CM-09 domain Contains authorizes data visibility only, not business process execution', () => {
  const links = [{ viewer: 'A', target: 'B' }];
  assert.equal(domainAccess(links, 'A', 'B', 'data'), 'candidate');
  assert.equal(domainAccess(links, 'A', 'B', 'process'), 'denied');
  assert.equal(domainAccess(links, 'B', 'A', 'data'), 'denied');
});
test('CM-11 what-if warning does not reject preview, approve plan, or mutate actual source', () => {
  const actual = { positions: [{ id: 'M1', dept: 'D1' }] };
  const result = previewMove(actual, { position: 'M1', to: 'D2', layers: 5 }, 4);
  assert.deepEqual(result.proposal.positions, [{ id: 'M1', dept: 'D2' }]);
  assert.deepEqual(actual.positions, [{ id: 'M1', dept: 'D1' }]);
  assert.deepEqual(result.warnings, ['layers-guideline']);
  assert.equal(result.status, 'preview');
  assert.equal(result.approval, 'not-observed');
  assert.equal(result.effect, 'not-observed');
});

test('CM-13 position hierarchy direct ancestor can offer write, indirect ancestor only read', () => {
  const base = {
    model: 'position', enabled: true, sameOrParentBU: false,
    maxDepth: 3, tablePrivileges: ['read', 'write'],
  };
  assert.deepEqual(hierarchyAccessCandidate({ ...base, distance: 1, action: 'write' }),
    { status: 'candidate', source: 'synthetic-hierarchy-rule' });
  assert.deepEqual(hierarchyAccessCandidate({ ...base, distance: 2, action: 'write' }),
    { status: 'not-admitted', reason: 'indirect-position-read-only' });
  assert.deepEqual(hierarchyAccessCandidate({ ...base, distance: 2, action: 'read' }),
    { status: 'candidate', source: 'synthetic-hierarchy-rule' });
  assert.deepEqual(hierarchyAccessCandidate({ ...base, distance: 4, action: 'read' }),
    { status: 'not-admitted', reason: 'path-outside-approved-depth' });
});
test('CM-13 manager and position hierarchy have different BU scope despite same graph path', () => {
  const base = { enabled: true, sameOrParentBU: false, distance: 1,
    maxDepth: 3, action: 'read', tablePrivileges: ['read'] };
  assert.deepEqual(hierarchyAccessCandidate({ ...base, model: 'manager' }),
    { status: 'not-admitted', reason: 'manager-bu-scope' });
  assert.deepEqual(hierarchyAccessCandidate({ ...base, model: 'position' }),
    { status: 'candidate', source: 'synthetic-hierarchy-rule' });
  assert.deepEqual(hierarchyAccessCandidate({ ...base, model: 'position',
    tablePrivileges: [] }),
    { status: 'not-admitted', reason: 'missing-table-privilege' });
});
test('CM-14 owning team and access team differ even when both can read same record', () => {
  const row = { id: 'C1', ownerTeam: 'team-own' };
  const owner = teamRecordCandidate({
    team: { type: 'owner', id: 'team-own' }, row, sharedToTeam: null, tableRead: true,
  });
  const access = teamRecordCandidate({
    team: { type: 'access', id: 'team-access' }, row,
    sharedToTeam: 'team-access', tableRead: true,
  });
  assert.deepEqual(owner, { owns: true, canReadCandidate: true });
  assert.deepEqual(access, { owns: false, canReadCandidate: true });
  assert.deepEqual(teamRecordCandidate({
    team: { type: 'access', id: 'team-access' }, row,
    sharedToTeam: 'team-access', tableRead: false,
  }), { owns: false, canReadCandidate: false });
});
