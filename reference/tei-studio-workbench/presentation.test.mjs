import test from "node:test";
import assert from "node:assert/strict";
import {
  readCatalog,
  readValidation,
  readInspection,
  relatedGraph,
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
