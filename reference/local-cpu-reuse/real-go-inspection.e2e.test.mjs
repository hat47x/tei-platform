// Opt-in real-Go E2E: build cmd/tei, invoke the unchanged Node research runner.
// This test runs only in a trusted checkout with Go >= go.mod requirement.
// No Canonical writes, model training or cross-application disclosure. Go build may fetch modules.
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join, resolve} from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {qualifyReferenceQuerySignature} from './qualify-query-signature.mjs';
import {observeReferenceProviderRouting} from './observe-provider-routing.mjs';
import {compareTwoReferenceApplications,proposeReferenceQueryAdapter,
  TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

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

  // Two actual Reference Applications in the SAME explicitly chosen public workspace:
  // observed declaration equality is not semantic implementation equivalence.
  const observations=TWO_APPLICATION_REFERENCE_SUBJECTS.map(subject=>{
    const command=spawnSync(binary,['inspect','-workspace',workspace,
      '-subject-kind',subject.kind,'-subject-id',subject.id],{
      cwd:root,encoding:'utf8',timeout:30000,maxBuffer:1024*1024
    });
    assert.equal(command.status,0,failMessage('Two-Application inspect',command));
    assert.equal(command.stderr,'');
    return JSON.parse(command.stdout);
  });
  const comparison=compareTwoReferenceApplications(observations);
  assert.equal(comparison.schemaVersion,'tei.research.cross-application-reference/v0');
  assert.equal(comparison.sharedInvocationDeclared,true);
  assert.equal(comparison.observation,'shared_query_invocation_declared');
  assert.equal(comparison.reuseDecision,'needs_semantic_review');
  assert.equal(comparison.acceptance,'not_asserted');
  assert.equal(comparison.contractConformance,'not_verified');
  assert.equal(comparison.declaredQuery.contract,'master_data.customer');
  assert.equal(comparison.declaredQuery.query,'find_by_id');
  assert.equal(comparison.declaredQuery.compatibleMajor,2);
  assert.deepEqual(comparison.contextDifferences,['relation','datasourceAlias']);
  const proposal=proposeReferenceQueryAdapter(observations);
  assert.equal(proposal.status,'candidate_requires_semantic_review');
  assert.equal(proposal.commonInvocation.contract,'master_data.customer');
  assert.equal(proposal.commonInvocation.query,'find_by_id');
  assert.equal(proposal.commonInvocation.compatibleMajor,2);
  assert.deepEqual(proposal.applicationContext.map(x=>x.datasourceAlias),
    ['master','customer_master']);
  assert.deepEqual(proposal.applicationContext.map(x=>x.relation),
    ['sales.order.customer','repair_intake.intake.customer']);
  assert.equal(proposal.commonInvocation.datasourceAlias,undefined);
  assert.equal(proposal.commonInvocation.relation,undefined);
  assert.equal(proposal.executionPermitted,false);
  assert.equal(proposal.implementationSelected,false);
  assert.equal(proposal.accepted,false);
  assert.equal(proposal.verified.contractSemantics,false);
  assert.equal(proposal.verified.readAuthorization,false);

  // Existing Go 'world' command returns normalized M0. Keep the entire World in memory;
  // never print it, and check Application/Pattern overlap with the separate inspect reads.
  const normalized=spawnSync(binary,['world','-workspace',workspace],{
    cwd:root,encoding:'utf8',timeout:30000,maxBuffer:2*1024*1024
  });
  assert.equal(normalized.status,0,failMessage('Normalized World read',normalized));
  assert.equal(normalized.stderr,'');
  const worldEnvelope=JSON.parse(normalized.stdout);
  // #2354: tei read commands emit one tei.cli.<command>/v0 envelope; unwrap at the consume boundary.
  if(worldEnvelope.schemaVersion!=='tei.cli.world/v0'||worldEnvelope.ok!==true||
    worldEnvelope.result===undefined)
    throw new Error('INVALID_TEI_RESULT: normalized world envelope');
  const world=worldEnvelope.result;
  const typed=qualifyReferenceQuerySignature(world,observations);
  assert.equal(typed.status,'declared_signature_matched');
  assert.equal(typed.contract.id,'master_data.customer');
  assert.deepEqual(typed.contract.version,{major:2,minor:4});
  assert.deepEqual(typed.inputTypes,[{input:'id',type:'customer.customer_id',
    targetAttribute:'customer.customer.customer_id'}]);
  assert.deepEqual(typed.output,{semanticData:'customer.customer',cardinality:'zero-or-one'});
  assert.equal(typed.executionPermitted,false);
  assert.equal(typed.implementationSelected,false);
  assert.equal(typed.accepted,false);
  assert.equal(typed.verified.typedDeclaration,true);
  assert.equal(typed.verified.providerImplementation,false);
  assert.equal(typed.verified.authorization,false);
  assert.equal(typed.verified.atomicWorkspaceSnapshot,false);

  // Read only already-normalized Environment declarations. No credentials, SQL,
  // plugin resolution or database connection are attempted.
  const routed=observeReferenceProviderRouting(world,observations,{environmentId:'development'});
  assert.equal(routed.status,'application_routes_declared');
  assert.equal(routed.sameInstanceDeclared,true);
  assert.deepEqual(routed.routes.map(x=>x.alias),['master','customer_master']);
  assert.deepEqual(routed.routes.map(x=>x.instance),['master_data.dev','master_data.dev']);
  assert.deepEqual(routed.routes.map(x=>x.provider),['oracle','oracle']);
  assert.equal(routed.verified.pluginExecutorResolution,false);
  assert.equal(routed.verified.providerHealth,false);
  assert.equal(routed.verified.callerAuthorization,false);
  assert.equal(routed.executionPermitted,false);
  assert.equal(routed.implementationSelected,false);
  assert.equal(routed.accepted,false);
});
