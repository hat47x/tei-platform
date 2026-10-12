import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {proposeDFDReviewEdit, EDIT_SCHEMA} from './dfd-edit-proposal.mjs';

const root=dirname(fileURLToPath(import.meta.url));
const read=name=>JSON.parse(readFileSync(join(root,name),'utf8'));
const fixtures=[
  ['approval',read('dfd-order-approval/order-approval-procedure.json'),
    read('dfd-order-approval/review-proposal.json')],
  ['order-handling',read('order-entry-business-procedure.json'),
    read('order-entry-dfd-review-proposal.json')]
];
const reviewHash='sha256:'+'a'.repeat(64);
function edit(procedure,review){
  return {schema:EDIT_SCHEMA,id:'boundaries',origin:'studio',
    intent:'change only a business trigger label',
    procedureId:procedure.procedure.id,baseReviewSHA256:reviewHash,
    operation:{kind:'replace-trigger-label',trigger:review.triggers[0].id,
      label:'business notification candidate'}};
}
const clone=x=>JSON.parse(JSON.stringify(x));

for(const [caseName,procedure,review] of fixtures){
  test(caseName+': exact business review preserves noncanonical authority',()=>{
    const before=JSON.stringify(review);
    const p=proposeDFDReviewEdit(procedure,review,reviewHash,edit(procedure,review));
    assert.equal(p.authority,'unaccepted-proposal-only');
    assert.equal(p.readyForAcceptance,false);
    assert.deepEqual(p.sourceWrites,[]);
    assert.equal(p.modelRevisionStatus,'unverified');
    assert.equal(p.proposedReview.triggers[0].id,review.triggers[0].id);
    assert.equal(p.proposedReview.triggers[0].kind,review.triggers[0].kind);
    assert.equal(JSON.stringify(review),before);
  });

  const unauthorized=[
    ['runtime trigger',r=>{r.triggers[0].runtimeBinding={cron:'* * * * *'};}],
    ['authorization role',r=>{r.businessRoles[0].authorizationRole='admin';}],
    ['external party permission',r=>{r.externalParties[0].permissions=['write'];}],
    ['concrete store writer',r=>{r.stores[0].table='public.orders';}],
    ['responsibility execution',r=>{r.responsibilities[0].execute=true;}],
    ['control automatic action',r=>{r.flows[0].runtimeAction='run';}],
    ['information side effect',r=>{r.informationFlows[0].updateDatabase=true;}]
  ];
  for(const [name,patch] of unauthorized){
    test(caseName+': reject unauthorized nested '+name,()=>{
      const changed=clone(review);
      patch(changed);
      assert.throws(()=>proposeDFDReviewEdit(procedure,changed,reviewHash,
        edit(procedure,changed)),/UNKNOWN_OR_INVALID_FIELD/);
      assert.equal(JSON.stringify(review),JSON.stringify(clone(review)));
    });
  }

  test(caseName+': sequence cannot secretly carry a branch condition',()=>{
    const changed=clone(review);
    const sequence=changed.flows.find(f=>f.kind==='sequence');
    assert.ok(sequence);
    sequence.condition='only if manager approved';
    assert.throws(()=>proposeDFDReviewEdit(procedure,changed,reviewHash,
      edit(procedure,changed)),/SEQUENCE_CANNOT_CLAIM_CONDITION/);
  });
}
