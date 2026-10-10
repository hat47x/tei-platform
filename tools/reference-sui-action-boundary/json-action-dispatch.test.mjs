import assert from "node:assert/strict";
import test from "node:test";
import { ActionCommitError, createJSONActionDispatcher } from "./json-action-dispatch.mjs";

const intent = { applicationID: "sui", documentID: "doc", actionID: "sui.move", expectedRevision: "r1", payload: { cardId: "a", x: 1, y: 2 } };

test("same-origin commit serializes one intent and returns server revision", async () => {
  const requests = [];
  const send = createJSONActionDispatcher("/actions/commit", async (path, init) => {
    requests.push({ path, init });
    return { ok: true, status: 200, json: async () => ({ revision: "r1:1" }) };
  });
  assert.equal(await send(intent), "r1:1");
  assert.equal(requests.length, 1);
  assert.equal(requests[0].path, "/actions/commit");
  assert.equal(requests[0].init.credentials, "same-origin");
  assert.equal(requests[0].init.headers["X-TEI-Action"], "commit");
  assert.deepEqual(JSON.parse(requests[0].init.body), intent);
});

test("denial and conflict are distinguishable and cannot be swallowed", async () => {
  for (const [status, code] of [[403,"action_denied"], [409,"revision_conflict"]]) {
    const send = createJSONActionDispatcher("/actions/commit", async () => ({ ok:false, status, json: async () => ({ error:code }) }));
    await assert.rejects(send(intent), (e) => e instanceof ActionCommitError && e.status===status && e.code===code);
  }
});

test("rejects network-path and absolute URLs to avoid sending session credentials elsewhere", () => {
  for (const path of ["https://bad.example/commit", "//bad.example/commit", "commit", ""]) {
    assert.throws(() => createJSONActionDispatcher(path, async () => ({})), TypeError);
  }
});

test("untrusted or malformed server response is an error", async () => {
  for (const response of [
    { ok: true, status:200, json:async () => ({}) },
    { ok: true, status:200, json:async () => ({revision: null}) },
    { ok: true, status:200, json:async () => {throw new Error("bad json")} },
  ]) {
    const send = createJSONActionDispatcher("/actions/commit", async () => response);
    await assert.rejects(send(intent), (e) => e instanceof ActionCommitError && e.code==="invalid_response");
  }
});
