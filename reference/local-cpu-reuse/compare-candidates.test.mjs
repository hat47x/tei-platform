import test from 'node:test';
import assert from 'node:assert/strict';
import {CandidateInputError,comparePair,findReuseCandidates} from './compare-candidates.mjs';
const base={kind:'ui',source:{locator:'synthetic/app-a/ui.yaml',digest:'sha256:synthetic-a',revision:'r1'},
  contract:{id:'std.ui.header-detail',fingerprint:'fp:v1'},
  signature:{inputs:[{name:'record',type:'std.record',nullable:false}],outputs:[],effects:[],failures:['validation']},
  platform:{target:'web',framework:'lit'},labels:['order','entry']};
const a={...structuredClone(base),id:'app-a.order-entry',application:'app-a'};
const b={...structuredClone(base),id:'app-b.ticket-input',application:'app-b',labels:['ticket','capture'],source:{...base.source,locator:'synthetic/app-b/ui.yaml'}};
const clone=structuredClone;
const hasCode=code=>e=>e instanceof CandidateInputError && e.code===code;

test('same pinned UI contract and signature yield only a candidate despite different names',()=>{
  const r=comparePair(a,b);
  assert.equal(r.state,'reuse_candidate');assert.equal(r.acceptance,'not_accepted');assert.equal(r.authority,'read_only_candidate');
  assert.ok(r.lexicalHint < 0.5); // token overlap is only a hint, not authority
});
test('same business UI appearance but different semantic contract is not accepted as reuse',()=>{
  const x=clone(b);x.contract.id='expense.approve';x.labels=['order','entry'];
  const r=comparePair(a,x);assert.equal(r.state,'needs_semantic_review');assert.equal(r.reason,'unverified_semantic_equivalence');
});
test('signature and effects incompatibility dominates matching names and contract',()=>{
  const x=clone(b);x.signature.effects=['inventory.reserve'];x.labels=['order','entry'];
  assert.equal(comparePair(a,x).reason,'signature_or_effects_mismatch');
});
test('same semantic UI pattern across different framework is not treated as directly reusable UI implementation',()=>{
  const x=clone(b);x.platform.framework='react';
  assert.equal(comparePair(a,x).state,'pattern_candidate');
});
test('same contract ID but different fingerprint is contract drift',()=>{
  const x=clone(b);x.contract.fingerprint='fp:v2';
  assert.equal(comparePair(a,x).reason,'contract_fingerprint_drift');
});
test('missing verified contract requires review',()=>{
  const x=clone(b);delete x.contract;
  assert.equal(comparePair(a,x).reason,'missing_authoritative_contract');
});
test('different subject kind is incompatible even if surface text matches',()=>{
  const x=clone(b);x.kind='logic';delete x.platform;
  assert.equal(comparePair(a,x).reason,'different_function_kind');
});
test('explicit authorized scope suppresses excluded application without identity leak',()=>{
  const secret={...clone(a),id:'secret.payroll',application:'secret',source:{...a.source,locator:'private/payroll'}};
  const r=findReuseCandidates([a,b,secret],{allowedApplications:['app-a','app-b']});
  assert.equal(r.length,1);assert.equal(JSON.stringify(r).includes('secret'),false);assert.equal(JSON.stringify(r).includes('payroll'),false);
});
test('no implicit authorization for workspace-wide search',()=>{
  assert.throws(()=>findReuseCandidates([a,b]),hasCode('authorization_scope_required'));
});
test('stable rank and pairs regardless of input ordering',()=>{
  const x={...clone(b),id:'app-c.order-input',application:'app-c',source:{...b.source,locator:'synthetic/c'}};
  const opts={allowedApplications:['app-a','app-b','app-c']};
  assert.deepEqual(findReuseCandidates([a,x,b],opts),findReuseCandidates([b,a,x],opts));
});
test('duplicate IDs cannot silently overwrite source identities',()=>{
  const dup={...clone(b),id:a.id};
  assert.throws(()=>findReuseCandidates([a,dup],{allowedApplications:['app-a','app-b']}),hasCode('duplicate_id'));
});
test('missing source provenance and malformed input fail closed',()=>{
  const x=clone(b);x.source.revision='';
  assert.throws(()=>comparePair(a,x),hasCode('missing_source_provenance'));
});
test('same pinned business logic contract is only a candidate, never an accepted rule',()=>{
  const x=clone(a), y=clone(b);
  x.kind='logic';y.kind='logic';delete x.platform;delete y.platform;
  x.contract.id='std.logic.amount';y.contract.id='std.logic.amount';
  const result=comparePair(x,y);
  assert.equal(result.state,'reuse_candidate');assert.equal(result.acceptance,'not_accepted');
});
test('different business effects block direct substitution even when name and types match',()=>{
  const x=clone(a), y=clone(b);
  x.kind='logic';y.kind='logic';delete x.platform;delete y.platform;
  x.signature.effects=['order.update'];y.signature.effects=['invoice.post'];
  assert.equal(comparePair(x,y).state,'incompatible');
});
