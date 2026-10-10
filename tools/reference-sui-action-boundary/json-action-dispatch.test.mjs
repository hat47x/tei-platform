import assert from "node:assert/strict";
import test from "node:test";
import { ActionCommitError, createJSONActionDispatcher } from "./json-action-dispatch.mjs";

const intent = { protocolVersion: "1", applicationID: "sui", resourceID: "doc", actionID: "sui.move", expectedRevision: "r1", payload: { cardId: "a", x: 1, y: 2 } };

test("same-origin commit serializes one intent and returns server revision", async () => {
  const requests = [];
  const send = createJSONActionDispatcher("/actions/commit", async (path, init) => {
    requests.push({ path, init });
    return { ok: true, status: 200, json: async () => ({ protocolVersion: "1", revision: "r1:1" }) };
  });
  assert.equal(await send(intent), "r1:1");
  assert.equal(requests.length, 1);
  assert.equal(requests[0].path, "/actions/commit");
  assert.equal(requests[0].init.credentials, "same-origin");
  assert.equal(requests[0].init.mode, "same-origin");
  assert.equal(requests[0].init.redirect, "error");
  assert.equal(requests[0].init.headers["X-TEI-Action"], "commit");
  assert.deepEqual(JSON.parse(requests[0].init.body), intent);
});

test("denial and conflict are distinguishable and cannot be swallowed", async () => {
  for (const [status, code] of [[403,"action_denied"], [409,"revision_conflict"]]) {
    const send = createJSONActionDispatcher("/actions/commit", async () => ({ ok:false, status, json: async () => ({ protocolVersion: "1", error:code }) }));
    await assert.rejects(send(intent), (e) => e instanceof ActionCommitError && e.status===status && e.code===code);
  }
});

test("rejects network-path and absolute URLs to avoid sending session credentials elsewhere", () => {
  for (const path of ["https://bad.example/commit", "//bad.example/commit", "/\\bad.example/commit", "/\t/bad.example/commit", "commit", ""]) {
    assert.throws(() => createJSONActionDispatcher(path, async () => ({})), TypeError);
  }
});

test("untrusted or malformed server response is an error", async () => {
  for (const response of [
    { ok: true, status:200, json:async () => ({}) },
    { ok: true, status:200, json:async () => ({protocolVersion: "1", revision: null}) },
    { ok: true, status:200, json:async () => {throw new Error("bad json")} },
    { ok: true, status:200, json:async () => ({protocolVersion: "1", revision: "r2", error: "revision_conflict"}) },
    { ok: true, status:200, json:async () => ({protocolVersion: "1", revision: "r2", unexpected: true}) },
    { ok: true, status:201, json:async () => ({protocolVersion: "1", revision: "r2"}) },
    { ok: true, status:200, json:async () => [ {protocolVersion: "1", revision: "r2"} ] },
  ]) {
    const send = createJSONActionDispatcher("/actions/commit", async () => response);
    await assert.rejects(send(intent), (e) => e instanceof ActionCommitError && e.code==="invalid_response");
  }
});

test("second application uses identical protocol but different resource and payload", async () => {
  const requests = [];
  const send = createJSONActionDispatcher("/actions/commit", async (_path, init) => {
    requests.push(JSON.parse(init.body));
    return { ok: true, status: 200, json: async () => ({ protocolVersion: "1", revision: "v2" }) };
  });
  const other = {
    protocolVersion: "1", applicationID: "inventory", resourceID: "sku/p100",
    actionID: "inventory.adjust", expectedRevision: "v1",
    payload: { sku: "p100", delta: 4 },
  };
  assert.equal(await send(other), "v2");
  assert.deepEqual(requests, [other]);
});

test("rejects obsolete document field and missing version before fetching", async () => {
  let requests = 0;
  const send = createJSONActionDispatcher("/actions/commit", async () => { requests++; return {}; });
  for (const value of [
    { ...intent, protocolVersion: "2" },
    { ...intent, protocolVersion: undefined },
    { ...intent, resourceID: "" },
    { ...intent, resourceID: undefined, documentID: "doc" },
    { ...intent, payload: null },
  ]) {
    await assert.rejects(send(value), TypeError);
  }
  assert.equal(requests, 0);
});

test("malformed error envelopes cannot be reported as valid Action denials", async () => {
  const invalidBodies = [
    { protocolVersion: "1", error: "revision_conflict", revision: "r2" },
    { protocolVersion: "1", error: "" },
    { protocolVersion: "1", error: "Invalid Error" },
    { protocolVersion: "1", error: 403 },
    { protocolVersion: "1", revision: "r2" },
    { protocolVersion: "1", error: "action_denied", unexpected: true },
  ];
  for (const body of invalidBodies) {
    const send = createJSONActionDispatcher("/actions/commit", async () => ({
      ok: false, status: 403, json: async () => body,
    }));
    await assert.rejects(send(intent), (e) => e instanceof ActionCommitError &&
      e.code === "invalid_response" && e.status === 403);
  }
});
