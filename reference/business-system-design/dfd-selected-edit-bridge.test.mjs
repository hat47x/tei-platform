import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {dirname,join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {previewSelectedDFDEdit,BINDING_SCHEMA} from './dfd-selected-edit-bridge.mjs';

const dir=dirname(fileURLToPath(import.meta.url));
const read=name=>JSON.parse(readFileSync(join(dir,name),'utf8'));
const procedure=read('dfd-order-approval/order-approval-procedure.json');
const review=read('dfd-order-approval/review-proposal.json');
const procedureID=procedure.procedure.id;
const hash=a=>'sha256:'+a.repeat(64);
function projection(p=procedure,r=review) {
  return {
    schemaVersion:'business-dfd-projection-v0',authority:'derived-read-only',
    canonicalProcedureUsed:false,procedureId:p.procedure.id,
    roleAuthorizationBinding:'not-inferred',
    diagnostics:[],
    steps:p.steps.map((s,i)=>({id:s.id,label:s.businessAction,
      realization:s.realization,functionRefs:(s.applicationPatterns||[]).map(id=>({id,status:'resolved'})),
      noApplicationFunction:(s.applicationPatterns||[]).length===0,
      source:{file:'order-approval-procedure.json',pointer:'/steps/'+i}})),
    triggers:r.triggers.map((t,i)=>({...t,authority:'proposal',
      source:{file:'review-proposal.json',pointer:'/triggers/'+i},runtimeBinding:'unbound'})),
    flows:r.flows.map((f,i)=>({...f,authority:'proposal',
      source:{file:'review-proposal.json',pointer:'/flows/'+i}})),
    information:{
      roles:r.businessRoles.map(x=>({...x,authority:'proposal'})),
      externalParties:r.externalParties.map(x=>({...x,authority:'proposal'})),
      stores:r.stores.map(x=>({...x,authority:'proposal',dataStatus:x.semanticData?'resolved':'not-declared'})),
      responsibilities:r.responsibilities.map(x=>({...x,authority:'proposal'})),
      informationFlows:r.informationFlows.map(x=>({...x,authority:'proposal'}))
    }
  };
}
const base=projection();
const clone=x=>JSON.parse(JSON.stringify(x));
const proposal=(selection,operation)=>({
  schema:BINDING_SCHEMA,id:'DFD-CANDIDATE-1',origin:'studio',
  intent:'業務上のレビュー候補だけを記述する',
  procedureSHA256:hash('1'),reviewSHA256:hash('2'),projectionSHA256:hash('3'),
  selection,operation
});
const act=(b,p=base,proc=procedure,r=review)=>previewSelectedDFDEdit(p,proc,r,b);

test('a manual Step may acquire a business role without any Function or RBAC authority',()=>{
  const step=base.steps.find(s=>s.noApplicationFunction);
  assert.ok(step);
  const b=proposal({kind:'step',id:step.id},
    {kind:'assign-business-role',step:step.id,role:'approver'});
  const before=JSON.stringify({base,review,procedure});
  const p=act(b);
  assert.equal(p.authority,'unaccepted-proposal-only');
  assert.equal(p.selected.scope,'reference-procedure:'+procedureID);
  assert.equal(p.selected.acceptedIdentity,'unverified');
  assert.equal(p.sharedAcceptanceCandidate,null);
  assert.equal(p.readyForAcceptance,false);
  assert.equal(p.editableCanonicalSource,false);
  assert.deepEqual(p.sourceWrites,[]);
  assert.equal(p.change.kind,'responsibility');
  assert.equal(JSON.stringify({base,review,procedure}),before);
});

test('a selected trigger can change its label but cannot change its runtime binding',()=>{
  const b=proposal({kind:'trigger',id:'TR-ORDER'},
    {kind:'replace-trigger-label',trigger:'TR-ORDER',label:'新しい業務上の説明'});
  const p=act(b);
  assert.equal(p.change.kind,'trigger');
  assert.equal(p.proposedReview.triggers[0].kind,'event');
  assert.equal(p.proposedReview.triggers[0].label,'新しい業務上の説明');
  assert.equal(p.freshness,'unverified-no-model-revision-binding');
  assert.throws(()=>act({...b,operation:{...b.operation,runtimeCron:'*/5 * * * *'}}),
    /UNKNOWN_OR_INVALID_FIELD/);
});

test('a selected Step can propose explicit exception without executable Handoff',()=>{
  const b=proposal({kind:'step',id:'APP-40'},
    {kind:'add-control-transition',from:'APP-40',to:'APP-00',
      transitionKind:'exception',condition:'承認保留'});
  const p=act(b);
  assert.equal(p.proposedReview.flows.at(-1).kind,'exception');
  assert.equal(p.readyForAcceptance,false);
});

test('party to Step transfers information without a direct Store write',()=>{
  const b=proposal({kind:'party',id:'supplier'},
    {kind:'add-information-flow',id:'INFO-NEW',
      from:{kind:'party',id:'supplier'},to:{kind:'step',id:'APP-40'},payload:'在庫回答'});
  const p=act(b);
  assert.equal(p.change.kind,'information-flow');
  assert.equal(p.proposedReview.informationFlows.at(-1).payload,'在庫回答');
});

test('a stale projection of any authored relation blocks selected editing',()=>{
  const mutations=[
    x=>x.steps[0].label='改変された業務作業',
    x=>x.steps[0].functionRefs=[],
    x=>x.triggers[0].label='古い契機',
    x=>x.flows[0].to='APP-40',
    x=>x.information.roles[0].label='別の役割',
    x=>x.information.externalParties[0].label='別の相手方',
    x=>x.information.stores[0].semanticData='sales.other',
    x=>x.information.responsibilities[0].role='approver',
    x=>x.information.informationFlows[0].payload='別の情報'
  ];
  for(const mutate of mutations){
    const p=clone(base);mutate(p);
    assert.throws(()=>act(proposal({kind:'trigger',id:'TR-ORDER'},
      {kind:'replace-trigger-label',trigger:'TR-ORDER',label:'候補'}),p),
      /DFD_REVIEW_SNAPSHOT_MISMATCH_/);
  }
});

test('rejects wrong selected kind, renamed target or unknown step',()=>{
  const operation={kind:'assign-business-role',step:'APP-40',role:'approver'};
  for(const selected of [{kind:'step',id:'APP-10'},
    {kind:'trigger',id:'TR-ORDER'},{kind:'step',id:'missing'}]){
    assert.throws(()=>act(proposal(selected,operation)),
      /DFD_SELECTION_EDIT_TARGET_MISMATCH|UNRESOLVED_OR_AMBIGUOUS_DFD_SELECTION/);
  }
});
test('neither supplied SHA fields nor labels confer accepted Source authority',()=>{
  const b=proposal({kind:'trigger',id:'TR-ORDER'},
    {kind:'replace-trigger-label',trigger:'TR-ORDER',label:'候補'});
  for(const v of [
    {...b,reviewSHA256:'not-a-sha'},
    {...b,baseModelRevision:hash('a')},
    {...b,selection:{...b.selection,sourceWrite:true}},
    {...b,origin:'acceptance'}
  ])assert.throws(()=>act(v),/INVALID_SELECTED_EDIT_BINDING|UNSUPPORTED_BINDING_ATTRIBUTE/);
  assert.throws(()=>act(b,{...base,canonicalProcedureUsed:true}),
    /INVALID_DFD_PRODUCER_AUTHORITY_OR_DIAGNOSTIC/);
  assert.throws(()=>act(b,{...base,diagnostics:[{severity:'error'}]}),
    /INVALID_DFD_PRODUCER_AUTHORITY_OR_DIAGNOSTIC/);
});
test('refuses a different local Procedure even when Step identifiers coincide',()=>{
  const b=proposal({kind:'trigger',id:'TR-ORDER'},
    {kind:'replace-trigger-label',trigger:'TR-ORDER',label:'候補'});
  assert.throws(()=>act(b,{...base,procedureId:'different.procedure'}),
    /DFD_REVIEW_PROCEDURE_IDENTITY_MISMATCH/);
});
