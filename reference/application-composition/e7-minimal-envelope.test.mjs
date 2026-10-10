import test from 'node:test';
import assert from 'node:assert/strict';
import {minimizePairedSourceEvidence} from './e7-minimal-envelope.mjs';

const sha = c => 'sha256:' + c.repeat(64);
const app = {kind:'application',id:'order_entry'};
const rule = {kind:'semantic-validation',id:'sales.order_line.quantity_positive'};
const doc = {kind:'semantic-data',id:'sui.card'};
const fileApp = 'applications/order-entry/application.tei';
const fileRule = 'semantic/sales/validation.tei';
const fileExtra = 'sui/secret/private-design.tei';
const ref = (subject, artifactPath, role='definition') => ({subject,artifactPath,role});
const fixture = () => ({
  format:'tei.e5-paired-source-observation.v1',
  qualification:'read-only-paired-input-not-an-authority',
  report:{application:'order_entry', status:'observed', qualification:'declaration-and-source-index-only',
    issues:[],dependencies:[
      {relation:'application',subject:app,source:{path:fileApp,fingerprint:sha('a')}},
      {relation:'uses-validation',subject:rule,source:{path:fileRule,fingerprint:sha('b')}},
    ]},
  index:{artifacts:[
    {path:fileApp,fingerprint:sha('a')},{path:fileRule,fingerprint:sha('b')},
    {path:fileExtra,fingerprint:sha('c')},
  ],references:[ref(app,fileApp),ref(rule,fileRule),ref(doc,fileExtra)]},
});

test('only explicitly selected source evidence is disclosed', () => {
  const original=fixture(), got=minimizePairedSourceEvidence(original);
  assert.equal(got.status,'projected');
  assert.equal(got.fullClosure,false);
  assert.equal(got.authority,'none');
  assert.deepEqual(got.accounting,{selectedSubjects:2,sourceArtifacts:2,sourceReferences:2});
  assert.deepEqual(got.evidence.index.artifacts.map(a=>a.path),[fileApp,fileRule]);
  assert.ok(!JSON.stringify(got.evidence).includes('private-design'));
  assert.equal(original.index.artifacts.length,3);
});
test('selected authoring and implementation references are retained', () => {
  const data=fixture();data.index.references.push(ref(rule,fileApp,'authoring'),ref(rule,fileRule,'implementation'));
  const got=minimizePairedSourceEvidence(data);
  assert.equal(got.status,'projected');
  assert.equal(got.evidence.index.references.length,4);
});
test('changed irrelevant artifact never enters the exported projection', () => {
  const a=fixture(),b=fixture();b.index.artifacts[2].fingerprint=sha('d');
  assert.deepEqual(minimizePairedSourceEvidence(a),minimizePairedSourceEvidence(b));
});
test('missing or duplicate definition refuses a projection', () => {
  const a=fixture();a.index.references.splice(1,1);
  assert.equal(minimizePairedSourceEvidence(a).reason,'definition_mismatch');
  const b=fixture();b.index.references.push(ref(rule,fileRule));
  assert.equal(minimizePairedSourceEvidence(b).reason,'definition_mismatch');
});
test('changing a definition source does not silently select the new location', () => {
  const a=fixture();a.index.references[1].artifactPath=fileApp;
  assert.equal(minimizePairedSourceEvidence(a).reason,'definition_mismatch');
});
test('fingerprint mismatch or missing indexed artifact fails closed', () => {
  const a=fixture();a.index.artifacts[1].fingerprint=sha('e');
  assert.equal(minimizePairedSourceEvidence(a).reason,'fingerprint_mismatch');
  const b=fixture();b.index.artifacts.splice(1,1);
  assert.equal(minimizePairedSourceEvidence(b).reason,'missing_selected_artifact');
});
test('duplicate selected artifact is never a winner', () => {
  const a=fixture();a.index.artifacts.push({...a.index.artifacts[1]});
  assert.equal(minimizePairedSourceEvidence(a).reason,'ambiguous_artifact');
});
test('unresolved or fake-authority report is not distributable evidence', () => {
  const a=fixture();a.report.status='unresolved';
  assert.equal(minimizePairedSourceEvidence(a).reason,'incomplete_or_unqualified_input');
  const b=fixture();b.qualification='runtime-allowed';
  assert.equal(minimizePairedSourceEvidence(b).reason,'incomplete_or_unqualified_input');
});
test('missing input, unsupported path, and duplicate dependency are rejected', () => {
  assert.equal(minimizePairedSourceEvidence({}).reason,'incomplete_or_unqualified_input');
  const a=fixture();a.report.dependencies[0].source.path='../escaped.tei';
  assert.equal(minimizePairedSourceEvidence(a).reason,'invalid_dependency');
  const b=fixture();b.report.dependencies.push({...b.report.dependencies[1]});
  assert.equal(minimizePairedSourceEvidence(b).reason,'duplicate_dependency');
});
test('returned output is independent from the caller-owned Index', () => {
  const input=fixture(),out=minimizePairedSourceEvidence(input);
  out.evidence.index.artifacts[0].path='modified';
  assert.equal(input.index.artifacts[0].path,fileApp);
});

test('unrecognized fields are not leaked even when inserted into a selected reference', () => {
  const a=fixture();
  a.index.references[0].undocumentedSecret='PRIVATE-1';
  a.index.artifacts[0].credential='PRIVATE-2';
  a.report.credential='PRIVATE-3';
  a.report.dependencies[0].source.extra='PRIVATE-4';
  const out=minimizePairedSourceEvidence(a);
  assert.equal(out.status,'projected');
  assert.ok(!JSON.stringify(out.evidence).includes('PRIVATE-'));
});

test('E7a rejects a fake dependency relation before exposing its source path', () => {
  const a=fixture();
  a.report.dependencies[1].relation='undocumented-native-secret';
  assert.equal(minimizePairedSourceEvidence(a).reason,'unverified_dependency_report');
});

test('E7a requires the report to contain exactly its selected Application definition', () => {
  const absent=fixture();
  absent.report.dependencies.splice(0,1);
  assert.equal(minimizePairedSourceEvidence(absent).reason,'unverified_dependency_report');
  const changed=fixture();
  changed.report.application='unrelated_application';
  assert.equal(minimizePairedSourceEvidence(changed).reason,'unverified_dependency_report');
});

test('E7a will not export forged private Source via a fabricated report', () => {
  const a=fixture();
  a.report.dependencies=[{
    relation:'fabricated-secret',
    subject:{kind:'semantic-data',id:'private.internal'},
    source:{path:fileExtra,fingerprint:sha('c')},
  }];
  assert.equal(minimizePairedSourceEvidence(a).reason,'unverified_dependency_report');
});
