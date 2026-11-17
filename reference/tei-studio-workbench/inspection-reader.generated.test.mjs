// Compare the regenerated Reader to the product's current (still hand-written)
// Reader and independently specified outcomes. This is R0 parity only: the
// committed Wails bundle must not be claimed to use this part until R1/R2.
import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { readInspection as currentReadInspection } from "./presentation.ts";
import { readInspection as generatedReadInspection, inspectionReaderProvenance } from "./inspection-reader.generated.ts";

const here = (name) => new URL(name, import.meta.url);
const subject = { kind: "semantic-data", id: "sales.order" };
const catalog = { entries: [], modelRevision: "r", worldFingerprint: "w1", sourceIndexFingerprint: "s1" };
const limits = [
  "inspection-is-a-derived-read-only-projection",
  "source-location-is-not-semantic-acceptance",
  "source-location-is-not-write-authority",
  "local-read-is-not-disclosure-permission",
];
const ok = {
  schemaVersion: "tei.cli.inspect/v0", effect: "read-only", subject,
  canonical: { id: "sales.order", attributes: [{ id: "sales.order.customer_id" }] },
  worldFingerprint: "w1", sourceIndexFingerprint: "s1",
  sourceStatus: "indexed", authorityLimits: limits,
};
function outcome(read, ...args) {
  try { return { value: read(...args) }; }
  catch (error) { return { error: error.message }; }
}
const cases = [
  ["valid current reply", ok, subject, catalog, "value"],
  ["canonical null", { ...ok, canonical: null }, subject, catalog, "value"],
  ["sourceStatus partial", { ...ok, sourceStatus: "partial" }, subject, catalog, "value"],
  ["extra limits survive", { ...ok, authorityLimits: [...limits, "additional-scope"] }, subject, catalog, "value"],
  ["missing sourceStatus", { ...ok, sourceStatus: undefined }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["null sourceStatus", { ...ok, sourceStatus: null }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["missing limits", (({ authorityLimits, ...rest }) => rest)(ok), subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["non-array limits", { ...ok, authorityLimits: "read-only" }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["limit with wrong type", { ...ok, authorityLimits: [...limits, null] }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["missing one required limit", { ...ok, authorityLimits: limits.slice(1) }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["write effect", { ...ok, effect: "write" }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["different qualified id", { ...ok, subject: { kind: subject.kind, id: "sales.customer" } }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["different kind", { ...ok, subject: { kind: "semantic-attribute", id: subject.id } }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["missing canonical", (({ canonical, ...rest }) => rest)(ok), subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["invalid world fingerprint", { ...ok, worldFingerprint: 1 }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["missing index", { ...ok, sourceIndexFingerprint: undefined }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["subject without id", { ...ok, subject: { kind: subject.kind } }, subject, catalog, "INVALID_SUBJECT_RESPONSE"],
  ["array response", [ok], subject, catalog, "INVALID_MODEL_RESPONSE"],
  ["null response", null, subject, catalog, "INVALID_MODEL_RESPONSE"],
  ["old world", ok, subject, { ...catalog, worldFingerprint: "w2" }, "STALE_MODEL_PROJECTION"],
  ["old source index", ok, subject, { ...catalog, sourceIndexFingerprint: "s2" }, "STALE_MODEL_PROJECTION"],
  ["missing catalog", ok, subject, null, "STALE_MODEL_PROJECTION"],
];
for (const [name, raw, expected, snapshot, want] of cases) {
  test(`generated Reader matches independent product behavior: ${name}`, () => {
    const actual = outcome(generatedReadInspection, raw, expected, snapshot);
    const baseline = outcome(currentReadInspection, raw, expected, snapshot);
    assert.deepEqual(actual, baseline);
    if (want === "value") {
      assert.equal(actual.error, undefined);
      assert.equal(actual.value.version, "w1:s1");
      assert.equal(actual.value.text, JSON.stringify(raw.canonical, null, 2));
      assert.equal(actual.value.sourceStatus, raw.sourceStatus);
      assert.equal(actual.value.authorityLimits.length, raw.authorityLimits.length);
      assert.match(actual.value.authorityLimits[0], /読み取り専用/);
      if (raw.authorityLimits.length > 4)
        assert.match(actual.value.authorityLimits.at(-1), /追加の利用制限/);
    } else assert.equal(actual.error, want);
  });
}
test("generated Reader names the exact current contract Source", () => {
  const contract = JSON.parse(readFileSync(here("./inspection-reader.contract.json"), "utf8"));
  const digest = "sha256:" + createHash("sha256").update(JSON.stringify(contract)).digest("hex");
  assert.equal(inspectionReaderProvenance.contractDigest, digest);
  assert.equal(inspectionReaderProvenance.part, contract.part.id);
  assert.equal(inspectionReaderProvenance.operation, contract.operation.bridge);
  assert.deepEqual(Object.values(contract.requiredLimits), limits);
});
test("invalid projections do not count as generated Reader use", () => {
  const dataset = {};
  globalThis.document = { documentElement: { dataset } };
  try {
    assert.throws(() => generatedReadInspection({ ...ok, authorityLimits: undefined }, subject, catalog));
    assert.throws(() => generatedReadInspection({ ...ok, sourceStatus: undefined }, subject, catalog));
    assert.throws(() => generatedReadInspection(ok, subject, { ...catalog, worldFingerprint: "old" }));
    assert.deepEqual(dataset, {});
    generatedReadInspection(ok, subject, catalog);
    generatedReadInspection(ok, subject, catalog);
    assert.deepEqual(dataset, {
      teiInspectionReader: inspectionReaderProvenance.contractDigest,
      teiInspectionReaderCalls: "2",
    });
  } finally { delete globalThis.document; }
});
