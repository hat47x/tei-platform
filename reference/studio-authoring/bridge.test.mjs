import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { projectWorld, candidateFromEdit, indexWorld } from "./bridge.mjs";

const revision = "sha256:" + "a".repeat(64);
function fixture() {
  return {
    semantics: [
      { id: "sales", data: [
        { id: "sales.order", attributes: [{ id: "sales.order.total", type: "std.decimal" }],
          relations: [{ id: "sales.order.lines", target: "sales.order_line", cardinality: "many" }] },
        { id: "sales.order_line", attributes: [
          { id: "sales.order_line.price", type: "std.decimal" }
        ] }
      ] },
      { id: "customer", data: [{ id: "customer.customer", attributes: [] }] }
    ],
    applications: [{ id: "order_entry", patterns: [
      { id: "order_entry.order_lines", kind: "header-detail", relation: "sales.order.lines" },
      { id: "order_entry.record", kind: "record", data: "sales.order" }
    ] }]
  };
}
function edit() {
  return { schema: "tei.reference.studio-edit/v0", id: "candidate-1",
    origin: "studio", intent: "明細から受注へ戻る参照を明示する",
    view: "er", subject: "data:sales.order_line", operation: "add-relation",
    relationID: "sales.order_line.order", symbol: "order",
    targetData: "sales.order", cardinality: "one" };
}

test("the same accepted relation ID is navigable across table, ER and function", () => {
  const world = fixture(), original = JSON.stringify(world);
  const result = projectWorld(world, revision, "relation:sales.order.lines");
  assert.equal(result.authority, "derived-readonly");
  assert.equal(result.editable, false);
  assert.equal(result.views.table.value.relations[0].id, "sales.order.lines");
  assert.equal(result.views.er.edges[0].id, "sales.order.lines");
  assert.equal(result.views.function.items[0].relation.id, "sales.order.lines");
  assert.equal(result.views.business.status, "unresolved");
  assert.equal(result.sourceNavigation.status, "unresolved");
  assert.equal(JSON.stringify(world), original, "projection must not mutate World");
});

test("diagram editing makes an existing Shared Acceptance change-candidate, not a source patch", () => {
  const world = fixture(), before = JSON.stringify(world);
  const out = candidateFromEdit(world, revision, edit());
  assert.deepEqual(out.delta, [{
    op: "add-relation", data: "sales.order_line", id: "sales.order_line.order",
    symbol: "order", target: "sales.order", cardinality: "one"
  }]);
  assert.equal(out.schema, "tei.reference.change-candidate/v0");
  assert.equal(out.baseModelRevision, revision);
  assert.equal(JSON.stringify(world), before);
  const ai = candidateFromEdit(world, revision, { ...edit(), origin: "ai" });
  assert.deepEqual(ai.delta, out.delta, "Studio/AI change enters the same boundary");
});

test("table edit uses source-owned attribute, never the derived table cell", () => {
  const proposed = candidateFromEdit(fixture(), revision, {
    schema: "tei.reference.studio-edit/v0", id: "candidate-2",
    origin: "studio", intent: "価格の正値制約を追加する", view: "table",
    subject: "attribute:sales.order_line.price", operation: "add-validation",
    validationID: "sales.order_line.price_positive", validationKind: "positive"
  });
  assert.deepEqual(proposed.delta, [{
    op: "add-validation", namespace: "sales",
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: "sales.order_line.price"
  }]);
});

test("accepted validations reappear in the table lens with their original ID", () => {
  const world = fixture();
  world.semantics[0].validations = [{
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: "sales.order_line.price"
  }];
  const projected = projectWorld(world, revision, "validation:sales.order_line.price_positive");
  assert.equal(projected.views.table.value.id, "sales.order_line");
  assert.deepEqual(projected.views.table.value.validations, [{
    subject: "validation:sales.order_line.price_positive",
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: { subject: "attribute:sales.order_line.price", id: "sales.order_line.price" },
    relation: null
  }]);
  assert.throws(() => candidateFromEdit(world, revision, {
    schema: "tei.reference.studio-edit/v0", id: "duplicate-validation",
    origin: "studio", intent: "既存Validationを再追加しない",
    view: "table", subject: "attribute:sales.order_line.price",
    operation: "add-validation", validationID: "sales.order_line.price_positive",
    validationKind: "positive"
  }), /DUPLICATE_ID/);
});

test("only source-compatible relation IDs are converted to candidate operations", () => {
  assert.throws(() => candidateFromEdit(fixture(), revision, {
    ...edit(), relationID: "sales.order_line.parent"
  }), /RELATION_ID_MUST_MATCH_SOURCE_SYMBOL/);
});

test("N05: direct derived-cell edits, arbitrary patches and layout edits fail closed", () => {
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), derived: true }), /DERIVED_IS_READONLY/);
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), operation: "set-layout" }), /UNSUPPORTED_EDIT/);
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), operation: "patch-file" }), /UNSUPPORTED_EDIT/);
});

test("invalid references, ambiguities and unsupported edits do not become meanings", () => {
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), targetData: "sales.missing" }), /INVALID_RELATION/);
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), relationID: "sales.order.lines" }), /DUPLICATE_ID/);
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), subject: "data:sales.missing" }), /UNRESOLVED_SUBJECT/);
  assert.throws(() => candidateFromEdit(fixture(), revision, { ...edit(), cardinality: "guessed" }), /INVALID_RELATION/);
  assert.throws(() => projectWorld(fixture(), revision, "relation:sales.missing"), /UNRESOLVED_SUBJECT/);
  assert.throws(() => projectWorld(fixture(), "not-a-revision", "data:sales.order"), /INVALID_REVISION/);
  const ambiguous = fixture();
  ambiguous.semantics[0].data.push({ id: "sales.order", attributes: [] });
  assert.throws(() => indexWorld(ambiguous), /AMBIGUOUS_ID/);
});

test("broken Relation target is unresolved, never silently materialized as Data", () => {
  const world = fixture();
  world.semantics[0].data[0].relations[0].target = "missing.order_line";
  const p = projectWorld(world, revision, "relation:sales.order.lines");
  assert.equal(p.views.er.edges[0].status, "unresolved");
  assert.equal(p.views.er.edges[0].target, null);
});

test("dangling Function relation stays unresolved rather than becoming absent", () => {
  const world = fixture();
  world.applications[0].patterns[0].relation = "sales.unknown_relation";
  const result = projectWorld(world, revision, "pattern:order_entry.order_lines");
  const selected = result.views.function.items[0];
  assert.equal(selected.relationReferenceStatus, "unresolved");
  assert.equal(selected.unresolvedRelationID, "sales.unknown_relation");
  assert.equal(selected.relation, null);
});

test("the existing Source Index supplies a precise definition locator without changing identity", () => {
  const source = {
    artifacts: [{ path: "semantic/sales/order.tei",
      fingerprint: "sha256:" + "a".repeat(64) }],
    references: [{
      subject: { kind: "semantic-relation", id: "sales.order.lines" },
      artifactPath: "semantic/sales/order.tei",
      role: "definition",
      range: {
        start: { offset: 120, line: 7, column: 14 },
        end: { offset: 125, line: 7, column: 19 }
      }
    }]
  };
  const p = projectWorld(fixture(), revision, "relation:sales.order.lines", { sourceIndex: source });
  assert.equal(p.sourceNavigation.status, "source-index-backed");
  assert.equal(p.sourceNavigation.subject.id, "sales.order.lines");
  assert.equal(p.sourceNavigation.subject.kind, "semantic-relation");
  assert.equal(p.sourceNavigation.artifactPath, "semantic/sales/order.tei");
  assert.equal(p.sourceNavigation.range.start.line, 7);

  assert.equal(projectWorld(fixture(), revision, "data:sales.order",
    { sourceIndex: source }).sourceNavigation.status, "unresolved");
  assert.throws(() => projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex: { artifacts: [], references: {} } }), /INVALID_SOURCE_INDEX/);
  assert.throws(() => projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex: { ...source, references: [...source.references, source.references[0]] } }),
    /AMBIGUOUS_SOURCE_DEFINITION/);
  assert.throws(() => projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex: { ...source, artifacts: [{ ...source.artifacts[0],
      path: "../escape" }] } }), /INVALID_OR_DUPLICATE_SOURCE_ARTIFACT/);
  assert.throws(() => projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex: { ...source, references: [{
      ...source.references[0], range: { start: { offset: -1, line: 0, column: 0 },
        end: { offset: 1, line: 1, column: 2 } }
    }] } }), /INVALID_SOURCE_ANCHOR/);
});

test("optional #1125 Business reference joins exact Pattern ID without becoming accepted", () => {
  const world = fixture();
  const dfd = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "order-entry.order-handling", diagnostics: [],
    steps: [
      { id: "BP-03", noApplicationFunction: false,
        functionRefs: [{ id: "order_entry.order_lines", status: "resolved",
          relation: "sales.order.lines" }],
        source: { file: "fixture.json", pointer: "/steps/3" } },
      { id: "BP-04", noApplicationFunction: true,
        realization: "manual-unresolved", functionRefs: [],
        source: { file: "fixture.json", pointer: "/steps/4" } }
    ]
  };
  const p = projectWorld(world, revision, "relation:sales.order.lines", { businessDFD: dfd });
  assert.equal(p.views.business.status, "reference-only");
  assert.equal(p.views.business.authority, "noncanonical-reference");
  assert.equal(p.views.business.freshness, "unverified-no-model-revision-binding");
  assert.equal(p.views.business.items[0].functionRefs[0].id, "order_entry.order_lines");
  assert.equal(p.views.business.items[0].source.pointer, "/steps/3");
  assert.equal(p.views.business.manualOrExternalUnlinked[0].stepID, "BP-04");
  assert.equal(p.views.business.items.some(v => v.stepID === "BP-04"), false,
    "manual step must not be linked by proximity");
  assert.throws(() => projectWorld(world, revision, "relation:sales.order.lines", {
    businessDFD: { ...dfd, canonicalProcedureUsed: true }
  }), /UNSUPPORTED_BUSINESS_DFD/);
  const mismatch = structuredClone(dfd);
  mismatch.steps[0].functionRefs[0].relation = "sales.missing";
  const m = projectWorld(world, revision, "relation:sales.order.lines", { businessDFD: mismatch });
  assert.equal(m.views.business.status, "unresolved");
  assert.equal(m.views.business.items[0].functionRefs[0].status, "unresolved");
  const duplicated = structuredClone(dfd);
  duplicated.steps.push(structuredClone(duplicated.steps[0]));
  assert.throws(() => projectWorld(world, revision, "relation:sales.order.lines", {
    businessDFD: duplicated
  }), /INVALID_OR_DUPLICATE_BUSINESS_STEP/);
});

test("local selection never expands into a global dense matrix", () => {
  const world = fixture();
  for (let i = 0; i < 2000; i++) {
    world.semantics[0].data.push({ id: "sales.item_" + i, attributes: [] });
  }
  const result = projectWorld(world, revision, "data:sales.order_line");
  assert.ok(result.views.er.edges.length <= 1);
  assert.ok(result.views.function.items.length <= 1);
});

test("CLI consumes snapshot artifacts without mutating inputs", () => {
  const dir = mkdtempSync(join(tmpdir(), "tei-studio-reference-"));
  try {
    const worldPath = join(dir, "world.json"), revPath = join(dir, "revision.json");
    const editPath = join(dir, "edit.json");
    writeFileSync(worldPath, JSON.stringify(fixture()));
    writeFileSync(revPath, JSON.stringify({ modelRevision: revision }));
    writeFileSync(editPath, JSON.stringify(edit()));
    const sourcePath = join(dir, "source-index.json");
    writeFileSync(sourcePath, JSON.stringify({
      artifacts: [{ path: "semantic/sales/order.tei",
        fingerprint: "sha256:" + "1".repeat(64) }],
      references: [{
        subject: { kind: "semantic-relation", id: "sales.order.lines" },
        role: "definition", artifactPath: "semantic/sales/order.tei",
        range: { start: { offset: 2, line: 1, column: 3 },
          end: { offset: 7, line: 1, column: 8 } }
      }]
    }));
    const paths = [worldPath, revPath, editPath, sourcePath];
    const before = paths.map(p => readFileSync(p, "utf8"));
    const cli = fileURLToPath(new URL("./cli.mjs", import.meta.url));
    const inspect = spawnSync(process.execPath, [cli, "inspect", "-world-file", worldPath,
      "-revision-file", revPath, "-focus", "relation:sales.order.lines"], { encoding: "utf8" });
    assert.equal(inspect.status, 0, inspect.stderr);
    assert.equal(JSON.parse(inspect.stdout).views.function.items[0].id, "order_entry.order_lines");
    const inspectIndexed = spawnSync(process.execPath, [cli, "inspect",
      "-world-file", worldPath, "-revision-file", revPath,
      "-source-index-file", sourcePath, "-focus", "relation:sales.order.lines"],
    { encoding: "utf8" });
    assert.equal(inspectIndexed.status, 0, inspectIndexed.stderr);
    assert.equal(JSON.parse(inspectIndexed.stdout).sourceNavigation.status, "source-index-backed");
    const propose = spawnSync(process.execPath, [cli, "propose", "-world-file", worldPath,
      "-revision-file", revPath, "-edit-file", editPath], { encoding: "utf8" });
    assert.equal(propose.status, 0, propose.stderr);
    assert.equal(JSON.parse(propose.stdout).delta[0].op, "add-relation");
    assert.deepEqual(paths.map(p => readFileSync(p, "utf8")), before);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
