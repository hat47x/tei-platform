import test from "node:test";
import assert from "node:assert/strict";
import { readPinnedSubject, catalogIdentity } from "./pin-reader.ts";

const subjectA = { kind: "semantic-data", id: "sales.order" };
const subjectB = { kind: "semantic-data", id: "sales.invoice" };
const request = {
  generation: 2, workspaceId: "reference-workspace", workspaceEpoch: 4,
  subject: subjectA,
};
const catalog = {
  entries: [{ subject: subjectA, sourceStatus: "fresh", ambiguous: false }],
  modelRevision: "model-1", worldFingerprint: "world-1",
  sourceIndexFingerprint: "index-1",
};
const inspection = {
  schemaVersion: "tei.operations.inspect/v0", effect: "read-only", subject: subjectA,
  canonical: { privateDetails: "not handed off by this reader" },
  worldFingerprint: "world-1", sourceIndexFingerprint: "index-1",
  sourceStatus: "fresh",
  // The real Go InspectSubject supplies these read-only limits. A Pin must not
  // weaken that contract or accept a fixture with a silently missing Grant.
  authorityLimits: [
    "inspection-is-a-derived-read-only-projection",
    "source-location-is-not-semantic-acceptance",
    "source-location-is-not-write-authority",
    "local-read-is-not-disclosure-permission",
  ],
};

function clientWith(result) {
  const calls = [];
  return {
    calls,
    async inspectSubject(...args) {
      calls.push(args);
      return result;
    },
  };
}

test("PIN-PRODUCT-01: call real Product Reader interface with the original request scope", async () => {
  const client = clientWith({ ok: true, value: inspection });
  const output = await readPinnedSubject(client, request, catalog);
  assert.deepEqual(client.calls, [["reference-workspace", "semantic-data", "sales.order"]]);
  assert.deepEqual(output, { version: "world-1:index-1" });
  assert.equal("canonical" in output, false);
});

test("PIN-PRODUCT-02: an ambiguous Catalog refuses to call the Host Reader", async () => {
  const client = clientWith({ ok: true, value: inspection });
  const ambiguous = {
    ...catalog,
    entries: [{ subject: subjectA, ambiguous: true, sourceStatus: "fresh" }],
  };
  await assert.rejects(readPinnedSubject(client, request, ambiguous), /SUBJECT_AMBIGUOUS/);
  assert.equal(client.calls.length, 0);
});

test("PIN-PRODUCT-03: missing Subject refuses to call the Host Reader", async () => {
  const client = clientWith({ ok: true, value: inspection });
  const missing = { ...catalog, entries: [{ subject: subjectB, ambiguous: false, sourceStatus: "fresh" }] };
  await assert.rejects(readPinnedSubject(client, request, missing), /SUBJECT_NOT_FOUND/);
  assert.equal(client.calls.length, 0);
});

test("PIN-PRODUCT-04: wrong Subject and stale World or Source Index never qualify", async () => {
  const mismatches = [
    [{ ...inspection, subject: subjectB }, /INVALID_INSPECTION_RESPONSE/],
    [{ ...inspection, worldFingerprint: "world-old" }, /STALE_MODEL_PROJECTION/],
    [{ ...inspection, sourceIndexFingerprint: "index-old" }, /STALE_MODEL_PROJECTION/],
  ];
  for (const [reply, reason] of mismatches) {
    const client = clientWith({ ok: true, value: reply });
    await assert.rejects(readPinnedSubject(client, request, catalog), reason);
    assert.equal(client.calls.length, 1);
  }
});

test("PIN-PRODUCT-05: Host denial is not confused with a readable Product projection", async () => {
  const client = clientWith({ ok: false, failure: { code: "PERMISSION_DENIED" } });
  await assert.rejects(readPinnedSubject(client, request, catalog), /PERMISSION_DENIED/);
  assert.equal(client.calls.length, 1);
});

// This is a real call to a fake Host interface, not a live Wails/OS or
// disclosure/AI handoff test. Latest Source/Grant must be verified separately.

test("PIN-PRODUCT-06: a same-name Subject in a changed Catalog is not the pinned origin", () => {
  const pinned = catalogIdentity(catalog);
  assert.ok(pinned);
  assert.equal(catalogIdentity({ ...catalog }), pinned);
  // An unchanged Subject Kind/ID is insufficient when the Workspace is revised.
  for (const changed of [
    { ...catalog, modelRevision: "model-2" },
    { ...catalog, worldFingerprint: "world-2" },
    { ...catalog, sourceIndexFingerprint: "index-2" },
  ]) {
    assert.equal(changed.entries[0].subject.id, request.subject.id);
    assert.notEqual(catalogIdentity(changed), pinned);
  }
  assert.equal(catalogIdentity(null), null);
});

test("PIN-PRODUCT-07: blank revision is unknown, not a qualified Catalog identity", async () => {
  for (const key of ["modelRevision", "worldFingerprint", "sourceIndexFingerprint"]) {
    for (const missing of ["", "  "]) {
      const unknown = { ...catalog, [key]: missing };
      const client = clientWith({ ok: true, value: inspection });
      assert.equal(catalogIdentity(unknown), null);
      await assert.rejects(readPinnedSubject(client, request, unknown),
        /CATALOG_IDENTITY_UNAVAILABLE/);
      assert.equal(client.calls.length, 0);
    }
  }
  assert.ok(catalogIdentity(catalog));
});

test("PIN-PRODUCT-08: a Host reply missing authority limits never qualifies Pin", async () => {
  for (const limits of [undefined, [], inspection.authorityLimits.slice(1)]) {
    const reply = { ...inspection, authorityLimits: limits };
    const client = clientWith({ ok: true, value: reply });
    await assert.rejects(readPinnedSubject(client, request, catalog),
      /INVALID_INSPECTION_RESPONSE/);
    assert.deepEqual(client.calls,
      [["reference-workspace", "semantic-data", "sales.order"]]);
  }
});
