import test from 'node:test';
import assert from 'node:assert/strict';
import {project, selectView, boundedAbsence} from './coverage.mjs';
const one={scope:'one',kind:'operation',id:'shared'};
const two={scope:'two',kind:'operation',id:'shared'};
const output={scope:'one',kind:'artifact',id:'output'};
const evidence={scope:'one',kind:'evidence',id:'validation'};
const fixture=()=>({
  snapshot:{workspace:'W',modelRevision:'R10',sourceIndexFingerprint:'S10',projectionRevision:'P10'},
  subjects:[one,two,output,evidence],
  relations:[{id:'flow',type:'produces',roles:[{role:'producer',subject:one},{role:'output',subject:output},{role:'evidence',subject:evidence}]}],
});
const closure=snapshot=>({snapshot,scopes:['one','two'],subjectsComplete:true,relationsComplete:true,rolesComplete:true,compositionComplete:true,readerPinned:true,disclosureQualified:true,omitted:0,opaque:0,evidenceId:'synthetic-fixture'});
const missing={kind:'relation',scope:'one',type:'unseen'};
test('two scopes with same kind/id and two views do not collapse identity',()=>{
  const f=fixture(),v=project(f);
  assert.notEqual(v.table.subjects[0].key,v.table.subjects[1].key);
  assert.deepEqual(selectView(v,v.table.subjects[0].viewId,f.snapshot),{status:'candidate',qualifiedKey:['one','operation','shared']});
  assert.deepEqual(selectView(v,v.graph.nodes[1].viewId,f.snapshot),{status:'candidate',qualifiedKey:['two','operation','shared']});
  assert.deepEqual(selectView(v,v.graph.nodes[0].viewId,{...f.snapshot,modelRevision:'R11'}),{status:'stale'});
});
test('ternary relation retains roles in both graph and relation table',()=>{
  const f=fixture(),v=project(f);
  assert.deepEqual(v.graph.edges.map(e=>e.role),['producer','output','evidence']);
  assert.deepEqual(v.table.relations[0].roles.map(r=>r.role),['producer','output','evidence']);
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces'},closure(f.snapshot)),{status:'present-in-snapshot'});
});
test('negative claim is limited to the proven closed scope',()=>{
  const f=fixture(),v=project(f);
  assert.deepEqual(boundedAbsence(v,missing,closure(f.snapshot)),{status:'absent-in-declared-snapshot',evidenceId:'synthetic-fixture'});
  assert.equal(boundedAbsence(v,{...missing,scope:'unbounded'},closure(f.snapshot)).status,'unknown');
});
test('a missing relation role makes absence unknown even with all subjects',()=>{
  const f=fixture(),v=project(f);
  v.graph.edges.pop();
  assert.deepEqual(boundedAbsence(v,missing,closure(f.snapshot)),{status:'unknown',reason:'role_coverage_loss'});
});
test('omitted nodes or relation node make absence unknown',()=>{
  const f=fixture(),v=project(f);
  v.graph.nodes.shift();
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).reason,'subject_coverage_loss');
  const w=project(f);w.graph.nodes.pop();
  assert.equal(boundedAbsence(w,missing,closure(f.snapshot)).reason,'relation_coverage_loss');
});
test('filtered, opaque, unpinned, undisclosed and incomplete views do not claim absence',()=>{
  const f=fixture(),v=project(f);
  for(const k of [{omitted:1},{opaque:1},{readerPinned:false},{disclosureQualified:false},{compositionComplete:false},{subjectsComplete:false},{relationsComplete:false},{rolesComplete:false}]){
    assert.equal(boundedAbsence(v,missing,{...closure(f.snapshot),...k}).status,'unknown');
  }
});
test('cross-revision Source witness and stale graph are rejected separately',()=>{
  const f=fixture(),v=project(f);
  assert.deepEqual(boundedAbsence(v,missing,closure({...f.snapshot,sourceIndexFingerprint:'S11'})),{status:'unknown',reason:'mixed_revision'});
  v.graph.revision='other';
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).reason,'mixed_revision');
});
test('duplicate qualified node or missing typed endpoint never becomes a graph fact',()=>{
  const f=fixture();f.subjects.push({...one});
  assert.throws(()=>project(f),/duplicate_subject/);
  const g=fixture();g.relations[0].roles[1].subject={...output,id:'missing'};
  assert.throws(()=>project(g),/missing_endpoint/);
});
test('layout only is not a semantic/Source change',()=>{
  const f=fixture(),v=project(f),w=structuredClone(v);
  w.graph.nodes[0].x=42;
  assert.deepEqual(boundedAbsence(w,missing,closure(f.snapshot)),boundedAbsence(v,missing,closure(f.snapshot)));
  assert.deepEqual(w.table,v.table);
});

test('a visible record is not a public claim when disclosure is unqualified',()=>{
  const f=fixture(),v=project(f);
  assert.deepEqual(boundedAbsence(v,{kind:'subject',scope:'one',type:'operation',id:'shared'},
    {...closure(f.snapshot),disclosureQualified:false}),
    {status:'unknown',reason:'disclosure_unqualified'});
});
test('invalid query and duplicate table occurrence do not manufacture negative certainty',()=>{
  const f=fixture(),v=project(f);
  assert.equal(boundedAbsence(v,{...missing,id:''},closure(f.snapshot)).reason,'invalid_query');
  v.table.subjects.push({...v.table.subjects[0]});
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).reason,'subject_coverage_loss');
});

test('selection refuses mismatched Graph/Table publication even with matching Source',()=>{
  const f=fixture(),v=project(f);
  v.table.revision='stale-table';
  assert.deepEqual(selectView(v,v.graph.nodes[0].viewId,f.snapshot),{status:'stale'});
});

test('qualified row drift cannot manufacture positive or absent claims',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  v.table.subjects[0].scope='two'; // retained key still says scope one
  assert.deepEqual(boundedAbsence(v,{kind:'subject',scope:'one',type:'operation',id:'shared'},w),
    {status:'unknown',reason:'subject_identity_mismatch'});
  assert.equal(boundedAbsence(v,{kind:'subject',scope:'one',type:'nonexistent'},w).status,'unknown');
  assert.deepEqual(selectView(v,v.graph.nodes[0].viewId,f.snapshot),{status:'unresolved'});
});

test('graph node ID and qualified key drift never rebind a visible node',()=>{
  const f=fixture(),v=project(f);
  v.graph.nodes[0].key=v.table.subjects[1].key; // wrong scope, old DOM identity retained
  assert.deepEqual(selectView(v,v.graph.nodes[0].viewId,f.snapshot),{status:'unresolved'});
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).status,'unknown');
});

test('graph duplicate cannot conceal a missing scoped node',()=>{
  const f=fixture(),v=project(f);
  v.graph.nodes[1]={...v.graph.nodes[0]};
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).reason,'subject_coverage_loss');
});

test('relation ID is part of exact query instead of any same-type match',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces',id:'flow'},w),
    {status:'present-in-snapshot'});
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces',id:'other'},w),
    {status:'absent-in-declared-snapshot',evidenceId:'synthetic-fixture'});
});

test('relation role and graph edge drift reject both positive and negative claims',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  v.table.relations[0].roles[1].role='relabelled';
  assert.equal(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces'},w).status,'unknown');
  const g=project(f);
  g.graph.edges[0].relationId='another-relation';
  assert.deepEqual(boundedAbsence(g,missing,w),{status:'unknown',reason:'role_coverage_loss'});
});

test('source relocation with same model fingerprint still makes old projection stale',()=>{
  const f=fixture(),v=project(f),moved={...f.snapshot,sourceIndexFingerprint:'S11'};
  assert.deepEqual(selectView(v,v.table.subjects[0].viewId,moved),{status:'stale'});
  assert.deepEqual(boundedAbsence(v,missing,closure(moved)),{status:'unknown',reason:'mixed_revision'});
});
