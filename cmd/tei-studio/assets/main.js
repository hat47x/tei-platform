import {Call} from "/wails/runtime.js";
import {createReadOnlyClient} from "./client.mjs";
import {createPreviewScope} from "./preview-scope.mjs";
import {summarizeView} from "./view-format.mjs";

const api = createReadOnlyClient((name, ...args) => Call.ByName(name, ...args));
const workspace = document.querySelector("#workspace");
const state = document.querySelector("#state");
const output = document.querySelector("#output");
const xeadPath = document.querySelector("#xead-path");
const observeButton = document.querySelector("#observe");
const previewState = document.querySelector("#preview-state");
const previewResult = document.querySelector("#preview-result");
const previewScope = createPreviewScope();
const applicationInput = document.querySelector("#application-id");
const patternInput = document.querySelector("#pattern-id");
const viewApplicationButton = document.querySelector("#view-application");
const viewPatternButton = document.querySelector("#view-pattern");
const viewPackageButton = document.querySelector("#view-package");
const packageInput = document.querySelector("#package-id");
const viewState = document.querySelector("#view-state");
const viewOutput = document.querySelector("#view-output");
const capabilityInput = document.querySelector("#command-capability");
const commandInputs = document.querySelector("#command-inputs");
const commandPeriods = document.querySelector("#command-periods");
const planCommandButton = document.querySelector("#plan-command");
const executeCommandButton = document.querySelector("#execute-command");
const commandState = document.querySelector("#command-state");
const commandOutput = document.querySelector("#command-output");
const actions = ["catalog", "validate", "inspect"].map((id) => document.getElementById(id));

function display(label, response) {
  state.textContent = response.ok ? label + "：Operation応答（診断の有無は結果を参照）"
    : label + "：" + response.failure.code;
  output.textContent = JSON.stringify(response, null, 2);
}
function selectedID() { return workspace.value; }
function parseInputs(value) {
  const out = {};
  for (const part of value.split(",")) {
    const index = part.indexOf("=");
    if (index <= 0) continue;
    out[part.slice(0, index).trim()] = part.slice(index + 1).trim();
  }
  return out;
}
function displayCommand(label, response) {
  commandState.textContent = response.ok ? label + "：Command応答" : label + "：" + response.failure.code;
  commandOutput.textContent = JSON.stringify(response, null, 2);
}
async function runCommand(label, action) {
  const id = selectedID();
  if (!id) return;
  planCommandButton.disabled = true;
  executeCommandButton.disabled = true;
  commandState.textContent = "実行中…";
  try {
    displayCommand(label, await action(id));
  } finally {
    planCommandButton.disabled = !selectedID();
    executeCommandButton.disabled = !selectedID();
  }
}
function renderView(value) {
  viewOutput.replaceChildren();
  for (const section of summarizeView(value)) {
    text(viewOutput, "h3", section.title);
    if (Array.isArray(section.header) && section.header.length) {
      const table = document.createElement("table");
      const head = document.createElement("tr");
      for (const label of section.header) text(head, "th", label);
      table.appendChild(head);
      for (const row of section.rows || []) {
        const tr = document.createElement("tr");
        for (const cell of row) text(tr, "td", cell);
        table.appendChild(tr);
      }
      viewOutput.appendChild(table);
    } else {
      for (const row of section.rows || []) text(viewOutput, "p", row.join("："));
    }
  }
}
function displayView(label, response) {
  viewState.textContent = response.ok ? label + "：ビュー応答" : label + "：" + response.failure.code;
  if (!response.ok) {
    viewOutput.textContent = JSON.stringify(response, null, 2);
    return;
  }
  renderView(response.value);
}
async function runView(button, label, action) {
  const id = selectedID();
  if (!id) return;
  viewApplicationButton.disabled = true;
  viewPatternButton.disabled = true;
  viewPackageButton.disabled = true;
  viewState.textContent = "実行中…";
  try {
    displayView(label, await action(id));
  } finally {
    viewApplicationButton.disabled = !selectedID();
    viewPatternButton.disabled = !selectedID();
  viewPackageButton.disabled = !selectedID();
    viewPackageButton.disabled = !selectedID();
  }
}
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
  viewApplicationButton.disabled = !selectedID();
  viewPatternButton.disabled = !selectedID();
  viewPackageButton.disabled = !selectedID();
  planCommandButton.disabled = !selectedID();
  executeCommandButton.disabled = !selectedID();
}
workspace.addEventListener("change", invalidatePreview);
xeadPath.addEventListener("input", invalidatePreview);
observeButton.addEventListener("click", async () => {
  const id = selectedID();
  const relativePath = xeadPath.value.trim();
  if (!id || !relativePath) return;
  clearPreview();
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
viewApplicationButton.addEventListener("click", () => runView(viewApplicationButton, "application", (id) => api.viewApplication(id, applicationInput.value.trim())));
planCommandButton.addEventListener("click", () => runCommand("plan", (id) => api.planCommand(id, applicationInput.value.trim(), patternInput.value.trim(), capabilityInput.value.trim(), parseInputs(commandInputs.value))));
executeCommandButton.addEventListener("click", () => runCommand("execute", (id) => api.executeCommandWithPeriods(id, applicationInput.value.trim(), patternInput.value.trim(), capabilityInput.value.trim(), parseInputs(commandInputs.value), parseInputs(commandPeriods.value))));
viewPatternButton.addEventListener("click", () => runView(viewPatternButton, "pattern", (id) => api.viewPattern(id, applicationInput.value.trim(), patternInput.value.trim())));
viewPackageButton.addEventListener("click", () => runView(viewPackageButton, "package", (id) => api.viewPackage(id, applicationInput.value.trim(), packageInput.value.trim())));
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
  viewApplicationButton.disabled = !selectedID();
  viewPatternButton.disabled = !selectedID();
  viewPackageButton.disabled = !selectedID();
  planCommandButton.disabled = !selectedID();
  executeCommandButton.disabled = !selectedID();
}





