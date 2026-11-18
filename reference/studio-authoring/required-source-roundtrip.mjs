// The existing offline Studio editor, Shared Acceptance and product Inspector
// run against one disposable tei.authoring/v1 Source. This automated test is
// neither a production Wails invocation nor a human's semantic acceptance.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.argv.length !== 4)
  throw new Error("usage: node required-source-roundtrip.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const installRoot = resolve(process.argv[2]);
const outputDir = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(installRoot, "node_modules/playwright-core/index.mjs")).href);
const cli = join(root, "reference/studio-authoring/cli.mjs");
const temp = mkdtempSync(join(tmpdir(), "tei-studio-required-loop-"));
const workspace = join(temp, "workspace");
const sourcePath = "semantic/studio/inspection-response.tei.md";
const source = join(workspace, sourcePath);
const subjectID = "studio.inspection_response.snapshot_fingerprint";
const dataID = "studio.inspection_response";
const subject = "attribute:" + subjectID;
const documentBefore = [
  "# Studio data",
  "",
  "文章の「no」や説明を変更しない。",
  "",
  "<!-- tei: data InspectionResponse in studio -->",
  "| attribute | type | required | display |",
  "| --- | --- | --- | --- |",
  "| schemaVersion | std.text | yes | Schema |",
  "| snapshotFingerprint | std.text | no | Snapshot |",
  "",
  "別表はモデル定義ではない。",
  "",
  "| required | other |",
  "| --- | --- |",
  "| no | stays |",
  ""
].join("\n");
const documentAfter = documentBefore.replace(
  "| snapshotFingerprint | std.text | no | Snapshot |",
  "| snapshotFingerprint | std.text | yes | Snapshot |");
const digest = bytes => "sha256:" + createHash("sha256").update(bytes).digest("hex");
const exec = (bin, args) => execFileSync(bin, args, {
  cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024, timeout: 120_000
});
const go = (...args) => exec("go", ["run", ...args]);
const node = (...args) => exec(process.execPath, args);
const acceptance = (command, ...args) => JSON.parse(go(
  "./tools/reference-shared-acceptance", command, "-workspace", workspace, ...args));
const revision = () => acceptance("revision").modelRevision;
const inspection = () => {
  const report = JSON.parse(go("./cmd/tei", "inspect", "-workspace", workspace,
    "-subject-kind", "semantic-data", "-subject-id", dataID));
  assert.equal(report.schemaVersion, "tei.cli.inspect/v0");
  assert.equal(report.ok, true);
  assert.equal(report.result?.effect, "read-only");
  assert.equal(report.result?.sourceStatus, "indexed");
  assert.equal(report.result?.canonical?.id, dataID);
  const attribute = report.result.canonical.attributes.find(a => a.id === subjectID);
  assert.ok(attribute, "product Inspector did not read the exact Data attribute");
  return { attribute, report };
};
const getForm = () => node(cli, "edit-form", "-workspace", workspace,
  "-focus", subject, "-edit-mode", "required");
const writeHTML = (name, html) => {
  const dest = join(temp, name + ".html");
  writeFileSync(dest, html, { mode: 0o600 });
  return dest;
};
const failed = (bin, args, expected) => {
  const result = spawnSync(bin, args, {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024, timeout: 120_000
  });
  if (result.error) throw result.error;
  assert.notEqual(result.status, 0, "expected an explicit refusal");
  assert.match(result.stderr + "\n" + result.stdout, expected);
};
mkdirSync(join(workspace, "semantic/studio"), { recursive: true });
mkdirSync(outputDir, { recursive: true });
writeFileSync(join(workspace, "tei.workspace.yaml"), [
  "workspace: studio_reader", "formatVersion: 1",
  "sources:", "  - semantic", "languages:", "  - tei.authoring/v1", ""
].join("\n"));
writeFileSync(source, documentBefore);
const sourceFingerprintBefore = digest(readFileSync(source));
const revisionBefore = revision();
const inspectedBefore = inspection();
// Canonical omits false-valued options; absence is not an explicit false cell.
assert.equal(inspectedBefore.attribute.required === true, false);

let browser;
try {
  browser = await chromium.launch({ channel: "chrome", headless: true,
    args: ["--no-sandbox"] });
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  const errors = [], requests = [];
  page.on("pageerror", e => errors.push(e.message));
  page.on("request", req => {
    if (/^https?:/i.test(req.url())) requests.push(req.url());
  });

  const beforeHTML = getForm();
  assert.match(beforeHTML, /Content-Security-Policy/);
  await page.goto(pathToFileURL(writeHTML("before", beforeHTML)).href,
    { waitUntil: "load" });
  assert.equal(await page.locator("#subject").inputValue(), subject);
  assert.equal(await page.locator("#model-revision").inputValue(), revisionBefore);
  assert.match(await page.locator("body").innerText(), /現在の必須指定：no/);
  await page.screenshot({ path: join(outputDir, "before.png"), fullPage: true });
  await page.locator("#candidate-id").fill("required-real-chrome-source-roundtrip");
  await page.locator("#intent").fill("Data定義の必須指定を審査候補として明示する");
  await page.locator("#required-choice").selectOption("yes");
  await page.locator('button[type="submit"]').click();

  const proposal = await page.locator("#proposal-json").inputValue();
  const edit = JSON.parse(proposal);
  assert.equal(edit.schema, "tei.reference.studio-edit/v0");
  assert.equal(edit.expectedModelRevision, revisionBefore);
  assert.equal(edit.subject, subject);
  assert.equal(edit.operation, "set-attribute-required");
  assert.equal(edit.required, true);
  const editPath = join(temp, "edit.json");
  writeFileSync(editPath, proposal, { mode: 0o600 });
  const candidate = JSON.parse(node(cli, "propose", "-workspace", workspace,
    "-edit-file", editPath));
  assert.equal(candidate.schema, "tei.reference.change-candidate/v0");
  assert.equal(candidate.baseModelRevision, revisionBefore);
  assert.deepEqual(candidate.delta, [
    { op: "set-attribute-required", id: subjectID, required: true }
  ]);
  const candidatePath = join(temp, "candidate.json");
  writeFileSync(candidatePath, JSON.stringify(candidate) + "\n", { mode: 0o600 });

  const reviewed = JSON.parse(node(cli, "review-edit", "-workspace", workspace,
    "-edit-file", editPath));
  assert.equal(reviewed.schema, "tei.reference.studio-review-edit/v0");
  assert.equal(reviewed.modelRevision, revisionBefore);
  assert.equal(reviewed.sourceRevisionAfterReview, revisionBefore);
  assert.equal(reviewed.sourceWriteByThisCommand, false);
  assert.equal(reviewed.explicitAcceptance, "not-performed");
  assert.equal(reviewed.preview.acceptable, true);
  assert.equal(reviewed.candidate.id, candidate.id);
  assert.ok(/^sha256:[0-9a-f]{64}$/.test(reviewed.review.reviewFingerprint));
  assert.equal(revision(), revisionBefore);
  assert.equal(readFileSync(source, "utf8"), documentBefore);

  // The Source writer, not the browser, must require the exact Review binding.
  failed("go", ["run", "./tools/reference-shared-acceptance", "accept",
    "-workspace", workspace, "-candidate", candidatePath],
  /expected-review-fingerprint/);
  assert.equal(readFileSync(source, "utf8"), documentBefore);
  const decision = acceptance("accept", "-candidate", candidatePath,
    "-expected-review-fingerprint", reviewed.review.reviewFingerprint);
  assert.equal(decision.decision, "accepted");
  assert.equal(decision.sourceChanged, true);
  assert.equal(readFileSync(source, "utf8"), documentAfter);
  assert.notEqual(digest(readFileSync(source)), sourceFingerprintBefore);

  const revisionAfter = revision();
  assert.notEqual(revisionAfter, revisionBefore);
  const inspectedAfter = inspection();
  assert.equal(inspectedAfter.attribute.required, true);
  assert.equal(inspectedAfter.attribute.id, inspectedBefore.attribute.id);
  assert.notEqual(inspectedAfter.report.result.worldFingerprint,
    inspectedBefore.report.result.worldFingerprint);
  failed(process.execPath, [cli, "review-edit", "-workspace", workspace,
    "-edit-file", editPath], /STUDIO_EDIT_STALE_MODEL_REVISION/);
  failed("go", ["run", "./tools/reference-shared-acceptance", "accept",
    "-workspace", workspace, "-candidate", candidatePath,
    "-expected-review-fingerprint", reviewed.review.reviewFingerprint],
  /stale|revision|changed|STAGE|STALE|Review|review/i);

  const afterHTML = getForm();
  await page.goto(pathToFileURL(writeHTML("after", afterHTML)).href,
    { waitUntil: "load" });
  assert.equal(await page.locator("#model-revision").inputValue(), revisionAfter);
  assert.match(await page.locator("body").innerText(), /現在の必須指定：yes/);
  await page.screenshot({ path: join(outputDir, "after.png"), fullPage: true });
  assert.deepEqual(errors, []);
  assert.deepEqual(requests, []);

  const receipt = {
    schema: "tei.reference.studio-required-source-roundtrip/v0",
    head: process.env.GITHUB_SHA || null,
    attempt: "one-disposable-workspace",
    sourcePath,
    subject,
    before: { revision: revisionBefore, sourceFingerprint: sourceFingerprintBefore,
      required: false },
    after: { revision: revisionAfter, sourceFingerprint: digest(readFileSync(source)),
      required: true },
    realChromeDOMEvent: true,
    reviewedViaSharedAcceptance: true,
    acceptViaSharedAcceptance: true,
    productInspectorReload: true,
    reopenedEditorShowsChangedValue: true,
    staleDraftRefused: true,
    sourceOtherBytesPreserved: true,
    externalNetworkRequests: requests.length,
    authority: { humanAcceptance: "not-observed", ownerSemanticApproval: "not-established",
      productionWailsUI: "not-observed", formalBuildIntent: "not-established" }
  };
  writeFileSync(join(outputDir, "required-source-roundtrip.json"),
    JSON.stringify(receipt, null, 2) + "\n", { mode: 0o600 });
  process.stdout.write(JSON.stringify(receipt, null, 2) + "\n");
} finally {
  await browser?.close();
  rmSync(temp, { recursive: true, force: true });
}
