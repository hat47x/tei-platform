import test from "node:test";
import assert from "node:assert/strict";
import {
  initialPinState, pinSubject, unpinSubject, restorePinIntent,
  invalidatePin, pinRequest, acceptPinObservation,
} from "./pin-state.ts";

const A = { kind: "semantic-data", id: "sales.order" };
const B = { kind: "semantic-data", id: "sales.invoice" };
const inspected = { state: "inspected" };

test("PIN-ABA-01: old reply cannot qualify a new pin of the same subject", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const old = pinRequest(state);
  state = pinSubject(unpinSubject(state), "workspace-a", 1, A);
  assert.equal(acceptPinObservation(state, old, inspected), state);
  assert.equal(state.observation.state, "unqualified");
  const current = pinRequest(state);
  state = acceptPinObservation(state, current, inspected);
  assert.equal(state.observation.state, "inspected");
});

test("PIN-ABA-02: undo restores intent but not previous inspection", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const saved = state.intent;
  const old = pinRequest(state);
  state = restorePinIntent(unpinSubject(state), saved);
  assert.deepEqual(state.intent.subject, A);
  assert.equal(state.observation.state, "unqualified");
  assert.equal(acceptPinObservation(state, old, inspected), state);
});

test("PIN-ABA-03: workspace A to B to A cannot revive old request", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const old = pinRequest(state);
  state = pinSubject(state, "workspace-b", 2, B);
  state = pinSubject(state, "workspace-a", 3, A);
  assert.equal(acceptPinObservation(state, old, inspected), state);
});

test("PIN-ABA-04: another subject or workspace cannot qualify the pin", () => {
  const state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const request = pinRequest(state);
  assert.equal(acceptPinObservation(state, { ...request, subject: B }, inspected), state);
  assert.equal(acceptPinObservation(state, { ...request, workspaceId: "workspace-b" }, inspected), state);
  assert.equal(acceptPinObservation(state, { ...request, workspaceEpoch: 2 }, inspected), state);
});

test("PIN-ABA-05: invalidation rejects an earlier same-subject request", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const old = pinRequest(state);
  state = invalidatePin(state);
  assert.equal(state.observation.state, "needs-requalification");
  assert.equal(acceptPinObservation(state, old, inspected), state);
  state = acceptPinObservation(state, pinRequest(state), inspected);
  assert.equal(state.observation.state, "inspected");
});

test("PIN-ABA-06: local generation cannot assert Source freshness", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const old = pinRequest(state);
  state = invalidatePin(state);
  assert.equal(acceptPinObservation(state, old, inspected), state);
  assert.equal(state.observation.state, "needs-requalification");
});

test("unpin leaves no request and cannot accept a late result", () => {
  let state = pinSubject(initialPinState(), "workspace-a", 1, A);
  const old = pinRequest(state);
  state = unpinSubject(state);
  assert.equal(pinRequest(state), null);
  assert.equal(acceptPinObservation(state, old, inspected), state);
});
