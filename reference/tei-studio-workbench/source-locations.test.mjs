import test from "node:test";
import assert from "node:assert/strict";
import { readSourceLocations } from "./source-locations.ts";

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
