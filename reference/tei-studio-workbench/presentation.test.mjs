import test from "node:test";
import assert from "node:assert/strict";
import {
  readCatalog,
  readValidation,
  readInspection,
  relatedGraph,
  readXeadObservation,
} from "./presentation.ts";
const data = {
  subject: { kind: "semantic-data", id: "sales.order" },
  sourceStatus: "indexed",
};
test("catalog rejects mutation projections and duplicate subjects", () => {
  assert.throws(
    () => readCatalog({ effect: "write", entries: [], modelRevision: "r" }),
    /INVALID_CATALOG/,
  );
  assert.throws(
    () =>
      readCatalog({
        effect: "read-only",
        entries: [data, data],
        modelRevision: "r",
        worldFingerprint: "world-r",
        sourceIndexFingerprint: "source-r",
      }),
    /INVALID_CATALOG_ENTRY/,
  );
});
test("validation preserves direct syntax errors, warnings, and empty results distinctly", () => {
  const error = { code: "PARSE", severity: "error", message: "invalid source" };
  assert.equal(
    readValidation({ Direct: error, Diagnostics: null })[0].severity,
    "error",
  );
  assert.deepEqual(readValidation({ Direct: null, Diagnostics: [] }), []);
  assert.equal(
    readValidation({
      Direct: null,
      Diagnostics: [{ diagnostic: { ...error, severity: "warning" } }],
    })[0].severity,
    "warning",
  );
  assert.throws(() => readValidation({}), /INVALID_VALIDATION/);
});
test("a response for a different subject cannot replace the selected projection", () => {
  const raw = {
    effect: "read-only",
    subject: { kind: "semantic", id: "other" },
    canonical: {},
    worldFingerprint: "w",
    sourceIndexFingerprint: "s",
  };
  assert.throws(() => readInspection(raw, data.subject), /INVALID_INSPECTION/);
});
test("graph contains actual catalog relations and reports clipped neighbours", () => {
  const children = Array.from({ length: 10 }, (_, i) => ({
    subject: { kind: "semantic-attribute", id: `sales.order.f${i}` },
    parent: data.subject,
    sourceStatus: "indexed",
  }));
  const graph = relatedGraph([...children, data], data.subject);
  assert.equal(graph.hidden, 3);
  assert.equal(graph.input.nodes.length, 8);
  assert.equal(graph.input.edges.length, 7);
  const keys = new Set(graph.input.nodes.map((n) => n.id));
  assert.ok(
    graph.input.edges.every((e) => keys.has(e.source) && keys.has(e.target)),
  );
});
test("XEAD presenter treats omitted collections as empty and exposes metadata only", () => {
  const source = { sourceKind: "table", sourceID: "T1" };
  const view = readXeadObservation({
    schemaVersion: "tei.operations.xead-observation/v0",
    effect: "read-only",
    authorityLimits: [
      "xead-observation-is-not-canonical-meaning",
      "xead-source-identifiers-are-not-canonical-identities",
      "xead-observation-does-not-authorize-edits-or-execution",
      "unsupported-source-constructs-remain-unresolved",
    ],
    observation: { tables: [{ source, name: "Orders", fields: [{ source: { sourceKind: "field", sourceID: "F1" }, name: "Number", dataType: { sourceKind: "type", sourceID: "string" }, value: "SECRET" }] }] },
  });
  assert.deepEqual(view, {
    tables: [{ source: { kind: "table", id: "T1" }, name: "Orders", fields: [{ source: { kind: "field", id: "F1" }, name: "Number", dataType: { kind: "type", id: "string" } }] }],
    diagnostics: [],
  });
  assert.throws(() => readXeadObservation({ effect: "read-only", schemaVersion: "v0", observation: { tables: null } }), /INVALID_XEAD/);
  assert.throws(() => readXeadObservation({ effect: "write", schemaVersion: "tei.operations.xead-observation/v0", authorityLimits: [], observation: {} }), /INVALID_XEAD/);
  assert.throws(() => readXeadObservation({ effect: "read-only", schemaVersion: "tei.operations.xead-observation/v1", authorityLimits: [], observation: {} }), /INVALID_XEAD/);
  assert.throws(() => readXeadObservation({ effect: "read-only", schemaVersion: "tei.operations.xead-observation/v0", observation: {} }), /INVALID_XEAD/);
  assert.throws(() => readXeadObservation({
    effect: "read-only", schemaVersion: "tei.operations.xead-observation/v0",
    authorityLimits: ["xead-observation-is-not-canonical-meaning"], observation: {},
  }), /INVALID_XEAD/);
  assert.throws(() => readXeadObservation({
    effect: "read-only", schemaVersion: "tei.operations.xead-observation/v0",
    authorityLimits: ["xead-observation-is-not-canonical-meaning", "xead-source-identifiers-are-not-canonical-identities", "xead-observation-does-not-authorize-edits-or-execution", "unsupported-source-constructs-remain-unresolved"],
    observation: { tables: [{ source, name: 5 }] },
  }), /INVALID_XEAD/);
});
