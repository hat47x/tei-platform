// JS port of the read-only three-view navigation. It names one Bridge method
// and checks the response envelope; it holds no model and no selection.
const method = "main.StudioBridge.NavigateThreeView";

export function createThreeViewClient(callByName) {
  if (typeof callByName !== "function") throw new TypeError("callByName must be a function");
  return Object.freeze({
    // pin is {occurrence, sourceArtifactDigest} or null. It is sent as given:
    // looking at another subject never changes it.
    async navigate(workspaceID, sourcePath, pin = null) {
      try {
        const result = await callByName(method, workspaceID, sourcePath,
          pin ? pin.occurrence : "", pin ? pin.sourceArtifactDigest : "");
        if (result === null || typeof result !== "object" || typeof result.ok !== "boolean" ||
            (result.ok && (result.value === null || typeof result.value !== "object")) ||
            (!result.ok && (!result.failure || typeof result.failure.code !== "string"))) {
          return {ok: false, failure: {code: "INVALID_HOST_RESPONSE", message: "Host returned an invalid envelope"}};
        }
        return result;
      } catch {
        // Never display arbitrary exception text from runtime/Host to the UI.
        return {ok: false, failure: {code: "HOST_TRANSPORT_FAILURE", message: "Wails service is unavailable"}};
      }
    },
  });
}

// The pure reading of one navigation value for one focused occurrence. Focus
// is what the user is looking at; it is separate from the pin and never
// written into it.
export function describeFocus(value, occurrence) {
  const subjects = new Map(value.subjects.map((subject) => [subject.occurrence, subject]));
  const subject = subjects.get(occurrence);
  if (!subject) return null;
  const label = (id) => {
    const other = subjects.get(id);
    if (!other) return id;
    return (other.sourceFact.name || "(名称なし)") + " [" + other.sourceFact.kind + "]";
  };
  return {
    subject,
    relations: value.relations
      .filter((relation) => relation.from === occurrence || relation.to === occurrence)
      .map((relation) => ({
        occurrence: relation.occurrence, kind: relation.kind, nature: relation.nature,
        operations: relation.operations || "",
        direction: relation.from === occurrence ? "out" : "in",
        other: relation.from === occurrence ? relation.to : relation.from,
        otherLabel: label(relation.from === occurrence ? relation.to : relation.from),
      })),
    crud: value.crud.filter((row) => row.function === occurrence || row.data === occurrence),
    fields: (value.fields || []).filter((field) => field.ownerOccurrence === occurrence),
    fieldReferences: (value.fieldReferences || []).filter((reference) => reference.owner === occurrence),
  };
}
