import test from 'node:test';
import assert from 'node:assert/strict';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {observeE27} from './e27_residual_selection_audit.mjs';
import {assessE28,compareE28,observeE28} from './e28_workflow_measurement.mjs';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'../..');
const evidenceRef='https://github.com/hat47x/tei-platform/pull/1505';
const shaA='a'.repeat(40), shaB='b'.repeat(40);
const scenario={id:'order-entry-project-advice',consumer:'order_entry',arm:'existing-source',
  baseRevision:shaA,target:'web'};
const step=(id,operation,extra={})=>({
  id,operation,choiceId:'web.ui-kit-target-adoption',evidenceRef,...extra,
});
const submission=(events,extra={})=>({
  kind:'tei-e28-workflow-submission-v0',complete:false,
  scenario:{...scenario},events,...extra,
});

test('E28 live E1/E27 checkout reports observations, not fabricated work hours or AI questions',async()=>{
  const report=await observeE28(root);
  assert.equal(report.kind,'tei-e28-workflow-measurement-v0');
  assert.deepEqual(report.sourceSummary,{
    sourceObserved:4,stage0ObservedOnly:1,selectionUnresolved:3,
    runtimeProofUnresolved:2,sourceUnresolved:0,
  });
  assert.equal(report.scope,'observed-e1-source-only');
  assert.equal(report.scenario,null);
  assert.equal(report.submittedCounts,null);
  assert.equal(report.verifiedCounts,null);
  assert.equal(report.runtimeAuthorityVerified,false);
});

test('E28 synthetic submitted workflow counts events and distinct changed files only as unverified',async()=>{
  const audit=await observeE27(root);
  const source={path:'reference/ui-customization/web-react/project-extension.mjs',
    beforeBlob:shaA,afterBlob:shaB};
  const events=[
    step('decision1','human-decision'),
    step('question1','ai-question'),
    step('connection1','manual-connection'),
    step('edit1','changed-source',source),
    step('edit2','changed-source',{...source,beforeBlob:shaB,afterBlob:shaA}),
    step('test1','validation-run',{outcome:'passed'}),
    step('impact1','requalification-run',{outcome:'unresolved'}),
  ];
  const report=assessE28(audit,submission(events));
  assert.equal(report.scope,'partial-not-verified');
  assert.deepEqual(report.submittedCounts,{
    humanDecisions:1,aiQuestions:1,manualConnections:1,
    changedFiles:1,validationRuns:1,requalificationRuns:1,
  });
  assert.equal(report.verifiedCounts,null);
  assert.equal(report.compatibilityVerified,false);
  assert.equal(report.runtimeAuthorityVerified,false);
  const claimed=assessE28(audit,submission(events,{complete:true}));
  assert.equal(claimed.scope,'claimed-complete-not-verified');
  assert.equal(claimed.verifiedCounts,null);
});

test('E28 absent evidence or falsified source revision cannot become a qualified work step',async()=>{
  const audit=await observeE27(root);
  for (const event of [
    step('bad1','changed-source',{path:'../secret',beforeBlob:shaA,afterBlob:shaB}),
    step('bad2','changed-source',{path:'a/b',beforeBlob:shaA,afterBlob:shaA}),
    step('bad3','changed-source',{path:'a/b',beforeBlob:'invalid',afterBlob:shaB}),
    step('bad4','ai-question',{evidenceRef:'http://example.com/'}),
    step('bad5','validation-run',{outcome:'passed',path:'a/b'}),
    step('bad6','validation-run',{outcome:'approved'}),
  ]) {
    assert.throws(()=>assessE28(audit,submission([event])),/invalid_|unexpected_/);
  }
  assert.throws(()=>assessE28(audit,submission([
    step('same','ai-question'),step('same','ai-question'),
  ])),/invalid_workflow_event/);
  assert.throws(()=>assessE28(audit,submission([
    {...step('foreign','human-decision'),choiceId:'sui.native-kit-target'},
  ])),/unqualified_workflow_choice/);
  assert.throws(()=>assessE28(audit,submission([],{scenario:{...scenario,baseRevision:'main'}})),
    /invalid_workflow_submission/);
});

test('E28 stale source observation fails closed without affecting unrelated choice',async()=>{
  const audit=await observeE27(root);
  const tampered={...audit,choices:audit.choices.map(choice=>
    choice.id==='web.ui-kit-target-adoption'
      ? {...choice,status:'source-unresolved',missing:['order.web-endpoint']} : choice)};
  assert.throws(()=>assessE28(tampered,submission([
    step('one','manual-connection'),
  ])),/unqualified_workflow_choice/);
  assert.doesNotThrow(()=>assessE28(tampered,submission([
    {...step('unrelated','human-decision'),choiceId:'web.database-binding'},
  ])));
});

test('E28 paired options give reported deltas, never a verified productivity gain',async()=>{
  const audit=await observeE27(root);
  const one=assessE28(audit,submission([step('a','manual-connection')],{complete:true}));
  const profileScenario={...scenario,arm:'profile-candidate'};
  const two=assessE28(audit,submission([
    step('a','manual-connection'),step('b','manual-connection'),
    step('c','ai-question'),
  ],{scenario:profileScenario,complete:true}));
  const comparison=compareE28(one,two);
  assert.equal(comparison.status,'reported-only-not-verified');
  assert.equal(comparison.reportedDelta.manualConnections,1);
  assert.equal(comparison.reportedDelta.aiQuestions,1);
  assert.equal(comparison.verifiedImprovement,null);
  assert.deepEqual(compareE28(one,assessE28(audit,submission([],{
    scenario:{...profileScenario,target:'native'},complete:true,
  }))),{status:'not-comparable',reportedDelta:null,verifiedImprovement:null});
});

test('E28 no unverified report is promoted into Runtime permission or SUI acceptance',async()=>{
  const audit=await observeE27(root);
  const candidate=assessE28(audit,submission([
    {...step('sui','validation-run',{outcome:'passed'}),
      choiceId:'sui.authorized-backend-commit'},
  ],{scenario:{...scenario,consumer:'sui_stage0',target:'native'},complete:true}));
  assert.equal(candidate.sourceSummary.stage0ObservedOnly,1);
  assert.equal(candidate.runtimeAuthorityVerified,false);
  assert.equal(candidate.compatibilityVerified,false);
  assert.equal(candidate.verifiedCounts,null);
});

test('E29 partial submissions cannot produce a reported productivity delta even with identical source baselines',async()=>{
  const audit=await observeE27(root);
  const profile={...scenario,arm:'profile-candidate'};
  const partialExisting=assessE28(audit,submission([
    step('a','manual-connection'),
  ]));
  const partialProfile=assessE28(audit,submission([
    step('b','manual-connection'),step('c','ai-question'),
  ],{scenario:profile}));
  const completeExisting=assessE28(audit,submission([
    step('d','manual-connection'),
  ],{complete:true}));
  const completeProfile=assessE28(audit,submission([
    step('e','manual-connection'),step('f','ai-question'),
  ],{scenario:profile,complete:true}));
  const notComparable={status:'not-comparable',reportedDelta:null,verifiedImprovement:null};
  assert.deepEqual(compareE28(partialExisting,partialProfile),notComparable);
  assert.deepEqual(compareE28(partialExisting,completeProfile),notComparable);
  assert.deepEqual(compareE28(completeExisting,partialProfile),notComparable);
  assert.equal(compareE28(completeExisting,completeProfile).status,
    'reported-only-not-verified');
  assert.equal(completeExisting.verifiedCounts,null);
  assert.equal(completeProfile.verifiedCounts,null);
});
