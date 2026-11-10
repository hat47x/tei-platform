import test from 'node:test';
import assert from 'node:assert/strict';
import {createActionFormStarter} from './starter.mjs';

const fields = [
  {name:'credential',type:'password',label:'Credential'},
  {name:'row',type:'number',label:'Row'},
];
function deferred() {
  let resolve, reject;
  const promise = new Promise((ok, no) => {resolve=ok; reject=no;});
  return {promise, resolve, reject};
}
function harness(invoke) {
  const state = [];
  const React = {
    createElement:(type,props,...children)=>({type,props:props??{},children}),
    useState:(initial)=>{
      const index=state.length;
      state.push(initial);
      return [initial,(value)=>{state[index]=value;}];
    },
    useRef:(initial)=>({current:initial}),
  };
  const UI=createActionFormStarter(React,{fields,invoke});
  const tree=UI();
  const form=tree.children.find(node=>node?.type==='form');
  const password={value:'example-password'};
  const event={preventDefault(){},currentTarget:{elements:{namedItem:(name)=>name==='credential'?password:null}}};
  return {form,event,password,state};
}
async function withFormData(values,run) {
  const original=globalThis.FormData;
  globalThis.FormData=class {get(name){return values[name];}};
  try {await run();} finally {globalThis.FormData=original;}
}
const completed={kind:'completed',stage:'completed'};

test('one in-flight invoke per form even when same onSubmit closure fires twice',async()=>{
  const first=deferred();
  let invokes=0;
  const {form,event,state,password}=harness(()=>{invokes++;return first.promise;});
  await withFormData({credential:'example-password',row:'0'},async()=>{
    const a=form.props.onSubmit(event);
    const b=form.props.onSubmit(event);
    assert.equal(invokes,1,'second immediate submit must be ignored before re-render');
    assert.equal(state[0],true);
    first.resolve(completed);
    await Promise.all([a,b]);
    assert.equal(state[0],false);
    assert.equal(password.value,'');
  });
});

test('in-flight guard is released on transport failure; subsequent deliberate submit succeeds',async()=>{
  const first=deferred();
  let invokes=0;
  const {form,event,state}=harness(()=>++invokes===1?first.promise:Promise.resolve(completed));
  await withFormData({credential:'example-password',row:'1'},async()=>{
    const a=form.props.onSubmit(event);
    const blocked=form.props.onSubmit(event);
    assert.equal(invokes,1);
    first.reject(Error('synthetic network failure'));
    await Promise.all([a,blocked]);
    assert.equal(state[0],false);
    assert.deepEqual(state[1],{kind:'input_or_transport_error',stage:'unavailable'});
    await form.props.onSubmit(event);
    assert.equal(invokes,2);
    assert.deepEqual(state[1],{kind:'completed',stage:'completed',command:null,observation:null});
  });
});

test('invalid row releases the guard without invoking action',async()=>{
  let invokes=0;
  const {form,event,state}=harness(async()=>{invokes++;return completed;});
  const values={credential:'example-password',row:'-1'};
  await withFormData(values,async()=>{
    await form.props.onSubmit(event);
    assert.equal(invokes,0);
    assert.deepEqual(state[1],{kind:'input_or_transport_error',stage:'unavailable'});
    values.row='2';
    await form.props.onSubmit(event);
    assert.equal(invokes,1);
    assert.equal(state[0],false);
    assert.equal(state[1].kind,'completed');
  });
});
