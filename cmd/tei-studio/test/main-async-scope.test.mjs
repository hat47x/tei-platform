// Headless DOM/RPC characterization of the *actual* Studio bootstrap, not just a scope helper.
import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {runInNewContext} from "node:vm";
import {createReadOnlyClient} from "../assets/client.mjs";
import {createPreviewScope} from "../assets/preview-scope.mjs";
import {summarizeView} from "../assets/view-format.mjs";

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return {promise, resolve, reject};
}
class Element {
  constructor(id = "") {
    this.id = id;
    this.textContent = "";
    this.value = "";
    this.disabled = true;
    this.children = [];
    this.handlers = new Map();
  }
  addEventListener(event, handler) {
    const list = this.handlers.get(event) || [];
    list.push(handler);
    this.handlers.set(event, list);
  }
  emit(event) {
    return (this.handlers.get(event) || []).map(handler => handler());
  }
  appendChild(child) { this.children.push(child); }
  replaceChildren(...children) { this.children = children; this.textContent = ""; }
}

async function boot() {
  const elements = new Map();
  const get = id => {
    if (!elements.has(id)) elements.set(id, new Element(id.replace(/^#/, "")));
    return elements.get(id);
  };
  get("#workspace").value = "W1";
  const document = {
    querySelector: get,
    getElementById: id => get("#" + id),
    createElement: tag => new Element(tag),
  };
  const calls = [];
  const Call = {ByName(method, ...args) {
    if (method.endsWith(".ListWorkspaces")) {
      return Promise.resolve({ok: true, value: [
        {id: "W1", label: "Alpha"}, {id: "W2", label: "Beta"},
      ]});
    }
    const task = deferred();
    calls.push({method, args, ...task});
    return task.promise;
  }};
  const src = readFileSync(new URL("../assets/main.js", import.meta.url), "utf8")
    .replace(/^\uFEFF/, "")
    .replace(/^import .*;\r?\n/gm, "");
  // The real ESM bootstrap contains a top-level await. Wrap it to run with
  // an injected Wails bridge and a minimal deterministic DOM.
  await runInNewContext("(async () => {\n" + src + "\n})()", {
    document, Call, createReadOnlyClient, createPreviewScope, summarizeView,
  });
  return {
    calls,
    element: id => get("#" + id),
    fire: (id, event = "click") => get("#" + id).emit(event)[0],
  };
}
const success = value => ({ok: true, value});

test("Command: W1 late success and finally cannot overwrite W2 in-flight/result", async () => {
  const h = await boot();
  const first = h.fire("execute-command");
  assert.equal(h.calls.length, 1);
  assert.equal(h.calls[0].args[0], "W1");
  h.element("workspace").value = "W2";
  h.fire("workspace", "change");
  assert.equal(h.element("command-output").textContent, "");
  const second = h.fire("execute-command");
  assert.equal(h.calls[1].args[0], "W2");
  h.calls[0].resolve(success({mode: "reference", marker: "OLD"}));
  await first;
  assert.equal(h.element("execute-command").disabled, true, "old finally must not enable busy W2");
  assert.doesNotMatch(h.element("command-output").textContent, /OLD/);
  h.calls[1].resolve(success({mode: "reference", marker: "NEW"}));
  await second;
  assert.match(h.element("command-output").textContent, /NEW/);
  assert.equal(h.element("execute-command").disabled, false);
});

test("Command: AllowList changes A→B→A invalidate old response and denial", async () => {
  const h = await boot();
  h.element("command-allowed").value = "sales.a";
  const first = h.fire("plan-command");
  h.element("command-allowed").value = "sales.b";
  h.fire("command-allowed", "input");
  h.element("command-allowed").value = "sales.a";
  h.fire("command-allowed", "input");
  const second = h.fire("plan-command");
  assert.equal(h.calls.length, 2);
  h.calls[0].reject(new Error("private old error"));
  await first;
  assert.equal(h.element("plan-command").disabled, true);
  assert.doesNotMatch(h.element("command-state").textContent, /HOST_TRANSPORT_FAILURE/);
  h.calls[1].resolve(success({capability: "fresh-plan"}));
  await second;
  assert.match(h.element("command-output").textContent, /fresh-plan/);
  assert.equal(h.element("plan-command").disabled, false);
});

test("View: workspace/parameter changes reject late results within the lane", async () => {
  const h = await boot();
  const first = h.fire("view-application");
  h.element("workspace").value = "W2";
  h.fire("workspace", "change");
  const second = h.fire("view-pattern");
  h.calls[0].resolve(success({application: "OLD", patterns: []}));
  await first;
  assert.equal(h.element("view-pattern").disabled, true);
  assert.doesNotMatch(h.element("view-state").textContent, /application：ビュー応答/);
  h.calls[1].resolve(success({application: "NEW", patterns: []}));
  await second;
  assert.match(h.element("view-state").textContent, /pattern：ビュー応答/);
  assert.equal(h.element("view-pattern").disabled, false);
  h.element("pattern-id").value = "changed";
  h.fire("pattern-id", "input");
  assert.equal(h.element("view-state").textContent, "");
  assert.equal(h.element("view-output").children.length, 0);
});

test("Workbench: old catalog cannot overwrite a new inspect response", async () => {
  const h = await boot();
  const first = h.fire("catalog");
  h.element("subject-id").value = "new-subject";
  h.fire("subject-id", "input");
  const second = h.fire("inspect");
  assert.equal(h.calls[0].args[0], "W1");
  assert.equal(h.calls[1].args[2], "new-subject");
  h.calls[0].resolve(success({marker: "OLD"}));
  await first;
  assert.equal(h.element("inspect").disabled, true);
  assert.doesNotMatch(h.element("output").textContent, /OLD/);
  h.calls[1].resolve(success({marker: "NEW"}));
  await second;
  assert.match(h.element("output").textContent, /NEW/);
  assert.equal(h.element("inspect").disabled, false);
});

test("Same-lane rapid second click is suppressed; unrelated preview edits do not unlock it", async () => {
  const h = await boot();
  const first = h.fire("execute-command");
  h.fire("execute-command");
  assert.equal(h.calls.length, 1, "synchronous reentry must not dispatch twice");
  h.element("xead-path").value = "other.xead";
  h.fire("xead-path", "input");
  assert.equal(h.element("execute-command").disabled, true);
  h.calls[0].resolve(success({mode: "reference"}));
  await first;
  assert.equal(h.element("execute-command").disabled, false);
});
