// Coordinates belong to a disposable DFD view, never to Business Procedure Source.
// The producer snapshot is caller-supplied: only the surrounding workspace can
// verify its provenance, freshness and Source Acceptance authority.
export const LAYOUT_SCHEMA = 'tei.reference.business-dfd-view-layout/v0';
export const EDIT_SCHEMA = 'tei.reference.business-dfd-layout-edit/v0';
export const PREVIEW_SCHEMA = 'tei.reference.business-dfd-layout-preview/v0';
const SHA = /^sha256:[0-9a-f]{64}$/;
const MAX_POSITION = 100000;

function requireID(id, subject) {
  if (typeof id !== 'string' || id.length === 0 || id !== id.trim()) {
    throw new Error('INVALID_ID: ' + subject);
  }
  return id;
}
function object(value, allowed, name) {
  if (!value || typeof value !== 'object' || Array.isArray(value) ||
      Object.keys(value).some(key => !allowed.includes(key))) {
    throw new Error('INVALID_FIELDS: ' + name);
  }
}
function requireDigest(hash, label) {
  if (!SHA.test(hash)) throw new Error('INVALID_DIGEST: ' + label);
  return hash;
}
function nodeKey(kind, id) {
  return JSON.stringify([kind, id]);
}
function nodesInProjection(dfd) {
  if (!dfd || dfd.schemaVersion !== 'business-dfd-projection-v0' ||
      dfd.authority !== 'derived-read-only' || dfd.canonicalProcedureUsed !== false ||
      !Array.isArray(dfd.steps) || !Array.isArray(dfd.triggers) ||
      !Array.isArray(dfd.diagnostics) || typeof dfd.procedureId !== 'string' ||
      dfd.procedureId.length === 0 ||
      dfd.diagnostics.some(d => d.severity === 'error')) {
    throw new Error('UNQUALIFIED_DFD_PROJECTION');
  }
  const known = new Set();
  function collect(kind, entries) {
    if (!Array.isArray(entries)) throw new Error('INVALID_PROJECTION_ELEMENTS: ' + kind);
    for (const entry of entries) {
      const key = nodeKey(kind, requireID(entry.id, kind));
      if (known.has(key)) throw new Error('DUPLICATE_DFD_NODE: ' + key);
      known.add(key);
    }
  }
  collect('step', dfd.steps);
  collect('trigger', dfd.triggers);
  collect('business-role', dfd.information?.roles ?? []);
  collect('party', dfd.information?.externalParties ?? []);
  collect('store', dfd.information?.stores ?? []);
  const functions = new Set();
  for (const step of dfd.steps) {
    if (!Array.isArray(step.functionRefs)) throw new Error('INVALID_STEP_FUNCTION_REFS');
    for (const ref of step.functionRefs) {
      functions.add(requireID(ref.id, 'function'));
    }
  }
  for (const id of functions) known.add(nodeKey('function', id));
  return known;
}
function coordinate(value, name) {
  if (typeof value !== 'number' || !Number.isFinite(value) ||
      Math.abs(value) > MAX_POSITION) throw new Error('INVALID_COORDINATE: ' + name);
  return value;
}
export function previewLayoutMove(dfd, projectionDigest, layout, currentLayoutDigest, edit) {
  requireDigest(projectionDigest, 'projection');
  requireDigest(currentLayoutDigest, 'layout');
  const known = nodesInProjection(dfd);
  object(layout, ['schemaVersion', 'authority', 'procedureId', 'projectionSHA256', 'nodes'], 'layout');
  if (layout.schemaVersion !== LAYOUT_SCHEMA || layout.authority !== 'view-only' ||
      layout.procedureId !== dfd.procedureId ||
      layout.projectionSHA256 !== projectionDigest || !Array.isArray(layout.nodes)) {
    throw new Error('STALE_OR_UNSUPPORTED_VIEW_LAYOUT');
  }
  const positions = new Map();
  for (const entry of layout.nodes) {
    object(entry, ['kind', 'id', 'x', 'y'], 'position');
    const key = nodeKey(requireID(entry.kind, 'position kind'), requireID(entry.id, 'position id'));
    if (!known.has(key)) throw new Error('UNKNOWN_LAYOUT_NODE: ' + key);
    if (positions.has(key)) throw new Error('DUPLICATE_LAYOUT_NODE: ' + key);
    coordinate(entry.x, 'x'); coordinate(entry.y, 'y');
    positions.set(key, entry);
  }
  object(edit, ['schema', 'id', 'origin', 'intent', 'procedureId', 'projectionSHA256',
    'baseLayoutSHA256', 'operation'], 'edit');
  if (edit.schema !== EDIT_SCHEMA || !['studio', 'ai', 'cli'].includes(edit.origin) ||
      edit.procedureId !== dfd.procedureId ||
      requireDigest(edit.projectionSHA256, 'edit projection') !== projectionDigest ||
      requireDigest(edit.baseLayoutSHA256, 'edit layout') !== currentLayoutDigest) {
    throw new Error('STALE_OR_UNSUPPORTED_LAYOUT_EDIT');
  }
  requireID(edit.id, 'edit id');
  requireID(edit.intent, 'edit intent');
  object(edit.operation, ['kind', 'nodeKind', 'nodeId', 'x', 'y'], 'operation');
  if (edit.operation.kind !== 'move-node') throw new Error('UNSUPPORTED_LAYOUT_OPERATION');
  const {nodeKind, nodeId} = edit.operation;
  const key = nodeKey(requireID(nodeKind, 'node kind'), requireID(nodeId, 'node id'));
  if (!known.has(key)) throw new Error('UNKNOWN_DFD_NODE: ' + key);
  const x = coordinate(edit.operation.x, 'x');
  const y = coordinate(edit.operation.y, 'y');
  const previous = positions.get(key);
  if (previous && previous.x === x && previous.y === y) throw new Error('NO_EFFECT_MOVE');
  const next = structuredClone(layout);
  const index = next.nodes.findIndex(row => nodeKey(row.kind, row.id) === key);
  const newPosition = {kind: nodeKind, id: nodeId, x, y};
  if (index < 0) next.nodes.push(newPosition);
  else next.nodes[index] = newPosition;
  next.nodes.sort((a, b) => nodeKey(a.kind, a.id).localeCompare(nodeKey(b.kind, b.id), 'en'));
  return {
    schema: PREVIEW_SCHEMA, authority: 'derived-view-only',
    procedureId: dfd.procedureId,
    projectionSHA256: projectionDigest,
    baseLayoutSHA256: currentLayoutDigest,
    projectionFreshness: 'caller-supplied-unverified',
    sourceWrites: [], readyForAcceptance: false,
    canonicalMeaningChanged: false,
    operation: {kind: 'move-node', nodeKind, nodeId, x, y},
    proposedLayout: next
  };
}
