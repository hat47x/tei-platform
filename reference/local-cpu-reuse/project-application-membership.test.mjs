import test from 'node:test';
import assert from 'node:assert/strict';
import {projectInspectionV0,linkApplicationPattern,InspectionProjectionError} from './project-inspection-v0.mjs';

function env(kind,canonical) {
  const subject={kind,id:canonical.id};
  return {schemaVersion:'tei.operations.inspect/v0',effect:'read-only',
    subject,canonical,worldFingerprint:'world:fixture',sourceIndexFingerprint:'source:fixture',
    sourceStatus:'indexed',sources:{subject,
      artifacts:[{path:'apps/example.tei',fingerprint:'sha256:fixture'}],
      references:[{subject,artifactPath:'apps/example.tei',role:'definition'}]
    }};
}
const app=()=>env('application',{id:'order_entry',semanticUses:['sales','customer'],validations:['sales.order_line.quantity_positive'],
  databases:[{alias:'sales',target:'sales_core',access:'read-write',requires:[]}],
  patterns:[{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'},
    {id:'order_entry.submit_order',kind:'command',capability:'sales.order.submit'}]});
const pattern=(id='order_entry.order_lines')=>env('application-pattern',{id,kind:'header-detail',relation:'sales.order.lines'});

test('actual Go Application shape yields a membership-only projection, not business approval',()=>{
  const result=projectInspectionV0(app());
  assert.equal(result.features.coverage,'membership_only');
  assert.deepEqual(result.features.declaredPatternIds,['order_entry.order_lines','order_entry.submit_order']);
  assert.equal(result.authority,'derived_observation_only');
  assert.equal(result.contractConformance,'not_verified');
});
test('explicit ID membership in matching normalized-world snapshot is a declared link only',()=>{
  const result=linkApplicationPattern(projectInspectionV0(app()),projectInspectionV0(pattern()));
  assert.deepEqual(result,{status:'declared_membership',accepted:false});
});
test('pattern is not a member merely because its ID starts with application name',()=>{
  const result=linkApplicationPattern(projectInspectionV0(app()),projectInspectionV0(pattern('order_entry.similar_name')));
  assert.equal(result.status,'unresolved_membership');
});
test('different workspace snapshot cannot authorize a declared pattern relationship',()=>{
  const p=pattern();p.worldFingerprint='world:changed';
  const result=linkApplicationPattern(projectInspectionV0(app()),projectInspectionV0(p));
  assert.equal(result.status,'stale_or_different_snapshot');
});
test('duplicate Canonical Pattern identity is partial and never establishes membership',()=>{
  const a=app();a.canonical.patterns.push({...a.canonical.patterns[0]});
  const observed=projectInspectionV0(a);
  assert.equal(observed.completeness,'partial');
  assert.ok(observed.diagnostics.includes('duplicate_pattern_identity'));
  assert.equal(linkApplicationPattern(observed,projectInspectionV0(pattern())).status,'unknown');
});
test('unsupported nested Pattern property is not silently treated as a complete observation',()=>{
  const a=app();a.canonical.patterns[0].unknown='invented';
  const observed=projectInspectionV0(a);
  assert.equal(observed.completeness,'partial');
  assert.ok(observed.diagnostics.includes('patterns[0].unknown'));
});
test('invalid databases field does not silently project into an empty set',()=>{
  const a=app();a.canonical.databases='invalid';
  assert.throws(()=>projectInspectionV0(a),
    e=>e instanceof InspectionProjectionError&&e.code==='malformed_canonical');
});
