import test from 'node:test';
import assert from 'node:assert/strict';
import {observeReferenceProviderRouting,QueryProviderRoutingError} from './observe-provider-routing.mjs';
import {TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

// Minimal normalized Go-shaped Reference, no actual credentials, plugin runtime or SQL.
function fixture(){
  const apps=[
    {id:'order_entry',alias:'master',relation:'sales.order.customer'},
    {id:'repair_intake',alias:'customer_master',relation:'repair_intake.intake.customer'}
  ].map(({id,alias,relation})=>({
    id,databases:[{alias,target:'master_data',access:'read',
      requires:[{contract:'master_data.customer',compatibleMajor:2}]}],
    patterns:[{id:id+'.customer_reference',kind:'reference',relation,
      datasource:alias,contractQuery:{contract:'master_data.customer',query:'find_by_id',
        inputs:[{targetAttribute:'customer.customer.customer_id',input:'id'}]}}]
  }));
  const world={
    applications:apps,
    databases:[{id:'master_data',kind:'relational',ownership:'external-observed',
      contracts:[{id:'master_data.customer',version:{major:2,minor:4},
        reads:['customer.customer'],queries:[{name:'find_by_id',
          inputs:[{name:'id',type:'customer.customer_id'}],
          result:'customer.customer',resultCardinality:'zero-or-one'}]}]}],
    semantics:[{id:'customer',data:[{id:'customer.customer',
      attributes:[{id:'customer.customer.customer_id',type:'customer.customer_id'}]}]}],
    environments:[{id:'development',
      instances:[{id:'master_data.dev',definition:'master_data',provider:'oracle'}],
      bindings:[{application:'order_entry',alias:'master',instance:'master_data.dev'},
        {application:'repair_intake',alias:'customer_master',instance:'master_data.dev'}]}]
  };
  const models=[apps[0],apps[0].patterns[0],apps[1],apps[1].patterns[0]];
  const reports=TWO_APPLICATION_REFERENCE_SUBJECTS.map((s,i)=>({
    schemaVersion:'tei.cli.inspect/v0',ok:true,
    result:{schemaVersion:'tei.operations.inspect/v0',effect:'read-only',
      subject:{...s},canonical:structuredClone(models[i]),
      worldFingerprint:'world:fixture',sourceIndexFingerprint:'index:fixture',
      sourceStatus:'indexed',sources:{subject:{...s},
        artifacts:[{path:'reference/'+i+'.tei',fingerprint:'sha256:fixture-'+i}],
        references:[{subject:{...s},artifactPath:'reference/'+i+'.tei',role:'definition'}]}}
  }));
  return {world,reports};
}
const rejected=code=>e=>e instanceof QueryProviderRoutingError&&e.code===code;

test('two distinct aliases explicitly route to one Oracle instance without runtime authority',()=>{
  const {world,reports}=fixture(),r=observeReferenceProviderRouting(world,reports);
  assert.equal(r.status,'application_routes_declared');
  assert.equal(r.environment,'development');
  assert.deepEqual(r.target,{database:'master_data',contract:'master_data.customer',
    query:'find_by_id'});
  assert.deepEqual(r.routes.map(x=>x.alias),['master','customer_master']);
  assert.deepEqual(r.routes.map(x=>x.instance),['master_data.dev','master_data.dev']);
  assert.deepEqual(r.routes.map(x=>x.provider),['oracle','oracle']);
  assert.equal(r.sameInstanceDeclared,true);
  assert.equal(r.verified.applicationRouting,true);
  assert.equal(r.verified.pluginExecutorResolution,false);
  assert.equal(r.verified.providerHealth,false);
  assert.equal(r.verified.callerAuthorization,false);
  assert.equal(r.executionPermitted,false);
  assert.equal(r.implementationSelected,false);
  assert.equal(r.accepted,false);
});
test('optional missing Application binding is a visible gap, not a fake default',()=>{
  const {world,reports}=fixture();
  world.environments[0].bindings.pop();
  const r=observeReferenceProviderRouting(world,reports);
  assert.equal(r.status,'application_routes_incomplete');
  assert.equal(r.routes[1].status,'application_binding_not_declared');
  assert.equal(r.routes[1].instance,null);
  assert.equal(r.sameInstanceDeclared,false);
  assert.equal(r.executionPermitted,false);
});
test('duplicate App-alias binding refuses ambiguous connection',()=>{
  const {world,reports}=fixture();
  world.environments[0].bindings.push({...world.environments[0].bindings[0]});
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('AMBIGUOUS_APPLICATION_INSTANCE_BINDING'));
});
test('unknown Database instance prevents implicit provider fallback',()=>{
  const {world,reports}=fixture();
  world.environments[0].bindings[0].instance='unknown';
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('INVALID_DATABASE_INSTANCE_ROUTE'));
});
test('a Database instance of another definition cannot satisfy this target',()=>{
  const {world,reports}=fixture();
  world.environments[0].instances[0].definition='sales_core';
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('INVALID_DATABASE_INSTANCE_ROUTE'));
});
test('unsupported provider is not inferred as Oracle',()=>{
  const {world,reports}=fixture();
  world.environments[0].instances[0].provider='undocumented';
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('INVALID_DATABASE_INSTANCE_ROUTE'));
});
test('explicit executor binding is observation only, not plugin resolution',()=>{
  const {world,reports}=fixture();
  world.contractQueryExecutorBindings=[{environment:'development',instance:'master_data.dev',
    target:{database:'master_data',contract:'master_data.customer',query:'find_by_id'},
    executor:'database.query.oracle'}];
  const r=observeReferenceProviderRouting(world,reports);
  assert.deepEqual(r.routes.map(x=>x.executor),
    ['explicit_binding_declared_unverified','explicit_binding_declared_unverified']);
  assert.equal(r.verified.pluginExecutorResolution,false);
  assert.equal(r.executionPermitted,false);
  assert.equal(r.accepted,false);
});
test('same provider capability is never auto-selected without target binding',()=>{
  const {world,reports}=fixture();
  world.contractQueryExecutorBindings=[{environment:'development',instance:'master_data.dev',
    target:{database:'master_data',contract:'master_data.customer',query:'find_other'},
    executor:'database.query.oracle'}];
  const r=observeReferenceProviderRouting(world,reports);
  assert.deepEqual(r.routes.map(x=>x.executor),['not_observed','not_observed']);
});
test('executor bound in a different Environment cannot count',()=>{
  const {world,reports}=fixture();
  world.contractQueryExecutorBindings=[{environment:'production',instance:'master_data.dev',
    target:{database:'master_data',contract:'master_data.customer',query:'find_by_id'},
    executor:'database.query.oracle'}];
  const r=observeReferenceProviderRouting(world,reports);
  assert.deepEqual(r.routes.map(x=>x.executor),['not_observed','not_observed']);
});
test('duplicate executor binding is ambiguous and never chosen',()=>{
  const {world,reports}=fixture(),b={environment:'development',
    instance:'master_data.dev',
    target:{database:'master_data',contract:'master_data.customer',query:'find_by_id'},
    executor:'database.query.oracle'};
  world.contractQueryExecutorBindings=[b,{...b}];
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('AMBIGUOUS_EXECUTOR_BINDING'));
});
test('unmodeled explicit binding fields are not silently discarded',()=>{
  const {world,reports}=fixture();
  world.contractQueryExecutorBindings=[{environment:'development',instance:'master_data.dev',
    target:{database:'master_data',contract:'master_data.customer',query:'find_by_id'},
    executor:'database.query.oracle',policy:'new'}];
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('INVALID_EXECUTOR_BINDING'));
});
test('invalid selected Application route may not discard a new tenant constraint',()=>{
  const {world,reports}=fixture();
  world.environments[0].bindings[0].tenant='other';
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('INVALID_APPLICATION_INSTANCE_BINDING'));
});
test('distinct declared instances do not imply the same live Database',()=>{
  const {world,reports}=fixture();
  world.environments[0].instances.push({id:'master_data.other',definition:'master_data',provider:'oracle'});
  world.environments[0].bindings[1].instance='master_data.other';
  const r=observeReferenceProviderRouting(world,reports);
  assert.equal(r.status,'application_routes_declared');
  assert.equal(r.sameInstanceDeclared,false);
  assert.equal(r.executionPermitted,false);
});
test('missing selected Environment is not substituted with development',()=>{
  const {world,reports}=fixture();
  assert.throws(()=>observeReferenceProviderRouting(world,reports,{environmentId:'production'}),
    rejected('ENVIRONMENT_NOT_FOUND'));
});
test('duplicate Environment identity is an error, not first-item selection',()=>{
  const {world,reports}=fixture();
  world.environments.push(structuredClone(world.environments[0]));
  assert.throws(()=>observeReferenceProviderRouting(world,reports),
    rejected('ENVIRONMENT_NOT_UNIQUE'));
});
test('a non-matching typed Query returns no Provider route or execution permission',()=>{
  const {world,reports}=fixture();
  world.applications[1].patterns[0].contractQuery.query='find_other';
  reports[2].result.canonical.patterns[0].contractQuery.query='find_other';
  reports[3].result.canonical.contractQuery.query='find_other';
  const r=observeReferenceProviderRouting(world,reports);
  assert.equal(r.status,'query_signature_not_qualified');
  assert.deepEqual(r.routes,[]);
  assert.equal(r.executionPermitted,false);
});
