#!/usr/bin/env node
// Opt-in parent-lane cross-PR E1-A probe. Test-only Acceptance, synthetic provider.
// Run only in a trusted, clean, TEMPORARY integration worktree that contains #1124.
import assert from "node:assert/strict";
import { cpSync, openSync, closeSync, lstatSync, mkdirSync, mkdtempSync,
  readFileSync, realpathSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { spawn, execFileSync } from "node:child_process";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

if (process.env.TEI_CP_LIVE_R1_SMOKE !== "1") {
  process.stderr.write("Set TEI_CP_LIVE_R1_SMOKE=1 for scratch-only E1-A test.\n");
  process.exit(2);
}
const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
assert.equal(process.cwd(), root, "run at repository root");
const baseline = join(root, "reference/workspace");
const temp = mkdtempSync(join(tmpdir(), "tei-cp-r1-web-"));
const workspace = join(temp, "workspace");
const validationID = "sales.order_line.unit_price_positive";
const expectedR = "e5231839a5dcc62cce1181a2731fcb7c70d16736"; // fixed Gate B baseline
let child;

function command(executable, args) {
  return execFileSync(executable, args, {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    env: { ...process.env, GOENV: "off", GOWORK: "off", GOFLAGS: "" }
  });
}
function go(args) { return command("go", args); }

// A second, test-only Web binary uses a Go build overlay. This does not modify
// tracked source or accepted Workspace bytes; it does NOT qualify an external Provider.
function buildNegativePriceTarget(binary) {
  const originalPath = join(root, "tools/reference-first-web-target/reference_order_contract_query_fixture.go");
  const original = readFileSync(originalPath, "utf8");
  const firstLinePrice = /(core\.MustCanonicalID\("sales\.order_line\.unit_price"\):\s*)"4000\.00"/g;
  const matched = [...original.matchAll(firstLinePrice)];
  assert.equal(matched.length, 1,
    "expected exactly one first-order-line unit-price fixture declaration");
  const rewritten = original.replace(firstLinePrice, '$1"0"');
  const replacement = join(temp, "unit-price-negative-fixture.go");
  writeFileSync(replacement, rewritten, { mode: 0o600 });
  const overlay = writeJSON("unit-price-negative-overlay.json",
    { Replace: { [originalPath]: replacement } });
  go(["build", "-mod=readonly", "-overlay", overlay, "-o", binary,
    "./tools/reference-first-web-target"]);
  return {
    sourceFile: "tools/reference-first-web-target/reference_order_contract_query_fixture.go",
    originalSHA256: createHash("sha256").update(original).digest("hex"),
    overlaySHA256: createHash("sha256").update(rewritten).digest("hex"),
    override: "first synthetic line unit_price 4000.00 to 0 (selected order 1001)",
    authority: "test-only-build-overlay",
  };
}

// N11 negative control: run a real HTTP server whose synthetic Command
// succeeds but whose separate Observation read deliberately fails.
// The overlay is isolated to this temporary build; neither tracked files
// nor the accepted Workspace are modified.
function buildUnobservedTarget(binary) {
  const originalPath = join(root, "tools/reference-first-web-target/command_scenario.go");
  const original = readFileSync(originalPath, "utf8");
  const observationRead = /func\(context\.Context\) \(any, error\) \{\s*value, exists := state\[orderID\]\s*if !exists \{\s*return nil, fmt\.Errorf\("order state not found"\)\s*\}\s*return value, nil\s*\}/g;
  const matched = [...original.matchAll(observationRead)];
  assert.equal(matched.length, 1,
    "expected exactly one synthetic post-Command Observation read boundary");
  const rewritten = original.replace(observationRead,
    'func(context.Context) (any, error) { return nil, fmt.Errorf("synthetic observation unavailable (N11 test only)") }');
  const replacement = join(temp, "n11-unobserved-command-scenario.go");
  writeFileSync(replacement, rewritten, { mode: 0o600 });
  const overlay = writeJSON("n11-unobserved-overlay.json",
    { Replace: { [originalPath]: replacement } });
  go(["build", "-mod=readonly", "-overlay", overlay, "-o", binary,
    "./tools/reference-first-web-target"]);
  return {
    sourceFile: "tools/reference-first-web-target/command_scenario.go",
    originalSHA256: createHash("sha256").update(original).digest("hex"),
    overlaySHA256: createHash("sha256").update(rewritten).digest("hex"),
    override: "synthetic post-Command Observation read reports unavailable",
    authority: "test-only-build-overlay"
  };
}

async function stopTarget() {
  if (!child || child.exitCode !== null || child.signalCode !== null) return;
  child.kill("SIGTERM");
  await Promise.race([
    new Promise(resolveExit => child.once("exit", resolveExit)), wait(3000)
  ]);
  if (child.exitCode === null && child.signalCode === null) child.kill("SIGKILL");
}

function acceptance(args) {
  return JSON.parse(go(["run", "./tools/reference-shared-acceptance", ...args]));
}
function files(dir) {
  const result = new Map();
  function visit(base) {
    for (const item of readdirSync(base, { withFileTypes: true })) {
      const full = join(base, item.name), stat = lstatSync(full);
      assert.equal(stat.isSymbolicLink(), false, "scratch contains symlink");
      if (stat.isDirectory()) visit(full);
      else {
        assert.ok(stat.isFile(), "unexpected scratch file type");
        const path = relative(dir, full).replaceAll("\\", "/");
        if (!path.startsWith(".tei/")) {
          result.set(path, createHash("sha256").update(readFileSync(full)).digest("hex"));
        }
      }
    }
  }
  visit(dir);
  return result;
}
function changed(before, after) {
  return [...new Set([...before.keys(), ...after.keys()])]
    .filter(p => before.get(p) !== after.get(p)).sort();
}
function sha256(data) {
  return createHash("sha256").update(data).digest("hex");
}
// A target-derived projection is evidence of loaded Application meaning, not
// evidence that a caller-provided R/T label matches the loaded Source bytes.
function manifestSemantics(manifest) {
  return {
    applicationId: manifest.applicationId,
    semanticUses: manifest.semanticUses ?? [],
    validations: manifest.validations ?? [],
    patterns: manifest.patterns ?? [],
    unresolved: manifest.unresolved ?? []
  };
}
function writeJSON(filename, value) {
  const path = join(temp, filename);
  writeFileSync(path, JSON.stringify(value, null, 2) + "\n", { mode: 0o600 });
  return path;
}
function wait(milliseconds) {
  return new Promise(resolveWait => setTimeout(resolveWait, milliseconds));
}
async function openPort() {
  return new Promise((resolvePort, reject) => {
    const server = createServer();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      server.close(error => error ? reject(error) : resolvePort(address.port));
    });
  });
}
async function ready(base) {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (child.exitCode !== null || child.signalCode !== null) {
      throw new Error("Web Target exited before healthcheck");
    }
    try {
      const response = await fetch(base + "/healthz", { signal: AbortSignal.timeout(900) });
      if (response.ok) return;
    } catch { /* retry until bounded deadline */ }
    await wait(100);
  }
  throw new Error("Web Target did not become healthy");
}

// Optional evidence is published outside the checkout only after all assertions.
// A receipt is a bounded observation of a test-harness run, not real Authority.
function publishReceipts(submissions) {
  const requested = process.env.TEI_CP_EVIDENCE_DIR;
  if (!requested) return null;
  assert.ok(isAbsolute(requested), "evidence directory must be absolute");
  const target = resolve(requested);
  const parent = realpathSync(dirname(target)); // parent must already exist
  const sourceRoot = realpathSync(root);
  const scratchRoot = realpathSync(temp);
  const forbiddenRoots = [sourceRoot, scratchRoot];
  // In the composed test, root is a disposable worktree; the caller's
  // original repository and the composition scratch must also be protected.
  for (const key of ["TEI_CP_ORIGINAL_REPOSITORY_ROOT", "TEI_CP_INTEGRATION_TEMP_DIR"]) {
    if (process.env[key]) forbiddenRoots.push(realpathSync(process.env[key]));
  }
  for (const forbidden of forbiddenRoots) {
    assert.ok(parent !== forbidden && !parent.startsWith(forbidden + sep),
      "evidence directory must be outside repository and ephemeral scratch");
  }
  mkdirSync(target, { mode: 0o700 }); // refuse existing dirs, never overwrite
  for (const [name, data] of Object.entries(submissions)) {
    const safe = JSON.stringify(data, null, 2) + "\n";
    assert.equal(safe.includes("editor-secret"), false, "credential in evidence");
    assert.equal(safe.includes("wrong-secret"), false, "credential in evidence");
    writeFileSync(join(target, name), safe, { flag: "wx", mode: 0o600 });
  }
  return target;
}

async function main() {
  files(baseline); // reject symlinks/irregular fixtures before copying
  cpSync(baseline, workspace, { recursive: true, errorOnExist: true });
  const beforeFiles = files(workspace);
  const r0 = acceptance(["revision", "-workspace", workspace]).modelRevision;
  assert.match(r0, /^sha256:[0-9a-f]{64}$/);
  const edit = writeJSON("studio-edit.json", {
    schema: "tei.reference.studio-edit/v0",
    id: "cp-scratch-unit-price", origin: "studio",
    intent: "Test-only selection of unit-price validation for order_entry",
    view: "table", subject: "attribute:sales.order_line.unit_price",
    operation: "add-validation", validationID, validationKind: "positive",
    applicationUse: {
      application: "order_entry", placement: "applications/order-entry/application.tei"
    }
  });
  const candidate = JSON.parse(command(process.execPath, [
    "reference/studio-authoring/cli.mjs", "propose", "-workspace", workspace,
    "-edit-file", edit
  ]));
  assert.equal(candidate.baseModelRevision, r0);
  assert.deepEqual(candidate.delta.map(x => x.op),
    ["add-validation", "add-application-validation-use"]);
  const proposal = writeJSON("candidate.json", candidate);
  const preview = acceptance(["preview", "-workspace", workspace, "-candidate", proposal]);
  assert.equal(preview.acceptable, true);
  const review = acceptance(["review", "-workspace", workspace, "-candidate", proposal,
    "-format", "json"]);
  assert.equal(review.previewAcceptable, true);
  assert.match(review.reviewFingerprint, /^sha256:[0-9a-f]{64}$/);
  assert.equal(review.candidateFingerprint, preview.candidateFingerprint,
    "review must refer to the previewed candidate");
  assert.deepEqual(changed(beforeFiles, files(workspace)), [], "preview/review wrote source");
  // A deliberate test-harness decision, NEVER a human approval.
  const decision = acceptance(["accept", "-workspace", workspace, "-candidate", proposal,
    "-expected-review-fingerprint", review.reviewFingerprint]);
  assert.equal(decision.decision, "accepted");
  assert.equal(decision.candidateFingerprint, preview.candidateFingerprint,
    "accepted decision must refer to the previewed candidate");
  assert.equal(decision.reviewBinding?.fingerprint, review.reviewFingerprint,
    "accepted decision must link to the displayed review");
  assert.equal(decision.reviewBinding?.projection?.candidateFingerprint,
    preview.candidateFingerprint,
    "accepted decision's embedded review must refer to the candidate");
  assert.equal(decision.sourceChanged, true);
  const r1 = acceptance(["revision", "-workspace", workspace]).modelRevision;
  assert.match(r1, /^sha256:[0-9a-f]{64}$/);
  assert.notEqual(r1, r0);
  assert.equal(decision.modelRevisionAfter, r1,
    "accepted decision must identify the reloaded source revision");
  assert.match(decision.derivedWorldSha256, /^sha256:[0-9a-f]{64}$/,
    "accepted decision must retain a derived World digest");
  assert.deepEqual(changed(beforeFiles, files(workspace)), [
    "applications/order-entry/application.tei", "semantic/sales/validation.tei"
  ]);
  // Both producers use workspace.Load and encoding/json. Compare actual
  // serialized bytes, not two caller-provided digest labels: this witnesses
  // the accepted Decision's derived World in a separate CLI invocation.
  const worldJSON = go(["run", "./cmd/tei", "world", "-workspace", workspace]);
  const worldDigest = "sha256:" + createHash("sha256").update(worldJSON).digest("hex");
  assert.equal(worldDigest, decision.derivedWorldSha256,
    "a separate World CLI read differs from the accepted derived World");
  const world = JSON.parse(worldJSON);
  const app = world.applications.find(a => a.id === "order_entry");
  assert.ok(app.validations.includes(validationID), "R1 Application did not select rule");
  assert.equal(app.validations.length, 4);

  const head = command("git", ["rev-parse", "HEAD"]).trim();
  assert.match(head, /^[0-9a-f]{40}$/);
  const binary = join(temp, "first-web-target");
  go(["build", "-mod=readonly", "-o", binary, "./tools/reference-first-web-target"]);
  const port = await openPort();
  const addr = "127.0.0.1:" + port;
  const serverLog = openSync(join(temp, "server.log"), "wx", 0o600);
  try {
    child = spawn(binary, [
    "-workspace", workspace, "-application", "order_entry", "-baseline-id", "B",
    "-baseline-revision", expectedR, "-target-revision", head,
    "-legacy-users", "reference/legacy-auth/users.json",
    "-legacy-authorization", "reference/legacy-auth/authorization.json",
    "-listen", addr
    ], { cwd: root, stdio: ["ignore", serverLog, serverLog] });
  } finally { closeSync(serverLog); }
  await ready("http://" + addr);
  const manifestResponse = await fetch("http://" + addr + "/manifest.json",
    { signal: AbortSignal.timeout(3000) });
  assert.equal(manifestResponse.status, 200);
  const manifest = await manifestResponse.json();
  assert.equal(manifest.applicationId, "order_entry");
  assert.equal(manifest.targetBuildRevision, head);
  assert.equal(manifest.baselineRepositoryRevision, expectedR);
  // Web projects canonical Validation IDs in sorted order, whereas the
  // accepted World keeps declaration order. Validate the full ID set without
  // turning presentation order into a false semantic mismatch.
  assert.deepEqual([...manifest.validations].sort(), [...app.validations].sort(),
    "running Web manifest did not project the accepted R1 validation selection");
  assert.deepEqual([...(manifest.semanticUses ?? [])].sort(),
    [...(app.semanticUses ?? [])].sort(),
    "running Web manifest did not project the accepted R1 Semantic uses");
  // Check Canonical Pattern identity and kind, not merely a passing command
  // response. The Web projection may add target-local presentation details.
  const patternsByID = patterns => patterns.map(({ id, kind }) => ({ id, kind }))
    .sort((a, b) => a.id.localeCompare(b.id));
  assert.deepEqual(patternsByID(manifest.patterns), patternsByID(app.patterns),
    "running Web manifest lost or changed an accepted R1 function Pattern");
  const loadedMeaning = manifestSemantics(manifest);
  const loadedMeaningSHA256 = sha256(JSON.stringify(loadedMeaning));

  const body = JSON.stringify({
    loginId: "legacy-order-editor", credential: "editor-secret",
    from: "2026-09-18T08:30:00Z", row: 0
  });
  const submitResponse = await fetch("http://" + addr + "/scenario/submit", {
    method: "POST", headers: { "content-type": "application/json" },
    body, signal: AbortSignal.timeout(5000)
  });
  assert.equal(submitResponse.status, 200);
  const raw = await submitResponse.text();
  assert.equal(raw.includes("editor-secret"), false, "Web response leaked credential");
  const receipt = JSON.parse(raw);
  assert.equal(receipt.targetBuildRevision, head);
  assert.equal(receipt.result.stage, "completed");
  assert.equal(receipt.result.validations.length, 4);
  assert.deepEqual(receipt.result.validations.map(v => v.validation),
    app.validations);
  assert.ok(receipt.result.validations.every(v => v.result === "satisfied"));
  assert.ok(receipt.result.command, "Web did not dispatch Command");
  assert.equal(receipt.result.observation?.nature, "synthetic");
  const positiveLaunch = {
    address: addr, processID: child.pid,
    binarySHA256: createHash("sha256").update(readFileSync(binary)).digest("hex")
  };
  await stopTarget();
  assert.equal(acceptance(["revision", "-workspace", workspace]).modelRevision, r1);

  // R0 stale-source control: launch the SAME Web binary with the original,
  // unaccepted Source while supplying exactly the SAME caller revision labels.
  // A manifest-based independent consumer must detect that the selected R1
  // business rule is absent. The current Web server itself does NOT reject
  // the mismatched caller labels; never advertise this as a runtime gate.
  assert.equal(acceptance(["revision", "-workspace", baseline]).modelRevision, r0,
    "baseline Source changed while running the accepted R1 scenario");
  const stalePort = await openPort();
  const staleAddr = "127.0.0.1:" + stalePort;
  const staleLog = openSync(join(temp, "stale-source-server.log"), "wx", 0o600);
  try {
    child = spawn(binary, [
      "-workspace", baseline, "-application", "order_entry", "-baseline-id", "B",
      "-baseline-revision", expectedR, "-target-revision", head,
      "-legacy-users", "reference/legacy-auth/users.json",
      "-legacy-authorization", "reference/legacy-auth/authorization.json",
      "-listen", staleAddr
    ], { cwd: root, stdio: ["ignore", staleLog, staleLog] });
  } finally { closeSync(staleLog); }
  await ready("http://" + staleAddr);
  const staleManifestResponse = await fetch("http://" + staleAddr + "/manifest.json",
    { signal: AbortSignal.timeout(3000) });
  assert.equal(staleManifestResponse.status, 200);
  const staleManifest = await staleManifestResponse.json();
  assert.equal(staleManifest.applicationId, manifest.applicationId);
  assert.equal(staleManifest.targetBuildRevision, manifest.targetBuildRevision);
  assert.equal(staleManifest.baselineRepositoryRevision, manifest.baselineRepositoryRevision);
  assert.equal(staleManifest.handoverBaselineId, manifest.handoverBaselineId);
  assert.equal(staleManifest.validations.includes(validationID), false,
    "unaccepted R0 unexpectedly already contains the R1 unit-price rule");
  assert.notDeepEqual(manifestSemantics(staleManifest), loadedMeaning,
    "stale R0 Web incorrectly appears identical to accepted R1 despite revision labels");
  const staleLaunch = {
    address: staleAddr, processID: child.pid,
    binarySHA256: sha256(readFileSync(binary)),
    manifestSemanticSHA256: sha256(JSON.stringify(manifestSemantics(staleManifest)))
  };
  assert.equal(staleLaunch.binarySHA256, positiveLaunch.binarySHA256,
    "stale Source mismatch test unexpectedly used a different Web binary");
  await stopTarget();
  assert.equal(acceptance(["revision", "-workspace", workspace]).modelRevision, r1);
  assert.equal(acceptance(["revision", "-workspace", baseline]).modelRevision, r0);

  // Do not use a quantity failure as evidence for the new unit-price rule.
  // Rebuild only the synthetic adapter using an isolated Go overlay, then run
  // the actual HTTP endpoint on a separate ephemeral loopback listener.
  const negativeBinary = join(temp, "first-web-target-negative-unit-price");
  const negativeFixture = buildNegativePriceTarget(negativeBinary);
  const negativePort = await openPort();
  const negativeAddr = "127.0.0.1:" + negativePort;
  const negativeLog = openSync(join(temp, "negative-server.log"), "wx", 0o600);
  try {
    child = spawn(negativeBinary, [
      "-workspace", workspace, "-application", "order_entry", "-baseline-id", "B",
      "-baseline-revision", expectedR, "-target-revision", head,
      "-legacy-users", "reference/legacy-auth/users.json",
      "-legacy-authorization", "reference/legacy-auth/authorization.json",
      "-listen", negativeAddr
    ], { cwd: root, stdio: ["ignore", negativeLog, negativeLog] });
  } finally { closeSync(negativeLog); }
  await ready("http://" + negativeAddr);
  const negativeManifestResponse = await fetch("http://" + negativeAddr + "/manifest.json",
    { signal: AbortSignal.timeout(3000) });
  assert.equal(negativeManifestResponse.status, 200);
  const negativeManifest = await negativeManifestResponse.json();
  assert.equal(negativeManifest.applicationId, manifest.applicationId);
  assert.equal(negativeManifest.targetBuildRevision, head);
  assert.deepEqual(manifestSemantics(negativeManifest), loadedMeaning,
    "unit-price overlay must not change the accepted Application projection");
  const negativeResponse = await fetch("http://" + negativeAddr + "/scenario/submit", {
    method: "POST", headers: { "content-type": "application/json" },
    body, signal: AbortSignal.timeout(5000)
  });
  // Business rule denial is a completed evaluation (HTTP 200), not a
  // transport or execution error. The side effect and Observation must not run.
  assert.equal(negativeResponse.status, 200);
  const negativeRaw = await negativeResponse.text();
  assert.equal(negativeRaw.includes("editor-secret"), false, "negative Web response leaked credential");
  const negativeReceipt = JSON.parse(negativeRaw);
  assert.equal(negativeReceipt.targetBuildRevision, head);
  assert.equal(negativeReceipt.result.stage, "validation-not-satisfied");
  assert.equal(negativeReceipt.executionError, undefined,
    "business validation denial must not be reported as an execution error");
  assert.deepEqual(negativeReceipt.result.validations.map(v => v.validation),
    app.validations, "the same accepted R1 validation selection must be exercised");
  assert.deepEqual(negativeReceipt.result.validations.map(v => v.result),
    app.validations.map(id => id === validationID ? "not-satisfied" : "satisfied"),
    "only the newly accepted unit-price rule should reject this order");
  assert.equal(Boolean(negativeReceipt.result.command), false, "invalid unit price reached Command");
  assert.equal(Boolean(negativeReceipt.result.observation), false, "invalid unit price reached Observation");
  const negativeLaunch = {
    address: negativeAddr, processID: child.pid,
    binarySHA256: createHash("sha256").update(readFileSync(negativeBinary)).digest("hex")
  };
  assert.notEqual(positiveLaunch.binarySHA256, negativeLaunch.binarySHA256,
    "test-only overlay unexpectedly produced the same Web binary");
  await stopTarget();
  assert.equal(acceptance(["revision", "-workspace", workspace]).modelRevision, r1);

  // N11: Command succeeds but the explicitly separate Observation read cannot
  // produce a result. This control must not be replaced by the unit-price
  // validation refusal, since that refusal never dispatches a Command.
  const n11Binary = join(temp, "first-web-target-n11-unobserved");
  const n11Fixture = buildUnobservedTarget(n11Binary);
  const n11Port = await openPort();
  const n11Addr = "127.0.0.1:" + n11Port;
  const n11Log = openSync(join(temp, "n11-server.log"), "wx", 0o600);
  try {
    child = spawn(n11Binary, [
      "-workspace", workspace, "-application", "order_entry", "-baseline-id", "B",
      "-baseline-revision", expectedR, "-target-revision", head,
      "-legacy-users", "reference/legacy-auth/users.json",
      "-legacy-authorization", "reference/legacy-auth/authorization.json",
      "-listen", n11Addr
    ], { cwd: root, stdio: ["ignore", n11Log, n11Log] });
  } finally { closeSync(n11Log); }
  await ready("http://" + n11Addr);
  const n11ManifestResponse = await fetch("http://" + n11Addr + "/manifest.json",
    { signal: AbortSignal.timeout(3000) });
  assert.equal(n11ManifestResponse.status, 200);
  const n11Manifest = await n11ManifestResponse.json();
  assert.equal(n11Manifest.targetBuildRevision, head);
  assert.deepEqual(manifestSemantics(n11Manifest), loadedMeaning,
    "N11 read-failure overlay must not alter the accepted Application projection");
  const n11Response = await fetch("http://" + n11Addr + "/scenario/submit", {
    method: "POST", headers: { "content-type": "application/json" },
    body, signal: AbortSignal.timeout(5000)
  });
  // The synthetic business Command completed, but the follow-up read failed:
  // the request must report an Execution Error (422), not business success.
  assert.equal(n11Response.status, 422);
  const n11Raw = await n11Response.text();
  assert.equal(n11Raw.includes("editor-secret"), false, "N11 Web response leaked credential");
  const n11Receipt = JSON.parse(n11Raw);
  assert.equal(n11Receipt.targetBuildRevision, head);
  assert.equal(n11Receipt.result.stage, "observation");
  assert.deepEqual(n11Receipt.result.validations.map(v => v.validation), app.validations);
  assert.ok(n11Receipt.result.validations.every(v => v.result === "satisfied"),
    "N11 must not hide a validation denial");
  assert.ok(n11Receipt.result.command, "N11 did not dispatch a successful Command");
  assert.equal(Boolean(n11Receipt.result.observation), false,
    "failed Observation was incorrectly recorded as successful");
  assert.match(n11Receipt.executionError,
    /synthetic observation unavailable \(N11 test only\)/);
  const n11Launch = {
    address: n11Addr, processID: child.pid,
    binarySHA256: createHash("sha256").update(readFileSync(n11Binary)).digest("hex")
  };
  assert.notEqual(positiveLaunch.binarySHA256, n11Launch.binarySHA256);
  assert.notEqual(negativeLaunch.binarySHA256, n11Launch.binarySHA256);
  await stopTarget();
  assert.equal(acceptance(["revision", "-workspace", workspace]).modelRevision, r1);
  assert.deepEqual(changed(beforeFiles, files(workspace)), [
    "applications/order-entry/application.tei", "semantic/sales/validation.tei"
  ]);
  const evidenceDir = publishReceipts({
    "workspace-r1.json": {
      evidenceNature: "test-only-managed-source-lineage",
      integrationHead: head, sourceBefore: r0, sourceAfter: r1,
      observedBeforeLaunch: true, observedAfterHttp: true,
      changedManagedSourceFiles: changed(beforeFiles, files(workspace)),
      changedSourceDigests: changed(beforeFiles, files(workspace)).map(path => ({
        path, beforeSHA256: beforeFiles.get(path),
        afterSHA256: files(workspace).get(path)
      })),
      candidateFingerprint: preview.candidateFingerprint,
      reviewFingerprint: review.reviewFingerprint,
      reviewCandidateFingerprint: review.candidateFingerprint,
      decisionCandidateFingerprint: decision.candidateFingerprint,
      decisionReviewFingerprint: decision.reviewBinding?.fingerprint,
      decisionModelRevisionAfter: decision.modelRevisionAfter,
      decisionDerivedWorldSHA256: decision.derivedWorldSha256,
      independentlyReloadedWorldSHA256: worldDigest,
      decision: "automated-scratch-test-only", humanApproval: false,
      expiresWithScratchWorkspace: true
    },
    "target-launch.json": {
      evidenceNature: "running-first-web-target-with-synthetic-provider",
      integrationHead: head, targetBuildRevision: head,
      baselineRepositoryRevision: expectedR, application: "order_entry",
      loopbackAddress: positiveLaunch.address, targetProcessID: positiveLaunch.processID,
      targetBinarySHA256: positiveLaunch.binarySHA256,
      // E1 preflight has one launch-receipt locator: explicitly record all
      // three distinct synthetic test binaries rather than claiming parity.
      negativeTargetProcessID: negativeLaunch.processID,
      negativeTargetLoopbackAddress: negativeLaunch.address,
      negativeTargetBinarySHA256: negativeLaunch.binarySHA256,
      negativeTargetFixtureOverlaySHA256: negativeFixture.overlaySHA256,
      negativeUsesSameBinary: false,
      n11TargetProcessID: n11Launch.processID,
      n11TargetLoopbackAddress: n11Launch.address,
      n11TargetBinarySHA256: n11Launch.binarySHA256,
      n11TargetOverlaySHA256: n11Fixture.overlaySHA256,
      n11UsesSameBinary: false,
      staleTargetProcessID: staleLaunch.processID,
      staleTargetLoopbackAddress: staleLaunch.address,
      staleTargetBinarySHA256: staleLaunch.binarySHA256,
      staleUsesSameBinaryAsPositive: true,
      staleManifestSemanticSHA256: staleLaunch.manifestSemanticSHA256,
      workspaceSourceRevisionBeforeLaunch: r1,
      observedManifestSemanticSHA256: loadedMeaningSHA256,
      projectionMatchesAcceptedWorld: true,
      healthResponseWasOK: true, runtimeAuthorityVerified: false
    },
    "web-positive.json": {
      evidenceNature: "running-web-http-positive-synthetic-provider",
      integrationHead: head, acceptedSourceRevisionClaim: r1,
      selectedValidationIDs: receipt.result.validations.map(v => v.validation),
      validationResults: receipt.result.validations.map(v => v.result),
      stage: receipt.result.stage,
      httpStatus: submitResponse.status,
      commandObserved: Boolean(receipt.result.command),
      observationNature: receipt.result.observation.nature,
      manifestSemanticSHA256: loadedMeaningSHA256,
      httpResponseSHA256: sha256(raw),
      independentDatabaseRead: false,
      negativeUnitPriceTest: "RUNNING_SERVER_SYNTHETIC_FIXTURE_OVERLAY"
    },
    "target-negative-launch.json": {
      evidenceNature: "test-only-Go-overlay-Web-target",
      integrationHead: head, targetBuildRevisionLabel: head,
      loopbackAddress: negativeLaunch.address, targetProcessID: negativeLaunch.processID,
      targetBinarySHA256: negativeLaunch.binarySHA256,
      sameBinaryAsPositive: false, acceptedSourceRevisionClaim: r1,
      overlay: negativeFixture, runtimeAuthorityVerified: false
    },
    "web-negative-unit-price.json": {
      evidenceNature: "running-web-http-negative-synthetic-fixture-overlay",
      integrationHead: head, acceptedSourceRevisionClaim: r1,
      selectedValidation: validationID, evaluation: "not-satisfied",
      stage: negativeReceipt.result.stage, httpStatus: negativeResponse.status,
      commandDispatched: Boolean(negativeReceipt.result.command),
      observationExecuted: Boolean(negativeReceipt.result.observation),
      targetBinarySHA256: negativeLaunch.binarySHA256,
      manifestSemanticSHA256: loadedMeaningSHA256,
      httpResponseSHA256: sha256(negativeRaw),
      productionTargetUnmodified: true, independentDatabaseRead: false
    },
    "target-n11-launch.json": {
      evidenceNature: "test-only-Go-overlay-Web-target-N11",
      integrationHead: head, targetBuildRevisionLabel: head,
      loopbackAddress: n11Launch.address, targetProcessID: n11Launch.processID,
      targetBinarySHA256: n11Launch.binarySHA256,
      acceptedSourceRevisionClaim: r1,
      overlay: n11Fixture, runtimeAuthorityVerified: false
    },
    "n11-unobserved-control.json": {
      evidenceNature: "running-web-http-synthetic-Observation-read-failure",
      integrationHead: head, acceptedSourceRevisionClaim: r1,
      httpStatus: n11Response.status, stage: n11Receipt.result.stage,
      commandCompleted: Boolean(n11Receipt.result.command),
      observationReadAttempted: true,
      observationResultRecorded: Boolean(n11Receipt.result.observation),
      executionErrorNature: "synthetic-read-unavailable",
      targetBinarySHA256: n11Launch.binarySHA256,
      manifestSemanticSHA256: loadedMeaningSHA256,
      httpResponseSHA256: sha256(n11Raw),
      productionTargetUnmodified: true, independentDatabaseRead: false
    },
    "stale-source-control.json": {
      evidenceNature: "running-web-http-stale-source-R0-label-spoof-control",
      integrationHead: head, acceptedSourceRevisionClaim: r1,
      actuallyLoadedWorkspaceSourceRevision: r0,
      suppliedBaselineRevisionLabel: expectedR,
      suppliedTargetBuildRevisionLabel: head,
      applicationId: staleManifest.applicationId,
      staleManifestSemanticSHA256: staleLaunch.manifestSemanticSHA256,
      acceptedManifestSemanticSHA256: loadedMeaningSHA256,
      sameTargetBinaryAsPositive: true,
      targetBinarySHA256: staleLaunch.binarySHA256,
      acceptedR1RuleMissing: !staleManifest.validations.includes(validationID),
      mismatchDetectedByExternalTestHarness: true,
      webServerRejectsMismatchedLabels: false,
      currentRuntimeVersionBindingEnforced: false,
      independentDatabaseRead: false
    }
  });
  const evidenceFiles = evidenceDir ? [
    "workspace-r1.json", "target-launch.json", "web-positive.json",
    "target-negative-launch.json", "web-negative-unit-price.json",
    "target-n11-launch.json", "n11-unobserved-control.json",
    "stale-source-control.json"
  ] : [];
  // CI currently retains job logs, but not these temporary files as artifacts.
  // Publishing each file's exact checksum to stderr leaves a bounded
  // human-reviewable witness without disclosing synthetic HTTP response bodies.
  const evidenceSHA256 = Object.fromEntries(evidenceFiles.map(name =>
    [name, sha256(readFileSync(join(evidenceDir, name)))]));
  process.stderr.write("TEI #1163 synthetic witness: " + JSON.stringify({
    integrationHead: head, acceptedScratchSourceR1: r1,
    manifestSemanticSHA256: loadedMeaningSHA256,
    allThreeWebManifestsMatch: true,
    staleSourceControl: {
      actualLoadedRevision: r0,
      acceptedRevisionClaim: r1,
      sameTargetBinary: true, callerRevisionLabelsIdentical: true,
      staleManifestSemanticSHA256: staleLaunch.manifestSemanticSHA256,
      semanticMismatchDetected: true, targetRejectedStaleLabels: false
    },
    positiveWeb: { httpStatus: submitResponse.status,
      commandCompleted: Boolean(receipt.result.command),
      observationNature: receipt.result.observation.nature },
    negativeUnitPrice: { httpStatus: negativeResponse.status,
      selectedValidation: validationID, stage: negativeReceipt.result.stage,
      commandDispatched: Boolean(negativeReceipt.result.command) },
    n11: { httpStatus: n11Response.status,
      stage: n11Receipt.result.stage,
      commandCompleted: Boolean(n11Receipt.result.command),
      observationResultRecorded: Boolean(n11Receipt.result.observation),
      targetBinarySHA256: n11Launch.binarySHA256 },
    evidenceSHA256, evidenceBodiesArchivedByThisScript: false,
    independentDatabaseRead: false, productGate: "UNQUALIFIED"
  }) + "\n");
  process.stdout.write(JSON.stringify({
    status: "E1_A_B_LIVE_HTTP_SYNTHETIC_PROVIDER_ONLY", // #1119 workflow-compatible; N11 in dedicated fields
    authority: "test-harness-acceptance-not-human",
    repositoryIntegrationHead: head, baselineRepositoryRevision: expectedR,
    sourceR0: r0, acceptedScratchSourceR1: r1, exactValidation: validationID,
    validationCount: receipt.result.validations.length,
    httpStatus: submitResponse.status, commandObserved: true,
    independentDatabaseRead: false,
    negativeUnitPriceRunningWeb: "TESTED_WITH_SYNTHETIC_OVERLAY_BINARY",
    negativeUnitPriceRejected: negativeReceipt.result.stage === "validation-not-satisfied",
    n11UnobservedReadControl: "TESTED_WITH_SYNTHETIC_OBSERVATION_FAILURE_OVERLAY",
    n11ReportsExecutionError: n11Response.status === 422 &&
      Boolean(n11Receipt.result.command) && !n11Receipt.result.observation,
    staleSourceMismatchDetectedByHarness: true,
    staleSourceRejectedByTarget: false,
    manifestSemanticSHA256: loadedMeaningSHA256,
    threeWebManifestsMatchAcceptedApplication: true,
    n11HttpStatus: n11Response.status,
    n11CommandCompleted: Boolean(n11Receipt.result.command),
    n11ObservationResultRecorded: Boolean(n11Receipt.result.observation),
    evidenceDirectory: evidenceDir,
    evidenceFiles, evidenceSHA256,
    limits: ["temporary-scratch", "synthetic-provider", "three-distinct-test-binaries",
      "n11-synthetic-read-unavailable", "no-real-db", "not-human-approved",
      "not-E1-complete", "not-product-gate"]
  }, null, 2) + "\n");
}
try {
  await main();
} finally {
  await stopTarget();
  rmSync(temp, { recursive: true, force: true });
}
