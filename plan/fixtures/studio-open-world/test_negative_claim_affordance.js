/* Synthetic read-only Consumer pressure: empty result is not proof of absence.
   This code has no TEI Parser/Provider/Host and cannot grant an Action. */
(function () {
  "use strict";
  const pass = [];
  function check(ok, name) { if (!ok) throw Error("FAIL " + name); pass.push(name); }
  function equal(a,b) { return JSON.stringify(a) === JSON.stringify(b); }
  const role = {predicate: "not-exists", query: "approval.pending@v1",
    depends: ["approval.status", "approval.membership"],
    output: "order.submit.enabled", meaningRevision: "M1"};
  function candidate(rows, read, query = role) {
    /* These are stipulated, already scoped rows; this does not execute a Query. */
    const base = {render: {rows: rows.map(x => ({id: x.id, status: x.status}))},
      semantic: {status: "not-established"}, offer: {enabled: false}};
    if (!read.disclosed) return {...base, render: {status: "unavailable"}};
    if (read.scope !== "order:O1" || read.meaningRevision !== query.meaningRevision ||
        read.query !== query.query || read.epoch !== 7 || !read.current ||
        !read.readAllowed || !read.routeQualified || !read.operatorQualified) return base;
    const pending = rows.some(x => x.status === "pending");
    /* A valid positive witness refutes 'no pending' without complete enumeration. */
    if (pending) return {...base, semantic: {status: "refuted", basis: "positive-witness"}};
    if (!(read.complete && read.rowsetCovered && read.predicateCovered &&
          read.orderEquivalent)) return base;
    return {...base, semantic: {status: "supported-in-scope", basis: "complete-negative-read"}};
  }
  function defaultRead(overrides) {
    return {scope: "order:O1", meaningRevision: "M1", query: "approval.pending@v1",
      epoch: 7, current: true, disclosed: true, readAllowed: true, routeQualified: true,
      operatorQualified: true, complete: true, rowsetCovered: true,
      predicateCovered: true, orderEquivalent: true, ...(overrides || {})};
  }
  const full = candidate([], defaultRead());
  check(full.render.rows.length === 0, "NC-01-structural-empty");
  check(full.semantic.status === "supported-in-scope", "NC-02-qualified-empty-scope-only");
  check(full.offer.enabled === false, "NC-03-claim-not-action-grant");
  const partial = candidate([], defaultRead({complete: false}));
  check(equal(full.render, partial.render) && partial.semantic.status === "not-established",
    "NC-04-same-empty-partial-insufficient");
  check(candidate([], defaultRead({rowsetCovered: false})).semantic.status === "not-established",
    "NC-05-rowset-coverage");
  check(candidate([], defaultRead({predicateCovered: false})).semantic.status === "not-established",
    "NC-06-hidden-predicate-dependency");
  check(candidate([], defaultRead({orderEquivalent: false})).semantic.status === "not-established",
    "NC-07-pushdown-filter-limit-order");
  check(candidate([], defaultRead({routeQualified: false})).semantic.status === "not-established",
    "NC-08-provider-route-unknown");
  check(candidate([], defaultRead({operatorQualified: false})).semantic.status === "not-established",
    "NC-09-role-operator-pin-unknown");
  check(candidate([], defaultRead({epoch: 8})).semantic.status === "not-established",
    "NC-10-selection-epoch-stale");
  check(candidate([], defaultRead({meaningRevision: "M2"})).semantic.status === "not-established",
    "NC-11-role-meaning-revised");
  check(candidate([], defaultRead({scope: "order:O2"})).semantic.status === "not-established",
    "NC-12-cross-context-reuse-stopped");
  const one = [{id: "P1", status: "pending"}];
  const found = candidate(one, defaultRead({complete: false, rowsetCovered: false}));
  check(found.semantic.status === "refuted" && found.offer.enabled === false,
    "NC-13-positive-witness-one-way-asymmetry");
  const hiddenA = candidate([], defaultRead({disclosed: false, complete: true}));
  const hiddenB = candidate(one, defaultRead({disclosed: false, complete: false}));
  check(equal(hiddenA, hiddenB) && !JSON.stringify(hiddenA).includes("P1"),
    "NC-14-non-disclosure-no-existence-leak");
  const before = candidate([], defaultRead());
  const after = candidate([], defaultRead());
  const changelog = [{time: 1, kind: "insert", id: "P1"}, {time: 2, kind: "remove", id: "P1"}];
  check(equal(before.render, after.render) && before.semantic.status === after.semantic.status &&
    changelog.length > 0, "NC-15-aba-current-empty-not-no-history");
  function invalidation(changed, knownCompleteChangeFootprint) {
    if (!knownCompleteChangeFootprint) return "recheck-unknown";
    return role.depends.some(p => changed.includes(p)) ? "recheck-required" : "unaffected-candidate";
  }
  check(invalidation(["approval.membership"], true) === "recheck-required",
    "NC-16-empty-row-still-depends-on-membership");
  check(invalidation(["approval.status"], true) === "recheck-required",
    "NC-17-nonoutput-predicate-dependency");
  check(invalidation(["unrelated.title"], true) === "unaffected-candidate",
    "NC-18-unrelated-field-not-blanket-invalidation");
  check(invalidation([], false) === "recheck-unknown",
    "NC-19-unobserved-change-is-not-no-change");
  const q2 = { ...role, meaningRevision: "M2" };
  check(candidate([], defaultRead(), q2).semantic.status === "not-established",
    "NC-20-new-meaning-does-not-inherit-evidence");
  if (typeof console !== "undefined") console.log(pass.length + " negative-claim synthetic checks PASS");
  return {count: pass.length, cases: pass};
})()
