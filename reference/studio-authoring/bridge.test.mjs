import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { projectWorld, candidateFromEdit, indexWorld, bindReviewArtifacts, navigateReferenceBusinessStep, verifySourceAnchor } from "./bridge.mjs";

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
    op: "add-relation", addRelation: {
      data: "sales.order_line", id: "sales.order_line.order",
      symbol: "order", target: "sales.order", cardinality: "one"
    }
  }]);
  assert.equal(Object.hasOwn(out.delta[0], "data"), false,
    "root data means a Go DataSpec; relation needs addRelation");
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
    applicationSelection: [], evaluationAuthority: "not-verified-by-projection",
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

test("a rule alone is not an Application-selected Command validation", () => {
  const world = fixture();
  world.applications[0].semanticUses = ["sales"];
  const baseline = {
    schema: "tei.reference.studio-edit/v0", id: "price-positive",
    origin: "studio", intent: "正値Validationを提案する", view: "table",
    subject: "attribute:sales.order_line.price", operation: "add-validation",
    validationID: "sales.order_line.price_positive", validationKind: "positive"
  };
  const alone = candidateFromEdit(world, revision, baseline);
  assert.equal(alone.delta.length, 1);
  assert.equal(alone.delta[0].op, "add-validation");
  // A semantic rule not selected by the Application is not equivalent to
  // Driver evaluation. The Bridge never reports execution readiness.
  const p = projectWorld(world, revision, "data:sales.order_line");
  assert.equal(p.realizationReadiness.status, "not-verified");

  const paired = candidateFromEdit(world, revision, {
    ...baseline,
    applicationUse: {
      application: "order_entry",
      placement: "applications/order-entry/application.tei"
    }
  });
  assert.deepEqual(paired.delta, [
    alone.delta[0],
    { op: "add-application-validation-use",
      placement: "applications/order-entry/application.tei",
      applicationValidationUse: {
        application: "order_entry", validation: "sales.order_line.price_positive"
      } }
  ]);
  const ai = candidateFromEdit(world, revision, {
    ...baseline, origin: "ai",
    applicationUse: {
      application: "order_entry",
      placement: "applications/order-entry/application.tei"
    }
  });
  assert.deepEqual(ai.delta, paired.delta);
});

test("paired rule/use rejects missing scope, invalid Application and unsafe paths", () => {
  const world = fixture();
  world.applications[0].semanticUses = ["sales"];
  const edit = {
    schema: "tei.reference.studio-edit/v0", id: "pair",
    origin: "studio", intent: "rule and selection", view: "table",
    subject: "attribute:sales.order_line.price",
    operation: "add-validation", validationID: "sales.order_line.price_positive",
    validationKind: "positive", applicationUse: {
      application: "order_entry", placement: "applications/order-entry/application.tei"
    }
  };
  assert.throws(() => candidateFromEdit(world, revision,
    { ...edit, applicationUse: { ...edit.applicationUse, application: "missing" } }),
    /APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE/);
  assert.throws(() => candidateFromEdit(world, revision,
    { ...edit, applicationUse: { ...edit.applicationUse, placement: "../../reference/world.json" } }),
    /UNSAFE_APPLICATION_SOURCE_PLACEMENT/);
  world.applications[0].semanticUses = ["customer"];
  assert.throws(() => candidateFromEdit(world, revision, edit),
    /APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE/);
});

test("accepted rule selection displays declared use without claiming actual target effect", () => {
  const world = fixture();
  world.semantics[0].validations = [{
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: "sales.order_line.price"
  }];
  world.applications[0].validations = ["sales.order_line.price_positive"];
  const p = projectWorld(world, revision, "validation:sales.order_line.price_positive");
  const v = p.views.table.value.validations[0];
  assert.deepEqual(v.applicationSelection,
    [{ application: "order_entry", status: "source-declared" }]);
  assert.equal(v.evaluationAuthority, "not-verified-by-projection");
  assert.equal(p.realizationReadiness.status, "not-verified");
  assert.ok(p.realizationReadiness.sourceAcceptedDoesNotImply.includes("provider-write"));
});

test("review-edit refuses mismatched Preview/Review or stale base; no Approval implied", () => {
  const candidate = candidateFromEdit(fixture(), revision, edit());
  const fingerprint = "sha256:" + "b".repeat(64);
  const reviewFingerprint = "sha256:" + "c".repeat(64);
  const preview = { schema: "tei.reference.change-preview/v0",
    candidateId: candidate.id, baseModelRevision: revision,
    currentModelRevision: revision, candidateFingerprint: fingerprint,
    acceptable: true };
  const review = { schema: "tei.reference.approval-review/v0",
    candidateId: candidate.id, candidateFingerprint: fingerprint,
    reviewFingerprint, previewAcceptable: true };
  const coherent = bindReviewArtifacts(candidate, preview, review, revision);
  assert.equal(coherent.status, "reviewed-consistent");
  assert.equal(coherent.readyToAccept, false,
    "showing the reviewed projection does not record approval");
  assert.throws(() => bindReviewArtifacts(candidate, preview,
    { ...review, candidateFingerprint: "sha256:" + "f".repeat(64) }, revision),
    /REVIEW_BINDING_MISMATCH/);
  assert.throws(() => bindReviewArtifacts(candidate,
    { ...preview, currentModelRevision: "sha256:" + "d".repeat(64) },
    review, revision), /REVIEW_BINDING_MISMATCH/);
  assert.throws(() => bindReviewArtifacts(candidate, preview,
    { ...review, previewAcceptable: false }, revision),
    /REVIEW_BINDING_MISMATCH/);
  assert.throws(() => bindReviewArtifacts(candidate, preview,
    { ...review, evidenceFingerprint: "sha256:" + "e".repeat(64) }, revision),
    /REVIEW_BINDING_MISMATCH/);
  const rejected = bindReviewArtifacts(candidate,
    { ...preview, acceptable: false },
    { ...review, previewAcceptable: false }, revision);
  assert.equal(rejected.status, "not-acceptable");
  assert.equal(rejected.readyToAccept, false);
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

test("source index rejects unnormalized path and incomplete end positions before displaying an anchor", () => {
  const source = {
    artifacts: [{ path: "semantic/sales/order.tei",
      fingerprint: "sha256:" + "a".repeat(64) }],
    references: [{
      subject: { kind: "semantic-relation", id: "sales.order.lines" },
      artifactPath: "semantic/sales/order.tei", role: "definition",
      range: { start: { offset: 2, line: 1, column: 3 },
        end: { offset: 7, line: 1, column: 8 } }
    }]
  };
  const run = input => projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex: input });
  assert.equal(run(source).sourceNavigation.byteVerification.status, "not-checked");
  for (const path of ["semantic//sales/order.tei", "semantic/./sales/order.tei",
    "semantic/sales/order.tei/", "./semantic/sales/order.tei"]) {
    assert.throws(() => run({ ...source, artifacts: [{
      ...source.artifacts[0], path }] }), /INVALID_OR_DUPLICATE_SOURCE_ARTIFACT/);
  }
  for (const end of [{ offset: 7, line: 0, column: 8 },
    { offset: 7, line: 1, column: 0 }, { offset: 7, line: -1, column: 8 },
    { offset: 7, line: 1.5, column: 8 }, { offset: 7, line: 1, column: 2 },
    { offset: 7, line: 0, column: 1 }]) {
    const invalid = structuredClone(source);
    invalid.references[0].range.end = end;
    assert.throws(() => run(invalid), /INVALID_SOURCE_ANCHOR/);
  }
});

test("nested subject scope cannot be flattened to an unscoped Source Index definition", () => {
  const base = {
    artifacts: [{ path: "semantic/sales/order.tei",
      fingerprint: "sha256:" + "b".repeat(64) }],
    references: [{
      subject: { kind: "semantic-relation", id: "sales.order.lines" },
      artifactPath: "semantic/sales/order.tei", role: "definition",
      range: { start: { offset: 2, line: 1, column: 3 },
        end: { offset: 7, line: 1, column: 8 } }
    }]
  };
  const run = sourceIndex => projectWorld(fixture(), revision,
    "relation:sales.order.lines", { sourceIndex });
  assert.equal(run(base).sourceNavigation.status, "source-index-backed");
  const nested = structuredClone(base);
  nested.references[0].subject.scope = "app:order_entry";
  const ref = run(nested).sourceNavigation;
  assert.equal(ref.status, "unresolved");
  assert.ok(ref.reason.includes("#1120"));
  const duplicateScope = structuredClone(base);
  duplicateScope.references[0].subject.scope = "app:one";
  duplicateScope.references.push({
    ...structuredClone(duplicateScope.references[0]),
    subject: { ...duplicateScope.references[0].subject, scope: "app:two" }
  });
  assert.equal(run(duplicateScope).sourceNavigation.status, "unresolved",
    "same kind and ID with different scoped meanings must never select either");
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

test("C0 scoped ID collision cannot be silently resolved by flat Studio token", () => {
  const anchor = {
    subject: { kind: "semantic-relation", id: "sales.order.lines" },
    role: "definition", artifactPath: "semantic/sales/order.tei",
    range: { start: { offset: 1, line: 1, column: 1 },
      end: { offset: 2, line: 1, column: 2 } }
  };
  const sourceIndex = {
    artifacts: [{ path: "semantic/sales/order.tei",
      fingerprint: "sha256:" + "f".repeat(64) }],
    references: [
      { ...anchor, scope: "tenant-a" },
      { ...anchor, scope: "tenant-b" }
    ]
  };
  const result = projectWorld(fixture(), revision, "relation:sales.order.lines",
    { sourceIndex });
  assert.equal(result.sourceNavigation.status, "unresolved");
  assert.ok(result.sourceNavigation.reason.includes("typed Source Resolver"));
});

test("even a single scoped Source locator is unresolved without typed C0 resolver", () => {
  const anchor = {
    subject: { kind: "semantic-relation", id: "sales.order.lines" },
    role: "definition", scope: "orders-v1",
    artifactPath: "semantic/sales/order.tei",
    range: { start: { offset: 1, line: 1, column: 1 },
      end: { offset: 2, line: 1, column: 2 } }
  };
  const index = {
    artifacts: [{ path: "semantic/sales/order.tei",
      fingerprint: "sha256:" + "f".repeat(64) }],
    references: [anchor]
  };
  const p = projectWorld(fixture(), revision, "relation:sales.order.lines", {
    sourceIndex: index
  });
  assert.equal(p.sourceNavigation.status, "unresolved");
  assert.ok(p.sourceNavigation.reason.includes("typed Source Resolver"));
});

test("reference-local Business step ID matching a Pattern ID is not an accepted Function", () => {
  const source = fixture();
  const dfd = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "local-procedure", diagnostics: [],
    steps: [{ id: "order_entry.order_lines", noApplicationFunction: false, functionRefs: [] }]
  };
  const p = projectWorld(source, revision, "relation:sales.order.lines",
    { businessDFD: dfd });
  assert.equal(p.views.business.authority, "noncanonical-reference");
  assert.equal(p.views.business.items.length, 0);
  assert.equal(p.views.business.freshness, "unverified-no-model-revision-binding");
});

test("a manual Business Step can point directly to exact Validation without acquiring a Function", () => {
  const w = fixture();
  w.semantics[0].validations = [{
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: "sales.order_line.price"
  }];
  const dfd = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "order-entry.order-handling", diagnostics: [],
    steps: [{
      id: "BP-05", realization: "validation-support", noApplicationFunction: true,
      functionRefs: [], explicitMeaning: [
        { kind: "semantic-validation", id: "sales.order_line.price_positive",
          status: "resolved-id-not-accepted", authority: "procedure-reference-only",
          source: { file: "procedure-reference.json", pointer: "/steps/5/validations/0" } },
        { kind: "semantic-data", id: "sales.order_line",
          status: "resolved-id-not-accepted", authority: "procedure-reference-only" }
      ]
    }]
  };
  const p = projectWorld(w, revision, "validation:sales.order_line.price_positive",
    { businessDFD: dfd });
  assert.equal(p.views.business.status, "reference-only");
  assert.equal(p.views.business.authority, "noncanonical-reference");
  assert.equal(p.views.business.items[0].stepID, "BP-05");
  assert.deepEqual(p.views.business.items[0].functionRefs, []);
  assert.equal(p.views.business.items[0].explicitMeaning[0].subject,
    "validation:sales.order_line.price_positive");
  assert.equal(p.views.business.items[0].explicitMeaning[0].status,
    "exact-id-matches-current-world");
  assert.equal(p.views.business.manualOrExternalUnlinked[0].stepID, "BP-05");
  assert.equal(p.views.business.freshness, "unverified-no-model-revision-binding");
  assert.equal(p.editable, false);
  // The same explicit step can be reached from Data without inventing a
  // dependency on a sibling Field or a generated Function.
  const onData = projectWorld(w, revision, "data:sales.order_line",
    { businessDFD: dfd });
  assert.equal(onData.views.business.items[0].explicitMeaning[0].subject,
    "data:sales.order_line");
  const unrelated = projectWorld(w, revision, "data:sales.order",
    { businessDFD: dfd });
  assert.equal(unrelated.views.business.items.length, 0,
    "same Procedure does not imply every Data depends on each step");
});

test("stale, spoofed, or wrong-kind Business direct references cannot become linked meaning", () => {
  const w = fixture();
  w.semantics[0].validations = [{
    id: "sales.order_line.price_positive", kind: "positive",
    attribute: "sales.order_line.price"
  }];
  const base = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "p", diagnostics: [],
    steps: [{ id: "BP-05", noApplicationFunction: true, functionRefs: [],
      explicitMeaning: [{ kind: "semantic-validation",
        id: "sales.order_line.price_positive",
        status: "resolved-id-not-accepted", authority: "procedure-reference-only" }] }]
  };
  const focus = "validation:sales.order_line.price_positive";
  const stale = structuredClone(base);
  stale.steps[0].explicitMeaning[0].status = "unresolved";
  const s = projectWorld(w, revision, focus, { businessDFD: stale });
  assert.equal(s.views.business.status, "unresolved");
  assert.equal(s.views.business.items[0].explicitMeaning[0].status, "unresolved");
  assert.equal(s.views.business.diagnostics[0].code,
    "BUSINESS_DIRECT_MEANING_STALE_OR_UNRESOLVED");
  const forged = structuredClone(base);
  forged.steps[0].explicitMeaning[0].authority = "accepted";
  assert.equal(projectWorld(w, revision, focus, {
    businessDFD: forged }).views.business.status, "unresolved");
  const wrongKind = structuredClone(base);
  wrongKind.steps[0].explicitMeaning[0].kind = "semantic-relation";
  assert.equal(projectWorld(w, revision, focus, {
    businessDFD: wrongKind }).views.business.items.length, 0);
  const malformed = structuredClone(base);
  malformed.steps[0].explicitMeaning = { id: "sales.order_line.price_positive" };
  assert.throws(() => projectWorld(w, revision, focus, {
    businessDFD: malformed }), /step.explicitMeaning: array required/);
  const inventedKind = structuredClone(base);
  inventedKind.steps[0].explicitMeaning[0].kind = "unknown-business-meaning";
  assert.throws(() => projectWorld(w, revision, focus, {
    businessDFD: inventedKind }), /INVALID_BUSINESS_DIRECT_MEANING/);
  const fabricatedState = structuredClone(base);
  fabricatedState.steps[0].explicitMeaning[0].status = "accepted";
  assert.throws(() => projectWorld(w, revision, focus, {
    businessDFD: fabricatedState }), /INVALID_BUSINESS_DIRECT_MEANING/);
  assert.throws(() => projectWorld(w, revision, focus, {
    businessDFD: { ...base, procedureId: "" }
  }), /UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA/);
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

test("source anchor verification binds digest to the actual byte buffer", () => {
  const bytes = Uint8Array.from([97, 10, 98, 99, 10]); // a\\nbc\\n
  const digest = data => "sha256:" +
    createHash("sha256").update(data).digest("hex");
  const fingerprint = digest(bytes);
  const navigation = {
    status: "source-index-backed",
    artifactFingerprint: fingerprint,
    range: {
      start: { offset: 2, line: 2, column: 1 },
      end: { offset: 4, line: 2, column: 3 }
    }
  };
  const verified = verifySourceAnchor(navigation, bytes, digest);
  assert.equal(verified.status, "source-index-backed");
  assert.equal(verified.byteVerification.status, "verified-at-read");
  assert.equal(verified.byteVerification.fingerprint, fingerprint);
  assert.equal(verified.byteVerification.range, "offset-and-line-column-matched");
  assert.ok(verified.caveat.includes("no filesystem lock"));
  assert.equal(navigation.byteVerification, undefined,
    "the read-only navigation input must not be modified");

  // The checksum is calculated over the same bytes passed to the locator
  // check. A matching arbitrary fingerprint supplied separately is no
  // longer sufficient to mark the anchor as verified.
  const changed = Uint8Array.from([97, 10, 88, 99, 10]);
  assert.throws(() => verifySourceAnchor(navigation, changed, digest),
    /SOURCE_ARTIFACT_FINGERPRINT_STALE/);
  const wrongChecksum = structuredClone(navigation);
  wrongChecksum.artifactFingerprint = "sha256:" + "f".repeat(64);
  assert.throws(() => verifySourceAnchor(wrongChecksum, bytes, digest),
    /SOURCE_ARTIFACT_FINGERPRINT_STALE/);
  assert.throws(() => verifySourceAnchor(navigation, bytes, () => "invalid"),
    /SOURCE_ANCHOR_DIGEST_INVALID/);
  assert.throws(() => verifySourceAnchor(navigation, bytes, fingerprint),
    /SOURCE_ANCHOR_VERIFICATION_INPUT_INVALID/);

  const staleOffset = structuredClone(navigation);
  staleOffset.range.end.offset = 6;
  assert.throws(() => verifySourceAnchor(staleOffset, bytes, digest),
    /SOURCE_ANCHOR_RANGE_INVALID/);
  const staleLine = structuredClone(navigation);
  staleLine.range.start.line = 1;
  assert.throws(() => verifySourceAnchor(staleLine, bytes, digest),
    /SOURCE_ANCHOR_POSITION_STALE/);
  const staleEnd = structuredClone(navigation);
  staleEnd.range.end.column = 4;
  assert.throws(() => verifySourceAnchor(staleEnd, bytes, digest),
    /SOURCE_ANCHOR_POSITION_STALE/);
  assert.throws(() => verifySourceAnchor(navigation, "untrusted text", digest),
    /SOURCE_ANCHOR_VERIFICATION_INPUT_INVALID/);
});

test("source index columns are byte-based even for multibyte Japanese text", () => {
  // Go sourceindex.PositionAt counts bytes rather than JavaScript UTF-16 units.
  const content = Uint8Array.from([0xe3, 0x81, 0x82, 0x0a, 0x61]); // あ\\na
  const digest = data => "sha256:" +
    createHash("sha256").update(data).digest("hex");
  const position = {
    status: "source-index-backed", artifactFingerprint: digest(content),
    range: {
      start: { offset: 3, line: 1, column: 4 },
      end: { offset: 4, line: 2, column: 1 }
    }
  };
  const result = verifySourceAnchor(position, content, digest);
  assert.equal(result.byteVerification.status, "verified-at-read");
  const characterColumn = structuredClone(position);
  characterColumn.range.start.column = 2; // UTF-16 character count: incorrect
  assert.throws(() => verifySourceAnchor(characterColumn, content, digest),
    /SOURCE_ANCHOR_POSITION_STALE/);
});

test("business-local step navigation returns typed World focus without canonical authority", () => {
  const world = fixture();
  world.semantics[0].validations = [
    { id: "sales.order_line.price_positive", kind: "positive",
      attribute: "sales.order_line.price" }
  ];
  const dfd = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "order-entry.order-handling", diagnostics: [],
    steps: [{
      id: "BP-05", noApplicationFunction: true, functionRefs: [],
      source: { file: "ref-business.json", pointer: "/steps/5" },
      explicitMeaning: [
        { kind: "semantic-validation", id: "sales.order_line.price_positive",
          status: "resolved-id-not-accepted", authority: "procedure-reference-only",
          source: { file: "ref-business.json", pointer: "/steps/5/validations/0" } },
        { kind: "semantic-capability", id: "security.authorize",
          status: "unresolved", authority: "procedure-reference-only" }
      ]
    }]
  };
  const original = JSON.stringify(world);
  const result = navigateReferenceBusinessStep(world, revision, dfd, "BP-05");
  assert.equal(result.schema, "tei.reference.studio-business-step-navigation/v0");
  assert.equal(result.subject.scope, "reference-procedure:order-entry.order-handling");
  assert.equal(result.status, "unresolved",
    "mixed current and unresolved references cannot look fully sound");
  assert.deepEqual(result.resolutionSummary,
    { matched: 1, unresolved: 1, producerErrors: 0 });
  assert.equal(result.candidates[0].focus, "validation:sales.order_line.price_positive");
  assert.equal(result.candidates[0].status, "identity-match-only");
  assert.equal(result.candidates[1].status, "unresolved");
  assert.equal(result.editable, false);
  assert.equal(result.acceptanceTargetStatus, "unresolved-use-shared-acceptance");
  assert.equal(result.freshness, "unverified-no-model-revision-binding");
  assert.equal(JSON.stringify(world), original);
  assert.equal(projectWorld(world, revision, result.candidates[0].focus).focus.subject,
    result.candidates[0].focus);
  assert.throws(() => navigateReferenceBusinessStep(world, revision, dfd, "BP-MISSING"),
    /UNRESOLVED_REFERENCE_BUSINESS_STEP/);
});

test("business-local step navigation fails closed on DFD errors, duplicate IDs and forged authority", () => {
  const world = fixture();
  const base = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: "local", diagnostics: [],
    steps: [{ id: "S1", functionRefs: [{ id: "order_entry.record",
      status: "resolved", data: "sales.order" }] }]
  };
  const ok = navigateReferenceBusinessStep(world, revision, base, "S1");
  assert.equal(ok.status, "reference-only");
  assert.deepEqual(ok.resolutionSummary,
    { matched: 1, unresolved: 0, producerErrors: 0 });
  assert.equal(ok.candidates[0].focus, "pattern:order_entry.record");
  assert.equal(ok.candidates[0].status, "identity-match-only");
  assert.equal(ok.editable, false);
  const stale = structuredClone(base);
  stale.steps[0].functionRefs[0].data = "sales.order_line";
  assert.equal(navigateReferenceBusinessStep(world, revision, stale, "S1")
    .candidates[0].status, "unresolved");
  const producerError = structuredClone(base);
  producerError.diagnostics = [{ code: "INVALID", severity: "error" }];
  const blocked = navigateReferenceBusinessStep(world, revision, producerError, "S1");
  assert.equal(blocked.status, "unresolved");
  assert.equal(blocked.candidates[0].status, "unresolved");
  assert.deepEqual(blocked.resolutionSummary,
    { matched: 0, unresolved: 1, producerErrors: 1 });
  const duplicate = structuredClone(base);
  duplicate.steps.push(structuredClone(duplicate.steps[0]));
  assert.throws(() => navigateReferenceBusinessStep(world, revision, duplicate, "S1"),
    /INVALID_OR_DUPLICATE_BUSINESS_STEP/);
  const forged = structuredClone(base);
  forged.authority = "accepted";
  assert.throws(() => navigateReferenceBusinessStep(world, revision, forged, "S1"),
    /UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA/);
});

test("real Reference World and Procedure fixture retain explicit non-Function Validation step", () => {
  // This is a data-level fixture reconstruction of the optional #1125 DFD
  // projection shape, not a Go DFD producer execution or Business Acceptance.
  const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
  const world = JSON.parse(readFileSync(join(root, "reference/world.json"), "utf8"));
  const fixture = JSON.parse(readFileSync(join(root,
    "reference/business-system-design/order-entry-business-procedure.json"), "utf8"));
  assert.equal(fixture.canonical, false);
  assert.equal(fixture.artifactRole, "reference-authoring-fixture");
  const step = fixture.steps.find(x => x.id === "BP-05");
  assert.ok(step && !(step.applicationPatterns || []).length);
  assert.ok(step.validations.includes("sales.order_line.quantity_positive"));
  const businessDFD = {
    schemaVersion: "business-dfd-projection-v0",
    authority: "derived-read-only", canonicalProcedureUsed: false,
    procedureId: fixture.procedure.id, diagnostics: [],
    steps: [{
      id: step.id, realization: step.realization,
      noApplicationFunction: true, functionRefs: [],
      explicitMeaning: step.validations.map((id, i) => ({
        kind: "semantic-validation", id,
        status: "resolved-id-not-accepted",
        authority: "procedure-reference-only",
        source: { file: "reference/business-system-design/order-entry-business-procedure.json",
          pointer: "/steps/5/validations/" + i }
      }))
    }]
  };
  const result = projectWorld(world, revision,
    "validation:sales.order_line.quantity_positive", { businessDFD });
  assert.equal(result.views.business.authority, "noncanonical-reference");
  assert.equal(result.views.business.items[0].stepID, "BP-05");
  assert.deepEqual(result.views.business.items[0].functionRefs, []);
  assert.equal(result.views.business.items[0].explicitMeaning[0].status,
    "exact-id-matches-current-world");
  assert.equal(result.views.business.freshness,
    "unverified-no-model-revision-binding");
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
    const dfdPath = join(dir, "reference-dfd.json");
    writeFileSync(dfdPath, JSON.stringify({
      schemaVersion: "business-dfd-projection-v0",
      authority: "derived-read-only", canonicalProcedureUsed: false,
      procedureId: "order-entry.order-handling", diagnostics: [],
      steps: [{ id: "BP-01", functionRefs: [{
        id: "order_entry.record", status: "resolved", data: "sales.order"
      }] }]
    }));
    const paths = [worldPath, revPath, editPath, sourcePath, dfdPath];
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
    assert.equal(JSON.parse(inspectIndexed.stdout).sourceNavigation.byteVerification.status,
      "not-checked", "detached Index snapshot cannot certify current file bytes");
    const businessStep = spawnSync(process.execPath, [cli, "inspect",
      "-world-file", worldPath, "-revision-file", revPath,
      "-focus", "step:BP-01", "-business-dfd", dfdPath], { encoding: "utf8" });
    assert.equal(businessStep.status, 0, businessStep.stderr);
    const businessResult = JSON.parse(businessStep.stdout);
    assert.equal(businessResult.subject.scope,
      "reference-procedure:order-entry.order-handling");
    assert.equal(businessResult.candidates[0].focus, "pattern:order_entry.record");
    assert.equal(businessResult.candidates[0].status, "identity-match-only");
    assert.equal(businessResult.editable, false);
    const noBusinessDFD = spawnSync(process.execPath, [cli, "inspect",
      "-world-file", worldPath, "-revision-file", revPath,
      "-focus", "step:BP-01"], { encoding: "utf8" });
    assert.notEqual(noBusinessDFD.status, 0);
    const propose = spawnSync(process.execPath, [cli, "propose", "-world-file", worldPath,
      "-revision-file", revPath, "-edit-file", editPath], { encoding: "utf8" });
    assert.equal(propose.status, 0, propose.stderr);
    assert.equal(JSON.parse(propose.stdout).delta[0].op, "add-relation");
    const unsafeReview = spawnSync(process.execPath, [cli, "review-edit",
      "-world-file", worldPath, "-revision-file", revPath, "-edit-file", editPath],
      { encoding: "utf8" });
    assert.notEqual(unsafeReview.status, 0,
      "review-edit requires a live workspace and cannot trust unrelated snapshots");
    assert.match(unsafeReview.stderr, /INVALID_INPUT/,
      "negative test must reject detached snapshots by design, not due to CLI crash");
    assert.deepEqual(paths.map(p => readFileSync(p, "utf8")), before);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
