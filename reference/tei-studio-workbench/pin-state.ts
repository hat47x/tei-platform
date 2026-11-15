/** Local Workbench intent only: not a Source revision, Grant or execution capability. */
export type PinnedSubject = { kind: string; id: string };
export type PinIntent = {
  workspaceId: string;
  workspaceEpoch: number;
  subject: PinnedSubject;
  generation: number;
};
export type PinObservation =
  | { state: "unqualified" }
  | { state: "inspected" }
  | { state: "needs-requalification" }
  | { state: "unavailable" };
export type PinState = {
  generation: number;
  intent: PinIntent | null;
  observation: PinObservation;
};
export type PinRequest = {
  generation: number;
  workspaceId: string;
  workspaceEpoch: number;
  subject: PinnedSubject;
};

export function initialPinState(): PinState {
  return { generation: 0, intent: null, observation: { state: "unqualified" } };
}

export function pinSubject(
  previous: PinState, workspaceId: string, workspaceEpoch: number, subject: PinnedSubject,
): PinState {
  const generation = previous.generation + 1;
  return {
    generation,
    intent: {
      workspaceId, workspaceEpoch,
      subject: { kind: subject.kind, id: subject.id },
      generation,
    },
    observation: { state: "unqualified" },
  };
}

export function unpinSubject(previous: PinState): PinState {
  return {
    generation: previous.generation + 1,
    intent: null,
    observation: { state: "unqualified" },
  };
}

/** Undo restores only the target intent, never an earlier Reader observation. */
export function restorePinIntent(
  previous: PinState, old: PinIntent | null,
): PinState {
  if (!old) return unpinSubject(previous);
  return pinSubject(previous, old.workspaceId, old.workspaceEpoch, old.subject);
}

export function invalidatePin(
  previous: PinState, state: "needs-requalification" | "unavailable" = "needs-requalification",
): PinState {
  return {
    generation: previous.generation + 1,
    intent: previous.intent
      ? { ...previous.intent, generation: previous.generation + 1 }
      : null,
    observation: previous.intent ? { state } : { state: "unqualified" },
  };
}

export function pinRequest(state: PinState): PinRequest | null {
  const intent = state.intent;
  if (!intent) return null;
  return {
    generation: state.generation,
    workspaceId: intent.workspaceId,
    workspaceEpoch: intent.workspaceEpoch,
    subject: { ...intent.subject },
  };
}

/** A successful reply may qualify only the exact intent/request generation.
 * Caller must additionally validate the Product Reader's response and Source revision.
 */
export function acceptPinObservation(
  previous: PinState,
  request: PinRequest,
  observation: PinObservation,
): PinState {
  const intent = previous.intent;
  if (!intent ||
    request.generation !== previous.generation ||
    request.generation !== intent.generation ||
    request.workspaceId !== intent.workspaceId ||
    request.workspaceEpoch !== intent.workspaceEpoch ||
    request.subject.kind !== intent.subject.kind ||
    request.subject.id !== intent.subject.id) return previous;
  return { ...previous, observation };
}
