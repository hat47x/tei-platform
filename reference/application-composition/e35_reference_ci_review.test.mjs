import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {reviewE35ReferenceCI} from './e35_reference_ci_review.mjs';

const fixture=JSON.parse(await readFile(
  new URL('./e35_project_reference_ci_evidence.json',import.meta.url),'utf8'));
const clone=()=>structuredClone(fixture);
const review=input=>reviewE35ReferenceCI(input);

test('E35 real pinned Go/Node, real Chrome and PR health runs all reported success',()=>{
  const result=review(fixture);
  assert.deepEqual(result.pinned,{
    repository:'hat47x/tei-platform',pr:1567,
    head:'5fde80b0385b471f1c15d701f3c9f61761405c27',observedDraft:true});
  assert.deepEqual(result.observed,{sourceBlobs:6,workflowRuns:3,criticalSteps:7});
  assert.equal(result.status,'pinned-reference-ci-reported-success');
  assert.deepEqual(result.unresolved,[]);
  assert.equal(result.independentEvidenceAuthentication,false);
});

test('E35 previous failure cannot be overwritten by a success from a different revision',()=>{
  const newer=reviewE35ReferenceCI(fixture,{requestedHead:'a'.repeat(40)});
  assert.equal(newer.status,'not-supported-at-requested-head');
  assert.ok(newer.unresolved.includes('requested-revision-not-checked'));
  const oldRun=clone();
  oldRun.workflowRuns[1].headSha='b'.repeat(40);
  const report=review(oldRun);
  assert.equal(report.status,'not-supported-at-requested-head');
  assert.deepEqual(report.unresolved,['real-chrome-reference:different-head']);
});

test('E35 failure, pending or cancelled smoke is not silently promoted by green Chrome',()=>{
  for(const outcome of ['failure','pending','skipped']){
    const altered=clone();
    altered.workflowRuns[0].stepConclusions[
      '実order_entry APIからの読み取り専用接続を検証']=outcome;
    const report=review(altered);
    assert.equal(report.status,'not-supported-at-requested-head');
    assert.ok(report.unresolved.includes('go-node-readonly-integration:not-proven-success'));
  }
  const queued=clone();queued.workflowRuns[0].status='in_progress';
  queued.workflowRuns[0].conclusion=null;
  queued.workflowRuns[0].jobConclusion=null;
  assert.equal(review(queued).status,'not-supported-at-requested-head');
});

test('E35 unknown Git blobs, duplicate paths or missing source paths are not evidence',()=>{
  const forged=clone();
  forged.files[0].blobSha='c'.repeat(40);
  assert.throws(()=>review(forged),/invalid_or_duplicate_source_blob/);
  const dup=clone();
  dup.files[1]=structuredClone(dup.files[0]);
  assert.throws(()=>review(dup),/invalid_or_duplicate_source_blob/);
  const missing=clone();missing.files.pop();
  assert.throws(()=>review(missing),/invalid_e35_snapshot/);
  const unrelated=clone();unrelated.files[0].path='internal/composition/registry.go';
  assert.throws(()=>review(unrelated),/invalid_or_duplicate_source_blob/);
});

test('E35 forged, duplicated or omitted CI runs cannot be counted',()=>{
  const duplicate=clone();
  duplicate.workflowRuns[1].runId=duplicate.workflowRuns[0].runId;
  assert.throws(()=>review(duplicate),/invalid_or_duplicate_ci_evidence/);
  const forged=clone();
  forged.workflowRuns[1].workflow='Fake Host Production Approval';
  assert.throws(()=>review(forged),/invalid_or_duplicate_ci_evidence/);
  const extra=clone();extra.workflowRuns[1].criticalSteps.push('production signed');
  assert.throws(()=>review(extra),/invalid_or_duplicate_ci_evidence/);
  const omitted=clone();omitted.workflowRuns.pop();
  assert.throws(()=>review(omitted),/invalid_e35_snapshot/);
});

test('E35 passing all diagnostic checks leaves every product and labor authority unverified',()=>{
  const result=review(fixture);
  for(const key of ['publishedServiceContract','conformanceVerified',
    'executableArtifactVerified','currentHostPermissionVerified',
    'approvedActionMeaningVerified','productionDeploymentVerified',
    'manualActionsMeasured','aiQuestionsMeasured']) {
    assert.equal(result[key],null,key);
  }
  assert.equal(result.provenance,
    'github-source-and-ci-metadata-pinned-not-reverified-at-runtime');
});

test('E35 cannot accept a real PR with an unknown, malformed or different head',()=>{
  const invalid=clone();invalid.headSha='main';
  assert.throws(()=>review(invalid),/invalid_e35_snapshot/);
  const wrongPR=clone();wrongPR.pullRequest=1674;
  assert.throws(()=>review(wrongPR),/invalid_e35_snapshot/);
  const reversed=clone();reversed.wasDraft=false;
  assert.throws(()=>review(reversed),/invalid_e35_snapshot/);
  assert.equal(reviewE35ReferenceCI(fixture,{requestedHead:'f'.repeat(40)}).status,
    'not-supported-at-requested-head');
});
