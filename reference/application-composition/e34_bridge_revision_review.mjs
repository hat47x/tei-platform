// E34 is a bounded, read-only comparison of two REAL pinned PR revisions.
// It reuses E32's GitHub file snapshot; it is not a product Source Index,
// Provider registry, implementation attestation or runtime authorization.
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const sha40 = x => typeof x === 'string' && /^[0-9a-f]{40}$/.test(x);
const allowed = new Map([
  ['tools/reference-project-advice-lit/README.md',{status:'modified',role:'documentation'}],
  ['tools/reference-project-advice-lit/main.go',{status:'modified',role:'diagnostic-host'}],
  ['tools/reference-project-advice-lit/contract_bridge.go',{status:'added',role:'diagnostic-host'}],
  ['tools/reference-project-advice-lit/contract_bridge_test.go',{status:'added',role:'verification'}],
]);
const error = message => {throw new TypeError(message);};

export function reviewE34BridgeDelta(previous,delta) {
  if (previous?.kind !== 'tei-e32-github-pr-files-evidence-v0' ||
      previous.repository !== 'hat47x/tei-platform' ||
      previous.pullRequest !== 1567 || !sha40(previous.headSha) ||
      !Array.isArray(previous.files) || previous.files.length !== 20 ||
      delta?.kind !== 'tei-e34-project-bridge-fixed-revision-delta-v0' ||
      delta.repository !== previous.repository ||
      delta.pullRequest !== previous.pullRequest ||
      delta.source !== 'github-compare-commits' ||
      delta.previousHead !== previous.headSha || !sha40(delta.currentHead) ||
      delta.currentHead === delta.previousHead ||
      !Array.isArray(delta.files) || delta.files.length !== allowed.size)
    error('invalid_e34_revision_footprint');

  const previousByPath = new Map();
  for (const f of previous.files) {
    if (typeof f?.path !== 'string' || previousByPath.has(f.path) ||
        !sha40(f.gitBlobSha) || f.status !== 'added' ||
        !Number.isSafeInteger(f.additions) || f.additions < 0 ||
        f.deletions !== 0) error('invalid_e32_source_snapshot');
    previousByPath.set(f.path,f);
  }
  const observed = new Set();
  let additions=0,deletions=0,created=0,changed=0;
  for(const f of delta.files) {
    const expected=allowed.get(f?.path);
    if(!expected || observed.has(f.path) || f.status!==expected.status ||
       f.role!==expected.role || !sha40(f.afterBlob) ||
       !Number.isSafeInteger(f.additions) || f.additions < 0 ||
       !Number.isSafeInteger(f.deletions) || f.deletions < 0)
      error('invalid_e34_changed_file');
    observed.add(f.path);
    const old=previousByPath.get(f.path);
    if(f.status==='added') {
      if(old!==undefined || f.beforeBlob!==null || f.deletions!==0)
        error('invalid_e34_added_source');
      created++;
    } else {
      if(!old || !sha40(f.beforeBlob) || f.beforeBlob!==old.gitBlobSha ||
          f.beforeBlob===f.afterBlob) error('invalid_e34_prior_source');
      changed++;
    }
    additions+=f.additions;
    deletions+=f.deletions;
  }
  if(observed.size!==allowed.size) error('incomplete_e34_source_delta');
  return {
    kind:'tei-e34-bridge-candidate-source-review-v0',
    pinned:{repository:delta.repository,pullRequest:delta.pullRequest,
      before:delta.previousHead,after:delta.currentHead},
    observed:{previousFiles:previousByPath.size,changedPaths:observed.size,
      addedFiles:created,modifiedPriorFiles:changed,
      priorUnmodifiedPaths:previousByPath.size-changed,
      additions,deletions,netAddedLines:additions-deletions,
      totalNewFilesInDraft:previousByPath.size+created},
    evidence:'fixed-github-source-diff-metadata-not-retrieved-live',
    // Presence of contract_bridge.go is NOT verified #951 conformance.
    bridgeSourcePresent:true,
    providerContractAccepted:null,
    artifactDigestVerified:null,
    hostCurrentPermissionVerified:null,
    actionAcceptedMeaningVerified:null,
    productionDeploymentVerified:null,
    humanEffortMeasured:null,
    aiQuestionsMeasured:null,
  };
}

if (process.argv[1] && import.meta.url===pathToFileURL(resolve(process.argv[1])).href) {
  const dir = new URL('.',import.meta.url);
  const previous=JSON.parse(await readFile(new URL('e32_project_plugin_pr1567_snapshot.json',dir),'utf8'));
  const delta=JSON.parse(await readFile(new URL('e34_project_bridge_pr1567_delta.json',dir),'utf8'));
  process.stdout.write(JSON.stringify(reviewE34BridgeDelta(previous,delta),null,2)+'\n');
}
