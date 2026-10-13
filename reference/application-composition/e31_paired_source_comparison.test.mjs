import test from 'node:test';
import assert from 'node:assert/strict';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {observeE27} from './e27_residual_selection_audit.mjs';
import {assessE28,compareE28} from './e28_workflow_measurement.mjs';
import {compareE31PairedSource} from './e31_paired_source_comparison.mjs';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'../..');
const tei='d3035976455e165ff7fc566c9c12429070fcca2f';
const suiOld='72368e7c1822fa894e2323a38f207117a521d2ed';
const suiNew='4bd493f9942890d52117461886207d628f3554e2';
const ref='https://github.com/hat47x/tei-platform/issues/1016';
const submit=(audit,{arm='existing-source',consumer='sui_stage0',
  complete=true,baseRevision=tei,target='native'}={})=>{
  const scenario={id:'same-work-request',consumer,arm,baseRevision,target};
  const event={id:'review1',operation:'human-decision',
    choiceId:consumer==='sui_stage0'?'sui.native-kit-target':'web.ui-kit-target-adoption',
    evidenceRef:ref};
  return assessE28(audit,{kind:'tei-e28-workflow-submission-v0',complete,
    scenario,events:[event]});
};
const pair=(existingSui,candidateSui,teiHash=tei)=>({
  existing:{tei:teiHash,sui:existingSui},candidate:{tei:teiHash,sui:candidateSui},
});

test('E31 SUI revision change cannot masquerade as same-work baseline',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{});
  const candidate=submit(audit,{arm:'profile-candidate'});
  // Current E28 has only one baseRevision and cannot express the SUI change.
  assert.equal(compareE28(existing,candidate).status,'reported-only-not-verified');
  const checked=compareE31PairedSource({existing,candidate,
    claimedRevisions:pair(suiOld,suiNew)});
  assert.deepEqual(checked,{
    status:'not-comparable',reason:'missing-or-mismatched-sui-revision',
    reportedDelta:null,verifiedImprovement:null,
  });
});

test('E31 same two claimed revisions permit only a reported comparison',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{});
  const candidate=submit(audit,{arm:'profile-candidate'});
  const checked=compareE31PairedSource({existing,candidate,
    claimedRevisions:pair(suiNew,suiNew)});
  assert.equal(checked.status,'reported-paired-source-only');
  assert.equal(checked.e28Status,'reported-only-not-verified');
  assert.equal(checked.reportedDelta.humanDecisions,0);
  assert.equal(checked.sourceRevisionsVerified,false);
  assert.equal(checked.runtimeAuthorityVerified,false);
  assert.equal(checked.verifiedImprovement,null);
});

test('E31 unknown SUI revision is not inferred from equal TEI hashes',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{});
  const candidate=submit(audit,{arm:'profile-candidate'});
  for(const revisions of [
    {existing:{tei},candidate:{tei}},
    {existing:{tei,sui:suiOld},candidate:{tei}},
    {existing:{tei,sui:'main'},candidate:{tei,sui:suiOld}},
    {existing:{tei,sui:suiOld},candidate:{tei,sui:''}},
  ]) {
    const checked=compareE31PairedSource({existing,candidate,claimedRevisions:revisions});
    assert.equal(checked.status,'not-comparable');
    assert.equal(checked.reason,'missing-or-mismatched-sui-revision');
  }
});

test('E31 TEI revision declaration must match each E28 source baseline',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{});
  const candidate=submit(audit,{arm:'profile-candidate'});
  const wrong='a'.repeat(40);
  const checked=compareE31PairedSource({existing,candidate,
    claimedRevisions:pair(suiOld,suiOld,wrong)});
  assert.equal(checked.status,'not-comparable');
  assert.equal(checked.reason,'missing-or-mismatched-tei-revision');
});

test('E31 E29 incomplete workflow cannot gain qualification through matching hashes',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{complete:false});
  const candidate=submit(audit,{arm:'profile-candidate'});
  const checked=compareE31PairedSource({existing,candidate,
    claimedRevisions:pair(suiOld,suiOld)});
  assert.equal(checked.status,'not-comparable');
  assert.equal(checked.reason,'e28-workflow-comparison-refused');
});

test('E31 existing Web consumer does not require a fictional SUI dependency',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{consumer:'order_entry',target:'web'});
  const candidate=submit(audit,{consumer:'order_entry',target:'web',
    arm:'profile-candidate'});
  const checked=compareE31PairedSource({existing,candidate,
    claimedRevisions:{existing:{tei},candidate:{tei}}});
  assert.equal(checked.status,'reported-paired-source-only');
  assert.equal(checked.sourceRevisionsVerified,false);
  const mismatched=compareE31PairedSource({existing,candidate,
    claimedRevisions:pair(suiOld,suiNew)});
  assert.equal(mismatched.reason,'missing-or-mismatched-sui-revision');
});

test('E31 cannot turn a different consumer or an invalid revision into a comparison',async()=>{
  const audit=await observeE27(root);
  const existing=submit(audit,{consumer:'sui_stage0'});
  const other=submit(audit,{consumer:'order_entry',target:'web',arm:'profile-candidate'});
  assert.equal(compareE31PairedSource({existing,candidate:other,
    claimedRevisions:pair(suiOld,suiOld)}).reason,'unknown-or-different-consumers');
  assert.equal(compareE31PairedSource({existing,candidate:submit(audit,{arm:'profile-candidate'}),
    claimedRevisions:{existing:{tei:'main',sui:suiOld},candidate:{tei,sui:suiOld}}})
    .reason,'missing-or-mismatched-tei-revision');
});
