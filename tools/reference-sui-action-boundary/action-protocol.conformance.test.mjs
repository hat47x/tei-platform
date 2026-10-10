import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { createNativeActionBridge } from "./native-action-bridge.mjs";
import { createJSONActionDispatcher } from "./json-action-dispatch.mjs";

const schema = JSON.parse(readFileSync(new URL("./action-protocol-v1.schema.json", import.meta.url), "utf8"));

test("wire envelope matches published JSON Schema's version and field set", () => {
  const required = [
    "protocolVersion", "applicationID", "resourceID", "actionID",
    "expectedRevision", "payload",
  ];
  assert.equal(schema.$schema, "https://json-schema.org/draft/2020-12/schema");
  assert.equal(schema.additionalProperties, false);
  assert.deepEqual([...schema.required].sort(), [...required].sort());
  assert.deepEqual(Object.keys(schema.properties).sort(), [...required].sort());
  assert.equal(schema.properties.protocolVersion.const, "1");
});

test("one common browser path commits both SUI and unrelated inventory actions", async () => {
  const sent = [];
  const dispatch = createJSONActionDispatcher("/actions/commit", async (path, init) => {
    const input = JSON.parse(init.body);
    sent.push({ path, input });
    return { ok: true, status: 200, json: async () => ({ protocolVersion: "1", revision: "v2" }) };
  });

  const cases = [
    {
      begin: { applicationID: "sui", resourceID: "canvas-1", actionID: "sui.move", expectedRevision: "v1" },
      old: { cardId: "a", x: 10, y: 20 },
      final: { cardId: "a", x: 12, y: 30 },
    },
    {
      begin: { applicationID: "inventory", resourceID: "warehouse/sku-1", actionID: "inventory.adjust", expectedRevision: "v1" },
      old: { sku: "sku-1", delta: 2 },
      final: { sku: "sku-1", delta: 4 },
    },
  ];
  for (const action of cases) {
    const bridge = createNativeActionBridge(dispatch);
    bridge.begin(action.begin);
    bridge.preview(action.old);
    assert.equal(sent.length, cases.indexOf(action));
    bridge.preview(action.final);
    assert.equal(await bridge.drop(), "v2");
    assert.equal(bridge.phase, "idle");
  }
  assert.equal(sent.length, 2);
  assert.deepEqual(sent.map(({ input }) => input), cases.map(({ begin, final }) => ({
    protocolVersion: "1", ...begin, payload: final,
  })));
});

test("cancel and conflict never dispatch or silently retry", async () => {
  const sent = [];
  const dispatch = createJSONActionDispatcher("/actions/commit", async (_path, init) => {
    sent.push(JSON.parse(init.body));
    return { ok: false, status: 409, json: async () => ({ protocolVersion: "1", error: "revision_conflict" }) };
  });
  const bridge = createNativeActionBridge(dispatch);
  const begin = { applicationID: "inventory", resourceID: "item-1", actionID: "inventory.adjust", expectedRevision: "v1" };
  bridge.begin(begin);
  bridge.preview({ sku: "item-1", delta: 2 });
  bridge.cancel();
  assert.equal(sent.length, 0);
  bridge.begin(begin);
  bridge.preview({ sku: "item-1", delta: 3 });
  await assert.rejects(bridge.drop(), /revision_conflict/);
  assert.equal(sent.length, 1);
  assert.equal(bridge.phase, "idle");
});
