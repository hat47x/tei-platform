/**
 * Read-only Organization Query dependency pressure reference.
 * Output-field value lineage is NOT a complete Query/Claim dependency footprint.
 * No SQLGlot, TEI query executor, Provider I/O, real RLS or edit side effects.
 * Run: node --test reference/org-semantic-views/verify-query-dependencies.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, scenario, membershipVersionsAt } from './source-fixture.mjs';

const DISPLAY_FIELDS = Object.freeze(['id', 'person', 'dept']);
const routes = Object.fromEntries(scenario.syntheticProviderRoutes.map(r => [r.id, r]));
const m = (id, data = fixture) => data.allocations.find(a => a.relation === id);
const projectIds = (rows) => rows.map(r => r.id);
const sorted = (ids) => [...ids].sort();

// Synthetic query contract: no new TEI public Query schema or hidden built-in SQL.
function queryD1({
  data = fixture, routeId = 'replica', minAllocation = null,
  orderBy = 'id', projects = 'ignore', memberCompleteness = 'complete',
  projectCompleteness = 'complete', observation = 'read-1',
} = {}) {
  if (data.sourceRevision !== scenario.sourceRevisionRef)
    return { status: 'unresolved', reason: 'model-revision-mismatch' };
  const route = routes[routeId];
  if (!route) return { status: 'unresolved', reason: 'unbound-route' };
  if (!['id', 'allocation-desc'].includes(orderBy) ||
      !['ignore', 'exists', 'not-exists'].includes(projects) ||
      (minAllocation !== null && (!Number.isInteger(minAllocation) || minAllocation < 0)))
    return { status: 'unsupported', reason: 'query-operator' };
  if (memberCompleteness !== 'complete')
    return { status: 'unresolved', reason: 'incomplete-membership-scope' };
  if (projects !== 'ignore' && projectCompleteness !== 'complete')
    return { status: 'unresolved', reason: 'incomplete-project-scope' };

  let selected = membershipVersionsAt(data, route.validAt, route.knownAt)
    .filter(r => r.dept === 'D1');
  const predicateDependencies = [];
  if (minAllocation !== null || orderBy === 'allocation-desc') {
    // Lookup is a *synthetic* explicit dependency, not actual physical read evidence.
    for (const rel of selected) {
      const alloc = m(rel.id, data);
      if (!alloc || !Number.isInteger(alloc.basisPoints))
        return { status: 'unresolved', reason: 'missing-hidden-allocation-input' };
    }
    if (minAllocation !== null) {
      predicateDependencies.push('Allocation.basisPoints');
      selected = selected.filter(r => m(r.id, data).basisPoints >= minAllocation);
    }
  }
  if (projects !== 'ignore') {
    predicateDependencies.push('Project.department/exists');
    const hasProjects = data.projects.some(p => p.dept === 'D1');
    selected = selected.filter(() => projects === 'exists' ? hasProjects : !hasProjects);
  }
  if (orderBy === 'allocation-desc') {
    selected.sort((a, b) => m(b.id, data).basisPoints - m(a.id, data).basisPoints ||
      a.id.localeCompare(b.id));
  } else selected.sort((a, b) => a.id.localeCompare(b.id));
  const sortDependencies = orderBy === 'allocation-desc' ? ['Allocation.basisPoints'] : [];
  return {
    status: 'candidate',
    rows: selected.map(r => ({ id: r.id, person: r.person, dept: r.dept })),
    outputFieldValueLineage: ['Membership.id','Membership.person','Membership.dept'],
    predicateDependencies, sortDependencies,
    requestedScope: 'D1:Membership', grain: 'Membership occurrence',
    route: routeId, readCohort: { provider:route.provider, mapping:route.mapping,
      snapshot:route.snapshot, validAt:route.validAt, knownAt:route.knownAt,
      observation },
    // This explicitly does not say what the real Provider actually acquired.
    actualPhysicalAcquisition: 'not_observed',
  };
}
function dependencyEligibility(change, claim) {
  if (!change || change.observed !== true || change.scope !== 'D1' ||
    change.readCohortQualified !== true)
    return 'unresolved';
  const deps = {
    plain: ['Membership'],
    filtered: ['Membership','Allocation'],
    ordered: ['Membership','Allocation'],
    projectExists: ['Membership','Project'],
    capacityRule: ['Membership','Allocation','DepartmentCapacity'],
  }[claim];
  if (!deps) return 'unresolved';
  if (change.type === 'unknown') return 'unresolved';
  // Candidate invalidation advice ONLY: a complete actual read footprint would
  // still be needed before a product cache could reuse a previous answer.
  return deps.includes(change.type) ? 'recompute-candidate' : 'unaffected-candidate';
}
const modAllocation = (id, value) => ({
  ...fixture,
  allocations: fixture.allocations.map(a => a.relation === id
    ? { ...a, basisPoints: value } : a),
});
const modProjects = (newProjects) => ({ ...fixture, projects:newProjects });
const observed = (type) => ({ type, scope:'D1', observed:true, readCohortQualified:true });

test('base D1 Member list has exactly two original relation occurrences', () => {
  const q=queryD1();
  assert.deepEqual(projectIds(q.rows),['A1','A2']);
  assert.deepEqual(q.outputFieldValueLineage,
    ['Membership.id','Membership.person','Membership.dept']);
});
test('WHERE uses hidden Allocation despite identical projected fields', () => {
  const q=queryD1({minAllocation:70});
  assert.deepEqual(projectIds(q.rows),['A1']);
  assert.deepEqual(q.outputFieldValueLineage,queryD1().outputFieldValueLineage);
  assert.deepEqual(q.predicateDependencies,['Allocation.basisPoints']);
});
test('changing only WHERE threshold can change which relation rows exist', () => {
  assert.deepEqual(projectIds(queryD1({minAllocation:50}).rows),['A1','A2']);
  assert.deepEqual(projectIds(queryD1({minAllocation:70}).rows),['A1']);
});
test('hidden Allocation change can remove A2 without changing its direct relation', () => {
  const before=queryD1({minAllocation:50});
  const after=queryD1({minAllocation:50,data:modAllocation('A2',40)});
  assert.deepEqual(projectIds(before.rows),['A1','A2']);
  assert.deepEqual(projectIds(after.rows),['A1']);
  assert.equal(after.rows[0].person,'P1');
});
test('plain Data Context membership display does not depend on Allocation in this fixture', () => {
  const before=queryD1();
  const after=queryD1({data:modAllocation('A2',40)});
  assert.deepEqual(before.rows,after.rows);
  assert.deepEqual(before.predicateDependencies,[]);
});
test('non-visible Allocation controls ORDER BY and not output value lineage', () => {
  const q=queryD1({orderBy:'allocation-desc'});
  assert.deepEqual(projectIds(q.rows),['A1','A2']);
  assert.deepEqual(q.sortDependencies,['Allocation.basisPoints']);
  assert.deepEqual(q.outputFieldValueLineage,queryD1().outputFieldValueLineage);
});
test('ORDER BY changes order without changing output set, grain or relation IDs', () => {
  const updated=modAllocation('A2',120);
  const before=queryD1({orderBy:'allocation-desc'});
  const after=queryD1({orderBy:'allocation-desc',data:updated});
  assert.deepEqual(projectIds(before.rows),['A1','A2']);
  assert.deepEqual(projectIds(after.rows),['A2','A1']);
  assert.deepEqual(sorted(projectIds(before.rows)),sorted(projectIds(after.rows)));
  assert.equal(after.grain,before.grain);
});
test('missing hidden Sort input is unresolved, not an arbitrary fallback to id order', () => {
  const data={ ...fixture, allocations:fixture.allocations.filter(a=>a.relation!=='A2') };
  assert.deepEqual(queryD1({data,orderBy:'allocation-desc'}),
    {status:'unresolved',reason:'missing-hidden-allocation-input'});
});
test('missing hidden WHERE input is unresolved instead of dropping an unknown row', () => {
  const data={ ...fixture, allocations:fixture.allocations.filter(a=>a.relation!=='A2') };
  assert.equal(queryD1({data,minAllocation:50}).status,'unresolved');
});
test('EXISTS Project query can filter membership rows without projecting project field', () => {
  const q=queryD1({projects:'exists'});
  assert.deepEqual(projectIds(q.rows),['A1','A2']);
  assert.deepEqual(q.predicateDependencies,['Project.department/exists']);
  assert.deepEqual(q.outputFieldValueLineage,queryD1().outputFieldValueLineage);
});
test('removing two related Projects retracts EXISTS rows while memberships are unchanged', () => {
  const a=queryD1({projects:'exists'});
  const b=queryD1({projects:'exists',data:modProjects([])});
  assert.deepEqual(projectIds(a.rows),['A1','A2']);
  assert.deepEqual(projectIds(b.rows),[]);
});
test('NOT EXISTS is non-monotone: removing Project rows ADDS membership rows', () => {
  const a=queryD1({projects:'not-exists'});
  const b=queryD1({projects:'not-exists',data:modProjects([])});
  assert.deepEqual(projectIds(a.rows),[]);
  assert.deepEqual(projectIds(b.rows),['A1','A2']);
});
test('NOT EXISTS cannot assert absence from partially fetched Project relation', () => {
  assert.deepEqual(queryD1({projects:'not-exists',projectCompleteness:'unknown'}),
    {status:'unresolved',reason:'incomplete-project-scope'});
});
test('even EXISTS full claim cannot be established from unqualified partial Project scope', () => {
  assert.equal(queryD1({projects:'exists',projectCompleteness:'partial'}).status,'unresolved');
});
test('partial D1 Membership scope cannot prove complete ordered or filtered result', () => {
  assert.equal(queryD1({memberCompleteness:'partial',orderBy:'allocation-desc'}).status,
    'unresolved');
  assert.equal(queryD1({memberCompleteness:'unknown',minAllocation:50}).status,
    'unresolved');
});
test('same semantic model revision can accompany different result cohorts', () => {
  const replica=queryD1();
  const primary=queryD1({routeId:'primary'});
  assert.equal(replica.status,'candidate');assert.equal(primary.status,'candidate');
  assert.deepEqual(projectIds(replica.rows),['A1','A2']);
  assert.deepEqual(projectIds(primary.rows),['A1']);
  assert.equal(replica.readCohort.snapshot,primary.readCohort.snapshot); // both s7
  assert.notEqual(replica.readCohort.knownAt,primary.readCohort.knownAt);
});
test('same rows under distinct read observations are not qualified as the same cache key', () => {
  const a=queryD1({observation:'read-1'});
  const b=queryD1({observation:'read-2'});
  assert.deepEqual(a.rows,b.rows);
  assert.notEqual(a.readCohort.observation,b.readCohort.observation);
});
test('a stale semantic model revision refuses Query claim reuse', () => {
  assert.deepEqual(queryD1({data:{...fixture,sourceRevision:'org-synthetic-r6'}}),
    {status:'unresolved',reason:'model-revision-mismatch'});
});
test('unsupported Query operator does not silently become an identity Query', () => {
  assert.deepEqual(queryD1({orderBy:'random'}),
    {status:'unsupported',reason:'query-operator'});
});
test('single-field hidden update may affect filtered and ordered claims, not plain projection', () => {
  assert.equal(dependencyEligibility(observed('Allocation'),'filtered'),'recompute-candidate');
  assert.equal(dependencyEligibility(observed('Allocation'),'ordered'),'recompute-candidate');
  assert.equal(dependencyEligibility(observed('Allocation'),'plain'),'unaffected-candidate');
});
test('Capacity change invalidates Rule candidate but not plain membership display', () => {
  assert.equal(dependencyEligibility(observed('DepartmentCapacity'),'capacityRule'),
    'recompute-candidate');
  assert.equal(dependencyEligibility(observed('DepartmentCapacity'),'plain'),
    'unaffected-candidate');
});
test('Project relationship change affects EXISTS but not capacity Rule in synthetic footprint', () => {
  assert.equal(dependencyEligibility(observed('Project'),'projectExists'),'recompute-candidate');
  assert.equal(dependencyEligibility(observed('Project'),'capacityRule'),'unaffected-candidate');
});
test('unknown mutation scope or unqualified read cohort must not be called unaffected', () => {
  assert.equal(dependencyEligibility({...observed('Allocation'),readCohortQualified:false},
    'plain'),'unresolved');
  assert.equal(dependencyEligibility({...observed('Allocation'),observed:false},
    'plain'),'unresolved');
  assert.equal(dependencyEligibility({...observed('Allocation'),scope:'unknown'},
    'plain'),'unresolved');
});
test('invalidation candidate is not actual Provider cache proof or execution authority', () => {
  const q=queryD1({minAllocation:50});
  assert.equal(q.actualPhysicalAcquisition,'not_observed');
  assert.equal('grant' in q,false);
  assert.equal('receipt' in q,false);
  assert.equal('queryValidatorPass' in q,false);
});
test('pure synthetic Query/Claim comparison never changes shared Source or Scenario', () => {
  const before=JSON.stringify({fixture,scenario});
  queryD1({minAllocation:50,orderBy:'allocation-desc'});
  queryD1({projects:'not-exists'});
  assert.equal(JSON.stringify({fixture,scenario}),before);
});
