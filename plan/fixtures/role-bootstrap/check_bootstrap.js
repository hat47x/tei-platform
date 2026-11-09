/* Bounded read-only reference for #2620 Role bootstrap / eager vs deferred dependencies.
 * No TEI parser, extension resolver, authority, target or production schema.
 * Run with: node plan/fixtures/role-bootstrap/check_bootstrap.js
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
  // In this fixture, tokens stand for exact content identities. Real TEI must
  // verify bytes via its existing Extension Lock, not trust descriptor claims.
  function pinFor(d) {
    return { version: d.version, contract: d.contractFingerprint,
      implementation: d.implementationFingerprint };
  }
  function pinMatches(d, pins) {
    const pin = pins && pins[d.key];
    return Boolean(pin && d.contractFingerprint && d.implementationFingerprint &&
      pin.version === d.version && pin.contract === d.contractFingerprint &&
      pin.implementation === d.implementationFingerprint);
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
      if (!pinMatches(d, pins)) return fail("missing-or-stale-pin", key);
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
      if (!pinMatches(entries.get(key), pins)) return fail("missing-or-stale-pin", key);
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
    required: ["steps"], eager: [], effect: "pure",
    contractFingerprint: "function-v1-contract", implementationFingerprint: "function-v1-impl" };
  const F2 = { key: "tei.function@v2", version: "v2", name: "Function",
    required: ["steps", "output"], eager: [], effect: "pure",
    contractFingerprint: "function-v2-contract", implementationFingerprint: "function-v2-impl" };
  const MODEL = { key: "tei.model@v1", version: "v1", name: "Model", required: ["steps"], eager: [],
    contractFingerprint: "model-v1-contract", implementationFingerprint: "model-v1-impl" };
  const SOURCE = { active: true, occurrence: "doc.md:10", fields: { steps: [] } };
  const PINS = { [F1.key]: pinFor(F1), [MODEL.key]: pinFor(MODEL) };
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
    { ...PINS, [F2.key]: pinFor(F2) }), x => x.code === "ambiguous-or-missing-role");
  const A = { key: "role.A@1", version: "1", name: "A", eager: ["role.B@1"],
    contractFingerprint: "A-contract", implementationFingerprint: "A-impl" };
  const B = { key: "role.B@1", version: "1", name: "B", eager: ["role.A@1"],
    contractFingerprint: "B-contract", implementationFingerprint: "B-impl" };
  check("RB-05 eager cycle", inspect(SOURCE, [{ key: A.key }], [A, B], { [A.key]:pinFor(A), [B.key]:pinFor(B) }),
    x => x.code === "eager-cycle" && x.detail.includes("role.B@1"));
  const TREE = { key: "role.Tree@1", version: "1", name: "Tree", deferred: ["role.Tree@1"],
    contractFingerprint: "Tree-contract", implementationFingerprint: "Tree-impl" };
  check("RB-06 valid deferred recursive Ref",
    inspect(SOURCE, [{ key: TREE.key }], [TREE], { [TREE.key]:pinFor(TREE) }),
    x => x.status === "role-candidate");
  check("RB-07 independent multiple Roles", one,
    x => x.roles.length === 2 && x.proof === "shape-and-pins-only");
  check("RB-08 quoted instructions remain data-only",
    inspect({ active: false, occurrence: "quote:7", fields: { role:"trusted-script", command:"run-as-admin" } },
      [{ key: F1.key }], [F1], PINS), x => x.status === "data-only");
  const before = inspect(SOURCE, [{ key: F1.key }], [F1], PINS);
  const after = inspect(SOURCE, [{ key: F2.key }], [F2], { [F2.key]:pinFor(F2) });
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
  const BNoEager = { ...B, eager: [], implementationFingerprint: "B-impl-no-eager" };
  const BDeferred = { ...BNoEager, deferred: [A.key], implementationFingerprint: "B-impl-deferred-A" };
  check("RB-14 eager + deferred mixed reference", inspect(SOURCE, [{ key:A.key }],
    [A, BDeferred], { [A.key]:pinFor(A), [B.key]:pinFor(BDeferred) }),
    x => x.status === "role-candidate");
  check("RB-15 missing required shape", inspect(SOURCE, [{ key:F2.key }], [F2], { [F2.key]:pinFor(F2) }),
    x => x.code === "missing-required-field");
  check("RB-16 unpinned eager dependency is not silently loaded",
    inspect(SOURCE, [{ key:A.key }], [A, BNoEager], { [A.key]:pinFor(A) }),
    x => x.code === "missing-or-stale-pin" && x.detail === B.key);
  check("RB-17 explicitly pinned eager dependency is qualified",
    inspect(SOURCE, [{ key:A.key }], [A, BNoEager], { [A.key]:pinFor(A), [B.key]:pinFor(BNoEager) }),
    x => x.status === "role-candidate" && x.roles.length === 1);
  check("RB-18 duplicate provider identifier is unresolved",
    inspect(SOURCE, [{ key: F1.key }], [F1, F1], PINS),
    x => x.code === "duplicate-provider");
  check("RB-19 missing eager dependency is unresolved",
    inspect(SOURCE, [{ key:A.key }], [A], { [A.key]:pinFor(A) }),
    x => x.code === "missing-provider" && x.detail === B.key);
  const changedF1 = { ...F1, implementationFingerprint: "function-v1-changed-impl" };
  const changedB = { ...BNoEager, implementationFingerprint: "B-impl-changed" };
  check("RB-20 root same-version implementation drift rejects old Pin",
    inspect(SOURCE, [{ key:F1.key }], [changedF1], PINS),
    x => x.code === "missing-or-stale-pin" && x.detail === F1.key);
  check("RB-21 eager dependency same-version implementation drift rejects old Pin",
    inspect(SOURCE, [{ key:A.key }], [A,changedB],
      { [A.key]:pinFor(A), [B.key]:pinFor(BNoEager) }),
    x => x.code === "missing-or-stale-pin" && x.detail === B.key);
  check("RB-22 exact updated dependency Pin can qualify its candidate",
    inspect(SOURCE, [{ key:A.key }], [A,changedB],
      { [A.key]:pinFor(A), [B.key]:pinFor(changedB) }),
    x => x.status === "role-candidate");
  const unrelated = { key: "other.role@1", version: "1", name: "Other",
    contractFingerprint: "other-contract", implementationFingerprint: "other-impl" };
  check("RB-23 adding unrelated provider preserves pinned selected result",
    [inspect(SOURCE, [{ key:F1.key }], [F1], PINS),
      inspect(SOURCE, [{ key:F1.key }], [unrelated,F1], PINS)],
    xs => xs[0].status === "role-candidate" &&
      JSON.stringify(xs[0]) === JSON.stringify(xs[1]));
  check("RB-24 version-only Pin is insufficient",
    inspect(SOURCE, [{ key:F1.key }], [F1], { [F1.key]:"v1" }),
    x => x.code === "missing-or-stale-pin");
  if (typeof console !== "undefined" && typeof console.log === "function") {
    console.log("PASS: " + passed + "/" + cases.length + " bounded Role bootstrap fixture cases");
  }
  return { passed, total: cases.length, cases };
})()
