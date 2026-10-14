import test from 'node:test';
import assert from 'node:assert/strict';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {observeSource, probes} from './e1-observe.mjs';
import {auditE27, observeE27} from './e27_residual_selection_audit.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const byID = report => new Map(report.choices.map(choice=>[choice.id,choice]));

test('E27 actual TEI sources show existing Web wiring but not accepted adoption or grants', async () => {
  const report = await observeE27(root);
  assert.deepEqual(report.summary, {
    scope:'bounded-e1-tei-source-only',
    sourceObserved:4,
    stage0ObservedOnly:1,
    selectionUnresolved:3,
    runtimeProofUnresolved:2,
    sourceUnresolved:0,
    actualManualActions:null,
    actualAIQuestions:null,
    acceptedProfiles:null,
  });
  assert.equal(report.choices.length,10);
  assert.equal(byID(report).get('web.database-binding').status,'source-observed');
  assert.equal(byID(report).get('web.provider-reference').status,'source-observed');
  assert.equal(byID(report).get('web.ui-kit-target-adoption').status,'selection-unresolved');
  assert.equal(byID(report).get('web.current-runtime-authority').status,'runtime-proof-unresolved');
  assert.equal(byID(report).get('sui.stage0-meaning').status,'stage0-observed-only');
  assert.equal(byID(report).get('sui.native-action-contract').status,'selection-unresolved');
  assert.equal(byID(report).get('sui.authorized-backend-commit').status,'runtime-proof-unresolved');
  assert.deepEqual(report.choices.filter(c=>c.status==='source-unresolved'),[]);
});

test('E27 one drifted Environment Source does not invalidate unrelated UI or SUI observations', async () => {
  const raw = await observeSource(root);
  const mutated = raw.observations.map(item=>item.id==='order.dev-binding'
    ? {...item,status:'unresolved',reason:'marker-missing'} : item);
  const result = auditE27(mutated);
  const choices = byID(result);
  assert.equal(choices.get('web.database-binding').status,'source-unresolved');
  assert.deepEqual(choices.get('web.database-binding').missing,['order.dev-binding']);
  assert.equal(choices.get('web.project-ui-extension').status,'source-observed');
  assert.equal(choices.get('sui.stage0-meaning').status,'stage0-observed-only');
  assert.equal(result.summary.sourceUnresolved,1);
});

test('E27 absent E1 evidence never implies a selected Kit or authorized Native commit', async () => {
  const raw = await observeSource(root);
  const stripped = raw.observations.filter(item=>item.id!=='sui.workspace-scope');
  const result = auditE27(stripped);
  const choices = byID(result);
  for(const id of ['sui.stage0-meaning','sui.native-action-contract',
    'sui.native-kit-target','sui.authorized-backend-commit']){
    assert.equal(choices.get(id).status,'source-unresolved');
    assert.deepEqual(choices.get(id).missing,['sui.workspace-scope']);
  }
  assert.equal(result.summary.sourceUnresolved,4);
  assert.equal(result.summary.actualAIQuestions,null);
});

test('E27 external source cannot impersonate an E1 observation', async () => {
  const {observations} = await observeSource(root);
  assert.equal(probes.length,14);
  assert.throws(() => auditE27([...observations,observations[0]]),/invalid_e1_observation/);
  assert.throws(() => auditE27([...observations,{...observations[0],id:'foreign.trusted'}]),
    /invalid_e1_observation/);
  assert.throws(() => auditE27(observations.map((item,i)=>i===0
    ? {...item,group:'sui_stage0'} : item)),/invalid_e1_observation/);
  assert.throws(() => auditE27(observations.map((item,i)=>i===0
    ? {...item,status:'observed',reason:'literal-present',sha256:'made-up'} : item)),
    /invalid_e1_observation/);
});

test('E27 all markers present still do not promote SUI Stage0 or Runtime authority', async () => {
  const {observations} = await observeSource(root);
  const report = auditE27(observations);
  const unresolved = report.choices.filter(c=>!['source-observed','stage0-observed-only'].includes(c.status));
  assert.equal(unresolved.length,5);
  assert.ok(unresolved.every(c=>['selection-unresolved','runtime-proof-unresolved'].includes(c.status)));
  assert.ok(report.choices.every(c=>c.owner && c.evidence.length>0));
  assert.equal(report.summary.acceptedProfiles,null);
});
