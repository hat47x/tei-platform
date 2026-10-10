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

/**
 * E7 is the authority *for its own limited Source Index observation*.
 * #1120 modelref owns typed reference validation and candidate Impact traversal.
 * This adapter never claims either input has authenticated provenance.
 *
 * @param {object} input.sourceDrift Output of E6/E7, NOT reimplemented here.
 * @param {object} input.typedProjection Candidate #1120 graph projection + its
 *   Validate diagnostics and Impact output, supplied by a trusted caller.
 */
export function qualifyCompositionReview({sourceDrift,typedProjection,application,scope}={}) {
  if (!nonempty(application) || !nonempty(scope) || !obj(sourceDrift) ||
      sourceDrift.authority !== 'none' || sourceDrift.fullClosure !== false ||
      sourceDrift.qualification !== 'application-declaration-and-direct-source-definitions-only' ||
      !['limited-unchanged','unresolved'].includes(sourceDrift.status)) {
    return unresolved('unqualified_source_observation');
  }
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
    // Show impact only when the candidate is the selected Application or a
    // scoped Pattern belonging to it; project-wide unrelated consumers are
    // intentionally not claimed as effects on this Application.
    const selected = keyText(candidate.consumer) === keyText(appKey) ||
      (candidate.consumer.subject.kind === 'application-pattern' &&
       candidate.consumer.subject.id.startsWith(application+'.'));
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
