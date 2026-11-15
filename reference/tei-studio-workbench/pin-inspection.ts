/**
 * Read-only Workbench Pin consumer. It owns no Source, Grant, external
 * disclosure or editor selection. An "inspected" result means only that the
 * current Host Inspect reply matches the Catalog snapshot used by this view.
 */
import { inspectEligibility, readInspection } from "./presentation.ts";
import type { Catalog, Subject } from "./presentation.ts";
import {
  initialPinState, pinSubject, unpinSubject, invalidatePin,
  pinRequest, acceptPinObservation,
} from "./pin-state.ts";
import type { PinState, PinObservation } from "./pin-state.ts";
import { createResponseScope } from "./responses.ts";

type Reply =
  | { ok: true; value: unknown }
  | { ok: false; failure: { code: string } };
type Inspect = (workspace: string, kind: string, subject: string) => Promise<Reply>;
type Scope = ReturnType<typeof createResponseScope>;

export function createPinInspection(
  scope: Scope,
  inspect: Inspect,
  currentCatalog: () => Catalog | null,
  report: (state: PinState) => void,
) {
  let state = initialPinState();
  const emit = (next: PinState) => { state = next; report(state); };
  const current = () => state;

  /** A Pin is local intent only, not a privileged object reference. */
  async function pin(workspaceId: string, epoch: number, subject: Subject): Promise<boolean> {
    scope.invalidate("pinned-inspection");
    emit(pinSubject(state, workspaceId, epoch, subject));
    return recheck(workspaceId, epoch);
  }

  /** Explicit recheck; does not touch the ordinary inspection/editor lane. */
  async function recheck(workspaceId: string, epoch: number): Promise<boolean> {
    const old = state.intent;
    if (!old) return false;
    scope.invalidate("pinned-inspection");
    if (old.workspaceId !== workspaceId) {
      emit(invalidatePin(state));
      return false;
    }
    // After a Workspace refresh (or A→B→A), only a user-requested recheck
    // may rebind the local intent to the current UI epoch.
    emit(old.workspaceEpoch !== epoch
      ? pinSubject(state, workspaceId, epoch, old.subject)
      : invalidatePin(state));

    const req = pinRequest(state);
    const snapshot = currentCatalog();
    if (!req || !snapshot || !inspectEligibility(snapshot, req.subject).ok) {
      emit(invalidatePin(state));
      return false;
    }
    await scope.run("pinned-inspection",
      () => inspect(req.workspaceId, req.subject.kind, req.subject.id),
      (reply) => {
        if (!reply.ok) throw Error("PIN_INSPECT_FAILED");
        const latest = currentCatalog();
        if (!latest || latest.modelRevision !== snapshot.modelRevision ||
            latest.worldFingerprint !== snapshot.worldFingerprint ||
            latest.sourceIndexFingerprint !== snapshot.sourceIndexFingerprint)
          throw Error("PIN_CATALOG_CHANGED");
        const view = readInspection(reply.value, req.subject, snapshot);
        const observation: PinObservation = view.sourceStatus === "indexed"
          ? { state: "inspected" }
          : { state: "needs-requalification" };
        emit(acceptPinObservation(state, req, observation));
      },
      () => {
        emit(acceptPinObservation(state, req, { state: "unavailable" }));
      });
    return state.observation.state === "inspected" &&
      state.intent?.generation === req.generation;
  }

  function release(): void {
    scope.invalidate("pinned-inspection");
    emit(unpinSubject(state));
  }

  /** A new Catalog or Workspace invalidates old observations, not intent. */
  function invalidate(): void {
    scope.invalidate("pinned-inspection");
    if (state.intent) emit(invalidatePin(state));
  }

  return { current, pin, recheck, release, invalidate };
}
