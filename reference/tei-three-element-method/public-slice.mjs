// Tiny read-only projection of an ALREADY QUALIFIED public slice.
// It is not a policy evaluator, Role resolver, Source reader or Acceptance gate.
const roles = ["data", "function", "business"];
const acceptable = new Set(["accepted", "unknown", "unsupported", "ambiguous"]);
const text = v => typeof v === "string" && v.length > 0 && v.trim() === v;
const invalid = () => { throw new Error("unqualified_public_slice"); };
const cmp = (a, b) => a < b ? -1 : a > b ? 1 : 0;
const subjectKey = s => JSON.stringify([s.scope, s.kind, s.id]);
const viewKey = (role, s) => JSON.stringify([role, s.scope, s.kind, s.id]);
const fields = Object.freeze({
  slice: ["scope", "workspaceObservation", "subjects", "relations"],
  subject: ["scope", "kind", "id", "roleStates"],
  relation: ["id", "kind", "from", "to"],
  endpoint: ["scope", "kind", "id", "role"],
});

const plainArray = value => {
  if (!Array.isArray(value) || Object.getPrototypeOf(value) !== Array.prototype) return false;
  const length = value.length;
  // Reject holes, non-index metadata, symbol keys, accessors and own iterators.
  if (Reflect.ownKeys(value).length !== length + 1) return false;
  for (let i = 0; i < length; i++) {
    const desc = Object.getOwnPropertyDescriptor(value, String(i));
    if (!desc || !Object.hasOwn(desc, "value")) return false;
  }
  return true;
};
const onlyFields = (value, allowed) => {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const proto = Object.getPrototypeOf(value);
  return (proto === Object.prototype || proto === null) &&
    Reflect.ownKeys(value).every(key => {
      if (typeof key !== "string" || !allowed.includes(key)) return false;
      // Avoid executing input getters as part of projection or validation.
      const desc = Object.getOwnPropertyDescriptor(value, key);
      return !!desc && Object.hasOwn(desc, "value");
    });
};

export function projectPublicTeaSlice(slice, snapshot = null) {
  // Optional caller-supplied equality witness; NOT a current Source/Policy proof.
  if (snapshot !== null && !text(snapshot)) invalid();
  if (!onlyFields(slice, fields.slice) || !text(slice.scope) ||
      !text(slice.workspaceObservation) ||
      !plainArray(slice.subjects) || !plainArray(slice.relations)) invalid();
  const identities = new Set();
  const projected = {data: [], function: [], business: []};
  const publicDiagnostics = [];
  for (const s of slice.subjects) {
    if (!onlyFields(s, fields.subject) || !text(s.scope) || s.scope !== slice.scope ||
        !text(s.kind) || !text(s.id) ||
        !onlyFields(s.roleStates, roles)) invalid();
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
    if (!onlyFields(r, fields.relation) || !text(r.id) || !text(r.kind) ||
        !onlyFields(r.from, fields.endpoint) || !onlyFields(r.to, fields.endpoint) ||
        ![r.from, r.to].every(endpoint => text(endpoint.scope) &&
          text(endpoint.kind) && text(endpoint.id) && roles.includes(endpoint.role))) invalid();
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
  return {...(snapshot === null ? {} : {snapshot}),
    workspaceObservation: slice.workspaceObservation, views, relationTable: edges,
    publicDiagnostics,
    coverage: {scope: slice.scope, allSourceCompleteness: "not-proven"}};
}
