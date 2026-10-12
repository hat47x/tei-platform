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
import { candidateFromEdit, projectWorld, navigateReferenceBusinessStep } from "./bridge.mjs";

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
  if (expectedSuccess && r.status !== 0) {
    // Shared Acceptance writes refusals and stage diagnostics to stdout.
    // Include both streams so a negative result cannot masquerade as a
    // launcher error. The test uses only scratch workspaces.
    const diagnosis = {
      exitStatus: r.status, stderr: r.stderr.slice(0, 8192),
      stdout: r.stdout.slice(0, 16384)
    };
    throw new Error(executable + " " + args.join(" ") + ": " +
      JSON.stringify(diagnosis));
  }
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
function sourceIndex(workspace) {
  return JSON.parse(tei(["source-index", "-workspace", workspace]).stdout);
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
  const beforeFocus = projectWorld(model, beforeRevision, edit.subject,
    { sourceIndex: sourceIndex(workspace) });
  checked(beforeFocus.authority === "derived-readonly", label + ": initial readonly");
  checked(beforeFocus.sourceNavigation.status === "source-index-backed",
    label + ": focus resolves to exact Source Index definition");

  const proposal = candidateFromEdit(model, beforeRevision, edit);
  const path = input(label, ".candidate.json", proposal);
  checked(proposal.baseModelRevision === beforeRevision, label + ": base revision");

  const derivedEdit = input(label, ".derived-edit.json", {
    ...edit, derived: true
  });
  const rejectedDerived = command(process.execPath, [
    join(root, "reference/studio-authoring/cli.mjs"),
    "review-edit", "-workspace", workspace, "-edit-file", derivedEdit
  ], false);
  checked(rejectedDerived.stderr.includes("DERIVED_IS_READONLY_OR_INVALID_EDIT"),
    label + ": Derived edit fails for the expected safety reason, not an unrelated CLI crash (N05)");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(before),
    label + ": Derived edit cannot produce Source changes (N05)");
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
  // Two independent actors have reviewed competing proposals based on R0.
  // The first Acceptance must invalidate the other actor's earlier Review.
  const rivalEdit = { ...edit, id: edit.id + "-rival",
    origin: edit.origin === "studio" ? "ai" : "studio",
    intent: edit.intent + "（別の編集者による競合提案）" };
  const rivalCandidate = candidateFromEdit(model, beforeRevision, rivalEdit);
  const rivalPath = input(label, ".rival-candidate.json", rivalCandidate);
  const rivalPreview = JSON.parse(acceptance([
    "preview", "-workspace", workspace, "-candidate", rivalPath
  ]).stdout);
  const rivalReview = JSON.parse(acceptance([
    "review", "-workspace", workspace, "-candidate", rivalPath, "-format", "json"
  ]).stdout);
  checked(rivalCandidate.id !== proposal.id &&
    rivalCandidate.origin.kind !== proposal.origin.kind &&
    JSON.stringify(rivalCandidate.delta) === JSON.stringify(proposal.delta),
    label + ": independent actors enter the same Shared Acceptance transport");
  checked(rivalPreview.acceptable === true &&
    rivalPreview.currentModelRevision === beforeRevision &&
    rivalReview.previewAcceptable === true &&
    /^sha256:[0-9a-f]{64}$/.test(rivalReview.reviewFingerprint),
    label + ": competing candidate passes independent R0 Preview and Review");

  const editPath = input(label, ".edit.json", edit);
  const studioReview = JSON.parse(command(process.execPath, [
    join(root, "reference/studio-authoring/cli.mjs"),
    "review-edit", "-workspace", workspace, "-edit-file", editPath
  ]).stdout);
  checked(studioReview.schema === "tei.reference.studio-review-edit/v0" &&
    studioReview.explicitAcceptance === "not-performed" &&
    studioReview.sourceWriteByThisCommand === false &&
    studioReview.modelRevision === beforeRevision,
    label + ": one-shot Studio review is read-only, not Acceptance");
  checked(studioReview.reviewConsistency.status === "reviewed-consistent" &&
    studioReview.reviewConsistency.readyToAccept === false,
    label + ": Preview and Review agree on R0 without conferring approval");
  checked(studioReview.preview.acceptable === true &&
    studioReview.review.reviewFingerprint === review.reviewFingerprint,
    label + ": Studio review delegates exactly to Shared Acceptance");
  checked(JSON.stringify(studioReview.candidate.delta) === JSON.stringify(proposal.delta),
    label + ": Studio review does not alter the proposed delta");
  if (label === "er") {
    const report = command(process.execPath, [
      join(root, "reference/studio-authoring/cli.mjs"),
      "inspect", "-workspace", workspace, "-focus", "relation:sales.order.lines",
      "-format", "html"
    ]).stdout;
    checked(report.includes("<!doctype html>") &&
      report.includes('id="business"') &&
      report.includes("source-index-backed") &&
      report.includes("verified-at-read"),
      "er: inspect HTML shows four lenses and the verified Source location");
    const reviewPage = command(process.execPath, [
      join(root, "reference/studio-authoring/cli.mjs"),
      "review-edit", "-workspace", workspace, "-edit-file", editPath,
      "-format", "html"
    ]).stdout;
    checked(reviewPage.includes("not-performed") &&
      reviewPage.includes("未適用") && reviewPage.includes("source"),
      "er: review HTML distinguishes proposal and unaccepted Source edits");
  }
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(before),
    label + ": Preview, Review and Studio review do not mutate Source");

  // This is an intentional, automated TEST decision in an isolated scratch.
  // It is not evidence that a person approved the content.
  const decision = JSON.parse(acceptance([
    "accept", "-workspace", workspace, "-candidate", path,
    "-expected-review-fingerprint", review.reviewFingerprint
  ]).stdout);
  checked(decision.decision === "accepted" && decision.sourceChanged === true,
    label + ": test harness Accepted source");
  checked(decision.modelRevisionBefore === beforeRevision &&
    decision.baseModelRevision === beforeRevision &&
    decision.candidateFingerprint === preview.candidateFingerprint &&
    preview.currentModelRevision === beforeRevision,
    label + ": R0 and candidate fingerprint are consistent across Preview/Decision");
  checked(decision.reviewBinding?.fingerprint === review.reviewFingerprint &&
    decision.reviewBinding?.projection?.reviewFingerprint === review.reviewFingerprint,
    label + ": Accepted Decision retains the exact reviewed projection");
  const afterRevision = revision(workspace);
  checked(decision.modelRevisionAfter === afterRevision &&
    preview.modelRevisionAfter === afterRevision,
    label + ": R1 matches accepted Decision and projected Source plan");
  checked(afterRevision !== beforeRevision, label + ": Source revision changed");
  const changed = changedPaths(before, managedDigests(workspace));
  const allowedPaths = Array.isArray(expectedPath) ? expectedPath.slice().sort() : [expectedPath];
  checked(JSON.stringify(changed) === JSON.stringify(allowedPaths),
    label + ": only the intended source file is changed: " + JSON.stringify(changed));
  const afterWorld = world(workspace);
  const after = projectWorld(afterWorld, afterRevision,
    expectedKind + ":" + expectedID, { sourceIndex: sourceIndex(workspace) });
  checked(after.modelRevision === afterRevision, label + ": new revision in projection");
  checked(after.sourceNavigation.status === "source-index-backed",
    label + ": accepted subject resolves to its exact updated Source anchor");
  const liveInspect = JSON.parse(command(process.execPath, [
    join(root, "reference/studio-authoring/cli.mjs"),
    "inspect", "-workspace", workspace,
    "-focus", expectedKind + ":" + expectedID
  ]).stdout);
  checked(liveInspect.modelRevision === afterRevision &&
    liveInspect.sourceNavigation.byteVerification?.status === "verified-at-read" &&
    liveInspect.sourceNavigation.byteVerification?.fingerprint ===
      liveInspect.sourceNavigation.artifactFingerprint,
    label + ": Source Index bytes and exact position are verified on live inspect");
  if (expectedKind === "relation") {
    checked(after.views.er.edges.some(v => v.id === expectedID),
      label + ": accepted Relation appears in reprojected ER");
    checked(after.views.table.value.relations.some(v => v.id === expectedID),
      label + ": accepted Relation appears in reprojected table");
  } else {
    const rule = after.views.table.value.validations.find(v => v.id === expectedID);
    checked(!!rule, label + ": accepted Validation appears in reprojected table");
    if (edit.applicationUse) {
      checked(rule.applicationSelection.some(u =>
        u.application === edit.applicationUse.application && u.status === "source-declared"),
        label + ": rule and explicit Application selection reproject together");
      checked((afterWorld.applications.find(a =>
        a.id === edit.applicationUse.application)?.validations || []).includes(expectedID),
        label + ": accepted Application declares the Validation");
      checked(after.realizationReadiness.status === "not-verified",
        label + ": Source Acceptance is not Target/Provider readiness");
    }
  }
  const beforeStale = managedDigests(workspace);
  const stale = acceptance([
    "preview", "-workspace", workspace, "-candidate", path
  ], false);
  const staleJSON = JSON.parse(stale.stdout);
  checked(staleJSON.acceptable === false &&
    (staleJSON.stages || []).some(v => v.name === "base-revision" && v.status === "fail"),
    label + ": stale revision fails closed (N06)");
  const staleAccept = acceptance([
    "accept", "-workspace", workspace, "-candidate", path,
    "-expected-review-fingerprint", review.reviewFingerprint
  ], false);
  const staleDecision = JSON.parse(staleAccept.stdout);
  checked(staleDecision.decision === "refused" && staleDecision.sourceChanged === false &&
    (staleDecision.refusals || []).some(d => d.code === "STALE_BASE_MODEL_REVISION"),
    label + ": stale Accepted Decision is explicitly refused (N06)");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(beforeStale),
    label + ": stale Preview and stale Accept cannot modify newer Source");
  // A distinct second actor's already-reviewed proposal also loses R0.
  // This demonstrates optimistic concurrency, not simultaneous OS writes.
  const rivalAccepted = JSON.parse(acceptance([
    "accept", "-workspace", workspace, "-candidate", rivalPath,
    "-expected-review-fingerprint", rivalReview.reviewFingerprint
  ], false).stdout);
  checked(rivalAccepted.decision === "refused" &&
    rivalAccepted.sourceChanged === false &&
    (rivalAccepted.refusals || []).some(r =>
      r.code === "STALE_BASE_MODEL_REVISION"),
    label + ": independently reviewed competing proposal is refused on R1 (N06)");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(beforeStale) &&
    revision(workspace) === afterRevision,
    label + ": rejected competing actor cannot overwrite Accepted R1");

  return {
    schema: "tei.reference.studio-source-loop-receipt/v0",
    name: label, scenario: "V03-V04-N06",
    evidenceClass: "temporary-synthetic-scratch",
    humanAcceptance: "not-observed",
    actualProviderResult: "not-observed",
    runtimeObservation: "not-observed",
    sourceLineage: {
      baseModelRevision: beforeRevision,
      previewCurrentRevision: preview.currentModelRevision,
      previewExpectedRevisionAfter: preview.modelRevisionAfter,
      acceptedDecisionBefore: decision.modelRevisionBefore,
      acceptedDecisionAfter: decision.modelRevisionAfter,
      verifiedReloadRevision: afterRevision,
      candidateFingerprint: preview.candidateFingerprint,
      reviewFingerprint: review.reviewFingerprint
    },
    changed, acceptedSubject: expectedID,
    staleCandidateRefused: true, concurrentRivalRefused: true
  };
}

// A fourth view is connected using the real #1125 Go producer; it remains a
// reference-only Business fixture, not a fourth Accepted Canonical source.
function testReferenceBusinessNavigation() {
  const workspace = newWorkspace("business-navigation");
  const before = managedDigests(workspace);
  const r0 = revision(workspace);
  const w = world(workspace);
  const dfd = JSON.parse(command("go", [
    "run", "./tools/reference-business-dfd", "-format", "json",
    "-workspace", workspace
  ]).stdout);
  checked(dfd.schemaVersion === "business-dfd-projection-v0" &&
    dfd.authority === "derived-read-only" &&
    dfd.canonicalProcedureUsed === false,
    "business: producer is explicitly noncanonical");
  const focus = "validation:sales.order_line.quantity_positive";
  const projected = projectWorld(w, r0, focus,
    { businessDFD: dfd, sourceIndex: sourceIndex(workspace) });
  checked(projected.modelRevision === r0 && projected.authority === "derived-readonly",
    "business: four lenses are one read-only World snapshot");
  checked(projected.views.table.value?.validations.some(v => v.subject === focus),
    "business: table retains exact Validation ID");
  checked(projected.views.er.edges.some(e => e.id === "sales.order.lines"),
    "business: ER retains exact Relation ID");
  checked(projected.views.function.items.some(p =>
    p.relation?.id === "sales.order.lines"),
    "business: Function retains the same Relation ID");
  const linked = projected.views.business.items.find(s => s.stepID === "BP-05");
  checked(!!linked && linked.explicitMeaning.some(v => v.subject === focus &&
    v.status === "exact-id-matches-current-world"),
    "business: Go-produced Step has an explicit Validation link");
  checked(projected.views.business.authority === "noncanonical-reference" &&
    projected.views.business.freshness === "unverified-no-model-revision-binding",
    "business: no Accepted Business or same-revision claim");
  const reverse = navigateReferenceBusinessStep(w, r0, dfd, "BP-05");
  checked(reverse.candidates.some(c =>
    c.focus === focus && c.status === "identity-match-only"),
    "business: reverse Step navigation retains exact Validation ID");
  checked(reverse.editable === false &&
    reverse.acceptanceTargetStatus === "unresolved-use-shared-acceptance",
    "business: reverse navigation cannot grant Source edit authority");
  const dfdPath = input("business-navigation", ".dfd.json", dfd);
  const cliNav = JSON.parse(command(process.execPath, [
    join(root, "reference/studio-authoring/cli.mjs"), "inspect",
    "-workspace", workspace, "-focus", "step:BP-05",
    "-business-dfd", dfdPath
  ]).stdout);
  checked(cliNav.schema === reverse.schema &&
    cliNav.modelRevision === r0 &&
    cliNav.candidates.some(c => c.focus === focus),
    "business: CLI reproduces the same read-only reverse navigation");
  checked(JSON.stringify(managedDigests(workspace)) === JSON.stringify(before),
    "business: DFD production and both navigation directions leave Source unchanged");
  checked(revision(workspace) === r0,
    "business: no revision changed during four-lens reference navigation");
  return {
    status: "PASS", authority: "noncanonical-reference",
    provenance: "Go-produced reference-business-dfd",
    modelRevision: r0, focus, stepID: "BP-05",
    humanAcceptance: "not-observed",
    actualProviderResult: "not-observed"
  };
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
    validationKind: "positive",
    applicationUse: { application: "order_entry",
      placement: "applications/order-entry/application.tei" }
  }, "validation", "sales.order_line.unit_price_positive",
    ["applications/order-entry/application.tei", "semantic/sales/validation.tei"]);
  const businessReference = testReferenceBusinessNavigation();
  const checkedHead = command("git", ["rev-parse", "HEAD"]).stdout.trim();
  process.stdout.write(JSON.stringify({
    status: "PASS", scope: "temporary scratch only; no human acceptance",
    head: checkedHead, checks, cases: [er, table], businessReference,
    claimLimits: [
      "not-human-approved", "not-actual-db-provider",
      "not-web-target-execution", "not-full-repository-qualification"
    ]
  }, null, 2) + "\n");
} finally {
  rmSync(created, { recursive: true, force: true });
}
