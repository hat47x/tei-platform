import test from 'node:test';
import assert from 'node:assert/strict';
import {REFERENCE_SUBJECTS,summarizeInspectionEvidence,EvidenceSummaryError} from './inspection-evidence-summary.mjs';
const observed=(s)=>({schemaVersion:'tei.research.inspection-features/v0',effect:'read-only',
  authority:'derived_observation_only',acceptance:'not_asserted',contractConformance:'not_verified',
  subject:{...s},completeness:'structurally_projected',diagnostics:[],
  provenance:{worldFingerprint:'world:w',sourceIndexFingerprint:'index:i',sourceStatus:'indexed'},
  features:{businessName:'private customer name'}});
const make=()=>({schemaVersion:'tei.research.inspection-batch/v0',effect:'read-only',
  authority:'derived_observation_only',acceptance:'not_asserted',worldFingerprint:'world:w',
  sourceIndexFingerprint:'index:i',observations:REFERENCE_SUBJECTS.map(observed),
  links:{applicationPattern:{status:'declared_membership',accepted:false},
    patternRelation:{status:'declared_reference_match',accepted:false}}});
const denied=cb=>assert.throws(cb,e=>e instanceof EvidenceSummaryError&&e.code==='INVALID_INSPECTION_EVIDENCE');

test('fixed genuine-shaped batch yields only allowlisted metadata',()=>{
  const input=make();input.links.applicationPattern.secret='private customer name';
  const result=summarizeInspectionEvidence(input),serialized=JSON.stringify(result);
  assert.equal(result.consistentSnapshot,true);assert.equal(result.subjectCount,5);
  for(const forbidden of ['order_entry','sales.order','private customer name','world:w','index:i'])
    assert.equal(serialized.includes(forbidden),false);
});
test('drifting World or Source Index is rejected instead of reporting consistency',()=>{
  for(const key of ['worldFingerprint','sourceIndexFingerprint']){
    const input=make();input.observations[3].provenance[key]='other';denied(()=>summarizeInspectionEvidence(input));
  }
});
test('reordered, omitted or extra subjects fail closed',()=>{
  const reordered=make();reordered.observations.reverse();denied(()=>summarizeInspectionEvidence(reordered));
  const missing=make();missing.observations.pop();denied(()=>summarizeInspectionEvidence(missing));
  const extra=make();extra.observations.push(observed(REFERENCE_SUBJECTS[0]));denied(()=>summarizeInspectionEvidence(extra));
});
test('non-authoritative contract claims cannot be summarized',()=>{
  const input=make();input.observations[1].contractConformance='verified';denied(()=>summarizeInspectionEvidence(input));
  const accepted=make();accepted.links.patternRelation.accepted=true;denied(()=>summarizeInspectionEvidence(accepted));
});
test('invalid or sensitive link statuses never propagate into stdout',()=>{
  const input=make();input.links.applicationPattern.status='private customer name';denied(()=>summarizeInspectionEvidence(input));
  const unknown=make();unknown.links.applicationPattern.status='unknown';
  assert.equal(summarizeInspectionEvidence(unknown).links.applicationPattern.status,'unknown');
});
test('invalid provenance and completeness are rejected',()=>{
  const input=make();input.observations[2].provenance.sourceStatus='private';denied(()=>summarizeInspectionEvidence(input));
  const partial=make();partial.observations[0].completeness='partial';
  assert.equal(summarizeInspectionEvidence(partial).subjects[0].completeness,'partial');
});
test('wrong batch schema or source subject identity is rejected',()=>{
  const input=make();input.schemaVersion='other';denied(()=>summarizeInspectionEvidence(input));
  const altered=make();altered.observations[0].subject.id='another_app';denied(()=>summarizeInspectionEvidence(altered));
});
