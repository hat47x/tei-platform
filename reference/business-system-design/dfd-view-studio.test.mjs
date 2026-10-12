import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';

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

test('a slower DFD read cannot replace a newer Procedure',async()=>{
  let release;
  const old=Buffer.from(JSON.stringify({...projection,procedureId:'old.procedure'}));
  const waiting=$('dfd-file').listeners.change({target:{files:[{
    arrayBuffer:()=>new Promise(resolve=>{release=()=>resolve(old.buffer.slice(old.byteOffset,old.byteOffset+old.length));})
  }]}});
  await input('dfd-file',{...projection,procedureId:'new.procedure'});
  assert.match($('status').textContent,/投影を読み込みました/);
  release();
  await waiting;
  assert.equal($('diagram').children.some(n=>n.dataset?.id==='BP-04'),true);
  assert.match($('status').textContent,/投影を読み込みました/);
  // Selecting a Step must still report the newer Procedure.
  const node=$('diagram').children.find(n=>n.dataset?.id==='BP-04');
  node.listeners.pointerdown({button:0,pointerId:91,clientX:0,clientY:0});
  assert.match($('details').children[0].textContent,/business.json/);
});

test('an older layout read cannot overwrite a newer DFD',async()=>{
  await input('dfd-file',projection);
  let release;
  const stale=Buffer.from(JSON.stringify({schemaVersion:'tei.reference.business-dfd-view-layout/v0',
    authority:'view-only',procedureId:'orders.review',
    projectionSHA256:'sha256:'+'0'.repeat(64),nodes:[]}));
  const waiting=$('layout-file').listeners.change({target:{files:[{
    arrayBuffer:()=>new Promise(resolve=>{release=()=>resolve(stale.buffer.slice(stale.byteOffset,stale.byteOffset+stale.length));})
  }]}});
  await input('dfd-file',{...projection,procedureId:'fresh.procedure'});
  release();
  await waiting;
  assert.match($('status').textContent,/投影を読み込みました/);
  assert.equal($('export').disabled,false);
  assert.equal($('apply').disabled,true);
});


const here=dirname(fileURLToPath(import.meta.url));
const reference=name=>JSON.parse(readFileSync(join(here,name),'utf8'));
const approvalProcedure=reference('dfd-order-approval/order-approval-procedure.json');
const approvalReview=reference('dfd-order-approval/review-proposal.json');
function approvalProjection(){
  const r=approvalReview,p=approvalProcedure;
  const source=(file,part,i)=>({file,pointer:'/'+part+'/'+i});
  return {
    schemaVersion:'business-dfd-projection-v0',authority:'derived-read-only',
    canonicalProcedureUsed:false,procedureId:p.procedure.id,diagnostics:[],
    roleAuthorizationBinding:'not-inferred',
    steps:p.steps.map((step,i)=>({id:step.id,label:step.businessAction,
      realization:step.realization,
      noApplicationFunction:(step.applicationPatterns||[]).length===0,
      functionRefs:(step.applicationPatterns||[]).map(id=>({id,status:'resolved'})),
      source:source('order-approval-procedure.json','steps',i)})),
    triggers:r.triggers.map((trigger,i)=>({...trigger,authority:'proposal',
      runtimeBinding:'unbound',source:source('review-proposal.json','triggers',i)})),
    flows:r.flows.map((v,i)=>({...v,authority:'proposal',
      source:source('review-proposal.json','flows',i)})),
    information:{
      roles:r.businessRoles.map((v,i)=>({...v,authority:'proposal',
        source:source('review-proposal.json','businessRoles',i)})),
      externalParties:r.externalParties.map((v,i)=>({...v,authority:'proposal',
        source:source('review-proposal.json','externalParties',i)})),
      stores:r.stores.map((v,i)=>({...v,authority:'proposal',
        dataStatus:v.semanticData?'resolved':'not-declared',
        source:source('review-proposal.json','stores',i)})),
      responsibilities:r.responsibilities.map((v,i)=>({...v,authority:'proposal',
        source:source('review-proposal.json','responsibilities',i)})),
      informationFlows:r.informationFlows.map((v,i)=>({...v,authority:'proposal',
        source:source('review-proposal.json','informationFlows',i)}))
    }
  };
}
function selectTrigger(){
  const node=$('diagram').children.find(x=>x.dataset?.kind==='trigger'&&x.dataset.id==='TR-ORDER');
  assert.ok(node);
  node.listeners.pointerdown({button:0,clientX:0,clientY:0,pointerId:55});
}
test('selected trigger generates a proposal-only browser review from exact source files',async()=>{
  const before=JSON.stringify(approvalReview);
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectTrigger();
  assert.equal($('trigger-label').disabled,false);
  assert.equal($('semantic-propose').disabled,false);
  $('trigger-label').value='担当者が受注確認を開始する';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  assert.equal($('semantic-preview').hidden,false);
  assert.equal($('semantic-export').disabled,false);
  const display=JSON.parse($('semantic-pending').textContent);
  assert.equal(display.reviewAuthority,'unaccepted-proposal-only');
  assert.equal(display.readyForAcceptance,false);
  assert.deepEqual(display.sourceWrites,[]);
  assert.equal(display.selected.sourceLocatorStatus,'reference-local-pointer-checked');
  assert.equal(display.selected.sourceFileAuthority,'unverified');
  assert.equal(JSON.stringify(approvalReview),before);
  $('trigger-label').value='別案';
  $('trigger-label').listeners.input();
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
});

test('browser refuses a mismatched DFD trigger and a tampered local source pointer',async()=>{
  const d=approvalProjection();
  d.triggers[0].label='古いレビューに基づく文言';
  await input('dfd-file',d);
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectTrigger();
  $('trigger-label').value='候補';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DFD_REVIEW_SNAPSHOT_MISMATCH_TRIGGERS/);
  assert.equal($('semantic-export').disabled,true);
  const bad=approvalProjection();
  bad.triggers[0].source.pointer='/triggers/999';
  await input('dfd-file',bad);
  selectTrigger();
  $('trigger-label').value='候補';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DFD_SELECTED_REFERENCE_LOCATOR_MISMATCH/);
  assert.equal($('semantic-export').disabled,true);
});

test('browser detects source bytes changing after initial file selection',async()=>{
  await input('dfd-file',approvalProjection());
  await input('review-file',approvalReview);
  const original=Buffer.from(JSON.stringify(approvalProcedure));
  const changed=Buffer.from(JSON.stringify({...approvalProcedure,procedure:{...approvalProcedure.procedure,id:'other-procedure'}}));
  let reads=0;
  const changingFile={arrayBuffer:async()=>{
    const buf=++reads===1?original:changed;
    return buf.buffer.slice(buf.byteOffset,buf.byteOffset+buf.length);
  }};
  await $('procedure-file').listeners.change({target:{files:[changingFile]}});
  selectTrigger();
  $('trigger-label').value='候補';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DFD_PROCEDURE_INPUT_CHANGED/);
  assert.equal($('semantic-export').disabled,true);
});

test('concurrent Procedure and Review loads stay independent',async()=>{
  await input('dfd-file',approvalProjection());
  const bytes=Buffer.from(JSON.stringify(approvalProcedure));
  const delayedProcedure={
    arrayBuffer:async()=>{
      await new Promise(resolve=>setTimeout(resolve,20));
      return bytes.buffer.slice(bytes.byteOffset,bytes.byteOffset+bytes.length);
    }
  };
  const pending=$('procedure-file').listeners.change({target:{files:[delayedProcedure]}});
  const reviewLoad=input('review-file',approvalReview);
  await Promise.all([pending,reviewLoad]);
  selectTrigger();
  $('trigger-label').value='手作業としての起動を説明する';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  assert.equal($('semantic-preview').hidden,false);
});

test('a trigger label modified during fingerprinting cannot resurrect a stale proposal',async()=>{
  await input('dfd-file',approvalProjection());
  await input('review-file',approvalReview);
  const raw=Buffer.from(JSON.stringify(approvalProcedure));
  let count=0;
  const delayedProcedure={arrayBuffer:async()=>{
    if(++count>1)await new Promise(resolve=>setTimeout(resolve,60));
    return raw.buffer.slice(raw.byteOffset,raw.byteOffset+raw.length);
  }};
  await $('procedure-file').listeners.change({target:{files:[delayedProcedure]}});
  selectTrigger();
  $('trigger-label').value='最初の説明文';
  const pending=$('semantic-propose').listeners.click();
  await new Promise(resolve=>setTimeout(resolve,5));
  $('trigger-label').value='途中で変更された説明文';
  $('trigger-label').listeners.input();
  await pending;
  assert.match($('status').textContent,/DFD_SELECTION_OR_SOURCE_CHANGED/);
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
});

function selectStep(id){
  const node=$('diagram').children.find(x=>x.dataset?.kind==='step'&&x.dataset.id===id);
  assert.ok(node,'Step '+id+' must exist');
  node.listeners.pointerdown({button:0,clientX:0,clientY:0,pointerId:77});
}

test('manual Step proposes an additional business Role without permission authority',async()=>{
  const snapshot=JSON.stringify({procedure:approvalProcedure,review:approvalReview});
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  assert.equal($('trigger-label').disabled,true);
  assert.equal($('step-role').disabled,false);
  assert.equal($('step-role').value,'approver');
  assert.equal($('semantic-propose').disabled,false);
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  const proposal=JSON.parse($('semantic-pending').textContent);
  assert.equal(proposal.change.kind,'responsibility');
  assert.equal(proposal.reviewAuthority,'unaccepted-proposal-only');
  assert.equal(proposal.readyForAcceptance,false);
  assert.deepEqual(proposal.sourceWrites,[]);
  assert.equal(proposal.selected.sourceLocatorStatus,'reference-local-pointer-checked');
  assert.equal($('semantic-preview').hidden,false);
  assert.equal($('semantic-export').disabled,false);
  assert.equal(JSON.stringify({procedure:approvalProcedure,review:approvalReview}),snapshot);
  assert.ok(approvalProcedure.steps.some(x=>x.id==='APP-10'&&!(x.applicationPatterns||[]).length));
});

test('a duplicate business Role and an unmatched Step cannot be proposed from the UI',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  assert.equal($('step-role').value,'approver');
  $('step-role').value='sales'; // Existing role, not an available choice.
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DUPLICATE_RESPONSIBILITY/);
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);

  await input('dfd-file',approvalProjection());
  selectStep('APP-20');
  assert.equal($('step-role').value,'sales');
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  assert.equal($('semantic-export').disabled,false);
  selectTrigger();
  assert.equal($('step-role').disabled,true);
  assert.equal($('semantic-export').disabled,true);
});

test('switching a selected Step or its Role invalidates the previous candidate',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  $('step-role').listeners.change();
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  selectStep('APP-20');
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
});

function chooseStepAction(action){
  $('step-action').value=action;
  $('step-action').listeners.change();
}
test('browser proposes a branch and exception from a selected business Step without execution authority',async()=>{
  const originals=JSON.stringify({procedure:approvalProcedure,review:approvalReview});
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  chooseStepAction('add-control-transition');
  assert.equal($('step-role').disabled,true);
  assert.equal($('step-transition-to').disabled,false);
  $('step-transition-to').value='APP-30';
  $('step-transition-to').listeners.change();
  $('step-transition-kind').value='branch';
  $('step-transition-kind').listeners.change();
  $('step-transition-condition').value='手作業で内容を再確認する';
  $('step-transition-condition').listeners.input();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  let candidate=JSON.parse($('semantic-pending').textContent);
  assert.equal(candidate.change.kind,'control-transition');
  assert.deepEqual(candidate.proposedBusinessOperation,{
    kind:'add-control-transition',from:'APP-10',to:'APP-30',
    transitionKind:'branch',condition:'手作業で内容を再確認する'
  });
  assert.equal(candidate.reviewAuthority,'unaccepted-proposal-only');
  assert.equal(candidate.readyForAcceptance,false);
  assert.deepEqual(candidate.sourceWrites,[]);
  assert.equal($('semantic-export').disabled,false);
  $('step-transition-kind').value='exception';
  $('step-transition-kind').listeners.change();
  assert.equal($('semantic-export').disabled,true);
  $('step-transition-condition').value='仕入先の回答が得られなかった場合';
  $('step-transition-condition').listeners.input();
  await $('semantic-propose').listeners.click();
  candidate=JSON.parse($('semantic-pending').textContent);
  assert.equal(candidate.proposedBusinessOperation.transitionKind,'exception');
  assert.equal(candidate.readyForAcceptance,false);
  assert.equal(JSON.stringify({procedure:approvalProcedure,review:approvalReview}),originals);
});

test('sequence has no condition, and a duplicate branch is refused',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  chooseStepAction('add-control-transition');
  $('step-transition-to').value='APP-30';
  $('step-transition-to').listeners.change();
  $('step-transition-kind').value='sequence';
  $('step-transition-kind').listeners.change();
  assert.equal($('step-transition-condition').disabled,true);
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  assert.equal(Object.hasOwn(JSON.parse($('semantic-pending').textContent)
    .proposedBusinessOperation,'condition'),false);
  selectStep('APP-20');
  chooseStepAction('add-control-transition');
  $('step-transition-to').value='APP-40';
  $('step-transition-to').listeners.change();
  $('step-transition-kind').value='branch';
  $('step-transition-kind').listeners.change();
  $('step-transition-condition').value='business approver accepts order (hypothesis)';
  $('step-transition-condition').listeners.input();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DUPLICATE_CONTROL_FLOW/);
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
});

test('Step to external party and conceptual Store are information-only proposals',async()=>{
  const originals=JSON.stringify({procedure:approvalProcedure,review:approvalReview});
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  chooseStepAction('add-information-flow');
  assert.equal($('step-info-id').disabled,false);
  assert.equal($('step-info-to-kind').value,'party');
  $('step-info-id').value='INFO-NEW-1';
  $('step-info-id').listeners.input();
  $('step-info-to-id').value='customer';
  $('step-info-to-id').listeners.change();
  $('step-info-payload').value='配送希望日の照会';
  $('step-info-payload').listeners.input();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/未受理の変更候補を確認/);
  let proposal=JSON.parse($('semantic-pending').textContent);
  assert.equal(proposal.change.kind,'information-flow');
  assert.deepEqual(proposal.proposedBusinessOperation,{
    kind:'add-information-flow',id:'INFO-NEW-1',
    from:{kind:'step',id:'APP-10'},to:{kind:'party',id:'customer'},
    payload:'配送希望日の照会'
  });
  assert.equal(proposal.readyForAcceptance,false);
  assert.deepEqual(proposal.sourceWrites,[]);
  $('step-info-to-kind').value='store';
  $('step-info-to-kind').listeners.change();
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('step-info-to-id').value,'orders');
  $('step-info-id').value='INFO-NEW-2';
  $('step-info-id').listeners.input();
  $('step-info-payload').value='注文情報の確認要求';
  $('step-info-payload').listeners.input();
  await $('semantic-propose').listeners.click();
  proposal=JSON.parse($('semantic-pending').textContent);
  assert.deepEqual(proposal.proposedBusinessOperation.to,{kind:'store',id:'orders'});
  assert.equal(proposal.readyForAcceptance,false);
  assert.equal(JSON.stringify({procedure:approvalProcedure,review:approvalReview}),originals);
  assert.equal($('semantic-export').disabled,false);
});

test('information ID duplication, invalid endpoint and Step re-selection discard proposals',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectStep('APP-10');
  chooseStepAction('add-information-flow');
  $('step-info-id').value='INFO-ORDER';
  $('step-info-id').listeners.input();
  $('step-info-payload').value='検討中の授受';
  $('step-info-payload').listeners.input();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DUPLICATE_INFORMATION_FLOW/);
  assert.equal($('semantic-export').disabled,true);
  $('step-info-id').value='INFO-NEW-3';
  $('step-info-id').listeners.input();
  $('step-info-to-id').value='unknown-party';
  $('step-info-to-id').listeners.change();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/UNRESOLVED_ENDPOINT/);
  assert.equal($('semantic-preview').hidden,true);
  $('step-info-to-id').value='customer';
  $('step-info-to-id').listeners.change();
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  selectStep('APP-20');
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('step-action').value,'assign-business-role');
  assert.equal($('step-info-to-kind').value,'party');
  assert.equal($('step-info-id').value,'');
  assert.equal($('step-info-payload').value,'');
});

function selectInformationOrigin(kind,id){
  const node=$('diagram').children.find(x=>x.dataset?.kind===kind&&x.dataset.id===id);
  assert.ok(node,kind+' '+id+' must be rendered');
  node.listeners.pointerdown({button:0,clientX:0,clientY:0,pointerId:81});
}

test('selected Party or Store produces inbound information-only review without acceptance',async()=>{
  const originals=JSON.stringify({procedure:approvalProcedure,review:approvalReview});
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  for(const [kind,id,flowID,target] of [
    ['party','supplier','INFO-INBOUND-PARTY','APP-15'],
    ['store','orders','INFO-INBOUND-STORE','APP-00']
  ]){
    selectInformationOrigin(kind,id);
    assert.equal($('step-action').value,'add-information-flow');
    assert.equal($('step-action').disabled,true);
    assert.equal($('trigger-label').disabled,true);
    assert.equal($('step-role').disabled,true);
    assert.equal($('step-transition-to').disabled,true);
    assert.equal($('step-info-to-kind').value,'step');
    assert.equal($('step-info-to-kind').disabled,true);
    assert.equal($('step-info-to-id').disabled,false);
    assert.equal($('semantic-propose').disabled,false);
    $('step-info-id').value=flowID;
    $('step-info-id').listeners.input();
    $('step-info-to-id').value=target;
    $('step-info-to-id').listeners.change();
    $('step-info-payload').value='参照情報の伝達';
    $('step-info-payload').listeners.input();
    await $('semantic-propose').listeners.click();
    assert.match($('status').textContent,/未受理の変更候補を確認/);
    const receipt=JSON.parse($('semantic-pending').textContent);
    assert.equal(receipt.change.kind,'information-flow');
    assert.deepEqual(receipt.proposedBusinessOperation,{
      kind:'add-information-flow',id:flowID,
      from:{kind,id},to:{kind:'step',id:target},payload:'参照情報の伝達'
    });
    assert.equal(receipt.selected.sourceLocatorStatus,'reference-local-pointer-checked');
    assert.equal(receipt.selected.sourceFileAuthority,'unverified');
    assert.equal(receipt.reviewAuthority,'unaccepted-proposal-only');
    assert.equal(receipt.readyForAcceptance,false);
    assert.deepEqual(receipt.sourceWrites,[]);
    assert.equal($('semantic-export').disabled,false);
    assert.equal(JSON.stringify({procedure:approvalProcedure,review:approvalReview}),originals);
  }
});

test('inbound selection rejects non-Step destinations, duplicate IDs and stale candidates',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectInformationOrigin('party','customer');
  $('step-info-id').value='INFO-DATE';
  $('step-info-id').listeners.input();
  $('step-info-payload').value='重複する情報';
  $('step-info-payload').listeners.input();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/DUPLICATE_INFORMATION_FLOW/);
  assert.equal($('semantic-export').disabled,true);

  $('step-info-id').value='INFO-INBOUND-NEW';
  $('step-info-id').listeners.input();
  $('step-info-to-kind').value='store';
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/有効な変更内容/);
  assert.equal($('semantic-preview').hidden,true);
  $('step-info-to-kind').value='step';
  $('step-info-to-id').value='unknown-step';
  $('step-info-to-id').listeners.change();
  await $('semantic-propose').listeners.click();
  assert.match($('status').textContent,/UNRESOLVED_ENDPOINT/);
  assert.equal($('semantic-export').disabled,true);

  $('step-info-to-id').value='APP-10';
  $('step-info-to-id').listeners.change();
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  selectInformationOrigin('store','orders');
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('step-info-id').value,'');
  assert.equal($('step-info-to-kind').value,'step');
});

test('clearing DFD and review pickers revokes pending proposals and disables exports',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectTrigger();
  $('trigger-label').value='新しい参照上の起動説明';
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  assert.equal($('semantic-export').disabled,false);

  await $('review-file').listeners.change({target:{files:[]}});
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('semantic-propose').disabled,true);
  assert.match($('status').textContent,/選択を解除/);

  await input('review-file',approvalReview);
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  await $('procedure-file').listeners.change({target:{files:[]}});
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('semantic-propose').disabled,true);

  await input('procedure-file',approvalProcedure);
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-preview').hidden,false);
  await $('dfd-file').listeners.change({target:{files:[]}});
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('semantic-propose').disabled,true);
  assert.equal($('export').disabled,true);
  assert.equal($('diagram').children.length,0);
  assert.match($('status').textContent,/選択を解除/);
});

test('DFD replacement invalidates previous proposal before async bytes arrive',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectTrigger();
  $('trigger-label').value='編集中の未受理候補';
  await $('semantic-propose').listeners.click();
  assert.equal($('semantic-export').disabled,false);

  const old=Buffer.from(JSON.stringify({...approvalProjection(),procedureId:'stale-read'}));
  const pending=$('dfd-file').listeners.change({target:{files:[{
    arrayBuffer:async()=>{
      await new Promise(resolve=>setTimeout(resolve,35));
      return old.buffer.slice(old.byteOffset,old.byteOffset+old.length);
    }
  }]}});
  assert.equal($('semantic-export').disabled,true);
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-propose').disabled,true);
  assert.equal($('export').disabled,true);
  assert.equal($('diagram').children.length,0);

  // A newer selection must win over a late result from the older file.
  await input('dfd-file',approvalProjection());
  await pending;
  assert.equal($('export').disabled,false);
  assert.match($('status').textContent,/投影を読み込みました/);
  assert.ok($('diagram').children.some(x=>x.dataset?.id==='TR-ORDER'));
  assert.equal($('semantic-preview').hidden,true);
  assert.equal($('semantic-export').disabled,true);
});

test('clearing a reference file defeats its outstanding asynchronous load',async()=>{
  await input('dfd-file',approvalProjection());
  await input('procedure-file',approvalProcedure);
  await input('review-file',approvalReview);
  selectTrigger();

  const bytes=Buffer.from(JSON.stringify(approvalReview));
  const pending=$('review-file').listeners.change({target:{files:[{
    arrayBuffer:async()=>{
      await new Promise(resolve=>setTimeout(resolve,35));
      return bytes.buffer.slice(bytes.byteOffset,bytes.byteOffset+bytes.length);
    }
  }]}});
  assert.equal($('semantic-propose').disabled,true);
  await $('review-file').listeners.change({target:{files:[]}});
  await pending;
  assert.equal($('semantic-propose').disabled,true);
  assert.equal($('semantic-export').disabled,true);
  assert.match($('status').textContent,/選択を解除/);

  await input('review-file',approvalReview);
  assert.equal($('semantic-propose').disabled,false);
});
