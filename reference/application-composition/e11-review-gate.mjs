import {assessPairedEvidence} from './e7-paired-evidence.mjs';
// #1016 E11: combine two pre-computed, read-only candidate diagnostics.
// Do not resolve a dependency graph, create a lock or infer runtime authority.
const obj = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const nonempty = x => typeof x === 'string' && !!x.trim() && x === x.trim();
const key = k => obj(k) && nonempty(k.scope) && obj(k.subject) &&
  nonempty(k.subject.kind) && nonempty(k.subject.id);
const keyText = k => JSON.stringify([k.scope,k.subject.kind,k.subject.id]);
const LIMIT = 'read-only-cross-view-review-candidates';
const base = {qualification:LIMIT,fullClosure:false,authority:'none',adoption:'not-decided'};
const unresolved = reason => ({...base,status:'unresolved',reason});

// E5 already owns Application -> Pattern source membership. Use its observed
// definitions, not an ID prefix, as the read-only ownership witness.
// Shape checks cannot authenticate the report or prove a common snapshot.
function declaredPatterns(report,application) {
  if (!obj(report) || report.application !== application ||
      report.status !== 'observed' ||
      report.qualification !== 'declaration-and-source-index-only' ||
      !Array.isArray(report.issues) || report.issues.length !== 0 ||
      !Array.isArray(report.dependencies)) return null;
  let appCount=0;
  const patterns=new Set();
  for (const item of report.dependencies) {
    if (!obj(item) || !obj(item.subject)) return null;
    if (item.relation === 'application') {
      if (item.subject.kind !== 'application' || item.subject.id !== application) return null;
      appCount++;
    } else if (item.relation === 'application-pattern') {
      if (item.subject.kind !== 'application-pattern' ||
          !nonempty(item.subject.id) || patterns.has(item.subject.id)) return null;
      patterns.add(item.subject.id);
    }
  }
  return appCount === 1 ? patterns : null;
}


/**
 * E7 is the authority *for its own limited Source Index observation*.
 * #1120 modelref owns typed reference validation and candidate Impact traversal.
 * This adapter never claims either input has authenticated provenance.
 *
 * @param {object} input.sourceDrift Output of E6/E7, NOT reimplemented here.
 * @param {object} input.sourceReport E5 parsed Application report from the
 *   same qualified Source observation (caller must establish snapshot binding).
 * @param {object} input.typedProjection Candidate #1120 graph projection + its
 *   Validate diagnostics and Impact output, supplied by a trusted caller.
 */
export function qualifyCompositionReview({sourceDrift,sourceReport,typedProjection,application,scope}={}) {
  if (!nonempty(application) || !nonempty(scope) || !obj(sourceDrift) ||
      sourceDrift.authority !== 'none' || sourceDrift.fullClosure !== false ||
      sourceDrift.qualification !== 'application-declaration-and-direct-source-definitions-only' ||
      !['limited-unchanged','unresolved'].includes(sourceDrift.status)) {
    return unresolved('unqualified_source_observation');
  }
  const patterns = declaredPatterns(sourceReport,application);
  if (!patterns) return unresolved('unqualified_application_source_report');
  if (!obj(typedProjection) || typedProjection.schema !== 'tei.modelref.v0' ||
      typedProjection.scope !== scope || typedProjection.application !== application ||
      !Array.isArray(typedProjection.nodes) || !Array.isArray(typedProjection.unresolved) ||
      !Array.isArray(typedProjection.diagnostics) || !Array.isArray(typedProjection.affected)) {
    return unresolved('unqualified_typed_projection');
  }
  if (typedProjection.unresolved.length || typedProjection.diagnostics.length) {
    return unresolved('unresolved_typed_references');
  }
  const appKey = {scope,subject:{kind:'application',id:application}};
  const known = new Set();
  for (const node of typedProjection.nodes) {
    if (!obj(node) || !key(node.key) || node.key.scope !== scope ||
        known.has(keyText(node.key))) return unresolved('ambiguous_typed_node');
    known.add(keyText(node.key));
  }
  if (!known.has(keyText(appKey))) return unresolved('application_not_in_typed_projection');
  for (const patternID of patterns) {
    if (!known.has(keyText({scope,subject:{kind:'application-pattern',id:patternID}}))) {
      return unresolved('declared_pattern_missing_from_projection');
    }
  }
  const refs = new Set();
  const review = [];
  for (const candidate of typedProjection.affected) {
    if (!obj(candidate) || !key(candidate.changed) || !key(candidate.consumer) ||
        !nonempty(candidate.reason) || candidate.changed.scope !== scope ||
        candidate.consumer.scope !== scope ||
        !known.has(keyText(candidate.changed)) || !known.has(keyText(candidate.consumer))) {
      return unresolved('invalid_typed_impact');
    }
    const unique = JSON.stringify([keyText(candidate.changed),keyText(candidate.consumer),candidate.reason]);
    if (refs.has(unique)) return unresolved('duplicate_typed_impact');
    refs.add(unique);
    // Pattern ownership is obtained from E5 typed World declarations, never
    // guessed from a naming prefix. Unrelated scoped patterns remain excluded.
    const selected = keyText(candidate.consumer) === keyText(appKey) ||
      (candidate.consumer.subject.kind === 'application-pattern' &&
       patterns.has(candidate.consumer.subject.id));
    if (selected) review.push({changed:candidate.changed,consumer:candidate.consumer,reason:candidate.reason});
  }
  review.sort((a,b) => JSON.stringify(a).localeCompare(JSON.stringify(b)));
  if (sourceDrift.status === 'unresolved') {
    return {...unresolved('source_requalification_required'),sourceReason:sourceDrift.reason||'unknown',reviewCandidates:review};
  }
  return {...base,status:review.length?'review-candidate':'limited-unchanged',reviewCandidates:review,
    dependencyProvenance:'caller-supplied-typed-candidates',
    sourceQualification:'direct-definitions-only'};
}


/**
 * Consume the real JSON shapes emitted by E5/E7 and #1120 derivedbridge:
 *  - baseline/current: E7 paired evidence envelopes
 *  - projection: derivedbridge.Projection, with graph/rules/unresolved/authorityLimits
 *  - diagnostics: modelref.Validate output
 *  - affected: modelref.Impact output
 *
 * This does NOT run Go's Validate/Impact, certify the caller-provided results,
 * or prove that the Go projection and the E5/E7 snapshot refer to the same
 * source revision. Until there is such a verified join, no "unchanged" verdict
 * can be given by this combined entry point.
 */
export function qualifyPairedCompositionReview({
  baseline,current,projection,diagnostics,affected,application,scope,
}={}) {
  if (!obj(baseline) || !obj(current) || !obj(current.report) ||
      !obj(current.index) || !obj(projection) || !obj(projection.graph) ||
      !Array.isArray(projection.unresolved) ||
      !Array.isArray(projection.authorityLimits) ||
      !Array.isArray(projection.rules) ||
      !Array.isArray(projection.graph.nodes) ||
      !Array.isArray(projection.graph.edges) ||
      !Array.isArray(diagnostics) || !Array.isArray(affected)) {
    return unresolved('incomplete_paired_or_typed_evidence');
  }
  const drift = assessPairedEvidence(baseline,current);
  const candidate = qualifyCompositionReview({
    sourceDrift:drift,
    sourceReport:current.report,
    typedProjection:{
      schema:projection.graph.schema,scope,application,
      nodes:projection.graph.nodes,unresolved:projection.unresolved,
      diagnostics,affected,
    },
    application,scope,
  });
  if (candidate.status === 'unresolved') return candidate;
  // E5/E7 source collection and #1120 projection are currently separate.
  // Do not turn a matching caller-supplied shape into snapshot provenance.
  return {...unresolved('unverified_shared_snapshot'),
    reviewCandidates:candidate.reviewCandidates,
    sourceQualification:candidate.sourceQualification,
    typedQualification:'caller-supplied-modelref-projection'};
}
