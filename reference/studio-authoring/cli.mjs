#!/usr/bin/env node
// Read-only Reference CLI: accepted World -> scoped projections / proposals.
// Use the existing reference-shared-acceptance for preview, review and accept.
import { readFileSync, mkdtempSync, writeFileSync, rmSync, lstatSync, realpathSync } from "node:fs";
import { tmpdir } from "node:os";
import { createHash } from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve, relative, isAbsolute, sep } from "node:path";
import { projectWorld, candidateFromEdit, bindReviewArtifacts, navigateReferenceBusinessStep, verifySourceAnchor } from "./bridge.mjs";
import { renderStudioHTML } from "./html.mjs";
import { renderStudioEditHTML } from "./editor.mjs";
import { bindTypedTrace } from "./typed-trace.mjs";
import { observeBusinessDFD, verifyBusinessDFDObservation } from "./business-observation.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const usage = [
  "usage:",
  "  node reference/studio-authoring/cli.mjs inspect -workspace DIR -focus relation:ID|step:ID [-business-dfd FILE]",
  "  node reference/studio-authoring/cli.mjs propose -workspace DIR -edit-file FILE",
  "  node reference/studio-authoring/cli.mjs review-edit -workspace DIR -edit-file FILE",
  "  node reference/studio-authoring/cli.mjs edit-form -workspace DIR -focus data:ID|attribute:ID",
  "  alternatively: -world-file FILE -revision-file FILE [-source-index-file FILE] (read-only artifacts)",
  "  inspect accepts -typed-scope OBSERVATION_SCOPE with -workspace to bind read-only Go #1120 trace",
  "  inspect and review-edit may use -format html for a static read-only page",
  "  output defaults to JSON; no command in this CLI writes authoritative source"
].join("\n");

function flags(argv) {
  const result = {};
  for (let i = 0; i < argv.length; i += 2) {
    if (!argv[i].startsWith("-") || !argv[i + 1] || ![
      "-workspace", "-world-file", "-revision-file", "-focus", "-edit-file", "-business-dfd", "-source-index-file", "-format", "-typed-scope"
    ].includes(argv[i]) || Object.hasOwn(result, argv[i])) {
      throw new Error("INVALID_FLAG: " + (argv[i] || ""));
    }
    result[argv[i]] = argv[i + 1];
  }
  return result;
}
function jsonFile(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}
function fromWorkspace(workspace, withSourceIndex) {
  // Resolve relative workspaces before switching the child process to repo root.
  workspace = resolve(workspace);
  // The two reads are not atomic. The Shared Acceptance Preview and Accept
  // must re-check the base revision and fail closed on concurrent edits.
  const run = (...args) => execFileSync("go", ["run", ...args], {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024
  });
  const getRevision = () => JSON.parse(run("./tools/reference-shared-acceptance",
    "revision", "-workspace", workspace)).modelRevision;
  const before = getRevision();
  const world = JSON.parse(run("./cmd/tei", "world", "-workspace", workspace));
  const sourceIndex = withSourceIndex
    ? JSON.parse(run("./cmd/tei", "source-index", "-workspace", workspace))
    : undefined;
  const after = getRevision();
  if (before !== after) throw new Error("SOURCE_CHANGED_DURING_PROJECTION");
  return { world, revision: after, sourceIndex };
}
// Only live Workspace inspection can verify bytes. Standalone World/Index
// snapshots remain informational; do not silently label them as verified.
function verifyLiveSourceNavigation(view, workspace, revision) {
  const location = view.sourceNavigation;
  if (location?.status !== "source-index-backed") return view;
  const rootReal = realpathSync(resolve(workspace));
  const parts = location.artifactPath.split("/");
  let path = rootReal;
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    if (!part || part === "." || part === "..")
      throw new Error("INVALID_SOURCE_ARTIFACT_PATH");
    path = join(path, part);
    const stat = lstatSync(path);
    if (stat.isSymbolicLink())
      throw new Error("SOURCE_ARTIFACT_SYMLINK_NOT_ALLOWED");
    if (i < parts.length - 1 ? !stat.isDirectory() : !stat.isFile())
      throw new Error("SOURCE_ARTIFACT_NOT_REGULAR_FILE");
  }
  const physical = realpathSync(path);
  const rel = relative(rootReal, physical);
  if (!rel || rel === ".." || rel.startsWith(".." + sep) || isAbsolute(rel))
    throw new Error("SOURCE_ARTIFACT_OUTSIDE_WORKSPACE");
  const bytes = readFileSync(physical);
  view.sourceNavigation = verifySourceAnchor(location, bytes, actualBytes =>
    "sha256:" + createHash("sha256").update(actualBytes).digest("hex"));
  // The Workspace may change between individual subprocesses. This guard
  // catches a changed Source revision; it is not a lock on concurrent writers.
  const latest = JSON.parse(execFileSync("go", ["run",
    "./tools/reference-shared-acceptance", "revision",
    "-workspace", rootReal], {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024
  })).modelRevision;
  if (latest !== revision) throw new Error("SOURCE_CHANGED_DURING_ANCHOR_VERIFICATION");
  return view;
}

function main(argv) {
  const [command, ...rest] = argv;
  if (!["inspect", "propose", "review-edit", "edit-form"].includes(command)) throw new Error("UNKNOWN_COMMAND");
  const opt = flags(rest);
  const fromFiles = !!opt["-world-file"] && !!opt["-revision-file"];
  const format = opt["-format"] || (command === "edit-form" ? "html" : "json");
  if (!["json", "html"].includes(format) || (command === "propose" && opt["-format"]) ||
      (command === "edit-form" && format !== "html"))
    throw new Error("UNSUPPORTED_STUDIO_OUTPUT_FORMAT");
  if ((opt["-typed-scope"] && (command !== "inspect" || fromFiles ||
        opt["-focus"]?.startsWith("step:"))) ||
      !!opt["-workspace"] === fromFiles ||
      (!!opt["-workspace"] && (opt["-world-file"] || opt["-revision-file"] ||
        opt["-source-index-file"])) ||
      (command === "inspect" && (!opt["-focus"] || opt["-edit-file"])) ||
      (["propose", "review-edit"].includes(command) && (!opt["-edit-file"] || opt["-focus"] ||
        opt["-business-dfd"] || opt["-source-index-file"])) ||
      (command === "review-edit" && (!opt["-workspace"] || fromFiles)) ||
      (command === "edit-form" && (!opt["-workspace"] || !opt["-focus"] ||
        opt["-edit-file"] || opt["-business-dfd"] || opt["-source-index-file"] ||
        opt["-world-file"] || opt["-revision-file"])))
    throw new Error("INVALID_INPUT");
  const { world, revision, sourceIndex } = fromFiles
    ? { world: jsonFile(opt["-world-file"]),
        revision: jsonFile(opt["-revision-file"]).modelRevision,
        sourceIndex: opt["-source-index-file"]
          ? jsonFile(opt["-source-index-file"]) : undefined }
    : fromWorkspace(opt["-workspace"], command === "inspect");
  if (command === "edit-form")
    return renderStudioEditHTML(world, revision, opt["-focus"]);
  if (command === "inspect") {
    const focus = opt["-focus"];
    const business = opt["-business-dfd"] ? observeBusinessDFD(opt["-business-dfd"]) : null;
    if (focus.startsWith("step:")) {
      if (!opt["-business-dfd"]) throw new Error("BUSINESS_STEP_REQUIRES_REFERENCE_DFD");
      const view = navigateReferenceBusinessStep(world, revision,
        business.value, focus.slice("step:".length));
      view.businessObservation = verifyBusinessDFDObservation(opt["-business-dfd"], business);
      return format === "html" ? renderStudioHTML(view) : view;
    }
    const view = projectWorld(world, revision, focus,
      { ...(business ? { businessDFD: business.value } : {}),
        ...(sourceIndex ? { sourceIndex } : {}) });
    const verified = fromFiles ? view : verifyLiveSourceNavigation(view, opt["-workspace"], revision);
    if (business) verified.businessObservation = verifyBusinessDFDObservation(opt["-business-dfd"], business);
    if (opt["-typed-scope"]) {
      // This scope names a read-only observation of the existing #1120
      // derived graph; it does NOT assert Source ownership or Business identity.
      const scope = opt["-typed-scope"];
      if (scope !== scope.trim() || !scope)
        throw new Error("INVALID_TYPED_TRACE_SCOPE");
      const colon = focus.indexOf(":");
      const kinds = { data: "semantic-data", relation: "semantic-relation",
        attribute: "semantic-attribute", validation: "semantic-validation",
        pattern: "application-pattern" };
      const kind = kinds[focus.slice(0, colon)];
      if (!kind) throw new Error("UNSUPPORTED_TYPED_STUDIO_FOCUS");
      const traced = spawnSync("go", ["run", "./cmd/tei", "trace",
        "-workspace", resolve(opt["-workspace"]), "-scope", scope,
        "-subject-kind", kind, "-subject-id", focus.slice(colon + 1),
        "-direction", "both", "-mode", "full"],
      { cwd: root, encoding: "utf8", maxBuffer: 32 * 1024 * 1024 });
      if (traced.error) throw traced.error;
      let report;
      try { report = JSON.parse(traced.stdout); }
      catch { throw new Error("GO_TYPED_TRACE_NO_JSON: " + traced.stderr); }
      if (traced.status !== 0)
        throw new Error("GO_TYPED_TRACE_FAILED: " +
          (report.error?.code || traced.stderr || String(traced.status)));
      verified.typedReferences = bindTypedTrace(verified, report, scope);
      // Go trace reloads the Workspace separately. This double revision
      // check limits mixed snapshots but does not lock concurrent writers.
      const current = JSON.parse(execFileSync("go", ["run",
        "./tools/reference-shared-acceptance", "revision",
        "-workspace", resolve(opt["-workspace"])],
      { cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024 })).modelRevision;
      if (current !== revision)
        throw new Error("SOURCE_CHANGED_DURING_TYPED_TRACE");
    }
    return format === "html" ? renderStudioHTML(verified,
      { sourceFromWorkspace: !fromFiles }) : verified;
  }
  const candidate = candidateFromEdit(world, revision, jsonFile(opt["-edit-file"]));
  if (command === "propose") return candidate;
  // The Review operation delegates entirely to Shared Acceptance; it does
  // not grant approval and never invokes accept/reject or writes a Source.
  const workspace = resolve(opt["-workspace"]);
  const temporary = mkdtempSync(join(tmpdir(), "tei-studio-review-"));
  try {
    const candidatePath = join(temporary, "candidate.json");
    writeFileSync(candidatePath, JSON.stringify(candidate) + "\n", { mode: 0o600 });
    const run = (cmd, extra = []) => {
      const result = spawnSync("go", ["run", "./tools/reference-shared-acceptance",
        cmd, "-workspace", workspace, "-candidate", candidatePath, ...extra],
        { cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024 });
      if (result.error) throw result.error;
      // Preview and Review return status 1 with JSON when not acceptable.
      // They are still valid diagnostics, not permission to Accept.
      if (![0, 1].includes(result.status))
        throw new Error("SHARED_ACCEPTANCE_FAILED: " + result.stderr);
      try { return JSON.parse(result.stdout); }
      catch { throw new Error("SHARED_ACCEPTANCE_NO_JSON: " + result.stderr); }
    };
    const preview = run("preview");
    const review = run("review", ["-format", "json"]);
    const binding = bindReviewArtifacts(candidate, preview, review, revision);
    const latest = JSON.parse(execFileSync("go", ["run",
      "./tools/reference-shared-acceptance", "revision", "-workspace", workspace],
      { cwd: root, encoding: "utf8" })).modelRevision;
    if (latest !== revision) throw new Error("SOURCE_CHANGED_DURING_REVIEW");
    const result = { schema: "tei.reference.studio-review-edit/v0",
      modelRevision: revision, sourceRevisionAfterReview: latest,
      sourceWriteByThisCommand: false, explicitAcceptance: "not-performed",
      candidate, preview, review, reviewConsistency: binding,
      realizationReadiness: { status: "not-verified" } };
    return format === "html" ? renderStudioHTML(result) : result;
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
}
try {
  const result = main(process.argv.slice(2));
  process.stdout.write(typeof result === "string" ? result : JSON.stringify(result, null, 2) + "\n");
} catch (err) {
  process.stderr.write(String(err.message) + "\n" + usage + "\n");
  process.exitCode = 1;
}
