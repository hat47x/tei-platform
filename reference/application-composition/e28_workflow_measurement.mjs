// E28: read-only, evidence-bound measurement submissions.
// Submitted events are NOT independently verified actions, questions or grants.
// E27 E1 source observations are recomputed from the current checkout.
import {observeE27} from './e27_residual_selection_audit.mjs';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const operations = Object.freeze({
  'human-decision':'humanDecisions',
  'ai-question':'aiQuestions',
  'manual-connection':'manualConnections',
  'changed-source':'changedFiles',
  'validation-run':'validationRuns',
  'requalification-run':'requalificationRuns',
});
const token = s => typeof s==='string' && /^[a-z][a-z0-9._-]*$/.test(s);
const sha = s => typeof s==='string' && /^[a-f0-9]{40}$/.test(s);
const pathOK = s => typeof s==='string' && s.length>0 && !s.startsWith('/') &&
  !s.includes('\\') && s.split('/').every(part=>part && part!=='.' && part!=='..');
const sourceRepository = s => typeof s==='string' && /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(s) &&
  !s.split('/').some(part=>part==='.' || part==='..');
const evidenceURL = s => typeof s==='string' && /^https:\/\/github\.com\/hat47x\/(tei-platform|sui-sensemaking)\/(pull|issues|commit|actions\/runs)\/[^?#\s]+$/.test(s);
const ownKeys = (value, keys) => value && typeof value==='object' &&
  !Array.isArray(value) && Object.keys(value).every(key=>keys.includes(key));
const countedFields = Object.values(operations);
const emptyCount = () => Object.fromEntries(countedFields.map(name=>[name,0]));
// Optional tracking coverage does not establish the authenticity of events.
function validateCoverage(coverage) {
  if (coverage===undefined) return;
  if (!ownKeys(coverage,countedFields) ||
      Object.values(coverage).some(value=>!['complete','untracked'].includes(value))) {
    throw new TypeError('invalid_workflow_coverage');
  }
}

function validateSubmission(submission, choices) {
  if (!ownKeys(submission,['kind','scenario','complete','events','coverage']) ||
      submission.kind!=='tei-e28-workflow-submission-v0' ||
      typeof submission.complete!=='boolean' || !Array.isArray(submission.events) ||
      !ownKeys(submission.scenario,['id','consumer','arm','baseRevision','target']) ||
      !token(submission.scenario.id) || !sha(submission.scenario.baseRevision) ||
      !token(submission.scenario.target) ||
      !['existing-source','profile-candidate'].includes(submission.scenario.arm) ||
      !['order_entry','sui_stage0'].includes(submission.scenario.consumer)) {
    throw new TypeError('invalid_workflow_submission');
  }
  validateCoverage(submission.coverage);
  const seen = new Set();
  for (const event of submission.events) {
    if (!ownKeys(event,['id','choiceId','operation','evidenceRef','path','repository','beforeBlob','afterBlob','outcome']) ||
        !token(event.id) || seen.has(event.id) ||
        !Object.hasOwn(operations,event.operation) || !evidenceURL(event.evidenceRef)) {
      throw new TypeError('invalid_workflow_event');
    }
    seen.add(event.id);
    const choice = choices.get(event.choiceId);
    if (!choice || choice.consumer!==submission.scenario.consumer ||
        choice.status==='source-unresolved') {
      throw new TypeError('unqualified_workflow_choice');
    }
    if (event.operation==='changed-source') {
      if (!pathOK(event.path) ||
          (event.repository!==undefined && !sourceRepository(event.repository)) ||
          !sha(event.beforeBlob) || !sha(event.afterBlob) ||
          event.beforeBlob===event.afterBlob || event.outcome!==undefined) {
        throw new TypeError('invalid_source_edit');
      }
    } else if (event.path!==undefined || event.repository!==undefined ||
        event.beforeBlob!==undefined || event.afterBlob!==undefined) {
      throw new TypeError('unexpected_source_edit');
    }
    if (['validation-run','requalification-run'].includes(event.operation)) {
      if (!['passed','failed','unresolved'].includes(event.outcome)) {
        throw new TypeError('invalid_verification_outcome');
      }
    } else if (event.outcome!==undefined) {
      throw new TypeError('unexpected_verification_outcome');
    }
  }
}

export function assessE28(audit, submission) {
  if (audit?.kind!=='tei-composition-e27-reference-selection-audit-v0' ||
      !Array.isArray(audit.choices) || audit.summary?.scope!=='bounded-e1-tei-source-only') {
    throw new TypeError('invalid_e27_audit');
  }
  const choices = new Map(audit.choices.map(c=>[c.id,c]));
  if (choices.size!==audit.choices.length) throw new TypeError('duplicate_e27_choice');
  const sourceSummary = {
    sourceObserved:audit.summary.sourceObserved,
    stage0ObservedOnly:audit.summary.stage0ObservedOnly,
    selectionUnresolved:audit.summary.selectionUnresolved,
    runtimeProofUnresolved:audit.summary.runtimeProofUnresolved,
    sourceUnresolved:audit.summary.sourceUnresolved,
  };
  if (submission===undefined || submission===null) {
    return {kind:'tei-e28-workflow-measurement-v0', scope:'observed-e1-source-only',
      sourceSummary, scenario:null, submittedCounts:null, claimedCoveredCounts:null,
      verifiedCounts:null, compatibilityVerified:false, runtimeAuthorityVerified:false};
  }
  validateSubmission(submission,choices);
  const submittedCounts = emptyCount();
  const changedPaths = new Set();
  for (const event of submission.events) {
    if (event.operation==='changed-source') changedPaths.add(
      (event.repository || '<unspecified-repository>')+'/'+event.path);
    else submittedCounts[operations[event.operation]]++;
  }
  submittedCounts.changedFiles=changedPaths.size;
  const claimedCoveredCounts=submission.coverage===undefined ? null :
    Object.fromEntries(countedFields.map(name=>[name,
      submission.complete && submission.coverage[name]==='complete' ?
        submittedCounts[name] : null]));
  return {kind:'tei-e28-workflow-measurement-v0',
    scope:submission.complete ? 'claimed-complete-not-verified' : 'partial-not-verified',
    sourceSummary, scenario:{...submission.scenario}, submittedCounts,
    claimedCoveredCounts, verifiedCounts:null, compatibilityVerified:false, runtimeAuthorityVerified:false};
}

export function compareE28(existing, candidate) {
  const a=existing?.scenario, b=candidate?.scenario;
  if (!a || !b || a.arm!=='existing-source' || b.arm!=='profile-candidate' ||
      a.id!==b.id || a.consumer!==b.consumer || a.baseRevision!==b.baseRevision ||
      a.target!==b.target || existing.scope!=='claimed-complete-not-verified' ||
      candidate.scope!=='claimed-complete-not-verified' ||
      !existing.submittedCounts || !candidate.submittedCounts ||
      !countedFields.every(name=>Number.isSafeInteger(existing.submittedCounts[name]) &&
        existing.submittedCounts[name]>=0 &&
        Number.isSafeInteger(candidate.submittedCounts[name]) &&
        candidate.submittedCounts[name]>=0) ||
      ((existing.claimedCoveredCounts===null || existing.claimedCoveredCounts===undefined) !==
       (candidate.claimedCoveredCounts===null || candidate.claimedCoveredCounts===undefined))) {
    return {status:'not-comparable', reportedDelta:null, verifiedImprovement:null};
  }
  if (existing.claimedCoveredCounts && candidate.claimedCoveredCounts) {
    const reportedDelta=Object.fromEntries(countedFields.map(name=>{
      const a=existing.claimedCoveredCounts[name], b=candidate.claimedCoveredCounts[name];
      return [name,Number.isSafeInteger(a) && a>=0 && Number.isSafeInteger(b) && b>=0 ?
        b-a : null];
    }));
    if (!Object.values(reportedDelta).some(delta=>delta!==null)) {
      return {status:'not-comparable', reportedDelta:null, verifiedImprovement:null};
    }
    return {status:'reported-coverage-only-not-verified', reportedDelta,
      verifiedImprovement:null};
  }
  const reportedDelta=Object.fromEntries(countedFields.map(name=>
    [name,candidate.submittedCounts[name]-existing.submittedCounts[name]]));
  return {status:'reported-only-not-verified', reportedDelta, verifiedImprovement:null};
}

export async function observeE28(root, submission) {
  return assessE28(await observeE27(root),submission);
}

if (process.argv[1] && import.meta.url===pathToFileURL(resolve(process.argv[1])).href) {
  // Default invocation is intentionally baseline-only; no fabricated workflow.
  const report=await observeE28(process.argv[2]||'.');
  process.stdout.write(JSON.stringify(report,null,2)+'\n');
  if (report.sourceSummary.sourceUnresolved) process.exitCode=1;
}
