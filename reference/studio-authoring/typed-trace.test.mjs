import test from "node:test";
import assert from "node:assert/strict";
import { bindTypedTrace } from "./typed-trace.mjs";

const hash = character => "sha256:" + character.repeat(64);
const key = { scope: "studio-observation", subject: {
  kind: "semantic-relation", id: "sales.order.lines"
} };
const range = { start: { offset: 12 }, end: { offset: 24 } };
const view = {
  schema: "tei.reference.studio-projection/v0", modelRevision: hash("a"),
  focus: { kind: "relation", id: "sales.order.lines",
    subject: "relation:sales.order.lines" },
  sourceNavigation: { status: "source-index-backed",
    artifactPath: "semantic/sales/order.tei",
    artifactFingerprint: hash("b"), range }
};
const report = () => ({
  schemaVersion: "tei.cli.trace/v0", ok: true, result: {
    schemaVersion: "tei.operations.trace/v0",
    snapshotFingerprint: hash("c"), worldFingerprint: hash("d"),
    sourceIndexFingerprint: hash("e"),
    authorityLimits: ["business-procedure-not-in-current-world"],
    projectionUnresolved: [],
    source: { status: "located", key,
      authorityLimits: ["source-location-is-not-write-authority"],
      references: [{ role: "definition", artifactPath: "semantic/sales/order.tei", range }],
      artifacts: [{ path: "semantic/sales/order.tei", fingerprint: hash("b") }] },
    trace: {
      schemaVersion: "tei.modelref.trace/v0", subject: key,
      authority: "candidate-read-only", snapshotFingerprint: hash("f"),
      direction: "both", mode: "full", total: 1, unresolved: 0,
      rows: [{ status: "resolved", edge: {
        from: key, to: { scope: "studio-observation",
          subject: { kind: "semantic-data", id: "sales.line" } },
        relation: "targets" },
        resolvedTo: { scope: "studio-observation",
          subject: { kind: "semantic-data", id: "sales.line" } }
      }]
    }
  }
});

test("Go typed reference remains read-only and matches Studio exact Source anchor", () => {
  const out = bindTypedTrace(view, report(), key.scope);
  assert.equal(out.status, "reference-only");
  assert.equal(out.sourceConsistency, "exact-anchor-match");
  assert.equal(out.modelRevision, view.modelRevision);
  assert.equal(out.total, 1);
  assert.equal(out.authority, "candidate-read-only");
  assert.equal(out.scopeAuthority, "caller-selected-observation-only");
  assert.doesNotMatch(JSON.stringify(out), /accepted-procedure/);
});
test("scope, kind, missing authority limits, and partial result are rejected", () => {
  for (const changed of [
    x => { x.result.trace.subject.scope = "other"; },
    x => { x.result.source.key.subject.kind = "semantic-data"; },
    x => { x.result.authorityLimits = []; },
    x => { x.result.trace.rows = []; },
    x => { x.result.trace.nextOffset = 1; },
    x => { x.result.trace.authority = "accepted"; }
  ]) {
    const forged = report();
    changed(forged);
    assert.throws(() => bindTypedTrace(view, forged, key.scope),
      /STUDIO_TYPED_TRACE_INVALID/);
  }
});
test("stale Source anchor or invalid reference never becomes qualified", () => {
  const stale = report();
  stale.result.source.artifacts[0].fingerprint = hash("9");
  const out = bindTypedTrace(view, stale, key.scope);
  assert.equal(out.status, "unresolved");
  assert.equal(out.sourceConsistency, "unresolved");
  const invalid = report();
  invalid.result.trace.rows[0].status = "unresolved";
  delete invalid.result.trace.rows[0].resolvedTo;
  invalid.result.trace.rows[0].issue = "wrong-kind";
  invalid.result.trace.unresolved = 1;
  assert.equal(bindTypedTrace(view, invalid, key.scope).status, "unresolved");
  invalid.result.trace.unresolved = 0;
  assert.throws(() => bindTypedTrace(view, invalid, key.scope),
    /STUDIO_TYPED_TRACE_INVALID/);
});
