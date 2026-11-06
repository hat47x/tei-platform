/** Independent read-only consumer of actual tei trace JSON for one fixed
 * repository test Workspace, not a product entrypoint or Source closure proof.
 */
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {observeDirectTrace} from './direct-trace.mjs';
if(process.argv.length !== 3) throw Error('usage: node verify-cli-trace.mjs <exact-cli-json>');
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
assert.equal(observed.absentClaim,'unknown');
assert.equal(observed.relationEntityIdentity,'not-provided');
assert.equal(observed.multiRoleClosure,'unproven');
assert.equal(observed.sourceAndPluginClosure,'unproven');
assert.equal(observed.authority,'read-only-candidate');
console.log('actual-cli-direct-reference-observation=PASS; source-closure=NOT-PROVEN; relation-identity=NOT-PROVEN');
