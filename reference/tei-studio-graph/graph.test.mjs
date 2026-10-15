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
    this.style = {};
    this.hidden = false;
    this.textContent = "";
  }
  focus() {
    this.ownerDocument.activeElement = this;
    this.dispatch("focus");
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
    const event = {key, preventDefault() { prevented = true; }};
    // SVG click/key events bubble from a circle/text child to the <g> button.
    for (let current = this; current; current = current.parentNode) {
      for (const listener of current.listeners.get(name) ?? []) listener(event);
    }
    return prevented;
  }
}
function container() {
  const doc = {
    activeElement: null,
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
  assert.ok(all(root, "text").some(e => e.textContent.includes("…")));
  assert.ok(all(root, "title").some(e => e.textContent === maliciousLabel));
  assert.equal(all(root, "g")[0].getAttribute("aria-label"), maliciousLabel);
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

test("narrow 240px panel scrolls the full-size 480px graph without shrinking labels", () => {
  const root = container();
  root.style.width = "240px"; // simulated host layout, not a layout engine
  const picked = [];
  const view = mountReadonlyGraph(root, buildPipelineGraph, {
    onSelect: id => picked.push(id),
  });
  const viewport = root.children[0];
  const svg = all(root, "svg")[0];
  assert.equal(viewport.getAttribute("data-graph-reference"), "read-only");
  assert.equal(viewport.style.overflowX, "auto");
  assert.equal(viewport.style.maxWidth, "100%");
  assert.equal(viewport.style.minWidth, "0");
  assert.equal(viewport.getAttribute("tabindex"), "0");
  assert.equal(svg.getAttribute("viewBox"), "0 0 480 160");
  assert.equal(svg.style.minWidth, "480px");
  assert.equal(svg.style.maxWidth, "none");
  assert.equal(svg.style.height, "auto");
  assert.equal(svg.style.fontSize, "14px");

  // Presentation-only selection remains intact even when the graphic is wide.
  all(root, "g")[2].dispatch("click");
  assert.deepEqual(picked, ["publish"]);
  assert.equal(view.getSelectedNode(), "publish");

  view.update({...buildPipelineGraph, width: 620});
  const replacement = all(root, "svg")[0];
  assert.equal(replacement.style.minWidth, "620px");
  assert.equal(replacement.getAttribute("viewBox"), "0 0 620 160");
  assert.equal(view.getSelectedNode(), "publish");
  view.dispose();
  assert.equal(root.children.length, 0);
});

test("transparent node centers are pointer targets and click bubbles to the selector", () => {
  const root = container();
  const selected = [];
  const view = mountReadonlyGraph(root, buildPipelineGraph, {onSelect: id => selected.push(id)});
  const groups = all(root, "g");
  const circle = groups[1].children.find(child => child.tagName === "circle");
  assert.ok(circle);
  assert.equal(circle.getAttribute("fill"), "transparent");
  assert.equal(circle.getAttribute("pointer-events"), "all");
  assert.equal(circle.getAttribute("stroke"), "currentColor");
  assert.equal(circle.getAttribute("r"), "12");

  // The actual pointer target is the circle, not the parent <g>.
  circle.dispatch("click");
  assert.deepEqual(selected, ["check"]);
  assert.equal(view.getSelectedNode(), "check");
  assert.equal(groups[1].getAttribute("aria-pressed"), "true");

  // The keyboard listener remains on <g> and supports both activation keys.
  assert.equal(groups[2].dispatch("keydown", "Enter"), true);
  assert.equal(groups[0].dispatch("keydown", " "), true);
  assert.deepEqual(selected, ["check", "publish", "source"]);
  assert.equal(view.getSelectedNode(), "source");
  view.dispose();
});

test("very long node labels are abbreviated but recoverable on focus and selection", () => {
  const root = container();
  const id = '["semantic-data","sales.order.customer_id"]';
  const longLabel = "SUPER_LONG_LABEL_" + "LONG_SEGMENT_".repeat(25);
  const chosen = [];
  const view = mountReadonlyGraph(root, {
    width: 480, height: 170, nodes: [{id, label: longLabel, x: 240, y: 70}], edges: [],
  }, {onSelect: id => chosen.push(id)});
  const group = all(root, "g")[0];
  const visible = all(group, "text")[0];
  const full = all(root, "div").find(e => e.getAttribute("data-graph-full-label") === "");
  assert.ok(full);
  assert.equal(full.hidden, true);
  assert.equal(group.getAttribute("aria-label"), longLabel);
  assert.equal(all(group, "title")[0].textContent, longLabel);
  assert.ok(visible.textContent.endsWith("…"));
  assert.ok(visible.textContent.length < longLabel.length);
  group.dispatch("focus");
  assert.equal(full.hidden, false);
  assert.equal(full.textContent, `ラベル全文: ${longLabel}`);
  const circle = all(group, "circle")[0];
  circle.dispatch("click");
  assert.deepEqual(chosen, [id]);
  assert.equal(view.getSelectedNode(), id);
  assert.equal(group.dispatch("keydown", "Enter"), true);
  assert.deepEqual(chosen, [id, id]);
  view.setSelectedNode(null);
  assert.equal(full.hidden, true);
  view.dispose();
  assert.equal(root.children.length, 0);
});

test("keyboard focus survives consumer graph.update and returns to viewport when node disappears", () => {
  const root = container();
  const calls = [];
  const view = mountReadonlyGraph(root, buildPipelineGraph, {
    onSelect(id) {
      calls.push(id);
      // Mirrors the Studio consumer's onSelect -> inspect -> graph.update.
      view.update(buildPipelineGraph);
      view.setSelectedNode(id);
    },
  });
  const check = all(root, "g")[1];
  check.focus();
  assert.strictEqual(root.ownerDocument.activeElement, check);
  assert.equal(check.dispatch("keydown", "Enter"), true);
  const replacement = all(root, "g")[1];
  assert.notStrictEqual(replacement, check);
  assert.strictEqual(root.ownerDocument.activeElement, replacement);
  assert.equal(replacement.dispatch("keydown", " "), true);
  assert.deepEqual(calls, ["check", "check"]);
  assert.equal(view.getSelectedNode(), "check");

  // The removed focused node cannot remain active. The scroll region owns
  // keyboard focus until the user chooses another node.
  view.update({width: 480, height: 160, nodes: [], edges: []});
  assert.strictEqual(root.ownerDocument.activeElement, root.children[0]);
  view.dispose();
});
