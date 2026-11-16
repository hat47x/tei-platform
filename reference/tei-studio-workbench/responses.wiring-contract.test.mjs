// Source-level wiring guard for the selected-view lane. These assertions are
// regression guards only: they do not observe a real WebView, a Host Grant, or
// current disclosure. They check that the *real* Workbench wiring keeps the
// properties the response-scope rows describe, so a future edit cannot quietly
// bypass the lane or let a transient selection release an explicit Pin.
import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { relatedGraph, subjectKey } from "./presentation.ts";

const main = readFileSync(new URL("./main.ts", import.meta.url), "utf8");

// Anchor slices on the function signature only, so re-indentation or an extra
// blank line in main.ts does not break the guard.
function sliceFrom(start, endMarker) {
  const first = main.indexOf(start);
  assert.ok(first >= 0, `Workbench wiring not found: ${start}`);
  const last = main.indexOf(endMarker, first + start.length);
  assert.ok(last > first, `Workbench slice end not found: ${endMarker}`);
  return main.slice(first + start.length, last)
    .split("\n")
    .filter(line => !line.trimStart().startsWith("//"))
    .join("\n");
}

const inspectBody = () => sliceFrom("async function inspect(subject: Subject) {",
  "async function validate(");
const selectionHandler = () => sliceFrom(
  "document.addEventListener(INSPECTOR_SELECTION_EVENT,", "workspace.addEventListener(\"change\"");
const workspaceLoad = () => sliceFrom("async function loadWorkspace() {",
  "const panel = mountCommandPanel(");
const pinnedReader = () => sliceFrom("async function refreshPinnedSubject() {",
  "function renderList()");

test("ASYNC-WIRING-01: every Inspect reply is published through the inspection lane only", () => {
  const source = inspectBody();
  assert.match(source, /responses\.run\(\s*["']inspection["']/);
  // The lane is the single publish point for the selected view.
  assert.equal((source.match(/responses\.run\(/g) ?? []).length, 1,
    "inspect() must issue exactly one lane run");
  assert.match(source, /inspectSubject\(/);
});

test("ASYNC-WIRING-02: a Workspace change resets the lanes before the new Catalog is read", () => {
  const source = workspaceLoad();
  const epoch = source.indexOf("++workspaceEpoch");
  const reset = source.indexOf("responses.reset()");
  const catalog = source.indexOf("responses.run(\"catalog\"") >= 0
    ? source.indexOf("responses.run(\"catalog\"")
    : source.indexOf("responses.run('catalog'");
  assert.ok(epoch >= 0 && reset > epoch && catalog > reset,
    "workspaceEpoch increment, lane reset and Catalog read must keep this order");
  assert.match(source, /inspectionEpoch/);
});

test("ASYNC-WIRING-03: selection and Pin are separate lanes; selection never releases a Pin", () => {
  const handler = selectionHandler();
  const inspect = inspectBody();
  assert.match(handler, /inspect\(subject\)/);
  // The Inspector selection route may Inspect, but it must not touch Pin state.
  for (const forbidden of [/pinState/, /unpinSubject/, /pinSubject/, /invalidatePin/])
    assert.doesNotMatch(handler, forbidden,
      `Inspector selection must not mutate Pin state: ${forbidden}`);
  assert.doesNotMatch(inspect, /pinState|unpinSubject|pinSubject|invalidatePin/,
    "inspect() must not mutate Pin state");
});

test("ASYNC-WIRING-04: background Pin reader stays on its own lane", () => {
  const source = pinnedReader();
  assert.match(source, /responses\.run\(\s*["']pinned-inspection["']/);
  assert.doesNotMatch(source, /responses\.run\(\s*["']inspection["']/,
    "the pinned lane must not publish into the selected-view lane");
});

test("ASYNC-WIRING-06: a drawing edge id is never a semantic Subject identity", () => {
  // Exercise the real projector. Drawing edges carry `parent:<subject key>`,
  // which must never coincide with a Subject key, so a drawn parent edge cannot
  // become an Inspect or Pin target through the graph selection path.
  const parent = { kind: "semantic-data", id: "sales.customer" };
  const child = { kind: "semantic-data", id: "sales.order" };
  const entries = [
    { subject: parent, sourceStatus: "indexed", ambiguous: false },
    { subject: child, parent, sourceStatus: "indexed", ambiguous: false },
  ];
  const projected = relatedGraph(entries, child);
  assert.equal(projected.input.edges.length, 1);
  const edge = projected.input.edges[0];
  assert.equal(edge.id, `parent:${subjectKey(child)}`);
  assert.notEqual(edge.id, subjectKey(child));
  assert.notEqual(edge.id, subjectKey(parent));
  assert.ok(!entries.some((e) => subjectKey(e.subject) === edge.id),
    "a drawing edge id must not resolve to any Catalog Subject");

  // The wiring resolves a graph selection only through Catalog entries, so the
  // edge id above finds no entry and emits nothing.
  const graphMount = sliceFrom("const graph = mountReadonlyGraph(", "function unwrap(reply");
  assert.match(graphMount, /entries\.find\(/);
  assert.match(graphMount, /subjectKey\(e\.subject\) === key/);
  assert.match(graphMount, /if \(entry\)\s*emitInspectorSelection/,
    "graph selection must be gated on a Catalog entry lookup");
});

test("ASYNC-WIRING-05: disposal invalidates every lane and both epochs", () => {
  const source = sliceFrom("pagehide", "{ once: true }");
  assert.match(source, /responses\.dispose\(\)/);
  assert.match(source, /disposed = true/);
  assert.match(source, /\+\+workspaceEpoch/);
  assert.match(source, /\+\+inspectionEpoch/);
});
