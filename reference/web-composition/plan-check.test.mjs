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

test('guard declaration in audit does not protect Action', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='identity').guards=[];
  p.steps.find(step=>step.id==='audit').guards=['authenticated'];
  assert.throws(()=>composeRequestPlan(p),/prerequisite of Action/);
});
test('guard declaration on Action itself does not protect earlier execution', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='csrf').guards=[];
  p.steps.find(step=>step.id==='invoke').guards=['csrf-protected'];
  assert.throws(()=>composeRequestPlan(p),/prerequisite of Action/);
});
test('ordering hints cannot replace required guard contracts', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='invoke').requires=['action.authorized'];
  p.steps.find(step=>step.id==='csrf').before=['invoke'];
  assert.throws(()=>composeRequestPlan(p),/prerequisite of Action/);
});
test('unused authentication output does not prove Action authentication', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='authorize').requires=['action.identity'];
  assert.throws(()=>composeRequestPlan(p),/prerequisite of Action/);
});

test('authentication label on shared context does not prove session identity', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='identity').guards=[];
  p.steps.find(step=>step.id==='context').guards=['authenticated'];
  assert.throws(()=>composeRequestPlan(p),/guard declaration lacks corresponding evidence: authenticated/);
});
test('CSRF label on unrelated context does not prove origin check', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='csrf').guards=[];
  p.steps.find(step=>step.id==='context').guards=['csrf-protected'];
  assert.throws(()=>composeRequestPlan(p),/guard declaration lacks corresponding evidence: csrf-protected/);
});
test('authorization label on identity step does not prove business grant', () => {
  const p=clone(base);
  p.steps.find(step=>step.id==='authorize').guards=[];
  p.steps.find(step=>step.id==='identity').guards.push('action-authorized');
  assert.throws(()=>composeRequestPlan(p),/guard declaration lacks corresponding evidence: action-authorized/);
});
test('authentication policy identity cannot be defined twice', () => {
  const p=clone(authentication);
  p.policies.push(clone(p.policies[0]));
  assert.throws(()=>validateAuthenticationPlan(p),/duplicate authentication policy identity/);
});
test('duplicate source identity mapping is not silently coalesced', () => {
  const p=clone(authentication);
  p.subjectLinks.push(clone(p.subjectLinks[0]));
  assert.throws(()=>validateAuthenticationPlan(p),/duplicate identity link declaration/);
});
test('empty authentication plan is not a viable provider selection', () => {
  const p=clone(authentication);
  p.providers=[];
  p.policies=[];
  p.subjectLinks=[];
  assert.throws(()=>validateAuthenticationPlan(p),/active policy and provider/);
});

test('unknown plugin guard without an evidence binding is rejected', () => {
  const p=clone(base);
  p.hostPolicy.requiredGuards.push('tenant-isolated');
  p.steps.find(step=>step.id==='context').guards=['tenant-isolated'];
  assert.throws(()=>composeRequestPlan(p),/required guard has no declared evidence contract/);
});
test('plugin guard with explicit evidence is a causal Action prerequisite', () => {
  const p=clone(base);
  p.hostPolicy.requiredGuards.push('tenant-isolated');
  p.hostPolicy.guardEvidence={'tenant-isolated':'tenant.boundary.checked'};
  p.steps.push({id:'tenant-check',scope:'server',requires:['identity.principal'],provides:['tenant.boundary.checked'],guards:['tenant-isolated']});
  p.steps.find(step=>step.id==='invoke').requires.push('tenant.boundary.checked');
  const result=composeRequestPlan(p);
  assert.ok(result.order.indexOf('tenant-check')<result.order.indexOf('invoke'));
});
test('plugin guard with only a declared before hint cannot guard an Action', () => {
  const p=clone(base);
  p.hostPolicy.requiredGuards.push('tenant-isolated');
  p.hostPolicy.guardEvidence={'tenant-isolated':'tenant.boundary.checked'};
  p.steps.push({id:'tenant-check',scope:'server',requires:['identity.principal'],provides:['tenant.boundary.checked'],guards:['tenant-isolated'],before:['invoke']});
  assert.throws(()=>composeRequestPlan(p),/prerequisite of Action/);
});
test('standard guard evidence cannot be redefined by a plugin', () => {
  const p=clone(base);
  p.hostPolicy.guardEvidence={'authenticated':'request.context'};
  assert.throws(()=>composeRequestPlan(p),/standard guard evidence cannot be overridden/);
});

test('action route cannot drop authenticated host guard', () => {
  const p=clone(base);p.hostPolicy.requiredGuards=p.hostPolicy.requiredGuards.filter(g=>g!=='authenticated');
  assert.throws(()=>composeRequestPlan(p),/mandatory protection: authenticated/);
});
test('action route cannot drop csrf host guard', () => {
  const p=clone(base);p.hostPolicy.requiredGuards=p.hostPolicy.requiredGuards.filter(g=>g!=='csrf-protected');
  assert.throws(()=>composeRequestPlan(p),/mandatory protection: csrf-protected/);
});
test('action route cannot omit denied or failed audit events', () => {
  for(const terminal of ['completed','rejected','error']){
    const p=clone(base);p.hostPolicy.terminalEvents=p.hostPolicy.terminalEvents.filter(e=>e!==terminal);
    assert.throws(()=>composeRequestPlan(p),new RegExp('mandatory terminal audit: '+terminal));
  }
});
test('different providers with the same issuer and subject may have separate explicit principals', () => {
  const p=clone(authentication);
  p.providers.push({id:'second-oidc',kind:'oidc',issuer:'https://example.invalid/issuer'});
  p.policies[0].allowedProviders.push('second-oidc');
  p.subjectLinks.push({providerId:'second-oidc',issuer:'https://example.invalid/issuer',subject:'fixed-sub-001',principalId:'separate-principal'});
  assert.equal(validateAuthenticationPlan(p).linkedSubjectCount,3);
});

test('the same external identity cannot change application PrincipalID', () => {
  const p=clone(authentication);
  p.subjectLinks.push({...p.subjectLinks[0],principalId:'attacker-principal'});
  assert.throws(()=>validateAuthenticationPlan(p),/conflicting principal mapping/);
});


test('protected Action rejects HTTP route aliases the Go Host cannot safely match', () => {
  for(const path of [
    '/scenario/%73ubmit',
    '/scenario//submit',
    '/scenario/./submit',
    '/scenario/../submit',
    '/scenario/submit/.',
    '/scenario/submit/..',
    '/scenario/submit?role=admin',
    '/scenario/submit#fragment',
    '/scenario/submit\\\\admin',
    '/scenario/sub mit',
    '/scenario/受注',
    '//scenario/submit'
  ]){
    const p=clone(base);
    p.route.path=path;
    assert.throws(()=>composeRequestPlan(p),/exact unambiguous path/,path);
  }
});

test('protected Action requires the exact POST method and explicit protected exposure', () => {
  for(const method of ['GET','get','post','POST ','PUT','OPTIONS']){
    const p=clone(base);
    p.route.method=method;
    assert.throws(()=>composeRequestPlan(p),/uppercase HTTP token|protected Action reference/,method);
  }
  for(const exposure of [undefined,'public','interactive-session','protected-typo']){
    const p=clone(base);
    p.route.exposure=exposure;
    assert.throws(()=>composeRequestPlan(p),exposure==='public' ? /public/ : /protected exposure/,String(exposure));
  }
  for(const invalid of [null,0,{},[],true,' ',undefined]){
    const p=clone(base);
    p.route.actionId=invalid;
    assert.throws(()=>composeRequestPlan(p),/action identity must be a nonempty string/,String(invalid));
  }
});

test('non-Action route cannot keep the business Action invocation step', () => {
  for(const exposure of ['public','protected']){
    const p=clone(base);
    p.route={id:'corporate.callback',method:'GET',path:'/login/callback/corporate',exposure};
    assert.throws(()=>composeRequestPlan(p),/Action invocation requires an explicit Action route/,exposure);
  }
});

test('public callback without an Action remains a distinct GET route', () => {
  const p=clone(base);
  p.route={id:'corporate.callback',method:'GET',path:'/login/callback/corporate',exposure:'public'};
  p.hostPolicy.requiredGuards=[];
  p.hostContracts=[];
  p.steps=[
    {id:'context',scope:'server',provides:['request.context']},
    {id:'callback-validation',scope:'server',requires:['request.context'],provides:['callback.validated']},
    {id:'audit',scope:'server',requires:['request.context'],onTerminal:['completed','rejected','error']}
  ];
  const composed=composeRequestPlan(p);
  assert.equal(composed.routeId,'corporate.callback');
  assert.ok(composed.order.includes('callback-validation'));
  assert.ok(!composed.order.includes('invoke'));
});


test('protected Action route must be byte-exact after Go net/url EscapedPath', () => {
  for(const path of [
    '/scenario/a!b',
    '/scenario/a"b',
    "/scenario/a'b",
    '/scenario/a*b',
    '/scenario/a(b)',
    '/scenario/a[b]',
    '/scenario/a{b}',
    '/scenario/a<b>',
    '/scenario/a|b',
    '/scenario/a^b',
    '/scenario/a\`b'
  ]){
    const p=clone(base);
    p.route.path=path;
    assert.throws(()=>composeRequestPlan(p),/exact unambiguous path/,path);
  }
});

test('protected Action retains Go URL paths that do not need escaping', () => {
  for(const path of ['/scenario/submit','/scenario/+@;,$','/scenario/a:bc&def=2']){
    const p=clone(base);
    p.route.path=path;
    assert.equal(composeRequestPlan(p).routeId,'sales.submit');
  }
});
