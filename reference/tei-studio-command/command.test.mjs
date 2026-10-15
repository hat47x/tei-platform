import assert from "node:assert/strict";
import { test } from "node:test";
import { createCommandPort, mountCommandPanel } from "./command.ts";

const commands = [
  { id: "inspect", label: "Inspect" },
  { id: "hidden", label: "Disabled", enabled: false },
];

test("success is reported only from an explicit Host result", async () => {
  const calls = [];
  const port = createCommandPort({
    commands,
    invoke: async (id, signal) => {
      calls.push([id, signal.aborted]);
      return { kind: "success", message: "done" };
    },
  });
  assert.equal(await port.execute("inspect"), true);
  assert.deepEqual(calls, [["inspect", false]]);
  assert.deepEqual(port.snapshot().status, { phase: "success", commandId: "inspect", message: "done" });
  port.dispose();
});

test("denied and failed outcomes remain distinct from success", async () => {
  let outcome = { kind: "denied", message: "permission refused" };
  const port = createCommandPort({ commands, invoke: async () => outcome });
  await port.execute("inspect");
  assert.equal(port.snapshot().status.phase, "denied");
  outcome = { kind: "failure", message: "upstream unavailable" };
  await port.execute("inspect");
  assert.equal(port.snapshot().status.phase, "failure");
  assert.equal(port.snapshot().status.message, "upstream unavailable");
  port.dispose();
});

test("exceptions and malformed Host results fail without leaking exception details", async () => {
  const port = createCommandPort({
    commands,
    invoke: async () => { throw new Error("secret workspace path"); },
  });
  await port.execute("inspect");
  assert.deepEqual(port.snapshot().status, {
    phase: "failure", commandId: "inspect", message: "実行に失敗しました。",
  });
  port.dispose();

  const malformed = createCommandPort({
    commands,
    invoke: async () => ({ kind: "unknown", message: "not success" }),
  });
  await malformed.execute("inspect");
  assert.equal(malformed.snapshot().status.phase, "failure");
  malformed.dispose();
});

test("unknown and visually disabled commands never dispatch", async () => {
  let called = 0;
  const port = createCommandPort({ commands, invoke: async () => { called++; return { kind: "success" }; } });
  assert.equal(await port.execute("missing"), false);
  assert.equal(await port.execute("hidden"), false);
  assert.equal(called, 0);
  assert.equal(port.snapshot().status.phase, "idle");
  port.dispose();
});

test("running state blocks double dispatch and late completion after dispose", async () => {
  let finish;
  let signal;
  let calls = 0;
  const port = createCommandPort({
    commands,
    invoke: (_id, s) => {
      signal = s;
      calls++;
      return new Promise((resolve) => { finish = resolve; });
    },
  });
  const states = [];
  port.subscribe((s) => { states.push(s.status.phase); });
  const pending = port.execute("inspect");
  assert.equal(port.snapshot().status.phase, "running");
  assert.equal(await port.execute("inspect"), false);
  assert.equal(calls, 1);
  port.dispose();
  assert.equal(signal.aborted, true);
  finish({ kind: "success", message: "late" });
  assert.equal(await pending, false);
  assert.deepEqual(states, ["idle", "running"]);
});

test("sets reject collisions and invalid data without changing previous snapshot", () => {
  const port = createCommandPort({ commands, invoke: async () => ({ kind: "success" }) });
  assert.throws(() => port.setCommands([{ id: "a", label: "A" }, { id: "a", label: "B" }]), /duplicate/);
  assert.throws(() => port.setCommands([{ id: " ", label: "A" }]), /invalid/);
  assert.equal(port.snapshot().commands.length, 2);
  assert.throws(() => port.setDiagnostics([{ id: "d", severity: "unknown", message: "bad" }]), /invalid/);
  port.setDiagnostics([{ id: "warning-1", severity: "warning", message: "check" }]);
  assert.equal(port.snapshot().diagnostics[0].severity, "warning");
  const snap = port.snapshot();
  snap.commands[0].label = "tampered";
  assert.equal(port.snapshot().commands[0].label, "Inspect");
  port.dispose();
  assert.throws(() => port.setCommands([]), /disposed/);
});

test("subscriber failure cannot change a Host success to UI failure", async () => {
  const port = createCommandPort({ commands, invoke: async () => ({ kind: "success" }) });
  port.subscribe(() => { throw new Error("subscriber failure"); });
  await port.execute("inspect");
  assert.equal(port.snapshot().status.phase, "success");
  port.dispose();
});

class FakeElement {
  constructor(tag, doc) {
    this.tag = tag;
    this.ownerDocument = doc;
    this.dataset = {};
    this.attributes = {};
    this.children = [];
    this.listeners = new Map();
    this.textContent = "";
    this.value = "";
    this.disabled = false;
    this.parent = null;
  }
  setAttribute(name, value) { this.attributes[name] = String(value); }
  append(...nodes) {
    for (const node of nodes) { node.parent = this; this.children.push(node); }
  }
  replaceChildren(...nodes) {
    for (const node of this.children) node.parent = null;
    this.children = [];
    this.append(...nodes);
  }
  addEventListener(name, fn) {
    if (!this.listeners.has(name)) this.listeners.set(name, new Set());
    this.listeners.get(name).add(fn);
  }
  removeEventListener(name, fn) { this.listeners.get(name)?.delete(fn); }
  emit(name, event = {}) {
    for (const fn of this.listeners.get(name) ?? []) fn(event);
  }
  querySelectorAll(selector) {
    assert.equal(selector, "button");
    const out = [];
    const visit = (node) => {
      for (const child of node.children) {
        if (child.tag === selector) out.push(child);
        visit(child);
      }
    };
    visit(this);
    return out;
  }
  focus() { this.ownerDocument.activeElement = this; }
  remove() {
    if (this.parent) this.parent.children = this.parent.children.filter((node) => node !== this);
    this.parent = null;
  }
}
function fakeTarget() {
  const doc = { activeElement: null };
  doc.createElement = (tag) => new FakeElement(tag, doc);
  return new FakeElement("main", doc);
}

test("DOM panel renders command/diagnostic as text, filters and delegates clicks", async () => {
  const target = fakeTarget();
  const calls = [];
  const panel = mountCommandPanel(target, {
    commands: [{ id: "view", label: "<svg onload=evil()>", description: "inspect only" },
      { id: "second", label: "Second" }],
    diagnostics: [{ id: "d1", severity: "error", message: "<script>unsafe</script>" }],
    invoke: async (id) => { calls.push(id); return { kind: "denied", message: "host refused" }; },
  });
  const [search, list, status, diagnostics] = panel.element.children;
  assert.equal(list.children[0].children[0].textContent, "<svg onload=evil()>");
  assert.equal(diagnostics.children[0].textContent, "error: <script>unsafe</script>");
  assert.equal(list.children.length, 2);
  search.value = "second";
  search.emit("input");
  assert.equal(list.children.length, 1);
  list.querySelectorAll("button")[0].emit("click");
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(calls, ["second"]);
  assert.match(status.textContent, /^拒否:/);
  assert.equal(panel.element.attributes["aria-busy"], "false");
  panel.dispose();
  assert.equal(target.children.length, 0);
});

test("keyboard arrows focus available commands; disposed panel refuses commands", async () => {
  const target = fakeTarget();
  const panel = mountCommandPanel(target, { commands, invoke: async () => ({ kind: "success" }) });
  const list = panel.element.children[1];
  let prevented = false;
  list.emit("keydown", { key: "ArrowDown", preventDefault: () => { prevented = true; } });
  assert.equal(target.ownerDocument.activeElement.dataset.commandId, "inspect");
  assert.equal(prevented, true);
  panel.dispose();
  assert.equal(await panel.execute("inspect"), false);
});

test("Abort via beginContext suppresses a late Host success in rendered UI", async () => {
  const target = fakeTarget();
  let finish;
  let signal;
  const panel = mountCommandPanel(target, {
    commands,
    invoke: (_id, currentSignal) => {
      signal = currentSignal;
      return new Promise(resolve => { finish = resolve; });
    },
  });
  const events = [];
  panel.subscribe(snapshot => events.push(snapshot.status.phase));
  const context = panel.beginContext();
  panel.setDiagnosticsForContext(context, [{ id: "old", severity: "warning", message: "old warning" }]);
  const pending = panel.execute("inspect");
  const [, , statusNode, diagnosticList] = panel.element.children;
  assert.equal(statusNode.textContent, "実行中…");
  assert.equal(diagnosticList.children.length, 1);

  panel.beginContext(); // UI has switched context while Host is still pending.
  assert.equal(signal.aborted, true);
  assert.equal(statusNode.textContent, "");
  assert.equal(diagnosticList.children.length, 0);
  const state = panel.snapshot();
  const eventCount = events.length;
  finish({ kind: "success", message: "LATE_OLD_WORKSPACE_SUCCESS" }); // Host did not cancel.
  assert.equal(await pending, false);
  assert.deepEqual(panel.snapshot(), state);
  assert.equal(events.length, eventCount, "late Host success must not notify the UI");
  assert.equal(statusNode.textContent, "");
  assert.equal(diagnosticList.children.length, 0);
  panel.dispose();
});

test("dispose suppresses a late rejected Host result and never remounts UI", async () => {
  const target = fakeTarget();
  let fail;
  let signal;
  const panel = mountCommandPanel(target, {
    commands,
    invoke: (_id, currentSignal) => {
      signal = currentSignal;
      return new Promise((_resolve, reject) => { fail = reject; });
    },
  });
  const phases = [];
  panel.subscribe(snapshot => phases.push(snapshot.status.phase));
  const pending = panel.execute("inspect");
  assert.equal(panel.element.children[2].textContent, "実行中…");
  panel.dispose();
  assert.equal(signal.aborted, true);
  assert.equal(target.children.length, 0);
  const snapshotAfterDispose = panel.snapshot();
  const count = phases.length;
  fail(new Error("SECRET_OLD_HOST_FAILURE")); // The Host can still settle its Promise.
  assert.equal(await pending, false);
  assert.deepEqual(panel.snapshot(), snapshotAfterDispose);
  assert.equal(phases.length, count, "late failure must not notify disposed subscribers");
  assert.equal(target.children.length, 0);
  assert.equal(JSON.stringify(panel.snapshot()).includes("SECRET_OLD_HOST_FAILURE"), false);
});
