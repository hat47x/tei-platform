// Opt-in real-Go E2E: build cmd/tei, invoke the unchanged Node research runner.
// This test runs only in a trusted checkout with Go >= go.mod requirement.
// No Canonical writes, model training, cross-application disclosure or network calls by this test.
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join, resolve} from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';

const root=fileURLToPath(new URL('../../',import.meta.url));
const runner=fileURLToPath(new URL('./run-real-inspection.mjs',import.meta.url));
const workspace=resolve(root,'reference/workspace');
const optIn=process.env.TEI_REAL_CLI_E2E==='1';
const failMessage=(operation,result)=>
  operation+' failed (status '+String(result.status)+', signal '+String(result.signal)+')';

test('real Go inspect -> Node read-only projection -> metadata-only summary', {
  skip:!optIn && 'opt-in: set TEI_REAL_CLI_E2E=1 in an authorized checkout with Go 1.26.9',
  timeout:240000
},async t=>{
  const dir=await mkdtemp(join(tmpdir(),'tei-real-inspect-'));
  t.after(()=>rm(dir,{recursive:true,force:true}));
  const binary=join(dir,process.platform==='win32'?'tei.exe':'tei');
  const build=spawnSync('go',['build','-o',binary,'./cmd/tei'],{
    cwd:root,encoding:'utf8',timeout:180000,maxBuffer:1024*1024,
    env:{...process.env,GOTOOLCHAIN:'local'}
  });
  assert.equal(build.status,0,failMessage('Go build',build));

  // Calls five exact inspect Subjects in separate child processes with shell:false.
  // Workspaces must be explicitly authorized; the checked-in Reference is the only target.
  const run=spawnSync(process.execPath,[runner,'--tei-bin',binary,'--workspace',workspace],{
    cwd:root,encoding:'utf8',timeout:180000,maxBuffer:2*1024*1024
  });
  assert.equal(run.status,0,failMessage('Real TEI inspect batch',run));
  assert.equal(run.stderr,'');
  let evidence;
  assert.doesNotThrow(()=>{evidence=JSON.parse(run.stdout);});
  assert.equal(evidence.schemaVersion,'tei.research.inspection-evidence/v0');
  assert.equal(evidence.effect,'read-only');
  assert.equal(evidence.acceptance,'not_asserted');
  assert.equal(evidence.consistentSnapshot,true);
  assert.equal(evidence.subjectCount,5);
  assert.deepEqual(evidence.subjects.map(s=>s.kind),[
    'application','application-pattern','semantic-relation','semantic-data','application-pattern'
  ]);
  assert.equal(evidence.links.applicationPattern.status,'declared_membership');
  assert.equal(evidence.links.patternRelation.status,'declared_reference_match');
  assert.equal(evidence.links.applicationPattern.accepted,false);
  assert.equal(evidence.links.patternRelation.accepted,false);
  for(const forbidden of ['order_entry','sales.order','master_data.customer',
    'worldFingerprint','sourceIndexFingerprint','canonical','sources','contractQuery'])
    assert.equal(run.stdout.includes(forbidden),false,'unredacted field: '+forbidden);

  // Observe a real CLI error envelope separately; never convert errors to a candidate.
  const missing=spawnSync(binary,['inspect','-workspace',workspace,
    '-subject-kind','semantic-data','-subject-id','sales.not_found'],{
    cwd:root,encoding:'utf8',timeout:30000,maxBuffer:1024*1024
  });
  assert.equal(missing.status,1,failMessage('Missing Subject',missing));
  const failed=JSON.parse(missing.stdout);
  assert.equal(failed.schemaVersion,'tei.cli.inspect/v0');
  assert.equal(failed.ok,false);
  assert.equal(failed.error.code,'SUBJECT_NOT_FOUND');
  assert.equal(failed.result,undefined);
});
