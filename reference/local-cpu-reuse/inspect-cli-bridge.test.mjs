import test from 'node:test';
import assert from 'node:assert/strict';
import {inspectSubjectBatch,projectCliInspectReport,InspectBridgeError} from './inspect-cli-bridge.mjs';

const selector=(kind,id)=>({kind,id});
const a=selector('application','order_entry');
const b=selector('application-pattern','order_entry.order_lines');
function report(s=a,canonical) {
  const subject={...s};
  const model=canonical ?? (s.kind==='application'?
    {id:s.id,patterns:[{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'}]} :
    {id:s.id,kind:'header-detail',relation:'sales.order.lines'});
  return {schemaVersion:'tei.cli.inspect/v0',ok:true,
    result:{schemaVersion:'tei.operations.inspect/v0',effect:'read-only',subject,
      canonical:model,worldFingerprint:'world:one',sourceIndexFingerprint:'source:one',
      sourceStatus:'indexed',
      sources:{subject,artifacts:[{path:'apps/order_entry.tei',fingerprint:'sha256:example'}],
        references:[{subject,artifactPath:'apps/order_entry.tei',role:'definition'}]}}};
}
const code=expected=>e=>e instanceof InspectBridgeError && e.code===expected;

test('existing TEI CLI report success unwraps into non-authoritative feature projection',()=>{
  const p=projectCliInspectReport(report(a),a);
  assert.equal(p.subject.id,'order_entry');
  assert.equal(p.features.coverage,'membership_only');
  assert.equal(p.acceptance,'not_asserted');
  assert.equal(p.authority,'derived_observation_only');
});
test('CLI error response with no result is rejected; never treated as missing semantics',()=>{
  const failed={schemaVersion:'tei.cli.inspect/v0',ok:false,error:{code:'SUBJECT_NOT_FOUND',message:'untrusted detail'}};
  assert.throws(()=>projectCliInspectReport(failed,a),code('INVALID_OR_FAILED_CLI_INSPECTION'));
});
test('a reported success with a conflicting error field fails closed',()=>{
  const bad=report(); bad.error={code:'DUPLICATE'};
  assert.throws(()=>projectCliInspectReport(bad,a),code('INVALID_OR_FAILED_CLI_INSPECTION'));
});
test('wrong subject in a valid CLI envelope cannot populate requested slot',()=>{
  assert.throws(()=>projectCliInspectReport(report(a),b),code('WRONG_SUBJECT_RETURNED'));
});
test('unknown CLI schema is rejected, even when embedded result is readable',()=>{
  const bad=report();bad.schemaVersion='tei.cli.inspect/v1';
  assert.throws(()=>projectCliInspectReport(bad,a),code('INVALID_OR_FAILED_CLI_INSPECTION'));
});
test('result projection error is normalized to bounded bridge code',()=>{
  const bad=report();bad.result.sources.references[0].artifactPath='missing-artifact';
  assert.throws(()=>projectCliInspectReport(bad,a),code('UNUSABLE_INSPECTION_RESULT'));
});
test('supported kind must be exact; do not accept guessed names',()=>{
  assert.throws(()=>projectCliInspectReport(report(),selector('semantic-capability','sales.order.submit')),
    code('INVALID_SUBJECT_SELECTOR'));
});
test('batch preserves exact order and one consistent observed workspace snapshot',async()=>{
  const all=await inspectSubjectBatch([a,b],async s=>report(s));
  assert.deepEqual(all.observations.map(x=>x.subject.id),['order_entry','order_entry.order_lines']);
  assert.equal(all.acceptance,'not_asserted');
  assert.equal(all.worldFingerprint,'world:one');
  assert.equal(all.sourceIndexFingerprint,'source:one');
});
test('batch rejects duplicate subjects before invoking external read',async()=>{
  let calls=0;
  await assert.rejects(()=>inspectSubjectBatch([a,a],async()=>{calls++;return report()}),code('DUPLICATE_SUBJECT'));
  assert.equal(calls,0);
});
test('batch detects drift in World fingerprint and returns no partial success',async()=>{
  const read=async s=>{const r=report(s);if(s.id===b.id)r.result.worldFingerprint='world:two';return r;};
  await assert.rejects(()=>inspectSubjectBatch([a,b],read),code('SNAPSHOT_DRIFT'));
});
test('batch detects drift in Source Index even when World meaning is unchanged',async()=>{
  const read=async s=>{const r=report(s);if(s.id===b.id)r.result.sourceIndexFingerprint='source:changed';return r;};
  await assert.rejects(()=>inspectSubjectBatch([a,b],read),code('SNAPSHOT_DRIFT'));
});
test('read error aborts batch without treating failure as unknown candidate',async()=>{
  await assert.rejects(()=>inspectSubjectBatch([a,b],async s=>{
    if(s.id===b.id)throw new InspectBridgeError('CLI_EXECUTION_FAILED');
    return report(s);
  }),code('CLI_EXECUTION_FAILED'));
});
