import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {reviewE34BridgeDelta} from './e34_bridge_revision_review.mjs';

const previous=JSON.parse(await readFile(
  new URL('./e32_project_plugin_pr1567_snapshot.json',import.meta.url),'utf8'));
const delta=JSON.parse(await readFile(
  new URL('./e34_project_bridge_pr1567_delta.json',import.meta.url),'utf8'));
const clone=x=>structuredClone(x);

test('E34 pinned real #1567 bridge revision changes precisely 4 paths',()=>{
  const report=reviewE34BridgeDelta(previous,delta);
  assert.deepEqual(report.pinned,{
    repository:'hat47x/tei-platform',pullRequest:1567,
    before:'be193266f1556e090b59cf2c237a23c0ca60236e',
    after:'596c522489dde0ad5184eba6e0220cd758f16636',
  });
  assert.deepEqual(report.observed,{
    previousFiles:20,changedPaths:4,addedFiles:2,modifiedPriorFiles:2,
    priorUnmodifiedPaths:18,additions:161,deletions:2,
    netAddedLines:159,totalNewFilesInDraft:22,
  });
});

test('E34 changed old Host and README blobs must match E32 exactly',()=>{
  const tampered=clone(delta);
  tampered.files.find(f=>f.path.endsWith('/main.go')).beforeBlob='a'.repeat(40);
  assert.throws(()=>reviewE34BridgeDelta(previous,tampered),/invalid_e34_prior_source/);
  const unbound=clone(delta);
  unbound.previousHead='a'.repeat(40);
  assert.throws(()=>reviewE34BridgeDelta(previous,unbound),/invalid_e34_revision_footprint/);
  const identical=clone(delta);
  identical.files.find(f=>f.path.endsWith('/main.go')).afterBlob=
    identical.files.find(f=>f.path.endsWith('/main.go')).beforeBlob;
  assert.throws(()=>reviewE34BridgeDelta(previous,identical),/invalid_e34_prior_source/);
});

test('E34 new bridge and bridge test must be new sources, not duplicate code',()=>{
  const bridge=clone(delta);
  bridge.files.find(f=>f.path.endsWith('/contract_bridge.go')).beforeBlob='b'.repeat(40);
  assert.throws(()=>reviewE34BridgeDelta(previous,bridge),/invalid_e34_added_source/);
  const duplicate=clone(delta);
  duplicate.files[1]=clone(duplicate.files[0]);
  assert.throws(()=>reviewE34BridgeDelta(previous,duplicate),/invalid_e34_changed_file/);
  const additional=clone(delta);
  additional.files[0].path='internal/composition/registry.go';
  assert.throws(()=>reviewE34BridgeDelta(previous,additional),/invalid_e34_changed_file/);
  const omitted=clone(delta);
  omitted.files.pop();
  assert.throws(()=>reviewE34BridgeDelta(previous,omitted),/invalid_e34_revision_footprint/);
});

test('E34 invalid SHAs, negative differences, and illegal role cannot pass',()=>{
  const badSha=clone(delta);badSha.files[1].afterBlob='main';
  assert.throws(()=>reviewE34BridgeDelta(previous,badSha),/invalid_e34_changed_file/);
  const negative=clone(delta);negative.files[1].additions=-1;
  assert.throws(()=>reviewE34BridgeDelta(previous,negative),/invalid_e34_changed_file/);
  const wrongRole=clone(delta);wrongRole.files[1].role='runtime-provider';
  assert.throws(()=>reviewE34BridgeDelta(previous,wrongRole),/invalid_e34_changed_file/);
  const forgedOld=clone(previous);forgedOld.files[0].gitBlobSha='branch';
  assert.throws(()=>reviewE34BridgeDelta(forgedOld,delta),/invalid_e32_source_snapshot/);
});

test('E34 source candidate and succeeded Chrome never imply runtime authority',()=>{
  const report=reviewE34BridgeDelta(previous,delta);
  assert.equal(report.bridgeSourcePresent,true);
  for (const key of ['providerContractAccepted','artifactDigestVerified',
    'hostCurrentPermissionVerified','actionAcceptedMeaningVerified',
    'productionDeploymentVerified','humanEffortMeasured','aiQuestionsMeasured']) {
    assert.equal(report[key],null,key);
  }
  assert.equal(report.evidence,'fixed-github-source-diff-metadata-not-retrieved-live');
});

test('E34 PR history is incremental: new bridge source is not all handwritten work',()=>{
  const report=reviewE34BridgeDelta(previous,delta);
  assert.equal(report.observed.netAddedLines,159);
  // Initial E32 PR had 1,267 lines added against its base; the pinned
  // change since that HEAD is +159 net. Neither is a productivity figure.
  assert.equal(1267+report.observed.netAddedLines,1426);
  assert.equal(report.humanEffortMeasured,null);
});
