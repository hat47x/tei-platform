import {Call} from "/wails/runtime.js";
import {createReadOnlyClient} from "./client.mjs";

const api = createReadOnlyClient((name, ...args) => Call.ByName(name, ...args));
const workspace = document.querySelector("#workspace");
const state = document.querySelector("#state");
const output = document.querySelector("#output");
const actions = ["catalog", "validate", "inspect"].map((id) => document.getElementById(id));

function display(label, response) {
  state.textContent = response.ok ? label + "：Operation応答（診断の有無は結果を参照）"
    : label + "：" + response.failure.code;
  output.textContent = JSON.stringify(response, null, 2);
}
function selectedID() { return workspace.value; }
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
}
