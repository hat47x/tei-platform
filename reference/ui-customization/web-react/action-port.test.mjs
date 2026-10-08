import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {createReferenceSubmitAction,UIActionError} from './action-port.mjs';
import {createActionFormStarter,starterMetadata} from './starter.mjs';
import {referenceUIFields,createReferenceOrderEntryUI} from './order-entry-example.mjs';
import {projectSidePanel} from './project-extension.mjs';

const valid={loginId:'worker',credential:'private-input',from:'2026-09-18T08:30:00Z',row:0};
function payload(stage='completed',extra={}) {return {
  handoverBaselineId:'B',baselineRepositoryRevision:'R',targetBuildRevision:'T',
  result:{applicationId:'order_entry',scenarioId:'order-entry.submit-order-web-slice',stage,
    command:{capability:'sales.order.submit',mode:'output',payload:{accepted:true}},
    observation:{readIdentity:'synthetic.order_state.read',nature:'synthetic',result:'submitted'},
    ...extra}};}
function fakeReact(){
  const states=[];
  return {states,createElement:(type,props,...children)=>({type,props:props??{},children}),
    useState:(x)=>{const index=states.length;states.push(x);return [x,(v)=>{states[index]=v;}];}};
}
function allNodes(node){return (!node||typeof node!=='object')?[]:[node,...node.children.flatMap(allNodes)];}
async function withServer(responseFn,callback) {
  const calls=[];
  const server=createServer(async(req,res)=>{
    let body='';for await(const part of req)body+=part;
    calls.push({method:req.method,url:req.url,body,headers:req.headers});
    const response=responseFn(req,body);
    res.writeHead(response.status??200,{'Content-Type':'application/json'});
    res.end(typeof response.json==='string'?response.json:JSON.stringify(response.json));
  });
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  try {
    const url='http://127.0.0.1:'+server.address().port;
    const fetchImpl=(path,init)=>fetch(url+path,init);
    await callback({calls,fetchImpl});
  } finally {await new Promise((resolve,reject)=>server.close(e=>e?reject(e):resolve()));}
}
test('same First Web Target JSON contract via real HTTP; no business logic duplicated',async()=>{
  await withServer(()=>({json:payload()}),async({calls,fetchImpl})=>{
    const result=await createReferenceSubmitAction({fetchImpl})(valid);
    assert.equal(result.kind,'completed');assert.equal(result.command.capability,'sales.order.submit');
    assert.equal(result.observation.readIdentity,'synthetic.order_state.read');
    assert.equal(calls[0].method,'POST');assert.equal(calls[0].url,'/scenario/submit');
    assert.deepEqual(JSON.parse(calls[0].body),valid);
    assert.equal(calls[0].headers['content-type'],'application/json');
    assert.equal(JSON.stringify(result).includes('private-input'),false);
  });
});
test('auth and authorization denial remain business outcomes, not execution errors',async()=>{
  for(const stage of ['authentication-denied','authorization-denied']){
    await withServer(()=>({json:payload(stage,{command:null,observation:null})}),async({fetchImpl})=>{
      assert.equal((await createReferenceSubmitAction({fetchImpl})(valid)).kind,'business_denied');
    });
  }
});
test('validation not-satisfied preserves authoritative validation identity',async()=>{
  await withServer(()=>({json:payload('validation-not-satisfied',{
    validations:[{validation:'sales.order.min-amount',result:'not-satisfied'}],command:null})}),async({fetchImpl})=>{
      const r=await createReferenceSubmitAction({fetchImpl})(valid);
      assert.equal(r.kind,'validation_not_satisfied');assert.deepEqual(r.validations,[{validation:'sales.order.min-amount',result:'not-satisfied'}]);
    });
});
test('HTTP 422 execution errors are distinct; no error text/credential leaks',async()=>{
  await withServer(()=>({status:422,json:{...payload('read').result,unused:0}}),async({fetchImpl})=>{
    await assert.rejects(createReferenceSubmitAction({fetchImpl})(valid),e=>e.code==='application_mismatch');
  });
  await withServer(()=>({status:422,json:{...payload('read'),executionError:'private-input secret'}}),async({fetchImpl})=>{
    const r=await createReferenceSubmitAction({fetchImpl})(valid);
    assert.equal(r.kind,'execution_error');assert.equal(JSON.stringify(r).includes('private-input'),false);
  });
});
test('unexpected app binding and silent HTTP errors fail closed',async()=>{
  assert.throws(()=>createReferenceSubmitAction({fetchImpl:fetch,applicationId:'repair_intake'}),/unsupported_binding/);
  await withServer(()=>({json:payload('completed',{applicationId:'wrong'})}),async({fetchImpl})=>{
    await assert.rejects(createReferenceSubmitAction({fetchImpl})(valid),e=>e.code==='application_mismatch');
  });
  await withServer(()=>({status:403,json:{error:'forbidden'}}),async({fetchImpl})=>{
    await assert.rejects(createReferenceSubmitAction({fetchImpl})(valid),e=>e.code==='http_error');
  });
});
test('UI-only input validation rejects wrong row before network read/write',async()=>{
  let calls=0;
  const client=createReferenceSubmitAction({fetchImpl:()=>{calls++;throw Error('should not call');}});
  await assert.rejects(client({...valid,row:-1}),e=>e instanceof UIActionError&&e.code==='invalid_input');
  assert.equal(calls,0);
});
test('React UI scaffold composes project side panel without editing default fields or action',()=>{
  const React=fakeReact(),invoke=async()=>({kind:'completed'});
  const Standard=createActionFormStarter(React,{fields:referenceUIFields,invoke});
  const baseline=Standard();
  const Customized=createActionFormStarter(React,{fields:referenceUIFields,invoke,slots:{aside:projectSidePanel}});
  const custom=Customized();
  const baselineFields=allNodes(baseline).filter(n=>n.type==='input').map(n=>n.props.name);
  const customFields=allNodes(custom).filter(n=>n.type==='input').map(n=>n.props.name);
  assert.deepEqual(baselineFields,customFields);
  assert.deepEqual(customFields,['loginId','credential','from','row']);
  assert.equal(allNodes(baseline).filter(n=>n.type==='aside').length,0);
  assert.equal(allNodes(custom).filter(n=>n.type==='aside').length,1);
  assert.equal(starterMetadata.framework,'react');
});
test('React starter clears password after submit; project slot cannot override callback binding',async()=>{
  const React=fakeReact();let input;
  const UI=createActionFormStarter(React,{fields:referenceUIFields,invoke:async(v)=>{input=v;return {kind:'business_denied',stage:'authentication-denied'};}});
  const tree=UI();const form=allNodes(tree).find(n=>n.type==='form');
  const password={value:'private-input'};
  const {FormData:NativeFormData}=globalThis;
  const old=globalThis.FormData;
  globalThis.FormData=class {constructor(){this.values=valid;}get(n){return this.values[n];}};
  try{await form.props.onSubmit({preventDefault(){},currentTarget:{elements:{namedItem:()=>password}}});}
  finally{globalThis.FormData=old;}
  assert.deepEqual(input,valid);assert.equal(password.value,'');
  assert.equal(React.states.some(s=>JSON.stringify(s).includes('private-input')),false);
});
test('unknown slots and invalid renderer fields fail before creating a UI',()=>{
  const React=fakeReact();
  assert.throws(()=>createActionFormStarter(React,{fields:referenceUIFields,invoke:()=>{},slots:{bad:()=>{}}}),/unsupported_slot/);
  assert.throws(()=>createActionFormStarter(React,{fields:[{name:'x',label:'X',type:'text'},{name:'x',label:'Y',type:'text'}],invoke:()=>{}}),/invalid_ui_field/);
  assert.equal(typeof createReferenceOrderEntryUI(React,{fetchImpl:()=>{}}),'function');
});
