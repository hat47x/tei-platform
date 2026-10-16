import test from "node:test";
import assert from "node:assert/strict";
import {createPreviewScope} from "../assets/preview-scope.mjs";

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return {promise, resolve, reject};
}

test("a newer path preview discards late success and failure", async () => {
  for (const late of ["success", "failure"]) {
    const scope = createPreviewScope();
    const old = deferred();
    const shown = [];
    const stale = scope.run(() => old.promise, (value) => shown.push(value), (error) => shown.push(error));
    scope.reset(); // path changed
    const fresh = await scope.run(async () => "current", (value) => shown.push(value), (error) => shown.push(error));
    if (late === "success") old.resolve("stale"); else old.reject("stale failure");
    assert.equal(await stale, false);
    assert.equal(fresh, true);
    assert.deepEqual(shown, ["current"]);
  }
});

test("workspace change invalidates earlier requests while new request publishes", async () => {
  const scope = createPreviewScope();
  const old = deferred();
  const shown = [];
  const stale = scope.run(() => old.promise, (value) => shown.push(value), (error) => shown.push(error));
  scope.reset();
  const current = await scope.run(async () => "workspace B", (value) => shown.push(value), (error) => shown.push(error));
  old.resolve("workspace A");
  assert.equal(await stale, false);
  assert.equal(current, true);
  assert.deepEqual(shown, ["workspace B"]);
});
