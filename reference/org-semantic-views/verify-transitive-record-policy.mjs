/**
 * ORG-VP: organization-path record-policy pressure (synthetic only).
 * No real identity, Grant, enforcement, Provider filtering, Studio, or Host Effect.
 * node --test reference/org-semantic-views/verify-transitive-record-policy.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';

const at = Object.freeze({ validAt:'2027-06-15',knownAt:'2027-07-01' });
const orgs = Object.freeze({
 H:{tenant:'T1'}, D1:{tenant:'T1'}, L1:{tenant:'T1'},
 L2:{tenant:'T1'}, S:{tenant:'T1'}, B:{tenant:'T1'},
 X:{tenant:'T2'},
});
const edge = (id,from,to,axis='command',extra={})=>({
 id,from,to,axis,validFrom:'2027-01-01',validTo:null,
 knownFrom:'2027-01-01',knownTo:null,...extra
});
const graph = () => ({
 revision:'org-graph-r1',complete:true,orgs,
 edges:[
  edge('E1','H','D1'),
  edge('E2','D1','L1'),
  edge('E3-old','D1','L2','command',{knownTo:'2027-07-03'}),
  edge('E3-corrected','D1','L2','command',{knownFrom:'2027-07-03',validTo:'2027-06-01'}),
  edge('E4','H','S'),
  edge('E5','S','L2'),
  edge('E6','D1','B','budget'),
 ],
});
const record=(id,ownerOrgs,extra={})=>({
 id,kind:'order',tenant:'T1',ownerOrgs,linksComplete:true,...extra
});
const R={
 r1: record('R1',['L1']),
 r2: record('R2',['L2']),
 r3: record('R3',['S']),
 r4: record('R4',['X'],{tenant:'T2'}),
 r5: record('R5',[]),
 r6: record('R6',['L1','X']),
 r7: record('R7',['L1']),
 r8: record('R8',['L1']),
 r9: record('R9',['L1'],{kind:'org_metadata'}),
};
const policy=()=>({
 id:'org-scope-order-policy',revision:'p-r1',tenant:'T1',dataKinds:['order'],
 axis:'command',scopeRoot:'D1',validAt:at.validAt,knownAt:at.knownAt,
 sourceRevision:'org-graph-r1',mode:'any',strictTree:false,active:true,
 actions:['read','update'],
 overrides:{R2:'read-only',R7:'exclude-from-this-policy',R8:'deny'},
});
const inWindow=(d,a,b)=>a<=d&&(b===null||d<b);
const clone=x=>JSON.parse(JSON.stringify(x));
function closure(g,p,maxDepth=20) {
 const root=p.scopeRoot,axis=p.axis,cut={validAt:p.validAt,knownAt:p.knownAt};
 const paths=[{org:root,nodes:[root],edgeIds:[]}],diagnostics=new Set();
 let truncated=false;
 const explore=(path)=>{
  const eligible=g.edges.filter(e=>e.from===path.org&&e.axis===axis&&
    inWindow(cut.validAt,e.validFrom,e.validTo)&&inWindow(cut.knownAt,e.knownFrom,e.knownTo));
  for(const e of eligible) {
   if(!g.orgs[e.to]||!g.orgs[e.from]||g.orgs[e.to].tenant!==p.tenant||g.orgs[e.from].tenant!==p.tenant){
     diagnostics.add('cross-tenant-or-unresolved-edge');continue;
   }
   if(path.nodes.includes(e.to)){diagnostics.add('cycle');continue;}
   if(path.edgeIds.length>=maxDepth){truncated=true;continue;}
   const next={org:e.to,nodes:[...path.nodes,e.to],edgeIds:[...path.edgeIds,e.id]};
   paths.push(next);explore(next);
  }
 };
 if(!g.orgs[root]||g.orgs[root].tenant!==p.tenant)diagnostics.add('invalid-scope-root');
 else explore(paths[0]);
 const parents=new Map();
 for(const edge of g.edges.filter(e=>e.axis===axis&&
    inWindow(cut.validAt,e.validFrom,e.validTo)&&inWindow(cut.knownAt,e.knownFrom,e.knownTo)&&
    g.orgs[e.from]?.tenant===p.tenant&&g.orgs[e.to]?.tenant===p.tenant)) {
   if(!parents.has(edge.to))parents.set(edge.to,new Set());
   parents.get(edge.to).add(edge.from);
 }
 if([...parents.values()].some(v=>v.size>1))diagnostics.add('multiple-parents');
 return {paths,nodes:[...new Set(paths.map(x=>x.org))],
   diagnostics:[...diagnostics].sort(),truncated,complete:g.complete};
}
const result=(status,reason,extra={})=>({status,reason,...extra});
function evaluate(g,p,r,depth=20) {
 if(!p.active||!p.dataKinds.includes(r.kind))return result('not-applicable','outside-policy');
 const override=p.overrides[r.id];
 if(override==='exclude-from-this-policy')return result('not-applicable','explicit-exclusion-not-permission');
 if(override==='deny')return result('deny-candidate','explicit-record-deny');
 if(r.tenant!==p.tenant)return result('deny-candidate','record-tenant-boundary');
 if(r.ownerOrgs.some(x=>!g.orgs[x]||g.orgs[x].tenant!==r.tenant))
   return result('deny-candidate','cross-tenant-or-unresolved-owner-link');
 if(g.revision!==p.sourceRevision)return result('unverified','stale-graph-revision');
 if(!['any','all'].includes(p.mode))return result('unverified','owner-rule-missing');
 const c=closure(g,p,depth);
 if(c.diagnostics.includes('invalid-scope-root')||c.diagnostics.includes('cycle')||
   (p.strictTree&&c.diagnostics.includes('multiple-parents')))
   return result('unverified','graph-structure-or-policy-profile');
 const matched=r.ownerOrgs.filter(x=>c.nodes.includes(x));
 const positive=p.mode==='any'?matched.length>0:(r.ownerOrgs.length>0&&matched.length===r.ownerOrgs.length&&r.linksComplete);
 if(positive) {
   const actions=override==='read-only'?['read']:p.actions.slice();
   return result('policy-eligible','scope-positive-path',{
     actions,matched,scopeRoot:p.scopeRoot,axis:p.axis,tenant:p.tenant,graphRevision:g.revision,policyRevision:p.revision,
     assurance:'synthetic-policy-candidate-only',
   });
 }
 if(!g.complete||c.truncated||!r.linksComplete)return result('unverified','open-path-or-link-scope');
 return result('deny-candidate','no-scope-path');
}
function hostSimulation(candidate,grant,action,ctx) {
 if(candidate.status!=='policy-eligible')return result('not-authorized','policy-not-eligible');
 if(!candidate.actions.includes(action))return result('not-authorized','read-only-or-operation-excluded');
 if(!grant||!grant.current||grant.epoch!==ctx.grantEpoch||
    grant.principal!==ctx.principal||grant.tenant!==ctx.tenant||
    grant.scopeRoot!==candidate.scopeRoot||grant.axis!==candidate.axis||
    grant.issuedPolicyRevision!==candidate.policyRevision||
    !grant.actions.includes(action))return result('unverified','actual-host-grant-not-established');
 return result('simulated-host-candidate','test-grant-is-not-real-provider-enforcement');
}
const hostGrant={current:true,epoch:4,principal:'U1',tenant:'T1',scopeRoot:'D1',axis:'command',
 issuedPolicyRevision:'p-r1',actions:['read','update']};
const hostCtx={principal:'U1',tenant:'T1',grantEpoch:4};

test('VP-01 typed command closure follows D1 children, not siblings',()=>{
 const c=closure(graph(),policy());
 assert.deepEqual(c.nodes,['D1','L1','L2']);assert.equal(c.nodes.includes('S'),false);
});
test('VP-02 paths and nodes are separate for multiple-parent DAG',()=>{
 const p=policy();p.scopeRoot='H';
 const c=closure(graph(),p);
 assert.equal(c.nodes.filter(x=>x==='L2').length,1);
 assert.equal(c.paths.filter(x=>x.org==='L2').length,2);
 assert.equal(c.diagnostics.includes('multiple-parents'),true);
});
test('VP-03 budget hierarchy is not command hierarchy',()=>{
 const c=closure(graph(),policy());
 assert.equal(c.nodes.includes('B'),false);
 const p=policy();p.axis='budget';assert.equal(evaluate(graph(),p,R.r1).status,'deny-candidate');
});
test('VP-04 cyclical hierarchy is detected, not silently expanded into authority',()=>{
 const g=graph();g.edges.push(edge('E-cycle','L1','D1'));
 const c=closure(g,policy());
 assert.equal(c.diagnostics.includes('cycle'),true);
 assert.equal(evaluate(g,policy(),R.r1).status,'unverified');
});
test('VP-05 depth limit is incomplete, not proof of no descendant',()=>{
 const p=policy();p.scopeRoot='H';
 const c=closure(graph(),p,1);
 assert.equal(c.truncated,true);
 assert.equal(evaluate(graph(),p,R.r1,1).status,'unverified');
});
test('VP-06 broad policy is explicitly scoped to a target Data kind',()=>{
 assert.equal(evaluate(graph(),policy(),R.r9).status,'not-applicable');
});
test('VP-07 ordinary organization metadata is not automatically protected',()=>{
 assert.equal(evaluate(graph(),policy(),R.r9).reason,'outside-policy');
});
test('VP-08 a verified Org path yields only a policy candidate',()=>{
 const x=evaluate(graph(),policy(),R.r1);
 assert.equal(x.status,'policy-eligible');assert.deepEqual(x.actions,['read','update']);
 assert.equal(x.assurance,'synthetic-policy-candidate-only');
});
test('VP-09 a sibling Org under same root is outside scoped reachability',()=>{
 assert.deepEqual(evaluate(graph(),policy(),R.r3),result('deny-candidate','no-scope-path'));
});
test('VP-10 cross-tenant record is never admitted via an Org link',()=>{
 assert.equal(evaluate(graph(),policy(),R.r4).reason,'record-tenant-boundary');
});
test('VP-11 mixed-tenant owner links are not bypassed by any-path selection',()=>{
 assert.equal(evaluate(graph(),policy(),R.r6).reason,'cross-tenant-or-unresolved-owner-link');
});
test('VP-12 per-record read-only exception preserves read but not update candidate',()=>{
 const x=evaluate(graph(),policy(),R.r2);
 assert.deepEqual(x.actions,['read']);
 assert.equal(hostSimulation(x,hostGrant,'update',hostCtx).status,'not-authorized');
});
test('VP-13 exclusion from this policy is not a public allow',()=>{
 const x=evaluate(graph(),policy(),R.r7);
 assert.deepEqual(x,result('not-applicable','explicit-exclusion-not-permission'));
 assert.equal(hostSimulation(x,hostGrant,'read',hostCtx).status,'not-authorized');
});
test('VP-14 explicit denial prevails over a positive scope path',()=>{
 assert.equal(evaluate(graph(),policy(),R.r8).reason,'explicit-record-deny');
});
test('VP-15 disabled broad policy does not turn into a permission',()=>{
 const p=policy();p.active=false;
 assert.equal(evaluate(graph(),p,R.r1).status,'not-applicable');
});
test('VP-16 known-at correction changes scoped reachability, not identity',()=>{
 const first=evaluate(graph(),policy(),R.r2);
 const later=policy();later.knownAt='2027-07-03';
 assert.equal(first.status,'policy-eligible');
 assert.equal(evaluate(graph(),later,R.r2).status,'deny-candidate');
 assert.equal(R.r2.ownerOrgs[0],'L2');
});
test('VP-17 valid-at earlier than retroactive cutoff preserves earlier path',()=>{
 const p=policy();p.validAt='2027-05-15';p.knownAt='2027-07-03';
 assert.equal(evaluate(graph(),p,R.r2).status,'policy-eligible');
});
test('VP-18 partial graph cannot prove missing path',()=>{
 const g=graph();g.complete=false;
 assert.equal(evaluate(g,policy(),R.r3).status,'unverified');
 assert.equal(evaluate(g,policy(),R.r1).status,'policy-eligible');
});
test('VP-19 a partial link set is safe for positive any, not negative all',()=>{
 const partial=record('RP',['L1'],{linksComplete:false});
 assert.equal(evaluate(graph(),policy(),partial).status,'policy-eligible');
 const p=policy();p.mode='all';
 assert.equal(evaluate(graph(),p,partial).status,'unverified');
});
test('VP-20 zero links with complete/unknown scope do not have same evidence',()=>{
 const x=R.r5;
 assert.deepEqual(evaluate(graph(),policy(),x),result('deny-candidate','no-scope-path'));
 assert.equal(evaluate(graph(),policy(),{...x,linksComplete:false}).status,'unverified');
});
test('VP-21 multi-org records need an explicit ANY/ALL business rule',()=>{
 const r=record('RM',['L1','S']);
 const p=policy();assert.equal(evaluate(graph(),p,r).status,'policy-eligible');
 p.mode='all';assert.equal(evaluate(graph(),p,r).status,'deny-candidate');
 p.mode='unset';assert.equal(evaluate(graph(),p,r).status,'unverified');
});
test('VP-22 strict single-parent profile rejects multiple parents',()=>{
 const p=policy();p.scopeRoot='H';p.strictTree=true;
 assert.equal(evaluate(graph(),p,R.r1).status,'unverified');
});
test('VP-23 a graph revision mismatch requalifies claims',()=>{
 const g=graph();g.revision='org-graph-r2';
 assert.equal(evaluate(g,policy(),R.r1).reason,'stale-graph-revision');
});
test('VP-24 policy eligibility does not establish current real Host Grant',()=>{
 const c=evaluate(graph(),policy(),R.r1);
 assert.equal(hostSimulation(c,null,'read',hostCtx).status,'unverified');
 assert.equal(hostSimulation(c,{...hostGrant,epoch:3},'read',hostCtx).status,'unverified');
});
test('VP-25 even matching synthetic Host grant has no real effect receipt',()=>{
 const c=evaluate(graph(),policy(),R.r1);
 const x=hostSimulation(c,hostGrant,'read',hostCtx);
 assert.equal(x.status,'simulated-host-candidate');
 assert.equal(Object.hasOwn(x,'effectReceipt'),false);
});
test('VP-26 nonmatching principal/tenant cannot be inferred from membership',()=>{
 const c=evaluate(graph(),policy(),R.r1);
 assert.equal(hostSimulation(c,hostGrant,'read',{...hostCtx,principal:'U2'}).status,'unverified');
 assert.equal(hostSimulation(c,hostGrant,'read',{...hostCtx,tenant:'T2'}).status,'unverified');
});
test('VP-27 external disclosure and SHO/AI export are never in generic read scope',()=>{
 const c=evaluate(graph(),policy(),R.r1);
 assert.equal(c.actions.includes('export'),false);
 assert.equal(hostSimulation(c,hostGrant,'export',hostCtx).status,'not-authorized');
});
test('VP-28 query count/paging cannot be attested by local post-filter candidates',()=>{
 const all=[R.r1,R.r2,R.r3,R.r4,R.r5,R.r6,R.r7,R.r8];
 const candidateRows=all.filter(r=>evaluate(graph(),policy(),r).status==='policy-eligible');
 assert.deepEqual(candidateRows.map(x=>x.id),['R1','R2']);
 const returnValue={sampleRows:candidateRows.length,actualProviderPolicyEnforced:false};
 assert.equal(returnValue.actualProviderPolicyEnforced,false);
 assert.notEqual(returnValue.sampleRows,all.length);
});
test('VP-29 changed policy revision requires consumer requalification',()=>{
 const old=evaluate(graph(),policy(),R.r1);
 const next=policy();next.revision='p-r2';
 const curr=evaluate(graph(),next,R.r1);
 assert.notEqual(old.policyRevision,curr.policyRevision);
 assert.equal(old.status,curr.status);
});
test('VP-30 cycle, multiple-path and tenant restrictions are axis-dependent',()=>{
 const g=graph();g.edges.push(edge('E-cross','D1','X'));
 const c=closure(g,policy());
 assert.equal(c.diagnostics.includes('cross-tenant-or-unresolved-edge'),true);
 assert.equal(c.nodes.includes('X'),false);
});

test('VP-31 a Host grant for D1 does not authorize policy expansion to H',()=>{
 const p=policy();p.scopeRoot='H';
 const candidate=evaluate(graph(),p,R.r1);
 assert.equal(candidate.status,'policy-eligible');
 assert.equal(hostSimulation(candidate,hostGrant,'read',hostCtx).status,'unverified');
});
test('VP-32 a command grant is not a budget-scope grant',()=>{
 const p=policy();p.axis='budget';
 const candidate=evaluate(graph(),p,record('RB',['B']));
 assert.equal(candidate.status,'policy-eligible');
 assert.equal(hostSimulation(candidate,hostGrant,'read',hostCtx).status,'unverified');
});
test('VP-33 changing policy revision invalidates an old Host binding',()=>{
 const p=policy();p.revision='p-r2';
 const candidate=evaluate(graph(),p,R.r1);
 assert.equal(candidate.status,'policy-eligible');
 assert.equal(hostSimulation(candidate,hostGrant,'read',hostCtx).status,'unverified');
});
