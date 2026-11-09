// Studio consumes the existing Go #1125 DFD producer, never its authoring
// authority. This is a read-only, separately reloaded Workspace observation.
import { lstatSync, readFileSync, realpathSync } from "node:fs";
import { execFileSync, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { resolve, relative, isAbsolute, sep } from "node:path";

const REVISION = /^sha256:[0-9a-f]{64}$/;
const roles = ["procedure", "review", "set"];
function fail(code) { throw new Error(code); }
function snapshot(role, path) {
  const absolute = resolve(path);
  const stat = lstatSync(absolute);
  if (!stat.isFile() || stat.isSymbolicLink())
    fail("BUSINESS_DFD_INVALID_INPUT_FILE_TYPE");
  const bytes = readFileSync(absolute);
  return { role, path: realpathSync(absolute),
    fingerprint: "sha256:" + createHash("sha256").update(bytes).digest("hex"),
    byteLength: bytes.length };
}
function same(a, b) {
  return a.role === b.role && a.path === b.path &&
    a.fingerprint === b.fingerprint && a.byteLength === b.byteLength;
}

// Source Index artifact membership is physical Source-set evidence only. An
// indexed JSON file does not prove that Business meaning was normalized or
// accepted by Shared Acceptance. Never infer a semantic Procedure identity.
function qualifySourceMembership(workspace, sourceIndex, snapshots) {
  if (sourceIndex === undefined) return snapshots.map(item => ({
    ...item, sourceMembership: "not-checked-no-source-index" }));
  if (!sourceIndex || !Array.isArray(sourceIndex.artifacts))
    fail("BUSINESS_DFD_SOURCE_INDEX_INVALID");
  const artifacts = new Map();
  for (const a of sourceIndex.artifacts) {
    if (!a || typeof a.path !== "string" || !a.path ||
        a.path.startsWith("/") || a.path.includes("\\") ||
        a.path.split("/").some(part => !part || part === "." || part === "..") ||
        /^[A-Za-z]:/.test(a.path) || !REVISION.test(a.fingerprint) ||
        artifacts.has(a.path)) fail("BUSINESS_DFD_SOURCE_INDEX_INVALID");
    artifacts.set(a.path, a.fingerprint);
  }
  const workspaceReal = realpathSync(workspace);
  return snapshots.map(item => {
    const rel = relative(workspaceReal, item.path);
    if (!rel || isAbsolute(rel) || rel === ".." || rel.startsWith(".." + sep))
      return { ...item, sourceMembership: "external-reference-not-managed" };
    const indexedPath = rel.split(sep).join("/");
    const expected = artifacts.get(indexedPath);
    if (!expected) return { ...item,
      sourceMembership: "external-reference-not-managed" };
    if (expected !== item.fingerprint)
      fail("BUSINESS_DFD_MANAGED_SOURCE_INDEX_FINGERPRINT_STALE");
    return { ...item, sourceMembership: "managed-source-artifact-verified",
      managedArtifactPath: indexedPath };
  });
}

// Caller must supply an actual live Workspace and the revision of a World
// independently loaded from it. Read-before/read-after guards do not create
// an atomic transaction or an Accepted Business Procedure.
export function produceBusinessDFD({
  workspace, modelRevision, procedure, review, set, repoRoot,
  readRevision, runProducer, sourceIndex
}) {
  if (!REVISION.test(modelRevision) || typeof workspace !== "string" ||
      !workspace.trim() || typeof repoRoot !== "string" || !repoRoot.trim() ||
      roles.some(role => typeof ({ procedure, review, set })[role] !== "string" ||
        !({ procedure, review, set })[role].trim()))
    fail("BUSINESS_DFD_INVALID_PRODUCER_INPUTS");
  const files = { procedure: resolve(procedure), review: resolve(review),
    set: resolve(set) }, workspaceRoot = resolve(workspace);
  const revision = readRevision || (() =>
    JSON.parse(execFileSync("go", ["run", "./tools/reference-shared-acceptance",
      "revision", "-workspace", workspaceRoot],
    { cwd: repoRoot, encoding: "utf8", maxBuffer: 16 * 1024 * 1024 })).modelRevision);
  const before = roles.map(role => snapshot(role, files[role]));
  if (revision() !== modelRevision) fail("BUSINESS_DFD_WORKSPACE_REVISION_CHANGED");
  const run = runProducer || (() => spawnSync("go", [
    "run", "./tools/reference-business-dfd",
    "-workspace", workspaceRoot, "-procedure", files.procedure,
    "-review", files.review, "-set", files.set, "-format", "json"
  ], { cwd: repoRoot, encoding: "utf8", maxBuffer: 32 * 1024 * 1024 }));
  const output = run({ workspace: workspaceRoot, files });
  if (output.error || output.status !== 0)
    fail("BUSINESS_DFD_GO_PRODUCER_FAILED");
  let value;
  try { value = JSON.parse(output.stdout); }
  catch { fail("BUSINESS_DFD_PRODUCER_INVALID_JSON"); }
  if (!value || value.schemaVersion !== "business-dfd-projection-v0" ||
      value.authority !== "derived-read-only" ||
      value.canonicalProcedureUsed !== false)
    fail("BUSINESS_DFD_PRODUCER_AUTHORITY_INVALID");
  const after = roles.map(role => snapshot(role, files[role]));
  if (!before.every((item, i) => same(item, after[i])))
    fail("BUSINESS_DFD_INPUT_CHANGED_DURING_PROJECTION");
  // The Source Index must come from the same caller-observed Workspace read.
  // A matching physical artifact fingerprint is NOT an Accepted Business
  // source schema, logical scope, human approval or Source write authority.
  const sources = qualifySourceMembership(workspaceRoot, sourceIndex, before);
  if (revision() !== modelRevision)
    fail("BUSINESS_DFD_WORKSPACE_REVISION_CHANGED");
  return { value, evidence: {
    schema: "tei.reference.studio-business-producer-observation/v0",
    evidenceClass: "go-derived-reference-observed",
    producer: "tools/reference-business-dfd",
    workspaceRevisionObserved: modelRevision,
    workspaceRevisionStatus: "checked-around-separate-process-not-atomic",
    sourceRevisionBinding: "not-established",
    acceptedBusinessProcedure: false, canonicalProcedureUsed: false,
    authority: "noncanonical-reference",
    sourceSetStatus: sourceIndex === undefined ? "not-checked-no-source-index" :
      sources.every(x => x.sourceMembership === "managed-source-artifact-verified")
        ? "all-input-artifacts-managed" : "contains-nonmanaged-reference-inputs",
    sources
  } };
}
