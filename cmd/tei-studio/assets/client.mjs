// Host-neutral JS port for the PoC A bootstrap screen.
// #3 can replace this screen without importing Go/Wails into reusable UI.
const names = Object.freeze({
  list: "main.StudioBridge.ListWorkspaces",
  catalog: "main.StudioBridge.CatalogWorkspace",
  validate: "main.StudioBridge.ValidateWorkspace",
  inspect: "main.StudioBridge.InspectSubject",
  observeXead: "main.StudioBridge.ObserveXead",
});

export function createReadOnlyClient(callByName) {
  if (typeof callByName !== "function") throw new TypeError("callByName must be a function");

  async function call(method, ...args) {
    try {
      const result = await callByName(names[method], ...args);
      if (result === null || typeof result !== "object" || typeof result.ok !== "boolean" ||
          (result.ok && !Object.hasOwn(result, "value")) ||
          (!result.ok && (!result.failure || typeof result.failure.code !== "string"))) {
        return {ok: false, failure: {code: "INVALID_HOST_RESPONSE", message: "Host returned an invalid envelope"}};
      }
      return result;
    } catch {
      // Never display arbitrary exception text from runtime/Host to the UI.
      return {ok: false, failure: {code: "HOST_TRANSPORT_FAILURE", message: "Wails service is unavailable"}};
    }
  }
  return Object.freeze({
    listWorkspaces: () => call("list"),
    catalogWorkspace: (workspaceID, namespace = "") => call("catalog", workspaceID, namespace),
    validateWorkspace: (workspaceID) => call("validate", workspaceID),
    inspectSubject: (workspaceID, kind, subjectID) => call("inspect", workspaceID, kind, subjectID),
    observeXead: (workspaceID, relativePath) => call("observeXead", workspaceID, relativePath),
  });
}
