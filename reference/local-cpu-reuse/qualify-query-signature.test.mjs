import test from 'node:test';
import assert from 'node:assert/strict';
import {qualifyReferenceQuerySignature,QuerySignatureEvidenceError} from './qualify-query-signature.mjs';
import {TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

function fixture() {
  const contexts=[
    {id:'order_entry',alias:'master',relation:'sales.order.customer'},
    {id:'repair_intake',alias:'customer_master',relation:'repair_intake.intake.customer'}
  ];
  const applications=contexts.map(({id,alias,relation})=>({
    id,databases:[{alias,target:'master_data',access:'read',
      requires:[{contract:'master_data.customer',compatibleMajor:2}]}],
    patterns:[{id:id+'.customer_reference',kind:'reference',relation,datasource:alias,
      contractQuery:{contract:'master_data.customer',query:'find_by_id',
        inputs:[{targetAttribute:'customer.customer.customer_id',input:'id'}]}}]
  }));
  const world={
    applications,databases:[{id:'master_data',kind:'relational',ownership:'external-observed',
      contracts:[{id:'master_data.customer',version:{major:2,minor:4},
        reads:['customer.customer'],queries:[{
          name:'find_by_id',inputs:[{name:'id',type:'customer.customer_id'}],
          result:'customer.customer',resultCardinality:'zero-or-one'
        }]}]}],
    semantics:[{id:'customer',data:[{id:'customer.customer',
      attributes:[{id:'customer.customer.customer_id',type:'customer.customer_id'},
        {id:'customer.customer.name',type:'std.text'}]}]}]
  };
  const canonicals=[applications[0],applications[0].patterns[0],
    applications[1],applications[1].patterns[0]];
  const reports=TWO_APPLICATION_REFERENCE_SUBJECTS.map((s,i)=>({
    schemaVersion:'tei.cli.inspect/v0',ok:true,
    result:{schemaVersion:'tei.operations.inspect/v0',effect:'read-only',
      subject:{...s},canonical:structuredClone(canonicals[i]),
      worldFingerprint:'world:reference',sourceIndexFingerprint:'index:reference',
      sourceStatus:'indexed',sources:{
        subject:{...s},artifacts:[{path:'reference/'+i+'.tei',fingerprint:'sha256:syn-'+i}],
        references:[{subject:{...s},artifactPath:'reference/'+i+'.tei',role:'definition'}]
      }}
  }));
  return {world,reports};
}
const denied=code=>error=>error instanceof QuerySignatureEvidenceError&&error.code===code;

test('reads Go-shaped v2.4 customer Query input and zero-or-one semantic result',()=>{
  const {world,reports}=fixture();
  const q=qualifyReferenceQuerySignature(world,reports);
  assert.equal(q.status,'declared_signature_matched');
  assert.equal(q.contract.id,'master_data.customer');
  assert.deepEqual(q.contract.version,{major:2,minor:4});
  assert.deepEqual(q.inputTypes,[{input:'id',type:'customer.customer_id',
    targetAttribute:'customer.customer.customer_id'}]);
  assert.deepEqual(q.output,{semanticData:'customer.customer',cardinality:'zero-or-one'});
  assert.deepEqual(q.applicationContext.map(x=>x.datasourceAlias),['master','customer_master']);
  assert.equal(q.verified.typedDeclaration,true);
  for(const k of ['providerImplementation','resultRuntimeShape','authorization',
    'sourceAuthenticity','atomicWorkspaceSnapshot']) assert.equal(q.verified[k],false);
  assert.equal(q.executionPermitted,false);assert.equal(q.accepted,false);
});
test('contract minor release observed; incompatible major never passes',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts[0].version.major=3;
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('INCOMPATIBLE_CONTRACT_MAJOR'));
});
test('a changed Query input type is not inferred equivalent from its label',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts[0].queries[0].inputs[0].type='std.text';
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('QUERY_INPUT_TYPE_MISMATCH'));
});
test('missing, additional and duplicate Query inputs fail closed',()=>{
  for(const edits of [
    q=>q.inputs=[],
    q=>q.inputs.push({name:'limit',type:'std.number'}),
    q=>q.inputs.push({name:'id',type:'customer.customer_id'})
  ]) {
    const {world,reports}=fixture();edits(world.databases[0].contracts[0].queries[0]);
    assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('QUERY_INPUT_BINDINGS_INCOMPLETE'));
  }
});
test('a missing result contract must not be treated as zero-or-one',()=>{
  const {world,reports}=fixture();
  delete world.databases[0].contracts[0].queries[0].resultCardinality;
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('INCOMPLETE_QUERY_SIGNATURE'));
});
test('unknown result Data cannot qualify declared output shape',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts[0].queries[0].result='customer.unknown';
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('QUERY_RESULT_DATA_NOT_FOUND'));
});
test('unknown Query-side policy is not silently dropped',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts[0].queries[0].scopePolicy='tenant';
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('INCOMPLETE_QUERY_SIGNATURE'));
});
test('unknown Contract-side effects are rejected rather than treated as pure read',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts[0].retryWrites=true;
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('UNMODELED_CONTRACT'));
});
test('duplicate Contract identity is not silently resolved to first',()=>{
  const {world,reports}=fixture();
  world.databases[0].contracts.push(structuredClone(world.databases[0].contracts[0]));
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('CONTRACT_NOT_UNIQUE'));
});
test('duplicate database identity cannot certify the Query source',()=>{
  const {world,reports}=fixture();
  world.databases.push(structuredClone(world.databases[0]));
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('DATABASE_NOT_UNIQUE'));
});
test('different World Application compared with real inspect report fails closed',()=>{
  const {world,reports}=fixture();
  world.applications[0].patterns[0].relation='sales.order.other';
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('WORLD_INSPECT_APPLICATION_DRIFT'));
});
test('new World pattern differs from observed inspect Canonical and cannot qualify',()=>{
  const {world,reports}=fixture();
  world.applications[1].patterns[0].contractQuery.query='find_other';
  assert.throws(()=>qualifyReferenceQuerySignature(world,reports),denied('WORLD_INSPECT_APPLICATION_DRIFT'));
});
test('two different Query declarations return non-qualified result without execution',()=>{
  const {world,reports}=fixture();
  world.applications[1].patterns[0].contractQuery.query='find_by_name';
  reports[3].result.canonical.contractQuery.query='find_by_name';
  reports[2].result.canonical.patterns[0].contractQuery.query='find_by_name';
  const result=qualifyReferenceQuerySignature(world,reports);
  assert.equal(result.status,'not_qualified_different_invocations');
  assert.equal(result.querySignatureDeclared,false);
  assert.equal(result.executionPermitted,false);
  assert.equal(result.accepted,false);
});
test('the World input is not mutated or turned into a selected provider',()=>{
  const {world,reports}=fixture();const before=structuredClone(world);
  const result=qualifyReferenceQuerySignature(world,reports);
  result.inputTypes[0].type='wrong';
  assert.deepEqual(world,before);
});
