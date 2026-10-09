import assert from "node:assert/strict";
import test from "node:test";
import { createNativeActionBridge } from "./native-action-bridge.mjs";

const action = (applicationID = "sui") => ({
  applicationID, documentID: "doc", actionID: "move", expectedRevision: "r1",
});

test("preview and cancel are local only, with no commit", async () => {
  const commits = [];
  const bridge = createNativeActionBridge((intent) => commits.push(intent));
  bridge.begin(action());
  bridge.preview({ cardId: "a", x: 1, y: 2 });
  bridge.preview({ cardId: "a", x: 3, y: 4 });
  assert.equal(commits.length, 0);
  assert.equal(bridge.cancel(), true);
  assert.equal(bridge.phase, "idle");
  assert.equal(commits.length, 0);
  await assert.rejects(bridge.drop(), /no active interaction/);
  assert.equal(commits.length, 0);
});

test("one drop sends exactly one application-owned action with final preview", async () => {
  const commits = [];
  const bridge = createNativeActionBridge(async (intent) => {
    commits.push(intent);
    return "r2";
  });
  bridge.begin(action());
  const preview = { cardId: "a", x: 1, y: 2 };
  bridge.preview(preview);
  preview.x = 99;
  bridge.preview({ cardId: "a", x: 3, y: 4 });
  const result = await bridge.drop();
  assert.equal(result, "r2");
  assert.deepEqual(commits, [{ ...action(), payload: { cardId: "a", x: 3, y: 4 } }]);
  assert.equal(bridge.phase, "idle");
  await assert.rejects(bridge.drop(), /no active interaction/);
  assert.equal(commits.length, 1);
});

test("duplicate drop is blocked while commit is pending", async () => {
  let release;
  const pending = new Promise((resolve) => { release = resolve; });
  const commits = [];
  const bridge = createNativeActionBridge(async (intent) => {
    commits.push(intent);
    await pending;
    return "r2";
  });
  bridge.begin(action());
  bridge.preview({ a: 1 });
  const first = bridge.drop();
  assert.equal(bridge.phase, "pending");
  assert.equal(bridge.cancel(), false);
  assert.throws(() => bridge.begin(action()), /already active/);
  await assert.rejects(bridge.drop(), /no active interaction/);
  release();
  assert.equal(await first, "r2");
  assert.equal(commits.length, 1);
  assert.equal(bridge.phase, "idle");
});

test("authorization denial and stale revision remain visible, never auto-retried", async () => {
  for (const reason of ["denied", "conflict"]) {
    const commits = [];
    const bridge = createNativeActionBridge((intent) => {
      commits.push(intent);
      throw new Error(reason);
    });
    bridge.begin(action());
    bridge.preview({ cardId: "a", x: 0, y: 0 });
    await assert.rejects(bridge.drop(), new RegExp(reason));
    assert.equal(bridge.phase, "idle");
    assert.equal(commits.length, 1);
  }
});

test("second unrelated application uses the same adapter without SUI fields", async () => {
  const commits = [];
  const bridge = createNativeActionBridge((intent) => {
    commits.push(intent);
    return "v2";
  });
  bridge.begin({ ...action("inventory"), actionID: "resize" });
  bridge.preview({ item: "p1", width: 230 });
  assert.equal(await bridge.drop(), "v2");
  assert.deepEqual(commits, [{
    applicationID: "inventory", documentID: "doc", actionID: "resize",
    expectedRevision: "r1", payload: { item: "p1", width: 230 },
  }]);
});

test("identity and drop-without-preview fail closed", async () => {
  const bridge = createNativeActionBridge(() => { throw Error("should not dispatch"); });
  assert.throws(() => bridge.begin({ ...action(), documentID: "" }), /required/);
  bridge.begin(action());
  await assert.rejects(bridge.drop(), /requires a final preview/);
  assert.equal(bridge.cancel(), true);
});
