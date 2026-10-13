import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,readFile,writeFile,chmod,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';

const runner=fileURLToPath(new URL('./run-real-inspection.mjs',import.meta.url));
const fixture=fileURLToPath(new URL('./fixtures/synthetic-inspect-cli.cjs',import.meta.url));
const skip=process.platform==='win32';
async function runFixture(drift=false) {
  const root=await mkdtemp(join(tmpdir(),'tei-cli-success-'));
  const binary=join(root,'synthetic-tei');
  const workspace=join(root,'workspace;not-a-shell');
  const log=join(root,'subjects.log');
  try {
    await writeFile(binary,'#!/usr/bin/env node\n'+await readFile(fixture,'utf8'),{mode:0o700});
    await chmod(binary,0o700);await writeFile(log,'');
    const env={...process.env,TEI_FIXTURE_WORKSPACE:workspace,TEI_FIXTURE_LOG:log,TEI_FIXTURE_DRIFT:drift?'1':'0'};
    const r=spawnSync(process.execPath,[runner,'--tei-bin',binary,'--workspace',workspace],
      {encoding:'utf8',env,timeout:30000,maxBuffer:1024*1024});
    return {r,calls:(await readFile(log,'utf8')).trim().split('\n')};
  } finally {await rm(root,{recursive:true,force:true});}
}

test('real child processes observe five fixed selectors and emit no Source identifiers', {skip},async()=>{
  const {r,calls}=await runFixture();
  assert.equal(r.status,0,r.stderr);
  const summary=JSON.parse(r.stdout);
  assert.equal(summary.subjectCount,5);assert.equal(summary.consistentSnapshot,true);
  assert.equal(summary.links.applicationPattern.status,'declared_membership');
  assert.equal(summary.links.patternRelation.status,'declared_reference_match');
  for(const sensitive of ['order_entry','sales.order','internal/only.tei','world:w','index:i'])
    assert.equal(r.stdout.includes(sensitive),false);
  assert.deepEqual(calls,[
    'application:order_entry','application-pattern:order_entry.order_lines',
    'semantic-relation:sales.order.lines','semantic-data:sales.order',
    'application-pattern:order_entry.customer_reference']);
});
test('third child with differing World fingerprint aborts without partial output', {skip},async()=>{
  const {r,calls}=await runFixture(true);
  assert.equal(r.status,1);assert.equal(r.stdout,'');
  assert.equal(r.stderr.trim(),'SNAPSHOT_DRIFT');assert.equal(calls.length,3);
});
