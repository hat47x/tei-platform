import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {beginGesture, previewGesture, finishGesture, commitIntent, BridgeError} from './bridge.mjs';
import {createSuiHttpPorts} from './sui-http-ports.mjs';

const document = () => ({version:1, id:'work-1', title:'example',
  createdAt:'2026-10-08T00:00:00Z', updatedAt:'2026-10-08T00:00:00Z',
  transform:{panX:0,panY:0,zoom:1},
  cards:[{id:'c1',text:'source note',x:10,y:12,holdState:'held',meta:{source:'opaque-source'}}],
  edges:[{id:'e1',fromId:'c1',toId:'c1',type:'unknown.edge.kind'}],islands:[],
  affiliations:[],reviewAttribution:{reviewState:'unreviewed'},opaqueFuture:{n:42}});
const gesture = (rev='r1') => finishGesture(previewGesture(beginGesture({applicationId:'sui',documentId:'work-1',targetKind:'card',targetId:'c1',expectedRevision:rev,initial:{x:10,y:12}}),{x:21,y:34}));
async function harness(run, {missingEtag=false, weakEtag=false, readDenied=false, conflict=false}={}) {
  let doc=document(), revision='r1'; const seen=[];
  const server=createServer(async (request,response)=>{
    seen.push({method:request.method, path:request.url, headers:request.headers});
    if(request.url!=='/api/docs/work-1'){response.writeHead(404);response.end();return;}
    if(readDenied && request.method==='GET'){response.writeHead(403);response.end();return;}
    if(request.method==='GET'){
      const etag=missingEtag?{}:{ETag:weakEtag?`W/"${revision}"`:`"${revision}"`};
      response.writeHead(200,{'Content-Type':'application/json',...etag}); response.end(JSON.stringify(doc));return;
    }
    if(request.method==='PUT'){
      const chunks=[];for await(const c of request) chunks.push(c);
      const body=JSON.parse(Buffer.concat(chunks).toString('utf8'));
      seen.at(-1).body=body;
      if(conflict||request.headers['if-match']!==`"${revision}"`){response.writeHead(409);response.end();return;}
      doc=body;revision='r2';response.writeHead(200,{'Content-Type':'application/json',ETag:'"r2"'});response.end(JSON.stringify(doc));return;
    }
    response.writeHead(405);response.end();
  });
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  try {
    const baseUrl=`http://127.0.0.1:${server.address().port}/api`;
    const make=(overrides={})=>createSuiHttpPorts({baseUrl,applicationId:'sui',authorize:async()=>true,
      validate:async()=>true, fetchImpl:fetch,now:()=> '2026-10-08T02:00:00Z',
      headers:{'Sui-Sensemaking-Tenant-Session-Version':'1'},...overrides});
    await run({make,seen,read:()=>structuredClone(doc)});
  }finally{await new Promise((resolve,reject)=>server.close(e=>e?reject(e):resolve()));}
}
async function expectCode(p,code){await assert.rejects(p,e=>e instanceof BridgeError&&e.code===code);}
test('real HTTP GET + conditional PUT retain DocumentV1 extensions and ETag',async()=>{
  await harness(async ({make,seen,read})=>{
    const receipt=await commitIntent(gesture(),make());
    assert.equal(receipt.revision,'r2');
    assert.deepEqual(seen.map(x=>x.method),['GET','PUT']);
    assert.equal(seen[1].headers['if-match'],'"r1"');
    assert.equal(seen[0].headers['sui-sensemaking-tenant-session-version'],'1');
    assert.deepEqual([read().cards[0].x,read().cards[0].y],[21,34]);
    assert.equal(read().cards[0].holdState,'held');
    assert.deepEqual(read().opaqueFuture,{n:42});
    assert.equal(read().edges[0].type,'unknown.edge.kind');
    assert.equal(read().updatedAt,'2026-10-08T02:00:00Z');
  });
});
test('missing server ETag rejects before any PUT (never falls back to LWW)',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture(),make()),'missing_strong_etag');
    assert.deepEqual(seen.map(x=>x.method),['GET']);
  },{missingEtag:true});
});
test('weak server ETag rejects before any PUT',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture(),make()),'missing_strong_etag');
    assert.deepEqual(seen.map(x=>x.method),['GET']);
  },{weakEtag:true});
});
test('stale application revision fails before PUT',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture('r0'),make()),'stale_revision');
    assert.deepEqual(seen.map(x=>x.method),['GET']);
  });
});
test('server refuses concurrent write and adapter does not retry',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture(),make()),'conditional_save_failed');
    assert.deepEqual(seen.map(x=>x.method),['GET','PUT']);
  },{conflict:true});
});
test('authorization before GET, including deny',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture(),make({authorize:async()=>false})),'unauthorized');
    assert.equal(seen.length,0);
  });
});
test('provider authorization denial remains distinct',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent(gesture(),make()),'provider_unauthorized');
    assert.deepEqual(seen.map(x=>x.method),['GET']);
  },{readDenied:true});
});
test('cannot inject If-Match or Cookie through passthrough request headers',async()=>{
  await harness(async({make,seen})=>{
    for(const headers of [{'If-Match':'*'},{Cookie:'secret'},{'content-type':'text/html'}]){
      assert.throws(()=>make({headers}),e=>e instanceof BridgeError&&e.code==='unsafe_request_header');
    }
    assert.equal(seen.length,0);
  });
});
test('validates document resource binding and id at transport',async()=>{
  await harness(async({make,seen})=>{
    await expectCode(commitIntent({...gesture(),documentId:'other'},make()),'document_not_found');
    assert.deepEqual(seen.map(x=>x.method),['GET']);
  });
});
