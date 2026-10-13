// Business DFD is a separately produced, noncanonical reference artifact.
// Its file bytes can be observed, but this does not bind it to the World
// revision, to an accepted Procedure, or to an authorized editor.
import { readFileSync, lstatSync, realpathSync } from "node:fs";
import { createHash } from "node:crypto";

export function observeBusinessDFD(path) {
  const stat = lstatSync(path);
  if (!stat.isFile() || stat.isSymbolicLink())
    throw new Error("INVALID_BUSINESS_DFD_ARTIFACT_TYPE");
  const bytes = readFileSync(path);
  const digest = "sha256:" + createHash("sha256").update(bytes).digest("hex");
  let value;
  try { value = JSON.parse(bytes.toString("utf8")); }
  catch { throw new Error("INVALID_BUSINESS_DFD_JSON"); }
  if (!value || value.schemaVersion !== "business-dfd-projection-v0" ||
      value.authority !== "derived-read-only" ||
      value.canonicalProcedureUsed !== false)
    throw new Error("UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA");
  return { value, evidence: {
    schema: "tei.reference.studio-business-observation/v0",
    evidenceClass: "external-reference-file-observed",
    artifactFingerprint: digest,
    artifactBytes: bytes.length,
    canonicalProcedureUsed: false,
    acceptedBusinessProcedure: false,
    sourceRevisionBinding: "not-established",
    authority: "noncanonical-reference",
    // Path is evidence of the observed artifact, not a Source ownership key.
    observedPath: realpathSync(path)
  } };
}
export function verifyBusinessDFDObservation(path, observed) {
  const current = observeBusinessDFD(path);
  if (current.evidence.artifactFingerprint !== observed.evidence.artifactFingerprint ||
      current.evidence.artifactBytes !== observed.evidence.artifactBytes)
    throw new Error("BUSINESS_DFD_CHANGED_DURING_INSPECT");
  return observed.evidence;
}
