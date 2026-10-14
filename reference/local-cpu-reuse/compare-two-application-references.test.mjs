import test from 'node:test';
import assert from 'node:assert/strict';
import {compareTwoReferenceApplications,proposeReferenceQueryAdapter,
  CrossApplicationObservationError,TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

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

test('proposal separates shared Query invocation from two Application contexts',()=>{
  const reports=make(),result=proposeReferenceQueryAdapter(reports);
  assert.equal(result.schemaVersion,'tei.research.query-invocation-adapter-proposal/v0');
  assert.equal(result.status,'candidate_requires_semantic_review');
  assert.deepEqual(result.commonInvocation,{
    databaseTarget:'master_data',contract:'master_data.customer',compatibleMajor:2,
    query:'find_by_id',bindings:[{targetAttribute:'customer.customer.customer_id',input:'id'}]
  });
  assert.deepEqual(result.applicationContext,[
    {application:'order_entry',pattern:'order_entry.customer_reference',
      relation:'sales.order.customer',datasourceAlias:'master'},
    {application:'repair_intake',pattern:'repair_intake.customer_reference',
      relation:'repair_intake.intake.customer',datasourceAlias:'customer_master'}
  ]);
  assert.deepEqual(result.contextDifferences,['relation','datasourceAlias']);
  assert.equal(result.commonInvocation.relation,undefined);
  assert.equal(result.commonInvocation.datasourceAlias,undefined);
  assert.equal(result.applicationContext[0].contract,undefined);
  assert.equal(result.accepted,false);
  assert.equal(result.executionPermitted,false);
  assert.equal(result.implementationSelected,false);
  assert.equal(result.authority,'derived_observation_only');
  assert.ok(Object.values(result.verified).every(x=>typeof x==='boolean'));
  assert.equal(result.verified.declarationMatch,true);
  assert.equal(result.verified.contractSemantics,false);
  assert.equal(result.verified.readAuthorization,false);
});
test('query drift keeps local context but provides no common adapter',()=>{
  const x=make();x[3].result.canonical.contractQuery.query='find_by_name';
  const p=proposeReferenceQueryAdapter(x);
  assert.equal(p.status,'not_shareable_from_declarations');
  assert.equal(p.commonInvocation,null);
  assert.equal(p.verified.declarationMatch,false);
  assert.equal(p.applicationContext.length,2);
  assert.equal(p.executionPermitted,false);
});
test('compatible major drift precludes the common invocation',()=>{
  const x=make();x[2].result.canonical.databases[0].requires[0].compatibleMajor=3;
  const p=proposeReferenceQueryAdapter(x);
  assert.equal(p.commonInvocation,null);
  assert.equal(p.status,'not_shareable_from_declarations');
});
test('different semantic binding precludes the common invocation',()=>{
  const x=make();x[3].result.canonical.contractQuery.inputs[0].targetAttribute='customer.customer.external_id';
  assert.equal(proposeReferenceQueryAdapter(x).commonInvocation,null);
});
test('alias resolution failure aborts adapter proposal',()=>{
  const x=make();x[2].result.canonical.databases[0].alias='wrong';
  assert.throws(()=>proposeReferenceQueryAdapter(x),denied('UNRESOLVED_DATASOURCE_ALIAS'));
});
test('non-read capability cannot create a query adapter proposal',()=>{
  const x=make();x[2].result.canonical.databases[0].access='read-write';
  assert.throws(()=>proposeReferenceQueryAdapter(x),denied('UNVERIFIED_READ_ONLY_ROUTE'));
});
test('stale snapshot aborts proposal instead of carrying partial wiring',()=>{
  const x=make();x[1].result.sourceIndexFingerprint='index:other';
  assert.throws(()=>proposeReferenceQueryAdapter(x),denied('SNAPSHOT_DRIFT'));
});
test('unmodeled Canonical policy aborts proposal instead of omitting a constraint',()=>{
  const x=make();x[3].result.canonical.contractQuery.policy='private';
  assert.throws(()=>proposeReferenceQueryAdapter(x),denied('INCOMPLETE_SOURCE_OBSERVATION'));
});
test('proposal is a detached copy; modifying it never rewrites observed reports',()=>{
  const input=make(),p=proposeReferenceQueryAdapter(input);
  p.commonInvocation.bindings[0].targetAttribute='mutated';
  p.applicationContext[0].relation='mutated';
  assert.equal(input[1].result.canonical.contractQuery.inputs[0].targetAttribute,
    'customer.customer.customer_id');
  assert.equal(input[1].result.canonical.relation,'sales.order.customer');
});
