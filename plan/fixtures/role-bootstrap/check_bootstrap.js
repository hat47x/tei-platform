/* Bounded read-only reference for #2620 Role bootstrap / eager vs deferred dependencies.
 * No TEI parser, extension resolver, authority, target or production schema.
 * Run with: node reference/role-bootstrap/check_bootstrap.js
 */
(function () {
  "use strict";
  function fail(code, detail) {
    return { status: "unresolved", code, detail: detail || "" };
  }
  function registry(declarations) {
    const map = new Map();
    for (const d of declarations) {
      if (map.has(d.key)) return fail("duplicate-provider", d.key);
      map.set(d.key, d);
    }
    return map;
  }
  function checkEager(rootKeys, entries, pins) {
    const visited = new Set(), pending = new Set(), path = [];
    function visit(key) {
      if (pending.has(key)) {
        const at = path.indexOf(key);
        return fail("eager-cycle", path.slice(at).concat([key]).join(" -> "));
      }
      if (visited.has(key)) return null;
      const d = entries.get(key);
      if (!d) return fail("missing-provider", key);
      if (!pins || pins[key] !== d.version) return fail("missing-or-stale-pin", key);
      pending.add(key); path.push(key);
      for (const dependency of d.eager || []) {
        const result = visit(dependency);
        if (result) return result;
      }
      path.pop(); pending.delete(key); visited.add(key);
      return null;
    }
    for (const key of [...rootKeys].sort()) {
      const error = visit(key);
      if (error) return error;
    }
    return null;
  }
  function inspect(source, requested, declarations, pins) {
    // Source content and role declarations can be displayed without evaluating it.
    if (!source || typeof source !== "object") return fail("invalid-data");
    if (!source.active) return { status: "data-only", occurrence: source.occurrence || "" };
    if (!Array.isArray(requested)) return fail("invalid-role-binding");
    if (!requested.length) return { status: "data-only", occurrence: source.occurrence || "" };
    const entries = registry(declarations);
    if (!(entries instanceof Map)) return entries;
    const selected = [];
    for (const request of requested) {
      let key = request.key;
      if (!key && request.name) {
        const named = [...entries.values()].filter(d => d.name === request.name);
        if (named.length !== 1) return fail("ambiguous-or-missing-role", request.name);
        key = named[0].key;
      }
      if (!key || !entries.has(key)) return fail("missing-provider", key);
      if (!pins || pins[key] !== entries.get(key).version) return fail("missing-or-stale-pin", key);
      selected.push(key);
    }
    const depProblem = checkEager(selected, entries, pins);
    if (depProblem) return depProblem;
    for (const key of selected) {
      const d = entries.get(key);
      for (const f of d.required || []) {
        if (!Object.prototype.hasOwnProperty.call(source.fields || {}, f)) {
          return fail("missing-required-field", f);
        }
      }
      // Deferred references point to identity, not to eagerly evaluated Role output.
      for (const target of d.deferred || []) {
        if (typeof target !== "string" || !target) return fail("invalid-deferred-ref", key);
      }
    }
    return {
      status: "role-candidate",
      roles: [...new Set(selected)].sort(),
      occurrence: source.occurrence || "",
      declaredEffects: [...new Set(selected)].sort().map(key => [key, entries.get(key).effect || "unknown"]),
      proof: "shape-and-pins-only",
      // Intentionally no "trusted", "executable", "authorized", or "accepted" result.
    };
  }
  const F1 = { key: "tei.function@v1", version: "v1", name: "Function",
    required: ["steps"], eager: [], effect: "pure" };
  const F2 = { key: "tei.function@v2", version: "v2", name: "Function",
    required: ["steps", "output"], eager: [], effect: "pure" };
  const MODEL = { key: "tei.model@v1", version: "v1", name: "Model", required: ["steps"], eager: [] };
  const SOURCE = { active: true, occurrence: "doc.md:10", fields: { steps: [] } };
  const PINS = { "tei.function@v1": "v1", "tei.model@v1": "v1" };
  let passed = 0;
  const cases = [];
  function check(label, observed, predicate) {
    if (!predicate(observed)) throw new Error(label + ": unexpected " + JSON.stringify(observed));
    passed++; cases.push(label);
  }

  check("RB-01 plain data without Role", inspect(SOURCE, [], [], {}), x => x.status === "data-only");
  const one = inspect(SOURCE, [{ key: F1.key }, { key: MODEL.key }], [F1, MODEL], PINS);
  const swapped = inspect(SOURCE, [{ key: MODEL.key }, { key: F1.key }], [MODEL, F1], PINS);
  check("RB-02 provider/order determinism", [one, swapped],
    xs => xs[0].status === "role-candidate" &&
      JSON.stringify(xs[0].roles) === JSON.stringify(xs[1].roles));
  check("RB-03 missing provider", inspect(SOURCE, [{ key: "missing@v1" }], [F1], PINS),
    x => x.code === "missing-provider");
  check("RB-04 same display name ambiguity", inspect(SOURCE, [{ name: "Function" }], [F1, F2],
    { ...PINS, [F2.key]: "v2" }), x => x.code === "ambiguous-or-missing-role");
  const A = { key: "role.A@1", version: "1", name: "A", eager: ["role.B@1"] };
  const B = { key: "role.B@1", version: "1", name: "B", eager: ["role.A@1"] };
  check("RB-05 eager cycle", inspect(SOURCE, [{ key: A.key }], [A, B], { [A.key]:"1", [B.key]:"1" }),
    x => x.code === "eager-cycle" && x.detail.includes("role.B@1"));
  const TREE = { key: "role.Tree@1", version: "1", name: "Tree", deferred: ["role.Tree@1"] };
  check("RB-06 valid deferred recursive Ref",
    inspect(SOURCE, [{ key: TREE.key }], [TREE], { [TREE.key]:"1" }),
    x => x.status === "role-candidate");
  check("RB-07 independent multiple Roles", one,
    x => x.roles.length === 2 && x.proof === "shape-and-pins-only");
  check("RB-08 quoted instructions remain data-only",
    inspect({ active: false, occurrence: "quote:7", fields: { role:"trusted-script", command:"run-as-admin" } },
      [{ key: F1.key }], [F1], PINS), x => x.status === "data-only");
  const before = inspect(SOURCE, [{ key: F1.key }], [F1], PINS);
  const after = inspect(SOURCE, [{ key: F2.key }], [F2], { [F2.key]: "v2" });
  check("RB-09 source unchanged, Role version changes qualification", [before,after],
    xs => xs[0].status === "role-candidate" && xs[1].code === "missing-required-field");
  check("RB-10 effect declaration != actual authorization", before,
    x => x.status === "role-candidate" && x.declaredEffects[0][1] === "pure" &&
      !("authorized" in x) && !("executable" in x));
  const occurrence2 = inspect({ ...SOURCE, occurrence: "cases.md:8" }, [{ key: F1.key }], [F1], PINS);
  check("RB-11 source occurrence retained independently", [before, occurrence2],
    xs => xs[0].roles[0] === xs[1].roles[0] && xs[0].occurrence !== xs[1].occurrence);
  check("RB-12 Skill/tool claim cannot be deduced", before,
    x => x.status === "role-candidate" && !("tools" in x) && !("accepted" in x));
  check("RB-13 unpinned provider rejected", inspect(SOURCE, [{ key: MODEL.key }], [MODEL], {}),
    x => x.code === "missing-or-stale-pin");
  check("RB-14 eager + deferred mixed reference", inspect(SOURCE, [{ key:A.key }],
    [A, { ...B, eager: [], deferred: [A.key] }], { [A.key]:"1", [B.key]:"1" }),
    x => x.status === "role-candidate");
  check("RB-15 missing required shape", inspect(SOURCE, [{ key:F2.key }], [F2], { [F2.key]:"v2" }),
    x => x.code === "missing-required-field");
  check("RB-16 unpinned eager dependency is not silently loaded",
    inspect(SOURCE, [{ key:A.key }], [A, { ...B, eager: [] }], { [A.key]:"1" }),
    x => x.code === "missing-or-stale-pin" && x.detail === B.key);
  check("RB-17 explicitly pinned eager dependency is qualified",
    inspect(SOURCE, [{ key:A.key }], [A, { ...B, eager: [] }], { [A.key]:"1", [B.key]:"1" }),
    x => x.status === "role-candidate" && x.roles.length === 1);
  check("RB-18 duplicate provider identifier is unresolved",
    inspect(SOURCE, [{ key: F1.key }], [F1, F1], PINS),
    x => x.code === "duplicate-provider");
  check("RB-19 missing eager dependency is unresolved",
    inspect(SOURCE, [{ key:A.key }], [A], { [A.key]:"1" }),
    x => x.code === "missing-provider" && x.detail === B.key);
  if (typeof console !== "undefined" && typeof console.log === "function") {
    console.log("PASS: " + passed + "/" + cases.length + " bounded Role bootstrap fixture cases");
  }
  return { passed, total: cases.length, cases };
})()
