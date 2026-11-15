// The generated Inspector reader is checked against things it was not derived
// from: the Stage0 hand-written reader (test-only oracle), the contract Source
// bytes on disk, and a DOM stand-in that observes actual use.
import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { readInspection, inspectionReaderProvenance } from "./presentation.ts";
import * as generated from "./inspection-reader.generated.ts";
import { seedReadInspection } from "./testdata/inspection-reader.seed-oracle.ts";

const here = (name) => new URL(name, import.meta.url);
const subject = { kind: "semantic-data", id: "sales.order" };
const catalog = { entries: [], modelRevision: "r", worldFingerprint: "w1", sourceIndexFingerprint: "s1" };
const ok = {
  schemaVersion: "tei.cli.inspect/v0",
  effect: "read-only",
  subject,
  canonical: { id: "sales.order", attributes: [{ id: "sales.order.customer_id" }] },
  worldFingerprint: "w1",
  sourceIndexFingerprint: "s1",
  sourceStatus: "indexed",
};
const outcome = (read, ...args) => {
  try {
    return { value: read(...args) };
  } catch (error) {
    return { error: error.message };
  }
};

// Every row runs through both readers. The expected column is written by hand
// from the contract, so agreement of the two readers alone cannot pass a row.
const cases = [
  ["accepted projection", ok, subject, catalog, "value"],
  ["canonical null is still a projection", { ...ok, canonical: null }, subject, catalog, "value"],
  ["missing sourceStatus reads as unknown", { ...ok, sourceStatus: undefined }, subject, catalog, "value"],
  ["write effect", { ...ok, effect: "write" }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["another subject id", { ...ok, subject: { kind: subject.kind, id: "sales.customer" } }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["another subject kind", { ...ok, subject: { kind: "semantic-attribute", id: subject.id } }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["canonical absent", (({ canonical, ...rest }) => rest)(ok), subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["world fingerprint not a string", { ...ok, worldFingerprint: 1 }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["index fingerprint absent", { ...ok, sourceIndexFingerprint: undefined }, subject, catalog, "INVALID_INSPECTION_RESPONSE"],
  ["subject without id", { ...ok, subject: { kind: subject.kind } }, subject, catalog, "INVALID_SUBJECT_RESPONSE"],
  ["response is an array", [ok], subject, catalog, "INVALID_MODEL_RESPONSE"],
  ["response is null", null, subject, catalog, "INVALID_MODEL_RESPONSE"],
  ["catalog world moved on", ok, subject, { ...catalog, worldFingerprint: "w2" }, "STALE_MODEL_PROJECTION"],
  ["catalog index moved on", ok, subject, { ...catalog, sourceIndexFingerprint: "s2" }, "STALE_MODEL_PROJECTION"],
  ["no catalog snapshot", ok, subject, null, "STALE_MODEL_PROJECTION"],
  ["no catalog argument", ok, subject, undefined, "STALE_MODEL_PROJECTION"],
];

for (const [name, raw, expected, snapshot, want] of cases) {
  test(`generated reader and Stage0 seed agree: ${name}`, () => {
    const got = outcome(readInspection, raw, expected, snapshot);
    const seed = outcome(seedReadInspection, raw, expected, snapshot);
    assert.deepEqual(got, seed);
    if (want === "value") {
      assert.equal(got.error, undefined);
      assert.equal(got.value.version, "w1:s1");
      assert.equal(got.value.text, JSON.stringify(raw.canonical, null, 2));
    } else {
      assert.equal(got.error, want);
    }
  });
}

test("the Workbench entry re-exports the generated part, not a second implementation", () => {
  assert.equal(readInspection, generated.readInspection);
  const presentation = readFileSync(here("./presentation.ts"), "utf8");
  assert.doesNotMatch(presentation, /function readInspection/);
  assert.match(presentation, /from "\.\/inspection-reader\.generated\.ts"/);
  // main.ts stays the pinned consumer; it reaches the part only through presentation.ts.
  const main = readFileSync(here("./main.ts"), "utf8");
  assert.doesNotMatch(main, /inspection-reader/);
  assert.doesNotMatch(main, /seed-oracle/);
});

test("the part names the contract Source that is on disk now", () => {
  // Recomputed here from the bytes, without the Go engine: a part generated
  // from an older contract revision fails this even if it still type-checks.
  const contract = JSON.parse(readFileSync(here("./inspection-reader.contract.json"), "utf8"));
  const digest = "sha256:" + createHash("sha256").update(JSON.stringify(contract)).digest("hex");
  assert.equal(inspectionReaderProvenance.contractDigest, digest);
  assert.equal(inspectionReaderProvenance.part, contract.part.id);
  assert.equal(inspectionReaderProvenance.version, contract.part.version);
  assert.equal(inspectionReaderProvenance.operation, contract.operation.bridge);
});

test("only an accepted projection records use, and each use is counted", () => {
  const dataset = {};
  globalThis.document = { documentElement: { dataset } };
  try {
    assert.throws(() => readInspection({ ...ok, effect: "write" }, subject, catalog));
    assert.throws(() => readInspection(ok, subject, { ...catalog, worldFingerprint: "w2" }));
    assert.deepEqual(dataset, {});
    readInspection(ok, subject, catalog);
    readInspection(ok, subject, catalog);
    assert.deepEqual(dataset, {
      teiInspectionReader: inspectionReaderProvenance.contractDigest,
      teiInspectionReaderCalls: "2",
    });
  } finally {
    delete globalThis.document;
  }
});
