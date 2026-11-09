import test from 'node:test';
import assert from 'node:assert/strict';
import {projectInspectionV0, InspectionProjectionError} from './project-inspection-v0.mjs';

function sample() {
  const subject={kind:'application-pattern',id:'order_entry.order_lines'};
  return {
    schemaVersion:'tei.operations.inspect/v0',effect:'read-only',subject,
    canonical:{id:subject.id,kind:'header-detail',relation:'sales.order.lines'},
    worldFingerprint:'world:test',sourceIndexFingerprint:'source:test',
    sourceStatus:'indexed',
    sources:{subject,artifacts:[{path:'app/patterns.tei',fingerprint:'sha256:test'}],
      references:[{subject,artifactPath:'app/patterns.tei',role:'definition'}]}
  };
}
const inconsistent=e=>e instanceof InspectionProjectionError&&e.code==='inconsistent_artifact_reference';

test('source references must have a matching indexed artifact',()=>{
  const x=sample();x.sources.references[0].artifactPath='another/private.txt';
  assert.throws(()=>projectInspectionV0(x),inconsistent);
});
test('duplicate artifact-path entries cannot hide a conflicting fingerprint',()=>{
  const x=sample();x.sources.artifacts.push({path:'app/patterns.tei',fingerprint:'sha256:different'});
  assert.throws(()=>projectInspectionV0(x),inconsistent);
});
test('valid source linkage is only observational, never an acceptance receipt',()=>{
  const result=projectInspectionV0(sample());
  assert.equal(result.completeness,'structurally_projected');
  assert.equal(result.acceptance,'not_asserted');
  assert.equal(result.authority,'derived_observation_only');
});
