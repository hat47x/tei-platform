/**
 * Synthetic Organization Rule Dependency Footprint and Consumer Disclosure.
 * The source's capacity data are ordinary data; only the explicitly simulated
 * consumer policies decide what to expose in this example.
 * NO actual TEI Rule evaluator, Query, authorization, Host, Provider or side effect.
 * Run: node --test reference/org-semantic-views/verify-rule-footprint.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {fixture, scenario, membershipVersionsAt} from './source-fixture.mjs';

// The displayed projection is deliberately narrower than a rule's input.
const displayMap={role:'Display',relation:'Membership',anchor:'D1',
  fields:['id','person','dept'],sourceRevision:fixture.sourceRevision};
const ruleMap={role:'Rule Dependency',id:'org.fte-within-capacity',anchor:'D1',
  paths:['Membership','Allocation','DepartmentCapacity'],grain:'Department',
  sourceRevision:fixture.sourceRevision,ruleRevision:'cap-rule-r1'};
const routeById=Object.fromEntries(scenario.syntheticProviderRoutes.map(x=>[x.id,x]));
const disclosurePolicy=Object.fromEntries(scenario.syntheticConsumerPolicies.map(x=>[x.id,x]));
function membersFor(dept,route,data=fixture) {
  return membershipVersionsAt(data,route.validAt,route.knownAt).filter(x=>x.dept===dept);
}
function display(source=fixture,route=routeById.replica) {
  return {contextRole:'Display',route:route.id,sourceRevision:source.sourceRevision,
    rows:membersFor(displayMap.anchor,route,source).map(x=>({
      id:x.id,person:x.person,dept:x.dept
    }))};
}
function evaluateRule({source=fixture,displayRoute='replica',ruleRoute='replica',
  completeness='complete',acquisition=['Membership','Allocation','DepartmentCapacity'],
  acquisitionWitness='complete',ruleRevision=ruleMap.ruleRevision,
  sourceRevision=ruleMap.sourceRevision,missingAllocation=null,
  providerRows=null}={}) {
  const rule=source.syntheticCapacityRules?.find(x=>x.id===ruleMap.id);
  if(!rule || rule.role!=='test-only-rule' || rule.operator!=='lte' ||
      rule.measure!=='allocationBasisPoints'||rule.limitData!=='departmentCapacities')
    return {status:'unsupported',reason:'rule-definition'};
  if(sourceRevision!==source.sourceRevision || ruleRevision!==rule.revision)
    return {status:'unresolved',reason:'stale-semantic-or-rule-version'};
  const dr=routeById[displayRoute],rr=routeById[ruleRoute];
  if(!dr||!rr) return {status:'unresolved',reason:'unresolved-provider-route'};
  // A snapshot token is not globally unique and does not prove a read cohort.
  const sameCohort=dr.id===rr.id && dr.provider===rr.provider &&
    dr.mapping===rr.mapping && dr.validAt===rr.validAt && dr.knownAt===rr.knownAt;
  if(!sameCohort)return {status:'unresolved',reason:'unqualified-read-cohort'};
  if(completeness!=='complete')return {status:'unresolved',reason:'input-slice-incomplete'};
  if(acquisitionWitness!=='complete')return {status:'unresolved',reason:'acquisition-unverified'};
  const needed=ruleMap.paths;
  if(needed.some(p=>!acquisition.includes(p)))
    return {status:'unresolved',reason:'missing-rule-read-path'};
  const rows=providerRows??membersFor(ruleMap.anchor,rr,source);
  // Only a closed synthetic fixture makes this check independently possible.
  const expected=membersFor(ruleMap.anchor,rr,source);
  const key=x=>x.id+':'+x.person+':'+x.dept;
  if(JSON.stringify(rows.map(key).sort())!==JSON.stringify(expected.map(key).sort()))
    return {status:'unresolved',reason:'contradictory-closed-fixture-slice'};
  const capacities=source.departmentCapacities?.filter(c=>c.department===ruleMap.anchor &&
    c.revision==='capacity-r1')??[];
  if(capacities.length!==1 || !Number.isInteger(capacities[0].basisPoints))
    return {status:'unresolved',reason:'capacity-input-missing'};
  const allocations=source.allocations.filter(a=>a.relation!==missingAllocation);
  let total=0;
  for(const row of rows) {
    const found=allocations.filter(a=>a.relation===row.id);
    if(found.length!==1 || !Number.isInteger(found[0].basisPoints))
      return {status:'unresolved',reason:'allocation-input-missing'};
    total+=found[0].basisPoints;
  }
  return {status:total<=capacities[0].basisPoints?'satisfied':'not_satisfied',
    grain:'Department',dept:ruleMap.anchor,sourceRevision:source.sourceRevision,
    ruleRevision:rule.revision,route:rr.id,readCohort:{route:rr.id,
      validAt:rr.validAt,knownAt:rr.knownAt,snapshot:rr.snapshot},
    usedRelations:rows.map(x=>x.id),
    observedFteBasisPoints:total,capacityBasisPoints:capacities[0].basisPoints};
}
function eligibility(ruleResult) {
  return ruleResult.status==='satisfied'
    ? {status:'candidate-only',action:'synthetic-review-allocation'}
    : {status:'unavailable',reason:ruleResult.status};
}
function disclose(ruleResult,consumer) {
  const policy=disclosurePolicy[consumer];
  if(!policy || !policy.canReadRuleOutcome) return {status:'unavailable'};
  const allowed=ruleResult.status==='satisfied'||ruleResult.status==='not_satisfied';
  if(!allowed)return {status:'unavailable'};
  return {
    status:'visible',ruleOutcome:ruleResult.status,
    ...(policy.canReadCapacity?{capacityBasisPoints:ruleResult.capacityBasisPoints}:{}),
    sourceRevision:ruleResult.sourceRevision,
    ruleRevision:ruleResult.ruleRevision
  };
}
function narrowReadOnlyView() {
  return {rows:display().rows,readScope:'D1 Membership',
    fields:displayMap.fields,sourceRevision:displayMap.sourceRevision};
}
const capAdjusted=n=>({
 ...fixture,
 departmentCapacities:fixture.departmentCapacities.map(c=>
   c.department==='D1'?{...c,basisPoints:n}:c)
});

test('one synthetic source supports display and rule input references without a second model',()=>{
  assert.equal(displayMap.sourceRevision,fixture.sourceRevision);
  assert.equal(ruleMap.sourceRevision,fixture.sourceRevision);
  assert.deepEqual(display().rows.map(x=>x.id),['A1','A2']);
});
test('display is a narrow Membership projection and has no capacity value',()=>{
  const view=narrowReadOnlyView();
  assert.equal(view.rows.some(r=>'capacityBasisPoints' in r),false);
  assert.equal(view.fields.includes('capacityBasisPoints'),false);
});
test('rule footprint includes Allocation and DepartmentCapacity absent from display',()=>{
  assert.deepEqual(ruleMap.paths,['Membership','Allocation','DepartmentCapacity']);
  assert.equal(displayMap.fields.includes('basisPoints'),false);
});
test('explicit synthetic inputs give D1 160 basis points versus capacity 180',()=>{
  const result=evaluateRule();
  assert.equal(result.status,'satisfied');
  assert.equal(result.observedFteBasisPoints,160);
  assert.equal(result.capacityBasisPoints,180);
});
test('changing rule input alone leaves displayed rows unchanged but changes the outcome',()=>{
  const initial=display();
  const restrictive=capAdjusted(150);
  assert.deepEqual(display(restrictive),initial);
  assert.equal(evaluateRule({source:restrictive}).status,'not_satisfied');
});
test('changing a display label can leave rule evaluation unchanged',()=>{
  const altered={...fixture,departments:fixture.departments.map(d=>
    d.id==='D1'?{...d,label:'Different label'}:d)};
  assert.deepEqual(display(altered),display());
  assert.equal(evaluateRule({source:altered}).observedFteBasisPoints,160);
});
test('a missing DepartmentCapacity input cannot be guessed from member count',()=>{
  const src={...fixture,departmentCapacities:[]};
  assert.deepEqual(evaluateRule({source:src}),{status:'unresolved',reason:'capacity-input-missing'});
});
test('a missing individual allocation is unresolved, not a zero-FTE value',()=>{
  assert.deepEqual(evaluateRule({missingAllocation:'A2'}),
    {status:'unresolved',reason:'allocation-input-missing'});
});
test('incomplete rule read cannot return satisfied or not_satisfied',()=>{
  assert.deepEqual(evaluateRule({completeness:'partial'}),
    {status:'unresolved',reason:'input-slice-incomplete'});
});
test('unknown completeness also cannot return a Rule decision',()=>{
  assert.equal(evaluateRule({completeness:'unknown'}).status,'unresolved');
});
test('missing physical acquisition witness cannot prove Rule input was read',()=>{
  assert.deepEqual(evaluateRule({acquisitionWitness:'unknown'}),
    {status:'unresolved',reason:'acquisition-unverified'});
});
test('Display-only physical read misses Rule input paths',()=>{
  assert.deepEqual(evaluateRule({acquisition:['Membership']}),
    {status:'unresolved',reason:'missing-rule-read-path'});
});
test('a read with Allocation but no capacity still cannot decide',()=>{
  assert.equal(evaluateRule({acquisition:['Membership','Allocation']}).status,'unresolved');
});
test('same snapshot label s7 on replica and primary does not bind the Rule cohort',()=>{
  assert.equal(routeById.replica.snapshot,routeById.primary.snapshot);
  assert.deepEqual(evaluateRule({ruleRoute:'primary'}),
    {status:'unresolved',reason:'unqualified-read-cohort'});
});
test('the opposite cross-cohort direction is equally unqualified',()=>{
  assert.equal(evaluateRule({displayRoute:'primary',ruleRoute:'replica'}).status,'unresolved');
});
test('same-route primary read is a different historical D1 FTE calculation',()=>{
  const x=evaluateRule({displayRoute:'primary',ruleRoute:'primary'});
  assert.equal(x.status,'satisfied');assert.equal(x.observedFteBasisPoints,100);
  assert.deepEqual(x.usedRelations,['A1']);
});
test('complete-labelled missing row is rejected by closed-fixture independent expectation',()=>{
  const a=membersFor('D1',routeById.replica);
  assert.deepEqual(evaluateRule({providerRows:a.slice(0,1)}),
    {status:'unresolved',reason:'contradictory-closed-fixture-slice'});
});
test('a stale Rule revision is not accepted as current Rule evidence',()=>{
  assert.deepEqual(evaluateRule({ruleRevision:'cap-rule-r0'}),
    {status:'unresolved',reason:'stale-semantic-or-rule-version'});
});
test('a stale Source revision fails even when all visible rows appear unchanged',()=>{
  assert.equal(evaluateRule({sourceRevision:'org-synthetic-r6'}).status,'unresolved');
});
test('a text Role field with an unrecognized operation cannot assert the Rule',()=>{
  const altered={...fixture,syntheticCapacityRules:[{...fixture.syntheticCapacityRules[0],operator:'gt'}]};
  assert.deepEqual(evaluateRule({source:altered}),
    {status:'unsupported',reason:'rule-definition'});
});
test('a synthetic satisfied Rule is only an operation candidate, not an actual grant',()=>{
  assert.deepEqual(eligibility(evaluateRule()),
    {status:'candidate-only',action:'synthetic-review-allocation'});
});
test('a negative Rule result does not create an approved write action',()=>{
  assert.deepEqual(eligibility(evaluateRule({source:capAdjusted(150)})),
    {status:'unavailable',reason:'not_satisfied'});
});
test('a member-list consumer gets neither capacity nor the Rule result',()=>{
  assert.deepEqual(disclose(evaluateRule(),'local-member-list'),{status:'unavailable'});
});
test('an external summary gets an equal response in two worlds with opposite Rule outcomes',()=>{
  const a=disclose(evaluateRule(),'external-summary');
  const b=disclose(evaluateRule({source:capAdjusted(150)}),'external-summary');
  assert.deepEqual(a,b);assert.deepEqual(a,{status:'unavailable'});
});
test('a declared review consumer may view both Rule status and capacity in this scenario',()=>{
  const x=disclose(evaluateRule(),'local-capacity-review');
  assert.equal(x.status,'visible');assert.equal(x.ruleOutcome,'satisfied');
  assert.equal(x.capacityBasisPoints,180);
});
test('missing Consumer policy gives uniform unavailable, not a Rule result',()=>{
  assert.deepEqual(disclose(evaluateRule(),'unknown-consumer'),{status:'unavailable'});
});
test('an unresolved Rule does not leak the status to otherwise eligible consumers',()=>{
  assert.deepEqual(disclose(evaluateRule({completeness:'partial'}),'local-capacity-review'),
    {status:'unavailable'});
});
test('the two display rows do not multiply FTE through synthetic project join fanout',()=>{
  const members=membersFor('D1',routeById.replica);
  const joined=members.flatMap(m=>fixture.projects.filter(x=>x.dept==='D1').map(p=>({m,p})));
  assert.equal(joined.length,4);
  assert.equal(evaluateRule().observedFteBasisPoints,160);
});
test('no Rule computation mutates accepted test source or the scenario',()=>{
  const a=JSON.stringify(fixture),b=JSON.stringify(scenario);
  evaluateRule();display();disclose(evaluateRule(),'local-capacity-review');
  assert.equal(JSON.stringify(fixture),a);assert.equal(JSON.stringify(scenario),b);
});
