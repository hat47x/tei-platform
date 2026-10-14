import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {observeE27} from './e27_residual_selection_audit.mjs';
import {submitE38SourceOnly} from './e38_real_workflow_gap.mjs';
const audit=await observeE27('.');
const delta=JSON.parse(await readFile(new URL('./e34_project_bridge_pr1567_delta.json',import.meta.url),'utf8'));
const clone=()=>structuredClone(delta);

test('E38 real E34 changed paths produce only two qualified E28 modified-source events',()=>{
 const result=submitE38SourceOnly(audit,delta);
 assert.equal(result.kind,'tei-e38-real-source-workflow-gap-v0');
 assert.equal(result.modifiedSourceRecords,2);
 assert.equal(result.newlyAddedSourcePaths,2);
 assert.equal(result.sourceEventsSubmittedToE28,2);
 assert.equal(result.e28Scope,'partial-not-verified');
 assert.equal(result.claimedCoveredCounts.changedFiles,null);
 assert.equal(result.verifiedCounts,null);
});
test('E38 source edits do not imply a human decision, AI question, connection or benefit',()=>{
 const result=submitE38SourceOnly(audit,delta);
 for(const key of ['actualHumanDecisions','actualAIQuestions',
 'actualManualConnections','improvementVerified'])assert.equal(result[key],null);
 assert.equal(result.runtimeAuthorityVerified,false);
});
test('E38 refuses missing, duplicated, forged or added-as-edited files',()=>{
 const duplicate=clone();duplicate.files[1]=structuredClone(duplicate.files[0]);
 assert.throws(()=>submitE38SourceOnly(audit,duplicate),/invalid_e38_diff/);
 const omitted=clone();omitted.files.pop();
 assert.throws(()=>submitE38SourceOnly(audit,omitted),/invalid_e38_source/);
 const forged=clone();forged.files[1].path='internal/composition/registry.go';
 assert.throws(()=>submitE38SourceOnly(audit,forged),/invalid_e38_diff_path/);
 const added=clone();added.files.find(f=>f.status==='added').beforeBlob='a'.repeat(40);
 assert.throws(()=>submitE38SourceOnly(audit,added),/invalid_e38_addition/);
});
test('E38 does not permit fake before/after equality or wrong repositories',()=>{
 const same=clone();const edit=same.files.find(f=>f.status==='modified');
 edit.beforeBlob=edit.afterBlob;
 assert.throws(()=>submitE38SourceOnly(audit,same),/invalid_e38_prior_blob/);
 const wrong=clone();wrong.repository='hat47x/not-tei';
 assert.throws(()=>submitE38SourceOnly(audit,wrong),/invalid_e38_source/);
});
test('E38 cannot derive a total changed file count or verified improvement',()=>{
 const report=submitE38SourceOnly(audit,delta);
 assert.equal(report.modifiedSourceRecords,2);
 assert.equal(report.newlyAddedSourcePaths,2);
 assert.equal(report.claimedCoveredCounts.changedFiles,null);
 assert.equal(report.improvementVerified,null);
});
