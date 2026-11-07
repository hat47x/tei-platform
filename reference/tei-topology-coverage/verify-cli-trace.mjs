/** Independent read-only consumer of actual tei trace JSON for one fixed
 * repository test Workspace, not a product entrypoint or Source closure proof.
 */
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {observeDirectTrace} from './direct-trace.mjs';
if(process.argv.length !== 4) throw Error('usage: node verify-cli-trace.mjs <exact-cli-json> <alternate-scope-cli-json>');
const raw=JSON.parse(readFileSync(process.argv[2],'utf8'));
const observed=observeDirectTrace(raw);
assert.equal(observed.status,'direct-edge-candidates',JSON.stringify(observed));
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
console.log('actual-cli-direct-reference-observation=PASS; caller-scope-only=OBSERVED; source-closure=NOT-PROVEN; relation-identity=NOT-PROVEN');
