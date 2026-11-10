import test from 'node:test';
import assert from 'node:assert/strict';
<<<<<<< HEAD
import {observeDirectTrace} from './direct-trace.mjs';
=======
import {observeDirectTrace, compareObservedLoadContext} from './direct-trace.mjs';
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
const K=(scope,kind,id)=>({scope,subject:{kind,id}});
const selected=K('reference-observation','semantic-relation','sales.order.customer');
const customer=K('reference-observation','semantic-data','customer.customer');
const parent=K('reference-observation','application-pattern','order_entry.customer_reference');
const fixture=()=>({schemaVersion:'tei.cli.trace/v0',ok:true,result:{
  schemaVersion:'tei.operations.trace/v0',snapshotFingerprint:'sha256:workspace',
  worldFingerprint:'sha256:world',sourceIndexFingerprint:'sha256:source',
<<<<<<< HEAD
  source:{key:selected,status:'located',references:[{subject:selected.subject,artifactPath:'semantic/sales/order.tei',role:'definition'}]},
  trace:{schemaVersion:'tei.modelref.trace/v0',snapshotFingerprint:'sha256:graph',
=======
  workspace:{name:'reference',formatVersion:1,manifest:{path:'tei.workspace.yaml',sha256:'sha256:manifest-bytes'}},
  source:{key:selected,status:'located',references:[{subject:selected.subject,artifactPath:'semantic/sales/order.tei',role:'definition'}]},
  trace:{schemaVersion:'tei.modelref.trace/v0',graphFingerprint:'sha256:graph',snapshotFingerprint:'sha256:graph',ruleFingerprint:'sha256:rules',
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
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
<<<<<<< HEAD
=======
  assert.equal(x.workspaceIdentity,'load-manifest-observed-only');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
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
<<<<<<< HEAD
=======

test('Workspace manifest identity must exist even if World/Source fingerprints match',()=>{
  for(const alter of [
    x=>delete x.result.workspace,
    x=>delete x.result.workspace.manifest,
    x=>x.result.workspace.manifest.sha256='',
    x=>x.result.workspace.formatVersion=0,
    x=>x.result.workspace.name=' reference ',
  ]){
    const f=fixture();alter(f);
    assert.deepEqual(observeDirectTrace(f),
      {status:'unknown',reason:'workspace_identity_unqualified'});
  }
});
test('Load-manifest identity is observed, not an Accepted identity or grant',()=>{
  const f=fixture();f.result.workspace.manifest.sha256='sha256:another-manifest';
  const x=observeDirectTrace(f);
  assert.equal(x.status,'direct-edge-candidates');
  assert.equal(x.workspaceIdentity,'load-manifest-observed-only');
  assert.equal(x.authority,'read-only-candidate');
  assert.equal(x.absentClaim,'unknown');
});

test('Source Index reference subject must match trace Kind/ID independently of scope',()=>{
  const f=fixture();
  f.result.source.references[0].subject={kind:'semantic-data',id:'customer.customer'};
  assert.deepEqual(observeDirectTrace(f),
    {status:'unknown',reason:'source_reference_identity_mismatch'});
  const bad=fixture();
  bad.result.source.references[0].artifactPath='';
  assert.deepEqual(observeDirectTrace(bad),
    {status:'unknown',reason:'source_reference_unqualified'});
});

test('same World and Source Index bytes do not join different Workspace manifests',()=>{
  const first=fixture(),other=fixture();
  other.result.workspace.name='other';
  other.result.workspace.manifest.sha256='sha256:other-manifest';
  // Deliberately leave the World, Source Index, and trace fingerprint identical.
  assert.equal(first.result.worldFingerprint,other.result.worldFingerprint);
  assert.equal(first.result.sourceIndexFingerprint,other.result.sourceIndexFingerprint);
  assert.equal(first.result.snapshotFingerprint,other.result.snapshotFingerprint);
  assert.deepEqual(compareObservedLoadContext(first,other),
    {status:'unknown',reason:'workspace_manifest_mismatch'});
  assert.notDeepEqual(observeDirectTrace(first).workspaceLoad,
    observeDirectTrace(other).workspaceLoad);
});

test('two caller-scope views of the same Workspace may only share bounded load context',()=>{
  const first=fixture(),other=fixture();
  const another=K('alternate-observation','semantic-relation','sales.order.customer');
  const endpoint=K('alternate-observation','semantic-data','customer.customer');
  const parentAlt=K('alternate-observation','application-pattern','order_entry.customer_reference');
  other.result.trace.subject=another;
  other.result.source.key=another;
  other.result.trace.rows[0].edge.from=another;
  other.result.trace.rows[0].edge.to=endpoint;
  other.result.trace.rows[0].resolvedTo=endpoint;
  other.result.trace.rows[1].edge.from=parentAlt;
  other.result.trace.rows[1].edge.to=another;
  other.result.trace.rows[1].resolvedTo=another;
  assert.deepEqual(compareObservedLoadContext(first,other),
    {status:'same-observed-load-context-only',authority:'read-only-candidate',
      physicalRootIdentity:'not-provided',readAttemptIdentity:'not-provided',
      sourceAndPluginClosure:'unproven',absentClaim:'unknown'});
  other.result.sourceIndexFingerprint='sha256:different-source';
  assert.deepEqual(compareObservedLoadContext(first,other),
    {status:'unknown',reason:'read_snapshot_mismatch'});
});

test('manifest identity alone cannot rescue malformed or undisclosed trace qualification',()=>{
  const first=fixture(),other=fixture();other.result.trace.mode='summary';
  assert.deepEqual(compareObservedLoadContext(first,other),
    {status:'unknown',reason:'trace_unqualified'});
  const a=observeDirectTrace(first);
  assert.equal(a.workspaceIdentity,'load-manifest-observed-only');
  assert.equal(a.authority,'read-only-candidate');
});

test('byte-identical copy is observationally equivalent but physical root/attempt remain unknown',()=>{
  // This is a synthetic example of two different physical roots returning
  // indistinguishable CLI packets: root location is NOT in the trace envelope.
  const original=fixture(),identicalCopy=structuredClone(original);
  assert.deepEqual(compareObservedLoadContext(original,identicalCopy),
    {status:'same-observed-load-context-only',authority:'read-only-candidate',
      physicalRootIdentity:'not-provided',readAttemptIdentity:'not-provided',
      sourceAndPluginClosure:'unproven',absentClaim:'unknown'});
  assert.equal(observeDirectTrace(original).workspaceIdentity,'load-manifest-observed-only');
});

test('Graph token is distinct from CLI Workspace continuation fingerprint (#3625)',()=>{
  const f=fixture();
  const x=observeDirectTrace(f);
  assert.equal(x.status,'direct-edge-candidates');
  assert.equal(x.graphFingerprint,'sha256:graph');
  assert.equal(x.snapshotFingerprint,'sha256:workspace');
  assert.notEqual(x.graphFingerprint,x.snapshotFingerprint);
  // GraphFingerprint is canonical; the inner snapshotFingerprint is a
  // deprecated alias and may be omitted during its removal cycle.
  delete f.result.trace.snapshotFingerprint;
  assert.equal(observeDirectTrace(f).graphFingerprint,'sha256:graph');
  const y=fixture();delete y.result.trace.graphFingerprint;
  assert.deepEqual(observeDirectTrace(y),
    {status:'unknown',reason:'unqualified_trace_shape'});
});

test('mismatched deprecated graph token alias is rejected, never treated as continuation',()=>{
  const f=fixture();f.result.trace.snapshotFingerprint='sha256:stale-graph-alias';
  assert.deepEqual(observeDirectTrace(f),
    {status:'unknown',reason:'graph_fingerprint_alias_mismatch'});
  const g=fixture();g.result.trace.ruleFingerprint='';
  assert.equal(observeDirectTrace(g).reason,'unqualified_trace_shape');
  const h=fixture();h.result.trace.continuation={
    fingerprint:'sha256:wrong-graph-token',field:'result.snapshotFingerprint'
  };
  assert.equal(observeDirectTrace(h).reason,'incomplete_direct_trace_page');
});

test('resolved edge target must match declared Ref target and optional scope',()=>{
  for(const alter of [
    f=>f.result.trace.rows[0].edge.to.subject.kind='application-pattern',
    f=>f.result.trace.rows[0].edge.to.subject.id='someone.else',
    f=>f.result.trace.rows[0].edge.to.scope='different-scope',
  ]){
    const f=fixture();
    // The fixture reuses the same Subject object in Edge.To and ResolvedTo.
    // Clone the declaration before mutating it, or the forged input would
    // change *both* sides and accidentally remain consistent.
    f.result.trace.rows[0].edge.to=structuredClone(f.result.trace.rows[0].edge.to);
    alter(f);
    assert.deepEqual(observeDirectTrace(f),
      {status:'unknown',reason:'resolved_edge_target_mismatch'});
  }
  for(const alter of [
    f=>delete f.result.trace.rows[0].edge.to,
    f=>f.result.trace.rows[0].edge.to={subject:{kind:'',id:'customer.customer'}},
  ]){
    const f=fixture();alter(f);
    assert.equal(observeDirectTrace(f).reason,'invalid_direct_target');
  }
  // Unqualified-to Scope is allowed; Graph's resolved key retains its scope.
  const g=fixture();
  // Keep the resolved scope intact while removing only the Ref's optional scope.
  g.result.trace.rows[0].edge.to=structuredClone(g.result.trace.rows[0].edge.to);
  delete g.result.trace.rows[0].edge.to.scope;
  assert.equal(observeDirectTrace(g).status,'direct-edge-candidates');
});
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
