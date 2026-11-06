/**
 * Organization View freshness and negative-dependency consumer pressure (synthetic).
 * Uses the merged org-synthetic-r7 fixture; no TEI Workbench, Provider, Host or GUI.
 * Run: node --test reference/org-semantic-views/verify-consumer-freshness.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, scenario, membershipVersionsAt } from './source-fixture.mjs';

const routes = Object.fromEntries(scenario.syntheticProviderRoutes.map(r => [r.id, r]));
const READ_COMPLETE = Object.freeze({ collectionComplete: true, physicalWitness: 'synthetic-assumed' });
const READ_PARTIAL = Object.freeze({ collectionComplete: false, physicalWitness: 'unknown' });

function members(routeId, department) {
  const route = routes[routeId];
  if (!route) return { state: 'unresolved', reason: 'unknown-route' };
  const rows = membershipVersionsAt(fixture, route.validAt, route.knownAt)
    .filter(m => m.dept === department);
  return { state: 'resolved', ids: rows.map(x => x.id).sort(), route };
}
function context({ routeId='replica', epoch=7, anchor='D1', instance='I1',
  rolePin='organization-role-r1', sourceRevision=fixture.sourceRevision,
  selectedOccurrence='A2', queryPin='membership-q1', grantEpoch='g1' }={}) {
  return Object.freeze({ routeId, epoch, anchor, instance, rolePin, sourceRevision,
    selectedOccurrence, queryPin, grantEpoch });
}
function readToken(ctx) {
  const route = routes[ctx.routeId];
  if (!route) return null;
  return { ...ctx, provider: route.provider, mapping: route.mapping,
    alias: route.alias, validAt: route.validAt, knownAt: route.knownAt,
    snapshot: route.snapshot };
}
function compareReadToken(a,b) {
  if (!a || !b) return 'unverified';
  if (a.sourceRevision!==b.sourceRevision || a.rolePin!==b.rolePin ||
      a.queryPin!==b.queryPin) return 'stale-meaning';
  if (a.epoch!==b.epoch || a.anchor!==b.anchor || a.instance!==b.instance ||
      a.selectedOccurrence!==b.selectedOccurrence) return 'stale-selection';
  if (a.provider!==b.provider || a.mapping!==b.mapping || a.alias!==b.alias ||
      a.validAt!==b.validAt || a.knownAt!==b.knownAt ||
      a.snapshot!==b.snapshot) return 'stale-acquisition';
  return 'matches-read-context';
}
function qualifyForDisplay(response, current, evidence) {
  const check=compareReadToken(response.token,readToken(current));
  if(check!=='matches-read-context')return {state:'discard',reason:check};
  if(!evidence || evidence.physicalWitness!=='synthetic-assumed')
    return {state:'unverified',reason:'read-witness-unknown'};
  if(!evidence.collectionComplete)
    return {state:'partial-only',reason:'cannot-make-absence-claim'};
  return {state:'candidate-for-display'};
}
function claimStatus({kind, department, dependencies, closureComplete},changed) {
  if(!closureComplete) return 'unverified-dependency';
  if(!Array.isArray(dependencies)) return 'unverified-dependency';
  if(changed.some(x => dependencies.includes(x))) return 'recheck';
  return 'retain-if-scope-unchanged';
}
function missingResponsibility({observedOwners, evidence, acceptedRule}) {
  if(!acceptedRule) return 'unresolved-owner-rule';
  if(acceptedRule==='single' && new Set(observedOwners).size>1) return 'violated-positive-witness';
  if(!evidence || !evidence.collectionComplete || evidence.physicalWitness!=='synthetic-assumed')
    return 'unverified';
  return observedOwners.length===0 ? 'unassigned-for-closed-scope' : 'candidate-only';
}
function changeIntent({selected,sourceRevision,current,dirty}) {
  if(!dirty || dirty.instance!==current.instance || dirty.epoch!==current.epoch)
    return {state:'unresolved',reason:'editing-session'};
  if(sourceRevision!==current.sourceRevision ||
     selected!==current.selectedOccurrence ||
     dirty.target!==current.selectedOccurrence ||
     dirty.anchor!==current.anchor)
    return {state:'needs-rebind',reason:'source-or-occurrence-changed',preserveDirty:true};
  return {state:'candidate-only',target:dirty.target,preserveDirty:true};
}
function saveGroupResult({declaredAtomic,perProvider}) {
  if(!declaredAtomic && perProvider.length>1) return 'not-atomic-by-declaration';
  if(perProvider.some(x=>x.status==='unknown'))return 'partial-or-unknown';
  if(perProvider.some(x=>x.status==='rejected'))return 'rejected';
  return 'candidate-not-effect-confirmation';
}

test('VF-01 R1/R2 share snapshot s7 yet differ in route, known-at, and D1 set',()=>{
  assert.equal(routes.replica.snapshot,routes.primary.snapshot);
  assert.deepEqual(members('replica','D1').ids,['A1','A2']);
  assert.deepEqual(members('primary','D1').ids,['A1']);
  assert.notEqual(routes.replica.knownAt,routes.primary.knownAt);
});
test('VF-02 same source and selection epoch do not certify replica to primary re-use',()=>{
  const old=readToken(context());
  const next=context({routeId:'primary'});
  assert.equal(compareReadToken(old,readToken(next)),'stale-acquisition');
  assert.deepEqual(qualifyForDisplay({token:old},next,READ_COMPLETE),
    {state:'discard',reason:'stale-acquisition'});
});
test('VF-03 old epoch response cannot overwrite newly selected context',()=>{
  const old=readToken(context());
  const next=context({epoch:8,anchor:'D2',selectedOccurrence:'A3'});
  assert.equal(qualifyForDisplay({token:old},next,READ_COMPLETE).reason,'stale-selection');
});
test('VF-04 unknown physical read cannot be upgraded by matching UI context',()=>{
  const ctx=context();
  assert.equal(qualifyForDisplay({token:readToken(ctx)},ctx,null).state,'unverified');
});
test('VF-05 partial slice remains displayable only with an explicit partial status',()=>{
  const ctx=context();
  assert.equal(qualifyForDisplay({token:readToken(ctx)},ctx,READ_PARTIAL).state,'unverified');
  assert.equal(qualifyForDisplay({token:readToken(ctx)},ctx,
    {collectionComplete:false,physicalWitness:'synthetic-assumed'}).state,'partial-only');
});
test('VF-06 the same display has different meaning when role pin changes',()=>{
  assert.equal(compareReadToken(readToken(context()),readToken(context({rolePin:'organization-role-r2'}))),
    'stale-meaning');
});
test('VF-07 source revision mismatch invalidates old display without deleting dirty edits',()=>{
  const ctx=context({sourceRevision:'org-synthetic-r8'});
  assert.equal(qualifyForDisplay({token:readToken(context())},ctx,READ_COMPLETE).reason,'stale-meaning');
  assert.equal(changeIntent({selected:'A2',sourceRevision:fixture.sourceRevision,current:ctx,
    dirty:{instance:'I1',epoch:7,target:'A2',anchor:'D1'}}).preserveDirty,true);
});
test('VF-08 reused selection of Person P2 does not authorize other relation occurrence',()=>{
  const ctx=context({selectedOccurrence:'A3',anchor:'D2'});
  const dirty={instance:'I1',epoch:7,target:'A2',anchor:'D1'};
  assert.equal(changeIntent({selected:'A2',sourceRevision:fixture.sourceRevision,current:ctx,dirty}).state,'needs-rebind');
});
test('VF-09 valid current candidate is not a real driver request or effect',()=>{
  const ctx=context();
  assert.deepEqual(changeIntent({selected:'A2',sourceRevision:fixture.sourceRevision,current:ctx,
    dirty:{instance:'I1',epoch:7,target:'A2',anchor:'D1'}}),
    {state:'candidate-only',target:'A2',preserveDirty:true});
});
test('VF-10 a positive double-responsibility witness disproves single owner without a complete page',()=>{
  assert.equal(missingResponsibility({observedOwners:['D1a','D1b'],evidence:READ_PARTIAL,acceptedRule:'single'}),
    'violated-positive-witness');
});
test('VF-11 empty/partial responsibility result cannot prove unassigned',()=>{
  assert.equal(missingResponsibility({observedOwners:[],evidence:READ_PARTIAL,acceptedRule:'single'}),'unverified');
  assert.equal(missingResponsibility({observedOwners:[],evidence:READ_COMPLETE,acceptedRule:'single'}),
    'unassigned-for-closed-scope');
});
test('VF-12 no Owner rule => neither absence nor grant are inferred',()=>{
  assert.equal(missingResponsibility({observedOwners:[],evidence:READ_COMPLETE,acceptedRule:null}),
    'unresolved-owner-rule');
});
test('VF-13 D1 retraction invalidates D1 member Claim but not proven-independent D2 Claim',()=>{
  const d1={kind:'member-set',department:'D1',dependencies:['Membership:D1'],closureComplete:true};
  const d2={kind:'member-set',department:'D2',dependencies:['Membership:D2'],closureComplete:true};
  assert.equal(claimStatus(d1,['Membership:D1']),'recheck');
  assert.equal(claimStatus(d2,['Membership:D1']),'retain-if-scope-unchanged');
});
test('VF-14 non-output WHERE, ORDER or EXISTS dependencies invalidate affected claims',()=>{
  const deps={kind:'result',department:'D1',dependencies:['Membership:D1','Allocation','ProjectExists:D1'],
    closureComplete:true};
  assert.equal(claimStatus(deps,['Allocation']),'recheck');
  assert.equal(claimStatus(deps,['ProjectExists:D1']),'recheck');
});
test('VF-15 unknown dependency closure cannot be treated as no effect',()=>{
  const claim={kind:'result',department:'D2',dependencies:['Membership:D2'],closureComplete:false};
  assert.equal(claimStatus(claim,['Allocation']),'unverified-dependency');
});
test('VF-16 two-provider SaveGroup has no implicit atomicity',()=>{
  assert.equal(saveGroupResult({declaredAtomic:false,perProvider:[
    {id:'one',status:'confirmed'},{id:'two',status:'unknown'}]}),'not-atomic-by-declaration');
  assert.equal(saveGroupResult({declaredAtomic:true,perProvider:[
    {id:'one',status:'confirmed'},{id:'two',status:'unknown'}]}),'partial-or-unknown');
});
test('VF-17 confirmed-looking replies are not final effect receipts',()=>{
  assert.equal(saveGroupResult({declaredAtomic:true,perProvider:[
    {id:'one',status:'confirmed'}]}),'candidate-not-effect-confirmation');
});
test('VF-18 sourceChanged even with nonzero exit does not count as Source unchanged',()=>{
  const observed={exitCode:1,sourceChanged:true,decision:'accepted'};
  assert.equal(observed.sourceChanged,true);
  const current=context({sourceRevision:'org-synthetic-r8'});
  assert.equal(compareReadToken(readToken(context()),readToken(current)),'stale-meaning');
});

/*
 * Distinguish late-response rejection from revoking content already shown,
 * and current-empty from an event-complete historical claim. These are
 * simulated evidence decisions, NOT a standard TEI UI/Host protocol.
 */
function existingDisplayQualification({ paintedToken, current, dependencies, changes }) {
  if (!paintedToken) return 'not-displayed';
  if (compareReadToken(paintedToken, readToken(current)) !== 'matches-read-context')
    return 'revoke-current-qualification';
  if (!Array.isArray(dependencies)) return 'unverified-dependencies';
  if (changes.some(x => dependencies.includes(x))) return 'recheck-and-mark-stale';
  return 'retain-current-qualification-if-dependency-closure-proven';
}
function historyClaim({ currentRows, observedTransitions, completeInterval, rule }) {
  if (rule === 'current-empty') return currentRows.length === 0 ? 'current-empty-for-fixture' : 'not-empty';
  if (rule !== 'never-had-owner') return 'unresolved-owner-rule';
  if (!completeInterval || !Array.isArray(observedTransitions)) return 'unverified-history';
  return observedTransitions.some(s => s.owners > 0) ? 'violated-by-intermediate-event' : 'no-owner-in-closed-interval';
}
function multiClaimCohort(a, b) {
  if (!a || !b || !a.physicalWitness || !b.physicalWitness) return 'unverified';
  if (a.snapshot===b.snapshot && a.provider===b.provider && a.knownAt===b.knownAt &&
    a.readSession===b.readSession) return 'same-synthetic-read-session';
  return 'distinct-cohorts';
}
test('VF-19 an already displayed qualified claim must be retracted on route change',()=>{
  const painted=readToken(context());
  assert.equal(existingDisplayQualification({
    paintedToken:painted,current:context({routeId:'primary'}),
    dependencies:['Membership:D1'],changes:[]}), 'revoke-current-qualification');
});
test('VF-20 old response rejection and existing display revocation are different obligations',()=>{
  const old=readToken(context()), current=context({epoch:8});
  assert.equal(qualifyForDisplay({token:old},current,READ_COMPLETE).state,'discard');
  assert.equal(existingDisplayQualification({
    paintedToken:old,current,dependencies:['Membership:D1'],changes:[]}),
    'revoke-current-qualification');
});
test('VF-21 final zero cannot prove no intermediate responsibility event',()=>{
  assert.equal(historyClaim({currentRows:[],observedTransitions:null,completeInterval:false,
    rule:'never-had-owner'}),'unverified-history');
  assert.equal(historyClaim({currentRows:[],observedTransitions:[
    {owners:0},{owners:1},{owners:0}],completeInterval:true,
    rule:'never-had-owner'}),'violated-by-intermediate-event');
});
test('VF-22 current-empty and throughout-empty are distinct Owner Rule semantics',()=>{
  const transitions=[{owners:0},{owners:1},{owners:0}];
  assert.equal(historyClaim({currentRows:[],observedTransitions:transitions,completeInterval:false,
    rule:'current-empty'}),'current-empty-for-fixture');
  assert.equal(historyClaim({currentRows:[],observedTransitions:transitions,completeInterval:false,
    rule:'never-had-owner'}),'unverified-history');
});
test('VF-23 identical snapshot labels do not prove shared multi-claim Cohort',()=>{
  assert.equal(multiClaimCohort({
    provider:'db-replica',snapshot:'s7',knownAt:'2027-07-01',readSession:'read-1',physicalWitness:true
  },{
    provider:'db-primary',snapshot:'s7',knownAt:'2027-07-03',readSession:'read-2',physicalWitness:true
  }),'distinct-cohorts');
});
test('VF-24 one toy read-session can be paired without proving an actual Provider read',()=>{
  const common={provider:'db-replica',snapshot:'s7',knownAt:'2027-07-01',
    readSession:'read-1',physicalWitness:'synthetic-assumption'};
  assert.equal(multiClaimCohort(common,{...common}), 'same-synthetic-read-session');
});
