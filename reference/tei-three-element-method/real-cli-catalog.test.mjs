// T18-local: real Go CLI model catalog/validate -> independent Node presentation.
// The Source is an ephemeral PUBLIC synthetic fixture. Roles/disclosure are NOT
// produced by Go CLI or a real Policy. No Workbench/Owner/Effect qualification.
import test from "node:test";
import assert from "node:assert/strict";
import {spawnSync} from "node:child_process";
import {createHash} from "node:crypto";
import {mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync} from "node:fs";
import {tmpdir} from "node:os";
import {join} from "node:path";
import {projectPublicTeaSlice} from "./public-slice.mjs";
import {bindQualifiedCatalogValidation} from "./catalog-validation.mjs";

const cli = process.env.TEI_CLI_BINARY;
const expectedHead = process.env.TEI_EXPECTED_HEAD;
if (!cli || !/^[a-f0-9]{40}$/.test(expectedHead ?? "")) {
  throw new Error("real_cli_test_requires_exact_binary_and_head");
}
const hash = value => createHash("sha256").update(value).digest("hex");
const call = (args, expectedExit) => {
  const run = spawnSync(cli, args, {encoding: "utf8", maxBuffer: 5 * 1024 * 1024,
    timeout: 120000});
  assert.equal(run.error, undefined, `CLI process error: ${args[0]}`);
  assert.equal(run.status, expectedExit,
    `unexpected CLI exit for ${args[0]}: ${run.status}, stdout:${run.stdout.slice(0, 150)}`);
  const response = JSON.parse(run.stdout);
  assert.equal(typeof response, "object");
  return {response, stdoutHash: hash(run.stdout)};
};

const manifest = "workspace: scn\nformatVersion: 1\nsources:\n  - semantic\n";
const model = "namespace scn\n\ndata RuleCandidate {\n    attribute sourceVersion type std.text\n}\n";
const badType = "namespace scn\n\ndata X {\n    attribute a type scn.nothere\n}\n";
function withFixture(bad, run) {
  const root = mkdtempSync(join(tmpdir(), "tei-tea-cli-"));
  const folder = join(root, "semantic");
  mkdirSync(folder);
  const paths = [join(root, "tei.workspace.yaml"), join(folder, "model.tei")];
  writeFileSync(paths[0], manifest);
  writeFileSync(paths[1], model);
  if (bad) {
    paths.push(join(folder, "extra.tei"));
    writeFileSync(paths[2], badType);
  }
  const before = paths.map(p => hash(readFileSync(p)));
  try { return run(root); }
  finally {
    assert.deepStrictEqual(paths.map(p => hash(readFileSync(p))), before,
      "real CLI/Node read-only Consumer must not mutate fixture files");
    rmSync(root, {recursive: true, force: true});
  }
}

const version = call(["version", "-json"], 0).response;
assert.equal(version.vcsRevision, expectedHead, "binary not built from PR exact head");
assert.equal(version.vcsModified ?? false, false, "dirty binary revision");

function observe(root, expectValid) {
  const catalogCall = call(["model", "catalog", "-workspace", root], 0);
  const e = catalogCall.response;
  assert.equal(e.schemaVersion, "tei.cli.model/v0");
  assert.equal(e.ok, true, "Catalog read should succeed even for invalid model");
  assert.equal(e.result.effect, "read-only");
  assert.equal(e.result.schemaVersion, "tei.operations.model-catalog/v0");
  assert.ok(e.result.modelRevision && e.result.worldFingerprint &&
    e.result.sourceIndexFingerprint, "Source/World observation absent");
  const actual = e.result.validation;
  assert.equal(actual.ok, expectValid, "real Catalog Validation verdict mismatch");
  assert.ok(Number.isSafeInteger(actual.errorCount) && actual.errorCount >= 0);
  assert.ok(Number.isSafeInteger(actual.warningCount) && actual.warningCount >= 0);

  // Independent CLI command: agreement is NOT an atomic multi-command read.
  const validateCall = call(["validate", "-workspace", root, "-json"], expectValid ? 0 : 1);
  const v = validateCall.response;
  assert.equal(v.schemaVersion, "tei.cli.validate/v0");
  assert.equal(v.ok, expectValid);
  assert.equal(v.worldFingerprint, e.result.worldFingerprint);
  assert.equal(v.sourceIndexFingerprint, e.result.sourceIndexFingerprint);
  assert.deepStrictEqual(v.workspace, e.result.workspace);
  const entries = e.result.entries.filter(x =>
    x.subject?.kind === "semantic-data" &&
    (expectValid ? x.subject?.id?.startsWith("scn.") : x.subject?.id === "scn.x") &&
    x.sourceStatus === "indexed");
  assert.equal(entries.length, 1, "expected one unambiguous actual CLI Subject");
  assert.notEqual(entries[0].ambiguous, true);
  assert.notEqual(entries[0].prelude, true);
  assert.equal(entries[0].sourceStatus, "indexed");
  return {
    catalog: e.result, subject: entries[0].subject,
    rawHashes: {catalog: catalogCall.stdoutHash, validate: validateCall.stdoutHash},
  };
}

function projectFromObservedCatalog(actual, witness) {
  const {catalog, subject} = actual;
  // Hashes below are only equality witnesses derived from the read response.
  // Role data is synthetic. The CLI DOES NOT attest a Function/Business Role.
  const snapshot = "observed:" + hash(JSON.stringify([
    catalog.modelRevision, catalog.worldFingerprint, catalog.sourceIndexFingerprint]));
  const publicInput = {scope: "synthetic-demo", workspaceObservation: witness,
    subjects: [{scope: "synthetic-demo", kind: subject.kind, id: subject.id,
      roleStates: {data: "accepted", function: "unknown", business: "unknown"}}],
    relations: []};
  const projection = projectPublicTeaSlice(publicInput, snapshot);
  const qualifiedInput = {scope: publicInput.scope, snapshot, workspaceObservation: witness,
    catalogReadSucceeded: true,
    // Safe here ONLY because the entire temporary fixture is publicly known.
    validation: {...catalog.validation}};
  const attached = bindQualifiedCatalogValidation(projection, qualifiedInput);
  assert.deepStrictEqual(attached.validation, catalog.validation);
  assert.equal(attached.catalogReadSucceeded, true);
  assert.equal(attached.allSourceCompleteness, "not-proven");
  assert.equal(projection.views.data.length, 1);
  assert.equal(projection.views.function.length, 0);
  assert.equal(projection.views.business.length, 0);
  assert.equal(projection.publicDiagnostics.length, 2);
  assert.equal(Object.hasOwn(attached, "editEligible"), false);
  assert.equal(Object.hasOwn(attached, "acceptedMeaning"), false);
  return {projection, attached, qualifiedInput};
}

test("T18 real Go CLI -> Node TEA Read/Validation projection: valid and unsound", () => {
  const valid = withFixture(false, root => observe(root, true));
  const invalid = withFixture(true, root => observe(root, false));
  assert.equal(valid.catalog.validation.errorCount, 0);
  assert.ok(invalid.catalog.validation.errorCount > 0);
  const a = projectFromObservedCatalog(valid, "test-observed-valid");
  const b = projectFromObservedCatalog(invalid, "test-observed-unsound");
  assert.equal(a.attached.validation.ok, true);
  assert.equal(b.attached.validation.ok, false);
  // Same test-owned presentation must never reinterpret bad validation as read failure.
  assert.equal(b.attached.catalogReadSucceeded, true);
  assert.notEqual(a.attached.snapshot, b.attached.snapshot);
  const mismatched = {...b.qualifiedInput, snapshot: a.attached.snapshot};
  assert.throws(() => bindQualifiedCatalogValidation(b.projection, mismatched),
    /unqualified_catalog_validation/);
  const crossWorkspace = {...b.qualifiedInput, workspaceObservation: "test-observed-valid"};
  assert.throws(() => bindQualifiedCatalogValidation(b.projection, crossWorkspace),
    /unqualified_catalog_validation/);
  console.log("T18_REAL_CLI_NODE_PASS", JSON.stringify({
    cliHead: expectedHead, valid: a.attached.validation.ok,
    unsound: b.attached.validation.ok, catalogRead: true, rolesFromCLI: false,
    sameAtomicRead: false, currentDisclosureProven: false,
  }));
});
