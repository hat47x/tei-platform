import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { readSourceLocations, previewRequiredEditDraft } from "./source-locations.ts";

const subject = { kind: "semantic-data", id: "sales.order" };
const catalog = {
  entries: [{ subject, sourceStatus: "indexed", ambiguous: false }],
  modelRevision: "model-A",
  worldFingerprint: "world-A",
  sourceIndexFingerprint: "source-A",
};
const authorityLimits = [
  "inspection-is-a-derived-read-only-projection",
  "source-location-is-not-semantic-acceptance",
  "source-location-is-not-write-authority",
  "local-read-is-not-disclosure-permission",
];
const fingerprint = "sha256:" + "a".repeat(64);
const artifact = { path: "orders/注文.tei", format: "tei", fingerprint };
const location = (i, role = "definition", path = artifact.path) => ({
  subject, artifactPath: path, role,
  range: {
    start: { offset: i * 3, line: i + 1, column: 1 },
    end: { offset: i * 3 + 3, line: i + 1, column: 4 },
  },
});
function inspection(references = [location(0), location(1, "implementation")]) {
  return {
    schemaVersion: "tei.operations.inspect/v0",
    effect: "read-only",
    subject,
    canonical: { id: subject.id },
    sourceStatus: "indexed",
    worldFingerprint: "world-A",
    sourceIndexFingerprint: "source-A",
    authorityLimits,
    sources: { subject, artifacts: [artifact], references },
  };
}

test("S-LOC retains all observed roles, relative paths, byte offsets and fingerprints", () => {
  const view = readSourceLocations(inspection(), subject, catalog);
  assert.deepEqual(
    [view.observedReferences, view.shownReferences.length, view.omittedReferences,
      view.completenessEstablished],
    [2, 2, 0, false],
  );
  assert.equal(view.shownReferences[0].workspaceRelativePath, "orders/注文.tei");
  assert.equal(view.shownReferences[0].artifactFingerprint, fingerprint);
  assert.equal(view.shownReferences[0].start.byteOffset, 0);
  assert.equal(view.shownReferences[1].role, "implementation");
  assert.equal(view.shownReferences[1].end.byteOffset, 6);
  // A byte offset is NOT CodeMirror's UTF-16 column or a position in display text.
  assert.equal(view.shownReferences[1].start.line, 2);
});

test("S-LOC caps displayed references without asserting Source Index completeness", () => {
  const view = readSourceLocations(
    inspection(Array.from({ length: 23 }, (_, i) => location(i))), subject, catalog,
  );
  assert.equal(view.observedReferences, 23);
  assert.equal(view.shownReferences.length, 20);
  assert.equal(view.omittedReferences, 3);
  assert.equal(view.completenessEstablished, false);
});

test("S-LOC treats Go nil references and empty results as observed zero, not absence", () => {
  const raw = {
    ...inspection([]), sourceStatus: "not-indexed",
    sources: { subject, artifacts: [], references: null },
  };
  const view = readSourceLocations(raw, subject, catalog);
  assert.equal(view.observedReferences, 0);
  assert.equal(view.sourceStatus, "not-indexed");
  assert.equal(view.completenessEstablished, false);
});

test("S-LOC rejects mismatched Catalog, Subject, write operation and missing limits", () => {
  const raw = inspection();
  assert.throws(() => readSourceLocations(
    { ...raw, sourceIndexFingerprint: "source-B" }, subject, catalog,
  ), /STALE_MODEL_PROJECTION/);
  assert.throws(() => readSourceLocations(
    { ...raw, subject: { kind: "semantic-data", id: "other" } }, subject, catalog,
  ), /INVALID_INSPECTION_RESPONSE/);
  assert.throws(() => readSourceLocations(
    { ...raw, effect: "write" }, subject, catalog,
  ), /INVALID_INSPECTION_RESPONSE/);
  assert.throws(() => readSourceLocations(
    { ...raw, authorityLimits: authorityLimits.slice(1) }, subject, catalog,
  ), /INVALID_INSPECTION_RESPONSE/);
  assert.throws(() => readSourceLocations(
    { ...raw, sources: { ...raw.sources, subject: { kind: "semantic-data", id: "other" } } },
    subject, catalog,
  ), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
  assert.throws(() => readSourceLocations(
    { ...raw, sources: undefined }, subject, catalog,
  ), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
});

test("S-LOC refuses missing or conflicting Artifact provenance and invalid byte ranges", () => {
  const raw = inspection();
  assert.throws(() => readSourceLocations({
    ...raw, sources: { ...raw.sources, artifacts: [] },
  }, subject, catalog), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
  assert.throws(() => readSourceLocations({
    ...raw, sources: { ...raw.sources, artifacts: [artifact, artifact] },
  }, subject, catalog), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
  assert.throws(() => readSourceLocations({
    ...raw, sources: { ...raw.sources, references: [
      { ...location(0), range: { start: location(0).range.end, end: location(0).range.start } },
    ] },
  }, subject, catalog), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
  assert.throws(() => readSourceLocations({
    ...raw, sources: { ...raw.sources, references: [
      { ...location(0), range: { start: { offset: 1, line: 0, column: 2 }, end: location(0).range.end } },
    ] },
  }, subject, catalog), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
});

test("S-LOC rejects absolute, traversal, drive and noncanonical paths", () => {
  const paths = ["/etc/passwd", "../secrets", "orders/../private", "a//b", "C:/drive", "a\\b", ""];
  for (const path of paths) {
    const raw = inspection([location(0, "definition", path)]);
    assert.throws(() => readSourceLocations(raw, subject, catalog),
      /INVALID_SOURCE_LOCATIONS_RESPONSE/, path);
  }
});

test("S-LOC carries untrusted Source path only as text metadata, not an open URL", () => {
  const malicious = "folder/<img src=x onerror=alert(1)>.tei";
  const raw = inspection([location(0, "authoring", malicious)]);
  const view = readSourceLocations({
    ...raw, sources: { ...raw.sources, artifacts: [{ ...artifact, path: malicious }] },
  }, subject, catalog);
  assert.equal(view.shownReferences[0].workspaceRelativePath, malicious);
  assert.equal("url" in view.shownReferences[0], false);
  assert.equal("canOpen" in view.shownReferences[0], false);
});

const attributeSubject = {
  kind: "semantic-attribute",
  id: "studio.inspection_response.snapshot_fingerprint",
};
const attributeRevision = "sha256:" + "b".repeat(64);
const attributeCatalog = {
  ...catalog,
  entries: [{ subject: attributeSubject, sourceStatus: "indexed", ambiguous: false }],
  modelRevision: attributeRevision,
};
function attributeInspection(refs = [location(0)]) {
  const references = refs.map(ref => ({ ...ref, subject: attributeSubject }));
  const base = inspection(references);
  return {
    ...base, subject: attributeSubject, canonical: { id: attributeSubject.id },
    sources: { ...base.sources, subject: attributeSubject, references },
  };
}
function requiredDraft(raw = attributeInspection(), cat = attributeCatalog) {
  return previewRequiredEditDraft(
    raw, attributeSubject, cat, "allowed-workspace", "required-1",
    "StudioのData属性を必須にする", true,
  );
}

test("required draft uses qualified attribute and model version without claiming Source write", () => {
  const candidate = requiredDraft();
  assert.deepEqual(candidate.edit, {
    schema: "tei.reference.studio-edit/v0",
    expectedModelRevision: attributeRevision,
    id: "required-1",
    origin: "studio",
    intent: "StudioのData属性を必須にする",
    view: "required",
    subject: "attribute:studio.inspection_response.snapshot_fingerprint",
    operation: "set-attribute-required",
    required: true,
  });
  assert.equal(candidate.workspaceHint, "allowed-workspace");
  assert.equal(candidate.sourceObservation.workspaceRelativePath, artifact.path);
  assert.equal(candidate.sourceObservation.artifactFingerprint, fingerprint);
  assert.equal(candidate.status, "unreviewed-draft");
  assert.equal(candidate.sourceIndexCompletenessEstablished, false);
  assert.equal(candidate.sourceBytesVerified, false);
  assert.equal(candidate.sourceWriteAuthorized, false);
  assert.equal("accept" in candidate, false);
  assert.equal("writer" in candidate, false);
  const no = previewRequiredEditDraft(
    attributeInspection(), attributeSubject, attributeCatalog,
    "allowed-workspace", "required-2", "任意に戻す", false,
  );
  assert.equal(no.edit.required, false);
});

test("required draft refuses non-attribute subjects, ambiguous Catalog and invalid revisions", () => {
  assert.throws(() => previewRequiredEditDraft(
    inspection(), subject, catalog, "allowed-workspace", "r", "intent", true,
  ), /REQUIRED_EDIT_NEEDS_ATTRIBUTE/);
  assert.throws(() => requiredDraft(attributeInspection(), {
    ...attributeCatalog,
    entries: [{ subject: attributeSubject, sourceStatus: "indexed", ambiguous: true }],
  }), /REQUIRED_EDIT_SUBJECT_SUBJECT_AMBIGUOUS/);
  // The Markdown cell is named snapshotFingerprint, but the Source-backed
  // canonical Subject ID uses snapshot_fingerprint. A display label is not an ID.
  assert.throws(() => previewRequiredEditDraft(
    attributeInspection(),
    { ...attributeSubject, id: "studio.inspection_response.snapshotFingerprint" },
    attributeCatalog, "allowed-workspace", "r", "intent", true,
  ), /REQUIRED_EDIT_SUBJECT_SUBJECT_NOT_FOUND/);
  assert.throws(() => requiredDraft(attributeInspection(), {
    ...attributeCatalog,
    modelRevision: "old-snapshot",
  }), /INVALID_REQUIRED_EDIT_MODEL_REVISION/);
  assert.throws(() => requiredDraft(attributeInspection(), {
    ...attributeCatalog,
    sourceIndexFingerprint: "source-B",
  }), /STALE_MODEL_PROJECTION/);
});

test("required draft never silently reuses a Source occurrence after a same-byte relocation", () => {
  const before = requiredDraft();
  const movedPath = "orders/relocated/注文.tei";
  const movedCatalog = {
    ...attributeCatalog,
    modelRevision: "sha256:" + "c".repeat(64),
    sourceIndexFingerprint: "source-relocated",
  };
  const oldInspect = attributeInspection();
  const relocatedInspect = {
    ...oldInspect,
    sourceIndexFingerprint: "source-relocated",
    sources: {
      ...oldInspect.sources,
      artifacts: [{ ...artifact, path: movedPath }],
      references: oldInspect.sources.references.map(ref => ({
        ...ref, artifactPath: movedPath,
      })),
    },
  };
  // Identical Artifact bytes do not make the old Source location current.
  assert.throws(() => requiredDraft(oldInspect, movedCatalog), /STALE_MODEL_PROJECTION/);
  assert.throws(() => requiredDraft(relocatedInspect, attributeCatalog), /STALE_MODEL_PROJECTION/);
  const after = requiredDraft(relocatedInspect, movedCatalog);
  assert.equal(before.sourceObservation.artifactFingerprint,
    after.sourceObservation.artifactFingerprint);
  assert.notEqual(before.sourceObservation.workspaceRelativePath,
    after.sourceObservation.workspaceRelativePath);
  assert.notEqual(before.edit.expectedModelRevision, after.edit.expectedModelRevision);
  assert.equal(before.status, "unreviewed-draft");
  assert.equal(after.status, "unreviewed-draft");
  assert.equal(after.sourceIndexCompletenessEstablished, false);
  assert.equal(after.sourceBytesVerified, false);
  assert.equal(after.sourceWriteAuthorized, false);
});

test("required draft refuses incomplete, missing or duplicate Source occurrences", () => {
  for (const refs of [
    [],
    [location(0, "implementation")],
    [location(0), location(1)],
    Array.from({ length: 23 }, (_, i) => location(i)),
  ]) {
    assert.throws(() => requiredDraft(attributeInspection(refs)),
      /REQUIRED_EDIT_SOURCE_OCCURRENCE_UNRESOLVED/);
  }
  assert.throws(() => requiredDraft({
    ...attributeInspection(),
    sourceStatus: "not-indexed",
  }), /REQUIRED_EDIT_SOURCE_OCCURRENCE_UNRESOLVED/);
  const tampered = attributeInspection();
  tampered.sources.artifacts[0] = { ...artifact, fingerprint: "unverified" };
  assert.throws(() => requiredDraft(tampered),
    /REQUIRED_EDIT_SOURCE_OCCURRENCE_UNRESOLVED/);
});

test("required draft rejects malformed inputs and cannot use a different Source subject", () => {
  const base = attributeInspection();
  assert.throws(() => previewRequiredEditDraft(
    base, attributeSubject, attributeCatalog, "", "r", "intent", true,
  ), /INVALID_REQUIRED_EDIT_WORKSPACE/);
  assert.throws(() => previewRequiredEditDraft(
    base, attributeSubject, attributeCatalog, "allowed-workspace", " r ", "intent", true,
  ), /INVALID_REQUIRED_EDIT_CANDIDATE_ID/);
  assert.throws(() => previewRequiredEditDraft(
    base, attributeSubject, attributeCatalog, "allowed-workspace", "r", "", true,
  ), /INVALID_REQUIRED_EDIT_INTENT/);
  assert.throws(() => previewRequiredEditDraft(
    base, attributeSubject, attributeCatalog, "allowed-workspace", "r", "intent", "yes",
  ), /INVALID_REQUIRED_EDIT_CHOICE/);
  assert.throws(() => requiredDraft({
    ...base, sources: { ...base.sources, subject: subject },
  }), /INVALID_SOURCE_LOCATIONS_RESPONSE/);
});

test("Workbench exposes only an explicitly requested unreviewed draft, invalidated on navigation", () => {
  const main = readFileSync(new URL("./main.ts", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  assert.match(html, /id="preview-required" type="button" disabled/);
  assert.match(html, /id="required-preview-json" hidden/);
  assert.match(main, /previewRequiredEditDraft\(/);
  assert.match(main, /addEventListener\("click", showRequiredPreview\)/);
  assert.match(main, /async function inspect\(subject: Subject\) \{[\s\S]*?clearRequiredPreview\(\);[\s\S]*?inspectEligibility/);
  assert.match(main, /async function loadWorkspace\(\) \{[\s\S]*?clearRequiredPreview\(\);[\s\S]*?responses\.reset\(\)/);
  assert.match(main, /current\.workspaceEpoch !== workspaceEpoch/);
  assert.match(main, /current\.inspectionEpoch !== inspectionEpoch/);
  assert.match(main, /current\.catalog !== catalogSnapshot/);
  assert.match(main, /subjectKey\(current\.subject\) !== subjectKey\(selected\)/);
  assert.match(main, /JSON\.stringify\(draft\.edit, null, 2\)/);
  assert.match(main, /draft\.sourceObservation/);
  assert.match(main, /observed\.workspaceRelativePath/);
  assert.match(main, /observed\.artifactFingerprint/);
  assert.match(main, /Source Indexの網羅性・原文照合・書込権限は未確認/);
  assert.doesNotMatch(html, /id="(accept-required|save-required)"/);
});
