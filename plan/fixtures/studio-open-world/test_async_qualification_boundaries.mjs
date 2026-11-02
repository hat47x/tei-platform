/* Delayed-response/qualification pressure using the real Workbench response-scope.
   Node >=22 with --experimental-strip-types (reference package specifies Node >=24).
   These tests do not install a policy listener, authenticate, or run a browser. */
import test from "node:test";
import assert from "node:assert/strict";
import { createResponseScope } from "../../../reference/tei-studio-workbench/responses.ts";

function deferred() {
  let resolve, reject;
  const promise = new Promise((ok, fail) => { resolve = ok; reject = fail; });
  return { promise, resolve, reject };
}
const viewer = (value, policy, expectedVersion, sink) => {
  // Test-local consumer guard: not an implementation of TEI Authorization.
  if (policy.version !== expectedVersion || !policy.mayDisclose) return;
  sink.push(value);
};

test("AQ-01: explicitly invalidated inspection lane drops a late value", async () => {
  const scope = createResponseScope(), held = deferred(), seen = [];
  const result = scope.run("inspection", () => held.promise, value => seen.push(value),
    error => { throw error; });
  scope.invalidate("inspection");
  held.resolve({ secret: "old permission" });
  assert.equal(await result, false);
  assert.deepEqual(seen, []);
  scope.dispose();
});

test("AQ-02: unannounced grant change does not invalidate the response lane", async () => {
  const scope = createResponseScope(), held = deferred(), seen = [];
  const policy = { version: 1, mayDisclose: true };
  const result = scope.run("inspection", () => held.promise, value => seen.push(value),
    error => { throw error; });
  policy.version = 2; policy.mayDisclose = false;
  held.resolve({ secret: "unqualified" });
  assert.equal(await result, true); // Deliberately documents a missing policy signal.
  assert.equal(seen.length, 1);
  scope.dispose();
});

test("AQ-03: invalidating one lane does not invalidate another lane", async () => {
  const scope = createResponseScope(), x = deferred(), y = deferred(), seen = [];
  const inspect = scope.run("inspection", () => x.promise, () => seen.push("inspect"), () => {});
  const xead = scope.run("xead", () => y.promise, () => seen.push("xead"), () => {});
  scope.invalidate("inspection");
  x.resolve({}); y.resolve({});
  assert.equal(await inspect, false);
  assert.equal(await xead, true);
  assert.deepEqual(seen, ["xead"]);
  scope.dispose();
});

test("AQ-04: a full response reset drops late values from every lane", async () => {
  const scope = createResponseScope(), x = deferred(), y = deferred(), seen = [];
  const a = scope.run("catalog", () => x.promise, () => seen.push("catalog"), () => {});
  const b = scope.run("inspection", () => y.promise, () => seen.push("inspection"), () => {});
  scope.reset(); x.resolve({}); y.resolve({});
  assert.equal(await a, false); assert.equal(await b, false);
  assert.deepEqual(seen, []);
  scope.dispose();
});

test("AQ-05: reset alone does not erase already rendered values", async () => {
  const scope = createResponseScope(), rendered = [];
  await scope.run("inspection", async () => ({ secret: "previously visible" }),
    v => rendered.push(v), () => {});
  scope.reset();
  assert.equal(rendered.length, 1); // A consumer must separately retract its view/cache.
  scope.dispose();
});

test("AQ-06: explicit render-commit recheck can drop a revoked value", async () => {
  const scope = createResponseScope(), held = deferred(), rendered = [];
  const policy = { version: 1, mayDisclose: true };
  const expected = policy.version;
  const result = scope.run("inspection", () => held.promise,
    value => viewer(value, policy, expected, rendered), () => {});
  policy.version = 2; policy.mayDisclose = false;
  held.resolve({ secret: "not eligible at commit" });
  assert.equal(await result, true); // The response scope is not a policy decision.
  assert.deepEqual(rendered, []);
  scope.dispose();
});

test("AQ-07: a new request may render only a currently disclosable projection", async () => {
  const scope = createResponseScope(), rendered = [];
  const policy = { version: 3, mayDisclose: true };
  await scope.run("inspection", async () => ({ public: 10 }),
    value => viewer(value, policy, 3, rendered), () => {});
  assert.deepEqual(rendered, [{ public: 10 }]);
  scope.dispose();
});

test("AQ-08: plugin/role meaning change is independent of response-scope ordering", async () => {
  const scope = createResponseScope(), held = deferred(), rendered = [];
  const role = { pin: "std.visibility@v1" }, requested = role.pin;
  const pending = scope.run("inspection", () => held.promise,
    value => { if (role.pin === requested) rendered.push(value); }, () => {});
  role.pin = "std.visibility@v2";
  held.resolve({ value: false });
  assert.equal(await pending, true);
  assert.deepEqual(rendered, []); // Role/source meaning needs its own qualification.
  scope.dispose();
});

test("AQ-09: even the failure callback is stale after lane invalidation", async () => {
  const scope = createResponseScope(), held = deferred(), failures = [];
  const pending = scope.run("inspection", () => held.promise, () => {},
    e => failures.push(String(e)));
  scope.invalidate("inspection");
  held.reject(Error("stale diagnosis"));
  assert.equal(await pending, false);
  assert.deepEqual(failures, []);
  scope.dispose();
});

test("AQ-10: disposal prevents future requests from publishing", async () => {
  const scope = createResponseScope(), seen = [];
  scope.dispose();
  const accepted = await scope.run("inspection", async () => ({ x: 1 }),
    value => seen.push(value), () => {});
  assert.equal(accepted, false);
  assert.deepEqual(seen, []);
});

test("AQ-11: old view must be cleared on policy withdrawal independently", async () => {
  const scope = createResponseScope(), dom = { text: "" };
  await scope.run("inspection", async () => ({ detail: "secret" }),
    v => { dom.text = v.detail; }, () => {});
  const policy = { mayDisclose: false };
  scope.reset();
  assert.equal(dom.text, "secret"); // Shows exact limit of ordering fence.
  if (!policy.mayDisclose) dom.text = ""; // Consumer-side explicit retraction.
  assert.equal(dom.text, "");
  scope.dispose();
});

test("AQ-12: same structure and revision, different grant is not the same display entitlement", () => {
  const publicSlice = { id: "O1", status: "draft" };
  const high = { ...publicSlice, internalNote: "restricted" };
  const project = (value, allowed) => Object.fromEntries(
    Object.entries(value).filter(([name]) => allowed.includes(name)));
  assert.deepEqual(project(high, ["id", "status"]), publicSlice);
  assert.notDeepEqual(project(high, ["id", "status", "internalNote"]), publicSlice);
});
