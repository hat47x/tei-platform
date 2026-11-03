import test from "node:test";
import assert from "node:assert/strict";
import {createQualifiedSelectionGuard} from "./qualified-selection.mjs";
import {normalizeGraph, GraphViewError} from "./graph.ts";

const source = modelRevision => ({workspaceId: "workspace-w", modelRevision});
const binding = (modelRevision, scope, viewId = "K") => ({
  ...source(modelRevision),
  bindings: [{viewId, scope, kind: "semantic-data", id: "sales.order"}],
});
const graph = (viewId = "K") => ({
  width: 300, height: 200,
  nodes: [{id: viewId, label: "order", x: 80, y: 80}],
  edges: [],
});

test("different scopes and repeated occurrences retain qualified identity", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const input = {
    ...source("R10"),
    bindings: [
      {viewId: "diagram-A:K", scope: "scope-one", kind: "semantic-data", id: "sales.order"},
      {viewId: "diagram-B:K", scope: "scope-one", kind: "semantic-data", id: "sales.order"},
      {viewId: "diagram-C:K", scope: "scope-two", kind: "semantic-data", id: "sales.order"},
    ],
  };
  const token = guard.stage(input);
  assert.deepEqual(guard.select("diagram-A:K"), {status: "stale"});
  input.bindings[0].scope = "attacker-mutated";
  const threeNodes = {...graph("diagram-A:K"), nodes: [
    {id: "diagram-A:K", label: "A", x: 50, y: 50},
    {id: "diagram-B:K", label: "B", x: 150, y: 50},
    {id: "diagram-C:K", label: "C", x: 250, y: 50},
  ]};
  assert.deepEqual(guard.publishGraph(token, {update: normalizeGraph}, threeNodes),
    {status: "published"});
  assert.deepEqual(guard.select("diagram-A:K"), {status: "resolved", ref: {
    scope: "scope-one", subject: {kind: "semantic-data", id: "sales.order"},
  }});
  assert.deepEqual(guard.select("diagram-B:K"), guard.select("diagram-A:K"));
  assert.equal(guard.select("diagram-C:K").ref.scope, "scope-two");
  assert.deepEqual(guard.select("unknown"), {status: "unbound"});
  guard.dispose();
});

test("failed graph replacement never publishes a new map while R10 remains current", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  assert.deepEqual(guard.publishGraph(guard.stage(binding("R10", "scope-one")),
    {update: normalizeGraph}, graph()), {status: "published"});
  const attempted = guard.stage(binding("R10", "scope-two"));
  const bad = {...graph(), edges: [{id: "e", source: "K", target: "not-there"}]};
  assert.throws(() => guard.publishGraph(attempted, {update: normalizeGraph}, bad),
    err => err instanceof GraphViewError && err.code === "missing_endpoint");
  // Cannot independently prove an atomic display after the failed attempt:
  // the old visual may survive, but it is not navigable until revalidated.
  assert.deepEqual(guard.select("K"), {status: "stale"});
  assert.throws(() => guard.publish(attempted, () => normalizeGraph(graph())),
    /unknown_selection_candidate/);
});

test("R10 to R11 invalidates G1 selection even if G2 render fails", () => {
  const guard = createQualifiedSelectionGuard();
  let visible = "G1";
  let inspectCalls = 0;
  guard.observeSource(source("R10"));
  guard.publishGraph(guard.stage(binding("R10", "scope-one")),
    {update: normalizeGraph}, graph());
  const oldSelected = guard.select("K");
  assert.equal(oldSelected.ref.scope, "scope-one");

  guard.observeSource(source("R11")); // same workspace, newer Source
  const g2 = guard.stage(binding("R11", "scope-two"));
  assert.throws(() => guard.publishGraph(g2, {update: normalizeGraph}, {
    ...graph(), edges: [{id: "bad", source: "K", target: "absent"}],
  }), /missing_endpoint/);
  if (guard.select("K").status === "resolved") inspectCalls++;
  assert.equal(visible, "G1");
  assert.deepEqual(guard.select("K"), {status: "stale"});
  assert.equal(inspectCalls, 0);
  // Previously returned read-only ref is a historical value, not current authority.
  assert.equal(oldSelected.ref.scope, "scope-one");
});

test("staged candidates cannot be reused across unknown source or ABA revisions", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const old = guard.stage(binding("R10", "scope-one"));
  guard.observeSource(source("R11"));
  guard.observeSource(source("R10"));
  let called = 0;
  assert.deepEqual(guard.publish(old, () => { called++; }), {status: "stale"});
  assert.equal(called, 0);
  const current = guard.stage(binding("R10", "scope-two"));
  guard.observeSource(null);
  assert.deepEqual(guard.publish(current, () => { called++; }), {status: "stale"});
  assert.equal(called, 0);
  assert.deepEqual(guard.select("K"), {status: "stale"});
});

test("source changing synchronously while rendering prevents map publication", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const token = guard.stage(binding("R10", "scope-one"));
  assert.deepEqual(guard.publish(token, () => {
    normalizeGraph(graph());
    guard.observeSource(source("R11"));
  }), {status: "stale"});
  assert.deepEqual(guard.select("K"), {status: "stale"});
});

test("invalid/unqualified, duplicate and unobserved candidate bindings fail closed", () => {
  const guard = createQualifiedSelectionGuard();
  assert.throws(() => guard.stage(binding("R10", "one")), /stale_selection_snapshot/);
  guard.observeSource(source("R10"));
  assert.throws(() => guard.stage(binding("R11", "one")), /stale_selection_snapshot/);
  assert.throws(() => guard.stage(binding("R10", "")), /invalid_qualified_key/);
  assert.throws(() => guard.stage({...source("R10"), bindings: [
    {viewId: "K", scope: "one", kind: "semantic-data", id: "sales.order"},
    {viewId: "K", scope: "two", kind: "semantic-data", id: "sales.order"},
  ]}), /duplicate_view_id/);
  const p = guard.stage(binding("R10", "one"));
  assert.throws(() => guard.publish(p, null), /invalid_render/);
  guard.invalidate();
  assert.deepEqual(guard.select("K"), {status: "stale"});
  guard.dispose();
  assert.throws(() => guard.observeSource(source("R10")), /selection_guard_disposed/);
});

test("a binding not present in displayed Graph nodes is rejected before update", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const token = guard.stage({...source("R10"), bindings: [
    {viewId: "diagram-A:K", scope: "one", kind: "semantic-data", id: "sales.order"},
    {viewId: "diagram-B:K", scope: "two", kind: "semantic-data", id: "sales.order"},
  ]});
  let updates = 0;
  assert.throws(() => guard.publishGraph(token, {
    update() { updates++; },
  }, graph("diagram-A:K")), /binding_not_rendered/);
  assert.equal(updates, 0);
  assert.deepEqual(guard.select("diagram-A:K"), {status: "stale"});
});

test("the Graph can show intentionally unbound nodes without granting inspect", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const withDecoration = {...graph("K"), nodes: [
    ...graph("K").nodes,
    {id: "annotation", label: "unbound", x: 200, y: 80},
  ]};
  assert.deepEqual(guard.publishGraph(guard.stage(binding("R10", "one")),
    {update: normalizeGraph}, withDecoration), {status: "published"});
  assert.equal(guard.select("K").ref.scope, "one");
  assert.deepEqual(guard.select("annotation"), {status: "unbound"});
});

test("async render callback cannot publish a binding", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const token = guard.stage(binding("R10", "one"));
  assert.throws(() => guard.publish(token, () => Promise.resolve()), /async_render_unsupported/);
  assert.deepEqual(guard.select("K"), {status: "stale"});
});

test("graph view must have a synchronous update method", () => {
  const guard = createQualifiedSelectionGuard();
  guard.observeSource(source("R10"));
  const token = guard.stage(binding("R10", "one"));
  assert.throws(() => guard.publishGraph(token, null, graph()), /invalid_graph_view/);
  assert.deepEqual(guard.select("K"), {status: "stale"});
});
