/** Independent read-only consumer of actual tei trace JSON for one fixed
 * repository test Workspace, not a product entrypoint or Source closure proof.
 */
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
<<<<<<< HEAD
import {observeDirectTrace} from './direct-trace.mjs';
if(process.argv.length !== 4) throw Error('usage: node verify-cli-trace.mjs <exact-cli-json> <alternate-scope-cli-json>');
const raw=JSON.parse(readFileSync(process.argv[2],'utf8'));
const observed=observeDirectTrace(raw);
assert.equal(observed.status,'direct-edge-candidates',JSON.stringify(observed));
=======
import {observeDirectTrace,compareObservedLoadContext} from './direct-trace.mjs';
if(process.argv.length !== 6) throw Error('usage: node verify-cli-trace.mjs <exact-cli-json> <alternate-scope-cli-json> <other-workspace-cli-json> <same-manifest-copy-cli-json>');
const raw=JSON.parse(readFileSync(process.argv[2],'utf8'));
const observed=observeDirectTrace(raw);
assert.equal(observed.status,'direct-edge-candidates',JSON.stringify(observed));
assert.equal(observed.graphFingerprint,raw.result.trace.graphFingerprint,
  'read-only graph identity must come from graphFingerprint, not the deprecated alias');
assert.equal(raw.result.trace.graphFingerprint,raw.result.trace.snapshotFingerprint,
  'v0 deprecated graph alias must match the current graph fingerprint');
assert.notEqual(raw.result.snapshotFingerprint,raw.result.trace.graphFingerprint,
  'Workspace-level continuation value must not be the Graph-only token');
const graphAliasAttack=structuredClone(raw);
graphAliasAttack.result.trace.snapshotFingerprint=raw.result.snapshotFingerprint;
assert.deepEqual(observeDirectTrace(graphAliasAttack),
  {status:'unknown',reason:'graph_fingerprint_alias_mismatch'},
  'a Workspace-level token substituted into the Graph alias must be rejected');
const edgeTargetAttack=structuredClone(raw);
const resolvedRow=edgeTargetAttack.result.trace.rows.find(x=>x.status==='resolved');
assert.ok(resolvedRow,'expected a resolved direct edge for negative control');
resolvedRow.edge.to.subject.id='forged.target';
assert.deepEqual(observeDirectTrace(edgeTargetAttack),
  {status:'unknown',reason:'resolved_edge_target_mismatch'},
  'a mutated declaration target must not be attached to the unchanged resolvedTo key');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
assert.deepEqual(observed.selected,['reference-observation','semantic-relation','sales.order.customer']);
const has=(fromKind,fromId,toKind,toId)=>observed.edges.some(e=>
  e.resolution==='resolved' &&
  e.from[0]==='reference-observation' && e.from[1]===fromKind && e.from[2]===fromId &&
  e.to?.[0]==='reference-observation' && e.to[1]===toKind && e.to[2]===toId);
assert.equal(has('application-pattern','order_entry.customer_reference','semantic-relation','sales.order.customer'),true,
  'actual CLI did not expose expected incoming direct edge');
assert.equal(has('semantic-relation','sales.order.customer','semantic-data','customer.customer'),true,
  'actual CLI did not expose expected outgoing direct edge');
assert.equal(observed.sourceNavigation,'located-navigation-only');
// Negative control on the actual CLI payload: SourceNavigation belongs to the
// selected subject, even when the Source is otherwise valid and located.
const crossBound=structuredClone(raw);
crossBound.result.source.key={scope:'reference-observation',subject:{
  kind:'application-pattern',id:'order_entry.customer_reference'
}};
assert.deepEqual(observeDirectTrace(crossBound),
  {status:'unknown',reason:'source_navigation_identity_mismatch'},
  'a located anchor for another subject must never be rebound');
<<<<<<< HEAD
=======
const wrongReference=structuredClone(raw);
wrongReference.result.source.references[0].subject={
  kind:'semantic-data',id:'customer.customer'
};
assert.deepEqual(observeDirectTrace(wrongReference),
  {status:'unknown',reason:'source_reference_identity_mismatch'},
  'Source Index reference for another subject must not be attributed to selector');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108

assert.equal(observed.absentClaim,'unknown');
assert.equal(observed.relationEntityIdentity,'not-provided');
assert.equal(observed.multiRoleClosure,'unproven');
assert.equal(observed.sourceAndPluginClosure,'unproven');
assert.equal(observed.authority,'read-only-candidate');
// The existing derivedbridge gives edges the scope supplied by the trace caller;
// it does not independently establish a Source-declared relation/subject scope.
const alternateRaw=JSON.parse(readFileSync(process.argv[3],'utf8'));
const alternate=observeDirectTrace(alternateRaw);
assert.equal(alternate.status,'direct-edge-candidates',JSON.stringify(alternate));
assert.deepEqual(alternate.selected,['alternate-observation','semantic-relation','sales.order.customer']);
assert.equal(alternate.scopeOrigin,'caller-supplied-projection-scope');
assert.equal(observed.scopeOrigin,'caller-supplied-projection-scope');
<<<<<<< HEAD
=======
assert.equal(observed.workspaceIdentity,'load-manifest-observed-only');
assert.deepEqual(raw.result.workspace, alternateRaw.result.workspace,
  'caller-scope-only observations must not silently switch Workspaces');
assert.equal(compareObservedLoadContext(raw,alternateRaw).status,
  'same-observed-load-context-only');
const otherWorkspaceRaw=JSON.parse(readFileSync(process.argv[4],'utf8'));
const otherWorkspace=observeDirectTrace(otherWorkspaceRaw);
assert.equal(otherWorkspace.status,'direct-edge-candidates',JSON.stringify(otherWorkspace));
assert.deepEqual(otherWorkspace.selected,observed.selected);
assert.equal(raw.result.worldFingerprint,otherWorkspaceRaw.result.worldFingerprint,
  'manifest-only Workspace change unexpectedly affected World projection');
assert.equal(raw.result.sourceIndexFingerprint,otherWorkspaceRaw.result.sourceIndexFingerprint,
  'manifest-only Workspace change unexpectedly affected Source Index projection');
assert.notDeepEqual(raw.result.workspace,otherWorkspaceRaw.result.workspace,
  'Workspace identity must reflect changed manifest bytes and name');
assert.deepEqual(compareObservedLoadContext(raw,otherWorkspaceRaw),
  {status:'unknown',reason:'workspace_manifest_mismatch'},
  'equal World/Source fingerprints must not join different Workspace manifests');
// Another real directory, byte-for-byte identical Manifest and Sources.
// The CLI does not carry a physical Root or Attempt identifier. Correctly
// report an *observed value match*, not a physically same Workspace.
const copiedRaw=JSON.parse(readFileSync(process.argv[5],'utf8'));
const copied=observeDirectTrace(copiedRaw);
assert.equal(copied.status,'direct-edge-candidates',JSON.stringify(copied));
assert.deepEqual(copied.selected,observed.selected);
assert.deepEqual(copiedRaw.result.workspace,raw.result.workspace);
assert.equal(copiedRaw.result.worldFingerprint,raw.result.worldFingerprint);
assert.equal(copiedRaw.result.sourceIndexFingerprint,raw.result.sourceIndexFingerprint);
assert.equal(copiedRaw.result.snapshotFingerprint,raw.result.snapshotFingerprint);
assert.deepEqual(compareObservedLoadContext(raw,copiedRaw),
  {status:'same-observed-load-context-only',authority:'read-only-candidate',
   physicalRootIdentity:'not-provided',readAttemptIdentity:'not-provided',
   sourceAndPluginClosure:'unproven',absentClaim:'unknown'},
  'same bytes should not be called the same physical root or read attempt');


>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
assert.equal(alternate.absentClaim,'unknown');
const withoutProjectionScope = x => x.edges.map(e=>({
  from:e.from.slice(1),to:e.to?.slice(1) ?? null,
  relationType:e.relationType,resolution:e.resolution,
})).sort((a,b)=>JSON.stringify(a).localeCompare(JSON.stringify(b)));
assert.deepEqual(withoutProjectionScope(observed),withoutProjectionScope(alternate),
  'the same declared direct dependencies differ after only changing caller scope');
assert.equal(raw.result.worldFingerprint,alternateRaw.result.worldFingerprint);
assert.equal(raw.result.sourceIndexFingerprint,alternateRaw.result.sourceIndexFingerprint);
assert.notEqual(observed.snapshotFingerprint,alternate.snapshotFingerprint,
  'selector-bound snapshot should change when the caller scope changes');
<<<<<<< HEAD
console.log('actual-cli-direct-reference-observation=PASS; caller-scope-only=OBSERVED; source-closure=NOT-PROVEN; relation-identity=NOT-PROVEN');
=======
console.log('actual-cli-direct-reference-observation=PASS; caller-scope-only=OBSERVED; different-manifest-same-world=REJECTED; snapshot-token-equal='+ (raw.result.snapshotFingerprint===otherWorkspaceRaw.result.snapshotFingerprint) +'; byte-identical-copy-root=NOT-IDENTIFIED; source-closure=NOT-PROVEN; relation-identity=NOT-PROVEN');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
