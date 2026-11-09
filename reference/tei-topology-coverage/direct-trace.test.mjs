import test from 'node:test';
import assert from 'node:assert/strict';
import {observeDirectTrace} from './direct-trace.mjs';
const K=(scope,kind,id)=>({scope,subject:{kind,id}});
const selected=K('reference-observation','semantic-relation','sales.order.customer');
const customer=K('reference-observation','semantic-data','customer.customer');
const parent=K('reference-observation','application-pattern','order_entry.customer_reference');
const fixture=()=>({schemaVersion:'tei.cli.trace/v0',ok:true,result:{
  schemaVersion:'tei.operations.trace/v0',snapshotFingerprint:'sha256:workspace',
  worldFingerprint:'sha256:world',sourceIndexFingerprint:'sha256:source',
  source:{key:selected,status:'located',references:[{subject:selected.subject,artifactPath:'semantic/sales/order.tei',role:'definition'}]},
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
  assert.equal(x.scopeOrigin,'caller-supplied-projection-scope');
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

test('located SourceNavigation must belong to exactly the trace subject',()=>{
  const good=observeDirectTrace(fixture());
  assert.equal(good.sourceNavigation,'located-navigation-only');
  const wrong=fixture();wrong.result.source.key=parent;
  assert.deepEqual(observeDirectTrace(wrong),
    {status:'unknown',reason:'source_navigation_identity_mismatch'});
  const differentScope=fixture();
  differentScope.result.source.key=K('other','semantic-relation','sales.order.customer');
  assert.equal(observeDirectTrace(differentScope).reason,'source_navigation_identity_mismatch');
});

test('missing SourceNavigation identity and located-without-references fail closed',()=>{
  for(const alter of [
    x=>delete x.result.source.key,
    x=>delete x.result.source,
    x=>x.result.source.status='located-invalid',
    x=>x.result.source.references=[],
    x=>x.result.source.references=undefined,
  ]){
    const q=fixture();alter(q);
    assert.deepEqual(observeDirectTrace(q),
      {status:'unknown',reason:'source_navigation_unqualified'});
  }
});

test('unresolved Source navigation remains a candidate, never a located origin',()=>{
  const f=fixture();f.result.source.status='unresolved';f.result.source.reason='ambiguous-source-anchor';
  delete f.result.source.references;
  const x=observeDirectTrace(f);
  assert.equal(x.status,'direct-edge-candidates');
  assert.equal(x.sourceNavigation,'unresolved');
  assert.equal(x.absentClaim,'unknown');
  assert.equal(x.scopeOrigin,'caller-supplied-projection-scope');
});

test('malformed projectionUnresolved cannot be silently read as complete',()=>{
  for(const value of [null, false, 3, [''], [null], {}]){
    const f=fixture();f.result.projectionUnresolved=value;
    assert.deepEqual(observeDirectTrace(f),
      {status:'unknown',reason:'invalid_projection_unresolved'});
  }
  const f=fixture();f.result.projectionUnresolved=['unmaterialized-to:semantic-data:unknown'];
  const o=observeDirectTrace(f);
  assert.equal(o.status,'direct-edge-candidates');
  assert.equal(o.hasUnresolvedProjection,true);
  assert.equal(o.absentClaim,'unknown');
});
