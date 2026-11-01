// Read-only Consumer binding for a successfully displayed graph/list snapshot.
// Model identities are qualified and never inferred from a view ID or label.
// This is not Source Acceptance, a permission check, or an authoritative cache.

const exact = value => typeof value === "string" && value.length > 0 &&
  value.trim() === value && !/[\r\n\t]/.test(value);

function requireSource(value) {
  if (!value || typeof value !== "object" || !exact(value.workspaceId) ||
      !exact(value.modelRevision)) {
    throw new TypeError("invalid_source_snapshot");
  }
  return {workspaceId: value.workspaceId, modelRevision: value.modelRevision};
}

function sameSource(left, right) {
  return left?.workspaceId === right?.workspaceId &&
    left?.modelRevision === right?.modelRevision;
}

function qualifiedKey(value) {
  if (!value || typeof value !== "object" ||
      !exact(value.scope) || !exact(value.kind) || !exact(value.id) ||
      (value.version !== undefined && !exact(value.version))) {
    throw new TypeError("invalid_qualified_key");
  }
  return Object.freeze({
    scope: value.scope,
    subject: Object.freeze({kind: value.kind, id: value.id}),
    ...(value.version === undefined ? {} : {version: value.version}),
  });
}

function buildBindings(input) {
  if (!input || typeof input !== "object" || !Array.isArray(input.bindings)) {
    throw new TypeError("invalid_selection_snapshot");
  }
  const source = requireSource(input);
  const bindings = new Map();
  for (const entry of input.bindings) {
    if (!entry || typeof entry !== "object" || !exact(entry.viewId)) {
      throw new TypeError("invalid_view_id");
    }
    if (bindings.has(entry.viewId)) {
      throw new TypeError("duplicate_view_id");
    }
    bindings.set(entry.viewId, qualifiedKey(entry));
  }
  return {source, bindings};
}

// A render callback must use an atomic Graph replacement: either it succeeds
// or it leaves the old visible diagram unchanged. Never publish a pending map
// merely because a render attempt was requested.
export function createQualifiedSelectionGuard() {
  let source = null;
  let generation = 0;
  let active = null;
  let disposed = false;
  const pending = new WeakMap();

  function live() {
    if (disposed) throw new Error("selection_guard_disposed");
  }

  return Object.freeze({
    observeSource(next) {
      live();
      if (next === null) {
        source = null;
        active = null;
        generation++;
        return;
      }
      const observed = requireSource(next);
      if (!sameSource(source, observed)) {
        source = observed;
        active = null;
        generation++;
      }
    },

    stage(input) {
      live();
      const snapshot = buildBindings(input);
      if (!source || !sameSource(source, snapshot.source)) {
        throw new Error("stale_selection_snapshot");
      }
      // Opaque token; bindings cannot be forged or changed after staging.
      const token = Object.freeze({});
      pending.set(token, {snapshot, generation});
      return token;
    },

    publish(token, render) {
      live();
      const candidate = token && pending.get(token);
      if (!candidate) throw new Error("unknown_selection_candidate");
      pending.delete(token);
      if (typeof render !== "function") throw new TypeError("invalid_render");
      if (candidate.generation !== generation ||
          !sameSource(source, candidate.snapshot.source)) {
        return {status: "stale"};
      }
      // Let the renderer throw. The previous active binding is not replaced.
      render();
      // Reentrant source changes invalidate this candidate even if rendering
      // completed. The resulting screen is stale until the next redraw.
      if (candidate.generation !== generation ||
          !sameSource(source, candidate.snapshot.source)) {
        active = null;
        return {status: "stale"};
      }
      active = candidate.snapshot;
      return {status: "published"};
    },

    select(viewId) {
      live();
      if (!active || !source || !sameSource(source, active.source)) {
        return {status: "stale"};
      }
      const ref = active.bindings.get(viewId);
      return ref ? {status: "resolved", ref} : {status: "unbound"};
    },

    invalidate() {
      live();
      source = null;
      active = null;
      generation++;
    },

    dispose() {
      disposed = true;
      source = null;
      active = null;
      generation++;
    },
  });
}
