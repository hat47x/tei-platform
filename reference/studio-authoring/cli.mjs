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

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const usage = [
  "usage:",
  "  node reference/studio-authoring/cli.mjs inspect -workspace DIR -focus relation:ID|step:ID [-business-dfd FILE]",
  "  node reference/studio-authoring/cli.mjs propose -workspace DIR -edit-file FILE",
  "  node reference/studio-authoring/cli.mjs review-edit -workspace DIR -edit-file FILE",
  "  alternatively: -world-file FILE -revision-file FILE [-source-index-file FILE] (read-only artifacts)",
  "  inspect and review-edit may use -format html for a static read-only page",
  "  output defaults to JSON; no command in this CLI writes authoritative source"
].join("\n");

function flags(argv) {
  const result = {};
  for (let i = 0; i < argv.length; i += 2) {
    if (!argv[i].startsWith("-") || !argv[i + 1] || ![
      "-workspace", "-world-file", "-revision-file", "-focus", "-edit-file", "-business-dfd", "-source-index-file", "-format"
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
  if (!["inspect", "propose", "review-edit"].includes(command)) throw new Error("UNKNOWN_COMMAND");
  const opt = flags(rest);
  const fromFiles = !!opt["-world-file"] && !!opt["-revision-file"];
  const format = opt["-format"] || "json";
  if (!["json", "html"].includes(format) || (command === "propose" && opt["-format"]))
    throw new Error("UNSUPPORTED_STUDIO_OUTPUT_FORMAT");
  if (!!opt["-workspace"] === fromFiles ||
      (!!opt["-workspace"] && (opt["-world-file"] || opt["-revision-file"] ||
        opt["-source-index-file"])) ||
      (command === "inspect" && (!opt["-focus"] || opt["-edit-file"])) ||
      (["propose", "review-edit"].includes(command) && (!opt["-edit-file"] || opt["-focus"] ||
        opt["-business-dfd"] || opt["-source-index-file"])) ||
      (command === "review-edit" && (!opt["-workspace"] || fromFiles)))
    throw new Error("INVALID_INPUT");
  const { world, revision, sourceIndex } = fromFiles
    ? { world: jsonFile(opt["-world-file"]),
        revision: jsonFile(opt["-revision-file"]).modelRevision,
        sourceIndex: opt["-source-index-file"]
          ? jsonFile(opt["-source-index-file"]) : undefined }
    : fromWorkspace(opt["-workspace"], command === "inspect");
  if (command === "inspect") {
    const focus = opt["-focus"];
    if (focus.startsWith("step:")) {
      if (!opt["-business-dfd"]) throw new Error("BUSINESS_STEP_REQUIRES_REFERENCE_DFD");
      const view = navigateReferenceBusinessStep(world, revision,
        jsonFile(opt["-business-dfd"]), focus.slice("step:".length));
      return format === "html" ? renderStudioHTML(view) : view;
    }
    const view = projectWorld(world, revision, focus,
      { ...(opt["-business-dfd"] ? { businessDFD: jsonFile(opt["-business-dfd"]) } : {}),
        ...(sourceIndex ? { sourceIndex } : {}) });
    const verified = fromFiles ? view : verifyLiveSourceNavigation(view, opt["-workspace"], revision);
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
