import test from 'node:test';
import assert from 'node:assert/strict';

class Element {
  constructor(tag){this.tag=tag;this.textContent='';this.children=[];this.listeners={};this.classList={toggle(){}};this.dataset={};this.hidden=false;this.disabled=false;this.value='';this.style={};this.viewBox={baseVal:{width:1100,height:600}};}
  setAttribute(k,v){
    if(k==='viewBox'){
      const [x,y,width,height]=String(v).split(' ').map(Number);
      this.viewBox.baseVal={x,y,width,height};
    }else this[k]=v;
  }
  append(...items){this.children.push(...items)}
  replaceChildren(...items){this.children=[...items]}
  addEventListener(type,fn){this.listeners[type]=fn}
  getBoundingClientRect(){return {width:1100,height:600}}
  setPointerCapture(){}
  click(){this.listeners.click?.()}
}
const elements=new Map();
const $ = id => {if(!elements.has(id))elements.set(id,new Element(id));return elements.get(id)};
globalThis.document={getElementById:$,createElementNS:(ns,tag)=>new Element(tag),createElement:tag=>new Element(tag)};
const nativeURL=globalThis.URL;
nativeURL.createObjectURL??=()=> 'blob:mock';
nativeURL.revokeObjectURL??=()=> {};
const projection={schemaVersion:'business-dfd-projection-v0',authority:'derived-read-only',canonicalProcedureUsed:false,
  procedureId:'orders.review',steps:[
    {id:'BP-04',label:'manual telephone call',noApplicationFunction:true,functionRefs:[],source:{file:'business.json',pointer:'/steps/0'}},
    {id:'BP-07',label:'submit order',noApplicationFunction:false,functionRefs:[{id:'order.submit',status:'resolved'}],source:{file:'business.json',pointer:'/steps/1'}}
  ],triggers:[],flows:[{from:'BP-04',to:'BP-07',kind:'sequence'}],
  handoffs:[{from:{procedure:'orders.review',step:'BP-04'},to:{procedure:'warehouse.prepare',step:'WH-00'},condition:'confirmed'}],
  diagnostics:[],information:{roles:[{id:'reviewer',label:'reviewer'}],externalParties:[],stores:[],informationFlows:[],responsibilities:[{step:'BP-04',role:'reviewer'}]}};
const file=value=>{const buf=Buffer.from(JSON.stringify(value));return{arrayBuffer:async()=>buf.buffer.slice(buf.byteOffset,buf.byteOffset+buf.length),text:async()=>buf.toString('utf8')}};
const input=(id, obj)=>$(id).listeners.change({target:{files:[file(obj)]}});

test('reference-only SVG UI loads, selects and drags a manual Step without source updates',async()=>{
  await import('./dfd-view-studio.mjs');
  await input('dfd-file',projection);
  assert.match($('status').textContent,/投影を読み込みました/);
  assert.equal($('export').disabled,false);
  const node=$('diagram').children.find(x=>x.dataset?.id==='BP-04');
  assert.ok(node,'explicit reference step must be rendered');
  node.listeners.pointerdown({button:0,clientX:20,clientY:30,pointerId:1});
  assert.match($('details').children[0].textContent,/business.json/);
  assert.match($('details').children[0].textContent,/warehouse.prepare/);
  assert.ok($('diagram').children.some(x=>x.class==='edge responsibility'));
  assert.ok($('diagram').children.some(x=>x.class==='edge function'));
  await $('diagram').listeners.pointerup({clientX:90,clientY:80});
  assert.equal($('apply').disabled,false);
  assert.equal($('preview').hidden,false);
  assert.match($('pending').textContent,/move-node/);
  $('apply').listeners.click();
  await new Promise(resolve=>setTimeout(resolve,30));
  assert.equal($('apply').disabled,true);
  assert.equal($('preview').hidden,true);
  assert.match($('status').textContent,/正本には書き込んでいません/);
  assert.equal(projection.steps[0].source.pointer,'/steps/0');
});

test('stale view file and producer errors cannot replace displayed layout',async()=>{
  const invalidLayout={schemaVersion:'tei.reference.business-dfd-view-layout/v0',authority:'view-only',procedureId:'orders.review',projectionSHA256:'sha256:'+'0'.repeat(64),nodes:[]};
  await input('layout-file',invalidLayout);
  assert.match($('status').textContent,/配置ファイルを拒否しました/);
  const invalid={...projection,diagnostics:[{severity:'error',code:'BAD'}]};
  await input('dfd-file',invalid);
  assert.match($('status').textContent,/DFD投影を読み込めません/);
  assert.equal($('export').disabled,true);
  const duplicate={...projection,steps:[projection.steps[0],projection.steps[0]]};
  await input('dfd-file',duplicate);
  assert.match($('status').textContent,/DFD投影を読み込めません/);
  assert.equal($('export').disabled,true);
});

test('large Procedure remains scrollable instead of clipping all Steps below initial board',async()=>{
  const steps=Array.from({length:32},(_,i)=>({id:'B-'+i,label:'step '+i,
    noApplicationFunction:true,functionRefs:[]}));
  await input('dfd-file',{...projection,steps,flows:[],handoffs:[],
    information:{...projection.information,responsibilities:[],informationFlows:[]}});
  assert.match($('status').textContent,/投影を読み込みました/);
  assert.equal($('diagram').children.filter(n=>n.dataset?.kind==='step').length,32);
  const height=$('diagram').viewBox.baseVal.height;
  assert.ok(height>600,'diagram height should expand with number of Steps: '+height);
  assert.ok(parseInt($('diagram').style.height,10)>600);
});

test('keyboard placement and pointer cancellation preserve view-only editing',async()=>{
  await input('dfd-file',projection);
  let node=$('diagram').children.find(x=>x.dataset?.id==='BP-04');
  assert.equal(node.role,'button');
  assert.equal(node.tabindex,'0');
  let prevented=false;
  node.listeners.keydown({key:'ArrowRight',shiftKey:false,preventDefault(){prevented=true;}});
  assert.equal(prevented,true);
  assert.equal($('apply').disabled,false);
  assert.match($('pending').textContent,/"x": 60/);
  $('apply').listeners.click();
  await new Promise(resolve=>setTimeout(resolve,30));
  assert.equal($('apply').disabled,true);
  assert.match($('status').textContent,/正本には書き込んでいません/);
  node=$('diagram').children.find(x=>x.dataset?.id==='BP-04');
  node.listeners.pointerdown({button:0,pointerId:41,clientX:20,clientY:30});
  $('diagram').listeners.pointercancel({pointerId:41});
  $('diagram').listeners.pointerup({pointerId:41,clientX:400,clientY:300});
  assert.equal($('apply').disabled,true,'canceled pointer must not submit a move');
  node=$('diagram').children.find(x=>x.dataset?.id==='BP-04');
  node.listeners.pointerdown({button:0,pointerId:42,clientX:20,clientY:30});
  $('diagram').listeners.pointerup({pointerId:43,clientX:400,clientY:300});
  assert.equal($('apply').disabled,true,'other pointer must not submit a move');
  $('diagram').listeners.pointerup({pointerId:42,clientX:60,clientY:80});
  assert.equal($('apply').disabled,false,'active pointer should still be recognized');
});

test('reloading a Procedure clears stale Source details and an in-progress drag',async()=>{
  await input('dfd-file',projection);
  const node=$('diagram').children.find(x=>x.dataset?.id==='BP-04');
  node.listeners.pointerdown({button:0,pointerId:70,clientX:0,clientY:0});
  assert.match($('details').children[0].textContent,/business.json/);
  await input('dfd-file',{...projection,procedureId:'another.procedure',
    steps:[{id:'NEW',label:'new',functionRefs:[],noApplicationFunction:true}],flows:[],handoffs:[],
    information:{...projection.information,responsibilities:[],informationFlows:[]}});
  assert.equal($('details').children.length,0);
  assert.match($('details').textContent,/要素を選択すると/);
  $('diagram').listeners.pointerup({pointerId:70,clientX:250,clientY:200});
  assert.equal($('apply').disabled,true);
  assert.doesNotMatch($('status').textContent,/配置変更の候補ができました/);
});

test('direction and distinct branch, exception, and information labels remain visible',async()=>{
  const steps=['S0','S1','S2'].map(id=>({id,label:id,functionRefs:[],noApplicationFunction:true}));
  const scenario={...projection,steps,triggers:[{id:'ORDER',kind:'event',targetStep:'S0',label:'order received'}],
    flows:[
      {from:'S0',to:'S1',kind:'branch',condition:'approved'},
      {from:'S0',to:'S2',kind:'exception',condition:'returned'}
    ],
    information:{roles:[],externalParties:[{id:'customer',label:'Customer'}],stores:[],
      responsibilities:[],informationFlows:[{id:'INFO',from:{kind:'party',id:'customer'},
        to:{kind:'step',id:'S0'},payload:'order details'}]}
  };
  await input('dfd-file',scenario);
  assert.match($('status').textContent,/投影を読み込みました/);
  const elements=$('diagram').children;
  for(const kind of ['branch','exception','information']){
    assert.ok(elements.some(el=>el.class==='edge '+kind),'missing directed '+kind+' arrow');
  }
  assert.ok(elements.some(el=>el['marker-end']==='url(#dfd-arrow)'));
  const captions=elements.filter(el=>el.class==='edge-label').map(el=>el.textContent);
  for(const label of ['branch: approved','exception: returned','order details']){
    assert.ok(captions.includes(label),'missing explicit label '+label);
  }
});
