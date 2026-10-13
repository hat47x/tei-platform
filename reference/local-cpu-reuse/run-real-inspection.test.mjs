import test from 'node:test';
import assert from 'node:assert/strict';
import {runReferenceInspection} from './run-real-inspection.mjs';
import {InspectBridgeError} from './inspect-cli-bridge.mjs';

const fixtureCanonical={
  'application:order_entry':{id:'order_entry',patterns:[
    {id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'},
    {id:'order_entry.customer_reference',kind:'reference',relation:'sales.order.customer'}]},
  'application-pattern:order_entry.order_lines':{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'},
  'semantic-relation:sales.order.lines':{id:'sales.order.lines',target:'sales.order_line',cardinality:'many',keys:[]},
  'semantic-data:sales.order':{id:'sales.order',attributes:[],relations:[],candidateKeys:[],functionalDependencies:[]},
  'application-pattern:order_entry.customer_reference':{id:'order_entry.customer_reference',kind:'reference',relation:'sales.order.customer',datasource:'master',
    contractQuery:{contract:'master_data.customer',query:'find_by_id',
      inputs:[{targetAttribute:'customer.customer.customer_id',input:'id'}]}}
};
function report(s) {
  const subject={kind:s.kind,id:s.id};
  return {schemaVersion:'tei.cli.inspect/v0',ok:true,
    result:{schemaVersion:'tei.operations.inspect/v0',effect:'read-only',
      subject,canonical:structuredClone(fixtureCanonical[s.kind+':'+s.id]),
      worldFingerprint:'world:example',sourceIndexFingerprint:'index:example',
      sourceStatus:'indexed',sources:{subject,
        artifacts:[{path:'workspace/patterns.tei',fingerprint:'sha256:example'}],
        references:[{subject,artifactPath:'workspace/patterns.tei',role:'definition'}]
      }}};
}

test('runReferenceInspection batches five exact known TEI Subjects without arbitrary discovery',async()=>{
  const calls=[];
  const actual=await runReferenceInspection('/safe/tei','/safe/workspace',async(bin,workspace,subject)=>{
    assert.equal(bin,'/safe/tei');assert.equal(workspace,'/safe/workspace');
    calls.push(subject.kind+':'+subject.id);return report(subject);
  });
  assert.deepEqual(calls,Object.keys(fixtureCanonical));
  assert.equal(actual.observations.length,5);
  assert.equal(actual.acceptance,'not_asserted');
  assert.equal(actual.authority,'derived_observation_only');
  assert.equal(actual.links.applicationPattern.status,'declared_membership');
  assert.equal(actual.links.patternRelation.status,'declared_reference_match');
  assert.equal(actual.links.patternRelation.accepted,false);
});
test('drifting second read aborts with no partial batch output',async()=>{
  let calls=0;
  await assert.rejects(()=>runReferenceInspection('binary','workspace',async(_b,_w,s)=>{
    const r=report(s);if(++calls===2)r.result.worldFingerprint='world:changed';return r;
  }),e=>e instanceof InspectBridgeError&&e.code==='SNAPSHOT_DRIFT');
  assert.equal(calls,2);
});
test('CLI-subject failure in one step fails the entire run',async()=>{
  await assert.rejects(()=>runReferenceInspection('binary','workspace',async(_b,_w,s)=>{
    if(s.id==='sales.order.lines')return {schemaVersion:'tei.cli.inspect/v0',ok:false,error:{code:'SUBJECT_NOT_FOUND'}};
    return report(s);
  }),e=>e instanceof InspectBridgeError&&e.code==='INVALID_OR_FAILED_CLI_INSPECTION');
});
test('unknown Canonical fields degrade links instead of claiming accepted reuse',async()=>{
  const result=await runReferenceInspection('binary','workspace',async(_b,_w,s)=>{
    const r=report(s);if(s.kind==='application')r.result.canonical.unknownFlag='unverified';
    return r;
  });
  assert.equal(result.links.applicationPattern.status,'unknown');
  assert.equal(result.links.applicationPattern.accepted,false);
});
