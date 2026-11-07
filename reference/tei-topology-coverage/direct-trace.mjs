/** Read-only evidence boundary for a qualified, direct CLI trace result.
 * NOT a Source reader, relation-entity decoder, closed-world proof, or policy.
 */
const record = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const exact = x => typeof x === 'string' && x.length > 0 && x.trim() === x && !/[\r\n\t]/.test(x);
const key = x => record(x) && exact(x.scope) && record(x.subject) && exact(x.subject.kind) && exact(x.subject.id)
  ? [x.scope, x.subject.kind, x.subject.id] : null;
const equalKey = (a, b) => a && b && a.every((v, i) => v === b[i]);
const unknown = reason => ({status:'unknown',reason});

// Internal Reference only. The caller must supply an already authorized, public
// slice before emitting any of this into a real Consumer. This code CANNOT
// evaluate current Policy, enumerate all Sources, or grant Inspect authority.
export function observeDirectTrace(envelope) {
  if (!record(envelope) || envelope.schemaVersion !== 'tei.cli.trace/v0' || envelope.ok !== true || !record(envelope.result))
    return unknown('cli_trace_unavailable');
  const r = envelope.result, trace = r.trace;
  if (r.schemaVersion !== 'tei.operations.trace/v0' || !record(trace) ||
      trace.schemaVersion !== 'tei.modelref.trace/v0' || trace.authority !== 'candidate-read-only' ||
      !exact(r.snapshotFingerprint) || !exact(r.worldFingerprint) || !exact(r.sourceIndexFingerprint) ||
      !exact(trace.snapshotFingerprint)) return unknown('unqualified_trace_shape');
  const selected = key(trace.subject);
  if (!selected || trace.mode !== 'full' || !['both','incoming','outgoing'].includes(trace.direction) ||
      !Number.isSafeInteger(trace.total) || trace.total < 0 ||
      !Number.isSafeInteger(trace.unresolved) || trace.unresolved < 0 ||
      !Array.isArray(trace.rows)) return unknown('unqualified_trace_shape');
  if ((trace.offset ?? 0) !== 0 || trace.nextOffset !== undefined || trace.rows.length !== trace.total)
    return unknown('incomplete_direct_trace_page');
  // SourceBridge Navigation carries the subject it located. A "located"
  // status from another subject must not be attached to this trace selector.
  // This is a same-packet consistency check, NOT a second Source or Policy read.
  if (!record(r.source) || !['located','unresolved'].includes(r.source.status))
    return unknown('source_navigation_unqualified');
  const sourceKey = key(r.source.key);
  if (!sourceKey) return unknown('source_navigation_unqualified');
  if (!equalKey(selected, sourceKey)) return unknown('source_navigation_identity_mismatch');
  if (r.source.status === 'located' &&
      (!Array.isArray(r.source.references) || r.source.references.length === 0))
    return unknown('source_navigation_unqualified');
  if (r.projectionUnresolved !== undefined &&
      (!Array.isArray(r.projectionUnresolved) ||
       r.projectionUnresolved.some(x => !exact(x))))
    return unknown('invalid_projection_unresolved');
  const edges=[];
  let unresolved=0;
  for(const row of trace.rows){
    if (!record(row) || !record(row.edge) || !exact(row.edge.relation) ||
        !['resolved','unresolved'].includes(row.status)) return unknown('invalid_direct_edge');
    const from=key(row.edge.from), target=key(row.resolvedTo);
    if (!from) return unknown('invalid_direct_edge');
    if (row.status === 'resolved' && !target) return unknown('unresolved_direct_edge');
    if (row.status === 'unresolved' && row.resolvedTo !== undefined) return unknown('invalid_direct_edge');
    if (trace.direction === 'outgoing' && !equalKey(from,selected)) return unknown('selector_mismatch');
    if (trace.direction === 'incoming' && (!target || !equalKey(target,selected))) return unknown('selector_mismatch');
    if (trace.direction === 'both' && !(equalKey(from,selected) || equalKey(target,selected))) return unknown('selector_mismatch');
    if (row.status === 'unresolved') unresolved++;
    edges.push({from, to:target, relationType:row.edge.relation, resolution:row.status});
  }
  if (unresolved !== trace.unresolved) return unknown('unresolved_count_mismatch');
  // Derived projection may be incomplete even when the current direct page is complete.
  // Graph scope and relation-entity identity cannot be reconstructed from an
  // Edge.Relation string. Do not pass these candidates into boundedAbsence().
  return {
    status:'direct-edge-candidates',
    selected,
    scopeOrigin:'caller-supplied-projection-scope',
    edges,
    snapshotFingerprint:r.snapshotFingerprint,
    graphFingerprint:trace.snapshotFingerprint,
    hasUnresolvedProjection:Array.isArray(r.projectionUnresolved) && r.projectionUnresolved.length > 0,
    sourceNavigation:r.source?.status === 'located' ? 'located-navigation-only' : 'unresolved',
    relationEntityIdentity:'not-provided',
    multiRoleClosure:'unproven',
    sourceAndPluginClosure:'unproven',
    absentClaim:'unknown',
    authority:'read-only-candidate',
  };
}
