import test from 'node:test';
import assert from 'node:assert/strict';
import {observeDirectTrace} from './direct-trace.mjs';
const K=(scope,kind,id)=>({scope,subject:{kind,id}});
const selected=K('reference-observation','semantic-relation','sales.order.customer');
const customer=K('reference-observation','semantic-data','customer.customer');
const parent=K('reference-observation','application-pattern','order_entry.customer_reference');
const fixture=()=>({schemaVersion:'tei.cli.trace/v0',ok:true,result:{
  schemaVersion:'tei.operations.trace/v0',snapshotFingerprint:'sha256:workspace',
  worldFingerprint:'sha256:world',sourceIndexFingerprint:'sha256:source',source:{status:'located'},
  trace:{schemaVersion:'tei.modelref.trace/v0',snapshotFingerprint:'sha256:graph',
    authority:'candidate-read-only',subject:selected,direction:'both',mode:'full',total:2,unresolved:0,
    rows:[{status:'resolved',edge:{from:selected,to:customer,relation:'targets'},resolvedTo:customer},
          {status:'resolved',edge:{from:parent,to:selected,relation:'references'},resolvedTo:selected}]
  }}});
test('two real-shaped direct references are candidates, never relation entities or absence',()=>{
  const x=observeDirectTrace(fixture());
  assert.equal(x.status,'direct-edge-candidates');
  assert.equal(x.edges.length,2);
  assert.deepEqual(x.selected,['reference-observation','semantic-relation','sales.order.customer']);
  assert.equal(x.sourceNavigation,'located-navigation-only');
  assert.equal(x.relationEntityIdentity,'not-provided');
  assert.equal(x.multiRoleClosure,'unproven');
  assert.equal(x.absentClaim,'unknown');
  assert.equal(x.authority,'read-only-candidate');
});
test('empty direct edge set does not become proof of relation absence',()=>{
  const f=fixture();f.result.trace.total=0;f.result.trace.rows=[];
  const x=observeDirectTrace(f);
  assert.equal(x.edges.length,0);assert.equal(x.absentClaim,'unknown');
});
test('summary, paged and stale/unqualified trace cannot be promoted',()=>{
  const s=fixture();s.result.trace.mode='summary';s.result.trace.rows=[];
  assert.equal(observeDirectTrace(s).status,'unknown');
  const p=fixture();p.result.trace.rows.pop();p.result.trace.nextOffset=1;
  assert.equal(observeDirectTrace(p).reason,'incomplete_direct_trace_page');
  const e=fixture();e.ok=false;e.error={code:'STALE_SNAPSHOT'};
  assert.equal(observeDirectTrace(e).reason,'cli_trace_unavailable');
});
test('unresolved target does not receive an invented qualified key',()=>{
  const f=fixture();f.result.trace.rows[0]={status:'unresolved',edge:{from:selected,to:{subject:customer.subject},relation:'targets'}};
  f.result.trace.unresolved=1;
  const x=observeDirectTrace(f);
  assert.equal(x.status,'direct-edge-candidates');
  assert.equal(x.edges[0].to,null);
  assert.equal(x.absentClaim,'unknown');
});
test('mismatched direction and row subject does not silently cross scope',()=>{
  const f=fixture();f.result.trace.rows[0].edge.from=K('other','semantic-relation','sales.order.customer');
  assert.equal(observeDirectTrace(f).reason,'selector_mismatch');
});
test('unvalidated schema, row count, evidence and authority are rejected',()=>{
  const cases=[
    x=>x.result.trace.authority='accepted',
    x=>x.result.trace.schemaVersion='different',
    x=>x.result.trace.total=3,
    x=>x.result.trace.unresolved=1,
    x=>x.result.trace.rows[0].edge.relation='',
    x=>x.result.sourceIndexFingerprint='',
  ];
  for(const mutate of cases){const f=fixture();mutate(f);assert.equal(observeDirectTrace(f).status,'unknown');}
});
test('different ModelRef relation labels never become identity/role declarations',()=>{
  const f=fixture();f.result.trace.rows[0].edge.relation='business-key';
  const r=observeDirectTrace(f);
  assert.equal(r.status,'direct-edge-candidates');
  assert.equal(r.edges[0].relationType,'business-key');
  assert.equal(r.relationEntityIdentity,'not-provided');
  assert.equal(r.multiRoleClosure,'unproven');
});
