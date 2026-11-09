import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';
import {EDIT_SCHEMA,LAYOUT_SCHEMA} from './dfd-view-layout.mjs';
const here = dirname(fileURLToPath(import.meta.url));
const sum = bytes => 'sha256:' + createHash('sha256').update(bytes).digest('hex');
const id='order-entry.approval-and-shipment-prep';
const dfd={schemaVersion:'business-dfd-projection-v0',authority:'derived-read-only',
  canonicalProcedureUsed:false,procedureId:id,steps:[{id:'APP-10',functionRefs:[]}],
  triggers:[],diagnostics:[],information:{roles:[],externalParties:[],stores:[]}};
function fixture(fn) {
  const dir=mkdtempSync(join(tmpdir(),'tei-dfd-layout-'));
  try {
    const p=join(dir,'projection.json'),l=join(dir,'layout.json'),e=join(dir,'edit.json');
    const projection=JSON.stringify(dfd)+'\n';
    const layout=JSON.stringify({schemaVersion:LAYOUT_SCHEMA,authority:'view-only',
      procedureId:id,projectionSHA256:sum(Buffer.from(projection)),nodes:[]})+'\n';
    writeFileSync(p,projection);writeFileSync(l,layout);
    const edit={schema:EDIT_SCHEMA,id:'move-test',origin:'studio',
      intent:'Move only view coordinates',procedureId:id,
      projectionSHA256:sum(Buffer.from(projection)),
      baseLayoutSHA256:sum(Buffer.from(layout)),
      operation:{kind:'move-node',nodeKind:'step',nodeId:'APP-10',x:10,y:20}};
    return fn({p,l,e,projection,layout,edit});
  } finally {rmSync(dir,{recursive:true,force:true});}
}
const run=(p,l,e)=>spawnSync(process.execPath,[join(here,'dfd-view-layout-cli.mjs'),
  '-projection',p,'-layout',l,'-edit',e],{encoding:'utf8'});
test('CLI hashes exact bytes and leaves all inputs unchanged',()=>fixture(({p,l,e,projection,layout,edit})=>{
  writeFileSync(e,JSON.stringify(edit));
  const output=run(p,l,e);
  assert.equal(output.status,0,output.stderr);
  const result=JSON.parse(output.stdout);
  assert.equal(result.inputSnapshotsVerified,true);
  assert.equal(result.canonicalMeaningChanged,false);
  assert.equal(result.readyForAcceptance,false);
  assert.deepEqual(result.sourceWrites,[]);
  assert.equal(result.proposedLayout.nodes[0].x,10);
  assert.equal(readFileSync(p,'utf8'),projection);
  assert.equal(readFileSync(l,'utf8'),layout);
}));
test('CLI refuses stale layout revision and unknown Step',()=>fixture(({p,l,e,edit})=>{
  writeFileSync(e,JSON.stringify({...edit,baseLayoutSHA256:'sha256:'+'0'.repeat(64)}));
  let output=run(p,l,e);
  assert.notEqual(output.status,0);
  assert.match(output.stderr,/STALE_OR_UNSUPPORTED_LAYOUT_EDIT/);
  assert.equal(output.stdout,'');
  writeFileSync(e,JSON.stringify({...edit,operation:{...edit.operation,nodeId:'unknown'}}));
  output=run(p,l,e);
  assert.notEqual(output.status,0);
  assert.match(output.stderr,/UNKNOWN_DFD_NODE/);
  assert.equal(output.stdout,'');
}));
