import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { produceBusinessDFD } from "./business-producer.mjs";

const R0 = "sha256:" + "a".repeat(64);
const dfd = { schemaVersion: "business-dfd-projection-v0",
  authority: "derived-read-only", canonicalProcedureUsed: false,
  procedureId: "local", diagnostics: [], steps: [] };
function fixture() {
  const dir = mkdtempSync(join(tmpdir(), "tei-business-producer-"));
  const paths = Object.fromEntries(["procedure", "review", "set"].map(role => {
    const file = join(dir, role + ".json");
    writeFileSync(file, JSON.stringify({ role }));
    return [role, file];
  }));
  const args = { workspace: dir, modelRevision: R0,
    repoRoot: dir, ...paths,
    readRevision: () => R0,
    runProducer: () => ({ status: 0, stdout: JSON.stringify(dfd) }) };
  return { dir, args };
}
test("Go DFD producer observation records independent input and Workspace evidence", () => {
  const f = fixture();
  try {
    const observation = produceBusinessDFD(f.args);
    assert.equal(observation.value.procedureId, "local");
    assert.equal(observation.evidence.authority, "noncanonical-reference");
    assert.equal(observation.evidence.acceptedBusinessProcedure, false);
    assert.equal(observation.evidence.sourceRevisionBinding, "not-established");
    assert.equal(observation.evidence.workspaceRevisionObserved, R0);
    assert.deepEqual(observation.evidence.sources.map(x => x.role),
      ["procedure", "review", "set"]);
    assert.ok(observation.evidence.sources.every(x =>
      /^sha256:[a-f0-9]{64}$/.test(x.fingerprint) && x.byteLength > 0));
  } finally { rmSync(f.dir, { recursive: true, force: true }); }
});
test("producer refuses a mutated source file, changed Workspace and error output", () => {
  const f = fixture();
  try {
    assert.throws(() => produceBusinessDFD({ ...f.args,
      runProducer: () => {
        writeFileSync(f.args.review, JSON.stringify({ role: "rewritten" }));
        return { status: 0, stdout: JSON.stringify(dfd) };
      } }), /BUSINESS_DFD_INPUT_CHANGED_DURING_PROJECTION/);
    let call = 0;
    assert.throws(() => produceBusinessDFD({ ...f.args,
      readRevision: () => ++call === 1 ? R0 : "sha256:" + "b".repeat(64) }),
    /BUSINESS_DFD_WORKSPACE_REVISION_CHANGED/);
    assert.throws(() => produceBusinessDFD({ ...f.args,
      readRevision: () => "sha256:" + "c".repeat(64),
      runProducer: () => { throw new Error("must not run"); } }),
    /BUSINESS_DFD_WORKSPACE_REVISION_CHANGED/);
    assert.throws(() => produceBusinessDFD({ ...f.args,
      runProducer: () => ({ status: 2, stdout: JSON.stringify(dfd) }) }),
    /BUSINESS_DFD_GO_PRODUCER_FAILED/);
    assert.throws(() => produceBusinessDFD({ ...f.args,
      runProducer: () => ({ status: 0, stdout: "{invalid" }) }),
    /BUSINESS_DFD_PRODUCER_INVALID_JSON/);
    assert.throws(() => produceBusinessDFD({ ...f.args,
      runProducer: () => ({ status: 0, stdout: JSON.stringify({
        ...dfd, authority: "accepted", canonicalProcedureUsed: true
      }) }) }), /BUSINESS_DFD_PRODUCER_AUTHORITY_INVALID/);
  } finally { rmSync(f.dir, { recursive: true, force: true }); }
});
test("producer refuses symlink and missing explicit input", () => {
  const f = fixture();
  try {
    const alias = join(f.dir, "alias.json");
    symlinkSync(f.args.procedure, alias);
    assert.throws(() => produceBusinessDFD({ ...f.args, procedure: alias }),
      /BUSINESS_DFD_INVALID_INPUT_FILE_TYPE/);
    assert.throws(() => produceBusinessDFD({ ...f.args, set: "" }),
      /BUSINESS_DFD_INVALID_PRODUCER_INPUTS/);
  } finally { rmSync(f.dir, { recursive: true, force: true }); }
});
