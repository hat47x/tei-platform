import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {summarizeE32PR} from './e32_plugin_pr_footprint.mjs';

const snapshot=JSON.parse(await readFile(
  new URL('./e32_project_plugin_pr1567_snapshot.json',import.meta.url),'utf8'));
const copy=()=>structuredClone(snapshot);

test('E32 pinned real PR #1567 metadata is 20 added files and 1,267 added lines',()=>{
  const report=summarizeE32PR(snapshot);
  assert.equal(report.snapshot.repository,'hat47x/tei-platform');
  assert.equal(report.snapshot.pr,1567);
  assert.equal(report.snapshot.headSha,'be193266f1556e090b59cf2c237a23c0ca60236e');
  assert.equal(report.snapshot.baseSha,'aa63f482b35e9d169f150aaaac801d56db288d72');
  assert.equal(report.snapshot.draftAtObservation,true);
  assert.equal(report.snapshot.mergedAtObservation,false);
  assert.deepEqual({
    files:report.observed.changedFiles,
    added:report.observed.filesAdded,
    preexistingChanged:report.observed.filesNotAdded,
    additions:report.observed.additions,
    deletions:report.observed.deletions,
  },{files:20,added:20,preexistingChanged:0,additions:1267,deletions:0});
});

test('E32 classify observed PR footprint, keeping bundled Lit distinct from handwritten sources',()=>{
  const report=summarizeE32PR(snapshot);
  assert.deepEqual(report.observed.byRole,{
    documentation:{files:2,addedLines:61},
    'project-logic':{files:2,addedLines:84},
    'diagnostic-host':{files:4,addedLines:423},
    'web-ui':{files:6,addedLines:251},
    'third-party':{files:2,addedLines:52},
    verification:{files:4,addedLines:396},
  });
  assert.equal(Object.values(report.observed.byRole).reduce((n,x)=>n+x.files,0),20);
  assert.equal(Object.values(report.observed.byRole).reduce((n,x)=>n+x.addedLines,0),1267);
  assert.equal(snapshot.files.find(f=>f.path.endsWith('/lit-core.min.js')).gitBlobSha,
    '30071d620d392394d56721f05fc4808e02e038e3');
});

test('E32 a GitHub file-diff observation never becomes measured human effort or adoption',()=>{
  const report=summarizeE32PR(snapshot);
  for(const key of [
    'actualHumanDecisions','actualAIQuestions','actualManualConnections',
    'additionalProfileBenefit','formalProviderSelectionProven',
    'acceptedBusinessMeaningProven','runtimeAuthorityProven','productionAdoptionProven',
  ]) assert.equal(report[key],null,key);
  assert.equal(report.verification,'caller-supplied-pinned-pr-metadata-not-live-refetched');
});

test('E32 duplicates and unsafe or misclassified paths are rejected instead of counted',()=>{
  const duplicate=copy();
  duplicate.files.push(structuredClone(duplicate.files[0]));
  assert.throws(()=>summarizeE32PR(duplicate),/invalid_e32_file_evidence/);
  for(const bad of ['../secrets','tools/../secrets','/absolute','tools\\bad.js']){
    const mutated=copy();
    mutated.files[0].path=bad;
    assert.throws(()=>summarizeE32PR(mutated),/invalid_e32_file_evidence/);
  }
  const misclassified=copy();
  misclassified.files.find(x=>x.path.endsWith('/advice.go')).role='third-party';
  assert.throws(()=>summarizeE32PR(misclassified),/invalid_e32_file_evidence/);
});

test('E32 invalid source identity, negative diff or unbacked status cannot count as source',()=>{
  const badHash=copy();badHash.files[0].gitBlobSha='main';
  assert.throws(()=>summarizeE32PR(badHash),/invalid_e32_file_evidence/);
  const negative=copy();negative.files[0].additions=-2;
  assert.throws(()=>summarizeE32PR(negative),/invalid_e32_file_evidence/);
  const fakeAddition=copy();fakeAddition.files[0].deletions=3;
  assert.throws(()=>summarizeE32PR(fakeAddition),/invalid_e32_file_evidence/);
  const wrongRevision=copy();wrongRevision.headSha='branch';
  assert.throws(()=>summarizeE32PR(wrongRevision),/invalid_e32_snapshot/);
});

test('E32 changing the claimed GitHub diff size changes only observed source, not work counts',()=>{
  const increased=copy();increased.files[0].additions+=1;
  const report=summarizeE32PR(increased);
  assert.equal(report.observed.additions,1268);
  assert.equal(report.actualAIQuestions,null);
  assert.equal(report.productionAdoptionProven,null);
});
