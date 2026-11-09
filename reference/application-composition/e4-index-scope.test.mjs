import test from 'node:test';
import assert from 'node:assert/strict';
import {assessIndexedSubjects} from './e4-index-scope.mjs';

const app={kind:'application',id:'order_entry'};
const validation={kind:'semantic-validation',id:'sales.order_line.line_number_positive'};
const main='applications/order-entry/application.tei';
const rules='semantic/sales/validation.tei';
const index=()=>({
  artifacts:[{path:main,fingerprint:'sha256:'+'a'.repeat(64)},{path:rules,fingerprint:'sha256:'+'b'.repeat(64)}],
  references:[{subject:app,artifactPath:main,role:'definition'},
    {subject:validation,artifactPath:rules,role:'definition'}],
});
const consumers=[{id:'application-only',subjects:[app]},{id:'validation-review',subjects:[validation]},
  {id:'explicit-application-and-validation',subjects:[app,validation]}];
const run=(currentIndex,baselineIndex=index(),uses=consumers)=>assessIndexedSubjects({baselineIndex,currentIndex,consumers:uses});

test('unchanged indexed subject content remains unchanged, without proving semantics',()=>{
  const outcome=run(index());
  assert.deepEqual(outcome.map(o=>o.status),['unchanged','unchanged','unchanged']);
  assert.equal(outcome[0].dependencyProvenance,'caller-supplied-subject-selection');
});
test('explicitly selected Validation change affects selected consumers only',()=>{
  const changed=index(); changed.artifacts[1].fingerprint='sha256:'+'c'.repeat(64);
  assert.deepEqual(run(changed).map(o=>o.status),['unchanged','unresolved','unresolved']);
});
test('a missing Application-to-Validation semantic edge cannot be inferred from Source Index',()=>{
  const changed=index(); changed.artifacts[1].fingerprint='sha256:'+'c'.repeat(64);
  assert.deepEqual(run(changed)[0].affected,[]); // A known limitation, not a compatibility claim.
});
test('irrelevant indexed artifact does not invalidate selected subjects',()=>{
  const changed=index(); changed.artifacts.push({path:'semantic/customer/data.tei',fingerprint:'sha256:'+'d'.repeat(64)});
  assert.equal(run(changed)[1].status,'unchanged');
});
test('source relocation requalifies even with identical content',()=>{
  const changed=index(); changed.artifacts[1].path='semantic/sales/validation-new.tei';
  changed.references[1].artifactPath='semantic/sales/validation-new.tei';
  assert.equal(run(changed)[1].affected[0].reason,'reference_set_changed');
});
test('role transition requalifies even with identical artifact fingerprint',()=>{
  const changed=index(); changed.references[1].role='authoring';
  assert.equal(run(changed)[1].affected[0].reason,'reference_set_changed');
});
test('absence of selected subject is unresolved',()=>{
  const changed=index(); changed.references.pop();
  assert.equal(run(changed)[1].affected[0].reason,'missing_indexed_subject');
});
test('changes to authoring contributions are considered where selected',()=>{
  const before=index();before.references.push({subject:app,artifactPath:rules,role:'authoring'});
  const after=structuredClone(before);after.artifacts[1].fingerprint='sha256:'+'d'.repeat(64);
  assert.equal(run(after,before)[0].status,'unresolved');
});
test('duplicate artifact path is not silently selected',()=>{
  const changed=index();changed.artifacts.push({...changed.artifacts[0]});
  assert.throws(()=>run(changed),/invalid_index_artifact/);
});
test('malformed or absent fingerprints are not accepted as unchanged',()=>{
  const changed=index();changed.artifacts[0].fingerprint='gitsha1-not-the-source-index-digest';
  assert.throws(()=>run(changed),/invalid_index_artifact/);
});
test('a dangling source reference fails closed',()=>{
  const changed=index();changed.references[0].artifactPath='missing.tei';
  assert.throws(()=>run(changed),/invalid_index_reference/);
});
test('invalid or duplicate consumer subject selections are rejected',()=>{
  assert.throws(()=>run(index(),index(),[{id:'x',subjects:[app,app]}]),/invalid_consumer_subjects/);
  assert.throws(()=>run(index(),index(),[{id:'x',subjects:[app]},{id:'x',subjects:[validation]}]),/invalid_consumer_subjects/);
});
