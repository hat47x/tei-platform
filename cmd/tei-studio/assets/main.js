import {Call} from "/wails/runtime.js";
import {createReadOnlyClient} from "./client.mjs";
import {createPreviewScope} from "./preview-scope.mjs";

const api = createReadOnlyClient((name, ...args) => Call.ByName(name, ...args));
const workspace = document.querySelector("#workspace");
const state = document.querySelector("#state");
const output = document.querySelector("#output");
const xeadPath = document.querySelector("#xead-path");
const observeButton = document.querySelector("#observe");
const previewState = document.querySelector("#preview-state");
const previewResult = document.querySelector("#preview-result");
const previewScope = createPreviewScope();
const actions = ["catalog", "validate", "inspect"].map((id) => document.getElementById(id));

function display(label, response) {
  state.textContent = response.ok ? label + "：Operation応答（診断の有無は結果を参照）"
    : label + "：" + response.failure.code;
  output.textContent = JSON.stringify(response, null, 2);
}
function selectedID() { return workspace.value; }
function clearPreview() {
  previewState.textContent = "";
  previewResult.replaceChildren();
}
function text(parent, tag, value) {
  const node = document.createElement(tag);
  node.textContent = String(value ?? "");
  parent.appendChild(node);
  return node;
}
function renderObservation(value) {
  const observation = value?.observation;
  if (!observation || typeof observation !== "object") {
    previewResult.replaceChildren();
    previewState.textContent = "失敗: INVALID_HOST_RESPONSE";
    return;
  }
  previewResult.replaceChildren();
  const tables = Array.isArray(observation.tables) ? observation.tables : [];
  for (const table of tables) {
    text(previewResult, "h3", `${table.name || "（名称なし）"} — ${table.source?.sourceKind || "Table"}:${table.source?.sourceID || ""}`);
    const grid = document.createElement("table");
    const head = document.createElement("tr");
    for (const label of ["フィールド", "Source ID", "Data type ID"]) text(head, "th", label);
    grid.appendChild(head);
    for (const field of Array.isArray(table.fields) ? table.fields : []) {
      const row = document.createElement("tr");
      text(row, "td", field.name || "（名称なし）");
      text(row, "td", field.source?.sourceID || "");
      text(row, "td", field.dataType?.sourceID || "");
      grid.appendChild(row);
    }
    previewResult.appendChild(grid);
  }
  const diagnostics = Array.isArray(observation.diagnostics) ? observation.diagnostics : [];
  text(previewResult, "h3", "診断");
  if (!diagnostics.length) text(previewResult, "p", "診断なし");
  for (const diagnostic of diagnostics) {
    text(previewResult, "p", [diagnostic.code, diagnostic.element, diagnostic.attribute,
      `件数 ${diagnostic.occurrences}`].filter(Boolean).join(" / "));
  }
  previewState.textContent = "限定観察結果（読取専用・非Canonical）";
}
function invalidatePreview() {
  previewScope.reset();
  clearPreview();
  observeButton.disabled = !selectedID() || !xeadPath.value.trim();
}
workspace.addEventListener("change", invalidatePreview);
xeadPath.addEventListener("input", invalidatePreview);
observeButton.addEventListener("click", async () => {
  const id = selectedID();
  const relativePath = xeadPath.value.trim();
  if (!id || !relativePath) return;
  previewState.textContent = "観察中…";
  await previewScope.run(() => api.observeXead(id, relativePath), (response) => {
    if (!response?.ok) {
      previewResult.replaceChildren();
      const code = typeof response?.failure?.code === "string" ? response.failure.code : "INVALID_HOST_RESPONSE";
      previewState.textContent = `失敗: ${code}`;
      return;
    }
    renderObservation(response.value);
  }, () => {
    previewResult.replaceChildren();
    previewState.textContent = "失敗: HOST_TRANSPORT_FAILURE";
  });
});
for (const action of actions) {
  action.addEventListener("click", async () => {
    if (!selectedID()) return;
    for (const button of actions) button.disabled = true;
    state.textContent = "実行中…";
    let response;
    if (action.id === "catalog") response = await api.catalogWorkspace(selectedID());
    if (action.id === "validate") response = await api.validateWorkspace(selectedID());
    if (action.id === "inspect") response = await api.inspectSubject(selectedID(),
      document.querySelector("#kind").value, document.querySelector("#subject-id").value);
    display(action.id, response);
    for (const button of actions) button.disabled = !selectedID();
  });
}
const listed = await api.listWorkspaces();
if (!listed.ok) {
  display("workspaces", listed);
} else {
  for (const item of listed.value) {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.label + " (" + item.id + ")";
    workspace.appendChild(option);
  }
  display("workspaces", listed);
  for (const button of actions) button.disabled = !selectedID();
  observeButton.disabled = !selectedID() || !xeadPath.value.trim();
}
