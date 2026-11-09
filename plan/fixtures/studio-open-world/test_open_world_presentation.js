/* Synthetic open-world TEI Studio presentation probe. No TEI parser, Source Acceptance,
   real Studio renderer, Provider, Host authority, or Business Action execution.
   Run with: node plan/fixtures/studio-open-world/test_open_world_presentation.js */
(function () {
  "use strict";
  const pass = [];
  function assert(ok, name) { if (!ok) throw Error("FAIL: " + name); pass.push(name); }
  function equal(a, b) { return JSON.stringify(a) === JSON.stringify(b); }
  function shape(x) {
    if (x === null) return "null";
    if (Array.isArray(x)) return "sequence";
    return typeof x === "object" ? "record" : typeof x;
  }
  function flatten(x, prefix, allowed, rows) {
    if (Array.isArray(x)) {
      x.forEach(function (v, i) { flatten(v, prefix + "[" + i + "]", allowed, rows); });
    } else if (x !== null && typeof x === "object") {
      Object.keys(x).sort().forEach(function (key) {
        flatten(x[key], prefix ? prefix + "." + key : key, allowed, rows);
      });
    } else if (allowed.has(prefix)) {
      rows.push({path: prefix, type: shape(x), value: x});
    }
    return rows;
  }
  function inspect(doc, allowedPaths, binding, plugins) {
    /* 'doc' is already an approved consumer-scoped data slice.
       The plain data field 'role' never authorizes its own interpretation. */
    const rows = flatten(doc, "", new Set(allowedPaths), []);
    const publicValues = Object.fromEntries(rows.map(function (r) { return [r.path, r.value]; }));
    const plugin = binding && binding.accepted && plugins[binding.id];
    const semantic = plugin
      ? {status: "qualified-candidate", role: binding.id, description: plugin(publicValues)}
      : {status: "unresolved"};
    return {columns: ["path", "type", "value"], rows: rows,
            semantic: semantic, actions: []};
  }
  function editCandidate(doc, path, replacement, source, expectedRevision) {
    const o = source && source.occurrences && source.occurrences[path];
    if (!o || !o.editable || o.count !== 1 || o.derived) return {status: "readonly"};
    if (source.revision !== expectedRevision || source.currentDigest !== source.expectedDigest)
      return {status: "stale"};
    if (!Object.prototype.hasOwnProperty.call(doc, path)) return {status: "readonly"};
    return {status: "candidate", path: path, replacement: replacement,
            revision: expectedRevision, sourceSpan: [o.start, o.end]};
  }
  const plugins = {
    "std.visibility@1": function (v) {
      return v["target"] && v["dependency"] ? "typed-predicate-reference" : "incomplete";
    },
    "sim.motion@2": function (v) {
      return v["controller"] && typeof v["frequency"] === "number"
        ? "time-dependent-operator-reference" : "incomplete";
    }
  };
  const vis = {target: "order.discount", dependency: "customer.kind",
               enabled: false, threshold: 0, optional: null};
  const allowVis = ["dependency", "enabled", "optional", "target", "threshold"];
  const a = inspect(vis, allowVis, {accepted: true, id: "std.visibility@1"}, plugins);
  assert(a.columns.join("|") === "path|type|value", "OW-01-generic-columns");
  assert(a.semantic.description === "typed-predicate-reference", "OW-02-visibility-as-plugin");
  assert(a.rows.some(function (r) { return r.path === "enabled" && r.type === "boolean" && r.value === false; }),
         "OW-03-bool-preserved");
  assert(a.rows.some(function (r) { return r.path === "threshold" && r.type === "number" && r.value === 0; }) &&
         a.rows.some(function (r) { return r.path === "optional" && r.type === "null"; }),
         "OW-04-null-zero-preserved");
  const motion = {controller: "bear.center_of_mass", frequency: 10, state: {step: 4}};
  const b = inspect(motion, ["controller", "frequency", "state.step"],
    {accepted: true, id: "sim.motion@2"}, plugins);
  assert(equal(a.columns, b.columns) && b.semantic.description === "time-dependent-operator-reference",
         "OW-05-second-domain-no-new-renderer");
  const custom = {chapter: "intro", facets: {remark: "new-plugin-specific"}};
  const c = inspect(custom, ["chapter", "facets.remark"],
    {accepted: true, id: "book.chapter@1"}, plugins);
  assert(equal(a.columns, c.columns) && c.semantic.status === "unresolved" &&
         c.rows.some(function (r) { return r.value === "new-plugin-specific"; }),
         "OW-06-unknown-role-visible-as-plain-data");
  const pluginsAfter = Object.assign({}, plugins, {"book.chapter@1": function (v) {
    return v["chapter"] ? "chapter-reference" : "incomplete";
  }});
  const d = inspect(custom, ["chapter", "facets.remark"],
    {accepted: true, id: "book.chapter@1"}, pluginsAfter);
  assert(equal(c.columns, d.columns) && equal(c.rows, d.rows) &&
         d.semantic.description === "chapter-reference", "OW-07-plugin-extension-unchanged-rows");
  const forged = inspect({role: "trusted-script", command: "delete-all"},
    ["role", "command"], null, pluginsAfter);
  assert(forged.semantic.status === "unresolved" && forged.actions.length === 0,
         "OW-08-untrusted-role-never-execution-grant");
  const unaccepted = inspect(vis, allowVis,
    {accepted: false, id: "std.visibility@1"}, pluginsAfter);
  assert(unaccepted.semantic.status === "unresolved", "OW-09-role-is-not-self-authorizing");
  const secretA = {customer: {name: "Public", secret: 40}, status: "draft"};
  const secretB = {customer: {name: "Public", secret: 90}, status: "draft"};
  const permitted = ["customer.name", "status"];
  assert(equal(inspect(secretA, permitted, null, pluginsAfter),
               inspect(secretB, permitted, null, pluginsAfter)),
         "OW-10-non-disclosed-values-do-not-change-view");
  const graph = {nodes: [{id: "A"}, {id: "B"}], edges: [{from: "A", to: "B"}]};
  const g = inspect(graph, ["nodes[0].id", "nodes[1].id", "edges[0].from", "edges[0].to"],
    {accepted: true, id: "unknown.graph@1"}, pluginsAfter);
  assert(g.rows.length === 4 && g.actions.length === 0, "OW-11-graph-as-standard-record-sequence");
  const src = {revision: "R1", currentDigest: "d1", expectedDigest: "d1",
               occurrences: {target: {editable: true, count: 1, derived: false, start: 3, end: 9}}};
  const editData = {target: "order.discount"};
  const edit = editCandidate(editData, "target", "order.discount.new", src, "R1");
  assert(edit.status === "candidate" && edit.sourceSpan[0] === 3, "OW-12-source-aware-edit-is-candidate");
  assert(editCandidate(editData, "target", "x", src, "R0").status === "stale",
         "OW-13-stale-source-prevents-edit");
  const ambiguous = {...src, occurrences: {target: {...src.occurrences.target, count: 2}}};
  assert(editCandidate(editData, "target", "x", ambiguous, "R1").status === "readonly",
         "OW-14-multiple-origin-readonly");
  const derived = {...src, occurrences: {target: {...src.occurrences.target, derived: true}}};
  assert(editCandidate(editData, "target", "x", derived, "R1").status === "readonly",
         "OW-15-derived-readonly");
  assert(a.rows.some(function (r) { return r.path === "enabled" && r.value === false; }) &&
         a.actions.length === 0 && b.actions.length === 0,
         "OW-16-visibility-state-not-host-authority");
  if (typeof console !== "undefined") console.log(pass.length + " synthetic checks PASS");
  return {passed: pass.length, names: pass};
})()
