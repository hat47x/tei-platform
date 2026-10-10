import test from 'node:test';
import assert from 'node:assert/strict';
import {assessParsedApplicationSourceDrift} from './e6-typed-index-bridge.mjs';

const app={kind:'application',id:'order_entry'};
const rule={kind:'semantic-validation',id:'sales.order_line.quantity_positive'};
const database={kind:'database',id:'sales_core'};
const contract={kind:'database-contract',id:'sales_core.order'};
const subjects=[
  {relation:'application',subject:app,path:'applications/order-entry/application.tei',digit:'a'},
  {relation:'uses-validation',subject:rule,path:'semantic/sales/validation.tei',digit:'b'},
  {relation:'database-target',subject:database,path:'databases/sales-core/database.tei',digit:'c'},
  {relation:'requires-contract',subject:contract,path:'databases/sales-core/contracts/order.tei',digit:'d'},
];
const sha=d=>'sha256:'+d.repeat(64);
const fixture=(entries=subjects)=>({
  report:{application:'order_entry',status:'observed',qualification:'declaration-and-source-index-only',
    issues:[],dependencies:entries.map(s=>({
      relation:s.relation,subject:structuredClone(s.subject),
      source:{path:s.path,fingerprint:sha(s.digit)}}))},
  index:{artifacts:entries.map(s=>({path:s.path,fingerprint:sha(s.digit)})),
    references:entries.map(s=>({subject:structuredClone(s.subject),artifactPath:s.path,role:'definition'}))},
});
const check=(old,newer)=>assessParsedApplicationSourceDrift({
  baselineReport:old.report,currentReport:newer.report,baselineIndex:old.index,currentIndex:newer.index,
});

test('identical explicit declarations produce only limited unchanged, never full closure',()=>{
  const v=fixture(),result=check(v,v);
  assert.deepEqual(result,{qualification:'application-declaration-and-direct-source-definitions-only',
    fullClosure:false,authority:'none',status:'limited-unchanged',dependencyCount:4});
});
test('changed Validation source makes application review unresolved',()=>{
  const older=fixture(),newer=fixture();newer.report.dependencies[1].source.fingerprint=sha('e');
  newer.index.artifacts[1].fingerprint=sha('e');
  const r=check(older,newer);
  assert.equal(r.status,'unresolved');
  assert.equal(r.reason,'indexed_source_changed');
  assert.deepEqual(r.affected.map(x=>x.subject),[rule]);
});
test('unrelated newly indexed artifact does not affect known dependencies',()=>{
  const old=fixture(),next=structuredClone(old);
  next.index.artifacts.push({path:'semantic/customer/customer.tei',fingerprint:sha('e')});
  assert.equal(check(old,next).status,'limited-unchanged');
});
test('newly added Validation declaration forces review even with complete index',()=>{
  const old=fixture(),extra={relation:'uses-validation',subject:{kind:'semantic-validation',
    id:'sales.order_line.line_number_positive'},path:'semantic/sales/new-rule.tei',digit:'e'};
  const next=fixture([...subjects,extra]);
  assert.equal(check(old,next).reason,'declared_dependency_set_changed');
});
test('report with unresolved definition cannot be used as an authority',()=>{
  const old=fixture(),next=fixture();next.report.status='unresolved';
  next.report.issues.push({reason:'source_definition_missing_or_ambiguous'});
  assert.equal(check(old,next).reason,'incomplete_or_untrusted_report');
});
test('report with changed qualification cannot be silently promoted',()=>{
  const old=fixture(),next=fixture();
  next.report.qualification='runtime-authorized';
  assert.equal(check(old,next).reason,'incomplete_or_untrusted_report');
});
test('injected dependency not in the actual Source Index fails closed',()=>{
  const old=fixture(),next=fixture();
  next.report.dependencies[1].source.path='semantic/sales/other.tei';
  assert.equal(check(old,next).reason,'report_index_source_mismatch');
});
test('a missing or duplicate definition Reference is not a verified join',()=>{
  const old=fixture(),next=fixture();next.index.references.splice(1,1);
  assert.equal(check(old,next).reason,'report_index_source_mismatch');
  const duplicated=fixture();duplicated.index.references.push({...duplicated.index.references[1]});
  assert.equal(check(old,duplicated).reason,'report_index_source_mismatch');
});
test('source fingerprint claim must match the Index artifact',()=>{
  const old=fixture(),next=fixture();next.index.artifacts[2].fingerprint=sha('f');
  assert.equal(check(old,next).reason,'report_index_source_mismatch');
});
test('duplicate E5 dependency cannot be treated as a selected winner',()=>{
  const old=fixture(),next=fixture();
  next.report.dependencies.push(structuredClone(next.report.dependencies[0]));
  assert.equal(check(old,next).reason,'duplicate_dependency_claim');
});
test('unexpected relation kind cannot be interpreted as a supported dependency',()=>{
  const old=fixture(),next=fixture();
  next.report.dependencies[1].relation='uses-pattern';
  assert.equal(check(old,next).reason,'invalid_dependency_claim');
});
test('application identity and source subject must agree',()=>{
  const old=fixture(),next=fixture();
  next.report.application='different_application';
  assert.equal(check(old,next).reason,'wrong_application_identity');
});
test('definition role replacement cannot claim the same trusted provenance',()=>{
  const old=fixture(),next=fixture();next.index.references[3].role='authoring';
  assert.equal(check(old,next).reason,'report_index_source_mismatch');
});
test('empty or malformed inputs return unresolved rather than exceptions or success',()=>{
  assert.equal(assessParsedApplicationSourceDrift({}).reason,'incomplete_or_untrusted_report');
  const old=fixture(),next=fixture();next.index.artifacts[0].fingerprint='not-a-sha';
  assert.equal(check(old,next).reason,'report_index_source_mismatch');
});

test('typed SemanticUses and Application Patterns are joined to E4 index subjects',()=>{
  const more=[
    {relation:'uses-semantic',subject:{kind:'semantic',id:'sales'},path:'semantic/sales/order.tei',digit:'e'},
    {relation:'application-pattern',subject:{kind:'application-pattern',id:'order_entry.submit_order'},
      path:'applications/order-entry/patterns/submit-order.tei',digit:'f'},
  ];
  const old=fixture([...subjects,...more]);
  const current=structuredClone(old);
  const result=check(old,current);
  assert.equal(result.status,'limited-unchanged');
  assert.equal(result.dependencyCount,6);
  current.index.artifacts[5].fingerprint=sha('a');
  current.report.dependencies[5].source.fingerprint=sha('a');
  const changed=check(old,current);
  assert.equal(changed.status,'unresolved');
  assert.deepEqual(changed.affected.map(x=>x.subject),[more[1].subject]);
});
test('missing typed Pattern or Semantic source cannot be inferred from a successful E5 declaration alone',()=>{
  const more=[
    {relation:'uses-semantic',subject:{kind:'semantic',id:'sales'},path:'semantic/sales/order.tei',digit:'e'},
    {relation:'application-pattern',subject:{kind:'application-pattern',id:'order_entry.submit_order'},
      path:'applications/order-entry/patterns/submit-order.tei',digit:'f'},
  ];
  const old=fixture([...subjects,...more]);
  const current=structuredClone(old);
  current.index.references.splice(5,1);
  assert.equal(check(old,current).reason,'report_index_source_mismatch');
  const stale=structuredClone(old);
  stale.report.dependencies.pop();
  assert.equal(check(old,stale).reason,'declared_dependency_set_changed');
});

test('Pattern capability dependency is not falsely treated as part of the covered closure',()=>{
  const old=fixture([...subjects,{
    relation:'application-pattern',subject:{kind:'application-pattern',id:'order_entry.submit_order'},
    path:'applications/order-entry/patterns/submit-order.tei',digit:'e',
  }]);
  const current=structuredClone(old);
  // A Pattern may refer to a Capability in a separate Source. E5 does not
  // derive this edge. The unchanged result must explicitly remain limited.
  current.index.artifacts.push({path:'semantic/sales/action.tei',fingerprint:sha('f')});
  current.index.references.push({
    subject:{kind:'semantic-capability',id:'sales.order.submit'},
    artifactPath:'semantic/sales/action.tei',role:'definition',
  });
  const result=check(old,current);
  assert.equal(result.status,'limited-unchanged');
  assert.equal(result.fullClosure,false);
  assert.equal(result.authority,'none');
});
