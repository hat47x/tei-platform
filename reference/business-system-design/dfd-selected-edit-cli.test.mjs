import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {BINDING_SCHEMA} from './dfd-selected-edit-bridge.mjs';

const root=dirname(fileURLToPath(import.meta.url));
const sha=buf=>'sha256:'+createHash('sha256').update(buf).digest('hex');
const read=name=>JSON.parse(readFileSync(join(root,name),'utf8'));
const procedure=read('dfd-order-approval/order-approval-procedure.json');
const review=read('dfd-order-approval/review-proposal.json');
const projection={
  schemaVersion:'business-dfd-projection-v0',authority:'derived-read-only',
  canonicalProcedureUsed:false,procedureId:procedure.procedure.id,diagnostics:[],
  steps:procedure.steps.map(s=>({id:s.id,label:s.businessAction,realization:s.realization,
    functionRefs:(s.applicationPatterns||[]).map(id=>({id,status:'resolved'}))})),
  triggers:review.triggers.map(x=>({...x})),
  flows:review.flows.map(x=>({...x})),
  information:{
    roles:review.businessRoles.map(x=>({...x})),
    externalParties:review.externalParties.map(x=>({...x})),
    stores:review.stores.map(x=>({...x})),
    responsibilities:review.responsibilities.map(x=>({...x})),
    informationFlows:review.informationFlows.map(x=>({...x}))
  }
};
function scratch(run) {
  const dir=mkdtempSync(join(tmpdir(),'tei-dfd-selected-'));
  try {
    const paths={p:join(dir,'projection.json'),s:join(dir,'procedure.json'),
      r:join(dir,'review.json'),b:join(dir,'binding.json')};
    const buffers={
      p:Buffer.from(JSON.stringify(projection)+'\n'),
      s:Buffer.from(JSON.stringify(procedure)+'\n'),
      r:Buffer.from(JSON.stringify(review)+'\n')
    };
    for(const [k,value] of Object.entries(buffers))writeFileSync(paths[k],value);
    const binding={
      schema:BINDING_SCHEMA,id:'EDIT-1',origin:'cli',intent:'業務上の起動契機の説明を見直す',
      projectionSHA256:sha(buffers.p),procedureSHA256:sha(buffers.s),
      reviewSHA256:sha(buffers.r),
      selection:{kind:'trigger',id:'TR-ORDER'},
      operation:{kind:'replace-trigger-label',trigger:'TR-ORDER',label:'注文の確認待ち'}
    };
    writeFileSync(paths.b,JSON.stringify(binding)+'\n');
    return run({paths,binding,buffers});
  }finally{rmSync(dir,{recursive:true,force:true});}
}
function execute(paths){
  return spawnSync(process.execPath,[join(root,'dfd-selected-edit-cli.mjs'),
    '-projection',paths.p,'-procedure',paths.s,
    '-review',paths.r,'-binding',paths.b],{encoding:'utf8'});
}
test('byte-pinned selected review preview has no Source writes',()=>scratch(({paths,buffers})=>{
  const result=execute(paths);
  assert.equal(result.status,0,result.stderr);
  const parsed=JSON.parse(result.stdout);
  assert.equal(parsed.inputSnapshotsVerified,true);
  assert.equal(parsed.inputSnapshots.projection,sha(buffers.p));
  assert.equal(parsed.inputSnapshots.review,sha(buffers.r));
  assert.equal(parsed.selected.id,'TR-ORDER');
  assert.equal(parsed.proposedReview.triggers[0].label,'注文の確認待ち');
  assert.equal(parsed.readyForAcceptance,false);
  assert.equal(parsed.sharedAcceptanceCandidate,null);
  assert.deepEqual(parsed.sourceWrites,[]);
  assert.deepEqual(readFileSync(paths.r),buffers.r);
}));
test('stale review contents are refused even if the edit still names a valid trigger',()=>scratch(({paths})=>{
  const reviewNext=JSON.parse(readFileSync(paths.r,'utf8'));
  reviewNext.triggers[0].label='edited after selection';
  writeFileSync(paths.r,JSON.stringify(reviewNext)+'\n');
  const result=execute(paths);
  assert.notEqual(result.status,0);
  assert.match(result.stderr,/STALE_OR_UNVERIFIED_DFD_EDIT_INPUT_SNAPSHOT/);
  assert.equal(result.stdout,'');
}));
test('a projection inconsistent with the actual review is refused after legitimate rehash',()=>scratch(({paths,binding})=>{
  const projectionNext=JSON.parse(readFileSync(paths.p,'utf8'));
  projectionNext.triggers[0].label='stale projection';
  const bytes=Buffer.from(JSON.stringify(projectionNext)+'\n');
  writeFileSync(paths.p,bytes);
  writeFileSync(paths.b,JSON.stringify({...binding,projectionSHA256:sha(bytes)})+'\n');
  const result=execute(paths);
  assert.notEqual(result.status,0);
  assert.match(result.stderr,/DFD_REVIEW_SNAPSHOT_MISMATCH_TRIGGERS/);
  assert.equal(result.stdout,'');
}));
test('selected element cannot be substituted for another valid Step',()=>scratch(({paths,binding})=>{
  writeFileSync(paths.b,JSON.stringify({...binding,selection:{kind:'step',id:'APP-40'}})+'\n');
  const result=execute(paths);
  assert.notEqual(result.status,0);
  assert.match(result.stderr,/DFD_SELECTION_EDIT_TARGET_MISMATCH/);
  assert.equal(result.stdout,'');
}));
