#!/usr/bin/env node
// One actual Chrome editor proposal -> Shared Acceptance -> authored Source
// readback. This is a scratch-only Reference test, not human approval.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync,
         readdirSync, lstatSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.env.TEI_STUDIO_ACCEPTED_SOURCE_LOOP !== "1" || process.argv.length !== 4)
  throw new Error("scratch-only: TEI_STUDIO_ACCEPTED_SOURCE_LOOP=1 node required-source-loop.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");
const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const playwrightRoot = resolve(process.argv[2]);
const outputDir = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(playwrightRoot, "node_modules/playwright-core/index.mjs")).href);
const cli = join(repo, "reference/studio-authoring/cli.mjs");
const created = mkdtempSync(join(tmpdir(), "tei-studio-required-accepted-"));
const workspace = join(created, "workspace");
const source = join(workspace, "semantic/studio/inspection-response.tei.md");
const dataID = "studio.inspection_response";
const attributeID = "studio.inspection_response.snapshot_fingerprint";
const sourceRel = "semantic/studio/inspection-response.tei.md";
const beforeText = [
  "# Studio Data required editing", "",
  "A mention of no outside the marked Data table must survive.", "",
  "<!-- tei: data InspectionResponse in studio -->",
  "| attribute | type | required | display |",
  "| --- | --- | --- | --- |",
  "| schemaVersion | std.text | yes | Schema |",
  "| snapshotFingerprint | std.text | no | Snapshot |", "",
  "Unmarked table:", "",
  "| required | other |",
  "| --- | --- |",
  "| no | keep |", ""
].join("\n");
const afterText = beforeText.replace(
  "| snapshotFingerprint | std.text | no | Snapshot |",
  "| snapshotFingerprint | std.text | yes | Snapshot |");
const hash = buffer => "sha256:" + createHash("sha256").update(buffer).digest("hex");
const originalHead = run("git", ["rev-parse", "HEAD"]).stdout.trim();

function run(command, args, expectedSuccess = true) {
  const p = spawnSync(command, args, {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 120_000,
  });
  if (p.error) throw p.error;
  if (expectedSuccess ? p.status !== 0 : p.status === 0)
    throw new Error("unexpected command status " + command + " " + args.join(" ") +
      " status=" + p.status + " stderr=" + p.stderr.slice(0, 1500) +
      " stdout=" + p.stdout.slice(0, 1500));
  return p;
}
function go(args, success = true) {
  return run("go", ["run", "./tools/reference-shared-acceptance", ...args], success);
}
function teiWorld() {
  return JSON.parse(run("go", ["run", "./cmd/tei", "world", "-workspace", workspace]).stdout);
}
function canonicalRequired() {
  const data = teiWorld().semantics.flatMap(ns => ns.data || []).find(d => d.id === dataID);
  assert.ok(data, "the actual TEI World must contain studio Data");
  const attributes = data.attributes || [];
  assert.equal(attributes.length, 2);
  const target = attributes.find(a => a.id === attributeID);
  assert.ok(target, "the exact authored Attribute must resolve");
  return target.required === true;
}
function revision() {
  return JSON.parse(go(["revision", "-workspace", workspace]).stdout).modelRevision;
}
function managed() {
  const out = new Map();
  function visit(dir, prefix = "") {
    for (const item of readdirSync(dir, { withFileTypes: true })) {
      const rel = prefix ? prefix + "/" + item.name : item.name;
      if (item.isSymbolicLink() || lstatSync(join(dir, item.name)).isSymbolicLink())
        throw new Error("symlink in Source test Workspace: " + rel);
      if (item.isDirectory()) visit(join(dir, item.name), rel);
      else if (item.isFile() && !rel.startsWith(".tei/"))
        out.set(rel, hash(readFileSync(join(dir, item.name))));
    }
  }
  visit(workspace);
  return out;
}
function difference(before, after) {
  return [...new Set([...before.keys(), ...after.keys()])]
    .filter(k => before.get(k) !== after.get(k)).sort();
}

let browser;
try {
  mkdirSync(dirname(source), { recursive: true });
  mkdirSync(outputDir, { recursive: true });
  writeFileSync(join(workspace, "tei.workspace.yaml"),
    "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n");
  writeFileSync(source, beforeText);
  const baseline = managed();
  const r0 = revision();
  assert.equal(canonicalRequired(), false);

  // CLI projects the actual World and revision into its CSP-pinned HTML.
  const form = run(process.execPath, [cli, "edit-form",
    "-workspace", workspace, "-focus", "attribute:" + attributeID,
    "-edit-mode", "required"]).stdout;
  assert.match(form, /Content-Security-Policy/);
  assert.match(form, /現在の必須指定：no/);
  assert.doesNotMatch(form, new RegExp(workspace.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  const formPath = join(created, "editor.html");
  writeFileSync(formPath, form, { mode: 0o600 });

  browser = await chromium.launch({
    channel: "chrome", headless: true, args: ["--no-sandbox"]
  });
  const context = await browser.newContext({viewport: {width: 1280, height: 900}});
  try {
    const page = await context.newPage();
    const errors = [], network = [];
    page.on("pageerror", error => errors.push(error.message));
    page.on("request", request => {
      if (/^https?:/i.test(request.url())) network.push(request.url());
    });
    await page.goto(pathToFileURL(formPath).href, { waitUntil: "load" });
    assert.equal(await page.locator("#subject").inputValue(), "attribute:" + attributeID);
    assert.equal(await page.locator("#model-revision").inputValue(), r0);
    await page.locator("#candidate-id").fill("chrome-required-accepted-source");
    await page.locator("#intent").fill("Studio Dataの既存属性を必須へ変更する");
    await page.locator("#required-choice").selectOption("yes");
    await page.locator('button[type="submit"]').click();
    const draft = JSON.parse(await page.locator("#proposal-json").inputValue());
    assert.deepEqual({
      schema: draft.schema, view: draft.view, operation: draft.operation,
      required: draft.required, revision: draft.expectedModelRevision,
      subject: draft.subject,
    }, {
      schema: "tei.reference.studio-edit/v0", view: "required",
      operation: "set-attribute-required", required: true,
      revision: r0, subject: "attribute:" + attributeID,
    });
    assert.match(await page.locator("#notice").innerText(), /未審査/);
    assert.deepEqual(errors, []);
    assert.deepEqual(network, []);
    await page.screenshot({path: join(outputDir, "required-editor.png"), fullPage: true});

    const draftPath = join(created, "draft.json");
    const candidatePath = join(created, "candidate.json");
    writeFileSync(draftPath, JSON.stringify(draft) + "\n", { mode: 0o600 });
    const candidate = JSON.parse(run(process.execPath, [cli, "propose",
      "-workspace", workspace, "-edit-file", draftPath]).stdout);
    assert.equal(candidate.schema, "tei.reference.change-candidate/v0");
    assert.equal(candidate.baseModelRevision, r0);
    assert.equal(candidate.origin.kind, "studio");
    assert.deepEqual(candidate.delta, [{
      op: "set-attribute-required", id: attributeID, required: true,
    }]);
    writeFileSync(candidatePath, JSON.stringify(candidate) + "\n", { mode: 0o600 });

    const projected = JSON.parse(run(process.execPath, [cli, "review-edit",
      "-workspace", workspace, "-edit-file", draftPath]).stdout);
    assert.equal(projected.sourceWriteByThisCommand, false);
    assert.equal(projected.explicitAcceptance, "not-performed");
    assert.equal(projected.reviewConsistency.status, "reviewed-consistent");
    assert.equal(projected.preview.acceptable, true);
    assert.equal(projected.review.previewAcceptable, true);
    assert.equal(projected.review.candidateId, candidate.id);
    const reviewFingerprint = projected.review.reviewFingerprint;
    assert.match(reviewFingerprint, /^sha256:[0-9a-f]{64}$/);
    const confirmed = JSON.parse(go(["review", "-workspace", workspace,
      "-candidate", candidatePath, "-format", "json"]).stdout);
    assert.equal(confirmed.reviewFingerprint, reviewFingerprint);

    // Neither a form submission nor review may mutate canonical Source.
    assert.deepEqual([...managed()], [...baseline]);
    assert.equal(revision(), r0);
    assert.equal(canonicalRequired(), false);

    const noReview = go(["accept", "-workspace", workspace,
      "-candidate", candidatePath], false);
    assert.match(noReview.stderr, /expected-review-fingerprint|review/i);
    assert.deepEqual([...managed()], [...baseline]);
    const decision = JSON.parse(go(["accept", "-workspace", workspace,
      "-candidate", candidatePath,
      "-expected-review-fingerprint", reviewFingerprint]).stdout);
    assert.equal(decision.decision, "accepted");
    assert.equal(decision.sourceChanged, true);
    assert.equal(readFileSync(source, "utf8"), afterText);
    assert.deepEqual(difference(baseline, managed()), [sourceRel]);
    const r1 = revision();
    assert.notEqual(r0, r1);
    assert.equal(canonicalRequired(), true);

    // The actual accepted Source is re-read through the same TEI/Studio CLI,
    // not a fabricated Reader response or the browser's own form state.
    const inspect = JSON.parse(run(process.execPath, [cli, "inspect",
      "-workspace", workspace, "-focus", "attribute:" + attributeID]).stdout);
    assert.equal(inspect.modelRevision, r1);
    assert.equal(inspect.sourceNavigation.status, "source-index-backed");

    const stale = go(["accept", "-workspace", workspace,
      "-candidate", candidatePath,
      "-expected-review-fingerprint", reviewFingerprint], false);
    assert.notEqual(stale.status, 0);
    assert.equal(readFileSync(source, "utf8"), afterText);
    assert.equal(revision(), r1);
    assert.equal(run("git", ["rev-parse", "HEAD"]).stdout.trim(), originalHead);
    // Transport ONLY the accepted Source and the separately recorded baseline
    // to a Windows job. Do not export the scratch Decision store as authority.
    for (const label of ["before-workspace", "accepted-workspace"]) {
      mkdirSync(join(outputDir, label, "semantic/studio"), {recursive:true});
      writeFileSync(join(outputDir, label, "tei.workspace.yaml"),
        readFileSync(join(workspace, "tei.workspace.yaml")));
      writeFileSync(join(outputDir, label, sourceRel),
        label === "before-workspace" ? beforeText : readFileSync(source));
    }
    assert.equal(hash(readFileSync(join(outputDir, "accepted-workspace", sourceRel))),
      hash(readFileSync(source)));
    assert.equal(readFileSync(join(outputDir, "before-workspace", sourceRel), "utf8"),
      beforeText);
    const receipt = {
      schema: "tei.reference.studio-required-source-loop/v0",
      head: originalHead,
      status: "PASS",
      actualBrowser: "chrome-headless",
      editorMode: "required",
      browserDraftSchema: draft.schema,
      candidateSchema: candidate.schema,
      modelRevisionBefore: r0,
      modelRevisionAfter: r1,
      changedManagedSourceFiles: difference(baseline, managed()),
      sourceBytesBefore: hash(Buffer.from(beforeText)),
      sourceBytesAfter: hash(readFileSync(source)),
      readerSourceIsRealTEIWorld: true,
      independentCanonicalRequiredBefore: false,
      independentCanonicalRequiredAfter: true,
      previewNonMutating: true,
      reviewNonMutating: true,
      explicitFingerprintAccepted: true,
      missingFingerprintRefused: true,
      staleReviewRefused: true,
      offlineNetworkRequests: network.length,
      humanMeaningAcceptance: "not-observed",
      productWailsEditor: "not-observed",
      windowsSelfBuildAfterAcceptance: "not-observed",
      buildAuthority: "not-established",
    };
    writeFileSync(join(outputDir, "required-source-loop-receipt.json"),
      JSON.stringify(receipt, null, 2) + "\n", { mode: 0o600 });
    process.stdout.write(JSON.stringify(receipt) + "\n");
  } finally {
    await context.close();
  }
} finally {
  if (browser) await browser.close();
  rmSync(created, {recursive: true, force: true});
}
