// ISSUE #993: browser-portable reference contract, not a production TEI Runtime binding.
// The caller owns application authority, persistence, and domain validation.
export class BridgeError extends Error {
  constructor(code) { super(code); this.name = 'BridgeError'; this.code = code; }
}
function fail(code) { throw new BridgeError(code); }
function nonempty(value) { return typeof value === 'string' && value.trim() !== ''; }
function point(value) {
  if (!value || !Number.isFinite(value.x) || !Number.isFinite(value.y)) fail('invalid_position');
  return {x: value.x, y: value.y};
}
function scope(value) {
  if (!value || !nonempty(value.applicationId) || !nonempty(value.documentId) ||
      !nonempty(value.targetId) || !nonempty(value.targetKind) || !nonempty(value.expectedRevision)) fail('invalid_scope');
}

// Gesture state contains no Document, persistence handle, or callback.
export function beginGesture(input) {
  scope(input);
  return {phase: 'active', applicationId: input.applicationId, documentId: input.documentId,
    targetId: input.targetId, targetKind: input.targetKind, expectedRevision: input.expectedRevision,
    initial: point(input.initial), current: point(input.initial)};
}
export function previewGesture(session, position) {
  if (session?.phase !== 'active') fail('gesture_not_active');
  return {...session, current: point(position)};
}
export function cancelGesture(session) {
  if (session?.phase !== 'active') fail('gesture_not_active');
  return {...session, phase: 'cancelled'};
}
export function finishGesture(session) {
  if (session?.phase !== 'active') fail('gesture_not_active');
  return {applicationId: session.applicationId, documentId: session.documentId,
    expectedRevision: session.expectedRevision, operation: 'position.set',
    target: {kind: session.targetKind, id: session.targetId}, position: point(session.current)};
}

// Generic command boundary. Storage must perform an *atomic* conditional save;
// the preflight revision check alone does not prevent a race.
export async function commitIntent(intent, ports) {
  if (!intent || !nonempty(intent.applicationId) || !nonempty(intent.documentId) ||
      !nonempty(intent.expectedRevision) || !nonempty(intent.operation) ||
      !intent.target || !nonempty(intent.target.kind) || !nonempty(intent.target.id)) fail('invalid_intent');
  const position = point(intent.position);
  if (!ports || !nonempty(ports.applicationId) || !['authorize', 'load', 'apply', 'validate', 'save'].every(k => typeof ports[k] === 'function')) fail('missing_port');
  if (intent.applicationId !== ports.applicationId) fail('application_mismatch');
  // Do not load sensitive Document data before checking operation authority.
  if (await ports.authorize(intent) !== true) fail('unauthorized');
  const snapshot = await ports.load(intent.documentId);
  if (!snapshot || snapshot.applicationId !== intent.applicationId ||
      snapshot.documentId !== intent.documentId || !nonempty(snapshot.revision) ||
      !snapshot.document) fail('snapshot_mismatch');
  if (snapshot.revision !== intent.expectedRevision) fail('stale_revision');
  // Document is JSON (SUI DocumentV1); JSON cloning preserves domain extension keys.
  const copy = JSON.parse(JSON.stringify(snapshot.document));
  const updated = await ports.apply(copy, {...intent, position});
  if (updated == null || await ports.validate(updated, intent) !== true) fail('validation_failed');
  const receipt = await ports.save({applicationId: intent.applicationId,
    documentId: intent.documentId, expectedRevision: intent.expectedRevision, document: updated});
  if (!receipt || receipt.ok !== true || !nonempty(receipt.revision)) fail('conditional_save_failed');
  return {revision: receipt.revision};
}

// SUI-specific adapter kept *outside* generic gesture / command mechanism.
// It does not validate all DocumentV1 rules: an actual SUI validator must be supplied.
export function moveSuiCard(document, intent) {
  if (intent.operation !== 'position.set' || intent.target?.kind !== 'card' ||
      document?.version !== 1 || document.id !== intent.documentId ||
      !Array.isArray(document.cards)) fail('unsupported_sui_document');
  let matches = 0;
  for (const card of document.cards) {
    if (card.id === intent.target.id) {
      matches++;
      card.x = intent.position.x;
      card.y = intent.position.y;
    }
  }
  if (matches !== 1) fail('target_not_unique');
  return document;
}
