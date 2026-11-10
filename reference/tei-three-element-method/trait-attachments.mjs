// Read-only TEA presentation binding for independently prequalified Trait claims.
// No Trait evaluator, Policy/Source/Role resolver, Owner acceptance or authority.
const ROLES = new Set(["data", "function", "business"]);
const LANES = new Set(["intent", "observation", "assessment"]);
const STATES = new Set(["proposal", "source-observed", "not-proven", "counterexample-observed"]);
const text = s => typeof s === "string" && s.length > 0 && s.trim() === s;
const fail = () => { throw new Error("unqualified_trait_attachment"); };
const record = (value, fields) => {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const p = Object.getPrototypeOf(value);
  if (p !== Object.prototype && p !== null) return false;
  return Reflect.ownKeys(value).every(k => {
    if (typeof k !== "string" || !fields.includes(k)) return false;
    const desc = Object.getOwnPropertyDescriptor(value, k);
    return desc && Object.hasOwn(desc, "value");
  });
};
const key = (role, scope, kind, id) => JSON.stringify([role, scope, kind, id]);
const sortKey = x => JSON.stringify([x.viewId, x.lane, x.claimId]);
const plainArray = value => {
  if (!Array.isArray(value) || Object.getPrototypeOf(value) !== Array.prototype) return false;
  const length = value.length;
  // No holes, own metadata, symbol keys, getters or own iterators.
  if (Reflect.ownKeys(value).length !== length + 1) return false;
  for (let i = 0; i < length; i++) {
    const desc = Object.getOwnPropertyDescriptor(value, String(i));
    if (!desc || !Object.hasOwn(desc, "value")) return false;
  }
  return true;
};

// "snapshot" must agree across projector, envelope, claims: local bind only.
// Both snapshot and workspaceObservation are caller-supplied equality witnesses,
// NOT independently verified physical Root, read attempt, Source or Policy evidence.
// "claims" is an already disclosed, already selected list: never raw private data.
export function bindPublicTraitClaims(publicProjection, envelope) {
  if (!record(envelope, ["scope", "workspaceObservation", "snapshot", "claims"]) ||
      !text(envelope.scope) || !text(envelope.workspaceObservation) ||
      !text(envelope.snapshot) ||
      !plainArray(envelope.claims) ||
      !record(publicProjection, ["snapshot", "workspaceObservation", "views", "relationTable", "publicDiagnostics", "coverage"]) ||
      !text(publicProjection.workspaceObservation) ||
      publicProjection.workspaceObservation !== envelope.workspaceObservation ||
      !text(publicProjection.snapshot) ||
      publicProjection.snapshot !== envelope.snapshot ||
      !record(publicProjection.views, ["data", "function", "business"]) ||
      !record(publicProjection.coverage, ["scope", "allSourceCompleteness"]) ||
      publicProjection.coverage.scope !== envelope.scope) fail();
  const published = new Map();
  for (const role of ROLES) {
    const views = publicProjection.views[role];
    if (!plainArray(views)) fail();
    for (const v of views) {
      if (!record(v, ["viewId", "ref", "x", "y"]) ||
          !record(v.ref, ["scope", "subject"]) ||
          !record(v.ref.subject, ["kind", "id"]) ||
          v.ref.scope !== envelope.scope || !text(v.ref.subject.kind) ||
          !text(v.ref.subject.id) || !text(v.viewId) ||
          v.viewId !== key(role, v.ref.scope, v.ref.subject.kind, v.ref.subject.id) ||
          published.has(v.viewId)) fail();
      published.set(v.viewId, role);
    }
  }
  const ids = new Set();
  const bindings = [];
  for (const claim of envelope.claims) {
    if (!record(claim, ["claimId", "scope", "workspaceObservation", "kind", "subjectId", "role", "traitId", "lane", "state", "evidenceId", "snapshot"]) ||
        ![claim.claimId, claim.scope, claim.workspaceObservation,
          claim.kind, claim.subjectId, claim.traitId,
          claim.evidenceId, claim.snapshot].every(text) ||
        claim.scope !== envelope.scope ||
        claim.workspaceObservation !== envelope.workspaceObservation ||
        claim.snapshot !== envelope.snapshot ||
        !ROLES.has(claim.role) || !LANES.has(claim.lane) || !STATES.has(claim.state) ||
        ids.has(claim.claimId)) fail();
    const viewId = key(claim.role, claim.scope, claim.kind, claim.subjectId);
    if (published.get(viewId) !== claim.role) fail();
    ids.add(claim.claimId);
    bindings.push({viewId, claimId: claim.claimId, traitId: claim.traitId,
      lane: claim.lane, state: claim.state, evidenceId: claim.evidenceId});
  }
  bindings.sort((a, b) => sortKey(a).localeCompare(sortKey(b), "en"));
  return {snapshot: envelope.snapshot,
    workspaceObservation: envelope.workspaceObservation, bindings,
    // A presentation join does not determine whether the world is complete.
    allSourceCompleteness: "not-proven"};
}
