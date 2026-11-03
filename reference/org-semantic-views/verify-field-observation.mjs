/** Read-only synthetic ORG-FO field observation pressure; NOT actual TEI or Provider evidence.
 * Run: node --test reference/org-semantic-views/verify-field-observation.mjs */
import test from 'node:test';
import assert from 'node:assert/strict';
const rows=[{id:'A1',person:'P1',dept:'D1'},{id:'A2',person:'P2',dept:'D1'}];
const val=(v,c='c1')=>({state:'value',value:v,cohort:c});
const snapshot=()=>({rows,complete:true,fields:{A1:val(100),A2:val(60)},capacity:val(180),
  cohort:'c1',source:'r1',rolePin:'role-r1',route:'primary',snapshot:'s7',read:'synthetic-assumed'});
const cp=x=>JSON.parse(JSON.stringify(x));
const display=o=>o.rows.map(({id,person,dept})=>({id,person,dept}));
const fault=(status,reason)=>({status,reason});
function field(o,id) {
 const x=o.fields[id];
 if(!x) return fault('unverified','field-unobserved');
 if(x.cohort!==o.cohort||!o.read) return fault('unverified','cohort-or-read');
 if(x.state==='null')return fault('unresolved','null-rule-unknown');
 if(x.state!=='value'||!Number.isFinite(x.value))return fault('unverified','field-state');
 return {status:'resolved',value:x.value};
}
function sum(o) {
 if(!o.complete)return fault('unverified','row-scope-partial');
 const v=o.rows.map(r=>field(o,r.id));
 const x=v.find(x=>x.status!=='resolved');
 return x||{status:'resolved',value:v.reduce((a,b)=>a+b.value,0)};
}
function capacityRule(o) {
 const x=sum(o);if(x.status!=='resolved')return x;
 if(!o.capacity||o.capacity.cohort!==o.cohort)return fault('unverified','capacity-read');
 if(o.capacity.state!=='value')return fault('unresolved','capacity-null-rule');
 return {status:'toy-rule',verdict:x.value<=o.capacity.value?'satisfied':'not-satisfied'};
}
function filter(o,limit) {
 if(!o.complete)return fault('unverified','partial-filter');
 const v=o.rows.map(r=>({id:r.id,x:field(o,r.id)}));
 if(v.some(r=>r.x.status!=='resolved'))return fault('unverified','predicate-field-unobserved');
 return {status:'resolved',ids:v.filter(r=>r.x.value>=limit).map(r=>r.id)};
}
function sort(o) {
 if(!o.complete)return fault('unverified','partial-sort');
 const v=o.rows.map(r=>({id:r.id,x:field(o,r.id)}));
 if(v.some(r=>r.x.status!=='resolved'))return fault('unverified','order-field-unobserved');
 return {status:'resolved',ids:v.sort((a,b)=>b.x.value-a.x.value||a.id.localeCompare(b.id)).map(r=>r.id)};
}
function responsibility(r,complete) {
 const unique=new Set(r.map(x=>x.org));
 if(unique.size>1)return 'positive-conflict-witness';
 if(!complete)return 'unverified';
 return unique.size?'single-responsibility':'unassigned-for-scope';
}
test('FO-01 complete rows do not prove fields',()=>{const x=snapshot();x.fields={};assert.equal(display(x).length,2);assert.equal(sum(x).status,'unverified')});
test('FO-02 partial fields block aggregate',()=>{const x=snapshot();delete x.fields.A2;assert.deepEqual(sum(x),fault('unverified','field-unobserved'))});
test('FO-03 observed null is not zero',()=>{const x=snapshot();x.fields.A2={state:'null',cohort:'c1'};assert.deepEqual(sum(x),fault('unresolved','null-rule-unknown'))});
test('FO-04 observed zero retains membership',()=>{const x=snapshot();x.fields.A2=val(0);assert.equal(sum(x).value,100);assert.equal(display(x).length,2)});
test('FO-05 complete inputs give toy-rule satisfied',()=>assert.equal(capacityRule(snapshot()).verdict,'satisfied'));
test('FO-06 capacity-only change flips rule, display unchanged',()=>{const a=snapshot(),b=cp(a);b.capacity=val(150);assert.deepEqual(display(a),display(b));assert.equal(capacityRule(b).verdict,'not-satisfied')});
test('FO-07 hidden WHERE field missing blocks result',()=>{const x=snapshot();delete x.fields.A2;assert.equal(filter(x,70).status,'unverified');assert.deepEqual(filter(snapshot(),70).ids,['A1'])});
test('FO-08 hidden ORDER field missing blocks ordering',()=>{const x=snapshot();delete x.fields.A1;assert.equal(sort(x).status,'unverified');assert.deepEqual(sort(snapshot()).ids,['A1','A2'])});
test('FO-09 partial rows keep positive witness but not all-FTE',()=>{const x=snapshot();x.rows=[rows[0]];x.complete=false;assert.equal(field(x,'A1').value,100);assert.equal(sum(x).status,'unverified')});
test('FO-10 two responsibility witnesses violate single rule despite partial',()=>{assert.equal(responsibility([{org:'D1a'},{org:'D1b'}],false),'positive-conflict-witness');assert.equal(responsibility([],false),'unverified');assert.equal(responsibility([],true),'unassigned-for-scope')});
test('FO-11 same snapshot label is not same read cohort',()=>{const x=snapshot();x.fields.A2=val(60,'other');assert.equal(x.snapshot,'s7');assert.equal(sum(x).status,'unverified')});
test('FO-12 changed Role Pin invalidates rule not generic display',()=>{const x=snapshot();const state=x.rolePin!=='role-r2'?'unverified':capacityRule(x).status;assert.equal(display(x).length,2);assert.equal(state,'unverified')});
test('FO-13 published report and current restatement are different references',()=>{const p={report:'R1',knownAt:'2027-07-01',fte:160},c={report:'R2',knownAt:'2027-07-03',fte:100};assert.notEqual(p.knownAt,c.knownAt);assert.equal(p.fte,160);assert.equal(c.fte,100)});
test('FO-14 same selected occurrence cannot validate old epoch',()=>{const r1={edge:'A2',epoch:7},r2={edge:'A2',epoch:8};assert.equal(r1.edge,r2.edge);assert.notEqual(r1.epoch,r2.epoch)});
test('FO-15 displayed columns cannot identify physical read set',()=>{const d=Object.keys(display(snapshot())[0]);const physical=[...d,'allocation'];assert.equal(d.includes('allocation'),false);assert.equal(physical.includes('allocation'),true)});
test('FO-16 protected specific fields do not classify all org data',()=>{const f={dept:{protected:false},taxId:{protected:true}};assert.deepEqual(Object.keys(f).filter(k=>!f[k].protected),['dept']);assert.equal(f.taxId.protected,true)});
