import test from "node:test";
import assert from "node:assert/strict";
import {bindPublicTraitClaims} from "./trait-attachments.mjs";
import {projectPublicTeaSlice} from "./public-slice.mjs";
const snapshot = "source:R1|role:P1|policy:G1";
const subject = {scope: "sales", kind: "semantic-data", id: "order-status"};
const viewId = role => JSON.stringify([role, subject.scope, subject.kind, subject.id]);
const projection = () => projectPublicTeaSlice({scope: "sales", subjects: [
  {scope: subject.scope, kind: subject.kind, id: subject.id,
    roleStates: {data: "accepted", function: "accepted", business: "accepted"}},
], relations: []});
const c = (claimId, role, lane, state) => ({claimId, scope: subject.scope, kind: subject.kind, subjectId: subject.id,
  role, traitId: "closed-domain", lane, state, evidenceId: `E-${claimId}`, snapshot});
// Attachments refer to one Subject. They are not computed conformance results.
const envelope = () => ({scope: "sales", snapshot, claims: [
  c("business-intent", "business", "intent", "proposal"),
  c("java-enum", "data", "observation", "source-observed"),
  c("db-no-check", "data", "observation", "source-observed"),
  c("writer-unknown", "function", "assessment", "not-proven"),
]});

test("T10: three views receive distinct independent intent/observation/assessment records", () => {
  const out = bindPublicTraitClaims(projection(), envelope());
  assert.equal(out.bindings.length, 4);
  assert.equal(out.bindings.filter(x => x.lane === "intent").length, 1);
  assert.equal(out.bindings.filter(x => x.lane === "assessment")[0].state, "not-proven");
  assert.equal(out.bindings.filter(x => x.lane === "observation").length, 2);
  assert.equal(out.bindings.find(x => x.claimId === "business-intent").viewId, viewId("business"));
  assert.equal(out.allSourceCompleteness, "not-proven");
});

test("T10: static enum or DB column never manufactures a conformance claim", () => {
  const e = envelope();
  e.claims = e.claims.filter(x => x.lane === "observation");
  const out = bindPublicTraitClaims(projection(), e);
  assert.equal(out.bindings.length, 2);
  assert.equal(out.bindings.some(x => x.lane === "assessment" || x.state === "proposal"), false);
});

test("T11: no strong-attribute or business-identity annotation is inferred", () => {
  const out = bindPublicTraitClaims(projection(), envelope());
  assert.equal(out.bindings.some(x => x.traitId === "strong-attribute" ||
    x.traitId === "business-identity"), false);
});

test("T12: old snapshot cannot be rebound to current projection", () => {
  const e = envelope();
  e.claims[0].snapshot = "source:R0|role:P1|policy:G1";
  assert.throws(() => bindPublicTraitClaims(projection(), e), /unqualified_trait_attachment/);
});

test("T12: scope mismatch cannot bind same display name", () => {
  const e = envelope();
  e.claims[0].scope = "billing";
  assert.throws(() => bindPublicTraitClaims(projection(), e), /unqualified_trait_attachment/);
});

test("T13: a private trait not represented in public projection cannot be attached", () => {
  const e = envelope();
  e.claims.push({...c("private-trait", "business", "assessment", "not-proven"), subjectId: "hidden-status"});
  assert.throws(() => bindPublicTraitClaims(projection(), e), err =>
    err.message === "unqualified_trait_attachment" && !err.message.includes("hidden"));
});

test("same selected public claims bind deterministically independent of input order", () => {
  const a = envelope(), b = envelope();
  b.claims.reverse();
  assert.deepStrictEqual(bindPublicTraitClaims(projection(), a), bindPublicTraitClaims(projection(), b));
});

test("duplicate claim IDs cannot hide conflicting roles", () => {
  const e = envelope();
  e.claims.push({...c("java-enum", "business", "intent", "proposal")});
  assert.throws(() => bindPublicTraitClaims(projection(), e), /unqualified_trait_attachment/);
});

test("unknown view and unqualified metadata fail closed", () => {
  const e = envelope();
  e.claims[0].role = "unknown-role";
  assert.throws(() => bindPublicTraitClaims(projection(), e), /unqualified_trait_attachment/);
  const extra = envelope();
  extra.claims[0].privateSourcePath = "/private/raw.tei";
  assert.throws(() => bindPublicTraitClaims(projection(), extra), /unqualified_trait_attachment/);
});

test("input getter is rejected without invoking it", () => {
  const e = envelope();
  let calls = 0;
  Object.defineProperty(e.claims[0], "claimId", {enumerable: true, get() {calls++; return "bad";}});
  assert.throws(() => bindPublicTraitClaims(projection(), e), /unqualified_trait_attachment/);
  assert.equal(calls, 0);
});
