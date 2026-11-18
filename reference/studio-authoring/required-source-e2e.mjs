#!/usr/bin/env node
// Full Reference editing candidate -> existing Shared Acceptance Source
// loop, executed only on an isolated temporary tei.authoring/v1 Workspace.
// This is NOT the Wails Studio Editor, a human Acceptance, or Build authority.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";

if (process.env.TEI_STUDIO_E2E !== "1") {
  process.stderr.write("Set TEI_STUDIO_E2E=1 for isolated Source-edit acceptance.\n");
  process.exit(2);
}
const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const studioCLI = join(root, "reference/studio-authoring/cli.mjs");
const scratch = mkdtempSync(join(tmpdir(), "tei-studio-required-source-"));
let checks = 0;
function checked(value, explanation) {
  assert.ok(value, explanation);
  checks++;
}
function command(exe, args, ok = true) {
  const r = spawnSync(exe, args, {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 120_000
  });
  if (r.error) throw r.error;
  if (ok && r.status !== 0)
    throw new Error("Command failed: " + exe + " " + args.join(" ") +
      "\n" + r.stderr + "\n" + r.stdout.slice(0, 7000));
  if (!ok && r.status === 0)
    throw new Error("Unsafe command unexpectedly succeeded: " + args.join(" "));
  return r;
}
const tei = args => command("go", ["run", "./cmd/tei", ...args]);
const acceptance = (args, ok = true) =>
  command("go", ["run", "./tools/reference-shared-acceptance", ...args], ok);
const studio = (args, ok = true) =>
  command(process.execPath, [studioCLI, ...args], ok);
const sha = bytes => "sha256:" + createHash("sha256").update(bytes).digest("hex");
const result = {};
try {
  const workspace = join(scratch, "source");
  mkdirSync(join(workspace, "semantic/studio"), { recursive: true });
  const manifest = "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n";
  const sourcePath = join(workspace, "semantic/studio/inspection-response.tei.md");
  const beforeSource = "# Studio response\n\n" +
    "This prose contains no and yes; preserve the original bytes.\n\n" +
    "<!-- tei: data InspectionResponse in studio -->\n" +
    "| attribute | type | required | display |\n" +
    "| --- | --- | --- | --- |\n" +
    "| schemaVersion | std.text | yes | Schema |\n" +
    "| snapshotFingerprint | std.text | no | Snapshot |\n";
  writeFileSync(join(workspace, "tei.workspace.yaml"), manifest);
  writeFileSync(sourcePath, beforeSource);
  const beforeRev = JSON.parse(acceptance(["revision", "-workspace", workspace]).stdout).modelRevision;
  const beforeWorld = JSON.parse(tei(["world", "-workspace", workspace]).stdout);
  const attributes = beforeWorld.semantics.flatMap(ns => ns.data || [])
    .find(x => x.id === "studio.inspection_response")?.attributes;
  const attribute = attributes?.find(a =>
    a.id === "studio.inspection_response.snapshot_fingerprint");
  checked(attribute?.required === false, "real authored Data starts optional");

  // The real Studio CLI renders the real inline editor JS using the World
  // and revision from this exact workspace. Execute that JS, not a fabricated
  // edit object. Its CSP-pinned script can only update the output textarea.
  const html = studio([
    "edit-form", "-workspace", workspace,
    "-focus", "attribute:studio.inspection_response.snapshot_fingerprint",
    "-edit-mode", "required"
  ]).stdout;
  const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)];
  checked(scripts.length === 1, "one product Reference editor script");
  const inline = scripts[0][1], digest = createHash("sha256").update(inline).digest("base64");
  checked(html.includes("script-src &#39;sha256-" + digest + "&#39;") &&
    html.includes("connect-src &#39;none&#39;"), "form CSP pins offline script");
  function initial(id) {
    assert.match(id, /^[a-z-]+$/);
    const match = html.match(new RegExp('<input id="' + id + '"[^>]*value="([^"]*)"'));
    assert.ok(match, "missing form Source field: " + id);
    return match[1];
  }
  const fields = new Map([
    ["view", {value: initial("view")}],
    ["subject", {value: initial("subject")}],
    ["model-revision", {value: initial("model-revision")}],
    ["candidate-id", {value: "studio-required-from-real-editor"}],
    ["intent", {value: "Change snapshotFingerprint to a required field"}],
    ["required-choice", {value: "yes"}],
    ["proposal-json", {value: "", focus() {}, select() {}}],
    ["notice", {textContent: ""}]
  ]);
  let submitted;
  fields.set("studio-edit", {addEventListener(name, callback) {
    assert.equal(name, "submit");
    submitted = callback;
  }});
  runInNewContext(inline, {
    document: {getElementById: id => fields.get(id) || null},
    JSON
  }, {timeout: 1200});
  checked(typeof submitted === "function", "CSP-pinned editor has submit handler");
  submitted({preventDefault() {}});
  const edit = JSON.parse(fields.get("proposal-json").value);
  checked(edit.operation === "set-attribute-required" && edit.required === true &&
    edit.expectedModelRevision === beforeRev &&
    edit.subject === "attribute:studio.inspection_response.snapshot_fingerprint",
    "real editor binds exact subject, revision and explicit new value");
  checked(sha(readFileSync(sourcePath)) === sha(Buffer.from(beforeSource)),
    "the editor made no Source changes");

  // Keep all proposals outside the Source Workspace, so revision changes
  // originate only from the Shared Acceptance writer.
  const editPath = join(scratch, "editor-proposal.json");
  const candidatePath = join(scratch, "shared-candidate.json");
  writeFileSync(editPath, JSON.stringify(edit) + "\n");
  const candidate = JSON.parse(studio([
    "propose", "-workspace", workspace, "-edit-file", editPath
  ]).stdout);
  writeFileSync(candidatePath, JSON.stringify(candidate) + "\n");
  checked(candidate.baseModelRevision === beforeRev &&
    candidate.delta.length === 1 &&
    candidate.delta[0].op === "set-attribute-required" &&
    candidate.delta[0].id === "studio.inspection_response.snapshot_fingerprint" &&
    candidate.delta[0].required === true,
    "actual Studio proposal enters the existing typed Acceptance transport");
  const studioReview = JSON.parse(studio([
    "review-edit", "-workspace", workspace, "-edit-file", editPath
  ]).stdout);
  const preview = JSON.parse(acceptance([
    "preview", "-workspace", workspace, "-candidate", candidatePath
  ]).stdout);
  const review = JSON.parse(acceptance([
    "review", "-workspace", workspace, "-candidate", candidatePath,
    "-format", "json"
  ]).stdout);
  checked(studioReview.reviewConsistency.status === "reviewed-consistent" &&
    studioReview.reviewConsistency.readyToAccept === false &&
    preview.acceptable === true && review.previewAcceptable === true &&
    studioReview.review.reviewFingerprint === review.reviewFingerprint,
    "Studio and Shared Acceptance produce the same unaccepted Review");
  checked(preview.semanticDiff?.length === 1 &&
    preview.semanticDiff[0].kind === "attribute" &&
    preview.semanticDiff[0].change === "changed" &&
    preview.changedCollections?.length === 1 &&
    preview.changedCollections[0] === "semantics",
    "semantic diff matches the typed attribute delta, not an accidental patch");
  checked(readFileSync(sourcePath, "utf8") === beforeSource,
    "Preview and Review do not change the original Source");
  acceptance(["accept", "-workspace", workspace, "-candidate", candidatePath], false);
  checked(readFileSync(sourcePath, "utf8") === beforeSource,
    "Accept without a reviewed fingerprint refuses the write");
  const decision = JSON.parse(acceptance([
    "accept", "-workspace", workspace, "-candidate", candidatePath,
    "-expected-review-fingerprint", review.reviewFingerprint
  ]).stdout);
  checked(decision.decision === "accepted" && decision.sourceChanged === true &&
    decision.modelRevisionBefore === beforeRev &&
    decision.reviewBinding?.fingerprint === review.reviewFingerprint,
    "explicit TEST Acceptance binds the exact review fingerprint");
  const afterSource = readFileSync(sourcePath, "utf8");
  const expected = beforeSource.replace(
    "| snapshotFingerprint | std.text | no | Snapshot |",
    "| snapshotFingerprint | std.text | yes | Snapshot |");
  checked(afterSource === expected &&
    readFileSync(join(workspace, "tei.workspace.yaml"), "utf8") === manifest,
    "only the intended Markdown cell changed; all other bytes preserved");
  const afterRev = JSON.parse(acceptance(["revision", "-workspace", workspace]).stdout).modelRevision;
  const afterWorld = JSON.parse(tei(["world", "-workspace", workspace]).stdout);
  const changed = afterWorld.semantics.flatMap(ns => ns.data || [])
    .find(x => x.id === "studio.inspection_response")?.attributes
    .find(a => a.id === "studio.inspection_response.snapshot_fingerprint");
  checked(changed?.required === true && afterRev !== beforeRev,
    "same accepted Source reloads as required=true at a new revision");
  acceptance([
    "accept", "-workspace", workspace, "-candidate", candidatePath,
    "-expected-review-fingerprint", review.reviewFingerprint
  ], false);
  checked(readFileSync(sourcePath, "utf8") === expected,
    "stale reviewed proposal cannot overwrite accepted Source again");
  result.schema = "tei.reference.studio-required-source-loop/v0";
  result.status = "PASS";
  result.checks = checks;
  result.editedSubject = candidate.delta[0].id;
  result.reviewedSource = {
    beforeRevision: beforeRev, afterRevision: afterRev,
    changedSourcePath: "semantic/studio/inspection-response.tei.md",
    sourceBeforeDigest: sha(Buffer.from(beforeSource)),
    sourceAfterDigest: sha(Buffer.from(afterSource)),
    requiredBefore: false, requiredAfter: true
  };
  result.authority = {
    editor: "reference-offline-script-vm", sourceWriteByForm: false,
    acceptance: "automated-scratch-test-only", humanAcceptance: "not-observed",
    studioDesktop: "not-invoked", buildIntent: "not-authorized"
  };
  process.stdout.write(JSON.stringify(result, null, 2) + "\n");
} finally {
  rmSync(scratch, {recursive: true, force: true});
}
