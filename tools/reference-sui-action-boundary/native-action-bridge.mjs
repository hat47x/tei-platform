// Browser-compatible reference adapter; no framework or process dependency.
// A native UI owns pointer capture, rendering and local preview. Only drop
// hands the final application-defined payload to the TEI action boundary.
export function createNativeActionBridge(dispatchCommit) {
  if (typeof dispatchCommit !== "function") {
    throw new TypeError("dispatchCommit must be a function");
  }

  let interaction = null;
  const requireActive = () => {
    if (!interaction || interaction.phase !== "preview") {
      throw new Error("no active interaction");
    }
    return interaction;
  };

  return {
    begin({ applicationID, documentID, actionID, expectedRevision }) {
      if (interaction) {
        throw new Error("interaction already active or committing");
      }
      if ([applicationID, documentID, actionID, expectedRevision]
          .some((value) => typeof value !== "string" || value.length === 0)) {
        throw new TypeError("action identity and expected revision are required");
      }
      interaction = {
        phase: "preview",
        identity: { applicationID, documentID, actionID, expectedRevision },
        hasPayload: false,
        payload: undefined,
      };
    },

    preview(payload) {
      const active = requireActive();
      // Prevent subsequent mutations of UI-owned state from changing the
      // payload that will be committed on drop.
      active.payload = structuredClone(payload);
      active.hasPayload = true;
      return structuredClone(active.payload);
    },

    cancel() {
      if (!interaction || interaction.phase !== "preview") {
        return false;
      }
      interaction = null;
      return true;
    },

    async drop() {
      const active = requireActive();
      if (!active.hasPayload) {
        throw new Error("drop requires a final preview payload");
      }
      const intent = {
        ...active.identity,
        payload: structuredClone(active.payload),
      };
      interaction = { phase: "pending" };
      try {
        return await dispatchCommit(intent);
      } finally {
        // A rejected/denied/conflicting operation must never be retried
        // implicitly with its now-stale expected revision.
        interaction = null;
      }
    },

    get phase() {
      return interaction?.phase ?? "idle";
    },
  };
}
