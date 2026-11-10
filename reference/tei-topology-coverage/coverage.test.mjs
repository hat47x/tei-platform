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
  relations:[{scope:'one',id:'flow',type:'produces',roles:[{role:'producer',subject:one},{role:'output',subject:output},{role:'evidence',subject:evidence}]}],
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
  g.graph.edges[0].relationKey='["two","flow"]';
  assert.deepEqual(boundedAbsence(g,missing,w),{status:'unknown',reason:'role_coverage_loss'});
});

test('source relocation with same model fingerprint still makes old projection stale',()=>{
  const f=fixture(),v=project(f),moved={...f.snapshot,sourceIndexFingerprint:'S11'};
  assert.deepEqual(selectView(v,v.table.subjects[0].viewId,moved),{status:'stale'});
  assert.deepEqual(boundedAbsence(v,missing,closure(moved)),{status:'unknown',reason:'mixed_revision'});
});

test('relation ownership is independent of participant scope',()=>{
  const f=fixture();f.relations[0].roles[0].subject=two;
  const v=project(f),w=closure(f.snapshot);
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces',id:'flow'},w),
    {status:'present-in-snapshot'});
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'two',type:'produces',id:'flow'},w),
    {status:'absent-in-declared-snapshot',evidenceId:'synthetic-fixture'});
});

test('same relation ID in distinct scopes remains two qualified relations',()=>{
  const f=fixture();
  f.relations.push({scope:'two',id:'flow',type:'produces',roles:[
    {role:'producer',subject:two},{role:'output',subject:output}
  ]});
  const v=project(f),w=closure(f.snapshot);
  assert.notEqual(v.table.relations[0].key,v.table.relations[1].key);
  assert.equal(new Set(v.graph.nodes.map(n=>n.viewId)).size,v.graph.nodes.length);
  assert.equal(new Set(v.graph.edges.map(e=>e.viewId)).size,v.graph.edges.length);
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'two',type:'produces',id:'flow'},w),
    {status:'present-in-snapshot'});
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'two',type:'produces',id:'missing'},w),
    {status:'absent-in-declared-snapshot',evidenceId:'synthetic-fixture'});
});

test('relation scope omission and duplicate qualified relation are rejected',()=>{
  const f=fixture();delete f.relations[0].scope;
  assert.throws(()=>project(f),/invalid_relation_scope/);
  const g=fixture();g.relations.push({...g.relations[0]});
  assert.throws(()=>project(g),/duplicate_relation/);
});

test('post-projection relation ownership drift fails closed',()=>{
  const f=fixture(),v=project(f);
  v.table.relations[0].scope='two';
  assert.deepEqual(boundedAbsence(v,missing,closure(f.snapshot)),
    {status:'unknown',reason:'relation_identity_mismatch'});
  assert.deepEqual(selectView(v,v.graph.nodes[0].viewId,f.snapshot),{status:'unresolved'});
});

test('relation graph node cannot be rebound to same id at a different scope',()=>{
  const f=fixture(),v=project(f);
  v.graph.nodes.find(n=>n.relationKey).relationKey='["two","flow"]';
  assert.equal(boundedAbsence(v,missing,closure(f.snapshot)).reason,'relation_coverage_loss');
});

test('relation query cannot infer presence from an endpoint in another scope',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'two',type:'produces'},w),
    {status:'absent-in-declared-snapshot',evidenceId:'synthetic-fixture'});
  assert.equal(boundedAbsence(v,{kind:'subject',scope:'two',type:'operation',id:'shared'},w).status,
    'present-in-snapshot');
});

test('relation type must be preserved across Graph/Table instead of only a label',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  v.graph.nodes.find(n=>n.relationKey).type='consumes';
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces'},w),
    {status:'unknown',reason:'relation_coverage_loss'});
  assert.deepEqual(selectView(v,v.table.subjects[0].viewId,f.snapshot),{status:'unresolved'});
});

test('table relation type drift without matching graph cannot assert presence',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  v.table.relations[0].type='consumes';
  assert.deepEqual(boundedAbsence(v,{kind:'relation',scope:'one',type:'consumes'},w),
    {status:'unknown',reason:'relation_coverage_loss'});
});

test('unsupported predicates cannot forge a negative relation answer',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  // The declared relation belongs to scope one, but points to scope two.
  // A query about incident participants is NOT equivalent to relation ownership.
  for(const extra of [
    {participantScope:'two'},
    {role:'producer'},
    {where:{participantScope:'two'}},
    {window:{end:'2026-11-06'}},
    {predicate:'missing-record'},
  ]){
    const answer=boundedAbsence(v,{kind:'relation',scope:'two',type:'produces',...extra},w);
    assert.deepEqual(answer,{status:'unknown',reason:'unsupported_query_predicate'});
  }
});

test('unsupported predicates also block apparently positive relation and subject queries',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  assert.deepEqual(boundedAbsence(v,{
    kind:'relation',scope:'one',type:'produces',role:'receiver'
  },w),{status:'unknown',reason:'unsupported_query_predicate'});
  assert.deepEqual(boundedAbsence(v,{
    kind:'subject',scope:'one',type:'operation',id:'shared',where:{enabled:false}
  },w),{status:'unknown',reason:'unsupported_query_predicate'});
  // The same queries without extension keep their bounded fixture semantics.
  assert.equal(boundedAbsence(v,{kind:'relation',scope:'two',type:'produces'},w).status,
    'absent-in-declared-snapshot');
  assert.equal(boundedAbsence(v,{kind:'relation',scope:'one',type:'produces'},w).status,
    'present-in-snapshot');
});

test('nested predicate and empty selector do not silently degrade to a broad scan',()=>{
  const f=fixture(),v=project(f),w=closure(f.snapshot);
  assert.equal(boundedAbsence(v,{
    kind:'relation',scope:'one',type:'produces',id:'flow',subjectKey:['two','operation','shared']
  },w).reason,'unsupported_query_predicate');
  assert.equal(boundedAbsence(v,{
    kind:'relation',scope:'one',type:'produces',id:''
  },w).reason,'invalid_query');
});

test('undisclosed synthetic worlds do not expose distinct revision or coverage diagnostics',()=>{
  const f=fixture(),a=project(f),b=project(f);
  // Different private revisions and contents must give the same local result
  // when the fixture is not disclosure-qualified.
  b.table.subjects.pop();
  b.graph.revision='private-revision';
  const query={kind:'relation',scope:'one',type:'missing'};
  const denied={...closure(f.snapshot),disclosureQualified:false};
  assert.deepEqual(boundedAbsence(a,query,denied),
    {status:'unknown',reason:'disclosure_unqualified'});
  assert.deepEqual(boundedAbsence(b,query,denied),
    {status:'unknown',reason:'disclosure_unqualified'});
});

test('disclosure denial precedes private predicate and shape checks',()=>{
  const f=fixture(),v=project(f),denied={...closure(f.snapshot),disclosureQualified:false};
  assert.deepEqual(boundedAbsence(v,{
    kind:'relation',scope:'two',type:'produces',participantScope:'two'
  },denied),{status:'unknown',reason:'disclosure_unqualified'});
  assert.deepEqual(boundedAbsence({revision:'private-shape'},{
    kind:'relation',scope:'two',type:'produces'
  },denied),{status:'unknown',reason:'disclosure_unqualified'});
});
