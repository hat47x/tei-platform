import {Call} from "/wails/runtime.js";
import {createReadOnlyClient} from "../client.mjs";
import {createThreeViewClient, describeFocus} from "./client.mjs";

const call = (name, ...args) => Call.ByName(name, ...args);
const host = createReadOnlyClient(call);
const threeView = createThreeViewClient(call);

const workspace = document.querySelector("#workspace");
const sourcePath = document.querySelector("#source-path");
const navigateButton = document.querySelector("#navigate");
const state = document.querySelector("#state");
const coverage = document.querySelector("#coverage");
const focusPanel = document.querySelector("#focus");
const pinState = document.querySelector("#pin-state");
const pinButton = document.querySelector("#pin");
const unpinButton = document.querySelector("#unpin");
const lists = {
  data: document.querySelector("#view-data"),
  function: document.querySelector("#view-function"),
  business: document.querySelector("#view-business"),
};

// current is the last answer shown, with the workspace and path it answers
// for. focus is what the user looks at. pin is a target the user fixed on
// purpose: only the two pin buttons change it, a new selection never does.
let current = null;
let focus = "";
let pin = null;
let generation = 0;

function element(tag, text, attributes = {}) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = text;
  for (const [name, value] of Object.entries(attributes)) node.setAttribute(name, value);
  return node;
}

function table(headers, rows) {
  const node = element("table");
  const head = element("tr");
  for (const header of headers) head.append(element("th", header));
  node.append(head);
  for (const row of rows) {
    const line = element("tr");
    for (const cell of row) line.append(cell instanceof Node ? cell : element("td", String(cell)));
    node.append(line);
  }
  return node;
}

function subjectLabel(subject) {
  const name = subject.sourceFact.name || "(名称なし)";
  const id = subject.sourceFact.sourceIdStatus === "missing" ? "Source IDなし"
    : subject.sourceFact.sourceId + (subject.sourceFact.sourceIdStatus === "duplicated" ? "（重複）" : "");
  const same = (subject.sameNameOccurrences || []).length ? "・同名の別対象あり" : "";
  return name + " — " + subject.sourceFact.kind + " / " + id + same;
}

function renderViews() {
  for (const list of Object.values(lists)) list.replaceChildren();
  if (!current) return;
  for (const subject of current.value.subjects) {
    const item = element("li", undefined, subject.parentOccurrence ? {class: "nested"} : {});
    const button = element("button", subjectLabel(subject), {
      type: "button", "aria-pressed": String(subject.occurrence === focus),
    });
    button.addEventListener("click", () => { focus = subject.occurrence; render(); });
    item.append(button);
    lists[subject.view].append(item);
  }
}

function jump(occurrence, label) {
  const cell = element("td");
  const button = element("button", label, {type: "button"});
  button.addEventListener("click", () => { focus = occurrence; render(); });
  cell.append(button);
  return cell;
}

function renderFocus() {
  focusPanel.replaceChildren();
  const described = current ? describeFocus(current.value, focus) : null;
  if (!described) {
    focusPanel.append(element("p", "対象を選択してください。", {class: "note"}));
    return;
  }
  const {subject} = described;
  focusPanel.append(element("h3", subjectLabel(subject)));
  focusPanel.append(table(["層", "内容"], [
    ["原典のSource Fact", subject.sourceFact.kind + " @ " + current.value.source.path + " " + subject.sourceFact.locator],
    ["構造候補", subject.view + " 視点（" + (subject.candidate.classifications || []).join(", ") + "）"],
    ["受理済みMeaning", subject.meaning.status === "not-bound" ? "結び付けなし（この投影は意味を受理しません）" : subject.meaning.status],
  ]));

  focusPanel.append(element("h4", "関係"));
  focusPanel.append(described.relations.length ? table(["向き", "関係", "相手", "操作", "根拠"],
    described.relations.map((relation) => [
      relation.direction === "out" ? "→" : "←", relation.kind,
      jump(relation.other, relation.otherLabel), relation.operations,
      relation.nature === "source-fact" ? "原典に明記" : "複数の事実から導出した候補",
    ])) : element("p", "この投影には関係がありません（存在しないことの証明ではありません）。", {class: "note"}));

  focusPanel.append(element("h4", "CRUD（Functionの宣言）"));
  focusPanel.append(described.crud.length ? table(["Function", "Data", "C", "R", "U", "D", "注記"],
    described.crud.map((row) => [
      jump(row.function, row.functionName || "(名称なし)"), jump(row.data, row.dataName || "(名称なし)"),
      row.create ? "○" : "", row.read ? "○" : "", row.update ? "○" : "", row.delete ? "○" : "",
      row.conflicting ? "宣言が複数あり要確認" : "",
    ])) : element("p", "この対象のCRUD宣言はありません。業務Actionに固有のCRUDは表示しません。", {class: "note"}));

  if (described.fields.length) {
    focusPanel.append(element("h4", "項目と項目レベルの参照"));
    focusPanel.append(table(["項目", "Source", "参照元"], described.fields.map((field) => [
      field.sourceFact.name || "(名称なし)", field.sourceFact.locator,
      described.fieldReferences.filter((reference) => reference.field === field.occurrence)
        .map((reference) => reference.kind).join(", "),
    ])));
    const loose = described.fieldReferences.filter((reference) => !reference.field);
    if (loose.length) {
      focusPanel.append(table(["未解決の項目参照", "理由"],
        loose.map((reference) => [reference.kind + " → " + (reference.referencedId || "(IDなし)"), reference.resolution])));
    }
  }
}

function renderPin() {
  pinButton.disabled = !current || !focus;
  unpinButton.disabled = !pin;
  if (!pin) {
    pinState.textContent = "固定していません。";
    return;
  }
  const reported = current && current.value.pin;
  const status = !reported ? "未確認（次の読取りで確認します）"
    : reported.status === "held" ? "保持中"
    : reported.status === "stale-source-changed" ? "失効：固定後にSourceが変わりました。同じ名前やIDの対象へは付け替えません。"
    : "対象が見つかりません";
  pinState.textContent = pin.label + " — " + status;
}

function render() {
  renderViews();
  renderFocus();
  renderPin();
}

async function navigate() {
  const requested = {workspaceID: workspace.value, sourcePath: sourcePath.value.trim()};
  const mine = ++generation;
  state.textContent = "読取り中…";
  const response = await threeView.navigate(requested.workspaceID, requested.sourcePath, pin);
  // An answer for an earlier request is not shown over a later one.
  if (mine !== generation) return;
  if (!response.ok) {
    current = null;
    focus = "";
    state.textContent = "読み取れませんでした：" + response.failure.code;
    coverage.textContent = "";
    render();
    return;
  }
  current = {...requested, value: response.value};
  if (!response.value.subjects.some((subject) => subject.occurrence === focus)) focus = "";
  const c = response.value.source.coverage;
  state.textContent = "読取り完了（読取り専用）";
  coverage.textContent = "範囲：このSource一件。" + (c.status === "complete"
    ? "全要素を配置しました。"
    : "部分的です（未配置 " + c.unplacedConstructs + "・未知 " + c.unknownConstructs +
      "・未解決の関係 " + c.unresolvedRelations + "）。ここに無いことは、存在しないことを意味しません。");
  render();
}

pinButton.addEventListener("click", () => {
  const described = current ? describeFocus(current.value, focus) : null;
  if (!described) return;
  pin = {
    occurrence: focus, sourceArtifactDigest: current.value.source.artifactDigest,
    label: subjectLabel(described.subject),
  };
  navigate();
});
unpinButton.addEventListener("click", () => { pin = null; navigate(); });
navigateButton.addEventListener("click", navigate);
// Another Workspace or path is another source: the shown answer no longer
// applies, but the pin stays until the user releases it.
for (const input of [workspace, sourcePath]) {
  input.addEventListener("change", () => { generation++; current = null; focus = ""; state.textContent = ""; coverage.textContent = ""; render(); });
}

const listed = await host.listWorkspaces();
if (listed.ok) {
  for (const item of listed.value) workspace.append(element("option", item.label + " (" + item.id + ")", {value: item.id}));
  navigateButton.disabled = listed.value.length === 0;
} else {
  state.textContent = "Workspace一覧を取得できませんでした：" + listed.failure.code;
}
render();
