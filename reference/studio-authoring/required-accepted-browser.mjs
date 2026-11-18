// A real Chrome edit -> existing Shared Acceptance Review/Accept -> real TEI
// Inspect roundtrip on ONE disposable tei.authoring/v1 Workspace.
// Automated Accept proves the mechanical boundary, never human acceptance,
// production Source authorization, or a rebuilt Wails Desktop.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.argv.length !== 4)
  throw new Error("usage: node required-accepted-browser.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");
const repo = fileURLToPath(new URL("../../", import.meta.url));
const install = resolve(process.argv[2]), out = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(install, "node_modules/playwright-core/index.mjs")).href);
const cli = join(repo, "reference/studio-authoring/cli.mjs");
const temporary = mkdtempSync(join(tmpdir(), "tei-studio-required-accepted-"));
const workspace = join(temporary, "workspace");
const sourceRelative = "semantic/studio/inspection-response.tei.md";
const sourcePath = join(workspace, sourceRelative);
const dataID = "studio.inspection_response";
const attributeID = "studio.inspection_response.snapshot_fingerprint";
const original = [
  "# Studio Data definition",
  "",
  "The word no in this explanation must remain unchanged.",
  "",
  "<!-- tei: data InspectionResponse in studio -->",
  "| attribute | type | required | display |",
  "| --- | --- | --- | --- |",
  "| schemaVersion | std.text | yes | Schema |",
  "| snapshotFingerprint | std.text | no | Snapshot |",
  "",
  "An ordinary, unmarked table remains ordinary text:",
  "",
  "| required | other |",
  "| --- | --- |",
  "| no | preserved |",
  ""
].join("\n");
const beforeLine = "| snapshotFingerprint | std.text | no | Snapshot |";
const afterLine = "| snapshotFingerprint | std.text | yes | Snapshot |";
const digest = bytes => "sha256:" + createHash("sha256").update(bytes).digest("hex");
const invoke = (bin, args) => execFileSync(bin, args, {
  cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024, timeout: 120_000
});
const acceptanceArgs = (...rest) =>
  ["run", "./tools/reference-shared-acceptance", ...rest];
const revision = () => JSON.parse(invoke("go",
  acceptanceArgs("revision", "-workspace", workspace))).modelRevision;
const inspect = () => {
  const result = JSON.parse(invoke("go", [
    "run", "./cmd/tei", "inspect", "-workspace", workspace,
    "-subject-kind", "semantic-data", "-subject-id", dataID
  ]));
  assert.equal(result.schemaVersion, "tei.cli.inspect/v0");
  assert.equal(result.ok, true);
  assert.equal(result.result?.effect, "read-only");
  assert.equal(result.result?.sourceStatus, "indexed");
  const refs = result.result?.sources?.references || [];
  const artifacts = result.result?.sources?.artifacts || [];
  assert.equal(refs.length, 1, "Data must have one exact Source Index reference");
  assert.equal(refs[0].artifactPath, sourceRelative);
  assert.equal(refs[0].subject?.kind, "semantic-data");
  assert.equal(refs[0].subject?.id, dataID);
  assert.equal(artifacts.length, 1);
  assert.equal(artifacts[0].path, sourceRelative);
  const canonical = result.result?.canonical;
  assert.equal(canonical?.id, dataID);
  assert.equal(canonical.attributes?.length, 2);
  const target = canonical.attributes.find(attr => attr.id === attributeID);
  assert.ok(target, "real Inspector did not return the authored Attribute");
  return { required: target.required === true, result };
};
const refuse = (args, label) => {
  const result = spawnSync("go", acceptanceArgs(...args), {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024, timeout: 120_000
  });
  assert.equal(result.error, undefined, label + ": could not invoke Shared Acceptance");
  assert.notEqual(result.status, 0, label + ": must fail closed");
  return result;
};
let browser;
mkdirSync(out, { recursive: true });
try {
  mkdirSync(join(workspace, "semantic/studio"), { recursive: true });
  writeFileSync(join(workspace, "tei.workspace.yaml"),
    "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n");
  writeFileSync(sourcePath, original);
  const initial = revision();
  const old = inspect();
  assert.equal(old.required, false);

  // Obtain both the form and its revision from this same actual Workspace.
  const html = invoke(process.execPath, [
    cli, "edit-form", "-workspace", workspace,
    "-focus", "attribute:" + attributeID, "-edit-mode", "required"
  ]);
  assert.match(html, /Content-Security-Policy/);
  const htmlPath = join(temporary, "edit.html");
  writeFileSync(htmlPath, html, { mode: 0o600 });
  browser = await chromium.launch({channel: "chrome", headless: true, args: ["--no-sandbox"]});
  const context = await browser.newContext({viewport: {width: 1280, height: 900}});
  try {
    const page = await context.newPage();
    const pageErrors = [], networkRequests = [];
    page.on("pageerror", error => pageErrors.push(error.message));
    page.on("request", req => {
      if (/^https?:/i.test(req.url())) networkRequests.push(req.url());
    });
    await page.goto(pathToFileURL(htmlPath).href, {waitUntil: "load"});
    assert.equal(await page.locator("#model-revision").inputValue(), initial);
    assert.equal(await page.locator("#subject").inputValue(), "attribute:" + attributeID);
    assert.match(await page.locator("body").innerText(), /現在の必須指定：no/);
    await page.locator("#candidate-id").fill("studio-required-real-chrome-roundtrip");
    await page.locator("#intent").fill("既存Data表の必須指定を変更し、同じSourceを再読込する");
    await page.locator("#required-choice").selectOption("yes");
    await page.locator('button[type="submit"]').click();
    const edit = JSON.parse(await page.locator("#proposal-json").inputValue());
    assert.equal(edit.expectedModelRevision, initial);
    assert.equal(edit.subject, "attribute:" + attributeID);
    assert.equal(edit.required, true);
    assert.equal(edit.operation, "set-attribute-required");
    assert.match(await page.locator("#notice").innerText(), /未審査/);
    assert.deepEqual(pageErrors, []);
    assert.deepEqual(networkRequests, []);
    await page.screenshot({path: join(out, "required-accepted-scratch.png"), fullPage: true});

    const editPath = join(temporary, "edit.json");
    const candidatePath = join(temporary, "candidate.json");
    writeFileSync(editPath, JSON.stringify(edit) + "\n", {mode: 0o600});
    const candidate = JSON.parse(invoke(process.execPath, [
      cli, "propose", "-workspace", workspace, "-edit-file", editPath
    ]));
    assert.equal(candidate.baseModelRevision, initial);
    assert.deepEqual(candidate.delta, [{
      op: "set-attribute-required", id: attributeID, required: true
    }]);
    writeFileSync(candidatePath, JSON.stringify(candidate) + "\n", {mode: 0o600});
    const reviewed = JSON.parse(invoke(process.execPath, [
      cli, "review-edit", "-workspace", workspace, "-edit-file", editPath
    ]));
    assert.equal(reviewed.schema, "tei.reference.studio-review-edit/v0");
    assert.equal(reviewed.sourceWriteByThisCommand, false);
    assert.equal(reviewed.explicitAcceptance, "not-performed");
    assert.equal(reviewed.modelRevision, initial);
    assert.equal(reviewed.sourceRevisionAfterReview, initial);
    assert.equal(reviewed.preview?.acceptable, true);
    const edits = reviewed.preview.sourceEdits || [];
    assert.equal(edits.length, 1);
    assert.equal(edits[0].path, sourceRelative);
    assert.equal(edits[0].insertedText, "yes");
    assert.equal(edits[0].removedText, "no");
    const fingerprint = reviewed.review?.reviewFingerprint;
    assert.match(fingerprint, /^sha256:[0-9a-f]{64}$/);
    assert.equal(readFileSync(sourcePath, "utf8"), original);
    assert.equal(revision(), initial);

    // An Editor or Preview cannot grant a Source write. A separate explicit
    // Review fingerprint is required by the existing acceptance CLI.
    refuse(["accept", "-workspace", workspace, "-candidate", candidatePath,
      "-expected-review-fingerprint", ""], "unreviewed Source write");
    assert.equal(readFileSync(sourcePath, "utf8"), original);
    const accepted = JSON.parse(invoke("go", acceptanceArgs(
      "accept", "-workspace", workspace, "-candidate", candidatePath,
      "-expected-review-fingerprint", fingerprint
    )));
    assert.equal(accepted.decision, "accepted");
    assert.equal(accepted.sourceChanged, true);
    const saved = readFileSync(sourcePath, "utf8");
    assert.equal(saved, original.replace(beforeLine, afterLine));
    assert.equal(saved.split(afterLine).length, 2);
    const current = revision();
    assert.notEqual(current, initial);
    const reloaded = inspect();
    assert.equal(reloaded.required, true);

    // A stale click/old candidate cannot silently replay on the new Source.
    const stale = refuse(["preview", "-workspace", workspace,
      "-candidate", candidatePath], "stale Source revision");
    assert.match(stale.stdout, /STALE_BASE_MODEL_REVISION/);
    assert.equal(readFileSync(sourcePath, "utf8"), saved);
    assert.deepEqual(pageErrors, []);
    assert.deepEqual(networkRequests, []);

    const receipt = {
      schema: "tei.reference.studio-required-source-roundtrip/v0",
      head: process.env.GITHUB_SHA || null,
      browser: "real-chrome-headless",
      initialModelRevision: initial, acceptedModelRevision: current,
      sourcePath: sourceRelative,
      sourceFingerprintBefore: digest(original),
      sourceFingerprintAfter: digest(saved),
      sourceOnlyOneCellChanged: true,
      realChromeDOMEvent: true, sourceIndexOwned: true,
      reviewFingerprintBound: true, previewSourceUnchanged: true,
      unreviewedWriteRefused: true, staleCandidateRefused: true,
      inspectBeforeRequired: old.required, inspectAfterRequired: reloaded.required,
      sourceChangedByExplicitAcceptance: true,
      authority: {
        humanAcceptance: "not-observed", ownerMeaningAcceptance: "not-observed",
        productionHostWrite: "not-observed", generatedDesktopReentry: "not-observed"
      }
    };
    writeFileSync(join(out, "required-accepted-receipt.json"),
      JSON.stringify(receipt, null, 2) + "\n", {mode: 0o600});
    process.stdout.write(JSON.stringify(receipt, null, 2) + "\n");
  } finally {
    await context.close();
  }
} finally {
  if (browser) await browser.close();
  rmSync(temporary, {recursive: true, force: true});
}
