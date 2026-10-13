import test from 'node:test';
import assert from 'node:assert/strict';
import {compareTwoReferenceApplications,CrossApplicationObservationError,
  TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

const copy=structuredClone;
const make=()=>{
  const ids=['order_entry','repair_intake'];
  const apps=ids.map((id,i)=>({id,semanticUses:[],
    databases:[{alias:i===0?'master':'customer_master',target:'master_data',
      access:'read',requires:[{contract:'master_data.customer',compatibleMajor:2}]}],
    patterns:[{id:id+'.customer_reference',kind:'reference',
      relation:i===0?'sales.order.customer':'repair_intake.intake.customer',
      datasource:i===0?'master':'customer_master',
      contractQuery:{contract:'master_data.customer',query:'find_by_id',
        inputs:[{targetAttribute:'customer.customer.customer_id',input:'id'}]}}]}));
  const models=[apps[0],apps[0].patterns[0],apps[1],apps[1].patterns[0]];
  return TWO_APPLICATION_REFERENCE_SUBJECTS.map((s,i)=>({
    schemaVersion:'tei.cli.inspect/v0',ok:true,
    result:{schemaVersion:'tei.operations.inspect/v0',effect:'read-only',subject:{...s},
      canonical:copy(models[i]),worldFingerprint:'world:reference',
      sourceIndexFingerprint:'index:reference',sourceStatus:'indexed',
      sources:{subject:{...s},artifacts:[{path:'reference/'+i+'.tei',
        fingerprint:'sha256:synthetic-'+i}],
        references:[{subject:{...s},artifactPath:'reference/'+i+'.tei',role:'definition'}]}}
  }));
};
const denied=(code)=>e=>e instanceof CrossApplicationObservationError&&e.code===code;

test('same shared customer query is a declaration observation, never accepted reuse',()=>{
  const result=compareTwoReferenceApplications(make());
  assert.equal(result.sharedInvocationDeclared,true);
  assert.equal(result.observation,'shared_query_invocation_declared');
  assert.equal(result.reuseDecision,'needs_semantic_review');
  assert.equal(result.acceptance,'not_asserted');
  assert.equal(result.contractConformance,'not_verified');
  assert.deepEqual(result.contextDifferences,['relation','datasourceAlias']);
  assert.equal(result.declaredQuery.contract,'master_data.customer');
  assert.equal(result.declaredQuery.compatibleMajor,2);
});
test('different query is not reported as a shared invocation',()=>{
  const x=make();x[3].result.canonical.contractQuery.query='find_by_name';
  const r=compareTwoReferenceApplications(x);
  assert.equal(r.sharedInvocationDeclared,false);
  assert.equal(r.declaredQuery,null);
  assert.equal(r.reuseDecision,'needs_semantic_review');
});
test('different required contract major blocks shared invocation',()=>{
  const x=make();x[2].result.canonical.databases[0].requires[0].compatibleMajor=3;
  assert.equal(compareTwoReferenceApplications(x).sharedInvocationDeclared,false);
});
test('different binding does not silently qualify as shared query',()=>{
  const x=make();x[3].result.canonical.contractQuery.inputs[0].targetAttribute='other.customer_id';
  assert.equal(compareTwoReferenceApplications(x).sharedInvocationDeclared,false);
});
test('missing alias fails closed instead of inventing a datasource',()=>{
  const x=make();x[2].result.canonical.databases[0].alias='renamed';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('UNRESOLVED_DATASOURCE_ALIAS'));
});
test('write capable datasource is not silently treated as read only',()=>{
  const x=make();x[2].result.canonical.databases[0].access='read-write';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('UNVERIFIED_READ_ONLY_ROUTE'));
});
test('duplicate input binding cannot be converted to equivalent query signature',()=>{
  const x=make();x[3].result.canonical.contractQuery.inputs.push({targetAttribute:'second',input:'id'});
  assert.throws(()=>compareTwoReferenceApplications(x),denied('DUPLICATE_QUERY_BINDING'));
});
test('unknown database metadata cannot be ignored in a comparison',()=>{
  const x=make();x[2].result.canonical.databases[0].tenantScope='private';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('INCOMPLETE_SOURCE_OBSERVATION'));
});
test('unknown nested invocation cannot be silently ignored',()=>{
  const x=make();x[3].result.canonical.contractQuery.policy='strict';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('INCOMPLETE_SOURCE_OBSERVATION'));
});
test('wrong membership aborts despite matching contract and query',()=>{
  const x=make();x[2].result.canonical.patterns=[];
  assert.throws(()=>compareTwoReferenceApplications(x),denied('UNVERIFIED_APPLICATION_MEMBERSHIP'));
});
test('World drift aborts even when declarations match',()=>{
  const x=make();x[3].result.worldFingerprint='world:stale';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('SNAPSHOT_DRIFT'));
});
test('source index drift aborts even when declarations match',()=>{
  const x=make();x[2].result.sourceIndexFingerprint='index:other';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('SNAPSHOT_DRIFT'));
});
test('unindexed source cannot be promoted to shared declaration',()=>{
  const x=make();x[2].result.sourceStatus='not-indexed';x[2].result.sources.references=[];
  assert.throws(()=>compareTwoReferenceApplications(x),denied('INCOMPLETE_SOURCE_OBSERVATION'));
});
test('unknown canonical application field causes abstention',()=>{
  const x=make();x[0].result.canonical.extraPolicy='new';
  assert.throws(()=>compareTwoReferenceApplications(x),denied('INCOMPLETE_SOURCE_OBSERVATION'));
});
test('incorrect subject count fails before any projection',()=>{
  const x=make();x.pop();
  assert.throws(()=>compareTwoReferenceApplications(x),denied('INVALID_SUBJECT_BATCH'));
});
