import test from "node:test";
import assert from "node:assert/strict";
import { createResponseScope } from "./responses.ts";
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
