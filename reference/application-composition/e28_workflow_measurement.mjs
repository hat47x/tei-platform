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
const evidenceURL = s => typeof s==='string' && /^https:\/\/github\.com\/hat47x\/(tei-platform|sui-sensemaking)\/(pull|issues|commit|actions\/runs)\/[^?#\s]+$/.test(s);
const ownKeys = (value, keys) => value && typeof value==='object' &&
  !Array.isArray(value) && Object.keys(value).every(key=>keys.includes(key));
const countedFields = Object.values(operations);
const emptyCount = () => Object.fromEntries(countedFields.map(name=>[name,0]));

function validateSubmission(submission, choices) {
  if (!ownKeys(submission,['kind','scenario','complete','events']) ||
      submission.kind!=='tei-e28-workflow-submission-v0' ||
      typeof submission.complete!=='boolean' || !Array.isArray(submission.events) ||
      !ownKeys(submission.scenario,['id','consumer','arm','baseRevision','target']) ||
      !token(submission.scenario.id) || !sha(submission.scenario.baseRevision) ||
      !token(submission.scenario.target) ||
      !['existing-source','profile-candidate'].includes(submission.scenario.arm) ||
      !['order_entry','sui_stage0'].includes(submission.scenario.consumer)) {
    throw new TypeError('invalid_workflow_submission');
  }
  const seen = new Set();
  for (const event of submission.events) {
    if (!ownKeys(event,['id','choiceId','operation','evidenceRef','path','beforeBlob','afterBlob','outcome']) ||
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
      if (!pathOK(event.path) || !sha(event.beforeBlob) || !sha(event.afterBlob) ||
          event.beforeBlob===event.afterBlob || event.outcome!==undefined) {
        throw new TypeError('invalid_source_edit');
      }
    } else if (event.path!==undefined || event.beforeBlob!==undefined || event.afterBlob!==undefined) {
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
      sourceSummary, scenario:null, submittedCounts:null,
      verifiedCounts:null, compatibilityVerified:false, runtimeAuthorityVerified:false};
  }
  validateSubmission(submission,choices);
  const submittedCounts = emptyCount();
  const changedPaths = new Set();
  for (const event of submission.events) {
    if (event.operation==='changed-source') changedPaths.add(event.path);
    else submittedCounts[operations[event.operation]]++;
  }
  submittedCounts.changedFiles=changedPaths.size;
  return {kind:'tei-e28-workflow-measurement-v0',
    scope:submission.complete ? 'claimed-complete-not-verified' : 'partial-not-verified',
    sourceSummary, scenario:{...submission.scenario}, submittedCounts,
    verifiedCounts:null, compatibilityVerified:false, runtimeAuthorityVerified:false};
}

export function compareE28(existing, candidate) {
  const a=existing?.scenario, b=candidate?.scenario;
  if (!a || !b || a.arm!=='existing-source' || b.arm!=='profile-candidate' ||
      a.id!==b.id || a.consumer!==b.consumer || a.baseRevision!==b.baseRevision ||
      a.target!==b.target || !existing.submittedCounts || !candidate.submittedCounts) {
    return {status:'not-comparable', reportedDelta:null, verifiedImprovement:null};
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
