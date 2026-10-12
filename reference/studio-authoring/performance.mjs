// #1124: Synthetic large-reference measurement, not a product acceptance gate.
// Source/Business authority and actual browser performance are not established here.
import assert from "node:assert/strict";
import { performance } from "node:perf_hooks";
import { projectWorld } from "./bridge.mjs";
import { renderStudioHTML } from "./html.mjs";

const SIZE = 10_000;
const revision = "sha256:" + "a".repeat(64);
const subject = "data:perf.target";
const data = Array.from({ length: SIZE }, (_, i) => {
  const id = "perf.item_" + i;
  return {
    id,
    attributes: [{ id: id + ".value", type: "std.decimal" }],
    relations: [{ id: id + ".target", target: "perf.target", cardinality: "one" }]
  };
});
data.push({ id: "perf.target", attributes: [] });
const world = {
  semantics: [{ id: "perf", data }],
  applications: [{
    id: "perf_ui",
    patterns: Array.from({ length: SIZE }, (_, i) => ({
      id: "perf_ui.pattern_" + i,
      kind: "reference",
      relation: "perf.item_" + i + ".target"
    }))
  }]
};

// Measure the real production projection and HTML renderer with a high-fan-in
// focus. The original World object is never passed to any write/accept method.
const started = performance.now();
const projection = projectWorld(world, revision, subject);
const projectionMs = performance.now() - started;
const renderingStarted = performance.now();
const html = renderStudioHTML(projection);
const htmlMs = performance.now() - renderingStarted;

assert.equal(projection.focus.subject, subject);
assert.equal(projection.modelRevision, revision);
assert.equal(projection.authority, "derived-readonly");
assert.equal(projection.editable, false);
assert.equal(projection.views.er.edges.length, SIZE);
assert.equal(projection.views.er.nodes.length, SIZE + 1);
assert.equal(projection.views.function.items.length, SIZE);
assert.equal(projection.views.business.status, "unresolved");
assert.equal(projection.sourceNavigation.status, "unresolved");
assert.match(html, /表示上限100件/);
assert.match(html, /残り9900件は省略しています/);
assert.match(html, /残り9901件は省略しています/);
assert.match(html, /id="er"/);
assert.match(html, /id="function"/);
assert.match(html, /id="business"/);
assert.doesNotMatch(html, /id="studio-edit"/);

process.stdout.write(JSON.stringify({
  schema: "tei.reference.studio-performance-receipt/v0",
  head: process.env.GITHUB_SHA || null,
  fixture: { kind: "synthetic", data: SIZE + 1, attributes: SIZE,
    relations: SIZE, patterns: SIZE, highFanIn: SIZE },
  result: { erNodes: projection.views.er.nodes.length,
    erEdges: projection.views.er.edges.length,
    functionItems: projection.views.function.items.length,
    htmlBytes: Buffer.byteLength(html, "utf8"),
    truncatedHTMLLists: true },
  observed: {
    nodeVersion: process.version,
    platform: process.platform,
    projectionMs: Number(projectionMs.toFixed(3)),
    htmlMs: Number(htmlMs.toFixed(3)),
    rssMiB: Number((process.memoryUsage().rss / (1024 * 1024)).toFixed(3))
  },
  authority: { sourceWrite: "not-performed", humanAcceptance: "not-observed",
    actualBrowser: "not-observed", acceptedBusiness: "not-established",
    physicalTarget: "not-observed" },
  qualification: "single-run synthetic measurement; not a latency SLO or browser benchmark"
}, null, 2) + "\n");
