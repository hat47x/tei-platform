// Scratch-only demonstration: actual offline Editor -> Shared Acceptance
// Review -> explicit CLI Accept -> real Workspace reload -> Editor read-back.
// Automated Review/Accept does not establish human or business authority.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.argv.length !== 4) {
  throw new Error("usage: node real-browser-required-accept.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");
}
const repo = fileURLToPath(new URL("../../", import.meta.url));
const playwrightRoot = resolve(process.argv[2]);
const outputDir = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(playwrightRoot, "node_modules/playwright-core/index.mjs")).href);
const cli = join(repo, "reference/studio-authoring/cli.mjs");
const scratch = mkdtempSync(join(tmpdir(), "tei-studio-required-accept-"));
const workspace = join(scratch, "workspace");
const sourceFile = join(workspace, "semantic", "studio", "inspection-response.tei.md");
const editFile = join(scratch, "studio-required-draft.json");
const candidateFile = join(scratch, "reviewable-candidate.json");
const focus = "attribute:studio.inspection_response.snapshot_fingerprint";
const attributeID = "studio.inspection_response.snapshot_fingerprint";
const oldRow = "| snapshotFingerprint | std.text | no | Snapshot |";
const newRow = "| snapshotFingerprint | std.text | yes | Snapshot |";
const initial = [
  "# Studio source document",
  "",
  "The word no in this prose is not part of the Data definition.",
  "",
  "<!-- tei: data InspectionResponse in studio -->",
  "| attribute | type | required | display |",
  "| --- | --- | --- | --- |",
  "| schemaVersion | std.text | yes | Schema |",
  oldRow,
  "",
  "Unmarked table:",
  "",
  "| required | value |",
  "| --- | --- |",
  "| no | untouched |",
  ""
].join("\n");
const sha = data => "sha256:" + createHash("sha256").update(data).digest("hex");

function run(bin, args, expect = 0) {
  const p = spawnSync(bin, args, {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 120_000
  });
  if (p.error) throw p.error;
  if (p.status !== expect) {
    throw new Error(JSON.stringify({ command: [bin, ...args],
      actualExit: p.status, expectedExit: expect,
      stderr: p.stderr.slice(0, 8192), stdout: p.stdout.slice(0, 8192) }));
  }
  return p;
}
const studio = args => run(process.execPath, [cli, ...args]).stdout;
const acceptance = (command, extra = [], expected = 0) =>
  run("go", ["run", "./tools/reference-shared-acceptance",
    command, "-workspace", workspace, ...extra], expected);
const revision = () => JSON.parse(acceptance("revision").stdout).modelRevision;
const world = () => JSON.parse(run("go", ["run", "./cmd/tei",
  "world", "-workspace", workspace]).stdout);
function requiredInWorld() {
  const d = world().semantics.flatMap(s => s.data || [])
    .find(d => d.id === "studio.inspection_response");
  assert.ok(d && Array.isArray(d.attributes), "real World lost Data identity");
  const a = d.attributes.find(a => a.id === attributeID);
  assert.ok(a, "real World lost Attribute identity");
  return a.required === true;
}

mkdirSync(join(workspace, "semantic", "studio"), { recursive: true });
mkdirSync(outputDir, { recursive: true });
writeFileSync(join(workspace, "tei.workspace.yaml"),
  "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n",
  { mode: 0o600 });
writeFileSync(sourceFile, initial, { mode: 0o600 });
const initialRevision = revision();
const initialSourceDigest = sha(readFileSync(sourceFile));
assert.equal(requiredInWorld(), false);

let browser;
try {
  const originalForm = studio([
    "edit-form", "-workspace", workspace, "-focus", focus,
    "-edit-mode", "required"
  ]);
  assert.match(originalForm, /現在の必須指定：no/);
  const originalFormPath = join(scratch, "editor-before.html");
  writeFileSync(originalFormPath, originalForm, { mode: 0o600 });
  browser = await chromium.launch({
    channel: "chrome", headless: true, args: ["--no-sandbox"]
  });
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  try {
    const page = await ctx.newPage();
    const errors = [], requests = [];
    page.on("pageerror", err => errors.push(err.message));
    page.on("request", req => {
      if (/^https?:/i.test(req.url())) requests.push(req.url());
    });
    await page.goto(pathToFileURL(originalFormPath).href, { waitUntil: "load" });
    assert.equal(await page.locator("#subject").inputValue(), focus);
    assert.equal(await page.locator("#model-revision").inputValue(), initialRevision);
    assert.equal(await page.locator("#subject").getAttribute("readonly"), "");
    assert.equal(await page.locator("#model-revision").getAttribute("readonly"), "");
    await page.locator("#candidate-id").fill("studio-required-browser-review");
    await page.locator("#intent").fill("既存Dataの必須指定を変更する");
    await page.locator("#required-choice").selectOption("yes");
    await page.locator('button[type="submit"]').click();
    const draft = await page.locator("#proposal-json").inputValue();
    const edit = JSON.parse(draft);
    assert.equal(edit.operation, "set-attribute-required");
    assert.equal(edit.required, true);
    assert.equal(edit.expectedModelRevision, initialRevision);
    assert.equal(edit.subject, focus);
    assert.deepEqual(errors, [], "CSP-pinned inline script failed in real Chrome");
    assert.deepEqual(requests, [], "offline Editor made a network request");
    assert.match(await page.locator("#notice").innerText(), /未審査/);
    writeFileSync(editFile, draft, { mode: 0o600 });
    await page.screenshot({
      path: join(outputDir, "required-editor-before.png"), fullPage: true
    });
  } finally {
    await ctx.close();
  }

  // A persisted draft must be re-read as a candidate, never treated as a
  // direct Source patch or silently approved. It lives OUTSIDE Workspace.
  const candidate = JSON.parse(studio([
    "propose", "-workspace", workspace, "-edit-file", editFile
  ]));
  assert.equal(candidate.schema, "tei.reference.change-candidate/v0");
  assert.equal(candidate.baseModelRevision, initialRevision);
  assert.equal(candidate.origin.kind, "studio");
  assert.deepEqual(candidate.delta, [{
    op: "set-attribute-required", id: attributeID, required: true
  }]);
  writeFileSync(candidateFile, JSON.stringify(candidate, null, 2) + "\n",
    { mode: 0o600 });
  const review = JSON.parse(studio([
    "review-edit", "-workspace", workspace, "-edit-file", editFile
  ]));
  assert.equal(review.schema, "tei.reference.studio-review-edit/v0");
  assert.equal(review.sourceWriteByThisCommand, false);
  assert.equal(review.explicitAcceptance, "not-performed");
  assert.equal(review.modelRevision, initialRevision);
  assert.equal(review.sourceRevisionAfterReview, initialRevision);
  assert.equal(review.preview.acceptable, true);
  assert.equal(review.review.previewAcceptable, true);
  assert.match(review.review.reviewFingerprint, /^sha256:[0-9a-f]{64}$/);
  assert.equal(sha(readFileSync(sourceFile)), initialSourceDigest);
  assert.equal(revision(), initialRevision);

  const acceptArgs = ["-candidate", candidateFile];
  // go run maps a child process exit(2) to exit(1), preserving stderr.
  const withoutReview = acceptance("accept", acceptArgs, 1);
  assert.match(withoutReview.stderr, /expected-review-fingerprint/);
  assert.equal(sha(readFileSync(sourceFile)), initialSourceDigest);

  // This is an explicit TEST-DRIVER Accept, not a human Review or policy
  // authority. Shared Acceptance alone decides whether to write Source.
  const accepted = JSON.parse(acceptance("accept", [
    ...acceptArgs, "-expected-review-fingerprint", review.review.reviewFingerprint
  ]).stdout);
  assert.equal(accepted.decision, "accepted");
  assert.equal(accepted.sourceChanged, true);
  const expectedSource = initial.replace(oldRow, newRow);
  assert.equal(readFileSync(sourceFile, "utf8"), expectedSource,
    "Accept changed prose, an unmarked table, or unrelated Markdown bytes");
  const finalRevision = revision();
  assert.notEqual(finalRevision, initialRevision);
  assert.equal(requiredInWorld(), true,
    "the actual Go Workspace loader did not reread the accepted Data");

  // The same Editor (no alternate projection) must now show the changed
  // value in a fresh browser document. It is not an embedded Wails test.
  const updatedHTML = studio([
    "edit-form", "-workspace", workspace, "-focus", focus,
    "-edit-mode", "required"
  ]);
  assert.match(updatedHTML, /現在の必須指定：yes/);
  const afterPath = join(scratch, "editor-after.html");
  writeFileSync(afterPath, updatedHTML, { mode: 0o600 });
  const reloadCtx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  try {
    const page = await reloadCtx.newPage();
    await page.goto(pathToFileURL(afterPath).href, { waitUntil: "load" });
    assert.equal(await page.locator("#model-revision").inputValue(), finalRevision);
    assert.match(await page.locator("main").innerText(), /現在の必須指定：yes/);
    await page.screenshot({
      path: join(outputDir, "required-editor-after.png"), fullPage: true
    });
  } finally {
    await reloadCtx.close();
  }

  const stale = run(process.execPath, [
    cli, "review-edit", "-workspace", workspace, "-edit-file", editFile
  ], 1);
  assert.match(stale.stderr, /STUDIO_EDIT_STALE_MODEL_REVISION/);
  const repeatedAccept = acceptance("accept", [
    ...acceptArgs, "-expected-review-fingerprint", review.review.reviewFingerprint
  ], 1);
  assert.ok(repeatedAccept.stdout || repeatedAccept.stderr);
  assert.equal(readFileSync(sourceFile, "utf8"), expectedSource);
  const receipt = {
    schema: "tei.reference.studio-required-browser-accept-roundtrip/v0",
    head: process.env.GITHUB_SHA || null,
    sourceFormat: "tei.authoring/v1",
    origin: "studio-offline-Chrome",
    sourcePath: "semantic/studio/inspection-response.tei.md",
    editorDraftSavedOutsideWorkspace: true,
    browserSelection: "required=yes",
    sharedReviewAcceptable: true,
    reviewFingerprintBound: true,
    acceptByTestDriver: true,
    sourceChangedOnlyAtAccept: true,
    exactCellOnlyChanged: true,
    goWorkspaceReread: true,
    chromeEditorReread: true,
    oldDraftStaleRefused: true,
    initialRevision,
    finalRevision,
    initialSourceDigest,
    finalSourceDigest: sha(readFileSync(sourceFile)),
    authority: {
      humanAcceptance: "not-observed",
      productionStudioUI: "not-observed",
      businessMeaningAcceptance: "not-established",
      formalBuildIntent: "not-established",
      WindowsWebView2: "not-observed"
    }
  };
  writeFileSync(join(outputDir, "required-roundtrip-receipt.json"),
    JSON.stringify(receipt, null, 2) + "\n", { mode: 0o600 });
  process.stdout.write(JSON.stringify(receipt, null, 2) + "\n");
} finally {
  if (browser) await browser.close();
  rmSync(scratch, { recursive: true, force: true });
}
