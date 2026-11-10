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
<<<<<<< HEAD
      !exact(trace.snapshotFingerprint)) return unknown('unqualified_trace_shape');
=======
      !exact(trace.graphFingerprint) || !exact(trace.ruleFingerprint))
    return unknown('unqualified_trace_shape');
  // #3625: the inner snapshotFingerprint is a deprecated alias of this Graph
  // token. The outer result.snapshotFingerprint is the *only* workspace-level
  // continuation value, and must not be substituted with the Graph token.
  if (trace.snapshotFingerprint !== undefined &&
      trace.snapshotFingerprint !== trace.graphFingerprint)
    return unknown('graph_fingerprint_alias_mismatch');
  // Load-time manifest provenance (#3181), not an Acceptance record or lock.
  const ws=r.workspace;
  if (!record(ws) || !record(ws.manifest) ||
      !exact(ws.manifest.path) || !exact(ws.manifest.sha256) ||
      !Number.isSafeInteger(ws.formatVersion) || ws.formatVersion <= 0 ||
      (ws.name !== undefined && !exact(ws.name)))
    return unknown('workspace_identity_unqualified');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
  const selected = key(trace.subject);
  if (!selected || trace.mode !== 'full' || !['both','incoming','outgoing'].includes(trace.direction) ||
      !Number.isSafeInteger(trace.total) || trace.total < 0 ||
      !Number.isSafeInteger(trace.unresolved) || trace.unresolved < 0 ||
      !Array.isArray(trace.rows)) return unknown('unqualified_trace_shape');
<<<<<<< HEAD
  if ((trace.offset ?? 0) !== 0 || trace.nextOffset !== undefined || trace.rows.length !== trace.total)
=======
  if ((trace.offset ?? 0) !== 0 || trace.nextOffset !== undefined ||
      trace.continuation !== undefined || trace.rows.length !== trace.total)
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
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
<<<<<<< HEAD
=======
  // SourceIndex.Reference has a subject but no Scope. Bind its Kind/ID
  // to the requested subject without inventing a Source-declared scope.
  if (r.source.references !== undefined) {
    if (!Array.isArray(r.source.references)) return unknown('source_navigation_unqualified');
    for (const ref of r.source.references) {
      if (!record(ref) || !record(ref.subject) ||
          !exact(ref.subject.kind) || !exact(ref.subject.id) ||
          !exact(ref.artifactPath)) return unknown('source_reference_unqualified');
      if (ref.subject.kind !== selected[1] || ref.subject.id !== selected[2])
        return unknown('source_reference_identity_mismatch');
    }
  }
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
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
<<<<<<< HEAD
=======
    // Row.ResolvedTo must agree with the declaration carried by Edge.To.
    // Empty Ref.Scope may be resolved by the graph; it is not a wildcard or
    // permission to change the declared target Kind/ID.
    const to = row.edge.to;
    if (!record(to) || !record(to.subject) ||
        !exact(to.subject.kind) || !exact(to.subject.id) ||
        (to.scope !== undefined && to.scope !== '' && !exact(to.scope)))
      return unknown('invalid_direct_target');
    if (row.status === 'resolved' &&
        (to.subject.kind !== target[1] || to.subject.id !== target[2] ||
         (to.scope && to.scope !== target[0])))
      return unknown('resolved_edge_target_mismatch');
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
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
<<<<<<< HEAD
    edges,
    snapshotFingerprint:r.snapshotFingerprint,
    graphFingerprint:trace.snapshotFingerprint,
=======
    workspaceIdentity:'load-manifest-observed-only',
    // Internal comparison metadata. NOT a public-disclosure or acceptance token.
    workspaceLoad:{
      name:ws.name ?? null,
      formatVersion:ws.formatVersion,
      manifestPath:ws.manifest.path,
      manifestSha256:ws.manifest.sha256,
    },
    edges,
    snapshotFingerprint:r.snapshotFingerprint,
    graphFingerprint:trace.graphFingerprint,
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
    hasUnresolvedProjection:Array.isArray(r.projectionUnresolved) && r.projectionUnresolved.length > 0,
    sourceNavigation:r.source?.status === 'located' ? 'located-navigation-only' : 'unresolved',
    relationEntityIdentity:'not-provided',
    multiRoleClosure:'unproven',
    sourceAndPluginClosure:'unproven',
    absentClaim:'unknown',
    authority:'read-only-candidate',
  };
}
<<<<<<< HEAD
=======

/** Check whether two traces may be compared as the *same observed load
 * context*. This remains read-only / no completeness, Policy, or Acceptance.
 * Caller must first qualify any public release through the real Policy.
 */
export function compareObservedLoadContext(first, second) {
  const a=observeDirectTrace(first), b=observeDirectTrace(second);
  if (a.status !== 'direct-edge-candidates' || b.status !== 'direct-edge-candidates')
    return unknown('trace_unqualified');
  if (JSON.stringify(a.workspaceLoad) !== JSON.stringify(b.workspaceLoad))
    return unknown('workspace_manifest_mismatch');
  if (first.result.worldFingerprint !== second.result.worldFingerprint ||
      first.result.sourceIndexFingerprint !== second.result.sourceIndexFingerprint)
    return unknown('read_snapshot_mismatch');
  // Caller scopes may differ. No inference about Source-declared scope, grants,
  // general Relation identity, or whether the actual read roots are closed.
  // Matching observable bytes do not identify a physical Root or a read
  // Attempt: a byte-for-byte copy can yield an indistinguishable CLI packet.
  return {status:'same-observed-load-context-only',authority:'read-only-candidate',
    physicalRootIdentity:'not-provided',readAttemptIdentity:'not-provided',
    sourceAndPluginClosure:'unproven',absentClaim:'unknown'};
}
>>>>>>> origin/ref/topology-direct-trace-main-port-20261108
