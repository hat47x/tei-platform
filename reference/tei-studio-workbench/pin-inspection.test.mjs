import test from "node:test";
import assert from "node:assert/strict";
import { createResponseScope } from "./responses.ts";
import { createPinInspection } from "./pin-inspection.ts";

const A = { kind: "semantic-data", id: "sales.order" };
const B = { kind: "semantic-data", id: "sales.invoice" };
const catalog = (fp = "w1", index = "s1") => ({
  entries: [A, B].map(subject => ({
    subject, sourceStatus: "indexed", ambiguous: false,
  })),
  modelRevision: "m1", worldFingerprint: fp, sourceIndexFingerprint: index,
});
const reply = (subject = A, sourceStatus = "indexed", world = "w1") => ({
  ok: true,
  value: {
    effect: "read-only", subject, canonical: { id: subject.id },
    worldFingerprint: world, sourceIndexFingerprint: "s1", sourceStatus,
  },
});
function deferred() {
  let resolve;
  const promise = new Promise(yes => { resolve = yes; });
  return { promise, resolve };
}
function manager(inspect, snapshot = () => catalog()) {
  const states = [];
  const scope = createResponseScope();
  const pin = createPinInspection(scope, inspect, snapshot, s => states.push(s));
  return { pin, scope, states };
}

test("PIN-UI-01: A reinspection leaves displayed B selection and editor buffer untouched", async () => {
  const pending = deferred();
  const displayed = { subject: B, dirtyText: "unsaved B draft", graphNode: "B" };
  const { pin } = manager(() => pending.promise);
  const task = pin.pin("workspace-a", 1, A);
  assert.equal(pin.current().observation.state, "needs-requalification");
  pending.resolve(reply(A));
  assert.equal(await task, true);
  assert.deepEqual(displayed, {
    subject: B, dirtyText: "unsaved B draft", graphNode: "B",
  });
  assert.equal(pin.current().observation.state, "inspected");
  assert.deepEqual(pin.current().intent.subject, A);
});

test("PIN-UI-02: old reply after Unpin/Re-pin A never qualifies the new intent", async () => {
  const pending = deferred();
  const next = deferred();
  let attempt = 0;
  const { pin } = manager(() => ++attempt === 1 ? pending.promise : next.promise);
  const old = pin.pin("workspace-a", 1, A);
  pin.release();
  const fresh = pin.pin("workspace-a", 1, A);
  pending.resolve(reply(A));
  assert.equal(await old, false);
  assert.notEqual(pin.current().observation.state, "inspected");
  next.resolve(reply(A));
  assert.equal(await fresh, true);
  assert.equal(pin.current().observation.state, "inspected");
});

test("PIN-UI-03: switching Workspace A→B→A requires explicit recheck", async () => {
  let reads = 0;
  const { pin } = manager(async () => { reads++; return reply(A); });
  assert.equal(await pin.pin("workspace-a", 1, A), true);
  pin.invalidate();
  assert.equal(pin.current().observation.state, "needs-requalification");
  assert.equal(await pin.recheck("workspace-b", 2), false);
  assert.equal(reads, 1);
  assert.equal(pin.current().intent.workspaceId, "workspace-a");
  assert.equal(await pin.recheck("workspace-a", 3), true);
  assert.equal(pin.current().intent.workspaceEpoch, 3);
  assert.equal(reads, 2);
});

test("PIN-UI-04: valid Inspect with unindexed Source is not a qualified Source pin", async () => {
  const { pin } = manager(async () => reply(A, "not-indexed"));
  assert.equal(await pin.pin("workspace-a", 1, A), false);
  assert.equal(pin.current().observation.state, "needs-requalification");
});

test("PIN-UI-05: wrong subject or mismatched Source revision is not accepted", async () => {
  const { pin: wrong } = manager(async () => reply(B));
  assert.equal(await wrong.pin("workspace-a", 1, A), false);
  assert.equal(wrong.current().observation.state, "unavailable");
  const { pin: stale } = manager(async () => reply(A, "indexed", "changed"));
  assert.equal(await stale.pin("workspace-a", 1, A), false);
  assert.equal(stale.current().observation.state, "unavailable");
});

test("PIN-UI-06: ambiguous Catalog forbids a read-only Host call", async () => {
  let invoked = 0;
  const ambiguous = () => ({
    ...catalog(),
    entries: [{ subject: A, ambiguous: true, sourceStatus: "indexed" }],
  });
  const { pin } = manager(async () => { invoked++; return reply(A); }, ambiguous);
  assert.equal(await pin.pin("workspace-a", 1, A), false);
  assert.equal(invoked, 0);
  assert.equal(pin.current().observation.state, "needs-requalification");
});

test("PIN-UI-07: catalog replacement during delayed Inspect cannot become inspected", async () => {
  const pending = deferred();
  let snapshot = catalog();
  const { pin } = manager(() => pending.promise, () => snapshot);
  const task = pin.pin("workspace-a", 1, A);
  snapshot = catalog("w2");
  pending.resolve(reply(A));
  assert.equal(await task, false);
  assert.equal(pin.current().observation.state, "unavailable");
});
