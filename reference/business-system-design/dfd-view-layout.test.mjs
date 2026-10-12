import test from 'node:test';
import assert from 'node:assert/strict';
import {previewLayoutMove, LAYOUT_SCHEMA, EDIT_SCHEMA, PREVIEW_SCHEMA} from './dfd-view-layout.mjs';

const hash = 'sha256:' + 'a'.repeat(64);
const layoutHash = 'sha256:' + 'b'.repeat(64);
const procedureId = 'order-entry.approval-and-shipment-prep';
const dfd = {
  schemaVersion: 'business-dfd-projection-v0', authority: 'derived-read-only',
  canonicalProcedureUsed: false, procedureId,
  steps: [
    {id: 'APP-10', functionRefs: []},
    {id: 'APP-40', functionRefs: [{id:'order_entry.submit_order',status:'resolved'}]}
  ],
  triggers: [{id:'TR-ORDER'}], diagnostics: [],
  information: {roles:[{id:'approver'}], externalParties:[{id:'customer'}], stores:[{id:'orders'}]}
};
const layout = {
  schemaVersion: LAYOUT_SCHEMA, authority: 'view-only', procedureId,
  projectionSHA256: hash, nodes:[{kind:'step',id:'APP-10',x:1,y:2}]
};
const edit = (overrides={}) => ({
  schema: EDIT_SCHEMA, id:'layout-review-1', origin:'studio', intent:'Improve reading order',
  procedureId, projectionSHA256: hash, baseLayoutSHA256:layoutHash,
  operation:{kind:'move-node',nodeKind:'step',nodeId:'APP-10',x:5,y:7},
  ...overrides
});
const run = (p=dfd,l=layout,e=edit(),d=hash,h=layoutHash) => previewLayoutMove(p,d,l,h,e);

test('moves a manual step as view data with no semantic or source change', () => {
  const old = JSON.stringify({dfd,layout});
  const result = run();
  assert.equal(result.schema,PREVIEW_SCHEMA);
  assert.equal(result.canonicalMeaningChanged,false);
  assert.equal(result.readyForAcceptance,false);
  assert.deepEqual(result.sourceWrites,[]);
  assert.equal(result.proposedLayout.nodes[0].x,5);
  assert.equal(JSON.stringify({dfd,layout}),old);
});
test('may position a party and Function only when they appear in the explicit projection', () => {
  for(const [nodeKind,nodeId] of [['party','customer'],['function','order_entry.submit_order']]){
    const result=run(dfd,layout,edit({operation:{kind:'move-node',nodeKind,nodeId,x:12,y:-20}}));
    assert.ok(result.proposedLayout.nodes.some(n=>n.kind===nodeKind&&n.id===nodeId));
  }
});
test('rejects stale layout, stale projection, and false Canonical authority', () => {
  assert.throws(()=>run(dfd,{...layout,projectionSHA256:layoutHash}),/STALE_OR_UNSUPPORTED_VIEW_LAYOUT/);
  assert.throws(()=>run(dfd,layout,edit({baseLayoutSHA256:hash})),/STALE_OR_UNSUPPORTED_LAYOUT_EDIT/);
  assert.throws(()=>run({...dfd,canonicalProcedureUsed:true}),/UNQUALIFIED_DFD_PROJECTION/);
  assert.throws(()=>run({...dfd,diagnostics:[{severity:'error'}]}),/UNQUALIFIED_DFD_PROJECTION/);
});
test('rejects arbitrary patches, unknown nodes, and attempted Authority promotion', () => {
  assert.throws(()=>run(dfd,layout,edit({operation:{kind:'patch-source',path:'/steps/0'}})),/INVALID_FIELDS/);
  assert.throws(()=>run(dfd,layout,edit({operation:{kind:'move-node',nodeKind:'step',nodeId:'missing',x:1,y:2}})),/UNKNOWN_DFD_NODE/);
  assert.throws(()=>run(dfd,{...layout,authority:'accepted'}),/STALE_OR_UNSUPPORTED_VIEW_LAYOUT/);
  assert.throws(()=>run(dfd,layout,{...edit(),target:'source'}),/INVALID_FIELDS/);
});
test('rejects invalid coordinates, unchanged nodes, duplicate or dangling layout positions', () => {
  assert.throws(()=>run(dfd,layout,edit({operation:{kind:'move-node',nodeKind:'step',nodeId:'APP-10',x:Infinity,y:2}})),/INVALID_COORDINATE/);
  assert.throws(()=>run(dfd,layout,edit({operation:{kind:'move-node',nodeKind:'step',nodeId:'APP-10',x:1,y:2}})),/NO_EFFECT_MOVE/);
  assert.throws(()=>run(dfd,{...layout,nodes:[...layout.nodes,...layout.nodes]}),/DUPLICATE_LAYOUT_NODE/);
  assert.throws(()=>run(dfd,{...layout,nodes:[{kind:'step',id:'phantom',x:0,y:0}]}),/UNKNOWN_LAYOUT_NODE/);
});
