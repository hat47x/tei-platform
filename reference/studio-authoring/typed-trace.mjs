// #1124 consumes the existing Go #1120 modelref/sourcebridge trace.
// This adapter is read-only: an observed scope is not Source ownership,
// a located anchor is not Acceptance, and an edge is not runtime evidence.
const SHA = /^sha256:[0-9a-f]{64}$/;
const KINDS = Object.freeze({
  data: "semantic-data", relation: "semantic-relation",
  attribute: "semantic-attribute", validation: "semantic-validation",
  pattern: "application-pattern"
});
function fail(message) { throw new Error("STUDIO_TYPED_TRACE_INVALID: " + message); }
function identity(value, key) {
  return value && typeof value === "object" &&
    value.scope === key.scope &&
    value.subject?.kind === key.subject.kind &&
    value.subject?.id === key.subject.id;
}
function bytesMatch(reference, navigation, artifacts) {
  return reference.role === "definition" &&
    reference.artifactPath === navigation.artifactPath &&
    reference.range?.start?.offset === navigation.range?.start?.offset &&
    reference.range?.end?.offset === navigation.range?.end?.offset &&
    artifacts.some(a => a.path === navigation.artifactPath &&
      a.fingerprint === navigation.artifactFingerprint);
}

// No model revision is in the #1120 trace response. The caller must obtain
// its actual Workspace revision before AND after invoking the Go trace.
// This adapter checks the transported identity/anchor, not filesystem locking.
export function bindTypedTrace(view, cliReport, observedScope) {
  if (!view || view.schema !== "tei.reference.studio-projection/v0" ||
      !SHA.test(view.modelRevision) || typeof observedScope !== "string" ||
      !observedScope.trim() || observedScope !== observedScope.trim())
    fail("missing projection, revision, or exact observed scope");
  const kind = KINDS[view.focus?.kind], id = view.focus?.id;
  if (!kind || typeof id !== "string" || !id.trim() ||
      view.focus?.subject !== view.focus.kind + ":" + id)
    fail("unsupported or inconsistent Studio focus");
  const key = { scope: observedScope, subject: { kind, id } };
  if (cliReport?.schemaVersion !== "tei.cli.trace/v0" ||
      cliReport.ok !== true || !cliReport.result)
    fail("Go typed trace did not succeed");
  const result = cliReport.result, trace = result.trace, source = result.source;
  if (result.schemaVersion !== "tei.operations.trace/v0" ||
      trace?.schemaVersion !== "tei.modelref.trace/v0" ||
      trace.authority !== "candidate-read-only" ||
      trace.mode !== "full" || trace.direction !== "both" ||
      !identity(trace.subject, key) || !identity(source?.key, key) ||
      !SHA.test(result.snapshotFingerprint) ||
      !SHA.test(result.worldFingerprint) ||
      !SHA.test(result.sourceIndexFingerprint) ||
      !SHA.test(trace.snapshotFingerprint) ||
      !Array.isArray(trace.rows) || !Number.isInteger(trace.total) ||
      trace.total !== trace.rows.length ||
      !Number.isInteger(trace.unresolved) ||
      trace.unresolved < 0 || trace.unresolved > trace.total ||
      trace.nextOffset != null)
    fail("unknown trace contract, mismatched identity, or partial trace");
  for (const row of trace.rows) {
    if (!row || !["resolved", "unresolved"].includes(row.status) ||
        !row.edge?.from?.subject?.kind || !row.edge?.from?.subject?.id ||
        !row.edge?.to?.subject?.kind || !row.edge?.to?.subject?.id ||
        typeof row.edge.relation !== "string" ||
        (row.status === "resolved"
          ? !row.resolvedTo?.subject?.kind || !row.resolvedTo?.subject?.id ||
            row.issue != null
          : typeof row.issue !== "string" || !row.issue))
      fail("invalid typed edge or resolution");
  }
  if (trace.rows.filter(row => row.status === "unresolved").length !== trace.unresolved)
    fail("unresolved count disagrees with trace rows");
  if (!["located", "unresolved"].includes(source.status))
    fail("unknown typed Source status");
  if (!Array.isArray(result.authorityLimits) ||
      !result.authorityLimits.includes("business-procedure-not-in-current-world") ||
      !Array.isArray(source.authorityLimits) ||
      !source.authorityLimits.includes("source-location-is-not-write-authority"))
    fail("missing Go authority boundary");
  let sourceConsistency = "not-comparable";
  const navigation = view.sourceNavigation;
  if (navigation?.status === "source-index-backed") {
    const candidates = source.references || [], artifacts = source.artifacts || [];
    if (!Array.isArray(candidates) || !Array.isArray(artifacts))
      fail("malformed Go source evidence");
    sourceConsistency = source.status === "located" &&
      candidates.some(r => bytesMatch(r, navigation, artifacts))
      ? "exact-anchor-match" : "unresolved";
  }
  const projectionUnresolved = result.projectionUnresolved || [];
  if (!Array.isArray(projectionUnresolved) ||
      projectionUnresolved.some(x => typeof x !== "string"))
    fail("invalid Go projection diagnostics");
  return {
    schema: "tei.reference.studio-typed-trace/v0",
    authority: "candidate-read-only",
    modelRevision: view.modelRevision,
    scope: observedScope,
    scopeAuthority: "caller-selected-observation-only",
    snapshotFingerprint: result.snapshotFingerprint,
    sourceIndexFingerprint: result.sourceIndexFingerprint,
    subject: key,
    sourceStatus: source.status,
    sourceConsistency,
    status: sourceConsistency === "unresolved" || trace.unresolved > 0
      ? "unresolved" : "reference-only",
    total: trace.total, unresolved: trace.unresolved,
    projectionUnresolved, rows: trace.rows,
    note: "Go #1120 typed trace; no accepted Business identity, edit authority, or Shared Acceptance. Model revision was guarded around the separate Go read, not locked."
  };
}
