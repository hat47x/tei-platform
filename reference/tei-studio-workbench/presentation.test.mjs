import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  readCatalog,
  inspectEligibility,
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
    totalTables: 1,
    totalFields: 1,
    totalDiagnostics: 0,
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
test("XEAD presenter caps rendered facts but reports exact totals", () => {
  const source = { sourceKind: "table", sourceID: "T1" };
  const fieldSource = { sourceKind: "field", sourceID: "F1" };
  const diagnostic = { code: "unsupported-element", element: "View", occurrences: 1 };
  const limits = [
    "xead-observation-is-not-canonical-meaning",
    "xead-source-identifiers-are-not-canonical-identities",
    "xead-observation-does-not-authorize-edits-or-execution",
    "unsupported-source-constructs-remain-unresolved",
  ];
  const view = readXeadObservation({
    schemaVersion: "tei.operations.xead-observation/v0", effect: "read-only", authorityLimits: limits,
    observation: {
      tables: Array.from({ length: 202 }, (_, i) => ({ source: { ...source, sourceID: `T${i}` }, fields: Array.from({ length: 11 }, (_, j) => ({ source: { ...fieldSource, sourceID: `F${i}-${j}` } })) })),
      diagnostics: Array.from({ length: 502 }, () => diagnostic),
    },
  });
  assert.deepEqual([view.totalTables, view.tables.length], [202, 200]);
  assert.deepEqual([view.totalFields, view.tables.reduce((n, t) => n + t.fields.length, 0)], [2222, 2000]);
  assert.deepEqual([view.totalDiagnostics, view.diagnostics.length], [502, 500]);
});

test("filtered and duplicate Catalog ambiguity remains visible and blocks Inspect", () => {
  const catalog = entries => readCatalog({
    effect: "read-only", entries, modelRevision: "rev-A",
    worldFingerprint: "world-A", sourceIndexFingerprint: "source-A",
  });
  const flagged = { ...data, ambiguous: true };
  const filtered = catalog([flagged]);
  assert.equal(filtered.entries.length, 1);
  assert.equal(filtered.entries[0].ambiguous, true);
  assert.deepEqual(inspectEligibility(filtered, data.subject),
    { ok: false, reason: "SUBJECT_AMBIGUOUS" });
  const both = catalog([flagged, { ...flagged, symbol: "other scope" }]);
  assert.equal(both.entries.length, 2);
  assert.ok(both.entries.every(e => e.ambiguous));
  assert.deepEqual(inspectEligibility(both, data.subject),
    { ok: false, reason: "SUBJECT_AMBIGUOUS" });
  assert.throws(() => catalog([flagged, { ...flagged, ambiguous: false }]),
    /INVALID_CATALOG_ENTRY/);
  assert.throws(() => catalog([{ ...flagged, ambiguous: "true" }]),
    /INVALID_CATALOG_ENTRY/);
});

test("unique, missing and unavailable Catalog subjects have distinct Inspect guards", () => {
  const catalog = readCatalog({
    effect: "read-only", entries: [data], modelRevision: "rev-A",
    worldFingerprint: "world-A", sourceIndexFingerprint: "source-A",
  });
  assert.equal(catalog.entries[0].ambiguous, false);
  assert.deepEqual(inspectEligibility(catalog, data.subject), { ok: true });
  assert.deepEqual(inspectEligibility(catalog, { kind: "semantic-data", id: "unknown" }),
    { ok: false, reason: "SUBJECT_NOT_FOUND" });
  assert.deepEqual(inspectEligibility(null, data.subject),
    { ok: false, reason: "CATALOG_UNAVAILABLE" });
});

test("Workbench common Inspect guard runs before any unqualified transport call", () => {
  // Static wiring check only. Four-route UI events and actual WebView remain
  // for subsequent Consumer integration; this is not P0 exit evidence.
  const workbench = readFileSync(new URL("./main.ts", import.meta.url), "utf8");
  const start = workbench.indexOf("async function inspect(subject: Subject)");
  const end = workbench.indexOf("async function validate(", start);
  assert.ok(start > 0 && end > start);
  const body = workbench.slice(start, end);
  assert.ok(body.indexOf("inspectEligibility(catalogSnapshot, subject)") >= 0);
  assert.ok(body.indexOf("if (!eligibility.ok)") <
    body.indexOf("client!.inspectSubject("));
  assert.match(body, /responses\.invalidate\("inspection"\)/);
  assert.match(body, /return false;/);
});
