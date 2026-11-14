import test from "node:test";
import assert from "node:assert/strict";
import { createResponseScope } from "./responses.ts";
import { readInspection, inspectEligibility } from "./presentation.ts";
import {
  initialPinState, pinSubject, unpinSubject, invalidatePin,
  pinRequest, acceptPinObservation,
} from "./pin-state.ts";

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const A = { kind: "semantic-data", id: "sales.order" };
const B = { kind: "semantic-data", id: "sales.invoice" };
const inspected = { state: "inspected" };

test("PIN-LANE-01: B selection remains independent of pinned A inspection", async () => {
  const scope = createResponseScope();
  let pin = pinSubject(initialPinState(), "workspace-a", 1, A);
  const request = pinRequest(pin);
  const pending = deferred();
  const selected = [];
  const pinTask = scope.run("pinned-inspection", () => pending.promise,
    () => { pin = acceptPinObservation(pin, request, inspected); },
    () => { pin = invalidatePin(pin, "unavailable"); });
  await scope.run("inspection", async () => "B", value => selected.push(value),
    error => { throw error; });
  pending.resolve("A");
  assert.equal(await pinTask, true);
  assert.deepEqual(selected, ["B"]);
  assert.equal(pin.observation.state, "inspected");
});

test("PIN-LANE-02: Unpin invalidates an in-flight pinned Reader reply", async () => {
  const scope = createResponseScope();
  let pin = pinSubject(initialPinState(), "workspace-a", 1, A);
  const request = pinRequest(pin), pending = deferred();
  const task = scope.run("pinned-inspection", () => pending.promise,
    () => { pin = acceptPinObservation(pin, request, inspected); },
    () => { throw Error("unexpected failure"); });
  scope.invalidate("pinned-inspection");
  pin = unpinSubject(pin);
  pending.resolve("A");
  assert.equal(await task, false);
  assert.equal(pin.intent, null);
});

test("PIN-LANE-03: Workspace reset rejects a late pinned failure", async () => {
  const scope = createResponseScope();
  let pin = pinSubject(initialPinState(), "workspace-a", 1, A);
  const pending = deferred(), errors = [];
  const task = scope.run("pinned-inspection", () => pending.promise,
    () => { errors.push("stale success"); },
    () => { errors.push("stale failure"); });
  scope.reset();
  pin = pinSubject(pin, "workspace-b", 2, B);
  pending.reject(Error("old workspace"));
  assert.equal(await task, false);
  assert.deepEqual(errors, []);
  assert.equal(pin.intent.workspaceId, "workspace-b");
});

test("PIN-LANE-04: newer pinned request suppresses older reply without touching selection", async () => {
  const scope = createResponseScope(), first = deferred(), second = deferred();
  const pinned = [], selected = [];
  const old = scope.run("pinned-inspection", () => first.promise,
    value => pinned.push(value), error => { throw error; });
  await scope.run("inspection", async () => "B", value => selected.push(value),
    error => { throw error; });
  const current = scope.run("pinned-inspection", () => second.promise,
    value => pinned.push(value), error => { throw error; });
  second.resolve("A-current");
  assert.equal(await current, true);
  first.resolve("A-old");
  assert.equal(await old, false);
  assert.deepEqual(pinned, ["A-current"]);
  assert.deepEqual(selected, ["B"]);
});


const catalog = {
  entries: [{ subject: A, sourceStatus: "indexed", ambiguous: false }],
  modelRevision: "model-v1",
  worldFingerprint: "world-v1",
  sourceIndexFingerprint: "source-v1",
};
const inspection = {
  effect: "read-only",
  subject: A,
  canonical: { name: "Order" },
  worldFingerprint: "world-v1",
  sourceIndexFingerprint: "source-v1",
  sourceStatus: "indexed",
};

test("PIN-READER-01: Product Inspector qualifies pinned A but cannot change viewed B", async () => {
  const scope = createResponseScope();
  let pin = pinSubject(initialPinState(), "workspace-a", 1, A);
  const request = pinRequest(pin);
  assert.ok(request);
  assert.equal(inspectEligibility(catalog, request.subject).ok, true);
  const viewed = { subject: B, text: "B stays visible", graph: "B graph", notice: "B notice" };
  let observedVersion = null;
  const accepted = await scope.run("pinned-inspection", async () => inspection,
    reply => {
      const projection = readInspection(reply, request.subject, catalog);
      observedVersion = projection.version;
      pin = acceptPinObservation(pin, request, { state: "inspected" });
    }, error => { throw error; });
  assert.equal(accepted, true);
  assert.equal(pin.observation.state, "inspected");
  assert.equal(observedVersion, "world-v1:source-v1");
  assert.deepEqual(viewed, {
    subject: B, text: "B stays visible", graph: "B graph", notice: "B notice",
  });
});

test("PIN-READER-02: wrong Subject or stale Source is not a qualified Pin observation", async () => {
  for (const mismatched of [
    { ...inspection, subject: B },
    { ...inspection, worldFingerprint: "stale-world" },
    { ...inspection, sourceIndexFingerprint: "stale-source" },
  ]) {
    const scope = createResponseScope();
    let pin = pinSubject(initialPinState(), "workspace-a", 1, A);
    const request = pinRequest(pin);
    assert.ok(request);
    const accepted = await scope.run("pinned-inspection", async () => mismatched,
      reply => {
        readInspection(reply, request.subject, catalog);
        pin = acceptPinObservation(pin, request, { state: "inspected" });
      }, error => {
        assert.match(String(error), /(INVALID_INSPECTION_RESPONSE|STALE_MODEL_PROJECTION)/);
        pin = acceptPinObservation(pin, request, { state: "needs-requalification" });
      });
    assert.equal(accepted, true);
    assert.equal(pin.observation.state, "needs-requalification");
  }
});
