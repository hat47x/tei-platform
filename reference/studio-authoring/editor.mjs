// Browser editor emits a reviewable draft; it never writes Source or approves.
import { createHash } from "node:crypto";
import { indexWorld } from "./bridge.mjs";
const REVISION = /^sha256:[0-9a-f]{64}$/;
const escape = v => String(v ?? "").replace(/[&<>"']/g, ch => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
})[ch]);
const SCRIPT = "\"use strict\";\n(() => {\n  const form = document.getElementById(\"studio-edit\");\n  const output = document.getElementById(\"proposal-json\");\n  const notice = document.getElementById(\"notice\");\n  const field = id => document.getElementById(id).value;\n  form.addEventListener(\"submit\", event => {\n    event.preventDefault();\n    output.value = \"\";\n    notice.textContent = \"\";\n    try {\n      const mode = field(\"view\"), subject = field(\"subject\");\n      const edit = {\n        schema: \"tei.reference.studio-edit/v0\",\n        expectedModelRevision: field(\"model-revision\"),\n        id: field(\"candidate-id\").trim(), origin: \"studio\",\n        intent: field(\"intent\").trim(), view: mode, subject,\n        operation: mode === \"er\" ? \"add-relation\" : \"add-validation\"\n      };\n      if (!edit.id || !edit.intent) throw new Error(\"候補IDと変更意図を入力してください。\");\n      if (mode === \"er\") {\n        edit.symbol = field(\"symbol\").trim();\n        if (!edit.symbol) throw new Error(\"関連の識別子を入力してください。\");\n        edit.relationID = subject.slice(\"data:\".length) + \".\" + edit.symbol;\n        edit.targetData = field(\"target-data\");\n        edit.cardinality = field(\"cardinality\");\n      } else {\n        edit.validationID = field(\"validation-id\").trim();\n        if (!edit.validationID) throw new Error(\"規則の識別子を入力してください。\");\n        edit.validationKind = \"positive\";\n        const opt = document.getElementById(\"use-application\");\n        if (opt && opt.checked) {\n          const application = field(\"application\");\n          const placement = field(\"placement\").trim();\n          if (!application || !placement)\n            throw new Error(\"利用アプリと確認済みの定義ファイル位置が必要です。\");\n          edit.applicationUse = { application, placement };\n        }\n      }\n      output.value = JSON.stringify(edit, null, 2) + \"\\n\";\n      notice.textContent = \"未審査の変更候補JSONを生成しました。保存後、共通受理機構で確認してください。\";\n      output.focus();\n      output.select();\n    } catch (error) {\n      notice.textContent = \"入力を確認してください：\" + error.message;\n    }\n  });\n})();";
const STYLE = "body{font:1rem/1.65 system-ui,sans-serif;max-width:58rem;margin:auto;padding:1.5rem;color:#1d2630;background:#f8fafc}header,main,aside{background:white;border:1px solid #cbd5e1;border-radius:.6rem;padding:1rem 1.4rem;margin:1rem 0}h1{font-size:1.7rem}h2{font-size:1.2rem}label{display:block;margin:.8rem 0 .2rem;font-weight:650}input:not([type=checkbox]),select,textarea{box-sizing:border-box;width:100%;padding:.65rem;border:1px solid #64748b;border-radius:.3rem;font:inherit}input[readonly]{background:#edf2f7}button{padding:.75rem 1.4rem;background:#174b83;border:none;border-radius:.3rem;color:white;font:inherit;cursor:pointer}button:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible{outline:3px solid #3b82f6}textarea{height:18rem;font-family:ui-monospace,monospace}small,.warning{color:#713b04}code{overflow-wrap:anywhere}";
function control(id, label, value = "", opts = {}) {
  return '<label for="' + id + '">' + escape(label) + '</label><input id="' + id +
    '" value="' + escape(value) + '"' + (opts.readonly ? " readonly" : "") +
    (opts.required ? " required" : "") + ' autocomplete="off">';
}
function optionsHTML(values) {
  return values.map(v => '<option value="' + escape(v) + '">' + escape(v) + '</option>').join("");
}
export function renderStudioEditHTML(world, revision, focus) {
  if (!REVISION.test(revision)) throw new Error("INVALID_STUDIO_EDIT_REVISION");
  if (typeof focus !== "string") throw new Error("INVALID_STUDIO_EDIT_FOCUS");
  const sep = focus.indexOf(":");
  const kind = focus.slice(0, sep), id = focus.slice(sep + 1);
  const ix = indexWorld(world);
  if (!["data", "attribute"].includes(kind) || !id ||
    !(kind === "data" ? ix.data : ix.attributes).has(id))
    throw new Error("UNSUPPORTED_OR_UNRESOLVED_STUDIO_EDIT_FOCUS");
  const er = kind === "data", model = er ? ix.data.get(id) : ix.attributes.get(id);
  const targets = [...ix.data.keys()].sort();
  if (er && !targets.length) throw new Error("STUDIO_EDIT_TARGET_DATA_UNAVAILABLE");
  const namespace = model.namespace;
  const appChoices = [...ix.applications.values()].filter(app =>
    Array.isArray(app.semanticUses) && app.semanticUses.includes(namespace)
  ).map(app => app.id).sort();
  const mode = er ? "er" : "table";
  const fields = er
    ? control("symbol", "新しい関連の短い識別子", "", { required: true }) +
      '<label for="target-data">参照するデータ</label><select id="target-data">' +
      optionsHTML(targets) + '</select>' +
      '<label for="cardinality">関連の多重度</label>' +
      '<select id="cardinality"><option value="one">1件</option>' +
      '<option value="many">複数件</option></select>' +
      '<p>関連IDはデータIDと短い識別子を組み合わせます。既存IDとの衝突や意味の妥当性は審査時に検証します。</p>'
    : control("validation-id", "新しい検証規則の識別子", id + "_positive", { required: true }) +
      '<p>最初の編集機能では正値検証のみ扱います。</p>' +
      (appChoices.length
        ? '<label><input id="use-application" type="checkbox"> アプリでの利用を同じ候補に含める</label>' +
          '<label for="application">利用するアプリ</label><select id="application">' +
          optionsHTML(appChoices) + '</select>' +
          control("placement", "正本で確認したアプリ定義ファイルの位置")
        : '<p class="warning">この意味名前空間を利用するアプリが明示されていないため、利用宣言は提案できません。</p>');
  const digest = createHash("sha256").update(SCRIPT).digest("base64");
  const csp = "default-src 'none'; script-src 'sha256-" + digest +
    "'; style-src 'unsafe-inline'; connect-src 'none'; img-src 'none'; " +
    "object-src 'none'; base-uri 'none'; form-action 'none'";
  return '<!doctype html><html lang="ja"><head><meta charset="utf-8">' +
    '<meta name="viewport" content="width=device-width,initial-scale=1">' +
    '<meta http-equiv="Content-Security-Policy" content="' + escape(csp) + '">' +
    '<title>TEI Studio 編集候補の作成</title><style>' + STYLE +
    '</style></head><body><header><h1>Studio 編集候補の作成</h1>' +
    '<p>正本から取得した識別子を使い、未審査の変更候補を作成します。</p>' +
    '<p class="warning">この画面では正本を書き換えたり、変更を受理したりしません。</p></header>' +
    '<main><form id="studio-edit">' +
    control("subject", "選択した正本の識別子", focus, { readonly: true }) +
    control("model-revision", "画面生成時のモデル版", revision, { readonly: true }) +
    '<input id="view" type="hidden" value="' + mode + '">' +
    control("candidate-id", "変更候補の識別子", "", { required: true }) +
    control("intent", "変更意図", "", { required: true }) + fields +
    '<button type="submit">候補JSONを生成</button></form>' +
    '<p id="notice" role="status" aria-live="polite"></p>' +
    '<label for="proposal-json">生成した候補JSON（コピーしてJSONファイルに保存）</label>' +
    '<textarea id="proposal-json" readonly spellcheck="false"></textarea></main>' +
    '<aside><h2>審査と受理は別の操作です</h2>' +
    '<p>保存したJSONを <code>node reference/studio-authoring/cli.mjs review-edit -workspace DIR -edit-file FILE -format html</code> で審査してください。</p>' +
    '<p>モデル版が変わると審査は拒否されます。画面を開き直して変更を再検討してください。ブラウザ内の検査は便宜的なもので、正本の所有権・型・差分・影響・権限は共通受理機構が判定します。</p>' +
    '</aside><script>' + SCRIPT + '</script></body></html>\n';
}