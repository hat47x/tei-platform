// Static, read-only Studio projection/review document. No browser script,
// event handlers, forms, external assets, or independent Source authority.
const acceptedSchemas = new Set([
  "tei.reference.studio-projection/v0",
  "tei.reference.studio-business-step-navigation/v0",
  "tei.reference.studio-review-edit/v0"
]);
function escapeText(value) {
  return String(value ?? "").replace(/[&<>"']/g, c => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;",
    '"': "&quot;", "'": "&#39;"
  })[c]);
}
function str(value) { return escapeText(value === undefined || value === null ? "未指定" : value); }
function heading(title, id) {
  return '<h2 id="' + id + '">' + escapeText(title) + '</h2>';
}
function detail(label, value) {
  return '<div><dt>' + escapeText(label) + '</dt><dd>' + str(value) + '</dd></div>';
}
function facts(entries) {
  return '<dl class="facts">' + entries.map(([k, v]) => detail(k, v)).join("") + '</dl>';
}
function entries(items, render) {
  if (!Array.isArray(items) || !items.length)
    return '<p class="empty">該当する項目はありません。</p>';
  const max = 100;
  return '<ul>' + items.slice(0, max).map(x => '<li>' + render(x) + '</li>').join("") +
    '</ul>' + (items.length > max
      ? '<p class="warning">表示上限100件。残り' + (items.length - max) + '件は省略しています。JSONで確認してください。</p>'
      : '');
}
function ref(v) {
  if (v === null || v === undefined) return "未解決";
  return str(v.subject ?? v.id ?? v);
}
function record(v) {
  return '<code>' + ref(v) + '</code>';
}
function safeJSON(value) {
  return '<pre>' + escapeText(JSON.stringify(value ?? null, null, 2)) + '</pre>';
}
function section(id, name, status, body) {
  return '<section aria-labelledby="' + id + '-title" id="' + id + '">' +
    '<h2 id="' + id + '-title">' + escapeText(name) + '</h2>' +
    '<p class="state">状態：' + str(status) + '</p>' + body + '</section>';
}
function document(title, intro, body) {
  const style = [
    "body{font-family:system-ui,sans-serif;max-width:78rem;margin:auto;padding:1.4rem;color:#1c2732;background:#f7f9fc;line-height:1.65}",
    "header,section{background:white;border:1px solid #ccd7e1;border-radius:.65rem;padding:1.1rem 1.4rem;margin:1rem 0}",
    "h1{font-size:1.65rem}h2{font-size:1.18rem;margin-top:.2rem}code,pre{font-family:ui-monospace,monospace;overflow-wrap:anywhere}",
    "pre{white-space:pre-wrap;background:#eef2f6;padding:.8rem;border-radius:.35rem;max-height:30rem;overflow:auto}",
    "nav a{margin-right:1.2rem;display:inline-block;color:#174d8c}nav{margin:1rem 0}",
    ".facts{display:grid;grid-template-columns:repeat(auto-fit,minmax(16rem,1fr));gap:.6rem}",
    "dt{font-size:.85rem;color:#43566a}dd{margin:0;overflow-wrap:anywhere;font-weight:600}",
    ".state,.warning{font-weight:600;color:#713b04}.empty{color:#556474}",
    "li{padding:.25rem 0}a:focus-visible{outline:3px solid #1d5d9b}footer{font-size:.9rem}"
  ].join("");
  return '<!doctype html><html lang="ja"><head><meta charset="utf-8">' +
    '<meta name="viewport" content="width=device-width,initial-scale=1">' +
    '<meta http-equiv="Content-Security-Policy" content="default-src &#39;none&#39;; ' +
    'style-src &#39;unsafe-inline&#39;; base-uri &#39;none&#39;; form-action &#39;none&#39;">' +
    '<title>' + escapeText(title) + '</title><style>' + style + '</style></head><body>' +
    '<header><h1>' + escapeText(title) + '</h1><p>' + escapeText(intro) + '</p>' +
    '<p class="warning">確認専用。編集・受理・権限付与・実環境への適用は行いません。</p></header>' +
    body + '<footer>TEI Studio参照表示。表示内容だけでは人間による受理、実行先の整合性、並行編集の安全性を証明しません。</footer>' +
    '</body></html>\n';
}
function lens(view, sourceFromWorkspace) {
  const views = view.views || {};
  const table = views.table || {}, er = views.er || {};
  const fn = views.function || {}, business = views.business || {};
  const src = view.sourceNavigation || {};
  const focus = view.focus || {};
  const navigation = '<nav aria-label="視点切替">' +
    [['table', "表"], ['er', "ER図"], ['function', "機能"], ['business', "業務"], ['source', "元の定義"]]
      .map(([id, label]) => '<a href="#' + id + '">' + escapeText(label) + '</a>').join("") +
    '</nav>';
  const tableBody = table.value
    ? facts([["データ", table.value.id]]) +
      heading("属性", "attributes") + entries(table.value.attributes, record) +
      heading("関連", "table-relations") + entries(table.value.relations, record) +
      heading("入力規則", "table-validations") + entries(table.value.validations,
        v => record(v) + ' ／評価対象：' +
          entries(v.applicationSelection, x => str(x.application) + ' (' + str(x.status) + ')'))
    : '<p class="empty">この選択要素に対応する表はありません。</p>';
  const erBody = heading("対象データ", "er-nodes") + entries(er.nodes, record) +
    heading("関連の候補", "er-edges") + entries(er.edges,
      r => record(r) + ' ／起点 ' + record(r.owner) +
        ' ／対象 ' + record(r.target) + ' ／多重度 ' + str(r.cardinality) +
        ' ／状態 ' + str(r.status));
  const functionBody = entries(fn.items, p =>
    record(p) + ' ／アプリ ' + str(p.application) +
    ' ／データ ' + record(p.data) + ' ／関連 ' + record(p.relation) +
    ' ／参照状態 ' + str(p.dataReferenceStatus) + '、' + str(p.relationReferenceStatus));
  const businessBody = '<p class="warning">業務手順は参照用資料に由来します。正本への受理と同一版の結合は未確認です。</p>' +
    facts([["出所", business.authority ?? "unknown"],
      ["鮮度", business.freshness ?? "未証明"]]) +
    entries(business.items, item =>
      '<strong>手順 ' + str(item.stepID) + '</strong> ／状態 ' + str(item.status) +
      ' ／機能 ' + entries(item.functionRefs,
        r => record(r) + ' (' + str(r.status) + ')') +
      ' ／意味参照 ' + entries(item.explicitMeaning,
        r => record(r) + ' (' + str(r.status) + ')')) +
    (business.diagnostics?.length
      ? '<details><summary>未解決事項</summary>' + safeJSON(business.diagnostics) + '</details>' : '');
  const sourceBody = facts([
    ["元定義の照合結果", src.status ?? "unresolved"],
    ["ファイル位置", src.artifactPath ?? "なし"],
    ["指紋検査", src.byteVerification?.status ?? "not-checked"],
    ["原文の開始", src.range?.start
      ? String(src.range.start.line) + "行 " + src.range.start.column + "列" : "なし"],
    ["原文の終了", src.range?.end
      ? String(src.range.end.line) + "行 " + src.range.end.column + "列" : "なし"]
  ]) + '<p>' + str(src.caveat ?? src.reason ?? "元定義は特定されていません。") + '</p>';
  const origin = sourceFromWorkspace
    ? "Workspaceから再取得した正本の読み取り結果"
    : "外部の資料から作成した投影（正本との同一性は未検証）";
  return document("Studioの視点別参照",
    "同じIDから表・ER・機能・業務を読み取り専用で確認します。",
    facts([["選択中のID", focus.subject ?? "未指定"],
      ["モデルの版", view.modelRevision],
      ["Worldの出所", origin],
      ["投影の権限", view.authority]]) +
    navigation +
    section("table", "表", table.status, tableBody) +
    section("er", "ER図", er.status, erBody) +
    section("function", "機能", fn.status, functionBody) +
    section("business", "業務", business.status, businessBody) +
    section("source", "元の定義", src.status, sourceBody));
}
function businessStep(view) {
  return document("業務手順からの逆向き参照",
    "参照用の業務手順に含まれる明示的なIDだけを照合します。",
    facts([["手順ID", view.subject?.id], ["手順のスコープ", view.subject?.scope],
      ["モデルの版", view.modelRevision], ["手順の権限", view.authority],
      ["参照状態", view.status], ["鮮度", view.freshness],
      ["編集可能", "いいえ"]]) +
    section("business", "参照候補", view.status,
      entries(view.candidates, c =>
        record(c) + ' ／状態 ' + str(c.status) + ' ／出所 ' + str(c.via))) +
    '<section><h2>受理の境界</h2><p>この手順は受理済みの業務正本ではありません。共通の受理機構を経由せず編集できません。</p></section>');
}
function reviewEdit(view) {
  const preview = view.preview || {}, review = view.review || {};
  const stages = preview.stages || [];
  return document("Studioの変更提案と審査",
    "Sourceへの変更はこの文書では実行されません。差分・影響・未解決事項を確認します。",
    facts([["変更案", view.candidate?.id], ["モデルの版", view.modelRevision],
      ["元の経路", view.candidate?.origin?.kind],
      ["共通受理のプレビュー", preview.acceptable === true ? "受理可能と判定" : "受理不可／未確認"],
      ["明示的な受理", view.explicitAcceptance],
      ["正本への書込み", view.sourceWriteByThisCommand === false ? "なし" : "未確認"],
      ["一貫性", view.reviewConsistency?.status],
      ["審査指紋", review.reviewFingerprint]]) +
    section("candidate", "意味の変更案", "未受理",
      entries(view.candidate?.delta, x => safeJSON(x))) +
    section("diff", "意味の差分", "審査用",
      entries([...(preview.semanticDiff || []), ...(preview.applicationDiff || [])],
        x => safeJSON(x))) +
    section("impact", "影響と未解決事項", "実行先の確認は別途必要",
      heading("影響", "impacts") + entries(preview.impact, x => safeJSON(x)) +
      heading("未解決事項", "unresolved") + entries(preview.unresolved, x => safeJSON(x))) +
    section("review", "受理機構の段階別結果", "参照のみ",
      entries(stages, x => str(x.name) + ' ／' + str(x.status) +
        (x.diagnostics?.length ? safeJSON(x.diagnostics) : ""))) +
    section("source", "予定される元定義の変更", "未適用",
      entries(preview.sourceEdits, x =>
        '<strong>' + str(x.path) + '</strong>' + safeJSON({
          layer: x.layer, beforeFingerprint: x.beforeFingerprint,
          afterFingerprint: x.afterFingerprint, insertedText: x.insertedText
        }))));
}
export function renderStudioHTML(result, options = {}) {
  if (!result || !acceptedSchemas.has(result.schema))
    throw new Error("UNSUPPORTED_STUDIO_HTML_SCHEMA");
  if (result.schema === "tei.reference.studio-projection/v0")
    return lens(result, options.sourceFromWorkspace === true);
  if (result.schema === "tei.reference.studio-business-step-navigation/v0")
    return businessStep(result);
  return reviewEdit(result);
}
