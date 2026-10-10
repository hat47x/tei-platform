// Optional browser-side confirmation boundary for resource-scoped Action commits.
// A successful POST alone must not change local canonical/application state.
export class ActionConfirmationError extends Error {
  constructor(code, committedRevision = null) {
    super(code);
    this.name = 'ActionConfirmationError';
    this.code = code;
    // Non-null only when the server already acknowledged a successful commit.
    // A reconciliation failure is not a rollback and must not trigger a retry.
    this.committedRevision = committedRevision;
  }
}

export function createConfirmedActionCommit({ dispatchCommit, readResource, applySnapshot, isCurrent }) {
  if ([dispatchCommit, readResource, applySnapshot, isCurrent].some((fn) => typeof fn !== 'function')) {
    throw new TypeError('dispatchCommit, readResource, applySnapshot and isCurrent are required');
  }
  const pending = new Set();
  // A Promise is truthy; allowing an async guard would bypass the stale-state
  // check. Require a synchronous, explicit true at both trust boundaries.
  const assertCurrent = (intent, committedRevision = null) => {
    let current = false;
    try { current = isCurrent(intent) === true; } catch { /* fail closed */ }
    if (!current) throw new ActionConfirmationError('local_state_changed', committedRevision);
  };

  return async (intent) => {
    if (!intent || typeof intent !== 'object' ||
        [intent.applicationID, intent.resourceID, intent.actionID, intent.expectedRevision]
          .some((value) => typeof value !== 'string' || value.length === 0)) {
      throw new TypeError('resource-scoped Action identity and revision are required');
    }
    // Do not use slash-separated identity segments; slashes are legal in IDs.
    const key = JSON.stringify([intent.applicationID, intent.resourceID]);
    if (pending.has(key)) {
      throw new ActionConfirmationError('action_in_flight');
    }
    assertCurrent(intent);
    pending.add(key);
    let committedRevision = null;
    try {
      // The server must own authorization, validation and atomic revision CAS.
      // Rejecting here must not invoke readResource or applySnapshot.
      committedRevision = await dispatchCommit(intent);
      if (typeof committedRevision !== 'string' || committedRevision.length === 0) {
        throw new ActionConfirmationError('invalid_commit_result');
      }
      let resource;
      try {
        // readResource must bypass caches and read the authoritative Store.
        resource = await readResource(intent);
      } catch {
        throw new ActionConfirmationError('resource_read_failed', committedRevision);
      }
      if (!resource || typeof resource !== 'object' || Array.isArray(resource) ||
          resource.applicationID !== intent.applicationID ||
          resource.resourceID !== intent.resourceID ||
          typeof resource.revision !== 'string' || resource.revision.length === 0 ||
          !Object.hasOwn(resource, 'value')) {
        throw new ActionConfirmationError('invalid_resource_snapshot', committedRevision);
      }
      if (resource.revision !== committedRevision) {
        // Concurrent remote changes may be legitimate, but this readback no
        // longer proves the exact acknowledged revision. Reconcile explicitly.
        throw new ActionConfirmationError('revision_changed_after_commit', committedRevision);
      }
      // Another local edit may have happened during the remote round-trip.
      assertCurrent(intent, committedRevision);
      try {
        // The application owns state and Undo semantics. This must be an
        // atomic guarded apply: never replace edits made after isCurrent.
        if (await applySnapshot(resource, intent) === false) {
          throw new ActionConfirmationError('snapshot_apply_failed', committedRevision);
        }
      } catch {
        throw new ActionConfirmationError('snapshot_apply_failed', committedRevision);
      }
      return committedRevision;
    } finally {
      pending.delete(key);
    }
  };
}
