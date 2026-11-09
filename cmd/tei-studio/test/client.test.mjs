import test from "node:test";
import assert from "node:assert/strict";
import {createReadOnlyClient} from "../assets/client.mjs";

test("each call carries explicit workspace ID without per-tab global selection", async () => {
  const seen = [];
  const client = createReadOnlyClient(async (...args) => {
    seen.push(args);
    return {ok: true, value: []};
  });
  await client.listWorkspaces();
  await client.catalogWorkspace("workspace-2", "ns");
  await client.validateWorkspace("workspace-1");
  await client.inspectSubject("workspace-2", "semantic", "x");
  await client.observeXead("workspace-1", "model.xead");
  await client.viewPattern("workspace-1", "order_entry", "order_entry.order_record");
  await client.viewApplication("workspace-1", "order_entry");
  await client.viewPackage("workspace-1", "order_entry", "order_entry.list_to_record");
  await client.planCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"});
  await client.executeCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"});
  await client.executeCommandWithPeriods("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}, {"sales.period": "closed"});
  await client.executeCommandWithPolicy("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}, {}, ["sales.order.submit"]);
  assert.deepEqual(seen, [
    ["main.StudioBridge.ListWorkspaces"],
    ["main.StudioBridge.CatalogWorkspace", "workspace-2", "ns"],
    ["main.StudioBridge.ValidateWorkspace", "workspace-1"],
    ["main.StudioBridge.InspectSubject", "workspace-2", "semantic", "x"],
    ["main.StudioBridge.ObserveXead", "workspace-1", "model.xead"],
    ["main.StudioBridge.ViewPattern", "workspace-1", "order_entry", "order_entry.order_record"],
    ["main.StudioBridge.ViewApplication", "workspace-1", "order_entry"],
    ["main.StudioBridge.ViewPackage", "workspace-1", "order_entry", "order_entry.list_to_record"],
    ["main.StudioBridge.PlanCommand", "workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}],
    ["main.StudioBridge.ExecuteCommand", "workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}],
    ["main.StudioBridge.ExecuteCommandWithPeriods", "workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}, {"sales.period": "closed"}],
    ["main.StudioBridge.ExecuteCommandWithPolicy", "workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", {"order": "ORD-1"}, {}, ["sales.order.submit"]],
  ]);
});
test("operation denial is not converted into transport success", async () => {
  const c = createReadOnlyClient(async () => ({ok: false, failure: {code:"WORKSPACE_NOT_ALLOWED", message:"denied"}}));
  const r = await c.validateWorkspace("unknown");
  assert.equal(r.ok, false);
  assert.equal(r.failure.code, "WORKSPACE_NOT_ALLOWED");
});
test("transport errors and malformed envelopes are distinct", async () => {
  const rejected = createReadOnlyClient(async () => {throw Error("private absolute path");});
  const invalid = createReadOnlyClient(async () => ({status:"success"}));
  assert.equal((await rejected.listWorkspaces()).failure.code, "HOST_TRANSPORT_FAILURE");
  assert.equal((await invalid.listWorkspaces()).failure.code, "INVALID_HOST_RESPONSE");
  assert.equal(JSON.stringify(await rejected.listWorkspaces()).includes("private absolute path"), false);
});





