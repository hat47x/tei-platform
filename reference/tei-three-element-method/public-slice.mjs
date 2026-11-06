// Tiny read-only projection of an ALREADY QUALIFIED public slice.
// It is not a policy evaluator, Role resolver, Source reader or Acceptance gate.
const roles = ["data", "function", "business"];
const acceptable = new Set(["accepted", "unknown", "unsupported", "ambiguous"]);
const text = v => typeof v === "string" && v.length > 0 && v.trim() === v;
const invalid = () => { throw new Error("unqualified_public_slice"); };
const cmp = (a, b) => a < b ? -1 : a > b ? 1 : 0;
const subjectKey = s => JSON.stringify([s.scope, s.kind, s.id]);
const viewKey = (role, s) => JSON.stringify([role, s.scope, s.kind, s.id]);

export function projectPublicTeaSlice(slice) {
  if (!slice || !text(slice.scope) || !Array.isArray(slice.subjects) ||
      !Array.isArray(slice.relations)) invalid();
  const identities = new Set();
  const projected = {data: [], function: [], business: []};
  const publicDiagnostics = [];
  for (const s of slice.subjects) {
    if (!s || !text(s.scope) || s.scope !== slice.scope ||
        !text(s.kind) || !text(s.id) || !s.roleStates ||
        typeof s.roleStates !== "object" || Array.isArray(s.roleStates)) invalid();
    const identity = subjectKey(s);
    if (identities.has(identity)) invalid();
    identities.add(identity);
    for (const [role, state] of Object.entries(s.roleStates)) {
      if (!roles.includes(role) || !acceptable.has(state)) invalid();
      if (state === "accepted") {
        projected[role].push({viewId: viewKey(role, s), ref: {
          scope: s.scope, subject: {kind: s.kind, id: s.id},
        }});
      } else {
        // Role presence and unresolved state must already be public-qualified.
        publicDiagnostics.push({role, subject: identity, status: state});
      }
    }
  }
  const views = {};
  for (const role of roles) {
    const rows = projected[role].sort((a, b) => cmp(a.viewId, b.viewId));
    views[role] = rows.map((row, i) => ({...row, x: 80, y: 40 + 72 * i}));
  }
  const edges = [];
  const edgeIds = new Set();
  for (const r of slice.relations) {
    if (!r || !text(r.id) || !text(r.kind) || !r.from || !r.to ||
        !roles.includes(r.from.role) || !roles.includes(r.to.role)) invalid();
    if (edgeIds.has(r.id)) invalid();
    edgeIds.add(r.id);
    if (![r.from, r.to].every(endpoint => endpoint.scope === slice.scope &&
        identities.has(subjectKey(endpoint)) &&
        views[endpoint.role].some(v => v.viewId === viewKey(endpoint.role, endpoint)))) {
      // Do not return an ID, an endpoint count or an existence diagnostic.
      invalid();
    }
    edges.push({id: r.id, kind: r.kind,
      from: viewKey(r.from.role, r.from), to: viewKey(r.to.role, r.to)});
  }
  edges.sort((a, b) => cmp(a.id, b.id));
  publicDiagnostics.sort((a, b) => cmp(JSON.stringify(a), JSON.stringify(b)));
  // This says nothing about elements or relations outside the authorized slice.
  return {views, relationTable: edges, publicDiagnostics,
    coverage: {scope: slice.scope, allSourceCompleteness: "not-proven"}};
}
