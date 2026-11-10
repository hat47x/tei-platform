import test from "node:test";
import assert from "node:assert/strict";
import {projectPublicTeaSlice} from "./public-slice.mjs";
import {bindQualifiedCatalogValidation} from "./catalog-validation.mjs";

const projection = () => projectPublicTeaSlice({
  scope: "sales", workspaceObservation: "load:A",
  subjects: [{scope: "sales", kind: "semantic-data", id: "order",
    roleStates: {data: "accepted", function: "accepted", business: "accepted"}}],
  relations: [],
}, "source:R1|role:P1|policy:G1");
const observation = () => ({
  scope: "sales", workspaceObservation: "load:A",
  snapshot: "source:R1|role:P1|policy:G1",
  catalogReadSucceeded: true,
  validation: {ok: false, errorCount: 1, warningCount: 0},
});

test("T18: Catalog read succeeds while Validation fails and three views remain readable", () => {
  const p = projection();
  const summary = bindQualifiedCatalogValidation(p, observation());
  assert.equal(summary.catalogReadSucceeded, true);
  assert.equal(summary.validation.ok, false);
  assert.equal(summary.validation.errorCount, 1);
  assert.equal(p.views.data.length, 1);
  assert.equal(p.views.function.length, 1);
  assert.equal(p.views.business.length, 1);
  assert.equal(Object.hasOwn(summary, "editEligible"), false);
});

test("T18: warning is not coerced into validation failure", () => {
  const input = observation();
  input.validation = {ok: true, errorCount: 0, warningCount: 3};
  const output = bindQualifiedCatalogValidation(projection(), input);
  assert.deepStrictEqual(output.validation, input.validation);
  assert.equal(output.allSourceCompleteness, "not-proven");
});

test("T18: failed or unobserved catalog read does not manufacture a validation verdict", () => {
  const missing = observation();
  delete missing.catalogReadSucceeded;
  assert.throws(() => bindQualifiedCatalogValidation(projection(), missing),
    /unqualified_catalog_validation/);
  const failure = observation();
  failure.catalogReadSucceeded = false;
  assert.throws(() => bindQualifiedCatalogValidation(projection(), failure),
    /unqualified_catalog_validation/);
  const noVerdict = observation();
  delete noVerdict.validation;
  assert.throws(() => bindQualifiedCatalogValidation(projection(), noVerdict),
    /unqualified_catalog_validation/);
});

test("T18: same view ID does not permit another Workspace or Snapshot verdict", () => {
  const a = observation();
  a.workspaceObservation = "load:B";
  assert.throws(() => bindQualifiedCatalogValidation(projection(), a),
    /unqualified_catalog_validation/);
  const b = observation();
  b.snapshot = "source:R0|role:P1|policy:G1";
  assert.throws(() => bindQualifiedCatalogValidation(projection(), b),
    /unqualified_catalog_validation/);
  const c = observation();
  c.scope = "billing";
  assert.throws(() => bindQualifiedCatalogValidation(projection(), c),
    /unqualified_catalog_validation/);
});

test("T18: malformed counts and booleans are not reinterpreted or defaulted", () => {
  for (const invalid of [
    {ok: "false", errorCount: 1, warningCount: 0},
    {ok: false, errorCount: -1, warningCount: 0},
    {ok: false, errorCount: 1.5, warningCount: 0},
    {ok: false, errorCount: 1, warningCount: Number.MAX_SAFE_INTEGER + 1},
  ]) {
    const input = observation();
    input.validation = invalid;
    assert.throws(() => bindQualifiedCatalogValidation(projection(), input),
      /unqualified_catalog_validation/);
  }
});

test("T18: unqualified fields and getters fail closed without emitting diagnostics", () => {
  const extra = observation();
  extra.validation.privateSourcePath = "/private/model.tei";
  assert.throws(() => bindQualifiedCatalogValidation(projection(), extra),
    err => err.message === "unqualified_catalog_validation" &&
      !err.message.includes("private"));
  const getter = observation();
  let calls = 0;
  Object.defineProperty(getter.validation, "errorCount", {enumerable: true,
    get() {calls++; return 10}});
  assert.throws(() => bindQualifiedCatalogValidation(projection(), getter),
    /unqualified_catalog_validation/);
  assert.equal(calls, 0);
});
