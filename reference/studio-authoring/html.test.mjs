import test from "node:test";
import assert from "node:assert/strict";
import { renderStudioHTML } from "./html.mjs";

const revision = "sha256:" + "a".repeat(64);
const projection = {
  schema: "tei.reference.studio-projection/v0",
  modelRevision: revision, authority: "derived-readonly",
  focus: { subject: "relation:sales.order.lines" },
  views: {
    table: { status: "available", value: {
      id: "sales.order", attributes: [{ id: "sales.order.total",
        subject: "attribute:sales.order.total" }],
      relations: [{ id: "sales.order.lines", subject: "relation:sales.order.lines" }],
      validations: []
    }},
    er: { status: "available", nodes: [
      { id: "sales.order", subject: "data:sales.order" }
    ], edges: [{
      id: "sales.order.lines", subject: "relation:sales.order.lines",
      owner: { id: "sales.order", subject: "data:sales.order" },
      target: null, cardinality: "many", status: "unresolved"
    }]},
    function: { status: "available", items: [{
      id: "order_entry.order_lines", subject: "pattern:order_entry.order_lines",
      application: "order_entry", relation: { id: "sales.order.lines",
        subject: "relation:sales.order.lines" },
      dataReferenceStatus: "not-declared", relationReferenceStatus: "source-backed"
    }]},
    business: { status: "reference-only", authority: "noncanonical-reference",
      freshness: "unverified-no-model-revision-binding",
      items: [{ stepID: "BP-05", status: "reference-only",
        functionRefs: [], explicitMeaning: [{ id: "sales.order.lines",
          subject: "relation:sales.order.lines", status: "exact-id-matches-current-world" }] }] }
  },
  sourceNavigation: { status: "source-index-backed",
    artifactPath: "semantic/sales/order.tei",
    byteVerification: { status: "verified-at-read" },
    range: { start: { line: 5, column: 1 }, end: { line: 6, column: 1 } } }
};
test("HTML displays all four lenses, exact IDs and distinct authority", () => {
  const html = renderStudioHTML(projection, { sourceFromWorkspace: true });
  for (const id of ["table", "er", "function", "business", "source"])
    assert.ok(html.includes('id="' + id + '"'));
  assert.match(html, /relation:sales.order.lines/);
  assert.match(html, /noncanonical-reference/);
  assert.match(html, /verified-at-read/);
  assert.match(html, /Workspaceから再取得した正本の読み取り結果/);
  assert.match(html, /default-src &#39;none&#39;/);
  assert.doesNotMatch(html, /<script|<form|onclick=/i);
});
test("HTML escapes all untrusted source text and cannot execute it", () => {
  const malicious = '<img src=x onerror="alert(1)">&';
  const unsafe = structuredClone(projection);
  unsafe.focus.subject = malicious;
  unsafe.views.table.value.id = malicious;
  unsafe.views.business.items[0].stepID = malicious;
  unsafe.sourceNavigation.artifactPath = malicious;
  const html = renderStudioHTML(unsafe);
  assert.doesNotMatch(html, /<img/);
  assert.doesNotMatch(html, /onerror="/);
  assert.ok(html.includes('&lt;img src=x onerror=&quot;alert(1)&quot;&gt;&amp;'));
  assert.match(html, /外部の資料から作成した投影/);
});
test("reverse Business navigation is explicitly noncanonical and not editable", () => {
  const html = renderStudioHTML({
    schema: "tei.reference.studio-business-step-navigation/v0",
    subject: { id: "BP-05", scope: "reference-procedure:order-entry" },
    modelRevision: revision, authority: "noncanonical-reference",
    status: "unresolved", freshness: "unverified-no-model-revision-binding",
    candidates: [{ id: "sales.order", subject: "data:sales.order",
      focus: "data:sales.order", status: "unresolved" }]
  });
  assert.match(html, /編集可能/);
  assert.match(html, /unresolved/);
  assert.match(html, /受理済みの業務正本ではありません/);
});
test("proposal review never labels a preview as approved", () => {
  const html = renderStudioHTML({
    schema: "tei.reference.studio-review-edit/v0",
    modelRevision: revision, explicitAcceptance: "not-performed",
    sourceWriteByThisCommand: false,
    reviewConsistency: { status: "reviewed-consistent", readyToAccept: false },
    candidate: { id: "er-demo", origin: { kind: "ai" }, delta: [
      { op: "add-relation", addRelation: {
        data: "sales.order", id: "sales.order.test", symbol: "test",
        target: "<script>alert(1)</script>", cardinality: "one"
      }}
    ] },
    preview: { acceptable: true, semanticDiff: [{ change: "added",
      id: "sales.order.test" }], unresolved: [
      { code: "TARGET_PROJECTION_NOT_EVALUATED" }
    ], sourceEdits: [{
      path: "semantic/sales/order.tei", insertedText: "<img src=x>"
    }], stages: [{ name: "decode", status: "pass" }] },
    review: { reviewFingerprint: revision }
  });
  assert.match(html, /明示的な受理/);
  assert.match(html, /not-performed/);
  assert.match(html, /未適用/);
  assert.match(html, /TARGET_PROJECTION_NOT_EVALUATED/);
  assert.doesNotMatch(html, /<script>|<img/);
  assert.match(html, /&lt;script&gt;/);
});
test("HTML refuses arbitrary documents, with no default authority", () => {
  assert.throws(() => renderStudioHTML({ schema: "unknown" }),
    /UNSUPPORTED_STUDIO_HTML_SCHEMA/);
});
