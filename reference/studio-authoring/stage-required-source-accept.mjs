#!/usr/bin/env node
// Scratch-only stage input. Drive the existing offline Editor script and
// Shared Acceptance; never treat this automated Accept as Owner authority.
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cpSync, existsSync, lstatSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";

if (process.argv.length !== 5)
  throw new Error("usage: node stage-required-source-accept.mjs BEFORE_WORKSPACE AFTER_WORKSPACE RECEIPT_JSON");

const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const [before, after, receiptPath] = process.argv.slice(2).map(p => resolve(p));
// Never allow a test-only automated Accept to be pointed at a product
// checkout or baseline. Both workspaces and the receipt live under one
// designated scratch root (the CI runner temp or the process temp).
const scratchRoots = [process.env.RUNNER_TEMP, tmpdir()].filter(Boolean).map(root => resolve(root));
function inside(root, value) {
  const rel = relative(root, value);
  return !!rel && rel !== ".." && !rel.startsWith(".." + sep) && !isAbsolute(rel);
}
if (!scratchRoots.some(root => [before, after, receiptPath].every(path => inside(root, path))))
  throw new Error("STAGE_REQUIRED_ACCEPT_REQUIRES_ONE_SCRATCH_ROOT");
const sourceRel = "semantic/studio/inspection-response.tei.md";
const attributeID = "studio.inspection_response.snapshot_fingerprint";
const dataID = "studio.inspection_response";
const sourceAt = dir => join(dir, ...sourceRel.split("/"));
const sha = bytes => "sha256:" + createHash("sha256").update(bytes).digest("hex");
const goEnv = { ...process.env, GOWORK: "off", GOTOOLCHAIN: "local", GOFLAGS: "" };

function command(bin, args) {
  return execFileSync(bin, args, {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 240_000, env: goEnv,
  });
}
const acceptance = (cmd, ...args) => command("go", [
  "run", "./tools/reference-shared-acceptance", cmd, "-workspace", after, ...args
]);
const revision = () => JSON.parse(acceptance("revision")).modelRevision;
const world = () => JSON.parse(command("go", ["run", "./cmd/tei", "world", "-workspace", after]));
function requiredOf(snapshot) {
  const data = snapshot.semantics.flatMap(ns => ns.data || []).find(d => d.id === dataID);
  const attribute = data?.attributes?.find(a => a.id === attributeID);
  assert.ok(attribute, "selected canonical Attribute is missing or ambiguous");
  return attribute.required === true;
}

// Match the existing Stage1/Stage2 Source digest contract: relative name,
// length, exact bytes, and a separator for each file. Reject symlinks.
function workspaceDigest(root) {
  const files = [];
  function walk(folder) {
    for (const entry of readdirSync(folder, { withFileTypes: true })) {
      const pathname = join(folder, entry.name);
      const stat = lstatSync(pathname);
      assert.equal(stat.isSymbolicLink(), false, "symlink in authored Source");
      if (stat.isDirectory()) walk(pathname);
      else {
        assert.equal(stat.isFile(), true, "unexpected nonregular Source entry");
        files.push(pathname);
      }
    }
  }
  walk(root);
  files.sort((a, b) => {
    const nameA = relative(root, a).replaceAll("\\", "/");
    const nameB = relative(root, b).replaceAll("\\", "/");
    return nameA < nameB ? -1 : nameA > nameB ? 1 : 0;
  });
  assert.ok(files.length >= 2, "authoring workspace needs Source and manifest");
  const hash = createHash("sha256");
  for (const filename of files) {
    const bytes = readFileSync(filename);
    hash.update(relative(root, filename).replaceAll("\\", "/"), "utf8");
    hash.update("\0").update(String(bytes.length)).update("\0").update(bytes).update("\n");
  }
  return "sha256:" + hash.digest("hex");
}

assert.notEqual(before, after);
assert.equal(existsSync(after), false, "the changed Workspace must not be reused");
assert.equal(existsSync(sourceAt(before)), true);
const baselineSource = readFileSync(sourceAt(before));
const originalRow = "| snapshotFingerprint | std.text | no | Snapshot |";
const editedRow = "| snapshotFingerprint | std.text | yes | Snapshot |";
const original = baselineSource.toString("utf8");
assert.equal(original.split(originalRow).length, 2, "expected exactly one old Data cell");
assert.equal(original.includes(editedRow), false);
const baselineDigest = workspaceDigest(before);
cpSync(before, after, { recursive: true, force: false, errorOnExist: true });
const revisionBefore = revision();
assert.equal(requiredOf(world()), false);

// The Stage2 Windows job uses the exact existing offline Studio form in
// real Edge. The VM remains an explicit non-browser mode for Node-only tests.
// Neither path grants Source write permission or Owner meaning acceptance.
const cli = join(repo, "reference/studio-authoring/cli.mjs");
const formHTML = command(process.execPath, [
  cli, "edit-form", "-workspace", after, "-focus", "attribute:" + attributeID,
  "-edit-mode", "required"
]);
const scripts = [...formHTML.matchAll(/<script>([\s\S]*?)<\/script>/g)];
assert.equal(scripts.length, 1);
const script = scripts[0][1];
const cspHash = createHash("sha256").update(script).digest("base64");
assert.ok(formHTML.includes("sha256-" + cspHash), "form script is not CSP-pinned");
assert.ok(formHTML.includes("connect-src &#39;none&#39;"), "form is not offline");
const values = {
  "model-revision": revisionBefore,
  "subject": "attribute:" + attributeID,
  "view": "required",
  "candidate-id": "stage2-required-" + cspHash.slice(0, 12).replaceAll("/", "_").replaceAll("+", "_"),
  "intent": "Studioで既存Dataの必須指定を変更する",
  "required-choice": "yes",
};
let editJSON;
let editorTransport = "exact-offline-form-script-vm";
let realBrowserEditing = "not-observed-in-this-attempt";
let browserScreenshot = null;
const browserPageErrors = [];
const browserNetworkRequests = [];
const playwrightRoot = process.env.TEI_STAGE_PLAYWRIGHT_ROOT?.trim();
if (playwrightRoot) {
  const { pathToFileURL } = await import("node:url");
  const modulePath = join(resolve(playwrightRoot), "node_modules/playwright-core/index.mjs");
  assert.ok(existsSync(modulePath), "PLAYWRIGHT_CORE_NOT_INSTALLED");
  const { chromium } = await import(pathToFileURL(modulePath).href);
  const browser = await chromium.launch({ channel: "msedge", headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 1240, height: 900 } });
    try {
      const page = await context.newPage();
      page.on("pageerror", error => browserPageErrors.push(error.message));
      page.on("request", request => {
        if (/^https?:/i.test(request.url())) browserNetworkRequests.push(request.url());
      });
      // This is a disposable form, not a saved Workspace source.
      const formFile = join(dirname(receiptPath), "tei-studio-stage-required-form.html");
      writeFileSync(formFile, formHTML, { flag: "wx", mode: 0o600 });
      await page.goto(pathToFileURL(formFile).href, { waitUntil: "load" });
      assert.equal(await page.title(), "TEI Studio 編集候補の作成");
      assert.equal(await page.locator("#model-revision").inputValue(), revisionBefore);
      assert.equal(await page.locator("#subject").inputValue(), "attribute:" + attributeID);
      assert.equal(await page.locator("#model-revision").getAttribute("readonly"), "");
      assert.equal(await page.locator("#subject").getAttribute("readonly"), "");
      await page.locator("#candidate-id").fill(values["candidate-id"]);
      await page.locator("#intent").fill(values.intent);
      await page.locator("#required-choice").selectOption("yes");
      await page.locator('button[type="submit"]').click();
      editJSON = await page.locator("#proposal-json").inputValue();
      assert.match(await page.locator("#notice").textContent(), /未審査/);
      assert.deepEqual(browserPageErrors, [], "broken offline Editor script");
      assert.deepEqual(browserNetworkRequests, [], "offline form attempted network access");
      browserScreenshot = join(dirname(receiptPath), "tei-studio-stage-required-form.png");
      await page.screenshot({ path: browserScreenshot, fullPage: true });
      editorTransport = "real-edge-offline-form-playwright";
      realBrowserEditing = "observed-in-this-attempt";
    } finally {
      await context.close();
    }
  } finally {
    await browser.close();
  }
} else {
  // Non-browser smoke explicitly does not establish UI interaction.
  const elements = new Map(Object.entries(values).map(([k, value]) => [k, { value }]));
  const output = { value: "", focus() {}, select() {} };
  const notice = { textContent: "" };
  elements.set("proposal-json", output);
  elements.set("notice", notice);
  let submit;
  elements.set("studio-edit", {
    addEventListener(event, callback) {
      assert.equal(event, "submit");
      submit = callback;
    }
  });
  runInNewContext(script, { document: { getElementById: id => elements.get(id) || null }, JSON });
  assert.equal(typeof submit, "function");
  let prevented = false;
  submit({ preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.match(notice.textContent, /未審査/);
  editJSON = output.value;
}
const edit = JSON.parse(editJSON);
assert.equal(edit.schema, "tei.reference.studio-edit/v0");
assert.equal(edit.expectedModelRevision, revisionBefore);
assert.equal(edit.subject, "attribute:" + attributeID);
assert.equal(edit.operation, "set-attribute-required");
assert.equal(edit.required, true);

// All transport and semantic decisions go through the existing real CLIs.
mkdirSync(dirname(receiptPath), { recursive: true });
const editFile = join(dirname(receiptPath), "tei-studio-stage-required-edit.json");
const candidateFile = join(dirname(receiptPath), "tei-studio-stage-required-candidate.json");
writeFileSync(editFile, editJSON, { flag: "wx", mode: 0o600 });
const candidateText = command(process.execPath, [
  cli, "propose", "-workspace", after, "-edit-file", editFile
]);
const candidate = JSON.parse(candidateText);
assert.equal(candidate.schema, "tei.reference.change-candidate/v0");
assert.equal(candidate.baseModelRevision, revisionBefore);
assert.deepEqual(candidate.delta, [{
  op: "set-attribute-required", id: attributeID, required: true
}]);
writeFileSync(candidateFile, candidateText, { flag: "wx", mode: 0o600 });
const reviewInput = ["-candidate", candidateFile];
const preview = JSON.parse(acceptance("preview", ...reviewInput));
assert.equal(preview.acceptable, true, JSON.stringify(preview.stages));
assert.equal(preview.sourceEdits.length, 1);
assert.equal(preview.sourceEdits[0].path, sourceRel);
assert.equal(preview.sourceEdits[0].insertedText, "yes");
assert.equal(preview.sourceEdits[0].removedText, "no");
assert.ok(readFileSync(sourceAt(after)).equals(baselineSource), "Preview changed Source");
const review = JSON.parse(acceptance("review", ...reviewInput, "-format", "json"));
assert.match(review.reviewFingerprint, /^sha256:[0-9a-f]{64}$/);
assert.ok(readFileSync(sourceAt(after)).equals(baselineSource), "Review changed Source");
const unreviewed = spawnSync("go", [
  "run", "./tools/reference-shared-acceptance", "accept",
  "-workspace", after, ...reviewInput
], { cwd: repo, encoding: "utf8", env: goEnv, timeout: 240_000 });
assert.notEqual(unreviewed.status, 0, "unreviewed Source write was allowed");
assert.ok(readFileSync(sourceAt(after)).equals(baselineSource));
const decision = JSON.parse(acceptance("accept", ...reviewInput,
  "-expected-review-fingerprint", review.reviewFingerprint));
assert.equal(decision.decision, "accepted", JSON.stringify(decision));
assert.equal(decision.sourceChanged, true);
const expectedSource = original.replace(originalRow, editedRow);
const savedSource = readFileSync(sourceAt(after));
assert.equal(savedSource.toString("utf8"), expectedSource, "Accept modified unrelated Source bytes");
assert.equal(requiredOf(world()), true, "actual accepted World did not reload the changed Data");
const revisionAfter = revision();
assert.notEqual(revisionAfter, revisionBefore);
assert.equal(workspaceDigest(before), baselineDigest, "the reference before Workspace changed");

const receipt = {
  schema: "tei.reference.studio-stage-source-accept/v0",
  source: sourceRel, data: dataID, attribute: attributeID,
  editorTransport,
  realBrowserEditing,
  browserScreenshotDigest: browserScreenshot ? sha(readFileSync(browserScreenshot)) : null,
  browserPageErrors, browserNetworkRequests,
  expectedModelRevision: revisionBefore, savedModelRevision: revisionAfter,
  formScriptDigest: "sha256:" + createHash("sha256").update(script).digest("hex"),
  editDigest: sha(Buffer.from(editJSON)),
  candidateDigest: sha(Buffer.from(candidateText)),
  candidateFingerprint: preview.candidateFingerprint,
  reviewFingerprint: review.reviewFingerprint,
  beforeWorkspaceDigest: baselineDigest,
  afterWorkspaceDigest: workspaceDigest(after),
  beforeSourceDigest: sha(baselineSource),
  afterSourceDigest: sha(savedSource),
  sourceChangedOneCell: true,
  previewAndReviewDidNotWriteSource: true,
  unreviewedAcceptRefused: true,
  automatedReferenceAccept: true,
  ownerMeaningAcceptance: "not-established",
  formalBuildIntent: "not-established",
};
writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + "\n", { flag: "wx", mode: 0o600 });
process.stdout.write(JSON.stringify(receipt) + "\n");
