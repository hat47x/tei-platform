import test from "node:test";
import assert from "node:assert/strict";
import { projectWorld, candidateFromEdit } from "./bridge.mjs";

const R0 = "sha256:" + "a".repeat(64);
const R1 = "sha256:" + "b".repeat(64);
const world = {
  semantics: [{ id: "sales", data: [
    { id: "sales.order", attributes: [], relations: [
      { id: "sales.order.lines", target: "sales.line", cardinality: "many" }
    ] },
    { id: "sales.line", attributes: [], relations: [] }
  ] }],
  applications: [{ id: "order_entry", patterns: [
    { id: "order_entry.lines", kind: "header-detail", relation: "sales.order.lines" }
  ] }]
};

test("#1124: three accepted-World lenses share a revision; Business without authority stays unresolved", () => {
  const before = JSON.stringify(world);
  const projection = projectWorld(world, R0, "relation:sales.order.lines");
  assert.equal(projection.modelRevision, R0);
  assert.equal(projection.views.table.value.relations[0].id, "sales.order.lines");
  assert.equal(projection.views.er.edges[0].id, "sales.order.lines");
  assert.equal(projection.views.function.items[0].relation.id, "sales.order.lines");
  assert.equal(projection.views.business.status, "unresolved");
  assert.equal(projection.authority, "derived-readonly");
  assert.equal(projection.editable, false);
  assert.equal(JSON.stringify(world), before);
});

test("#1124: a visual edit from R0 cannot be silently rebased onto R1", () => {
  const edit = {
    schema: "tei.reference.studio-edit/v0", id: "add-backlink",
    origin: "ai", intent: "explicit backlink", view: "er",
    subject: "data:sales.line", operation: "add-relation",
    relationID: "sales.line.order", symbol: "order",
    targetData: "sales.order", cardinality: "one",
    expectedModelRevision: R0
  };
  assert.equal(candidateFromEdit(world, R0, edit).baseModelRevision, R0);
  assert.throws(() => candidateFromEdit(world, R1, edit),
    /STUDIO_EDIT_STALE_MODEL_REVISION/);
});

test("#1124: caller cannot pass a forged accepted Business projection", () => {
  const forged = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "accepted", canonicalProcedureUsed: true,
    procedureId: "P1", steps: [], diagnostics: []
  };
  assert.throws(() => projectWorld(world, R0, "relation:sales.order.lines",
    { businessDFD: forged }), /UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA/);
});
