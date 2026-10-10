// Issue #1124: Reference-only consumer of an accepted World snapshot.
// No source writes, no independent acceptance/validation/authority engine.
export const EDIT_SCHEMA = "tei.reference.studio-edit/v0";
export const CANDIDATE_SCHEMA = "tei.reference.change-candidate/v0";
const REVISION = /^sha256:[0-9a-f]{64}$/;
const own = (obj, key) => Object.prototype.hasOwnProperty.call(obj, key);

function requireID(id, context) {
  if (typeof id !== "string" || !id.trim() || id !== id.trim()) {
    throw new Error(context + ": nonempty exact ID required");
  }
  return id;
}
function array(value, context) {
  if (value === undefined) return [];
  if (!Array.isArray(value)) throw new Error(context + ": array required");
  return value;
}
function insert(map, id, value, kind) {
  requireID(id, kind);
  if (map.has(id)) throw new Error("AMBIGUOUS_ID: " + kind + ":" + id);
  map.set(id, value);
}
function exactRevision(revision) {
  if (!REVISION.test(revision)) throw new Error("INVALID_REVISION");
  return revision;
}

// The accepted World is caller-supplied; this tool does not authenticate its
// provenance. Callers must obtain World and modelRevision from the same workspace.
export function indexWorld(world) {
  if (!world || typeof world !== "object" || !Array.isArray(world.semantics) ||
      !Array.isArray(world.applications)) throw new Error("INVALID_WORLD");
  const data = new Map(), relations = new Map(), attributes = new Map();
  const patterns = new Map(), namespaces = new Map();
  for (const ns of world.semantics) {
    insert(namespaces, ns.id, ns, "namespace");
    for (const d of array(ns.data, "namespace.data")) {
      insert(data, d.id, { ...d, namespace: ns.id }, "data");
      for (const attr of array(d.attributes, "data.attributes")) {
        insert(attributes, attr.id, { ...attr, owner: d.id, namespace: ns.id }, "attribute");
      }
      for (const relation of array(d.relations, "data.relations")) {
        insert(relations, relation.id, { ...relation, owner: d.id, namespace: ns.id }, "relation");
      }
    }
  }
  for (const app of world.applications) {
    requireID(app.id, "application");
    for (const pattern of array(app.patterns, "application.patterns")) {
      insert(patterns, pattern.id, { ...pattern, application: app.id }, "pattern");
    }
  }
  return { data, relations, attributes, patterns, namespaces };
}
function selected(index, subject) {
  if (typeof subject !== "string") throw new Error("INVALID_SUBJECT");
  const cut = subject.indexOf(":");
  if (cut < 1) throw new Error("INVALID_SUBJECT");
  const kind = subject.slice(0, cut), id = subject.slice(cut + 1);
  const map = { data: index.data, relation: index.relations,
    attribute: index.attributes, pattern: index.patterns }[kind];
  if (!map || !map.has(id)) throw new Error("UNRESOLVED_SUBJECT: " + subject);
  return { kind, id, value: map.get(id) };
}
function ownerData(s, index) {
  if (s.kind === "data") return s.id;
  if (s.kind === "relation" || s.kind === "attribute") return s.value.owner;
  if (s.value.data && index.data.has(s.value.data)) return s.value.data;
  if (s.value.relation && index.relations.has(s.value.relation))
    return index.relations.get(s.value.relation).owner;
  return null;
}

// Each view is a projection, not another editable copy of World.
// All IDs are original exact Canonical IDs; no label matching or synthetic edges.
export function projectWorld(world, revision, focus) {
  exactRevision(revision);
  const ix = indexWorld(world);
  const s = selected(ix, focus);
  const dataID = ownerData(s, ix);
  const node = dataID === null ? null : ix.data.get(dataID);
  const relatedRelations = node
    ? [...ix.relations.values()].filter(r => r.owner === dataID || r.target === dataID)
    : s.kind === "relation" ? [s.value] : [];
  const relationIDs = new Set(relatedRelations.map(r => r.id));
  const relatedPatterns = [...ix.patterns.values()].filter(p =>
    p.id === s.id ||
    (dataID !== null && p.data === dataID) ||
    (p.relation && relationIDs.has(p.relation)));
  const link = (kind, id) => ({ subject: kind + ":" + id, id });
  const er = relatedRelations.map(r => ({
    ...link("relation", r.id), owner: link("data", r.owner),
    target: ix.data.has(r.target) ? link("data", r.target) : null,
    cardinality: r.cardinality, status: ix.data.has(r.target) ? "source-backed" : "unresolved"
  }));
  const table = node ? {
    ...link("data", node.id),
    attributes: array(node.attributes, "data.attributes").map(a => link("attribute", a.id)),
    relations: array(node.relations, "data.relations").map(r => link("relation", r.id))
  } : null;
  return {
    schema: "tei.reference.studio-projection/v0",
    modelRevision: revision,
    authority: "derived-readonly",
    focus: { subject: focus, kind: s.kind, id: s.id },
    views: {
      table: { status: table ? "available" : "not-applicable", value: table },
      er: { status: er.length ? "available" : "not-applicable",
        nodes: node ? [link("data", node.id)] : [], edges: er },
      function: { status: relatedPatterns.length ? "available" : "not-applicable",
        items: relatedPatterns.map(p => ({
          ...link("pattern", p.id), application: p.application, kind: p.kind,
          data: p.data && ix.data.has(p.data) ? link("data", p.data) : null,
          relation: p.relation && ix.relations.has(p.relation)
            ? link("relation", p.relation) : null
        })) },
      business: { status: "unresolved", items: [],
        reason: "Accepted Procedure projection is not present in this World; #1125 adapter required" }
    },
    sourceNavigation: { status: "unresolved",
      reason: "Exact Source Index anchors have not been joined to this World" },
    editable: false
  };
}

function isText(x) { return typeof x === "string" && x.length > 0 && x.trim() === x; }
function ensureNotUsed(id, ix) {
  if (ix.data.has(id) || ix.relations.has(id) || ix.attributes.has(id) ||
      ix.patterns.has(id)) throw new Error("DUPLICATE_ID: " + id);
}

// Converts a proposed change to the existing Shared Acceptance candidate
// transport. It never patches World or files. Shared Acceptance owns all
// semantic validation, impact, source resolution and revision concurrency.
export function candidateFromEdit(world, revision, edit) {
  exactRevision(revision);
  if (!edit || edit.schema !== EDIT_SCHEMA || edit.readonly === true ||
      edit.derived === true) throw new Error("DERIVED_IS_READONLY_OR_INVALID_EDIT");
  if (!isText(edit.id) || !isText(edit.intent) ||
      !["studio", "ai", "file", "cli"].includes(edit.origin) ||
      !["er", "table"].includes(edit.view)) throw new Error("INVALID_EDIT");
  const ix = indexWorld(world);
  const s = selected(ix, edit.subject);
  let op;
  if (edit.view === "er" && edit.operation === "add-relation" && s.kind === "data") {
    if (!isText(edit.relationID) || !isText(edit.symbol) ||
        !isText(edit.targetData) || !ix.data.has(edit.targetData) ||
        !["one", "many"].includes(edit.cardinality)) throw new Error("INVALID_RELATION_PROPOSAL");
    ensureNotUsed(edit.relationID, ix);
    op = { op: "add-relation", data: s.id, id: edit.relationID,
      symbol: edit.symbol, target: edit.targetData, cardinality: edit.cardinality };
  } else if (edit.view === "table" && edit.operation === "add-validation" &&
             s.kind === "attribute") {
    if (!isText(edit.validationID) || edit.validationKind !== "positive")
      throw new Error("UNSUPPORTED_VALIDATION_PROPOSAL");
    ensureNotUsed(edit.validationID, ix);
    op = { op: "add-validation", namespace: s.value.namespace,
      id: edit.validationID, kind: "positive", attribute: s.id };
  } else {
    throw new Error("UNSUPPORTED_EDIT: use a source-owned semantic operation; never write a derived cell");
  }
  // Do not embed workspace path, inferred source location or derived view layout.
  return {
    schema: CANDIDATE_SCHEMA, id: edit.id, baseModelRevision: revision,
    origin: { kind: edit.origin },
    intent: edit.intent, delta: [op]
  };
}
