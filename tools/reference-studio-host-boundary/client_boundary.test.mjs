import test from "node:test";
import assert from "node:assert/strict";
import {createReadOnlyClient} from "../../cmd/tei-studio/assets/client.mjs";

// The browser adapter is not a security/authorization decision point.
// These checks use #1832's ACTUAL client.mjs imported in a fixed snapshot.
// An untrusted caller can still invoke runtime methods directly; the Go
// StudioBridge and Host must enforce the public capability boundary.
test("Studio browser port advertises only four read-only methods", async () => {
  const calls = [];
  const port = createReadOnlyClient(async (name, ...args) => {
    calls.push([name, ...args]);
    return {ok: true, value: []};
  });
  assert.deepEqual(Object.keys(port).sort(), [
    "catalogWorkspace", "inspectSubject", "listWorkspaces", "validateWorkspace",
  ]);
  assert.equal(Object.isFrozen(port), true);
  for (const name of ["save", "write", "accept", "execute", "runCommand", "readSource"]) {
    assert.equal(Object.hasOwn(port, name), false, `unexpected mutation capability: ${name}`);
  }
  await port.listWorkspaces();
  await port.catalogWorkspace("workspace-1");
  await port.validateWorkspace("workspace-1");
  await port.inspectSubject("workspace-1", "semantic", "x");
  assert.deepEqual(calls.map(([name]) => name), [
    "main.StudioBridge.ListWorkspaces",
    "main.StudioBridge.CatalogWorkspace",
    "main.StudioBridge.ValidateWorkspace",
    "main.StudioBridge.InspectSubject",
  ]);
});

// The client forwards an untrusted workspace ID, and therefore cannot be
// credited with independently authorizing it. Verify it does not convert
// the Host's denial into success.
test("Host denial on unknown workspace remains a refusal in the browser port", async () => {
  const calls = [];
  const port = createReadOnlyClient(async (name, ...args) => {
    calls.push([name, ...args]);
    return {ok: false, failure: {
      code: "WORKSPACE_NOT_ALLOWED", message: "workspace ID is not in this host's allowlist",
    }};
  });
  const result = await port.validateWorkspace("../../outside");
  assert.equal(result.ok, false);
  assert.equal(result.failure.code, "WORKSPACE_NOT_ALLOWED");
  assert.equal(result.value, undefined);
  assert.deepEqual(calls, [["main.StudioBridge.ValidateWorkspace", "../../outside"]]);
});

// A thrown Host/transport exception may include OS paths or secret details.
// The browser adapter must not echo its raw exception into displayed text.
test("Host transport exception cannot echo private paths in UI response", async () => {
  const secret = "PRIVATE_HOST_ROOT_SENTINEL\\C:\\Users\\secret";
  const port = createReadOnlyClient(async () => {
    throw new Error(`Wails invocation failed: ${secret}`);
  });
  const result = await port.catalogWorkspace("workspace-1");
  assert.equal(result.ok, false);
  assert.equal(result.failure.code, "HOST_TRANSPORT_FAILURE");
  assert.equal(JSON.stringify(result).includes("PRIVATE_HOST_ROOT_SENTINEL"), false);
});
