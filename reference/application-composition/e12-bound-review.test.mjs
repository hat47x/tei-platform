import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {mkdtempSync,rmSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {inspectEvidenceFiles} from './e12-bound-review.mjs';

const app='order_entry',scope='order-entry-reference';
const key=(kind,id)=>({scope,subject:{kind,id}});
const application=key('application',app);
const pattern=key('application-pattern',app+'.customer_reference');
const relation=key('semantic-relation','sales.order.customer');
const sha=c=>'sha256:'+c.repeat(64);
const f=[{relation:'application',subject:application.subject,path:'applications/order-entry/application.tei',fingerprint:sha('a')},
 {relation:'application-pattern',subject:pattern.subject,path:'applications/order-entry/patterns/customer-reference.tei',fingerprint:sha('b')}];
const paired=()=>({format:'tei.e5-paired-source-observation.v1',
 qualification:'read-only-paired-input-not-an-authority',
 report:{application:app,status:'observed',qualification:'declaration-and-source-index-only',issues:[],
  dependencies:f.map(x=>({relation:x.relation,subject:structuredClone(x.subject),source:{path:x.path,fingerprint:x.fingerprint}}))},
 index:{artifacts:f.map(x=>({path:x.path,fingerprint:x.fingerprint})),
  references:f.map(x=>({subject:structuredClone(x.subject),artifactPath:x.path,role:'definition'}))}});
const modelref=()=>({projection:{graph:{schema:'tei.modelref.v0',nodes:[application,pattern,relation].map(k=>({key:k})),edges:[]},
 rules:[],unresolved:[],authorityLimits:['source-evidence-not-bound']},diagnostics:[],affected:[]});
const cmd=fileURLToPath(new URL('./e12-bound-review.mjs',import.meta.url));
const writeInputs=(fn, {baseline=paired(),current=paired(),typed=modelref()}={})=>{
 const folder=mkdtempSync(join(tmpdir(),'tei-e12-'));
 const files=['baseline.json','current.json','modelref.json'].map(name=>join(folder,name));
 [baseline,current,typed].forEach((x,i)=>writeFileSync(files[i],JSON.stringify(x)));
 try {return fn({files,folder});}finally {rmSync(folder,{recursive:true,force:true});}
};
const run=(params={})=>writeInputs(({files})=>spawnSync(process.execPath,[cmd,...files,app,scope],{encoding:'utf8'}),params);

test('CLI consumes E7 paired Source and Go-shaped modelref; never certifies shared snapshot',()=>{
 const r=run(),out=JSON.parse(r.stdout);
 assert.equal(r.status,2);assert.equal(out.status,'unresolved');
 assert.equal(out.reason,'unverified_shared_snapshot');assert.equal(out.authority,'none');
 assert.equal(out.fullClosure,false);assert.equal(out.adoption,'not-decided');
});
test('typed Relation impact yields a review lead while rejecting success',()=>{
 const t=modelref();t.affected=[{changed:relation,consumer:pattern,reason:'version-changed'}];
 const r=run({typed:t}),out=JSON.parse(r.stdout);
 assert.equal(r.status,2);assert.equal(out.reviewCandidates.length,1);
 assert.equal(out.reviewCandidates[0].consumer.subject.id,pattern.subject.id);
});
test('Source definition fingerprint drift wins over an unchanged Go projection',()=>{
 const current=paired();current.report.dependencies[1].source.fingerprint=sha('f');current.index.artifacts[1].fingerprint=sha('f');
 const r=run({current}),out=JSON.parse(r.stdout);
 assert.equal(out.status,'unresolved');assert.equal(out.sourceReason,'indexed_source_changed');
});
test('changing Application identity in one envelope never silently chooses a winner',()=>{
 const current=paired();current.report.application='other';
 const out=JSON.parse(run({current}).stdout);
 assert.equal(out.status,'unresolved');
});
test('malformed index to E5 definition cannot become an E11 review success',()=>{
 const current=paired();current.report.dependencies[1].source.path='private/other.tei';
 const out=JSON.parse(run({current}).stdout);
 assert.equal(out.status,'unresolved');assert.equal(out.sourceReason,'report_index_source_mismatch');
});
test('unrecognized E5 dependency relation is not accepted',()=>{
 const current=paired();current.report.dependencies[1].relation='fabricated-secret';
 const out=JSON.parse(run({current}).stdout);
 assert.equal(out.sourceReason,'invalid_dependency_claim');
});
test('Go projection missing selected Pattern fails closed',()=>{
 const typed=modelref();typed.projection.graph.nodes.splice(1,1);
 const out=JSON.parse(run({typed}).stdout);
 assert.equal(out.reason,'declared_pattern_missing_from_projection');
});
test('Go unresolved and Validate diagnostics stop review regardless of E7 input',()=>{
 const t=modelref();t.projection.unresolved=['missing-reference'];
 assert.equal(JSON.parse(run({typed:t}).stdout).reason,'unresolved_typed_references');
 const other=modelref();other.diagnostics=[{code:'version-mismatch',at:'pattern'}];
 assert.equal(JSON.parse(run({typed:other}).stdout).reason,'unresolved_typed_references');
});
test('extra JSON value with secret token is not echoed to standard output',()=>{
 const current=paired();current.unexpectedCredential='LEAK_TEST_SECRET_123';
 const result=run({current});
 assert.ok(!result.stdout.includes('LEAK_TEST_SECRET_123'));
 assert.ok(!result.stderr.includes('LEAK_TEST_SECRET_123'));
});
test('invalid/missing JSON exits 1 and never echoes source filename or raw JSON',()=>{
 writeInputs(({files})=>{
  writeFileSync(files[1],'{secretBrokenJson');
  const r=spawnSync(process.execPath,[cmd,...files,app,scope],{encoding:'utf8'});
  assert.equal(r.status,1);assert.equal(r.stdout,'');
  assert.ok(!r.stderr.includes(files[1]));assert.ok(!r.stderr.includes('secretBrokenJson'));
 });
});
test('missing CLI arguments return usage error before source file reads',()=>{
 const r=spawnSync(process.execPath,[cmd],{encoding:'utf8'});
 assert.equal(r.status,1);assert.match(r.stderr,/Usage/);assert.equal(r.stdout,'');
});
test('oversized input is rejected before JSON parsing and never echoed',()=>{
 writeInputs(({files})=>{
  writeFileSync(files[2], 'x'.repeat(16*1024*1024+1));
  const r=spawnSync(process.execPath,[cmd,...files,app,scope],{encoding:'utf8'});
  assert.equal(r.status,1);assert.equal(r.stdout,'');assert.match(r.stderr,/Invalid review evidence input/);
 });
});
test('library API rejects missing inputs as errors, not positive evidence',async()=>{
 await assert.rejects(()=>inspectEvidenceFiles({application:app,scope}),/invalid_arguments/);
});
