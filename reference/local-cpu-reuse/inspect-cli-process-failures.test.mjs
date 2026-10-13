import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,writeFile,chmod,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';

// Actual subprocess boundary with a deliberately fake CLI; no Go/Workspace claim.
const runner=fileURLToPath(new URL('./run-real-inspection.mjs',import.meta.url));
const skip=process.platform==='win32';
async function withStub(body,check) {
  const dir=await mkdtemp(join(tmpdir(),'tei-cli-failure-'));
  const bin=join(dir,'synthetic-tei');
  try {
    await writeFile(bin,'#!/usr/bin/env node\n'+body+'\n',{mode:0o700});
    await chmod(bin,0o700);
    const r=spawnSync(process.execPath,[runner,'--tei-bin',bin,
      '--workspace',join(dir,'workspace;not-shell')],
      {encoding:'utf8',timeout:30000,maxBuffer:1024*1024});
    check(r);
  } finally {await rm(dir,{recursive:true,force:true});}
}
function failed(r,code) {
  assert.equal(r.status,1);
  assert.equal(r.stdout,'');
  assert.equal(r.stderr.trim(),code);
  assert.equal(r.stderr.includes('SYNTHETIC_PRIVATE_FIELD'),false);
}

test('child exit code masks its private stderr', {skip},async()=>{
  await withStub("process.stderr.write('SYNTHETIC_PRIVATE_FIELD');process.exit(7);",
    r=>failed(r,'TEI_INSPECT_EXECUTION_FAILED'));
});
test('child invalid JSON never leaks its stdout', {skip},async()=>{
  await withStub("process.stdout.write('SYNTHETIC_PRIVATE_FIELD');",
    r=>failed(r,'TEI_INSPECT_INVALID_JSON'));
});
test('success with unexpected child stderr is refused', {skip},async()=>{
  await withStub("process.stderr.write('SYNTHETIC_PRIVATE_FIELD');process.stdout.write('{}');",
    r=>failed(r,'TEI_INSPECT_UNEXPECTED_STDERR'));
});
test('oversized synchronous child stdout is refused before parsing', {skip},async()=>{
  await withStub("require('node:fs').writeFileSync(1,'x'.repeat(2100000));",
    r=>failed(r,'TEI_INSPECT_EXECUTION_FAILED'));
});
