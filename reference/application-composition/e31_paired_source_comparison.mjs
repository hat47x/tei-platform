// E31: compare the two Source revision claims of a SUI+TEI work session.
// This wraps the existing E28 read-only reported comparison. It is not a
// Source registry, lockfile, provenance verifier, compatibility gate or grant.
import {compareE28} from './e28_workflow_measurement.mjs';

const sha = x => typeof x==='string' && /^[a-f0-9]{40}$/.test(x);
const reject = reason=>({status:'not-comparable',reason,
  reportedDelta:null,verifiedImprovement:null});

export function compareE31PairedSource({existing,candidate,claimedRevisions}={}) {
  const a=existing?.scenario, b=candidate?.scenario;
  if (!a || !b || !['order_entry','sui_stage0'].includes(a.consumer) ||
      a.consumer!==b.consumer) return reject('unknown-or-different-consumers');
  const old=claimedRevisions?.existing, now=claimedRevisions?.candidate;
  if (!sha(old?.tei) || !sha(now?.tei) ||
      !sha(a.baseRevision) || !sha(b.baseRevision) ||
      old.tei!==a.baseRevision || now.tei!==b.baseRevision ||
      old.tei!==now.tei) return reject('missing-or-mismatched-tei-revision');

  // SUI is not an accepted TEI Application here. Both actual repositories
  // contribute to the review baseline: one TEI hash is insufficient.
  // Even matching hashes are caller declarations, not authenticated reads.
  const requiresSui=a.consumer==='sui_stage0' ||
    old.sui!==undefined || now.sui!==undefined;
  if (requiresSui && (!sha(old.sui) || !sha(now.sui) ||
      old.sui!==now.sui)) return reject('missing-or-mismatched-sui-revision');

  const report=compareE28(existing,candidate);
  if (!report || typeof report.status!=='string' ||
      !report.status.startsWith('reported-'))
    return reject('e28-workflow-comparison-refused');
  return {
    status:'reported-paired-source-only',
    e28Status:report.status,
    reportedDelta:report.reportedDelta,
    verifiedImprovement:null,
    sourceRevisionsVerified:false,
    runtimeAuthorityVerified:false,
    note:'revisions-and-work-events-are-caller-declared-not-independent-evidence',
  };
}
