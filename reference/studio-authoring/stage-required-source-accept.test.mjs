import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, mkdirSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
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
    assert.equal(proof.realBrowserEditing, "not-observed-in-this-attempt");
    assert.equal(proof.browserScreenshotDigest, null);
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

    // If real-browser mode was requested, missing Playwright MUST fail before
    // Preview/Review/Accept rather than quietly reverting to a simulated DOM.
    const browserAfter = join(temp, "browser-required-after");
    const missingReceipt = join(temp, "missing-browser-receipt.json");
    const missingBrowser = spawnSync(process.execPath,
      [driver, before, browserAfter, missingReceipt], {
        cwd: repo, encoding: "utf8", timeout: 240_000,
        env: {
          ...process.env, TEI_STAGE_PLAYWRIGHT_ROOT: join(temp, "not-installed"),
          GOWORK: "off", GOTOOLCHAIN: "local"
        },
      });
    assert.notEqual(missingBrowser.status, 0, "browser-required mode fell back to VM");
    assert.match(missingBrowser.stderr, /PLAYWRIGHT_CORE_NOT_INSTALLED/);
    assert.equal(readFileSync(join(browserAfter, source), "utf8"), baseline);
    assert.equal(existsSync(missingReceipt), false);
  } finally {
    rmSync(temp, { recursive: true, force: true });
  }
});

test("scratch-only Accept rejects symlinked Source, output and receipt paths before writing", {
  skip: process.platform === "win32"
}, () => {
  const temp = mkdtempSync(join(tmpdir(), "tei-studio-stage-symlink-"));
  try {
    const realBefore = join(temp, "before");
    const realOutput = join(temp, "other-output");
    mkdirSync(realBefore);
    mkdirSync(realOutput);
    const beforeAlias = join(temp, "before-link");
    const outputAlias = join(temp, "output-link");
    const danglingOutput = join(temp, "dangling-output");
    const receiptAlias = join(temp, "receipt-link.json");
    symlinkSync(realBefore, beforeAlias, "dir");
    symlinkSync(realOutput, outputAlias, "dir");
    symlinkSync(join(temp, "absent-output"), danglingOutput, "dir");
    symlinkSync(join(temp, "absent-receipt.json"), receiptAlias, "file");

    const attempts = [
      [beforeAlias, join(temp, "after-1"), join(temp, "receipt-1.json")],
      [realBefore, join(outputAlias, "after-2"), join(temp, "receipt-2.json")],
      [realBefore, danglingOutput, join(temp, "receipt-3.json")],
      [realBefore, join(temp, "after-4"), receiptAlias],
    ];
    for (const [before, after, receipt] of attempts) {
      const result = spawnSync(process.execPath, [driver, before, after, receipt], {
        cwd: repo, encoding: "utf8", timeout: 30_000,
      });
      assert.notEqual(result.status, 0, "symlinked scratch path was accepted");
      assert.match(result.stderr, /STAGE_REQUIRED_ACCEPT_SYMLINK_OR_PRODUCT_PATH/);
      assert.equal(existsSync(after), false, "a rejected input created an output Workspace");
      assert.equal(existsSync(receipt), false, "a rejected input created a receipt");
    }
    assert.equal(existsSync(join(realOutput, "after-2")), false);
  } finally {
    rmSync(temp, { recursive: true, force: true });
  }
});
