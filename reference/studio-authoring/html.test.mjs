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
test("review HTML surfaces eight categories and unknown coverage", () => {
  const codes = ["A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8"];
  const result = { schema: "tei.reference.studio-review-edit/v0",
    explicitAcceptance: "not-performed", sourceWriteByThisCommand: false,
    review: { coverage: { available: 2, partial: 4, notEvaluated: 2 },
      sections: codes.map((code, i) => ({
        code, title: code + "審査", coverage: i < 2 ? "available"
          : i < 6 ? "partial" : "not-evaluated",
        summary: "根拠-" + i, items: ["項目-" + code], sources: ["preview." + code],
        note: i === 4 ? "<img src=x onerror=alert(1)>" : ""
      }))
    }
  };
  const page = renderStudioHTML(result);
  for (const code of codes) assert.match(page, new RegExp(code + "　"));
  assert.match(page, /未評価 2/);
  assert.match(page, /根拠の出所/);
  assert.match(page, /&lt;img src=x onerror=alert/);
  assert.doesNotMatch(page, /<img/);
  assert.match(page, /not-performed/);
});
test("review HTML marks missing categories and duplicate results as unverified", () => {
  const result = { schema: "tei.reference.studio-review-edit/v0",
    explicitAcceptance: "not-performed", sourceWriteByThisCommand: false,
    review: { coverage: { available: 8, partial: 0, notEvaluated: 0 },
      sections: [
        { code: "A1", title: "正常", coverage: "available" },
        { code: "A1", title: "重複", coverage: "available" },
        { code: "A2", title: "正常", coverage: "available" }, null
      ]
    }
  };
  const page = renderStudioHTML(result);
  assert.match(page, /区分不足・重複・形式不正/);
  assert.match(page, /同一区分が重複しました/);
  assert.match(page, /当該区分を取得できません/);
  assert.match(page, /A8　/);
  assert.match(page, /not-evaluated/);
});
test("future review categories remain visible and explicitly unverified", () => {
  const page = renderStudioHTML({
    schema: "tei.reference.studio-review-edit/v0",
    explicitAcceptance: "not-performed", sourceWriteByThisCommand: false,
    preview: { acceptable: true },
    review: { coverage: { available: 9, partial: 0, notEvaluated: 0 },
      sections: [
        ...["A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8"].map(code => ({
          code, title: code + " 項目", coverage: "available"
        })),
        { code: "A9", title: "将来の追加審査", coverage: "available",
          items: ["<script>bad</script>"] }
      ]
    }
  });
  assert.match(page, /A9　将来の追加審査/);
  assert.match(page, /未知の審査区分/);
  assert.match(page, /Studioが未対応の審査区分/);
  assert.match(page, /評価範囲：not-evaluated/);
  assert.match(page, /&lt;script&gt;bad&lt;\/script&gt;/);
  assert.doesNotMatch(page, /<script>/);
});
test("review handoff carries the full unchanged candidate and exact fingerprint", () => {
  const candidate = {
    schema: "tei.reference.change-candidate/v0", id: "relation-proposal",
    origin: { kind: "studio" }, intent: "定義を明示する",
    baseModelRevision: revision, evidence: [],
    delta: [{ op: "add-relation", addRelation: {
      data: "sales.order_line", id: "sales.order_line.order",
      symbol: "order", target: "sales.order", cardinality: "one"
    }}]
  };
  const p = "sha256:" + "b".repeat(64), reviewFp = "sha256:" + "c".repeat(64);
  const html = renderStudioHTML({
    schema: "tei.reference.studio-review-edit/v0",
    modelRevision: revision, sourceRevisionAfterReview: revision,
    candidate, explicitAcceptance: "not-performed",
    sourceWriteByThisCommand: false,
    preview: { acceptable: true, candidateId: candidate.id,
      baseModelRevision: revision, currentModelRevision: revision,
      candidateFingerprint: p },
    review: { candidateId: candidate.id, candidateFingerprint: p,
      reviewFingerprint: reviewFp, previewAcceptable: true },
    reviewConsistency: { status: "reviewed-consistent",
      readyToAccept: false, candidateFingerprint: p,
      reviewFingerprint: reviewFp }
  });
  assert.match(html, /id="handoff"/);
  assert.match(html, /id="handoff-candidate"/);
  assert.match(html, /候補・版・審査の照合済み／人の判断は未実施/);
  assert.match(html, /relation-proposal/);
  assert.match(html, /sales.order_line.order/);
  assert.match(html, /tei.reference.change-candidate\/v0/);
  assert.match(html, /共通受理機構による再審査/);
  assert.match(html, /-candidate &quot;\$CANDIDATE&quot;/);
  assert.match(html, /人が差分・影響・権限・未解決点を確認/);
  assert.doesNotMatch(html, /<form|<script|onclick=/i);
});
test("handoff never presents an unacceptable or unbound review as verified", () => {
  const result = {
    schema: "tei.reference.studio-review-edit/v0", modelRevision: revision,
    sourceRevisionAfterReview: revision, explicitAcceptance: "not-performed",
    sourceWriteByThisCommand: false,
    candidate: { schema: "tei.reference.change-candidate/v0",
      id: "untrusted", baseModelRevision: revision },
    preview: { acceptable: false, candidateId: "untrusted",
      baseModelRevision: revision, currentModelRevision: revision,
      candidateFingerprint: "sha256:" + "d".repeat(64) },
    review: { candidateId: "untrusted", previewAcceptable: false,
      candidateFingerprint: "sha256:" + "d".repeat(64),
      reviewFingerprint: "sha256:" + "e".repeat(64) },
    reviewConsistency: { status: "not-acceptable", readyToAccept: false,
      candidateFingerprint: "sha256:" + "d".repeat(64),
      reviewFingerprint: "sha256:" + "e".repeat(64) }
  };
  const h1 = renderStudioHTML(result);
  assert.match(h1, /未確認・不整合のため受理の案内を保留/);
  assert.doesNotMatch(h1, /候補・版・審査の照合済み／人の判断は未実施/);
  const changed = structuredClone(result);
  changed.preview.acceptable = true;
  changed.review.previewAcceptable = true;
  changed.reviewConsistency.status = "reviewed-consistent";
  changed.sourceRevisionAfterReview = "sha256:" + "f".repeat(64);
  assert.match(renderStudioHTML(changed), /未確認・不整合のため受理の案内を保留/);
});
test("review handoff HTML-encodes malicious candidate JSON without dropping details", () => {
  const malicious = '<svg onload="alert(1)">&';
  const html = renderStudioHTML({
    schema: "tei.reference.studio-review-edit/v0",
    candidate: { id: malicious, intent: malicious,
      origin: { kind: "ai" }, delta: [{ arbitrary: malicious }] }
  });
  assert.match(html, /&lt;svg onload=&quot;alert\(1\)&quot;&gt;&amp;/);
  assert.match(html, /id="handoff-candidate"/);
  assert.doesNotMatch(html, /<svg|<script|onclick=/i);
});
test("HTML refuses arbitrary documents, with no default authority", () => {
  assert.throws(() => renderStudioHTML({ schema: "unknown" }),
    /UNSUPPORTED_STUDIO_HTML_SCHEMA/);
});
