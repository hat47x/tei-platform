/**
 * ORG-XD-06..09 + ORG-XD-10..12: synthetic, read-only qualification probes.
 * The same org-synthetic-r7 fixture is used for membership/FTE inputs.
 * Example Case/Grant/ACK facts and two-person approval are test assumptions,
 * NOT accepted TEI organization rules or real Provider/Host evidence.
 * Run: node --test reference/org-semantic-views/verify-decision-cutover.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, membershipVersionsAt } from './source-fixture.mjs';

const validAt = '2027-06-15';
const cutFields = ['sourceRevision', 'validAt', 'knownAt', 'scope',
  'readAttempt', 'provider', 'readSnapshot', 'publicationGeneration'];
function observe(knownAt, generation) {
  const rows = membershipVersionsAt(fixture, validAt, knownAt);
  const allocations = new Map(fixture.allocations.map(x => [x.relation, x.basisPoints]));
  if (allocations.size !== fixture.allocations.length ||
      rows.some(row => !allocations.has(row.id)) ||
      new Set(rows.map(row => row.id)).size !== rows.length) {
    throw new Error('synthetic occurrence/allocation ambiguity');
  }
  const meta = {sourceRevision:fixture.sourceRevision, validAt, knownAt,
    scope:'fixture:all-memberships', readAttempt:'read-'+generation,
    provider:'fixture-only', readSnapshot:'snapshot-'+generation,
    publicationGeneration:generation, coverage:'closed-synthetic'};
  return {
    members:{meta:{...meta}, value:rows.map(x=>x.id).sort()},
    fte:{meta:{...meta}, value:rows.reduce((sum,row)=>sum+allocations.get(row.id),0)}
  };
}
function qualifiedCurrentPair(members,fte) {
  if (!members || !fte || members.meta?.coverage !== 'closed-synthetic' ||
      fte.meta?.coverage !== 'closed-synthetic' ||
      cutFields.some(k => !members.meta[k] || members.meta[k] !== fte.meta[k])) {
    return 'hold-mixed-or-unknown-cut';
  }
  return 'fixture-consistent-candidate'; // Not a real publication or shared DB snapshot.
}
function reuseClaim(claim, prior, next) {
  // An empty dependency list is not proof that a business claim is constant.
  const dependencies=claim?.dependencies;
  if (!Array.isArray(dependencies) || dependencies.length===0 ||
      dependencies.some(key=>typeof key!=='string' || !key ||
        prior?.[key]===undefined || next?.[key]===undefined)) return 'unknown';
  return dependencies.every(key=>prior[key]===next[key])
    ? 'candidate-internal-reuse' : 'revalidate';
}
function disclosureTo(context, checked) {
  // This is only a fixture-side delivery candidate, not a Host grant.
  const fields=['tenant','issuer','principal','recipient','purpose','audience',
    'policyEpoch','projection','sourceRevision','readAttempt','readSnapshot',
    'publicationGeneration'];
  return checked?.allowed === true &&
    fields.every(k=>context?.[k]!==undefined && context[k]!==null &&
      context[k]!=='' && checked[k]===context[k])
    ? 'candidate-for-delivery' : 'deny-or-recheck';
}
function handoffDecision(input) {
  if (input.caseRevision !== input.ack?.caseRevision ||
      input.proposedTo !== input.ack?.receivedBy) return 'await-valid-ack';
  if (input.handlingUnit !== input.proposedTo) return 'await-case-responsibility-cutover';
  if (input.currentGrant?.verified !== true ||
      input.currentGrant?.caseRevision !== input.caseRevision ||
      input.currentGrant?.principal !== input.actorPrincipal ||
      input.currentGrant?.policyEpoch !== input.currentPolicyEpoch) return 'deny-current-grant';
  return 'candidate-at-effect-fence'; // Host authorization and effect are NOT executed.
}
function distinctApprovers(personA, personB, acceptedPersonLink) {
  if (!acceptedPersonLink || !personA || !personB) return 'unknown';
  return personA === personB ? 'conflict-same-person' : 'candidate-distinct-people';
}

const oldCut=observe('2027-07-01','G1');
const newCut=observe('2027-07-03','G2');

test('ORG-XD-06: independent old/new fixture cohorts have 4/3 memberships and 300/240 hundredths',()=>{
  assert.deepEqual(oldCut.members.value,['A1','A2','A3','A4']);
  assert.deepEqual(newCut.members.value,['A1','A3','A4']);
  assert.equal(oldCut.fte.value,300);
  assert.equal(newCut.fte.value,240);
  assert.equal(qualifiedCurrentPair(oldCut.members,oldCut.fte),'fixture-consistent-candidate');
  assert.equal(qualifiedCurrentPair(newCut.members,newCut.fte),'fixture-consistent-candidate');
});
test('ORG-XD-06: correct separate results cannot be combined as one current organization',()=>{
  assert.equal(qualifiedCurrentPair(newCut.members,oldCut.fte),'hold-mixed-or-unknown-cut');
  const spoof={...newCut.fte,meta:{...newCut.fte.meta, readAttempt:oldCut.fte.meta.readAttempt}};
  assert.equal(qualifiedCurrentPair(newCut.members,spoof),'hold-mixed-or-unknown-cut');
  assert.equal(qualifiedCurrentPair(newCut.members,{...newCut.fte,meta:{...newCut.fte.meta,coverage:'partial'}}),'hold-mixed-or-unknown-cut');
  // An explicit before/after comparison can show both cuts, without claiming one "now".
  assert.notEqual(oldCut.members.meta.publicationGeneration,newCut.members.meta.publicationGeneration);
});
test('ORG-XD-07: changing hidden Person Link invalidates person count but not a proven independent occurrence count',()=>{
  const before={cohort:'G1',membership:'m1',personLink:'p1',negativeRule:'r1'};
  const after={...before,personLink:'p2'};
  assert.equal(reuseClaim({dependencies:['cohort','membership']},before,after),'candidate-internal-reuse');
  assert.equal(reuseClaim({dependencies:['cohort','membership','personLink']},before,after),'revalidate');
  assert.equal(reuseClaim({dependencies:['cohort','membership','negativeRule']},before,{...before,negativeRule:'r2'}),'revalidate');
  assert.equal(reuseClaim({dependencies:['unknownRelation']},before,after),'unknown');
  assert.equal(reuseClaim({dependencies:[]},before,after),'unknown');
  assert.equal(reuseClaim({},before,after),'unknown');
  assert.equal(reuseClaim({dependencies:['']},before,after),'unknown');
});
test('ORG-XD-08: identical internal values do not extend disclosure to a new recipient, purpose or epoch',()=>{
  const checked={tenant:'T1',issuer:'issuer-1',principal:'viewer-1',
    recipient:'CLI',purpose:'review',audience:'internal',policyEpoch:'pe1',
    projection:'member-ids',sourceRevision:'org-r7',readAttempt:'attempt-1',
    readSnapshot:'snapshot-1',publicationGeneration:'G1',allowed:true};
  assert.equal(disclosureTo({...checked},checked),'candidate-for-delivery');
  for (const [key,value] of [['tenant','T2'],['issuer','issuer-2'],
    ['principal','viewer-2'],['recipient','SHO'],['purpose','export'],
    ['audience','public'],['policyEpoch','pe2'],['projection','person-edges'],
    ['sourceRevision','org-r8'],['readAttempt','attempt-2'],
    ['readSnapshot','snapshot-2'],['publicationGeneration','G2']]) {
    assert.equal(disclosureTo({...checked,[key]:value},checked),'deny-or-recheck',key);
  }
  assert.equal(disclosureTo({...checked},{...checked,allowed:false}),'deny-or-recheck');
  assert.equal(disclosureTo({...checked,principal:undefined},checked),'deny-or-recheck');
  assert.equal(disclosureTo({...checked,readSnapshot:''},checked),'deny-or-recheck');
});

// Synthetic qualification only: unchanged Membership output does not make
// an allocation-dependent claim reusable after the allocation revision moves.
test('ORG-XD-13: same memberships/people, new allocation revision invalidates FTE but not occurrence count',()=>{
  const before={cohort:'G1',members:'A1,A2,A3,A4',personLink:'person-link-r1',
    allocationRevision:'allocation-r1',capacityRuleRevision:'capacity-r1'};
  const after={...before,allocationRevision:'allocation-r2'};
  assert.equal(reuseClaim({dependencies:['cohort','members']},before,after),
    'candidate-internal-reuse');
  assert.equal(reuseClaim({dependencies:['cohort','members','personLink']},before,after),
    'candidate-internal-reuse');
  assert.equal(reuseClaim({dependencies:['cohort','members','allocationRevision']},before,after),
    'revalidate');
  assert.equal(reuseClaim({dependencies:['cohort','members','allocationRevision',
    'capacityRuleRevision']},before,after),'revalidate');
  // The r7 fixture has only one fixed allocation per Membership ID: this
  // does NOT demonstrate temporal allocation readback or real FTE evaluation.
});

// Same fields visible in the reference Web do not identify a Source revision,
// coverage proof, Provider snapshot or disclosure principal.
test('ORG-XD-14: identical display identity cannot qualify a changed source, read or recipient',()=>{
  const display={selectedOrderKey:'1001',environment:'reference',
    contractQuery:'order/contract/detail',rowCount:4};
  const visibleAfter={...display};
  assert.deepEqual(display,visibleAfter);
  const changedSource={...oldCut.fte,meta:{...oldCut.fte.meta,sourceRevision:'org-synthetic-r8'}};
  const changedScope={...oldCut.fte,meta:{...oldCut.fte.meta,scope:'fixture:page-1'}};
  const changedSnapshot={...oldCut.fte,meta:{...oldCut.fte.meta,readSnapshot:'new-snapshot'}};
  for (const other of [changedSource,changedScope,changedSnapshot]) {
    assert.equal(qualifiedCurrentPair(oldCut.members,other),'hold-mixed-or-unknown-cut');
  }
  const disclosed={tenant:'T1',issuer:'issuer-1',principal:'viewer-1',
    recipient:'CLI',purpose:'review',audience:'internal',policyEpoch:'pe1',
    projection:'member-ids',sourceRevision:'org-synthetic-r7',
    readAttempt:'read-G1',readSnapshot:'snapshot-G1',
    publicationGeneration:'G1',allowed:true};
  assert.equal(disclosureTo({...disclosed},disclosed),'candidate-for-delivery');
  assert.equal(disclosureTo({...disclosed,principal:'viewer-2'},disclosed),'deny-or-recheck');
  // Display identity is not evidence that another Consumer received the result.
});
test('ORG-XD-09: A-to-B-to-A displayed values do not reinstate an old response or grant',()=>{
  const aAgain={...oldCut.members,meta:{...oldCut.members.meta,
    publicationGeneration:'G3',readAttempt:'read-G3',readSnapshot:'snapshot-G3'}};
  assert.deepEqual(aAgain.value,oldCut.members.value);
  assert.notEqual(aAgain.meta.publicationGeneration,oldCut.members.meta.publicationGeneration);
  assert.equal(qualifiedCurrentPair(aAgain,oldCut.fte),'hold-mixed-or-unknown-cut');
});
test('ORG-XD-10: an affiliation transfer alone does not transfer Case responsibility or prove handoff',()=>{
  const base={caseRevision:'case-v2',handlingUnit:'D1',proposedTo:'D2',
    ack:null,actorPrincipal:'principal-D2',currentPolicyEpoch:'p7',
    currentGrant:{verified:true,caseRevision:'case-v2',principal:'principal-D2',policyEpoch:'p7'}};
  assert.equal(handoffDecision(base),'await-valid-ack');
  assert.equal(handoffDecision({...base,ack:{receivedBy:'D2',caseRevision:'case-v2'}}),'await-case-responsibility-cutover');
  const cutover={...base,handlingUnit:'D2',ack:{receivedBy:'D2',caseRevision:'case-v2'}};
  assert.equal(handoffDecision(cutover),'candidate-at-effect-fence');
  assert.equal(handoffDecision({...cutover,ack:{receivedBy:'D2',caseRevision:'case-v1'}}),'await-valid-ack');
});
test('ORG-XD-11: previously valid grant is rejected at effect boundary after policy or actor change',()=>{
  const candidate={caseRevision:'case-v2',handlingUnit:'D2',proposedTo:'D2',
    ack:{receivedBy:'D2',caseRevision:'case-v2'},actorPrincipal:'principal-D2',currentPolicyEpoch:'p7',
    currentGrant:{verified:true,caseRevision:'case-v2',principal:'principal-D2',policyEpoch:'p7'}};
  assert.equal(handoffDecision(candidate),'candidate-at-effect-fence');
  assert.equal(handoffDecision({...candidate,currentPolicyEpoch:'p8'}),'deny-current-grant');
  assert.equal(handoffDecision({...candidate,actorPrincipal:'another'}),'deny-current-grant');
  assert.equal(handoffDecision({...candidate,currentGrant:{...candidate.currentGrant,verified:false}}),'deny-current-grant');
  // Whether the external Effect committed cannot be inferred from this candidate.
  assert.equal('effectReceipt' in candidate,false);
});
test('ORG-XD-12: distinct positions/memberships do not prove two independent approvers',()=>{
  const p2Memberships=membershipVersionsAt(fixture,validAt,'2027-07-01').filter(x=>x.person==='P2');
  assert.deepEqual(p2Memberships.map(x=>x.id).sort(),['A2','A3']);
  assert.equal(distinctApprovers(p2Memberships[0].person,p2Memberships[1].person,true),'conflict-same-person');
  assert.equal(distinctApprovers('P1','P2',false),'unknown');
  assert.equal(distinctApprovers('P1','P2',true),'candidate-distinct-people');
  // Accepted Person Link does not imply trusted Principal binding or Host permission.
});
