import test from "node:test";
import assert from "node:assert/strict";
import {mountReadonlyGraph, normalizeGraph, GraphViewError} from "./graph.ts";
import {buildPipelineGraph} from "./sample-graph.mjs";

// Tiny local DOM substitute: verifies the module's DOM calls, not a real browser.
class FakeElement {
  constructor(tag, doc) {
    this.tagName = tag;
    this.ownerDocument = doc;
    this.parentNode = null;
    this.children = [];
    this.attributes = new Map();
    this.listeners = new Map();
    this.textContent = "";
  }
  setAttribute(name, value) { this.attributes.set(name, String(value)); }
  getAttribute(name) { return this.attributes.get(name) ?? null; }
  addEventListener(name, listener) {
    const listeners = this.listeners.get(name) ?? [];
    listeners.push(listener);
    this.listeners.set(name, listeners);
  }
  appendChild(element) {
    element.remove();
    element.parentNode = this;
    this.children.push(element);
    return element;
  }
  replaceChildren(...elements) {
    for (const child of this.children) child.parentNode = null;
    this.children = [];
    for (const child of elements) this.appendChild(child);
  }
  remove() {
    if (!this.parentNode) return;
    const parent = this.parentNode;
    parent.children = parent.children.filter(child => child !== this);
    this.parentNode = null;
  }
  dispatch(name, key) {
    let prevented = false;
    for (const listener of this.listeners.get(name) ?? []) {
      listener({key, preventDefault() { prevented = true; }});
    }
    return prevented;
  }
}
function container() {
  const doc = {
    createElement(tag) { return new FakeElement(tag, doc); },
    createElementNS(_ns, tag) { return new FakeElement(tag, doc); },
  };
  return doc.createElement("main");
}
function all(root, tag) {
  return [root, ...root.children.flatMap(c => all(c, tag))]
    .filter(e => e.tagName === tag);
}
function fails(code, run) {
  assert.throws(run, err => err instanceof GraphViewError && err.code === code);
}

test("graph data is copied and validates identities and endpoints", () => {
  const graph = normalizeGraph(buildPipelineGraph);
  assert.deepEqual(graph, buildPipelineGraph);
  assert.notStrictEqual(graph.nodes, buildPipelineGraph.nodes);
  assert.notStrictEqual(graph.nodes[0], buildPipelineGraph.nodes[0]);
  fails("duplicate_node", () => normalizeGraph({...graph, nodes: [...graph.nodes, graph.nodes[0]]}));
  fails("duplicate_edge", () => normalizeGraph({...graph, edges: [...graph.edges, graph.edges[0]]}));
  fails("missing_endpoint", () => normalizeGraph({
    ...graph, edges: [{id: "not-there", source: "source", target: "outside"}],
  }));
  fails("invalid_node", () => normalizeGraph({
    ...graph, nodes: [{id: " spaced ", label: "x", x: 0, y: 0}],
  }));
  fails("invalid_node", () => normalizeGraph({
    ...graph, nodes: [{id: "n", label: "x", x: Infinity, y: 0}],
  }));
  fails("unsupported_self_edge", () => normalizeGraph({
    ...graph, edges: [{id: "loop", source: "source", target: "source"}],
  }));
  fails("invalid_graph", () => normalizeGraph({...graph, width: NaN}));
});

test("mount renders text without HTML interpretation and allows an empty graph", () => {
  const root = container();
  const maliciousLabel = '<img src=x onerror=alert(1)>';
  const input = {...buildPipelineGraph, nodes: [
    {...buildPipelineGraph.nodes[0], label: maliciousLabel},
    ...buildPipelineGraph.nodes.slice(1),
  ]};
  const view = mountReadonlyGraph(root, input);
  assert.equal(root.children.length, 1);
  assert.equal(all(root, "svg").length, 1);
  assert.equal(all(root, "line").length, 2);
  assert.ok(all(root, "text").some(e => e.textContent === maliciousLabel));
  assert.equal(all(root, "img").length, 0);
  view.update({width: 100, height: 80, nodes: [], edges: []});
  assert.equal(all(root, "g").length, 0);
  assert.ok(all(root, "text").some(e => e.textContent === "表示するノードがありません"));
  view.dispose();
  assert.equal(root.children.length, 0);
});

test("selection is only notification, supports keyboard, and cannot grant an action", () => {
  const root = container();
  const selected = [];
  const view = mountReadonlyGraph(root, buildPipelineGraph, {onSelect: id => selected.push(id)});
  const groups = all(root, "g");
  assert.equal(groups.length, 3);
  groups[0].dispatch("click");
  assert.deepEqual(selected, ["source"]);
  assert.equal(view.getSelectedNode(), "source");
  assert.equal(groups[0].getAttribute("aria-pressed"), "true");
  assert.equal(groups[1].getAttribute("aria-pressed"), "false");
  assert.equal(groups[1].dispatch("keydown", "Enter"), true);
  assert.deepEqual(selected, ["source", "check"]);
  assert.equal(groups[1].dispatch("keydown", "Escape"), false);
  assert.equal(selected.length, 2);
  fails("selection_not_found", () => view.setSelectedNode("missing"));
  view.setSelectedNode(null);
  assert.equal(view.getSelectedNode(), null);
  view.dispose();
});

test("invalid updates keep the old display; stale events and disposed views are inert", () => {
  const root = container();
  const events = [];
  const view = mountReadonlyGraph(root, buildPipelineGraph, {onSelect: id => events.push(id)});
  const oldNode = all(root, "g")[0];
  oldNode.dispatch("click");
  const oldSvg = all(root, "svg")[0];
  fails("missing_endpoint", () => view.update({
    ...buildPipelineGraph,
    edges: [{id: "bad", source: "source", target: "absent"}],
  }));
  assert.strictEqual(all(root, "svg")[0], oldSvg);
  assert.equal(view.getSelectedNode(), "source");
  const next = {...buildPipelineGraph,
    nodes: buildPipelineGraph.nodes.slice(1),
    edges: buildPipelineGraph.edges.slice(1)};
  view.update(next);
  assert.notStrictEqual(all(root, "svg")[0], oldSvg);
  assert.equal(view.getSelectedNode(), null);
  oldNode.dispatch("click");
  assert.deepEqual(events, ["source"]);
  const currentNode = all(root, "g")[0];
  view.dispose();
  view.dispose();
  currentNode.dispatch("click");
  assert.deepEqual(events, ["source"]);
  fails("disposed", () => view.update(buildPipelineGraph));
  fails("disposed", () => view.getSelectedNode());
});

test("invalid mount is rejected before DOM mutation", () => {
  const root = container();
  fails("invalid_mount", () =>
    mountReadonlyGraph(root, buildPipelineGraph, {onSelect: "not a function"}));
  assert.equal(root.children.length, 0);
  fails("missing_endpoint", () => mountReadonlyGraph(root, {
    ...buildPipelineGraph, edges: [{id: "bad", source: "source", target: "none"}],
  }));
  assert.equal(root.children.length, 0);
});
