import test from 'node:test';
import assert from 'node:assert/strict';
import {qualifyCompositionReview as qualify, qualifyPairedCompositionReview as qualifyPaired} from './e11-review-gate.mjs';
const scope='order-entry-reference',app='order_entry';
const k=(kind,id,at=scope)=>({scope:at,subject:{kind,id}});
const appKey=k('application',app),pat=k('application-pattern',app+'.customer_reference');
const relation=k('semantic-relation','sales.order.customer');
const other=k('application-pattern','repair_intake.lookup');
const src={status:'limited-unchanged',qualification:'application-declaration-and-direct-source-definitions-only',fullClosure:false,authority:'none',dependencyCount:12};
const typed=()=>({schema:'tei.modelref.v0',scope,application:app,nodes:[appKey,pat,relation,other].map(key=>({key})),diagnostics:[],unresolved:[],affected:[]});
const sourceReport=()=>({application:app,status:'observed',
 qualification:'declaration-and-source-index-only',issues:[],dependencies:[
  {relation:'application',subject:{kind:'application',id:app}},
  {relation:'application-pattern',subject:{kind:'application-pattern',id:pat.subject.id}},
 ]});
const check=(s=src,t=typed(),params={})=>qualify({
 sourceDrift:s,sourceReport:sourceReport(),typedProjection:t,application:app,scope,...params});
test('read-only unchanged observations cannot become full closure or authorization',()=>{
 const r=check();assert.equal(r.status,'limited-unchanged');assert.equal(r.authority,'none');
 assert.equal(r.fullClosure,false);assert.equal(r.adoption,'not-decided');
});
test('typed impact on a selected Pattern requires targeted review',()=>{
 const t=typed();t.affected=[{changed:relation,consumer:pat,reason:'reference-version-changed'}];
 const r=check(src,t);assert.equal(r.status,'review-candidate');assert.equal(r.reviewCandidates.length,1);
 assert.equal(r.authority,'none');
});
test('typed impact reaching selected Application also demands review',()=>{
 const t=typed();t.affected=[{changed:relation,consumer:appKey,reason:'reference-version-changed'}];
 assert.equal(check(src,t).status,'review-candidate');
});
test('impact on an unrelated scoped consumer is not declared an effect on selected Application',()=>{
 const t=typed();t.affected=[{changed:relation,consumer:other,reason:'version-changed'}];
 const r=check(src,t);assert.equal(r.status,'limited-unchanged');assert.equal(r.reviewCandidates.length,0);
 assert.equal(r.fullClosure,false);
});
test('Source Index drift remains unresolved despite well-formed typed impact',()=>{
 const s={...src,status:'unresolved',reason:'indexed_source_changed'},t=typed();
 t.affected=[{changed:relation,consumer:pat,reason:'version-changed'}];
 const r=check(s,t);assert.equal(r.status,'unresolved');assert.equal(r.sourceReason,'indexed_source_changed');
 assert.equal(r.reviewCandidates.length,1);
});
test('typed unresolved dependencies forbid a false unchanged verdict',()=>{
 const t=typed();t.unresolved=['source-evidence-not-bound'];
 assert.equal(check(src,t).reason,'unresolved_typed_references');
});
test('typed Validate diagnostics forbid treating candidate graph as qualified',()=>{
 const t=typed();t.diagnostics=[{code:'missing-reference',at:'pattern'}];
 assert.equal(check(src,t).reason,'unresolved_typed_references');
});
test('missing selected Application in Graph cannot take another Application',()=>{
 const t=typed();t.nodes=t.nodes.filter(n=>n.key.subject.kind!=='application');
 assert.equal(check(src,t).reason,'application_not_in_typed_projection');
});
test('wrong scope / malformed identity / duplicated node are rejected',()=>{
 const t=typed();t.nodes.push({key:k('semantic-data','sales.order','other-scope')});
 assert.equal(check(src,t).reason,'ambiguous_typed_node');
 const u=typed();u.nodes.push({key:appKey});assert.equal(check(src,u).reason,'ambiguous_typed_node');
 const v=typed();v.affected=[{changed:k('semantic-relation','sales.order.customer','alien'),consumer:pat,reason:'x'}];
 assert.equal(check(src,v).reason,'invalid_typed_impact');
});
test('impact must refer to known nodes and cannot be duplicated',()=>{
 const t=typed();t.affected=[{changed:k('semantic-data','missing'),consumer:pat,reason:'version-changed'}];
 assert.equal(check(src,t).reason,'invalid_typed_impact');
 const u=typed(),entry={changed:relation,consumer:pat,reason:'version-changed'};
 u.affected=[entry,structuredClone(entry)];assert.equal(check(src,u).reason,'duplicate_typed_impact');
});
test('forged authoritative Source report and mismatched projection are refused',()=>{
 assert.equal(check({...src,authority:'runtime-allowed'}).reason,'unqualified_source_observation');
 assert.equal(check({...src,fullClosure:true}).reason,'unqualified_source_observation');
 assert.equal(check(src,{...typed(),application:'other'}).reason,'unqualified_typed_projection');
 assert.equal(check({...src,qualification:'complete-closure'}).reason,'unqualified_source_observation');
});
test('unspecified inputs and unverified graph do not infer deployment permission',()=>{
 const r=qualify({});assert.equal(r.status,'unresolved');assert.equal(r.authority,'none');
 const t=typed();t.schema='other';assert.equal(check(src,t).reason,'unqualified_typed_projection');
});

test('a Pattern with matching ID prefix is excluded unless E5 owns it',()=>{
 const t=typed(),impostor=k('application-pattern',app+'.unowned');
 t.nodes.push({key:impostor});
 t.affected=[{changed:relation,consumer:impostor,reason:'reference-version-changed'}];
 const r=check(src,t);
 assert.equal(r.status,'limited-unchanged');assert.equal(r.reviewCandidates.length,0);
});
test('a Pattern with different spelling belongs when typed E5 Application owns it',()=>{
 const t=typed();
 t.affected=[{changed:relation,consumer:other,reason:'reference-version-changed'}];
 const report=sourceReport();
 report.dependencies.push({relation:'application-pattern',subject:{kind:'application-pattern',id:other.subject.id}});
 const r=check(src,t,{sourceReport:report});
 assert.equal(r.status,'review-candidate');
 assert.equal(r.reviewCandidates.length,1);
 assert.equal(r.fullClosure,false);assert.equal(r.authority,'none');
});
test('missing or mismatched E5 Application witness cannot use naming convention',()=>{
 const t=typed();t.affected=[{changed:relation,consumer:pat,reason:'version-changed'}];
 assert.equal(check(src,t,{sourceReport:undefined}).reason,'unqualified_application_source_report');
 const report=sourceReport();report.application='repair_intake';
 assert.equal(check(src,t,{sourceReport:report}).reason,'unqualified_application_source_report');
 const incomplete=sourceReport();incomplete.dependencies.shift();
 assert.equal(check(src,t,{sourceReport:incomplete}).reason,'unqualified_application_source_report');
});
test('E5 duplicate Pattern owner or unresolved Report refuses a selected winner',()=>{
 const duplicate=sourceReport();
 duplicate.dependencies.push(structuredClone(duplicate.dependencies[1]));
 assert.equal(check(src,typed(),{sourceReport:duplicate}).reason,'unqualified_application_source_report');
 const bad=sourceReport();bad.status='unresolved';bad.issues=[{reason:'missing_definition'}];
 assert.equal(check(src,typed(),{sourceReport:bad}).reason,'unqualified_application_source_report');
});
test('missing declared Pattern in projection prevents a false unchanged result',()=>{
 const t=typed();t.nodes=t.nodes.filter(n=>JSON.stringify(n.key)!==JSON.stringify(pat));
 assert.equal(check(src,t).reason,'declared_pattern_missing_from_projection');
});
test('unresolved Source drift still remains unresolved with explicit Pattern membership',()=>{
 const t=typed();t.affected=[{changed:relation,consumer:pat,reason:'version-changed'}];
 const r=check({...src,status:'unresolved',reason:'source_fingerprint_changed'},t);
 assert.equal(r.status,'unresolved');
 assert.equal(r.sourceReason,'source_fingerprint_changed');
 assert.equal(r.reviewCandidates.length,1);
});

const sha = x => 'sha256:'+x.repeat(64);
const paired = () => ({
 format:'tei.e5-paired-source-observation.v1',
 qualification:'read-only-paired-input-not-an-authority',
 report:{application:app,status:'observed',qualification:'declaration-and-source-index-only',
  issues:[],dependencies:[
   {relation:'application',subject:{kind:'application',id:app},
    source:{path:'applications/order-entry/application.tei',fingerprint:sha('a')}},
   {relation:'application-pattern',subject:{kind:'application-pattern',id:pat.subject.id},
    source:{path:'applications/order-entry/patterns/customer-reference.tei',fingerprint:sha('b')}},
  ]},
 index:{artifacts:[
  {path:'applications/order-entry/application.tei',fingerprint:sha('a')},
  {path:'applications/order-entry/patterns/customer-reference.tei',fingerprint:sha('b')},
 ],references:[
  {subject:{kind:'application',id:app},
   artifactPath:'applications/order-entry/application.tei',role:'definition'},
  {subject:{kind:'application-pattern',id:pat.subject.id},
   artifactPath:'applications/order-entry/patterns/customer-reference.tei',role:'definition'},
 ]},
});
const goProjection=()=>({
 graph:{schema:'tei.modelref.v0',nodes:typed().nodes,edges:[]},
 rules:[],unresolved:[],authorityLimits:['source-evidence-not-bound'],
});
const pairedReview=({baseline=paired(),current=paired(),projection=goProjection(),
 diagnostics=[],affected=[],...rest}={})=>qualifyPaired({
 baseline,current,projection,diagnostics,affected,application:app,scope,...rest,
});

test('actual E7 and Go derivedbridge JSON shapes cannot assert a shared snapshot',()=>{
 const result=pairedReview();
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'unverified_shared_snapshot');
 assert.equal(result.fullClosure,false);
 assert.equal(result.authority,'none');
 assert.equal(result.adoption,'not-decided');
 assert.equal(result.reviewCandidates.length,0);
});
test('Go Impact candidates survive as review leads without claiming same snapshot',()=>{
 const result=pairedReview({affected:[
  {changed:relation,consumer:pat,reason:'reference-version-changed'},
 ]});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'unverified_shared_snapshot');
 assert.equal(result.reviewCandidates.length,1);
 assert.equal(result.reviewCandidates[0].consumer.subject.id,pat.subject.id);
});
test('E7 actual Source fingerprint mismatch takes precedence over typed review',()=>{
 const current=paired();
 current.report.dependencies[1].source.fingerprint=sha('c');
 current.index.artifacts[1].fingerprint=sha('c');
 const result=pairedReview({current,affected:[
  {changed:relation,consumer:pat,reason:'version-changed'},
 ]});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'source_requalification_required');
 assert.equal(result.sourceReason,'indexed_source_changed');
 assert.equal(result.reviewCandidates.length,1);
});
test('a missing or inconsistent E5 report refuses typed review candidates',()=>{
 const current=paired();delete current.report;
 assert.equal(pairedReview({current}).reason,'incomplete_paired_or_typed_evidence');
 const changed=paired();changed.report.application='repair_intake';
 assert.equal(pairedReview({current:changed}).reason,'unqualified_application_source_report');
});
test('missing Go graph outputs and malformed projection cannot be treated as empty impact',()=>{
 assert.equal(pairedReview({projection:{graph:{schema:'tei.modelref.v0'}}}).reason,
  'incomplete_paired_or_typed_evidence');
 assert.equal(pairedReview({diagnostics:null}).reason,'incomplete_paired_or_typed_evidence');
 const invalid=goProjection();invalid.graph.schema='tei.modelref.fake';
 assert.equal(pairedReview({projection:invalid}).reason,'unqualified_typed_projection');
});
test('Go-derived projection with unresolved references cannot return unchanged',()=>{
 const projection=goProjection();projection.unresolved=['unmaterialized-to:semantic-relation:sales.order.customer'];
 assert.equal(pairedReview({projection}).reason,'unresolved_typed_references');
 const diagnostics=[{code:'invalid-edge-type',at:'edges/0'}];
 assert.equal(pairedReview({diagnostics}).reason,'unresolved_typed_references');
});
test('typed impact on an unowned Pattern is excluded even when it shares prefix',()=>{
 const projection=goProjection();
 const counterfeit=k('application-pattern',app+'.impostor');
 projection.graph.nodes.push({key:counterfeit});
 const result=pairedReview({projection,affected:[
  {changed:relation,consumer:counterfeit,reason:'version-changed'},
 ]});
 assert.equal(result.reason,'unverified_shared_snapshot');
 assert.equal(result.reviewCandidates.length,0);
});
test('inconsistent current E7 definition source cannot be promoted by Go shape',()=>{
 const current=paired();
 current.report.dependencies[1].source.path='private/unindexed.tei';
 const result=pairedReview({current});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'source_requalification_required');
 assert.equal(result.sourceReason,'report_index_source_mismatch');
});
