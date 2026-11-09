// Contract-only transport tests against the REAL #1832 JS client port.
// The callByName callback is simulated: these are NOT Wails IPC tests.
// Run only in the fixed qualification snapshot containing #1832's path.
import test from "node:test";
import assert from "node:assert/strict";
import { createReadOnlyClient } from "../../cmd/tei-studio/assets/client.mjs";

const workspaceID = "workspace-1";
const subject = { kind: "semantic-data", id: "sales.order" };
const methods = {
  validate: "main.StudioBridge.ValidateWorkspace",
  inspect: "main.StudioBridge.InspectSubject",
  catalog: "main.StudioBridge.CatalogWorkspace",
};

// A successful Host method invocation is NOT a validation verdict.
// The mock envelope here characterizes JS transport only, and must not be
// mistaken for CLI/Go/Host semantic evidence generated from a real Workspace.
test("valid transport preserves all warning diagnostics and the exact input", async () => {
  const calls = [];
  const findings = Array.from({length: 3}, (_, i) => ({
    diagnostic: {code: `WARN_${i}`, severity: "warning", message: "fixture"},
    sources: [],
  }));
  const client = createReadOnlyClient(async (name, ...args) => {
    calls.push([name, ...args]);
    return {ok: true, value: {Direct: null, Diagnostics: findings}};
  });
  const value = await client.validateWorkspace(workspaceID);
  assert.deepEqual(calls, [[methods.validate, workspaceID]]);
  assert.equal(value.ok, true);
  assert.equal(value.value.Diagnostics.length, 3);
  assert.ok(value.value.Diagnostics.every(item => item.diagnostic.severity === "warning"));
  assert.equal(value.failure, undefined);
});

test("unknown Workspace and missing subject remain operation denials, not transport errors", async () => {
  const seen = [];
  const client = createReadOnlyClient(async (name, ...args) => {
    seen.push([name, ...args]);
    return name === methods.validate
      ? {ok: false, failure: {code: "WORKSPACE_NOT_ALLOWED", message: "denied"}}
      : {ok: false, failure: {code: "SUBJECT_NOT_FOUND", message: "not found"}};
  });
  const denied = await client.validateWorkspace("workspace-999");
  const missing = await client.inspectSubject(workspaceID, subject.kind, "sales.missing");
  assert.equal(denied.ok, false);
  assert.equal(denied.failure.code, "WORKSPACE_NOT_ALLOWED");
  assert.equal(missing.ok, false);
  assert.equal(missing.failure.code, "SUBJECT_NOT_FOUND");
  assert.deepEqual(seen, [
    [methods.validate, "workspace-999"],
    [methods.inspect, workspaceID, subject.kind, "sales.missing"],
  ]);
});

test("malformed envelope and transport exception do not manufacture semantic success", async () => {
  const malformed = [
    null, [],
    {status: "success"}, {ok: "true"},
    {ok: true}, {ok: false}, {ok: false, failure: {}},
  ];
  for (const value of malformed) {
    const result = await createReadOnlyClient(async () => value).catalogWorkspace(workspaceID, "");
    assert.equal(result.ok, false, `malformed ${JSON.stringify(value)}`);
    assert.equal(result.failure.code, "INVALID_HOST_RESPONSE");
  }
  const failed = createReadOnlyClient(async () => {
    throw Error("/private/host/workspace/secret.tei");
  });
  const result = await failed.inspectSubject(workspaceID, subject.kind, subject.id);
  assert.equal(result.ok, false);
  assert.equal(result.failure.code, "HOST_TRANSPORT_FAILURE");
  assert.ok(!JSON.stringify(result).includes("/private/host/workspace"));
});

// The current JS port checks only the minimal envelope shape. Stronger shape
// validation belongs to the presentation reader, NOT a duplicate Host adapter.
// In particular this test must not claim that arbitrary malformed value payloads
// have been validated by the JS port.
test("minimal Host envelope and semantic payload validation are separate contracts", async () => {
  const client = createReadOnlyClient(async () => ({ok: true, value: {}}));
  const acceptedByTransport = await client.validateWorkspace(workspaceID);
  assert.equal(acceptedByTransport.ok, true);
  assert.deepEqual(acceptedByTransport.value, {});
  // #1849 readValidation({}) rejects this with INVALID_VALIDATION_RESPONSE;
  // its real TypeScript presenter tests are owned by #Work, not duplicated here.
});
