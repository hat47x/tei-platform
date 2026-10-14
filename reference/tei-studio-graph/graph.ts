// PoC C: host-neutral, read-only graph presentation Reference.
// This module makes no TEI semantic, authority, filesystem, or command decisions.
export interface GraphNode {
  readonly id: string;
  readonly label: string;
  readonly x: number;
  readonly y: number;
}

export interface GraphEdge {
  readonly id: string;
  readonly source: string;
  readonly target: string;
  readonly label?: string;
}

export interface GraphInput {
  readonly width: number;
  readonly height: number;
  readonly nodes: readonly GraphNode[];
  readonly edges: readonly GraphEdge[];
}

export interface GraphOptions {
  readonly onSelect?: (nodeId: string) => void;
}

export interface MountedGraph {
  update(input: unknown): void;
  setSelectedNode(nodeId: string | null): void;
  getSelectedNode(): string | null;
  dispose(): void;
}

export class GraphViewError extends Error {
  readonly code: string;
  constructor(code: string) {
    super(code);
    this.code = code;
    this.name = "GraphViewError";
  }
}

const fail = (code: string): never => { throw new GraphViewError(code); };
const record = (v: unknown): v is Record<string, unknown> =>
  typeof v === "object" && v !== null && !Array.isArray(v);
const exactId = (v: unknown): v is string =>
  typeof v === "string" && v.length > 0 && v === v.trim();
const finite = (v: unknown): v is number =>
  typeof v === "number" && Number.isFinite(v);

// Copy only display fields. Never interpret IDs as authority, source paths,
// command names, HTML, or executable code.
export function normalizeGraph(input: unknown): GraphInput {
  if (!record(input) || !finite(input.width) || input.width <= 0 ||
      !finite(input.height) || input.height <= 0 ||
      !Array.isArray(input.nodes) || !Array.isArray(input.edges)) {
    return fail("invalid_graph");
  }
  const width = input.width;
  const height = input.height;
  const nodes: GraphNode[] = [];
  const edges: GraphEdge[] = [];
  const nodeIDs = new Set<string>();
  const edgeIDs = new Set<string>();

  for (const item of input.nodes) {
    if (!record(item) || !exactId(item.id) || typeof item.label !== "string" ||
        !finite(item.x) || !finite(item.y) ||
        item.x < 0 || item.x > width || item.y < 0 || item.y > height) {
      return fail("invalid_node");
    }
    if (nodeIDs.has(item.id)) return fail("duplicate_node");
    nodeIDs.add(item.id);
    nodes.push({id: item.id, label: item.label, x: item.x, y: item.y});
  }
  for (const item of input.edges) {
    if (!record(item) || !exactId(item.id) || !exactId(item.source) ||
        !exactId(item.target) ||
        (item.label !== undefined && typeof item.label !== "string")) {
      return fail("invalid_edge");
    }
    if (edgeIDs.has(item.id)) return fail("duplicate_edge");
    if (!nodeIDs.has(item.source) || !nodeIDs.has(item.target)) {
      return fail("missing_endpoint");
    }
    // The initial straight-line renderer does not render self-loops faithfully.
    if (item.source === item.target) return fail("unsupported_self_edge");
    edgeIDs.add(item.id);
    edges.push({
      id: item.id, source: item.source, target: item.target,
      ...(item.label === undefined ? {} : {label: item.label}),
    });
  }
  return {width, height, nodes, edges};
}

type NodeVisual = {group: SVGGElement; circle: SVGCircleElement};
type BuiltGraph = {svg: SVGSVGElement; groups: Map<string, NodeVisual>};

export function mountReadonlyGraph(
  container: HTMLElement,
  input: unknown,
  options: GraphOptions = {},
): MountedGraph {
  if (!container || !options || typeof options !== "object" ||
      typeof container.appendChild !== "function" ||
      !container.ownerDocument ||
      typeof container.ownerDocument.createElementNS !== "function" ||
      (options.onSelect !== undefined && typeof options.onSelect !== "function")) {
    return fail("invalid_mount");
  }
  const initial = normalizeGraph(input);
  const doc = container.ownerDocument;
  const ns = "http://www.w3.org/2000/svg";
  const svgElement = <K extends keyof SVGElementTagNameMap>(tag: K): SVGElementTagNameMap[K] =>
    doc.createElementNS(ns, tag);
  const attribute = (el: Element, name: string, value: string | number) =>
    el.setAttribute(name, String(value));
  const holder = doc.createElement("div");
  holder.setAttribute("data-graph-reference", "read-only");
  // Keep this viewport inside narrow grid/flex panels. The graph can scroll
  // horizontally instead of shrinking its labels down to unreadable text.
  holder.style.width = "100%";
  holder.style.maxWidth = "100%";
  holder.style.minWidth = "0";
  holder.style.overflowX = "auto";
  holder.style.overflowY = "hidden";
  holder.setAttribute("tabindex", "0");
  holder.setAttribute("role", "region");
  holder.setAttribute("aria-label", "グラフ表示（左右にスクロール可能）");
  let active: BuiltGraph | null = null;
  let revision = 0;
  let selected: string | null = null;
  let disposed = false;

  const requireLive = () => { if (disposed) fail("disposed"); };
  const setSelectedNode = (nodeId: string | null) => {
    requireLive();
    if (nodeId !== null && !active?.groups.has(nodeId)) fail("selection_not_found");
    selected = nodeId;
    for (const [id, {group, circle}] of active?.groups ?? []) {
      attribute(group, "aria-pressed", id === selected ? "true" : "false");
      attribute(group, "data-selected", id === selected ? "true" : "false");
      attribute(circle, "stroke-width", id === selected ? 3 : 1.5);
    }
  };

  const build = (graph: GraphInput, generation: number): BuiltGraph => {
    const svg = svgElement("svg");
    attribute(svg, "viewBox", `0 0 ${graph.width} ${graph.height}`);
    attribute(svg, "role", "group");
    attribute(svg, "aria-label", "読取り専用グラフ");
    attribute(svg, "width", "100%");
    // 1 viewBox unit is at least 1 CSS pixel. A 480-unit graph in a 240px
    // panel remains >=480px wide and is scrolled within holder, not scaled 50%.
    // Wider panels may scale up; no automatic graph layout is performed.
    svg.style.minWidth = `${graph.width}px`;
    svg.style.maxWidth = "none";
    svg.style.height = "auto";
    svg.style.fontSize = "14px";
    svg.style.display = "block";
    const groups = new Map<string, NodeVisual>();
    const byId = new Map(graph.nodes.map(n => [n.id, n]));

    for (const edge of graph.edges) {
      const from = byId.get(edge.source)!;
      const to = byId.get(edge.target)!;
      const line = svgElement("line");
      for (const [key, value] of Object.entries({
        x1: from.x, y1: from.y, x2: to.x, y2: to.y,
        stroke: "currentColor", "stroke-opacity": "0.65",
      })) attribute(line, key, value);
      svg.appendChild(line);
      if (edge.label) {
        const caption = svgElement("text");
        attribute(caption, "x", (from.x + to.x) / 2);
        attribute(caption, "y", (from.y + to.y) / 2 - 6);
        attribute(caption, "text-anchor", "middle");
        caption.textContent = edge.label;
        svg.appendChild(caption);
      }
    }

    for (const node of graph.nodes) {
      const group = svgElement("g");
      attribute(group, "role", "button");
      attribute(group, "tabindex", "0");
      attribute(group, "aria-label", node.label || node.id);
      const circle = svgElement("circle");
      for (const [key, value] of Object.entries({
        // A transparent *painted* fill preserves the outlined appearance but
        // makes the circle interior a pointer target. With fill="none" only
        // the stroke was hittable; the g element has no painted hit area.
        cx: node.x, cy: node.y, r: 12, fill: "transparent",
        "pointer-events": "all", stroke: "currentColor",
      })) attribute(circle, key, value);
      group.appendChild(circle);
      const label = svgElement("text");
      attribute(label, "x", node.x);
      attribute(label, "y", node.y + 29);
      attribute(label, "text-anchor", "middle");
      label.textContent = node.label; // text, never innerHTML
      group.appendChild(label);
      const notify = () => {
        if (disposed || generation !== revision) return;
        setSelectedNode(node.id);
        options.onSelect?.(node.id);
      };
      group.addEventListener("click", notify);
      group.addEventListener("keydown", event => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          notify();
        }
      });
      groups.set(node.id, {group, circle});
      svg.appendChild(group);
    }
    if (graph.nodes.length === 0) {
      const empty = svgElement("text");
      attribute(empty, "x", graph.width / 2);
      attribute(empty, "y", graph.height / 2);
      attribute(empty, "text-anchor", "middle");
      empty.textContent = "表示するノードがありません";
      svg.appendChild(empty);
    }
    return {svg, groups};
  };

  const replace = (graph: GraphInput) => {
    const nextRevision = revision + 1;
    const next = build(graph, nextRevision);
    holder.replaceChildren(next.svg);
    active = next;
    revision = nextRevision;
    if (selected !== null && !next.groups.has(selected)) selected = null;
    setSelectedNode(selected);
  };

  replace(initial);
  container.appendChild(holder);
  return {
    update(nextInput: unknown) {
      requireLive();
      // Invalid updates leave the previous graph and selection untouched.
      replace(normalizeGraph(nextInput));
    },
    setSelectedNode,
    getSelectedNode() { requireLive(); return selected; },
    dispose() {
      if (disposed) return;
      disposed = true;
      revision++;
      active = null;
      selected = null;
      holder.remove();
    },
  };
}
