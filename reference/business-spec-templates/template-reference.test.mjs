import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import {inspectTemplate,proposeAdoption,inspectUpgrade,TemplateError} from './template-reference.mjs';
const template=JSON.parse(fs.readFileSync(new URL('./order-entry.template.json',import.meta.url),'utf8'));
const deep=v=>structuredClone(v);
test('three-view seed references existing semantic, application and procedure assets without embedded UI',()=>{
  const result=inspectTemplate(template);
  assert.equal(result.refCount,9);assert.deepEqual(new Set(result.viewCoverage),new Set(['data','function','business']));
  assert.equal(JSON.stringify(template).includes('react'),false);
});
test('adoption without project decisions is unresolved, not accepted',()=>{
  const p=proposeAdoption(template,{projectId:'project-x'});
  assert.equal(p.accepted,false);assert.equal(p.unresolved.length,4);
  assert.equal(p.next,'explicit_shared_acceptance_required');
});
test('even all human-confirmed questions are still a proposal pending shared acceptance',()=>{
  const decisions=Object.fromEntries(template.projectDecisions.map(x=>[x.id,
    {status:'human-confirmed',answer:'For project x, policy decision '+x.id,sourceRef:'meeting-001'}]));
  const p=proposeAdoption(template,{projectId:'project-x',decisions});
  assert.equal(p.accepted,false);assert.equal(p.unresolved.length,0);assert.equal(p.answered.length,4);
});
test('AI proposed or ungrounded answer cannot masquerade as human confirmation',()=>{
  const p=proposeAdoption(template,{projectId:'p',decisions:{'submit-authority':{status:'ai-proposed',answer:'all users',sourceRef:'llm'}}});
  assert.ok(p.unresolved.includes('submit-authority'));
  const q=proposeAdoption(template,{projectId:'p',decisions:{'submit-authority':{status:'human-confirmed',answer:'all users'}}});
  assert.ok(q.unresolved.includes('submit-authority'));
});
test('UI renderer definition inside business template is rejected',()=>{
  const bad=deep(template);bad.framework='react';
  assert.throws(()=>inspectTemplate(bad),e=>e instanceof TemplateError&&e.code==='ui_layer_leak');
});
test('no duplicate source binding, no missing viewpoint and no unknown decision id',()=>{
  const duplicate=deep(template);duplicate.sourceBindings.push(deep(duplicate.sourceBindings[0]));
  assert.throws(()=>inspectTemplate(duplicate),e=>e.code==='invalid_view_coverage');
  const missing=deep(template);missing.sourceBindings=missing.sourceBindings.filter(x=>x.view!=='business');
  assert.throws(()=>inspectTemplate(missing),e=>e.code==='invalid_view_coverage');
  assert.throws(()=>proposeAdoption(template,{projectId:'p',decisions:{'pretend-rule':{status:'human-confirmed'}}}),e=>e.code==='unrecognized_project_decision');
});
test('upgrade changed source meaning requires review and does not mutate project',()=>{
  const proposal=proposeAdoption(template,{projectId:'project-x'}),old=structuredClone(proposal);
  const newer=deep(template);newer.templateVersion='0.2.0';
  newer.projectDecisions.push({id:'exception-path',question:'What exceptions apply?'});
  const r=inspectUpgrade(template,newer,proposal);
  assert.equal(r.requiresReview,true);assert.equal(r.acceptedProjectChanged,false);
  assert.deepEqual(proposal,old);
});
test('source-equivalent version bump is not an accepted meaning change',()=>{
  const newer=deep(template);newer.templateVersion='0.1.1';
  const r=inspectUpgrade(template,newer,proposeAdoption(template,{projectId:'p'}));
  assert.equal(r.changed,false);assert.equal(r.acceptedProjectChanged,false);
});
