// E35: bounded, read-only review of a fixed PR/CI observation.
// The snapshot is manually captured GitHub metadata, not an attestation.
// In particular: Go/Chrome test success is not #951 conformance,
// an accepted Application Action, a trusted Artifact or Host authority.
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const sha = x => typeof x==='string' && /^[0-9a-f]{40}$/.test(x);
const positive = x => Number.isSafeInteger(x) && x>0;
const requiredFiles = new Map([
  ['tools/reference-project-advice-lit/contract_bridge.go','7a44b4d0a84aefac33cc7f56c19027a5dbaab9d4'],
  ['tools/reference-project-advice-lit/contract_bridge_test.go','d454c1b377910830dc10353e8b8deb259d8e51c6'],
  ['tools/reference-project-advice-lit/smoke_order_entry.sh','c1920cb93fdff9e19278fd45c83316d932214c44'],
  ['tools/reference-project-advice-lit/main.go','dcc8df49f16f5fe38e3da4ed86d66bbdc4567b81'],
  ['tools/reference-project-advice-lit/web/advice-element.mjs','0f415f06f9e325b3d2f544cbbe7836f0c3f3e65f'],
  ['tools/reference-project-advice-lit/order_entry_context.go','6e05e787025d22895ba89fc664f76e9a58fc8ef3'],
]);
const roles = new Map([
  ['go-node-readonly-integration',{
    workflow:'TEI 案件Go・Lit補助参照テスト',
    criticalSteps:['Go形式と案件コードを検証','実order_entry APIからの読み取り専用接続を検証','UI通信とJavaScript構文を検証'],
  }],
  ['real-chrome-reference',{
    workflow:'TEI 案件Lit実Chrome参照検証',
    criticalSteps:['正確な対象版と実Chromeを確認','実Go API・Go案件HostとLitをChromeで操作'],
  }],
  ['pr-repository-health',{
    workflow:'TEI PR・main 健全性確認',
    criticalSteps:['正規の全体健全性確認を実行','PR候補HEADそのもののE1を検証'],
  }],
]);
const reject = what => {throw new TypeError(what)};

export function reviewE35ReferenceCI(snapshot,{requestedHead}={}) {
  if (snapshot?.kind!=='tei-e35-pinned-project-reference-ci-evidence-v0' ||
      snapshot.repository!=='hat47x/tei-platform' ||
      snapshot.pullRequest!==1567 || !sha(snapshot.headSha) ||
      snapshot.wasDraft!==true ||
      !Array.isArray(snapshot.files) || snapshot.files.length!==requiredFiles.size ||
      !Array.isArray(snapshot.workflowRuns) || snapshot.workflowRuns.length!==roles.size)
    reject('invalid_e35_snapshot');

  const observed=new Set();
  for (const file of snapshot.files) {
    if (!file || typeof file.path!=='string' || observed.has(file.path) ||
        !sha(file.blobSha) || requiredFiles.get(file.path)!==file.blobSha)
      reject('invalid_or_duplicate_source_blob');
    observed.add(file.path);
  }
  const found=new Map(), runs=new Set(), jobs=new Set();
  const unresolved=[];
  for (const check of snapshot.workflowRuns) {
    const expected=roles.get(check?.role);
    if (!expected || found.has(check.role) || !positive(check.runId) ||
        !positive(check.jobId) || runs.has(check.runId) || jobs.has(check.jobId) ||
        check.workflow!==expected.workflow || !sha(check.headSha) ||
        !Array.isArray(check.criticalSteps) ||
        check.criticalSteps.length!==expected.criticalSteps.length ||
        check.criticalSteps.some((s,i)=>s!==expected.criticalSteps[i]) ||
        !check.stepConclusions || typeof check.stepConclusions!=='object' ||
        Array.isArray(check.stepConclusions) ||
        Object.keys(check.stepConclusions).length!==expected.criticalSteps.length ||
        expected.criticalSteps.some(s=>!['success','failure','cancelled','skipped','pending']
          .includes(check.stepConclusions[s])) ||
        !['completed','queued','in_progress','pending'].includes(check.status) ||
        !['success','failure','cancelled',null].includes(check.conclusion) ||
        !['success','failure','cancelled',null].includes(check.jobConclusion))
      reject('invalid_or_duplicate_ci_evidence');
    found.set(check.role,check);
    runs.add(check.runId);
    jobs.add(check.jobId);
    if (check.headSha!==snapshot.headSha) unresolved.push(check.role+':different-head');
    if (check.status!=='completed' || check.conclusion!=='success' ||
        check.jobConclusion!=='success' ||
        expected.criticalSteps.some(s=>check.stepConclusions[s]!=='success'))
      unresolved.push(check.role+':not-proven-success');
  }
  if (found.size!==roles.size) reject('incomplete_ci_evidence');
  if (requestedHead!==undefined && (!sha(requestedHead) || requestedHead!==snapshot.headSha))
    unresolved.push('requested-revision-not-checked');
  return {
    kind:'tei-e35-project-reference-ci-review-v0',
    pinned:{repository:snapshot.repository,pr:snapshot.pullRequest,
      head:snapshot.headSha,observedDraft:true},
    observed:{sourceBlobs:observed.size,workflowRuns:found.size,
      criticalSteps:[...roles.values()].reduce((n,v)=>n+v.criticalSteps.length,0)},
    status:unresolved.length?'not-supported-at-requested-head':'pinned-reference-ci-reported-success',
    unresolved,
    provenance:'github-source-and-ci-metadata-pinned-not-reverified-at-runtime',
    // The exact same CI data, even with 3 green runs, cannot promote authority.
    publishedServiceContract:null,
    conformanceVerified:null,
    executableArtifactVerified:null,
    currentHostPermissionVerified:null,
    approvedActionMeaningVerified:null,
    productionDeploymentVerified:null,
    manualActionsMeasured:null,
    aiQuestionsMeasured:null,
    independentEvidenceAuthentication:false,
  };
}

if (process.argv[1] && import.meta.url===pathToFileURL(resolve(process.argv[1])).href) {
  const fixture=JSON.parse(await readFile(
    new URL('./e35_project_reference_ci_evidence.json',import.meta.url),'utf8'));
  process.stdout.write(JSON.stringify(reviewE35ReferenceCI(fixture),null,2)+'\n');
}
