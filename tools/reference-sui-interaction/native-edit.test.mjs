import test from 'node:test';
import assert from 'node:assert/strict';
import {BridgeError} from './bridge.mjs';
import {beginNativeEdit, previewNativeEdit, finishNativeEdit, rollbackNativeEdit, persistNativeEdit} from './native-edit.mjs';

// Fixture-only SUI behavior reconstructed from current CardView, CanvasShell,
// App.handleCardMove/commitCardDragSnapshot, geometry/bounds and moveCardToIsland.
// It is not SUI's authoritative runtime implementation.
const doc = () => ({version:1,id:'d1',title:'inquiry',cards:[
  {id:'c1',x:0,y:0,text:'hypothesis',holdState:'held',meta:{source:'ref:local'}},
  {id:'c2',x:410,y:80,text:'counterpoint'}],
  islands:[{id:'old',cardIds:['c1']},{id:'new',cardIds:['c2'],geometry:{type:'polygon',
    points:[{x:330,y:40},{x:900,y:40},{x:900,y:320},{x:330,y:320}]}}],
  edges:[{id:'e1',fromId:'c1',toId:'c2',type:'future.opaque'}],
  affiliations:[{id:'a1',cardId:'c1',islandId:'new'}],
  reviewAttribution:{reviewState:'unreviewed'},unknown:{keep:true}});
const fixtureAdapter = {
  preview(document,{targetId,event}) {
    if (!event || !Number.isFinite(event.zoom) || event.zoom<=0 ||
      !Number.isFinite(event.dx) || !Number.isFinite(event.dy)) throw new BridgeError('invalid_native_event');
    const card=document.cards.find(c=>c.id===targetId);
    if(!card) throw new BridgeError('target_not_found');
    if(event.dx===0 && event.dy===0) return {document,changed:false};
    const snap=v=>event.grid?Math.round(v/10)*10:v;
    card.x=snap(card.x+event.dx/event.zoom);
    card.y=snap(card.y+event.dy/event.zoom);
    return {document,changed:true};
  },
  finish(document,{targetId}) {
    const card=document.cards.find(c=>c.id===targetId);
    // This fixture uses a polygon *bounding box*, matching SUI's getIslandWorldBounds.
    const center={x:card.x+110,y:card.y+40};
    const target=document.islands.find(island=>{
      if(island.cardIds.includes(targetId)||!island.geometry?.points)return false;
      const p=island.geometry.points;
      return center.x>=Math.min(...p.map(q=>q.x))&&center.x<=Math.max(...p.map(q=>q.x))&&
        center.y>=Math.min(...p.map(q=>q.y))&&center.y<=Math.max(...p.map(q=>q.y));
    });
    if(target) {
      document.islands=document.islands.map(island=>island.id===target.id?
        {...island,cardIds:[...island.cardIds,targetId]}:
        {...island,cardIds:island.cardIds.filter(id=>id!==targetId)});
    }
    return {document};
  },
};
const begin=(adapter=fixtureAdapter)=>beginNativeEdit({applicationId:'sui',documentId:'d1',expectedRevision:'r1',
  targetId:'c1',document:doc(),adapter});
const dragToIsland=()=>finishNativeEdit(previewNativeEdit(begin(),{dx:620,dy:120,zoom:2,grid:true}),'pointerup');
const errorCode=async(p,code)=>assert.rejects(p,e=>e instanceof BridgeError&&e.code===code);

test('zoom, snap, native island membership and one undo step are one in-memory transaction',()=>{
  const edit=dragToIsland();
  assert.equal(edit.draft.cards[0].x,310);assert.equal(edit.draft.cards[0].y,60);
  assert.deepEqual(edit.draft.islands.map(i=>i.cardIds),[[],['c2','c1']]);
  assert.deepEqual(edit.historyStep.before.islands.map(i=>i.cardIds),[['c1'],['c2']]);
  assert.equal(edit.historyStep.before.cards[0].x,0);
  assert.equal(edit.historyStep.after.cards[0].x,310);
  assert.equal(edit.draft.cards[0].holdState,'held');
  assert.equal(edit.draft.cards[0].meta.source,'ref:local');
  assert.equal(edit.draft.edges[0].type,'future.opaque');
  assert.deepEqual(edit.draft.affiliations,[{id:'a1',cardId:'c1',islandId:'new'}]);
  assert.deepEqual(edit.draft.unknown,{keep:true});
});
test('multiple pointermove events do not create multiple undo steps',()=>{
  let e=begin();e=previewNativeEdit(e,{dx:320,dy:50,zoom:2,grid:false});
  e=previewNativeEdit(e,{dx:300,dy:70,zoom:2,grid:false});
  const finished=finishNativeEdit(e,'pointerup');
  assert.deepEqual([finished.draft.cards[0].x,finished.draft.cards[0].y],[310,60]);
  assert.equal(finished.historyStep.before.cards[0].x,0);
});
test('pointercancel currently finalizes SUI history, not a generic rollback',()=>{
  const e=finishNativeEdit(previewNativeEdit(begin(),{dx:620,dy:120,zoom:2,grid:false}),'pointercancel');
  assert.equal(e.phase,'finished');assert.equal(e.reason,'pointercancel');
  assert.ok(e.historyStep);assert.deepEqual(e.draft.islands.map(i=>i.cardIds),[[],['c2','c1']]);
});
test('explicit rollback restores original and rejects persist',async()=>{
  const e=rollbackNativeEdit(previewNativeEdit(begin(),{dx:100,dy:100,zoom:1}));
  assert.deepEqual(e.draft,e.original);
  await errorCode(persistNativeEdit(e,{}),'edit_not_finished');
});
test('zero-delta pointer interaction produces no dirty history and no storage request',async()=>{
  const e=finishNativeEdit(previewNativeEdit(begin(),{dx:0,dy:0,zoom:1}),'pointerup');
  assert.equal(e.changed,false);assert.equal(e.historyStep,null);
  assert.deepEqual(await persistNativeEdit(e),{saved:false,reason:'unchanged'});
});
test('finishing local edit does not persist until explicit Save',async()=>{
  const e=dragToIsland();let writes=0;let version='r1';let value=doc();
  assert.equal(writes,0);
  const ports={applicationId:'sui',authorize:async()=>true,validate:async()=>true,
    save:async({expectedRevision,document})=>{if(expectedRevision!==version)return {ok:false};
      value=document;version='r2';writes++;return {ok:true,revision:version};}};
  const result=await persistNativeEdit(e,ports);
  assert.deepEqual(result,{saved:true,revision:'r2'});
  assert.equal(writes,1);assert.equal(value.cards[0].x,310);
  assert.deepEqual(value.islands.map(i=>i.cardIds),[[],['c2','c1']]);
});
test('stale storage revision cannot overwrite even after local edit finished',async()=>{
  const e=dragToIsland();let writes=0;
  const ports={applicationId:'sui',authorize:async()=>true,validate:async()=>true,
    save:async({expectedRevision})=>{assert.equal(expectedRevision,'r1');return {ok:false};}};
  await errorCode(persistNativeEdit(e,ports),'conditional_save_failed');assert.equal(writes,0);
});
test('permission and full-domain validation fail closed before durable write',async()=>{
  const e=dragToIsland();let writes=0;
  const ports={applicationId:'sui',authorize:async()=>false,validate:async()=>true,
    save:async()=>{writes++;return {ok:true,revision:'r2'};}};
  await errorCode(persistNativeEdit(e,ports),'unauthorized');assert.equal(writes,0);
  ports.authorize=async()=>true;ports.validate=async()=>false;
  await errorCode(persistNativeEdit(e,ports),'validation_failed');assert.equal(writes,0);
});
test('cross application save binding rejected without calling privileged callbacks',async()=>{
  const e=dragToIsland();let called=false;
  await errorCode(persistNativeEdit(e,{applicationId:'other',authorize:async()=>{called=true;return true;},
    validate:async()=>true,save:async()=>({ok:true,revision:'r2'})}),'invalid_persist_ports');
  assert.equal(called,false);
});
test('bad native event and unknown target are rejected',()=>{
  assert.throws(()=>previewNativeEdit(begin(),{dx:5,dy:0,zoom:0}),/invalid_native_event/);
  const e=beginNativeEdit({applicationId:'sui',documentId:'d1',expectedRevision:'r1',targetId:'missing',
    document:doc(),adapter:fixtureAdapter});
  assert.throws(()=>previewNativeEdit(e,{dx:5,dy:0,zoom:1}),/target_not_found/);
});
test('generic transaction can use another native domain adapter',async()=>{
  const native={preview:(d,{event})=>({document:{...d,value:d.value+event.delta},changed:true}),
    finish:(d)=>({document:{...d,done:true}})};
  let e=beginNativeEdit({applicationId:'sketch',documentId:'s1',expectedRevision:'r1',targetId:'n1',
    document:{value:0,details:'untouched'},adapter:native});
  e=finishNativeEdit(previewNativeEdit(e,{delta:7}),'pointerup');
  assert.deepEqual(e.draft,{value:7,details:'untouched',done:true});
  assert.equal(e.historyStep.before.value,0);
});
