import type { Subject } from "./presentation.ts";

/** Public, read-only UI intent. Never a Product Inspect result or a Host Grant. */
export const INSPECTOR_SELECTION_EVENT = "tei:inspector-selection-request";

export type InspectorSelectionRequest = Readonly<{
  workspaceId: string;
  workspaceEpoch: number;
  subject: Readonly<Subject>;
}>;

function normalized(value: unknown): InspectorSelectionRequest | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const request = value as Record<string, unknown>;
  const subject = request.subject;
  if (!subject || typeof subject !== "object" || Array.isArray(subject)) return null;
  const entry = subject as Record<string, unknown>;
  if (typeof request.workspaceId !== "string" || !request.workspaceId ||
      !Number.isSafeInteger(request.workspaceEpoch) ||
      (request.workspaceEpoch as number) < 0 ||
      typeof entry.kind !== "string" || !entry.kind ||
      typeof entry.id !== "string" || !entry.id) return null;
  // Explicitly copy only routing fields; ignore untrusted Effect/Grant claims.
  return {
    workspaceId: request.workspaceId,
    workspaceEpoch: request.workspaceEpoch as number,
    subject: { kind: entry.kind, id: entry.id },
  };
}

export function emitInspectorSelection(target: EventTarget, request: InspectorSelectionRequest): void {
  const detail = normalized(request);
  if (!detail) throw Error("INVALID_INSPECTOR_SELECTION_REQUEST");
  target.dispatchEvent(new CustomEvent(INSPECTOR_SELECTION_EVENT, {
    detail, bubbles: true, composed: true, cancelable: false,
  }));
}

/** A fresh UI request only; caller must still check Catalog eligibility and Reader. */
export function inspectorSelectionFromEvent(
  event: Event, currentWorkspaceId: string, currentWorkspaceEpoch: number,
): Subject | null {
  if (event.type !== INSPECTOR_SELECTION_EVENT || !(event instanceof CustomEvent))
    return null;
  const request = normalized(event.detail);
  if (!request || request.workspaceId !== currentWorkspaceId ||
      request.workspaceEpoch !== currentWorkspaceEpoch) return null;
  return request.subject;
}
