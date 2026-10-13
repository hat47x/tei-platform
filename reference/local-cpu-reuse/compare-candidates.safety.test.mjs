import test from 'node:test';
import assert from 'node:assert/strict';
import {CandidateInputError,comparePair,findReuseCandidates} from './compare-candidates.mjs';
const entry=(application,id)=>({id,application,kind:'ui',
  source:{locator:application+'/form',digest:'sha256:synthetic',revision:'r1'},
  contract:{id:'std.ui.example',fingerprint:'synthetic-v1'},
  signature:{inputs:[{name:'customer',type:'string',nullable:false}],outputs:[],effects:['read'],failures:['denied']},
  platform:{target:'web',framework:'lit'},labels:['customer']});
const a=entry('app-a','app-a.a'),b=entry('app-b','app-b.b');
const hasCode=code=>e=>e instanceof CandidateInputError && e.code===code;

test('duplicate effects fail closed instead of being silently de-duplicated',()=>{
  const x=structuredClone(b);x.signature.effects=['read','read'];
  assert.throws(()=>comparePair(a,x),hasCode('duplicate_signature_item'));
});
test('duplicate failures fail closed instead of being silently de-duplicated',()=>{
  const x=structuredClone(b);x.signature.failures=['denied','denied'];
  assert.throws(()=>comparePair(a,x),hasCode('duplicate_signature_item'));
});
test('unknown semantic signature properties cannot disappear during equivalence check',()=>{
  const x=structuredClone(b);x.signature.transaction='serializable';
  assert.throws(()=>comparePair(a,x),hasCode('invalid_signature'));
});
test('unknown contract and typed-field semantics fail closed',()=>{
  const x=structuredClone(b);x.contract.version='breaking-v2';
  assert.throws(()=>comparePair(a,x),hasCode('invalid_contract'));
  const y=structuredClone(b);y.signature.inputs[0].constraint='validated-id';
  assert.throws(()=>comparePair(a,y),hasCode('invalid_field'));
});
test('unsupported top-level semantics and runtime-specific platforms fail closed',()=>{
  const x=structuredClone(b);x.policy='tenant-scoped';
  assert.throws(()=>comparePair(a,x),hasCode('unsupported_record_field'));
  const y=structuredClone(b);y.platform.host='native-shell';
  assert.throws(()=>comparePair(a,y),hasCode('missing_ui_platform'));
});
test('duplicate ID in a direct comparison fails closed',()=>{
  const x=structuredClone(b);x.id=a.id;
  assert.throws(()=>comparePair(a,x),hasCode('duplicate_id'));
});
test('positive and finite maxRecord/maxPair bounds are mandatory',()=>{
  const base={allowedApplications:['app-a','app-b']};
  for (const options of [{...base,maxPairs:Infinity},{...base,maxPairs:0},{...base,maxRecords:0},{...base,maxRecords:4097},{...base,maxPairs:65537}]) {
    assert.throws(()=>findReuseCandidates([a,b],options),hasCode('invalid_limits'));
  }
  assert.throws(()=>findReuseCandidates([a,b],null),hasCode('invalid_options'));
});
test('bound records before generating any pair results',()=>{
  assert.throws(()=>findReuseCandidates([a,b],{allowedApplications:['app-a','app-b'],maxRecords:1}),hasCode('record_limit_exceeded'));
});
test('cross-application pair count bounded, same-app-only records do not count as pairs',()=>{
  const c=entry('app-c','app-c.c');
  const options={allowedApplications:['app-a','app-b','app-c'],maxPairs:2};
  assert.throws(()=>findReuseCandidates([a,b,c],options),hasCode('pair_limit_exceeded'));
  const d=entry('app-a','app-a.d');
  assert.deepEqual(findReuseCandidates([a,d],{allowedApplications:['app-a'],maxPairs:1}),[]);
});
test('permutation of effects/failures and fields preserves deterministic comparison',()=>{
  const x=structuredClone(b);x.signature.effects=['update','read'];x.signature.failures=['gone','denied'];
  const y=structuredClone(a);y.signature.effects=['read','update'];y.signature.failures=['denied','gone'];
  assert.equal(comparePair(y,x).state,'reuse_candidate');
});
test('excluded application does not consume visible record/pair budget',()=>{
  const secret=entry('other','other.secret');
  assert.equal(findReuseCandidates([a,b,secret],{allowedApplications:['app-a','app-b'],maxRecords:2,maxPairs:1}).length,1);
});
