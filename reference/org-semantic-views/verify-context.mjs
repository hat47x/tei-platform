/**
 * Synthetic Function Data Context evidence: declared relation path does not imply
 * global retrieval, and "requested all" is not proof of actually received all.
 * Test-only read-only Reference; NOT a TEI Query/Context/Provider implementation.
 * Run: node --test reference/org-semantic-views/verify-context.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, baselineMemberships } from './source-fixture.mjs';

const def = Object.freeze({
  id: 'org.membership-by-department',
  relation: 'Membership', anchor: 'D1', grain: 'Relation occurrence',
  fields: ['id', 'person', 'dept'], requestedScope: 'all-for-selected-department',
  meaningRevision: fixture.sourceRevision,
});
const allowedFields = ['id','person','dept'];
function readContext(mapping, observation = {}, data = fixture) {
  if (!mapping || mapping.relation !== 'Membership' ||
      mapping.grain !== 'Relation occurrence' ||
      mapping.requestedScope !== 'all-for-selected-department' ||
      mapping.meaningRevision !== data.sourceRevision ||
      !data.departments.some(d => d.id === mapping.anchor) ||
      mapping.fields?.some(f => !allowedFields.includes(f)) ||
      !Array.isArray(mapping.fields) || !mapping.fields.includes('id')) {
    return { status: 'unsupported', reason: 'unresolved-explicit-binding' };
  }
  const all = baselineMemberships(data).filter(m => m.dept === mapping.anchor);
  const pageSize = observation.pageSize ?? all.length;
  if (!Number.isInteger(pageSize) || pageSize < 0)
    return { status: 'unsupported', reason: 'invalid-slice' };
  const rows = all.slice(0,pageSize).map(row=>Object.fromEntries(
    mapping.fields.map(field=>[field,row[field]])));
  const witness=observation.completeness ?? 'unknown';
  if (!['complete','partial','unknown'].includes(witness)) {
    return { status: 'unsupported', reason: 'unknown-completeness-evidence' };
  }
  // The closed fixture supplies an independent known answer for this synthetic test.
  // A real Provider must supply its *own* suitable completeness evidence.
  const completeness=witness==='complete' && rows.length<all.length
    ? 'inconsistent'
    : witness;
  return {
    status:'ok', definitionId:mapping.id, relation:mapping.relation,
    anchor:mapping.anchor, grain:mapping.grain, fields:mapping.fields,
    sourceRevision:data.sourceRevision, readEpoch:observation.epoch??7,
    providerSnapshot:observation.snapshot??'snap-3', requestedScope:mapping.requestedScope,
    resultScope:'selected-department', completeness, rows,
  };
}
function claimMembershipCount(context, department) {
  if (context.status !== 'ok' || context.anchor!==department ||
      context.completeness!=='complete') return {status:'unresolved'};
  return {status:'known',count:context.rows.length};
}
function claimExistence(context,person) {
  if(context.status!=='ok')return {status:'unresolved'};
  if(!context.fields.includes('person'))return {status:'unsupported'};
  if(context.rows.some(row=>row.person===person))return {status:'known',exists:true};
  return context.completeness==='complete'
    ? {status:'known',exists:false}
    : {status:'unresolved'};
}
function claimDistinct(context, department) {
  if(context.status!=='ok' || context.anchor!==department ||
    !context.fields.includes('person') || context.completeness!=='complete')
    return {status:'unresolved'};
  return {status:'known',count:new Set(context.rows.map(x=>x.person)).size};
}
function projectTwoConsumers(context) {
  if(context.status!=='ok') return {status:'unsupported'};
  const edges=context.rows.map(x=>({relationId:x.id,from:x.person,to:x.dept}));
  const tableRows=context.rows.map(x=>({relationId:x.id,subjectId:x.person,department:x.dept}));
  return {
    evidence:{sourceRevision:context.sourceRevision,anchor:context.anchor,
      epoch:context.readEpoch,completeness:context.completeness,
      grain:context.grain,providerSnapshot:context.providerSnapshot},
    tableRows,graphEdges:edges,
  };
}
const complete=(anchor='D1')=>readContext({...def,anchor},{completeness:'complete'});

test('explicit mapping requests D1 and returns A1/A2, not D2 A3/A4',()=>{
  const x=complete();
  assert.deepEqual(x.rows.map(r=>r.id),['A1','A2']);
  assert.equal(x.anchor,'D1'); assert.equal(x.completeness,'complete');
});
test('D2 requires its own explicit Context mapping',()=>{
  assert.deepEqual(complete('D2').rows.map(r=>r.id),['A3','A4']);
});
test('a complete D1 slice proves only the D1 count, not global four',()=>{
  const x=complete();
  assert.deepEqual(claimMembershipCount(x,'D1'),{status:'known',count:2});
  assert.deepEqual(claimMembershipCount(x,'D2'),{status:'unresolved'});
});
test('P2 occurs once in D1 but Person identity remains shared with D2',()=>{
  assert.equal(complete().rows.filter(r=>r.person==='P2').length,1);
  assert.equal(complete('D2').rows.filter(r=>r.person==='P2').length,1);
});
test('the presence of an Occupancy relation does not authorize following that path',()=>{
  assert.equal(complete().rows.some(r=>r.person==='P3'),false);
  assert.deepEqual(claimExistence(complete(),'P3'),{status:'known',exists:false});
});
test('no relation mapping is unsupported, not an implicit all-data graph traversal',()=>{
  assert.equal(readContext({...def,relation:null}).status,'unsupported');
});
test('unknown parent anchor is unresolved, not an automatic workspace scope',()=>{
  assert.equal(readContext({...def,anchor:'D9'}).status,'unsupported');
});
test('undeclared fields cannot be silently exported from the source',()=>{
  assert.equal(readContext({...def,fields:['id','person','hiddenPayroll']}).status,'unsupported');
});
test('partial page proves observed P1 but not absence of unobserved P2',()=>{
  const x=readContext(def,{pageSize:1,completeness:'partial'});
  assert.deepEqual(claimExistence(x,'P1'),{status:'known',exists:true});
  assert.deepEqual(claimExistence(x,'P2'),{status:'unresolved'});
});
test('requested all does not upgrade an unknown provider slice into complete',()=>{
  const x=readContext(def,{pageSize:2,completeness:'unknown'});
  assert.equal(x.requestedScope,'all-for-selected-department');
  assert.deepEqual(claimMembershipCount(x,'D1'),{status:'unresolved'});
});
test('declared all with reported partial still cannot prove final distinct count',()=>{
  const x=readContext(def,{pageSize:1,completeness:'partial'});
  assert.deepEqual(claimDistinct(x,'D1'),{status:'unresolved'});
});
test('contradictory complete witness fails synthetic closed-fixture validation',()=>{
  const x=readContext(def,{pageSize:1,completeness:'complete'});
  assert.equal(x.completeness,'inconsistent');
  assert.deepEqual(claimMembershipCount(x,'D1'),{status:'unresolved'});
});
test('missing person projection cannot be used for person distinct',()=>{
  const x=readContext({...def,fields:['id','dept']},{completeness:'complete'});
  assert.deepEqual(claimDistinct(x,'D1'),{status:'unresolved'});
});
test('two read-only consumers preserve membership IDs and limited scope',()=>{
  const x=projectTwoConsumers(complete());
  assert.deepEqual(x.tableRows.map(r=>r.relationId),x.graphEdges.map(r=>r.relationId));
  assert.deepEqual(x.tableRows.map(r=>r.relationId),['A1','A2']);
  assert.equal(x.evidence.anchor,'D1');
});
test('two consumers retain partial read and cannot present all of D1 as complete',()=>{
  const p=projectTwoConsumers(readContext(def,{pageSize:1,completeness:'partial'}));
  assert.equal(p.evidence.completeness,'partial');
  assert.equal(p.tableRows.length,1);
  assert.equal(p.graphEdges.length,1);
});
test('selection epoch and provider snapshot belong to the observation, not the source ID',()=>{
  const a=readContext(def,{completeness:'complete',epoch:7,snapshot:'snap-3'});
  const b=readContext(def,{completeness:'complete',epoch:8,snapshot:'snap-4'});
  assert.deepEqual(a.rows,b.rows);
  assert.notEqual(a.readEpoch,b.readEpoch);
  assert.notEqual(a.providerSnapshot,b.providerSnapshot);
});
test('D2 label-only change does not change D1 selected relation identities',()=>{
  const edited={...fixture,departments:[fixture.departments[0],{id:'D2',label:'Renamed'}]};
  assert.deepEqual(readContext(def,{completeness:'complete'},edited).rows,complete().rows);
});
test('no permission or actual effect is conferred by an authorized-looking mapping',()=>{
  const x=complete();const consumers=projectTwoConsumers(x);
  assert.equal('grant' in consumers.evidence,false);
  assert.equal('receipt' in consumers.evidence,false);
  assert.equal('commit' in x,false);
});

test('unknown source/model revision is not silently mapped into the current Context',()=>{
  const x=readContext({...def,meaningRevision:'org-synthetic-r8'},
    {completeness:'complete'});
  assert.deepEqual(x,{status:'unsupported',reason:'unresolved-explicit-binding'});
});
test('unspecified result grain or requested scope is not silently inferred',()=>{
  assert.equal(readContext({...def,grain:undefined}).status,'unsupported');
  assert.equal(readContext({...def,requestedScope:undefined}).status,'unsupported');
});
