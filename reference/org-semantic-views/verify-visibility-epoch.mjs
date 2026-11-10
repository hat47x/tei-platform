/**
 * ORG-VE synthetic consumer requalification after Org/Policy/Grant changes.
 * A read-only toy. This does NOT enforce actual Provider/Host authorization.
 * node --test reference/org-semantic-views/verify-visibility-epoch.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';

const clone = x => JSON.parse(JSON.stringify(x));
const current = () => ({
 source: 'org-model-r1', sourceLocator: 'org-source-a',
 meaning: 'case-handling-r1', policy: 'org-policy-r1',
 axis: 'command', root: 'D1', validAt: '2027-06-15',
 knownAt: '2027-07-01', principal: 'U1', tenant: 'T1',
 grantEpoch: 4, readCohort: 'provider-c1',
 queryFootprint: 'where-org-r1',
 selectionEpoch: 7, sessionEpoch: 9,
});
const fields = ['source', 'sourceLocator', 'meaning', 'policy','axis',
  'root','validAt','knownAt','principal','tenant','grantEpoch',
  'readCohort','queryFootprint','selectionEpoch','sessionEpoch'];
const response = (epoch=current(),more={}) => ({
 request: clone(epoch), rows: [{id:'CASE-1',unit:'L1',title:'Internal'}],
 complete: true, fieldComplete: true, route: 'list',
 execution: 'synthetic-provider-observed', status: 'ok',
 ...more,
});
function consumerResult(res,now,permit,needed='read') {
 if(res.status!=='ok')return {status:'unverified',why:'attempt-not-successful'};
 for(const key of fields){
   if(res.request[key]!==now[key])return {status:'stale',why:'context-drift',key};
 }
 if(!res.complete || !res.fieldComplete)return {status:'unverified',why:'partial-result'};
 if(res.execution!=='synthetic-provider-observed')return {status:'unverified',why:'provider-witness-missing'};
 if(!permit || permit.principal!==now.principal ||
   permit.epoch!==now.grantEpoch||permit.policy!==now.policy ||
   permit.root!==now.root||permit.axis!==now.axis ||
   !permit.actions.includes(needed) ||
   permit.tenant!==now.tenant)
   return {status:'unverified',why:'no-current-consumer-permit'};
 if(!permit.routes.includes(res.route))return {status:'unverified',why:'route-not-allowed'};
 return {status:'toy-eligible',rows:res.rows.map(x=>x.id),consumer:needed};
}
const permit = (e=current())=>({
 principal:e.principal,epoch:e.grantEpoch,policy:e.policy,
 root:e.root,axis:e.axis,tenant:e.tenant,
 actions:['read'],routes:['list','detail'],
});
const newUI=()=>({
 visible:[],cache:[],pending:[],audit:[],dirtyPatch:null,
 selectId:'CASE-1',activeContext:current(),styleVersion:1,
});
function accept(state,res,perm){
 const decision=consumerResult(res,state.activeContext,perm);
 if(decision.status==='toy-eligible'){
   state.visible=decision.rows;
   state.cache=[{readKey:clone(res.request),ids:decision.rows}];
 } else {
   state.audit.push({code:decision.status,reason:decision.why});
 }
 return decision;
}
function requalify(state,next){
 const changes=fields.filter(k=>state.activeContext[k]!==next[k]);
 state.activeContext=clone(next);
 if(changes.length){
   state.visible=[];state.cache=[];state.pending=[];
   if(state.dirtyPatch)state.dirtyPatch={...state.dirtyPatch,quarantined:true};
 }
 return changes;
}
function requestCount(rawRows,scope,appliedBeforePage){
 if(!appliedBeforePage)return {status:'unverified',why:'post-filter-count'};
 const matching=rawRows.filter(x=>scope.includes(x.unit));
 return {status:'toy-count-only',count:matching.length,firstPage:matching.slice(0,1).map(x=>x.id)};
}

test('VE-01 current toy list can be accepted only with matching consumer permit',()=>{
 const s=newUI(),x=accept(s,response(),permit());
 assert.equal(x.status,'toy-eligible');assert.deepEqual(s.visible,['CASE-1']);
});
test('VE-02 policy update invalidates cached/visible data, not an external copy',()=>{
 const s=newUI();accept(s,response(),permit());
 const captured=s.visible.slice();
 const next=current();next.policy='org-policy-r2';
 assert.deepEqual(requalify(s,next),['policy']);
 assert.deepEqual(s.visible,[]);assert.deepEqual(s.cache,[]);
 assert.deepEqual(captured,['CASE-1']); // irrevocable already exported copies
});
test('VE-03 a late response from pre-revocation epoch cannot repopulate UI',()=>{
 const s=newUI(),old=response(),oldPermit=permit();
 const next=current();next.grantEpoch=5;requalify(s,next);
 assert.equal(accept(s,old,oldPermit).status,'stale');
 assert.deepEqual(s.visible,[]);assert.equal(s.audit.length,1);
});
test('VE-04 same selected Case ID with later selection epoch is still stale',()=>{
 const s=newUI(),old=response(),p=permit();
 const next=current();next.selectionEpoch=8;requalify(s,next);
 assert.equal(s.selectId,'CASE-1');
 assert.deepEqual(accept(s,old,p),{status:'stale',why:'context-drift',key:'selectionEpoch'});
});
test('VE-05 A→B→A selection does not resurrect an old read',()=>{
 const s=newUI(),old=response(),p=permit();
 let next=current();next.selectionEpoch=8;requalify(s,next);
 next=current();next.selectionEpoch=9;requalify(s,next);
 assert.equal(accept(s,old,p).status,'stale');
});
test('VE-06 scope root broadening never borrows narrow existing grant',()=>{
 const old=response(),next=current();next.root='H';
 assert.equal(consumerResult(old,next,permit()).status,'stale');
 assert.equal(consumerResult(response(next),next,permit()).why,'no-current-consumer-permit');
});
test('VE-07 changing hierarchy axis requires new permit',()=>{
 const now=current();now.axis='budget';
 assert.equal(consumerResult(response(now),now,permit()).status,'unverified');
});
test('VE-08 a new policy version requires new grant even with same records',()=>{
 const now=current();now.policy='org-policy-r2';
 assert.equal(consumerResult(response(now),now,permit()).why,'no-current-consumer-permit');
});
test('VE-09 an unavailable or failed attempt cannot be accepted as empty',()=>{
 const x=consumerResult(response(current(),{status:'unavailable',rows:[]}),current(),permit());
 assert.deepEqual(x,{status:'unverified',why:'attempt-not-successful'});
});
test('VE-10 partial field or row observation blocks complete count/claims',()=>{
 const x=consumerResult(response(current(),{complete:false}),current(),permit());
 const y=consumerResult(response(current(),{fieldComplete:false}),current(),permit());
 assert.equal(x.why,'partial-result');assert.equal(y.why,'partial-result');
});
test('VE-11 list permission is not an export/disclosure permit',()=>{
 assert.equal(consumerResult(response(),current(),permit(),'export').why,'no-current-consumer-permit');
});
test('VE-12 unsupported Provider routes cannot inherit list qualification',()=>{
 assert.equal(consumerResult(response(current(),{route:'count'}),current(),permit()).why,'route-not-allowed');
});
test('VE-13 visible UI and external count have different enforcement requirements',()=>{
 const data=[{id:'CASE-1',unit:'L1'},{id:'CASE-2',unit:'S'}];
 assert.equal(requestCount(data,['L1'],false).status,'unverified');
 assert.deepEqual(requestCount(data,['L1'],true),{status:'toy-count-only',count:1,firstPage:['CASE-1']});
});
test('VE-14 dirty edit survives as quarantined draft, not authorized submission',()=>{
 const s=newUI();accept(s,response(),permit());
 s.dirtyPatch={target:'CASE-1',delta:{note:'proposed'},quarantined:false};
 const next=current();next.grantEpoch=5;
 requalify(s,next);
 assert.equal(s.dirtyPatch.target,'CASE-1');
 assert.equal(s.dirtyPatch.quarantined,true);
 assert.deepEqual(s.visible,[]);
});
test('VE-15 unchanged policy and identity allow selective reuse after pure style change',()=>{
 const s=newUI();accept(s,response(),permit());
 s.styleVersion=2;
 assert.deepEqual(requalify(s,current()),[]);
 assert.deepEqual(s.visible,['CASE-1']);
});
test('VE-16 same source bytes under another resolved locator still need requalification',()=>{
 const next=current();next.sourceLocator='org-source-b';
 assert.deepEqual(consumerResult(response(),next,permit()),
   {status:'stale',why:'context-drift',key:'sourceLocator'});
});
test('VE-17 Query hidden predicate version change invalidates same visible rows',()=>{
 const next=current();next.queryFootprint='where-org-r2';
 assert.equal(consumerResult(response(),next,permit()).status,'stale');
});
test('VE-18 Provider cohort change is not equivalent to a retained screenshot',()=>{
 const next=current();next.readCohort='provider-c2';
 assert.equal(consumerResult(response(),next,permit()).status,'stale');
});
test('VE-19 current permit from another tenant cannot be reused',()=>{
 const p=permit();p.tenant='T2';
 assert.equal(consumerResult(response(),current(),p).why,'no-current-consumer-permit');
});
test('VE-20 a response loaded before revocation is not retroactively erasable outside',()=>{
 const previouslyDelivered={id:'CASE-1',exportedAt:'2027-07-01',contents:'prior lawful disclosure'};
 const s=newUI();accept(s,response(),permit());
 const next=current();next.grantEpoch=5;requalify(s,next);
 assert.deepEqual(s.visible,[]);
 assert.equal(previouslyDelivered.contents,'prior lawful disclosure');
});
