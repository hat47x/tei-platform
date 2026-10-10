// Proposal-only DFD editing surface for #1125.
// It is deliberately incompatible with Shared Acceptance change-candidate/v0.
// No file write, accepted Source update, runtime binding, or Workflow creation.
export const EDIT_SCHEMA = "tei.reference.business-dfd-edit/v0";
export const PREVIEW_SCHEMA = "tei.reference.business-dfd-edit-preview/v0";
const REVIEW_SCHEMA = "business-dfd-review-v0";
const HASH = /^sha256:[0-9a-f]{64}$/;
const KINDS = new Set(["step", "party", "store"]);
const FLOW_KINDS = new Set(["sequence", "branch", "exception"]);

function exact(value, name) {
  if (typeof value !== "string" || !value || value !== value.trim()) {
    throw new Error("INVALID_EXACT_ID_OR_TEXT: " + name);
  }
  return value;
}
function list(value, name) {
  if (value === undefined) return [];
  if (!Array.isArray(value)) throw new Error("INVALID_ARRAY: " + name);
  return value;
}
function keys(value, allowed, context) {
  if (!value || Array.isArray(value) || typeof value !== "object" ||
      Object.keys(value).some(k => !allowed.includes(k))) {
    throw new Error("UNKNOWN_OR_INVALID_FIELD: " + context);
  }
}
function uniqueIds(items, name) {
  const ids = new Set();
  for (const item of items) {
    if (!item || typeof item !== "object") throw new Error("INVALID_ITEM: " + name);
    const id = exact(item.id, name + ".id");
    if (ids.has(id)) throw new Error("DUPLICATE_ID: " + name + ":" + id);
    ids.add(id);
  }
  return ids;
}
function endpoint(value, steps, parties, stores) {
  keys(value, ["kind", "id"], "endpoint");
  const kind = exact(value.kind, "endpoint.kind");
  const id = exact(value.id, "endpoint.id");
  if (!KINDS.has(kind)) throw new Error("UNSUPPORTED_ENDPOINT_KIND");
  const ids = { step: steps, party: parties, store: stores }[kind];
  if (!ids.has(id)) throw new Error("UNRESOLVED_ENDPOINT: " + kind + ":" + id);
  return {kind, id};
}
function validate(base, procedure) {
  keys(base, [
    "schemaVersion", "authority", "procedure", "triggers", "flows",
    "businessRoles", "externalParties", "stores", "responsibilities", "informationFlows"
  ], "review");
  if (base.schemaVersion !== REVIEW_SCHEMA || base.authority !== "proposal") {
    throw new Error("REVIEW_NOT_UNACCEPTED_PROPOSAL");
  }
  if (!procedure || procedure.canonical !== false ||
      procedure.schemaVersion !== "business-procedure-reference-v0" ||
      procedure.artifactRole !== "reference-authoring-fixture" ||
      base.procedure !== procedure.procedure?.id) {
    throw new Error("PROCEDURE_IDENTITY_OR_AUTHORITY_MISMATCH");
  }
  const steps = uniqueIds(list(procedure.steps, "steps"), "steps");
  const triggers = uniqueIds(list(base.triggers,"triggers"), "triggers");
  const roles = uniqueIds(list(base.businessRoles,"businessRoles"), "businessRoles");
  const parties = uniqueIds(list(base.externalParties,"externalParties"), "externalParties");
  const stores = uniqueIds(list(base.stores,"stores"), "stores");
  const info = uniqueIds(list(base.informationFlows,"informationFlows"), "informationFlows");
  const responsibilityKeys = new Set();
  for (const r of list(base.responsibilities,"responsibilities")) {
    const step = exact(r.step, "responsibility.step");
    const role = exact(r.role, "responsibility.role");
    if (!steps.has(step) || !roles.has(role)) throw new Error("UNRESOLVED_RESPONSIBILITY");
    const key = JSON.stringify([step, role]);
    if (responsibilityKeys.has(key)) throw new Error("DUPLICATE_RESPONSIBILITY");
    responsibilityKeys.add(key);
  }
  const transitionKeys = new Set();
  for (const f of list(base.flows, "flows")) {
    if (!steps.has(exact(f.from,"control.from")) ||
        !steps.has(exact(f.to,"control.to")) ||
        !FLOW_KINDS.has(f.kind)) throw new Error("UNRESOLVED_CONTROL_FLOW");
    if (["branch","exception"].includes(f.kind) && !exact(f.condition,"control.condition")) {
      throw new Error("MISSING_CONTROL_CONDITION");
    }
    const key=JSON.stringify([f.from,f.to,f.kind,f.condition || ""]);
    if (transitionKeys.has(key)) throw new Error("DUPLICATE_CONTROL_FLOW");
    transitionKeys.add(key);
  }
  for (const t of list(base.triggers,"triggers")) {
    if (!["human","event","schedule"].includes(t.kind) ||
        !steps.has(exact(t.targetStep,"trigger.targetStep"))) {
      throw new Error("UNRESOLVED_TRIGGER");
    }
    exact(t.label,"trigger.label");
  }
  for (const f of list(base.informationFlows,"informationFlows")) {
    const from = endpoint(f.from, steps, parties, stores);
    const to = endpoint(f.to, steps, parties, stores);
    if (from.kind !== "step" && to.kind !== "step") {
      throw new Error("INFORMATION_PROCESS_BOUNDARY_REQUIRED");
    }
    if (from.kind === to.kind && from.id === to.id) throw new Error("INFORMATION_SELF_LOOP");
    exact(f.payload,"information.payload");
  }
  return {steps, triggers, roles, parties, stores, info};
}
function clone(value) { return JSON.parse(JSON.stringify(value)); }
function operationKeys(op, names) { keys(op, names, "edit.operation"); }

// Neither the caller-supplied digest nor its equality proves current file bytes.
// Go DFD snapshot guards / Shared Acceptance own that verification.
export function proposeDFDReviewEdit(procedure, review, currentReviewSHA256, edit) {
  if (!HASH.test(currentReviewSHA256)) throw new Error("UNVERIFIED_REVIEW_SNAPSHOT");
  keys(edit, ["schema","id","origin","intent","baseReviewSHA256","procedureId","operation"],
       "edit");
  if (edit.schema !== EDIT_SCHEMA || !["studio","ai","cli","file"].includes(edit.origin) ||
      !HASH.test(edit.baseReviewSHA256) || edit.baseReviewSHA256 !== currentReviewSHA256 ||
      edit.procedureId !== procedure?.procedure?.id) {
    throw new Error("INVALID_EDIT_OR_STALE_REVIEW");
  }
  exact(edit.id, "edit.id");
  exact(edit.intent, "edit.intent");
  const ix = validate(review, procedure);
  const op = edit.operation;
  if (!op || typeof op !== "object" || Array.isArray(op)) throw new Error("INVALID_EDIT_OPERATION");
  const next = clone(review);
  let changed;
  switch (op.kind) {
    case "assign-business-role":
      operationKeys(op, ["kind","step","role"]);
      if (!ix.steps.has(exact(op.step,"operation.step")) ||
          !ix.roles.has(exact(op.role,"operation.role"))) {
        throw new Error("UNRESOLVED_ROLE_ASSIGNMENT");
      }
      if (list(next.responsibilities,"responsibilities").some(x =>
        x.step === op.step && x.role === op.role)) throw new Error("DUPLICATE_RESPONSIBILITY");
      next.responsibilities ??= [];
      next.responsibilities.push({step:op.step, role:op.role});
      changed = {kind:"responsibility", id:JSON.stringify([op.step,op.role]), action:"proposal-add"};
      break;
    case "replace-trigger-label": {
      operationKeys(op, ["kind","trigger","label"]);
      if (!ix.triggers.has(exact(op.trigger,"operation.trigger"))) throw new Error("UNKNOWN_TRIGGER");
      exact(op.label,"operation.label");
      const existing=next.triggers.find(x => x.id === op.trigger);
      if (existing.label === op.label) throw new Error("NO_EFFECT_EDIT");
      existing.label=op.label;
      changed={kind:"trigger",id:op.trigger,action:"proposal-update"};
      break;
    }
    case "add-control-transition": {
      operationKeys(op, ["kind","from","to","transitionKind","condition"]);
      if (!ix.steps.has(exact(op.from,"operation.from")) ||
          !ix.steps.has(exact(op.to,"operation.to")) ||
          !FLOW_KINDS.has(op.transitionKind)) throw new Error("INVALID_CONTROL_TRANSITION");
      if (op.transitionKind !== "sequence") exact(op.condition,"operation.condition");
      if (op.transitionKind === "sequence" && op.condition !== undefined) {
        throw new Error("SEQUENCE_CANNOT_CLAIM_CONDITION");
      }
      const transition={from:op.from,to:op.to,kind:op.transitionKind};
      if (op.condition !== undefined) transition.condition=op.condition;
      if (list(next.flows,"flows").some(f =>
          f.from === transition.from && f.to === transition.to &&
          f.kind === transition.kind && (f.condition || "") === (transition.condition || ""))) {
        throw new Error("DUPLICATE_CONTROL_FLOW");
      }
      next.flows ??= [];
      next.flows.push(transition);
      changed={kind:"control-transition",id:JSON.stringify(transition),action:"proposal-add"};
      break;
    }
    case "add-information-flow": {
      operationKeys(op, ["kind","id","from","to","payload"]);
      const id=exact(op.id,"operation.id");
      if (ix.info.has(id)) throw new Error("DUPLICATE_INFORMATION_FLOW");
      const from=endpoint(op.from,ix.steps,ix.parties,ix.stores);
      const to=endpoint(op.to,ix.steps,ix.parties,ix.stores);
      if (from.kind !== "step" && to.kind !== "step") {
        throw new Error("INFORMATION_PROCESS_BOUNDARY_REQUIRED");
      }
      if (from.kind === to.kind && from.id === to.id) throw new Error("INFORMATION_SELF_LOOP");
      exact(op.payload,"operation.payload");
      next.informationFlows ??= [];
      next.informationFlows.push({id,from,to,payload:op.payload});
      changed={kind:"information-flow",id,action:"proposal-add"};
      break;
    }
    default:
      throw new Error("UNSUPPORTED_EDIT_OPERATION_NO_ARBITRARY_PATCH");
  }
  validate(next, procedure);
  return {
    schema: PREVIEW_SCHEMA,
    authority: "unaccepted-proposal-only",
    procedureId: review.procedure,
    baseReviewSHA256: currentReviewSHA256,
    reviewSHA256Verification: "caller-supplied-not-cryptographically-verified",
    acceptanceStatus: "not-accepted",
    modelRevisionStatus: "unverified",
    sourceWrites: [],
    readyForAcceptance: false,
    proposedChange: changed,
    proposedReview: next
  };
}
