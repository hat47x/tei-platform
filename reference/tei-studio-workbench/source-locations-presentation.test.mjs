import test from "node:test";
import assert from "node:assert/strict";
import { readSourceLocations } from "./source-locations.ts";
import { presentSourceLocations } from "./source-locations-presentation.ts";

const subject = { kind: "semantic-data", id: "sales.order" };
const catalog = {
  entries: [{ subject, sourceStatus: "indexed", ambiguous: false }],
  modelRevision: "model-A",
  worldFingerprint: "world-A",
  sourceIndexFingerprint: "source-A",
};
const limits = [
  "inspection-is-a-derived-read-only-projection",
  "source-location-is-not-semantic-acceptance",
  "source-location-is-not-write-authority",
  "local-read-is-not-disclosure-permission",
];
const digest = "sha256:" + "a".repeat(64);
const path = "orders/注文.tei";
const reference = (i, relativePath = path) => ({
  subject, role: i === 0 ? "definition" : "implementation",
  artifactPath: relativePath,
  range: {
    start: { offset: i * 3, line: i + 1, column: 1 },
    end: { offset: i * 3 + 3, line: i + 1, column: 4 },
  },
});
function raw(references = [reference(0)]) {
  return {
    schemaVersion: "tei.operations.inspect/v0",
    effect: "read-only", subject, canonical: { id: subject.id },
    sourceStatus: "indexed",
    worldFingerprint: "world-A", sourceIndexFingerprint: "source-A",
    authorityLimits: limits,
    sources: {
      subject, artifacts: [{ path, format: "tei", fingerprint: digest }],
      references,
    },
  };
}
const present = (value) =>
  presentSourceLocations(readSourceLocations(value, subject, catalog));

test("SL1a preserves the validated Source path, role, byte range and fingerprint as inert display text", () => {
  const result = present(raw());
  assert.deepEqual(result, {
    statusText: "参照情報：indexed",
    countsText: "観測した参照 1 件／表示 1 件／省略 0 件",
    completenessText: "参照が全件そろっているかは確認されていません。",
    rows: [{
      pathText: path, roleText: "definition",
      rangeText: "元Sourceの行 1:1–1:4（バイト 0–3）",
      fingerprintText: digest,
    }],
  });
  assert.equal("url" in result.rows[0], false);
  assert.equal("canOpen" in result.rows[0], false);
  assert.equal("sourceBytes" in result.rows[0], false);
  assert.equal("write" in result, false);
});

test("SL1a displays the 20-reference cap and reports unobserved completeness without claiming an open path", () => {
  const result = present(raw(Array.from({ length: 23 }, (_, i) => reference(i))));
  assert.equal(result.rows.length, 20);
  assert.equal(result.countsText, "観測した参照 23 件／表示 20 件／省略 3 件");
  assert.match(result.completenessText, /確認されていません/);
  assert.match(result.rows[19].rangeText, /バイト 57–60/);
});

test("SL1a treats an indexed zero and a not-indexed zero as distinct observations, not proof of full coverage", () => {
  const indexed = present(raw([]));
  const unavailable = raw([]);
  unavailable.sourceStatus = "not-indexed";
  unavailable.sources.references = null;
  unavailable.sources.artifacts = [];
  const absent = present(unavailable);
  assert.equal(indexed.rows.length, 0);
  assert.equal(absent.rows.length, 0);
  assert.notEqual(indexed.statusText, absent.statusText);
  assert.equal(indexed.countsText, absent.countsText);
  assert.equal(indexed.completenessText, absent.completenessText);
});

test("SL1a preserves untrusted path characters as text, never producing markup or a file URL", () => {
  const strange = "orders/<img src=x onerror=alert(1)>.tei";
  const value = raw([reference(0, strange)]);
  value.sources.artifacts = [{ path: strange, format: "tei", fingerprint: digest }];
  const result = present(value);
  assert.equal(result.rows[0].pathText, strange);
  assert.equal(result.rows[0].pathText.startsWith("file://"), false);
  assert.equal("html" in result.rows[0], false);
  assert.equal("href" in result.rows[0], false);
});

test("SL1a never converts stale, wrong-subject or write Inspect replies into a normal empty display", () => {
  assert.throws(() => present({ ...raw(), sourceIndexFingerprint: "source-B" }), /STALE_MODEL_PROJECTION/);
  assert.throws(() => present({ ...raw(), subject: { kind: "semantic-data", id: "other" } }), /INVALID_INSPECTION_RESPONSE/);
  assert.throws(() => present({ ...raw(), effect: "write" }), /INVALID_INSPECTION_RESPONSE/);
});

test("SL1a does not accept an invented full-coverage claim", () => {
  const valid = readSourceLocations(raw(), subject, catalog);
  assert.throws(
    () => presentSourceLocations({ ...valid, completenessEstablished: true }),
    /INVALID_SOURCE_LOCATIONS_PRESENTATION/,
  );
});
