// #1016 E12: pass an actual E7 paired comparison and its current E5 report
// to E11. No second Source Index, graph resolver, or runtime authority.
import {assessPairedEvidence} from './e7-paired-evidence.mjs';
import {qualifyCompositionReview} from './e11-review-gate.mjs';

const obj = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const nonempty = x => typeof x === 'string' && !!x.trim() && x === x.trim();
const limit = {fullClosure:false,authority:'none',adoption:'not-decided',
  evidenceBoundary:'unverified-caller-supplied-snapshots'};
const unresolved = reason => ({...limit,status:'unresolved',reason});

// The current Source report MUST be taken from the compared E7 envelope, not
// supplied independently by the caller (prevents a stale-owner mixup).
// Modelref projection remains separately supplied and is NOT source-bound yet.
export function assessBoundCompositionReview({baseline,current,typedProjection,application,scope}={}) {
  if (!nonempty(application) || !nonempty(scope) || !obj(baseline) ||
      !obj(current) || !obj(baseline.report) || !obj(current.report)) {
    return unresolved('missing_paired_evidence');
  }
  if (baseline.report.application !== application || current.report.application !== application) {
    return unresolved('selected_application_mismatch');
  }
  const drift = assessPairedEvidence(baseline,current);
  if (drift.qualification !== 'application-declaration-and-direct-source-definitions-only' ||
      !['limited-unchanged','unresolved'].includes(drift.status)) {
    return {...unresolved('invalid_or_unqualified_paired_evidence'),sourceReason:drift.reason};
  }
  const result = qualifyCompositionReview({
    sourceDrift:drift,
    sourceReport:current.report,
    typedProjection,application,scope,
  });
  // E7 checks each supplied pair internally. No proof that #1120 projection
  // came from that same Workspace revision, nor that the JSON was trusted.
  return {...result,evidenceBoundary:limit.evidenceBoundary,modelrefSnapshotBound:false};
}
