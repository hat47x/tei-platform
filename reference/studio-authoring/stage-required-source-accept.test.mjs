import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const driver = join(repo, "reference/studio-authoring/stage-required-source-accept.mjs");
const source = "semantic/studio/inspection-response.tei.md";
const baseline = [
  "# Studio response", "",
  "<!-- tei: data InspectionResponse in studio -->",
  "| attribute | type | required | display |",
  "| --- | --- | --- | --- |",
  "| schemaVersion | std.text | yes | Schema |",
  "| snapshotFingerprint | std.text | no | Snapshot |", ""
].join("\n");

test("offline Studio candidate passes one real Review/Accept and reloads exact Source", { timeout: 240_000 }, () => {
  const temp = mkdtempSync(join(tmpdir(), "tei-studio-stage-source-"));
  try {
    const before = join(temp, "before"), after = join(temp, "after");
    const receipt = join(temp, "receipt.json");
    mkdirSync(join(before, "semantic/studio"), { recursive: true });
    writeFileSync(join(before, "tei.workspace.yaml"),
      "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n");
    writeFileSync(join(before, source), baseline);
    execFileSync(process.execPath, [driver, before, after, receipt], {
      cwd: repo, encoding: "utf8", timeout: 240_000,
      env: { ...process.env, GOWORK: "off", GOTOOLCHAIN: "local" }
    });
    const proof = JSON.parse(readFileSync(receipt, "utf8"));
    assert.equal(proof.schema, "tei.reference.studio-stage-source-accept/v0");
    assert.equal(proof.editorTransport, "exact-offline-form-script-vm");
    assert.equal(proof.ownerMeaningAcceptance, "not-established");
    assert.equal(proof.formalBuildIntent, "not-established");
    assert.equal(proof.sourceChangedOneCell, true);
    assert.equal(proof.unreviewedAcceptRefused, true);
    assert.notEqual(proof.expectedModelRevision, proof.savedModelRevision);
    assert.notEqual(proof.beforeWorkspaceDigest, proof.afterWorkspaceDigest);
    assert.equal(readFileSync(join(before, source), "utf8"), baseline);
    const revised = baseline.replace(
      "| snapshotFingerprint | std.text | no | Snapshot |",
      "| snapshotFingerprint | std.text | yes | Snapshot |");
    assert.equal(readFileSync(join(after, source), "utf8"), revised);

    // A repeated attempt must not clobber an existing accepted Workspace.
    const repeated = spawnSync(process.execPath, [driver, before, after, join(temp, "second.json")], {
      cwd: repo, encoding: "utf8", timeout: 30_000
    });
    assert.notEqual(repeated.status, 0);
    assert.equal(readFileSync(join(after, source), "utf8"), revised);
  } finally {
    rmSync(temp, { recursive: true, force: true });
  }
});
