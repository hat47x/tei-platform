/**
 * ORG-TS synthetic what-if overlay.
 * Read-only, standalone pressure test against the merged org-synthetic-r7 source.
 * This is NOT TEI execution, Owner-approved meaning, a Provider read, or Host Authority.
 * Run: node --test reference/org-semantic-views/verify-transition-simulation.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, membershipVersionsAt } from './source-fixture.mjs';

const CUT = Object.freeze({validAt:'2027-06-15',knownAt:'2027-07-01'});
const CAPACITY_REVISION = 'capacity-r1';
const RULE_REVISION = 'cap-rule-r1';
const clone = x => JSON.parse(JSON.stringify(x));
const ids = x => x.map(r=>r.id).sort();
const points = rows => rows.reduce((n,x)=>n+x.points,0);
const distinctPersons = rows => new Set(rows.map(x=>x.person)).size;
const byDept = rows => {
  const r={};
  for(const m of rows)r[m.dept]=(r[m.dept]||0)+m.points;
  return r;
};

function sourceSnapshot(cut = CUT) {
 const membership = membershipVersionsAt(fixture,cut.validAt,cut.knownAt);
 const allocation = new Map(fixture.allocations.map(x=>[x.relation,x.basisPoints]));
 const rows=membership.map(m=>{
   if(!allocation.has(m.id))throw Error('synthetic allocation missing: '+m.id);
   return {id:m.id,person:m.person,dept:m.dept,points:allocation.get(m.id)};
 });
 if(new Set(ids(rows)).size!==rows.length)throw Error('duplicate active relation ID');
 return {
   sourceRevision:fixture.sourceRevision,
   cut:{...cut},
   rows,
   capacityRevision:CAPACITY_REVISION,
   ruleRevision:RULE_REVISION,
   capacities:Object.fromEntries(fixture.departmentCapacities.map(x=>[x.department,x.basisPoints])),
   occupancy:clone(fixture.occupancy),
   caseResponsibility: [{caseId:'Q1',org:'D1',authority:'unknown'}],
   qualification:{collectionComplete:true,fieldsComplete:true,cohort:'synthetic-assumed'},
 };
}
const BASE = sourceSnapshot();
const realloc = Object.freeze({
 id:'plan-reallocate-r1', sourceRevision:fixture.sourceRevision,cut:{...CUT},
 ruleRevision:RULE_REVISION,
 changes:[
  {kind:'retire-occurrence',id:'A2'},
  {kind:'change-allocation',id:'A3',from:40,to:100}
 ],
});
const split = Object.freeze({
 id:'plan-split-r1', sourceRevision:fixture.sourceRevision,cut:{...CUT},
 ruleRevision:RULE_REVISION,
 changes:[
  {kind:'retire-occurrence',id:'A1'},
  {kind:'retire-occurrence',id:'A2'},
  {kind:'add-occurrence',record:{id:'A1new',person:'P1',dept:'D1a',points:100}},
  {kind:'add-occurrence',record:{id:'A2new',person:'P2',dept:'D1b',points:60}},
 ],
});
function preview(b,plan) {
 if(plan.sourceRevision!==b.sourceRevision||JSON.stringify(plan.cut)!==JSON.stringify(b.cut)||
    plan.ruleRevision!==b.ruleRevision)return {status:'stale-input',reason:'source/cut/rule-pin'};
 const s=clone(b);
 const active=new Map(s.rows.map(r=>[r.id,r]));
 for(const ch of plan.changes){
   if(ch.kind==='retire-occurrence'){
     if(!active.has(ch.id))return {status:'rejected',reason:'missing-target'};
     active.delete(ch.id);
   } else if(ch.kind==='change-allocation'){
     const target=active.get(ch.id);
     if(!target||target.points!==ch.from||!Number.isFinite(ch.to)||ch.to<0)
       return {status:'rejected',reason:'stale-or-invalid-allocation'};
     target.points=ch.to;
   } else if(ch.kind==='add-occurrence'){
     const r=ch.record;
     if(!r||active.has(r.id)||!fixture.persons.includes(r.person)||!Number.isFinite(r.points)||r.points<0)
       return {status:'rejected',reason:'invalid-new-occurrence'};
     active.set(r.id,clone(r));
   } else return {status:'rejected',reason:'unknown-change-kind'};
 }
 s.rows=[...active.values()].sort((a,b)=>a.id.localeCompare(b.id));
 return {status:'simulation-only',planId:plan.id,sourceRevision:b.sourceRevision,
   baselineCut:clone(b.cut),provenance:'what-if-overlay-only',
   effects:[],grant:null,ownerAcceptance:null,proposal:s};
}
function capacityCheck(s,capacity = s.capacities) {
 const q=s.qualification;
 if(!q.collectionComplete||!q.fieldsComplete||q.cohort!=='synthetic-assumed')
   return {status:'unverified',reason:'row/field/cohort'};
 const amounts=byDept(s.rows);
 const excess={};
 for(const [dept,n] of Object.entries(amounts)){
   if(!Object.hasOwn(capacity,dept)||!Number.isFinite(capacity[dept]))
     return {status:'unverified',reason:'capacity-not-observed',department:dept};
   if(n>capacity[dept])excess[dept]=n-capacity[dept];
 }
 return {status:'toy-only',verdict:Object.keys(excess).length?'violated':'satisfied',excess};
}
function compareFte(before,after,bridge) {
 if(!bridge||bridge.accepted!==true||bridge.measure!=='fte-points'||
   bridge.sourceRevision!==before.sourceRevision||after.sourceRevision!==before.sourceRevision||
   bridge.old.join(',')!=='D1'||bridge.successors.join(',')!=='D1a,D1b')
   return {status:'unresolved',reason:'no-compatible-accepted-comparison'};
 const sumFor=(s,departments)=>points(s.rows.filter(m=>departments.includes(m.dept)));
 return {status:'comparison-only',old:sumFor(before,bridge.old),successors:sumFor(after,bridge.successors)};
}

test('TS-01 baseline source cut has independent occurrence IDs and distinct Persons',()=>{
 assert.deepEqual(ids(BASE.rows),['A1','A2','A3','A4']);
 assert.deepEqual(byDept(BASE.rows),{D1:160,D2:140});
 assert.equal(points(BASE.rows),300);
 assert.equal(distinctPersons(BASE.rows),3);
});
test('TS-02 what-if realloc changes only its copied proposal, not source',()=>{
 const before=JSON.stringify(BASE),p=preview(BASE,realloc);
 assert.equal(p.status,'simulation-only');
 assert.deepEqual(ids(p.proposal.rows),['A1','A3','A4']);
 assert.deepEqual(byDept(p.proposal.rows),{D1:100,D2:200});
 assert.equal(JSON.stringify(BASE),before);
 assert.equal(fixture.membershipVersions.some(x=>x.id==='A2'),true);
});
test('TS-03 same world total does not guarantee same membership grain',()=>{
 const p=preview(BASE,realloc).proposal;
 assert.equal(points(p.rows),points(BASE.rows));
 assert.notEqual(p.rows.length,BASE.rows.length);
 assert.equal(distinctPersons(p.rows),3);
});
test('TS-04 scenario cannot act as approved effect or grant',()=>{
 const p=preview(BASE,realloc);
 assert.equal(p.provenance,'what-if-overlay-only');
 assert.deepEqual(p.effects,[]);
 assert.equal(p.grant,null);
 assert.equal(p.ownerAcceptance,null);
});
test('TS-05 independent overlays do not contaminate one another',()=>{
 const a=preview(BASE,realloc),b=preview(BASE,split);
 assert.equal(ids(a.proposal.rows).includes('A1new'),false);
 assert.equal(ids(b.proposal.rows).includes('A1new'),true);
 assert.deepEqual(ids(BASE.rows),['A1','A2','A3','A4']);
});
test('TS-06 trial capacity rule flags reallocation excess on D2',()=>{
 assert.deepEqual(capacityCheck(BASE),{status:'toy-only',verdict:'satisfied',excess:{}});
 assert.deepEqual(capacityCheck(preview(BASE,realloc).proposal),
   {status:'toy-only',verdict:'violated',excess:{D2:40}});
});
test('TS-07 candidate capacity relaxation is not business acceptance',()=>{
 const p=preview(BASE,realloc);
 const changed={...p.proposal.capacities,D2:220};
 assert.equal(capacityCheck(p.proposal,changed).verdict,'satisfied');
 assert.equal(p.ownerAcceptance,null);
 assert.deepEqual(p.effects,[]);
});
test('TS-08 missing successor capacity must not inherit old department capacity',()=>{
 const p=preview(BASE,split);
 assert.equal(capacityCheck(p.proposal).status,'unverified');
 assert.equal(capacityCheck(p.proposal).reason,'capacity-not-observed');
});
test('TS-09 partial row or missing allocation observations block optimistic result',()=>{
 const p=preview(BASE,realloc).proposal;
 p.qualification.collectionComplete=false;
 assert.equal(capacityCheck(p).status,'unverified');
 p.qualification.collectionComplete=true;p.qualification.fieldsComplete=false;
 assert.equal(capacityCheck(p).status,'unverified');
});
test('TS-10 mismatch source revision is rejected as stale',()=>{
 assert.equal(preview(BASE,{...realloc,sourceRevision:'different-source'}).status,'stale-input');
});
test('TS-11 same valid date but later known date invalidates old plan basis',()=>{
 const corrected=sourceSnapshot({validAt:CUT.validAt,knownAt:'2027-07-03'});
 assert.equal(preview(corrected,realloc).status,'stale-input');
 assert.equal(corrected.rows.some(r=>r.id==='A2'),false);
});
test('TS-12 change in Rule revision needs plan requalification',()=>{
 const changed=clone(BASE);changed.ruleRevision='cap-rule-r2';
 assert.equal(preview(changed,realloc).status,'stale-input');
});
test('TS-13 optimistic edit fails on stale allocation or unknown target',()=>{
 assert.deepEqual(preview(BASE,{...realloc,changes:[{kind:'change-allocation',id:'A3',from:60,to:100}]}),
   {status:'rejected',reason:'stale-or-invalid-allocation'});
 assert.equal(preview(BASE,{...realloc,changes:[{kind:'retire-occurrence',id:'A2new'}]}).status,'rejected');
});
test('TS-14 an unrecognized operation type never executes as best-effort',()=>{
 assert.deepEqual(preview(BASE,{...realloc,changes:[{kind:'commit',id:'A2'}]}),
   {status:'rejected',reason:'unknown-change-kind'});
});
test('TS-15 occupancy is independent of new department allocation',()=>{
 const p=preview(BASE,realloc);
 assert.deepEqual(p.proposal.occupancy,BASE.occupancy);
 assert.equal(p.proposal.occupancy.find(x=>x.id==='O1').person,'P2');
});
test('TS-16 Case responsibility does not migrate with member reallocation or split',()=>{
 const a=preview(BASE,realloc),b=preview(BASE,split);
 assert.deepEqual(a.proposal.caseResponsibility,BASE.caseResponsibility);
 assert.deepEqual(b.proposal.caseResponsibility,BASE.caseResponsibility);
 assert.equal(b.proposal.caseResponsibility[0].authority,'unknown');
});
test('TS-17 split uses new Org and Membership IDs while preserving distinct Persons',()=>{
 const p=preview(BASE,split).proposal;
 assert.deepEqual(ids(p.rows),['A1new','A2new','A3','A4']);
 assert.deepEqual(byDept(p.rows),{D1a:100,D1b:60,D2:140});
 assert.equal(distinctPersons(p.rows),3);
 assert.equal(points(p.rows),300);
});
test('TS-18 successor comparison cannot occur from a split alone',()=>{
 const p=preview(BASE,split).proposal;
 assert.equal(compareFte(BASE,p,null).status,'unresolved');
});
test('TS-19 a limited comparison bridge never grants identity or authority',()=>{
 const p=preview(BASE,split),bridge={accepted:true,measure:'fte-points',
  sourceRevision:BASE.sourceRevision,old:['D1'],successors:['D1a','D1b']};
 assert.deepEqual(compareFte(BASE,p.proposal,bridge),{status:'comparison-only',old:160,successors:160});
 assert.equal(compareFte(BASE,p.proposal,{...bridge,measure:'persons'}).status,'unresolved');
 assert.equal(p.grant,null);
});
test('TS-20 duplicate or unknown Person occurrence is rejected',()=>{
 const duplicate={...split,changes:[{kind:'add-occurrence',record:{id:'A1',person:'P1',dept:'D1b',points:100}}]};
 const stranger={...split,changes:[{kind:'add-occurrence',record:{id:'AX',person:'ghost',dept:'D1b',points:100}}]};
 assert.equal(preview(BASE,duplicate).status,'rejected');
 assert.equal(preview(BASE,stranger).status,'rejected');
});
