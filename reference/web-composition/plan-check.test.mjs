import assert from 'node:assert/strict';
import test from 'node:test';
import {composeRequestPlan, validateAuthenticationPlan} from './plan-check.mjs';

const clone = value => structuredClone(value);
const steps = [
  {id:'context',scope:'server',provides:['request.context']},
  {id:'identity',scope:'server',requires:['request.context'],provides:['identity.principal'],guards:['authenticated']},
  {id:'csrf',scope:'server',requires:['request.context'],provides:['csrf.checked'],guards:['csrf-protected']},
  {id:'authorize',scope:'server',requires:['identity.principal','action.identity'],provides:['action.authorized'],guards:['action-authorized'],before:['invoke']},
  {id:'invoke',scope:'server',requires:['action.authorized','csrf.checked']},
  {id:'audit',scope:'server',requires:['request.context'],onTerminal:['completed','rejected','error']}
];
const base = {
  route:{id:'sales.submit',method:'POST',path:'/scenario/submit',exposure:'protected',actionId:'order_entry.submit_order'},
  hostPolicy:{id:'interactive-write',requiredGuards:['authenticated','csrf-protected','action-authorized'],terminalEvents:['completed','rejected','error']},
  hostContracts:['action.identity'],
  steps
};
const authentication = {
  providers:[{id:'corporate',kind:'oidc',issuer:'https://example.invalid/issuer'},{id:'legacy',kind:'legacy',issuer:'legacy:order_entry'}],
  policies:[{id:'employee',allowedProviders:['corporate','legacy'],fallbackOnFailure:false}],
  subjectLinks:[
    {providerId:'corporate',issuer:'https://example.invalid/issuer',subject:'fixed-sub-001',principalId:'employee-001'},
    {providerId:'legacy',issuer:'legacy:order_entry',subject:'account-001',principalId:'employee-001'}
  ]
};

test('protected request orders authorization before action', () => {
  const r=composeRequestPlan(base);
  assert.ok(r.order.indexOf('authorize')<r.order.indexOf('invoke'));
});
test('authorization omitted', () => {
  const p=clone(base);p.steps=p.steps.filter(step=>step.id!=='authorize');
  assert.throws(()=>composeRequestPlan(p),/protection missing/);
});
test('CSRF omitted', () => {
  const p=clone(base);p.steps=p.steps.filter(step=>step.id!=='csrf');
  assert.throws(()=>composeRequestPlan(p),/protection missing/);
});
test('browser step is not server enforcement', () => {
  const p=clone(base);p.steps.find(step=>step.id==='identity').scope='browser';
  assert.throws(()=>composeRequestPlan(p),/scope/);
});
test('early denial and error require terminal coverage', () => {
  const p=clone(base);p.steps.find(step=>step.id==='audit').onTerminal=['completed'];
  assert.throws(()=>composeRequestPlan(p),/terminal coverage/);
});
test('public route must not invoke action implicitly', () => {
  const p=clone(base);p.route.exposure='public';
  assert.throws(()=>composeRequestPlan(p),/public/);
});
test('ordering cycle is rejected', () => {
  const p=clone(base);p.steps.find(step=>step.id==='context').after=['identity'];
  assert.throws(()=>composeRequestPlan(p),/cycle/);
});
test('exclusive middleware slot collision is rejected', () => {
  const p=clone(base);p.steps.find(step=>step.id==='csrf').exclusiveSlots=['session'];
  p.steps.find(step=>step.id==='identity').exclusiveSlots=['session'];
  assert.throws(()=>composeRequestPlan(p),/collision/);
});
test('explicit identity mappings may share the application principal', () => {
  assert.equal(validateAuthenticationPlan(authentication).linkedSubjectCount,2);
});
test('implicit fallback is rejected', () => {
  const p=clone(authentication);p.policies[0].fallbackOnFailure=true;
  assert.throws(()=>validateAuthenticationPlan(p),/fallback/);
});
test('email-only identity linking is rejected', () => {
  const p=clone(authentication);p.subjectLinks[0].subject='';p.subjectLinks[0].email='employee@example.invalid';
  assert.throws(()=>validateAuthenticationPlan(p),/subject linking/);
});
test('issuer mismatch is rejected', () => {
  const p=clone(authentication);p.subjectLinks[0].issuer='https://other.invalid';
  assert.throws(()=>validateAuthenticationPlan(p),/issuer/);
});

test('action cannot invoke without authorization dependency', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='invoke').requires=['csrf.checked'];
  assert.throws(()=>composeRequestPlan(p),/invocation must depend on authorization/);
});
test('host contracts must be unambiguous', () => {
  const p=clone(base);p.hostContracts.push('action.identity');
  assert.throws(()=>composeRequestPlan(p),/duplicate host contract/);
});
