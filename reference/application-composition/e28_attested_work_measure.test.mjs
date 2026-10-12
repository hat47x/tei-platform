import test from 'node:test';
import assert from 'node:assert/strict';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {observeE27} from './e27_residual_selection_audit.mjs';
import {measureE28Work,compareE28Work} from './e28_attested_work_measure.mjs';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'../..');
const tei='a2d92df59a802901274ec00302d09b81575a3422';
const sui='96b0d1d476527fdb9576b7c85ea572a4042eb74f';
const cover={
  manualActions:'complete',aiQuestions:'complete',
  sourceChanges:'complete',verifications:'complete',
};
const session=(change={})=>({
  id:'e28-example',consumer:'order_entry',approach:'existing-sources',
  requirementId:'order-custom-help',target:'web',baseline:{tei},
  mode:'illustrative',state:'closed',coverage:{...cover},...change,
});
const event=(id,kind,fields={})=>({
  id,sessionId:'e28-example',kind,observedAt:'2026-10-12T23:16:00Z',choiceId:'web.project-ui-extension',
  provenance:{kind:'human-attested',ref:'issue:1016'},...fields,
});
const records=[
  event('human1','manual-action'),
  event('question1','ai-question'),
  event('change1','source-change',{file:{repository:'hat47x/tei-platform',
    path:'reference/ui-customization/web-react/project-extension.mjs',
    before:'a'.repeat(40),after:'b'.repeat(40)}}),
  event('change2','source-change',{file:{repository:'hat47x/tei-platform',
    path:'reference/ui-customization/web-react/project-extension.mjs',
    before:'b'.repeat(40),after:'c'.repeat(40)}}),
  event('check1','verification',{outcome:'fail'}),
  event('check2','verification',{outcome:'pass'}),
];

test('E28 reads real E27 choices but an illustrative trace is NEVER a measured adoption result',async()=>{
  const audit=await observeE27(root);
  assert.equal(audit.choices.length,10);
  const report=measureE28Work({session:session(),events:records,audit});
  assert.equal(report.mode,'illustrative');
  assert.deepEqual(report.eventCounts,{'manual-action':1,'ai-question':1,
    'source-change':2,verification:2});
  assert.deepEqual(report.measurements,{
    manualActions:null,aiQuestions:null,changedFiles:null,verificationRuns:null});
  assert.equal(report.runtimeAuthority,null);
  assert.equal(report.realTargetQualified,null);
  assert.equal(report.acceptedBusinessMeaning,null);
});

test('E28 a partially tracked real session keeps UNKNOWN dimensions null, not zero',async()=>{
  const audit=await observeE27(root);
  const report=measureE28Work({
    session:session({mode:'caller-recorded',coverage:{
      ...cover,aiQuestions:'untracked',sourceChanges:'untracked'}}),
    events:[records[0],records[4]],audit,
  });
  assert.deepEqual(report.measurements,{
    manualActions:1,aiQuestions:null,changedFiles:null,verificationRuns:1});
  assert.deepEqual(report.verificationOutcomes,{pass:0,fail:1,unknown:0});
  assert.equal(report.runtimeAuthority,null);
  const open=measureE28Work({session:session({mode:'caller-recorded',state:'open'}),
    events:records,audit});
  assert.equal(open.measurements.manualActions,null);
  assert.equal(open.measurements.aiQuestions,null);
  assert.equal(open.measurements.changedFiles,null);
});

test('E28 a completed and covered caller log counts distinct files, not edit attempts',async()=>{
  const audit=await observeE27(root);
  const report=measureE28Work({session:session({mode:'caller-recorded'}),
    events:records,audit});
  assert.deepEqual(report.measurements,{
    manualActions:1,aiQuestions:1,changedFiles:1,verificationRuns:2});
  assert.deepEqual(report.verificationOutcomes,{pass:1,fail:1,unknown:0});
  assert.equal(report.source,'caller-supplied-events-not-independently-verified');
  assert.equal(report.acceptedBusinessMeaning,null);
});

test('E28 no event is not zero effort unless that dimension is explicitly covered and closed',async()=>{
  const audit=await observeE27(root);
  const partial=measureE28Work({session:session({mode:'caller-recorded',
    coverage:{...cover,aiQuestions:'untracked'}}),events:[],audit});
  assert.equal(partial.measurements.aiQuestions,null);
  assert.equal(partial.measurements.manualActions,0);
  const illustrative=measureE28Work({session:session(),events:[],audit});
  assert.equal(illustrative.measurements.manualActions,null);
});

test('E28 rejects duplicate, forged, cross-consumer, unsafe file and prompt-copy entries',async()=>{
  const audit=await observeE27(root);
  const base=session({mode:'caller-recorded'});
  const run=events=>measureE28Work({session:base,events,audit});
  assert.throws(()=>run([records[0],records[0]]),/duplicate_work_event/);
  assert.throws(()=>run([event('crossSession','manual-action',{
    sessionId:'other-session'})]),/invalid_work_event/);
  assert.throws(()=>run([event('foreign','manual-action',{choiceId:'sui.native-kit-target'})]),
    /invalid_work_event/);
  assert.throws(()=>run([event('injected','ai-question',{text:'copied private prompt'})]),
    /unexpected_work_event_field/);
  assert.throws(()=>run([event('badSHA','source-change',{file:{
    repository:'hat47x/tei-platform',path:'source/file.tei',
    before:'a'.repeat(40),after:'not-a-blob'}})]),/invalid_file_change/);
  assert.throws(()=>run([event('traversal','source-change',{file:{
    repository:'hat47x/tei-platform',path:'../secrets',
    before:'a'.repeat(40),after:'b'.repeat(40)}})]),/invalid_file_change/);
  assert.throws(()=>run([event('noChange','source-change',{file:{
    repository:'hat47x/tei-platform',path:'valid/path',
    before:'a'.repeat(40),after:'a'.repeat(40)}})]),/invalid_file_change/);
  assert.throws(()=>run([event('badTime','manual-action',{observedAt:'2026-02-30T23:16:00Z'})]),
    /invalid_work_event/);
  assert.throws(()=>measureE28Work({session:session({baseline:{tei:'branch-main'}}),
    events:[],audit}),/invalid_session/);
  assert.throws(()=>measureE28Work({session:session({consumer:'sui_stage0'}),
    events:[],audit}),/missing_sui_baseline/);
  assert.throws(()=>measureE28Work({session:base,events:[],audit:{...audit,kind:'unknown'}}),
    /invalid_audit_or_events/);
});

test('E28 compares only the same consumer, Target, requirement and source baselines',async()=>{
  const audit=await observeE27(root);
  const left=measureE28Work({session:session({mode:'caller-recorded'}),
    events:records,audit});
  const right=measureE28Work({
    session:session({id:'e28-with-profile',mode:'caller-recorded',
      approach:'profile-candidate'}),
    events:[{...records[0],sessionId:'e28-with-profile'},
      {...records[4],sessionId:'e28-with-profile'}],audit,
  });
  assert.deepEqual(compareE28Work(left,right),{
    status:'caller-reported-comparison-only',
    differences:{manualActions:0,aiQuestions:-1,changedFiles:-1,verificationRuns:-1},
    productAdoptionQualified:false,
    reason:'real-target-and-authority-require-independent-evidence',
  });
  const newBase={...right,baseline:{tei:'f'.repeat(40)}};
  assert.deepEqual(compareE28Work(left,newBase),{
    status:'not-comparable',reason:'different-task-baseline-or-approach'});
  const absentAI={...right,measurements:{...right.measurements,aiQuestions:null}};
  assert.deepEqual(compareE28Work(left,absentAI),{
    status:'not-comparable',reason:'incomplete-attested-coverage'});
});

test('E28 Stage0 can only record its own choices, never graduate to an accepted SUI App',async()=>{
  const audit=await observeE27(root);
  const info=session({id:'e28-sui',consumer:'sui_stage0',target:'native',
    mode:'caller-recorded',baseline:{tei,sui}});
  const entry=event('suiChoice','manual-action',{sessionId:'e28-sui',
    choiceId:'sui.native-kit-target'});
  const report=measureE28Work({session:info,events:[entry],audit});
  assert.equal(report.measurements.manualActions,1);
  assert.equal(report.runtimeAuthority,null);
  assert.equal(report.acceptedBusinessMeaning,null);
});
