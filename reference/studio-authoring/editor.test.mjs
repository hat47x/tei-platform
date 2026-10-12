import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { runInNewContext } from "node:vm";
import { renderStudioEditHTML } from "./editor.mjs";
import { candidateFromEdit } from "./bridge.mjs";

const rev = "sha256:" + "a".repeat(64);
const nextRev = "sha256:" + "b".repeat(64);
function world() {
  return {
    semantics: [{ id: "sales", data: [
      { id: "sales.order", attributes: [] },
      { id: "sales.order_line",
        attributes: [{ id: "sales.order_line.price", type: "std.decimal" }] }
    ] }],
    applications: [{ id: "order_entry", semanticUses: ["sales"], patterns: [] }]
  };
}
function simulate(html, data, useApplication = false) {
  const match = html.match(/<script>([\s\S]*?)<\/script>/);
  assert.ok(match, "the browser logic is embedded in the offline document");
  const digest = createHash("sha256").update(match[1]).digest("base64");
  assert.ok(html.includes("sha256-" + digest), "CSP pins exactly these script bytes");
  assert.match(html, /connect-src &#39;none&#39;/);
  const values = {
    "model-revision": rev,
    "candidate-id": "editing-demo",
    "intent": "定義から構造を追加する",
    "subject": data.subject,
    "view": data.view, ...data
  };
  const elements = new Map(Object.entries(values).map(([k, v]) =>
    [k, { value: v, checked: k === "use-application" && useApplication }]));
  const output = { value: "", focus(){}, select(){} };
  const notice = { textContent: "" };
  elements.set("proposal-json", output);
  elements.set("notice", notice);
  let submit;
  elements.set("studio-edit", { addEventListener(name, fn) {
    assert.equal(name, "submit");
    submit = fn;
  } });
  const document = { getElementById: id => elements.get(id) || null };
  runInNewContext(match[1], { document, JSON });
  assert.equal(typeof submit, "function");
  let prevented = false;
  submit({ preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  return { output: output.value, notice: notice.textContent };
}
test("ER form emits a typed, revision-bound proposal accepted by the existing bridge", () => {
  const html = renderStudioEditHTML(world(), rev, "data:sales.order_line");
  assert.match(html, /<form id="studio-edit"/);
  assert.match(html, /ブラウザ内の検査は便宜的/);
  assert.doesNotMatch(html, /<iframe|src="https?:|action="https?:/);
  const { output, notice } = simulate(html, {
    subject: "data:sales.order_line", view: "er",
    symbol: "order", "target-data": "sales.order", cardinality: "one"
  });
  const edit = JSON.parse(output);
  assert.equal(edit.expectedModelRevision, rev);
  assert.equal(edit.relationID, "sales.order_line.order");
  assert.equal(edit.origin, "studio");
  assert.match(notice, /未審査/);
  const candidate = candidateFromEdit(world(), rev, edit);
  assert.equal(candidate.delta[0].addRelation.id, edit.relationID);
  assert.throws(() => candidateFromEdit(world(), nextRev, edit),
    /STUDIO_EDIT_STALE_MODEL_REVISION/);
});
test("Table form optionally binds application use without Source writes", () => {
  const html = renderStudioEditHTML(world(), rev, "attribute:sales.order_line.price");
  const input = {
    subject: "attribute:sales.order_line.price", view: "table",
    "validation-id": "sales.order_line.price_positive",
    application: "order_entry",
    placement: "applications/order-entry/application.tei"
  };
  const alone = JSON.parse(simulate(html, input).output);
  assert.equal(alone.applicationUse, undefined);
  assert.equal(candidateFromEdit(world(), rev, alone).delta.length, 1);
  const paired = JSON.parse(simulate(html, input, true).output);
  assert.deepEqual(paired.applicationUse, {
    application: "order_entry",
    placement: "applications/order-entry/application.tei"
  });
  assert.equal(candidateFromEdit(world(), rev, paired).delta.length, 2);
});
test("Standalone HTML escapes untrusted subject names, never adds authority", () => {
  const w = world(), attack = 'sales.<img src=x onerror="1">';
  w.semantics[0].data.push({ id: attack, attributes: [] });
  const html = renderStudioEditHTML(w, rev, "data:" + attack);
  assert.match(html, /sales\.\&lt;img src=x onerror=&quot;1&quot;&gt;/);
  assert.doesNotMatch(html, /<img|<form[^>]+action=/i);
  assert.match(html, /form-action &#39;none&#39;/);
});
test("unsupported, missing and invalid-focus inputs are rejected", () => {
  const w = world();
  assert.throws(() => renderStudioEditHTML(w, rev, "pattern:foo"),
    /UNSUPPORTED_OR_UNRESOLVED_STUDIO_EDIT_FOCUS/);
  assert.throws(() => renderStudioEditHTML(w, rev, "data:missing"),
    /UNSUPPORTED_OR_UNRESOLVED_STUDIO_EDIT_FOCUS/);
  assert.throws(() => renderStudioEditHTML(w, "invalid", "data:sales.order"),
    /INVALID_STUDIO_EDIT_REVISION/);
});
test("Form does not create an application use without an explicit namespace declaration", () => {
  const w = world();
  w.applications[0].semanticUses = [];
  const html = renderStudioEditHTML(w, rev, "attribute:sales.order_line.price");
  assert.doesNotMatch(html, /id="use-application"/);
  const edit = JSON.parse(simulate(html, {
    view: "table", subject: "attribute:sales.order_line.price",
    "validation-id": "sales.order_line.price_positive"
  }).output);
  assert.equal(edit.applicationUse, undefined);
});
