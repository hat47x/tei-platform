import assert from "node:assert/strict";
import test from "node:test";
import {createThreeViewClient, describeFocus} from "../assets/three-view/client.mjs";

const value = {
  subjects: [
    {occurrence: "d1", view: "data", sourceFact: {kind: "Table", name: "Order"}},
    {occurrence: "f1", view: "function", sourceFact: {kind: "Function", name: "Order entry"}},
    {occurrence: "f2", view: "function", sourceFact: {kind: "Function", name: "Order entry"}},
    {occurrence: "a1", view: "business", sourceFact: {kind: "TaskAction", name: ""}},
  ],
  relations: [
    {occurrence: "r1", kind: "function-data-operation", nature: "source-fact", from: "f1", to: "d1", operations: "CR"},
    {occurrence: "r2", kind: "action-uses-function", nature: "source-fact", from: "a1", to: "f1"},
    {occurrence: "r3", kind: "action-data-potential-operation", nature: "structural-candidate", from: "a1", to: "d1", operations: "CR"},
  ],
  crud: [{function: "f1", data: "d1", create: true, read: true, update: false, delete: false, relations: ["r1"]}],
  fields: [{occurrence: "x1", ownerOccurrence: "d1", sourceFact: {kind: "TableField", name: "OrderNo"}}],
  fieldReferences: [{occurrence: "k1", kind: "key-member", owner: "d1", field: "x1", resolution: "resolved"}],
};

test("the client names one Bridge method and passes the pin as given", async () => {
  const calls = [];
  const client = createThreeViewClient(async (...args) => { calls.push(args); return {ok: true, value}; });
  await client.navigate("workspace-1", "design/order.xead");
  await client.navigate("workspace-1", "design/order.xead", {occurrence: "f1", sourceArtifactDigest: "sha256:x", label: "ignored"});
  assert.deepEqual(calls, [
    ["main.StudioBridge.NavigateThreeView", "workspace-1", "design/order.xead", "", ""],
    ["main.StudioBridge.NavigateThreeView", "workspace-1", "design/order.xead", "f1", "sha256:x"],
  ]);
});

test("a malformed or failed host answer never reaches the page as a value", async () => {
  for (const answer of [null, {}, {ok: true}, {ok: true, value: null}, {ok: false}, {ok: false, failure: {}}]) {
    const got = await createThreeViewClient(async () => answer).navigate("w", "p");
    assert.equal(got.ok, false);
    assert.equal(got.failure.code, "INVALID_HOST_RESPONSE");
  }
  const thrown = await createThreeViewClient(async () => { throw new Error("private path /secret"); }).navigate("w", "p");
  assert.deepEqual(thrown, {ok: false, failure: {code: "HOST_TRANSPORT_FAILURE", message: "Wails service is unavailable"}});
  assert.throws(() => createThreeViewClient(null), TypeError);
});

test("the same occurrence is reached from the Business view, the CRUD row and the fields", () => {
  const fromAction = describeFocus(value, "a1");
  assert.deepEqual(fromAction.relations.map((r) => [r.kind, r.direction, r.other, r.nature]), [
    ["action-uses-function", "out", "f1", "source-fact"],
    ["action-data-potential-operation", "out", "d1", "structural-candidate"],
  ]);
  // The business action has no CRUD row of its own.
  assert.deepEqual(fromAction.crud, []);

  const fromFunction = describeFocus(value, "f1");
  assert.equal(fromFunction.crud.length, 1);
  assert.equal(fromFunction.crud[0].data, "d1");

  const fromData = describeFocus(value, "d1");
  assert.equal(fromData.crud[0].function, "f1");
  assert.deepEqual(fromData.fields.map((f) => f.occurrence), ["x1"]);
  assert.deepEqual(fromData.fieldReferences.map((r) => r.field), ["x1"]);
});

test("a same-name function is a different occurrence with nothing of the other's", () => {
  const other = describeFocus(value, "f2");
  assert.deepEqual(other.relations, []);
  assert.deepEqual(other.crud, []);
  assert.equal(describeFocus(value, "missing"), null);
});
