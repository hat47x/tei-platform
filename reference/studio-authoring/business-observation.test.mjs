import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync, symlinkSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { observeBusinessDFD, verifyBusinessDFDObservation } from "./business-observation.mjs";

test("external Business DFD observation fingerprints exact bytes but never grants Source authority", () => {
  const dir = mkdtempSync(join(tmpdir(), "tei-studio-dfd-evidence-"));
  try {
    const path = join(dir, "business.json");
    const dfd = { schemaVersion: "business-dfd-projection-v0",
      authority: "derived-read-only", canonicalProcedureUsed: false,
      procedureId: "reference-local", diagnostics: [], steps: [] };
    writeFileSync(path, JSON.stringify(dfd));
    const before = observeBusinessDFD(path);
    assert.equal(before.evidence.acceptedBusinessProcedure, false);
    assert.equal(before.evidence.sourceRevisionBinding, "not-established");
    assert.match(before.evidence.artifactFingerprint, /^sha256:[0-9a-f]{64}$/);
    assert.deepEqual(verifyBusinessDFDObservation(path, before), before.evidence);
    writeFileSync(path, JSON.stringify(dfd) + "\n");
    assert.throws(() => verifyBusinessDFDObservation(path, before),
      /BUSINESS_DFD_CHANGED_DURING_INSPECT/);
    const changed = observeBusinessDFD(path);
    assert.notEqual(changed.evidence.artifactFingerprint, before.evidence.artifactFingerprint);
    writeFileSync(path, JSON.stringify({ ...dfd, canonicalProcedureUsed: true }));
    assert.throws(() => observeBusinessDFD(path),
      /UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA/);
    writeFileSync(path, "{bad json");
    assert.throws(() => observeBusinessDFD(path), /INVALID_BUSINESS_DFD_JSON/);
    writeFileSync(path, JSON.stringify(dfd));
    const link = join(dir, "alias.json");
    symlinkSync(path, link);
    assert.throws(() => observeBusinessDFD(link), /INVALID_BUSINESS_DFD_ARTIFACT_TYPE/);
  } finally { rmSync(dir, { recursive: true, force: true }); }
});
