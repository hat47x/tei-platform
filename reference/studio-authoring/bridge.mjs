// Issue #1124: Reference-only consumer of an accepted World snapshot.
// No source writes, no independent acceptance/validation/authority engine.
export const EDIT_SCHEMA = "tei.reference.studio-edit/v0";
export const CANDIDATE_SCHEMA = "tei.reference.change-candidate/v0";
const REVISION = /^sha256:[0-9a-f]{64}$/;

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
  const patterns = new Map(), namespaces = new Map(), validations = new Map();
  const applications = new Map(), validationConsumers = new Map();
  for (const ns of world.semantics) {
    insert(namespaces, ns.id, ns, "namespace");
    for (const validation of array(ns.validations, "namespace.validations")) {
      insert(validations, validation.id, { ...validation, namespace: ns.id }, "validation");
    }
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
    insert(applications, app.id, app, "application");
    for (const validationID of array(app.validations, "application.validations")) {
      requireID(validationID, "application.validation");
      const users = validationConsumers.get(validationID) || [];
      if (users.some(u => u.application === app.id))
        throw new Error("DUPLICATE_APPLICATION_VALIDATION_USE: " + app.id + ":" + validationID);
      users.push({ application: app.id, status: "source-declared" });
      validationConsumers.set(validationID, users);
    }
    for (const pattern of array(app.patterns, "application.patterns")) {
      insert(patterns, pattern.id, { ...pattern, application: app.id }, "pattern");
    }
  }
  return { data, relations, attributes, patterns, namespaces, validations, applications, validationConsumers };
}
function selected(index, subject) {
  if (typeof subject !== "string") throw new Error("INVALID_SUBJECT");
  const cut = subject.indexOf(":");
  if (cut < 1) throw new Error("INVALID_SUBJECT");
  const kind = subject.slice(0, cut), id = subject.slice(cut + 1);
  const map = { data: index.data, relation: index.relations,
    attribute: index.attributes, pattern: index.patterns, validation: index.validations }[kind];
  if (!map || !map.has(id)) throw new Error("UNRESOLVED_SUBJECT: " + subject);
  return { kind, id, value: map.get(id) };
}
function ownerData(s, index) {
  if (s.kind === "data") return s.id;
  if (s.kind === "relation" || s.kind === "attribute") return s.value.owner;
  if (s.kind === "validation") {
    const a = index.attributes.get(s.value.attribute);
    const r = index.relations.get(s.value.relation);
    return a?.owner || r?.owner || null;
  }
  if (s.value.data && index.data.has(s.value.data)) return s.value.data;
  if (s.value.relation && index.relations.has(s.value.relation))
    return index.relations.get(s.value.relation).owner;
  return null;
}

const SOURCE_KINDS = Object.freeze({
  data: "semantic-data",
  relation: "semantic-relation",
  attribute: "semantic-attribute",
  validation: "semantic-validation",
  pattern: "application-pattern"
});

// The Source Index is supplied by the existing TEI source-index producer.
// These locators are view metadata, not permission to edit or Acceptance
// authority. No label-to-path inference or second source ownership.
function sourceLens(sourceIndex, kind, id) {
  if (!sourceIndex) return { status: "unresolved",
    reason: "Source Index was not supplied" };
  if (!Array.isArray(sourceIndex.artifacts) ||
      !Array.isArray(sourceIndex.references)) {
    throw new Error("INVALID_SOURCE_INDEX");
  }
  const artifacts = new Map();
  for (const a of sourceIndex.artifacts) {
    if (typeof a.path !== "string" || !a.path ||
        a.path.startsWith("/") || a.path.includes("\\") ||
        /(^|\/)\.\.(\/|$)/.test(a.path) ||
        /^[A-Za-z]:/.test(a.path) ||
        a.path.split("/").some(part => !part || part === ".") ||
        !REVISION.test(a.fingerprint) || artifacts.has(a.path)) {
      throw new Error("INVALID_OR_DUPLICATE_SOURCE_ARTIFACT");
    }
    artifacts.set(a.path, a.fingerprint);
  }
  const canonicalKind = SOURCE_KINDS[kind];
  const refs = sourceIndex.references.filter(ref =>
    ref?.subject?.kind === canonicalKind && ref?.subject?.id === id &&
    ref.role === "definition"
  );
  if (refs.length === 0) return {
    status: "unresolved", subject: { kind: canonicalKind, id },
    reason: "No exact definition anchor in current Source Index"
  };
  // #1120 identifies a node by scope + kind + ID. Source Index today has
  // no scoped model reference; a caller-added scope tag is not enough to
  // establish the same accepted meaning. Refuse to silently drop scope.
  if (refs.some(r =>
    (r.scope !== undefined && r.scope !== null && r.scope !== "") ||
    (r.subject?.scope !== undefined && r.subject.scope !== null &&
     r.subject.scope !== "")))
    return { status: "unresolved", subject: { kind: canonicalKind, id },
      reason: "Scoped locator requires #1120 typed Source Resolver" };
  if (refs.length !== 1) throw new Error("AMBIGUOUS_SOURCE_DEFINITION: " + kind + ":" + id);
  const ref = refs[0], fingerprint = artifacts.get(ref.artifactPath);
  const begin = ref.range?.start, end = ref.range?.end;
  if (!fingerprint || !begin || !end ||
      !Number.isInteger(begin.offset) || !Number.isInteger(end.offset) ||
      begin.offset < 0 || end.offset < begin.offset ||
      !Number.isInteger(begin.line) || begin.line < 1 ||
      !Number.isInteger(begin.column) || begin.column < 1 ||
      !Number.isInteger(end.line) || end.line < 1 ||
      !Number.isInteger(end.column) || end.column < 1 ||
      end.line < begin.line ||
      (end.line === begin.line && end.column < begin.column)) {
    throw new Error("INVALID_SOURCE_ANCHOR");
  }
  return {
    status: "source-index-backed",
    subject: { kind: canonicalKind, id },
    artifactPath: ref.artifactPath,
    artifactFingerprint: fingerprint,
    range: ref.range,
    byteVerification: { status: "not-checked" },
    caveat: "Index only: raw Source bytes and range are unverified. Live inspect can verify them for that read; write authority remains with Shared Acceptance Preview."
  };
}

// The trusted adapter supplies the actual file bytes and a digest function,
// rather than an unrelated caller-provided fingerprint. Derive the SHA-256
// from the exact bytes whose range is inspected. This still cannot lock
// Source or grant authoring rights; Shared Acceptance rechecks R0.
export function verifySourceAnchor(navigation, bytes, sha256ForBytes) {
  if (navigation?.status !== "source-index-backed" ||
      !(bytes instanceof Uint8Array) || typeof sha256ForBytes !== "function")
    throw new Error("SOURCE_ANCHOR_VERIFICATION_INPUT_INVALID");
  const fingerprint = sha256ForBytes(bytes);
  if (!REVISION.test(fingerprint))
    throw new Error("SOURCE_ANCHOR_DIGEST_INVALID");
  if (navigation.artifactFingerprint !== fingerprint)
    throw new Error("SOURCE_ARTIFACT_FINGERPRINT_STALE");
  const start = navigation.range?.start, end = navigation.range?.end;
  const valid = p => p && Number.isInteger(p.offset) && p.offset >= 0 &&
    Number.isInteger(p.line) && p.line >= 1 &&
    Number.isInteger(p.column) && p.column >= 1;
  if (!valid(start) || !valid(end) || end.offset < start.offset ||
      end.offset > bytes.length)
    throw new Error("SOURCE_ANCHOR_RANGE_INVALID");
  let line = 1, column = 1, startMatches = false, endMatches = false;
  for (let offset = 0; offset <= end.offset; offset++) {
    if (offset === start.offset)
      startMatches = start.line === line && start.column === column;
    if (offset === end.offset) {
      endMatches = end.line === line && end.column === column;
      break;
    }
    if (bytes[offset] === 10) { line++; column = 1; }
    else column++;
  }
  if (!startMatches || !endMatches)
    throw new Error("SOURCE_ANCHOR_POSITION_STALE");
  return {
    ...navigation,
    byteVerification: {
      status: "verified-at-read", fingerprint,
      range: "offset-and-line-column-matched"
    },
    caveat: "Source bytes and range matched at this read only; no filesystem lock, semantic Acceptance, or write authority. Shared Acceptance must re-check the revision."
  };
}

// Consume #1125's read-only Business DFD JSON projection if explicitly supplied.
// This adapter intentionally does not parse Business Procedure fixtures or claim
// that their reference-local Step IDs belong to accepted Canonical meaning.
function businessLens(dfd, patterns, index, focus) {
  if (!dfd || dfd.schemaVersion !== "business-dfd-projection-v0" ||
      dfd.authority !== "derived-read-only" ||
      dfd.canonicalProcedureUsed !== false ||
      !isText(dfd.procedureId) ||
      !Array.isArray(dfd.steps) || !Array.isArray(dfd.diagnostics)) {
    throw new Error("UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA");
  }
  const inFocus = new Set(patterns.map(p => p.id));
  // Direct authoring references may be displayed only when the producer's
  // exact typed kind/ID is still present in this World. Never turn a
  // reference-local Step into an accepted Business or executable Function.
  const explicitKind = {
    data: "semantic-data", relation: "semantic-relation",
    validation: "semantic-validation"
  }[focus.kind];
  const explicitMap = {
    data: index.data, relation: index.relations,
    validation: index.validations
  }[focus.kind];
  const seenSteps = new Set(), matching = [], manual = [];
  const diagnostics = [];
  const meaningMaps = {
    "semantic-data": index.data,
    "semantic-relation": index.relations,
    "semantic-validation": index.validations
  };
  for (const step of dfd.steps) {
    if (!step || !isText(step.id) || seenSteps.has(step.id) ||
        !Array.isArray(step.functionRefs))
      throw new Error("INVALID_OR_DUPLICATE_BUSINESS_STEP");
    seenSteps.add(step.id);
    const links = [], directLinks = [], stepDiagnostics = [];
    // A Step's meaning is an entire set of explicit references. A matched
    // focus cannot make that Step sound if a sibling reference is stale.
    for (const r of step.functionRefs) {
      if (!r || !isText(r.id) || !["resolved", "unresolved"].includes(r.status))
        throw new Error("INVALID_BUSINESS_FUNCTION_REFERENCE");
      const canonical = index.patterns.get(r.id);
      const current = !!canonical &&
        r.status === "resolved" &&
        (r.data || "") === (canonical.data || "") &&
        (r.relation || "") === (canonical.relation || "") &&
        (r.capability || "") === (canonical.capability || "");
      if (!current) stepDiagnostics.push({
        code: "BUSINESS_PATTERN_REFERENCE_STALE_OR_UNRESOLVED",
        stepID: step.id, patternID: r.id
      });
      if (inFocus.has(r.id)) links.push({
        subject: "pattern:" + r.id, id: r.id,
        status: current ? "exact-id-matches-current-world" : "unresolved"
      });
    }
    for (const r of array(step.explicitMeaning, "step.explicitMeaning")) {
      if (!r || !isText(r.kind) || !isText(r.id) ||
          !["semantic-data", "semantic-relation", "semantic-validation",
            "semantic-capability"].includes(r.kind) ||
          !["resolved-id-not-accepted", "unresolved"].includes(r.status))
        throw new Error("INVALID_BUSINESS_DIRECT_MEANING");
      // Studio has no authoritative capability resolver; never infer it.
      const current = r.status === "resolved-id-not-accepted" &&
        r.authority === "procedure-reference-only" &&
        !!meaningMaps[r.kind]?.has(r.id);
      if (!current) stepDiagnostics.push({
        code: "BUSINESS_DIRECT_MEANING_STALE_OR_UNRESOLVED",
        stepID: step.id, kind: r.kind, id: r.id
      });
      if (r.kind === explicitKind && r.id === focus.id) directLinks.push({
        kind: r.kind, id: r.id, subject: focus.kind + ":" + r.id,
        status: current ? "exact-id-matches-current-world" : "unresolved",
        authority: "reference-only", source: r.source || null
      });
    }
    if (links.length || directLinks.length) {
      diagnostics.push(...stepDiagnostics);
      matching.push({
        stepID: step.id, procedureID: dfd.procedureId,
        status: stepDiagnostics.length ? "unresolved" : "reference-only",
        resolutionSummary: {
          declared: step.functionRefs.length +
            array(step.explicitMeaning, "step.explicitMeaning").length,
          unresolved: stepDiagnostics.length
        },
        functionRefs: links, explicitMeaning: directLinks,
        source: step.source || null
      });
    }
    if (step.noApplicationFunction === true && manual.length < 20)
      manual.push({
        stepID: step.id, realization: step.realization,
        status: "reference-only-no-application-function",
        source: step.source || null
      });
  }
  const hasProducerErrors = dfd.diagnostics.some(d => d.severity === "error");
  return {
    status: hasProducerErrors || diagnostics.length || !matching.length
      ? "unresolved" : "reference-only",
    authority: "noncanonical-reference",
    procedureID: dfd.procedureId,
    freshness: "unverified-no-model-revision-binding",
    items: matching,
    manualOrExternalUnlinked: manual,
    manualOrExternalTruncated: dfd.steps.filter(s => s.noApplicationFunction === true).length > 20,
    diagnostics: [...dfd.diagnostics, ...diagnostics],
    note: "Only explicit Pattern and typed semantic ID matches are shown. Manual/external steps can directly refer to semantic Data/Relation/Validation without becoming Functions. DFD remains noncanonical, and no same-revision binding or Source Acceptance is implied."
  };
}

// Navigate FROM a reference-local Business Step back to existing World
// meaning, without assigning that Step accepted identity or source ownership.
// This is a consumer of the opt-in #1125 DFD, never a new Source editor.
export function navigateReferenceBusinessStep(world, revision, dfd, stepID) {
  exactRevision(revision);
  if (!isText(stepID) || !dfd ||
      dfd.schemaVersion !== "business-dfd-projection-v0" ||
      dfd.authority !== "derived-read-only" ||
      dfd.canonicalProcedureUsed !== false ||
      !isText(dfd.procedureId) ||
      !Array.isArray(dfd.steps) || !Array.isArray(dfd.diagnostics))
    throw new Error("UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA");
  const index = indexWorld(world);
  const seen = new Set();
  for (const step of dfd.steps) {
    if (!step || !isText(step.id) || seen.has(step.id) ||
        !Array.isArray(step.functionRefs))
      throw new Error("INVALID_OR_DUPLICATE_BUSINESS_STEP");
    seen.add(step.id);
  }
  const step = dfd.steps.find(s => s.id === stepID);
  if (!step) throw new Error("UNRESOLVED_REFERENCE_BUSINESS_STEP");
  const producerValid = !dfd.diagnostics.some(d => d.severity === "error");
  const candidates = [];
  const add = (kind, id, via, resolved, source) => {
    candidates.push({
      focus: kind + ":" + id, kind, id, via,
      status: producerValid && resolved ? "identity-match-only" : "unresolved",
      authority: "reference-only", source: source || null
    });
  };
  for (const r of array(step.explicitMeaning, "step.explicitMeaning")) {
    if (!r || !isText(r.kind) || !isText(r.id) ||
        !["semantic-data", "semantic-relation", "semantic-validation",
          "semantic-capability"].includes(r.kind) ||
        !["resolved-id-not-accepted", "unresolved"].includes(r.status))
      throw new Error("INVALID_BUSINESS_DIRECT_MEANING");
    const kind = { "semantic-data": "data", "semantic-relation": "relation",
      "semantic-validation": "validation" }[r.kind];
    // Capabilities currently lack a canonical Studio focus map; unresolved.
    if (!kind) {
      add("capability", r.id, "explicit-business-semantic-reference", false, r.source);
      continue;
    }
    const target = { data: index.data, relation: index.relations,
      validation: index.validations }[kind].get(r.id);
    add(kind, r.id, "explicit-business-semantic-reference",
      !!target && r.status === "resolved-id-not-accepted" &&
      r.authority === "procedure-reference-only", r.source);
  }
  for (const r of step.functionRefs) {
    if (!r || !isText(r.id) || !["resolved", "unresolved"].includes(r.status))
      throw new Error("INVALID_BUSINESS_FUNCTION_REFERENCE");
    const pattern = index.patterns.get(r.id);
    add("pattern", r.id, "explicit-application-pattern",
      !!pattern && r.status === "resolved" &&
      (r.data || "") === (pattern.data || "") &&
      (r.relation || "") === (pattern.relation || "") &&
      (r.capability || "") === (pattern.capability || ""));
  }
  const verified = candidates.filter(c => c.status === "identity-match-only");
  const unresolvedCount = candidates.length - verified.length;
  return {
    schema: "tei.reference.studio-business-step-navigation/v0",
    authority: "noncanonical-reference",
    modelRevision: revision,
    subject: { kind: "step", id: step.id,
      scope: "reference-procedure:" + dfd.procedureId,
      authority: "reference-only" },
    // A single valid link cannot make a mixed stale/unresolved Step look sound.
    status: producerValid && verified.length > 0 && unresolvedCount === 0
      ? "reference-only" : "unresolved",
    resolutionSummary: {
      matched: verified.length, unresolved: unresolvedCount,
      producerErrors: dfd.diagnostics.filter(d => d.severity === "error").length
    },
    candidates,
    source: step.source || null,
    freshness: "unverified-no-model-revision-binding",
    editable: false, acceptanceTargetStatus: "unresolved-use-shared-acceptance",
    realizationReadiness: { status: "not-verified" },
    note: "Candidate focuses are read-only World identity matches. The Business Step is not Accepted Source; these links do not establish a shared revision, authoring location, execution effect or approval."
  };
}

// Each view is a projection, not another editable copy of World.
// All IDs are original exact Canonical IDs; no label matching or synthetic edges.
export function projectWorld(world, revision, focus, options = {}) {
  exactRevision(revision);
  const ix = indexWorld(world);
  const s = selected(ix, focus);
  const dataID = ownerData(s, ix);
  const node = dataID === null ? null : ix.data.get(dataID);
  const relatedRelations = node
    ? [...ix.relations.values()].filter(r => r.owner === dataID || r.target === dataID)
    : s.kind === "relation" ? [s.value] : [];
  const relationIDs = new Set(relatedRelations.map(r => r.id));
  const visibleDataIDs = new Set(node ? [node.id] : []);
  for (const r of relatedRelations) {
    if (ix.data.has(r.owner)) visibleDataIDs.add(r.owner);
    if (ix.data.has(r.target)) visibleDataIDs.add(r.target);
  }
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
  const relevantValidations = node
    ? [...ix.validations.values()].filter(v =>
      (v.attribute && ix.attributes.get(v.attribute)?.owner === dataID) ||
      (v.relation && ix.relations.get(v.relation)?.owner === dataID))
    : [];
  const table = node ? {
    ...link("data", node.id),
    attributes: array(node.attributes, "data.attributes").map(a => link("attribute", a.id)),
    relations: array(node.relations, "data.relations").map(r => link("relation", r.id)),
    validations: relevantValidations.map(v => ({
      ...link("validation", v.id), kind: v.kind,
      applicationSelection: ix.validationConsumers.get(v.id) || [],
      evaluationAuthority: "not-verified-by-projection",
      attribute: v.attribute && ix.attributes.has(v.attribute)
        ? link("attribute", v.attribute) : null,
      relation: v.relation && ix.relations.has(v.relation)
        ? link("relation", v.relation) : null
    }))
  } : null;
  return {
    schema: "tei.reference.studio-projection/v0",
    modelRevision: revision,
    authority: "derived-readonly",
    focus: { subject: focus, kind: s.kind, id: s.id },
    views: {
      table: { status: table ? "available" : "not-applicable", value: table },
      er: { status: er.length ? "available" : "not-applicable",
        nodes: [...visibleDataIDs].map(id => link("data", id)), edges: er },
      function: { status: relatedPatterns.length ? "available" : "not-applicable",
        items: relatedPatterns.map(p => ({
          ...link("pattern", p.id), application: p.application, kind: p.kind,
          data: p.data && ix.data.has(p.data) ? link("data", p.data) : null,
          relation: p.relation && ix.relations.has(p.relation)
            ? link("relation", p.relation) : null,
          dataReferenceStatus: !p.data ? "not-declared" :
            ix.data.has(p.data) ? "source-backed" : "unresolved",
          relationReferenceStatus: !p.relation ? "not-declared" :
            ix.relations.has(p.relation) ? "source-backed" : "unresolved",
          unresolvedDataID: p.data && !ix.data.has(p.data) ? p.data : null,
          unresolvedRelationID: p.relation && !ix.relations.has(p.relation) ? p.relation : null
        })) },
      business: options.businessDFD
        ? businessLens(options.businessDFD, relatedPatterns, ix, s)
        : { status: "unresolved", items: [],
          reason: "Accepted Procedure is not in World. Optional #1125 read-only DFD is not an accepted Procedure." }
    },
    sourceNavigation: sourceLens(options.sourceIndex, s.kind, s.id),
    realizationReadiness: { status: "not-verified",
      sourceAcceptedDoesNotImply: ["database-migration", "target-deployment", "runtime-authorization", "provider-write", "observation"],
      note: "Use #1121/#1126/#1127 providers for physical readiness; no migration is inferred from display-only changes." },
    editable: false
  };
}

function isText(x) { return typeof x === "string" && x.length > 0 && x.trim() === x; }
function ensureNotUsed(id, ix) {
  if (ix.data.has(id) || ix.relations.has(id) || ix.attributes.has(id) ||
      ix.patterns.has(id) || ix.validations.has(id)) throw new Error("DUPLICATE_ID: " + id);
}

// Converts a proposed change to the existing Shared Acceptance candidate
// transport. It never patches World or files. Shared Acceptance owns all
// semantic validation, impact, source resolution and revision concurrency.
export function candidateFromEdit(world, revision, edit) {
  exactRevision(revision);
  if (!edit || edit.schema !== EDIT_SCHEMA || edit.readonly === true ||
      edit.derived === true) throw new Error("DERIVED_IS_READONLY_OR_INVALID_EDIT");
  // An edit created from a visual snapshot must not be silently rebased when
  // review-edit later reads a newer accepted World. Older unbound CLI inputs
  // remain supported, but the browser form always supplies this precondition.
  if (edit.expectedModelRevision !== undefined) {
    if (!REVISION.test(edit.expectedModelRevision))
      throw new Error("STUDIO_EDIT_EXPECTED_REVISION_INVALID");
    if (edit.expectedModelRevision !== revision)
      throw new Error("STUDIO_EDIT_STALE_MODEL_REVISION");
  }
  if (!isText(edit.id) || !isText(edit.intent) ||
      !["studio", "ai", "file", "cli"].includes(edit.origin) ||
      !["er", "table"].includes(edit.view)) throw new Error("INVALID_EDIT");
  const ix = indexWorld(world);
  const s = selected(ix, edit.subject);
  let op;
  const ops = [];
  if (edit.view === "er" && edit.operation === "add-relation" && s.kind === "data") {
    if (!isText(edit.relationID) || !isText(edit.symbol) ||
        !isText(edit.targetData) || !ix.data.has(edit.targetData) ||
        !["one", "many"].includes(edit.cardinality)) throw new Error("INVALID_RELATION_PROPOSAL");
    ensureNotUsed(edit.relationID, ix);
    if (edit.relationID !== s.id + "." + edit.symbol)
      throw new Error("RELATION_ID_MUST_MATCH_SOURCE_SYMBOL");
    // Shared Acceptance expects an AddRelationSpec, not a root DataSpec.
    op = { op: "add-relation", addRelation: {
      data: s.id, id: edit.relationID, symbol: edit.symbol,
      target: edit.targetData, cardinality: edit.cardinality
    } };
  } else if (edit.view === "table" && edit.operation === "add-validation" &&
             s.kind === "attribute") {
    if (!isText(edit.validationID) || edit.validationKind !== "positive")
      throw new Error("UNSUPPORTED_VALIDATION_PROPOSAL");
    ensureNotUsed(edit.validationID, ix);
    op = { op: "add-validation", namespace: s.value.namespace,
      id: edit.validationID, kind: "positive", attribute: s.id };
    if (edit.applicationUse !== undefined) {
      const use = edit.applicationUse;
      if (!use || !isText(use.application) || !isText(use.placement))
        throw new Error("INVALID_APPLICATION_USE_REQUEST");
      const app = ix.applications.get(use.application);
      if (!app || !array(app.semanticUses, "application.semanticUses").includes(s.value.namespace))
        throw new Error("APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE");
      if (array(app.validations, "application.validations").includes(edit.validationID))
        throw new Error("VALIDATION_ALREADY_SELECTED");
      // Caller supplies a source location; only Shared Acceptance may decide
      // whether that placement is owned by the Application and writable.
      if (!/^applications\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_./-]+\.tei$/.test(use.placement) ||
          use.placement.split("/").includes("..") || use.placement.split("/").includes("."))
        throw new Error("UNSAFE_APPLICATION_SOURCE_PLACEMENT");
      ops.push({ op: "add-application-validation-use", placement: use.placement,
        applicationValidationUse: { application: use.application, validation: edit.validationID } });
    }
  } else {
    throw new Error("UNSUPPORTED_EDIT: use a source-owned semantic operation; never write a derived cell");
  }
  // Do not embed workspace path, inferred source location or derived view layout.
  return {
    schema: CANDIDATE_SCHEMA, id: edit.id, baseModelRevision: revision,
    origin: { kind: edit.origin },
    intent: edit.intent, delta: [op, ...ops]
  };
}


// This is a consumer-level coherence check, not an alternative Preview or
// acceptance policy. Two separate Shared Acceptance reads must describe the
// same candidate and R0; otherwise a Studio review packet is not usable.
export function bindReviewArtifacts(candidate, preview, review, revision) {
  exactRevision(revision);
  const fail = (why) => { throw new Error("REVIEW_BINDING_MISMATCH: " + why); };
  if (!candidate || candidate.schema !== CANDIDATE_SCHEMA ||
      candidate.baseModelRevision !== revision || !isText(candidate.id))
    fail("candidate");
  if (!preview || preview.schema !== "tei.reference.change-preview/v0" ||
      preview.candidateId !== candidate.id ||
      preview.baseModelRevision !== revision ||
      preview.currentModelRevision !== revision ||
      !REVISION.test(preview.candidateFingerprint || ""))
    fail("preview identity or revision");
  if (!review || review.schema !== "tei.reference.approval-review/v0" ||
      review.candidateId !== candidate.id ||
      review.candidateFingerprint !== preview.candidateFingerprint ||
      review.previewAcceptable !== preview.acceptable ||
      !REVISION.test(review.reviewFingerprint || ""))
    fail("review identity, semantic fingerprint or result");
  if ((review.evidenceFingerprint || "") !== (preview.evidenceFingerprint || ""))
    fail("evidence");
  if (preview.acceptable !== true) {
    // Failed reviews should be displayed to a human, never as a valid
    // source-edit candidate. Callers must not submit these to Accept.
    return { status: "not-acceptable", readyToAccept: false,
      candidateFingerprint: preview.candidateFingerprint,
      reviewFingerprint: review.reviewFingerprint };
  }
  return { status: "reviewed-consistent", readyToAccept: false,
    candidateFingerprint: preview.candidateFingerprint,
    reviewFingerprint: review.reviewFingerprint };
}
