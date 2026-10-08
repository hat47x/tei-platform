import test from 'node:test';
import assert from 'node:assert/strict';
import {BridgeError, beginGesture, previewGesture, cancelGesture, finishGesture, commitIntent, moveSuiCard} from './bridge.mjs';

const initial = {applicationId: 'sui', documentId: 'doc1', targetId: 'card1', targetKind: 'card', expectedRevision: 'r1', initial: {x: 1, y: 2}};
const document = () => ({version: 1, id:'doc1', title:'work', cards:[{id:'card1',text:'unreviewed',x:1,y:2,holdState:'held',meta:{source:'source://keep'}}],
  edges:[{id:'edge1',fromId:'card1',toId:'card1',type:'unknown.future.edge'}],
  islands:[], affiliations:[{id:'aff1',cardId:'card1',islandId:'some-island'}],
  reviewAttribution:{reviewState:'unreviewed'}, proprietary:{keep:'opaque'}});
function storeFactory({app = 'sui', doc = document(), rev='r1', allow = true, validator = true}={}) {
  let stored = structuredClone(doc); let revision = rev; let writes = 0; let loads = 0;
  const ports = {applicationId:app, authorize: async () => allow,
    load: async docId => { loads++; return {applicationId:app, documentId:docId, revision, document:stored}; },
    apply: moveSuiCard, validate: async () => validator,
    save: async request => {
      if (request.expectedRevision !== revision) return {ok:false};
      stored = request.document; revision='r'+(Number(revision.slice(1))+1); writes++;
      return {ok:true,revision};
    }};
  return {ports, get: () => structuredClone(stored), writes:()=>writes, loads:()=>loads, updateRevision:()=>{revision='r99';}};
}
function intent(x=8,y=9) {return finishGesture(previewGesture(beginGesture(initial), {x,y}));}
async function expectCode(promise, code) {await assert.rejects(promise, e => e instanceof BridgeError && e.code === code);}

test('preview and cancellation never mutate native SUI Document or storage', async()=>{
  const s=storeFactory(), before=s.get();
  const started=beginGesture(initial), preview=previewGesture(started,{x:10,y:20});
  assert.deepEqual(started.current,{x:1,y:2});
  assert.deepEqual(preview.current,{x:10,y:20});
  assert.equal(cancelGesture(preview).phase,'cancelled');
  assert.throws(()=>finishGesture(cancelGesture(preview)), /gesture_not_active/);
  assert.deepEqual(s.get(),before); assert.equal(s.writes(),0); assert.equal(s.loads(),0);
});
test('authorized commit updates only coordinates, preserving extensions and hold', async()=>{
  const s=storeFactory(), before=s.get();
  const receipt=await commitIntent(intent(),s.ports);
  assert.equal(receipt.revision,'r2'); assert.equal(s.writes(),1);
  const actual=s.get();
  assert.deepEqual(actual.cards[0],{...before.cards[0],x:8,y:9});
  actual.cards=before.cards; assert.deepEqual(actual,before);
});
test('cross application binding fails before auth/load/save', async()=>{
  const s=storeFactory({app:'other'});
  await expectCode(commitIntent(intent(),s.ports),'application_mismatch');
  assert.equal(s.loads(),0); assert.equal(s.writes(),0);
});
test('unauthorized fails before loading document', async()=>{
  const s=storeFactory({allow:false});
  await expectCode(commitIntent(intent(),s.ports),'unauthorized');
  assert.equal(s.loads(),0); assert.equal(s.writes(),0);
});
test('stale revision rejects and never overwrites', async()=>{
  const s=storeFactory({rev:'r2'});
  await expectCode(commitIntent(intent(),s.ports),'stale_revision');
  assert.equal(s.writes(),0);
});
test('concurrent update is rejected by storage CAS, not only initial preflight',async()=>{
  const s=storeFactory(); const port={...s.ports,validate: async()=>{s.updateRevision();return true;}};
  await expectCode(commitIntent(intent(),port),'conditional_save_failed');
  assert.equal(s.writes(),0);
});
test('domain validation failure leaves original Document unchanged', async()=>{
  const s=storeFactory({validator:false}), before=s.get();
  await expectCode(commitIntent(intent(),s.ports),'validation_failed');
  assert.deepEqual(s.get(),before); assert.equal(s.writes(),0);
});
test('missing/duplicate targets rejected without any write',async()=>{
  for(const cards of [[],[...document().cards,...document().cards]]){
    const s=storeFactory({doc:{...document(),cards}});
    await expectCode(commitIntent(intent(),s.ports),'target_not_unique');
    assert.equal(s.writes(),0);
  }
});
test('invalid numeric positions rejected before persistence',async()=>{
  const s=storeFactory();
  assert.throws(()=>previewGesture(beginGesture(initial),{x:Infinity,y:1}),/invalid_position/);
  await expectCode(commitIntent({...intent(),position:{x:NaN,y:1}},s.ports),'invalid_position');
  assert.equal(s.writes(),0);
});
test('missing security or validation port fails closed',async()=>{
  const s=storeFactory(); delete s.ports.validate;
  await expectCode(commitIntent(intent(),s.ports),'missing_port');
  assert.equal(s.writes(),0);
});
test('another domain uses same generic command boundary with its own adapter',async()=>{
  let saved={nodes:[{id:'n1',x:1,y:2}], other:{keep:true}};
  const otherIntent=finishGesture(previewGesture(beginGesture({...initial,applicationId:'diagram', targetKind:'node',targetId:'n1'}),{x:8,y:9}));
  const other={applicationId:'diagram',authorize:async()=>true,
    load:async()=>({applicationId:'diagram',documentId:'doc1',revision:'r1',document:saved}),
    apply:async (doc, i)=>{doc.nodes[0].x=i.position.x;doc.nodes[0].y=i.position.y;return doc;},
    validate:async()=>true,
    save:async o=>{saved=o.document;return {ok:true,revision:'r2'};}};
  await commitIntent(otherIntent,other);
  assert.deepEqual(saved,{nodes:[{id:'n1',x:8,y:9}],other:{keep:true}});
});
