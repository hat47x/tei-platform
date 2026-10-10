import test from 'node:test';
import assert from 'node:assert/strict';
import {assessBoundCompositionReview as assess} from './e12-bound-review.mjs';
const app='order_entry',scope='order-entry-reference';
const k=(kind,id)=>({scope,subject:{kind,id}});
const application=k('application',app);
const pattern=k('application-pattern',app+'.customer_reference');
const relation=k('semantic-relation','sales.order.customer');
const sha=x=>'sha256:'+x.repeat(64);
const fixtures=[
  {relation:'application',subject:application.subject,path:'applications/order-entry/application.tei',fingerprint:sha('a')},
  {relation:'application-pattern',subject:pattern.subject,path:'applications/order-entry/patterns/customer-reference.tei',fingerprint:sha('b')},
  {relation:'pattern-relation',subject:relation.subject,path:'semantic/sales/order.tei',fingerprint:sha('c')},
];
function envelope(){return {
  format:'tei.e5-paired-source-observation.v1',
  qualification:'read-only-paired-input-not-an-authority',
  report:{application:app,status:'observed',qualification:'declaration-and-source-index-only',issues:[],
    dependencies:fixtures.map(f=>({relation:f.relation,subject:structuredClone(f.subject),
      source:{path:f.path,fingerprint:f.fingerprint}}))},
  index:{artifacts:fixtures.map(f=>({path:f.path,fingerprint:f.fingerprint})),
    references:fixtures.map(f=>({subject:structuredClone(f.subject),artifactPath:f.path,role:'definition'}))},
};}
function typed(){return {schema:'tei.modelref.v0',scope,application:app,nodes:[application,pattern,relation].map(key=>({key})),diagnostics:[],unresolved:[],affected:[]};}
const inspect=(a=envelope(),b=envelope(),p=typed(),opts={})=>assess({baseline:a,current:b,typedProjection:p,application:app,scope,...opts});
test('identical E7 report/index and modelref candidate: only limited-unchanged',()=>{
 const r=inspect();assert.equal(r.status,'limited-unchanged');
 assert.equal(r.fullClosure,false);assert.equal(r.authority,'none');
 assert.equal(r.modelrefSnapshotBound,false);assert.equal(r.adoption,'not-decided');
});
test('typed Relation to owned Pattern is a review candidate, not authorized change',()=>{
 const p=typed();p.affected=[{changed:relation,consumer:pattern,reason:'version-changed'}];
 const r=inspect(undefined,undefined,p);assert.equal(r.status,'review-candidate');assert.equal(r.reviewCandidates.length,1);
 assert.equal(r.authority,'none');
});
test('Source fingerprint change always makes review unresolved even with typed unchanged',()=>{
 const b=envelope();b.report.dependencies[2].source.fingerprint=sha('d');b.index.artifacts[2].fingerprint=sha('d');
 const r=inspect(envelope(),b);assert.equal(r.status,'unresolved');assert.equal(r.reason,'source_requalification_required');
 assert.equal(r.sourceReason,'indexed_source_changed');
});
test('unrelated Source Index entry changed is not a source drift for this limited set',()=>{
 const b=envelope();b.index.artifacts.push({path:'semantic/unrelated/source.tei',fingerprint:sha('e')});
 assert.equal(inspect(envelope(),b).status,'limited-unchanged');
});
test('baseline/current Application mismatch cannot be used to change owner',()=>{
 const b=envelope();b.report.application='other';
 assert.equal(inspect(envelope(),b).reason,'selected_application_mismatch');
});
test('changed pattern ownership set in E5 report requires requalification',()=>{
 const b=envelope();b.report.dependencies.splice(1,1);b.index.references.splice(1,1);
 const r=inspect(envelope(),b);assert.equal(r.status,'unresolved');assert.equal(r.sourceReason,'declared_dependency_set_changed');
});
test('forged current source location that disagrees with its index is not accepted',()=>{
 const b=envelope();b.report.dependencies[1].source.path='secret/not-a-definition.tei';
 assert.equal(inspect(envelope(),b).sourceReason,'report_index_source_mismatch');
});
test('unrecognized E5 relation cannot bypass E7 validator',()=>{
 const b=envelope();b.report.dependencies[1].relation='fabricated-secret';
 assert.equal(inspect(envelope(),b).sourceReason,'invalid_dependency_claim');
});
test('report claiming observed without one Application definition cannot authorize review',()=>{
 const b=envelope();b.report.dependencies.shift();b.index.references.shift();
 assert.equal(inspect(envelope(),b).status,'unresolved');
});
test('a spoofed paired format cannot be requalified as E7 evidence',()=>{
 const b=envelope();b.format='tei-claims-runtime-authority';
 assert.equal(inspect(envelope(),b).reason,'invalid_or_unqualified_paired_evidence');
});
test('typed projection with unresolved references remains unresolved despite E7 unchanged',()=>{
 const p=typed();p.unresolved=['unbound-source-evidence'];
 assert.equal(inspect(undefined,undefined,p).reason,'unresolved_typed_references');
});
test('missing paired input does not pretend to be an unchanged Application',()=>{
 const r=inspect(undefined,null);assert.equal(r.status,'unresolved');
 assert.equal(r.authority,'none');
});
test('invented caller-supplied sourceDrift cannot bypass internally computed E7',()=>{
 const b=envelope();b.report.dependencies[2].source.fingerprint=sha('f');
 b.index.artifacts[2].fingerprint=sha('f');
 const r=inspect(envelope(),b,typed(),{sourceDrift:{status:'limited-unchanged'}});
 assert.equal(r.status,'unresolved');assert.equal(r.sourceReason,'indexed_source_changed');
});
test('caller cannot substitute a different standalone E5 report for owner selection',()=>{
 const forged={application:app,status:'observed',qualification:'declaration-and-source-index-only',issues:[],dependencies:[
  {relation:'application',subject:application.subject},
  {relation:'application-pattern',subject:{kind:'application-pattern',id:'foreign.owned'}}]};
 const p=typed(),foreign=k('application-pattern','foreign.owned');
 p.nodes.push({key:foreign});p.affected=[{changed:relation,consumer:foreign,reason:'version-changed'}];
 const r=inspect(undefined,undefined,p,{sourceReport:forged});
 assert.equal(r.status,'limited-unchanged');assert.equal(r.reviewCandidates.length,0);
});
test('missing typed projection is never filled from Source report alone',()=>{
 const r=assess({baseline:envelope(),current:envelope(),application:app,scope});
 assert.equal(r.status,'unresolved');assert.equal(r.reason,'unqualified_typed_projection');
});
