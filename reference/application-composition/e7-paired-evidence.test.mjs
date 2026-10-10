import test from 'node:test';
import assert from 'node:assert/strict';
import {assessPairedEvidence} from './e7-paired-evidence.mjs';

const subject={kind:'application',id:'order_entry'};
const validation={kind:'semantic-validation',id:'sales.order_line.quantity_positive'};
const sha=s=>'sha256:'+s.repeat(64);
const fixture=()=>({
  format:'tei.e5-paired-source-observation.v1',
  qualification:'read-only-paired-input-not-an-authority',
  report:{
    application:'order_entry',status:'observed',
    qualification:'declaration-and-source-index-only',
    issues:[],dependencies:[
      {relation:'application',subject:structuredClone(subject),source:{path:'applications/order-entry/application.tei',fingerprint:sha('a')}},
      {relation:'uses-validation',subject:structuredClone(validation),source:{path:'semantic/sales/validation.tei',fingerprint:sha('b')}},
    ],
  },
  index:{
    artifacts:[
      {path:'applications/order-entry/application.tei',fingerprint:sha('a')},
      {path:'semantic/sales/validation.tei',fingerprint:sha('b')},
    ],
    references:[
      {subject:structuredClone(subject),artifactPath:'applications/order-entry/application.tei',role:'definition'},
      {subject:structuredClone(validation),artifactPath:'semantic/sales/validation.tei',role:'definition'},
    ],
  },
});
const check=(a,b)=>assessPairedEvidence(a,b);

test('unchanged paired observations remain bounded, never an acceptance',()=>{
  const a=fixture(),result=check(a,structuredClone(a));
  assert.equal(result.status,'limited-unchanged');
  assert.equal(result.dependencyCount,2);
  assert.equal(result.fullClosure,false);
  assert.equal(result.authority,'none');
});
test('changed explicitly referenced Validation forces requalification',()=>{
  const a=fixture(),b=structuredClone(a);
  b.report.dependencies[1].source.fingerprint=sha('c');
  b.index.artifacts[1].fingerprint=sha('c');
  assert.equal(check(a,b).reason,'indexed_source_changed');
});
test('unrelated Source Index artifact does not invalidate selected subjects',()=>{
  const a=fixture(),b=structuredClone(a);
  b.index.artifacts.push({path:'semantic/unrelated.tei',fingerprint:sha('c')});
  assert.equal(check(a,b).status,'limited-unchanged');
});
test('a new declared reference cannot be silently omitted',()=>{
  const a=fixture(),b=structuredClone(a);
  const added={kind:'semantic-validation',id:'sales.other'};
  b.report.dependencies.push({relation:'uses-validation',subject:added,
    source:{path:'semantic/other.tei',fingerprint:sha('d')}});
  b.index.artifacts.push({path:'semantic/other.tei',fingerprint:sha('d')});
  b.index.references.push({subject:added,artifactPath:'semantic/other.tei',role:'definition'});
  assert.equal(check(a,b).reason,'declared_dependency_set_changed');
});
test('a fake transport version does not become accepted provenance',()=>{
  const a=fixture(),b=structuredClone(a);b.format='tei.e5-unsafe.v9';
  assert.equal(check(a,b).reason,'invalid_or_unqualified_evidence_envelope');
});
test('a forged qualification cannot be promoted to authority',()=>{
  const a=fixture(),b=structuredClone(a);b.qualification='runtime-authorized';
  assert.equal(check(a,b).reason,'invalid_or_unqualified_evidence_envelope');
});
test('unresolved E5 report remains unresolved after pairing',()=>{
  const a=fixture(),b=structuredClone(a);
  b.report.status='unresolved';
  b.report.issues.push({reason:'missing_source'});
  assert.equal(check(a,b).reason,'incomplete_or_untrusted_report');
});
test('E5 report content and indexed Source must agree',()=>{
  const a=fixture(),b=structuredClone(a);
  b.report.dependencies[1].source.path='semantic/other.tei';
  assert.equal(check(a,b).reason,'report_index_source_mismatch');
});
test('missing paired Source Index refuses to infer dependency closure',()=>{
  const a=fixture(),b=structuredClone(a);
  delete b.index;
  assert.equal(check(a,b).reason,'invalid_or_unqualified_evidence_envelope');
});
test('application identity change cannot reuse old selection',()=>{
  const a=fixture(),b=structuredClone(a);
  b.report.application='other_application';
  assert.equal(check(a,b).reason,'wrong_application_identity');
});

test('paired evidence detects a changed ContractQuery semantic input source',()=>{
  const older=fixture(),newer=structuredClone(older);
  const subject={kind:'semantic-attribute',id:'customer.customer.customer_id'};
  const path='semantic/customer/customer.tei';
  const source={path,fingerprint:sha('e')};
  for (const evidence of [older,newer]) {
    evidence.report.dependencies.push({relation:'pattern-contract-query-input-attribute',
      subject:structuredClone(subject),source:structuredClone(source)});
    evidence.index.artifacts.push({path,fingerprint:sha('e')});
    evidence.index.references.push({subject:structuredClone(subject),artifactPath:path,role:'definition'});
  }
  assert.equal(check(older,newer).status,'limited-unchanged');
  newer.report.dependencies.at(-1).source.fingerprint=sha('f');
  newer.index.artifacts.at(-1).fingerprint=sha('f');
  assert.equal(check(older,newer).reason,'indexed_source_changed');
});
