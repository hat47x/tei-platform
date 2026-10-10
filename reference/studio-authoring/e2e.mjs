#!/usr/bin/env node
// Opt-in exact-checkout integration pressure. All mutations target temp copies.
// A test harness Accept is NOT a human Acceptance or production authority.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { cpSync, lstatSync, mkdtempSync, readFileSync, readdirSync,
  rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { candidateFromEdit, projectWorld } from "./bridge.mjs";

if (process.env.TEI_STUDIO_E2E !== "1") {
  process.stderr.write("Set TEI_STUDIO_E2E=1 to run scratch-only Source Acceptance tests.\n");
  process.exit(2);
}
const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const baseline = join(root, "reference", "workspace");
const created = mkdtempSync(join(tmpdir(), "tei-studio-1124-"));
let checks = 0;
const checked = (condition, message) => { assert.ok(condition, message); checks++; };

function command(executable, args, expectedSuccess = true) {
  const r = spawnSync(executable, args, {
    cwd: root, encoding: "utf8", maxBuffer: 32 * 1024 * 1024
  });
  if (r.error) throw r.error;
  if (expectedSuccess && r.status !== 0)
    throw new Error(executable + " " + args.join(" ") + ": " + r.stderr);
  if (!expectedSuccess && r.status === 0)
    throw new Error("Expected failure but succeeded: " + executable + " " + args.join(" "));
  return r;
}
const tei = (args, success = true) =>
  command("go", ["run", "./cmd/tei", ...args], success);
const acceptance = (args, success = true) =>
  command("go", ["run", "./tools/reference-shared-acceptance", ...args], success);

function walk(dir, cb, prefix = "") {
  for (const item of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, item.name), rel = join(prefix, item.name);
    if (item.isSymbolicLink() || lstatSync(path).isSymbolicLink())
      throw new Error("REFUSE_SYMLINK_IN_FIXTURE: " + rel);
    if (item.isDirectory()) walk(path, cb, rel);
    else if (item.isFile()) cb(path, rel.replaceAll("\\", "/"));
    else throw new Error("REFUSE_NONREGULAR_FIXTURE: " + rel);
  }
}
function managedDigests(dir) {
  const digests = new Map();
  walk(dir, (path, rel) => {
    if (rel.startsWith(".tei/")) return; // Decision records are derived.
    const sha = createHash("sha256").update(readFileSync(path)).digest("hex");
    digests.set(rel, sha);
  });
  return [...digests].sort((a, b) => a[0].localeCompare(b[0]));
}
function changedPaths(before, after) {
  const b = new Map(before), a = new Map(after);
  return [...new Set([...b.keys(), ...a.keys()])]
    .filter(k => b.get(k) !== a.get(k)).sort();
}
function newWorkspace(label) {
  walk(baseline, () => {}); // refuse symlinks before copying
  const workspace = join(created, label, "workspace");
  cpSync(baseline, workspace, { recursive: true, errorOnExist: true });
  return workspace;
}
function revision(workspace) {
  return JSON.parse(acceptance(["revision", "-workspace", workspace]).stdout).modelRevision;
}
function world(workspace) {
  return JSON.parse(tei(["world", "-workspace", workspace]).stdout);
}
function input(label, suffix, value) {
  const path = join(created, label + suffix);
  writeFileSync(path, JSON.stringify(value, null, 2) + "\n");
  return path;
}
function testLoop(label, edit, expectedKind, expectedID, expectedPath) {
  const workspace = newWorkspace(label);
  const beforeRevision = revision(workspace);
  const before = managedDigests(workspace);
  const model = world(workspace);
  const beforeFocus = projectWorld(model, beforeRevision, edit.subject);
  checked(beforeFocus.authority === "derived-readonly", label + ": initial readonly");

  const proposal = candidateFromEdit(model, beforeRevision, edit);
  const path = input(label, ".candidate.json", proposal);
  checked(proposal.baseModelRevision === beforeRevision, label + ": base revision");

  const preview = JSON.parse(acceptance([
    "preview", "-workspace", workspace, "-candidate", path
  ]).stdout);
  checked(preview.acceptable === true, label + ": Preview must be acceptable");
  checked((preview.semanticDiff || []).some(d =>
    d.kind === expectedKind && d.id === expectedID && d.change === "added"
  ), label + ": exact expected Semantic Diff");
  const review = JSON.parse(acceptance([
    "review", "-workspace", workspace, "-candidate", path, "-format", "json"
  ]).stdout);
  checked(review.previewAcceptable === true &&
    /^sha256:[0-9a-f]{64}$/.test(review.reviewFingerprint),
    label + ": review binding");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(before),
    label + ": Preview and Review do not mutate Source");

  // This is an intentional, automated TEST decision in an isolated scratch.
  // It is not evidence that a person approved the content.
  const decision = JSON.parse(acceptance([
    "accept", "-workspace", workspace, "-candidate", path,
    "-expected-review-fingerprint", review.reviewFingerprint
  ]).stdout);
  checked(decision.decision === "accepted" && decision.sourceChanged === true,
    label + ": test harness Accepted source");
  const afterRevision = revision(workspace);
  checked(afterRevision !== beforeRevision, label + ": Source revision changed");
  const changed = changedPaths(before, managedDigests(workspace));
  checked(JSON.stringify(changed) === JSON.stringify([expectedPath]),
    label + ": only the intended source file is changed: " + JSON.stringify(changed));
  const afterWorld = world(workspace);
  const after = projectWorld(afterWorld, afterRevision,
    expectedKind + ":" + expectedID);
  checked(after.modelRevision === afterRevision, label + ": new revision in projection");
  if (expectedKind === "relation") {
    checked(after.views.er.edges.some(v => v.id === expectedID),
      label + ": accepted Relation appears in reprojected ER");
    checked(after.views.table.value.relations.some(v => v.id === expectedID),
      label + ": accepted Relation appears in reprojected table");
  } else {
    checked(after.views.table.value.validations.some(v => v.id === expectedID),
      label + ": accepted Validation appears in reprojected table");
  }
  const beforeStale = managedDigests(workspace);
  const stale = acceptance([
    "preview", "-workspace", workspace, "-candidate", path
  ], false);
  const staleJSON = JSON.parse(stale.stdout);
  checked(staleJSON.acceptable === false &&
    (staleJSON.stages || []).some(v => v.name === "base-revision" && v.status === "fail"),
    label + ": stale revision fails closed (N06)");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(beforeStale),
    label + ": stale preview leaves Accepted source intact");
  return { name: label, beforeRevision, afterRevision, changed, acceptedSubject: expectedID };
}

try {
  const er = testLoop("er", {
    schema: "tei.reference.studio-edit/v0", id: "e2e-relation",
    origin: "studio", intent: "明細から受注へ戻るRelationを追加する",
    view: "er", subject: "data:sales.order_line",
    operation: "add-relation", relationID: "sales.order_line.order",
    symbol: "order", targetData: "sales.order", cardinality: "one"
  }, "relation", "sales.order_line.order", "semantic/sales/order.tei");
  const table = testLoop("table", {
    schema: "tei.reference.studio-edit/v0", id: "e2e-price-positive",
    origin: "ai", intent: "明細単価に正値Validationを追加する",
    view: "table", subject: "attribute:sales.order_line.unit_price",
    operation: "add-validation", validationID: "sales.order_line.unit_price_positive",
    validationKind: "positive"
  }, "validation", "sales.order_line.unit_price_positive", "semantic/sales/validation.tei");
  process.stdout.write(JSON.stringify({
    status: "PASS", scope: "temporary scratch only; no human acceptance",
    checks, cases: [er, table]
  }, null, 2) + "\n");
} finally {
  rmSync(created, { recursive: true, force: true });
}
