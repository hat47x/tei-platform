import test from 'node:test';
import assert from 'node:assert/strict';
import {projectInspectionV0,linkPatternRelation,InspectionProjectionError} from './project-inspection-v0.mjs';

const subject=(kind,id)=>({kind,id});
function envelope(kind,canonical) {
  const s=subject(kind,canonical.id);
  return {
    schemaVersion:'tei.operations.inspect/v0',effect:'read-only',subject:s,canonical,
    worldFingerprint:'world:synthetic-fixture',sourceIndexFingerprint:'index:synthetic-fixture',
    sourceStatus:'indexed',
    sources:{
      subject:s,
      artifacts:[{path:'apps/order-entry/patterns.tei',fingerprint:'sha256:synthetic-fixture'}],
      references:[{subject:s,artifactPath:'apps/order-entry/patterns.tei',role:'definition'}]
    }
  };
}
const project=x=>projectInspectionV0(x);
const failure=code=>e=>e instanceof InspectionProjectionError&&e.code===code;

test('real Go application.Pattern-shaped header-detail projects relation, not full contract',()=>{
  const r=project(envelope('application-pattern',{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'}));
  assert.equal(r.features.relation,'sales.order.lines');
  assert.equal(r.completeness,'structurally_projected');
  assert.equal(r.authority,'derived_observation_only');
  assert.equal(r.acceptance,'not_asserted');
  assert.equal(r.contractConformance,'not_verified');
  assert.equal(r.provenance.worldFingerprint,'world:synthetic-fixture');
});
test('Go ContractQueryInvocation inputs are retained, not reinterpreted as signature',()=>{
  const c={id:'order_entry.customer_reference',kind:'reference',relation:'sales.order.customer',datasource:'master',
    contractQuery:{contract:'master_data.customer',query:'find_by_id',
      inputs:[{targetAttribute:'customer.customer.customer_id',input:'id'}]}};
  const r=project(envelope('application-pattern',c));
  assert.equal(r.features.contractQuery.inputs[0].targetAttribute,'customer.customer.customer_id');
  assert.equal(r.features.contractQuery.query,'find_by_id');
  assert.equal(r.contractConformance,'not_verified');
});
test('known relation cardinality and join keys project from semantic.Relation',()=>{
  const relation={id:'sales.order.customer',symbol:'customer',target:'customer.customer',cardinality:'one',
    keys:[{sourceAttribute:'sales.order.customer_id',targetAttribute:'customer.customer.customer_id'}]};
  const p=project(envelope('semantic-relation',relation));
  assert.equal(p.features.keys[0].sourceAttribute,'sales.order.customer_id');
  assert.equal(p.features.cardinality,'one');
});
test('logical keys and functional dependencies preserved independently of physical schema',()=>{
  const data={id:'sales.order',symbol:'Order',
    attributes:[{id:'sales.order.order_id',symbol:'orderId',type:'sales.order_id'}],
    relations:[{id:'sales.order.lines',symbol:'lines',target:'sales.order_line',cardinality:'many',keys:[]}],
    candidateKeys:[{id:'sales.order.key',symbol:'business-key',attributes:['sales.order.order_id']}],
    functionalDependencies:[{id:'sales.order.fd',symbol:'key-determines-status',
       determinants:['sales.order.order_id'],dependents:['sales.order.status']}]};
  const p=project(envelope('semantic-data',data));
  assert.equal(p.features.candidateKeys[0].attributes[0],'sales.order.order_id');
  assert.equal(p.features.functionalDependencies[0].dependents[0],'sales.order.status');
  assert.equal(p.features.relations[0].cardinality,'many');
});
test('declared header-detail relation link uses same workspace snapshot, does not assert acceptance',()=>{
  const p=project(envelope('application-pattern',{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'}));
  const r=project(envelope('semantic-relation',{id:'sales.order.lines',target:'sales.order_line',cardinality:'many',keys:[]}));
  assert.deepEqual(linkPatternRelation(p,r),{status:'declared_reference_match',accepted:false});
});
test('different World fingerprint is not compatible for linking',()=>{
  const p=project(envelope('application-pattern',{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'}));
  const other=envelope('semantic-relation',{id:'sales.order.lines',target:'sales.order_line',cardinality:'many'});
  other.worldFingerprint='world:another-revision';
  assert.equal(linkPatternRelation(p,project(other)).status,'stale_or_different_snapshot');
});
test('unresolved link is reported without guessing from similar names',()=>{
  const p=project(envelope('application-pattern',{id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'}));
  const r=project(envelope('semantic-relation',{id:'sales.order.other_lines',target:'sales.order_line',cardinality:'many'}));
  assert.equal(linkPatternRelation(p,r).status,'unresolved_reference');
});
test('unsupported canonical keys make extraction partial, without silently claiming completeness',()=>{
  const inp=envelope('application-pattern',{id:'order_entry.new',kind:'command',capability:'sales.order.submit',unknownAction:'secret'});
  const p=project(inp);
  assert.equal(p.completeness,'partial');
  assert.ok(p.diagnostics.includes('canonical.unknownAction'));
});
test('unsupported pattern kind and unindexed Source remain unresolved',()=>{
  const x=envelope('application-pattern',{id:'order_entry.experimental',kind:'gesture-swipe'});
  x.sourceStatus='not-indexed';x.sources.references=[];x.sources.artifacts=[];
  const p=project(x);
  assert.equal(p.completeness,'partial');
  assert.ok(p.diagnostics.includes('unknown_pattern_kind'));
  assert.ok(p.diagnostics.includes('source_not_indexed'));
});
test('malformed unknown nested contract query property cannot silently disappear',()=>{
  const x=envelope('application-pattern',{id:'order_entry.ref',kind:'reference',relation:'sales.order.customer',
    contractQuery:{contract:'master_data.customer',query:'find_by_id',inputs:[],opaqueSideEffect:'update'}});
  const p=project(x);
  assert.equal(p.completeness,'partial');
  assert.ok(p.diagnostics.includes('contractQuery.opaqueSideEffect'));
});
test('subject mismatch is rejected and cannot be relabeled as inspected identity',()=>{
  const x=envelope('semantic-relation',{id:'sales.order.lines',target:'sales.order_line',cardinality:'many'});
  x.subject={kind:'semantic-relation',id:'sales.order.customer'};
  assert.throws(()=>project(x),failure('canonical_subject_mismatch'));
});
test('cross-subject Source Index reference is rejected',()=>{
  const x=envelope('semantic-relation',{id:'sales.order.lines',target:'sales.order_line',cardinality:'many'});
  x.sources.references[0].subject={kind:'semantic-relation',id:'sales.order.customer'};
  assert.throws(()=>project(x),failure('mixed_subject_trace'));
});
test('source indexed with no references or artifact must not count as complete',()=>{
  const x=envelope('application-pattern',{id:'order_entry.submit_order',kind:'command',capability:'sales.order.submit'});
  x.sources.references=[];
  assert.throws(()=>project(x),failure('inconsistent_source_status'));
});
test('unsupported envelope schema/effect is rejected before taking any features',()=>{
  const x=envelope('application-pattern',{id:'order_entry.submit_order',kind:'command',capability:'sales.order.submit'});
  x.schemaVersion='tei.operations.inspect/v1';
  assert.throws(()=>project(x),failure('unsupported_inspection_envelope'));
  x.schemaVersion='tei.operations.inspect/v0';x.effect='write';
  assert.throws(()=>project(x),failure('unsupported_inspection_envelope'));
});
test('malformed canonical body never silently becomes empty structure',()=>{
  const x=envelope('semantic-data',{id:'sales.order',attributes:'not-an-array'});
  assert.throws(()=>project(x),failure('malformed_canonical'));
});
test('string canonical is explicitly JSON-decoded, still subject-checked',()=>{
  const x=envelope('application-pattern',{id:'order_entry.submit_order',kind:'command',capability:'sales.order.submit'});
  x.canonical=JSON.stringify(x.canonical);
  assert.equal(project(x).features.capability,'sales.order.submit');
});
