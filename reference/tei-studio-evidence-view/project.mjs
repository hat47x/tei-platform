/** EV-R0: synthetic, display-only evidence lanes; never an authority or policy engine. */
export const EV_R0_FIXTURE_SCHEMA = 'tei.reference.studio-evidence-r0-fixture/v0';

const unavailable = () => ({
  evidenceNature: 'synthetic-display-only',
  publication: 'unavailable',
  message: '現在の確認結果を表示できません',
  lanes: null,
});

const isRecord = value => value !== null && typeof value === 'object' &&
  !Array.isArray(value);
const invalid = () => { throw new Error('INVALID_EV_R0_PUBLIC_SLICE'); };
function exactKeys(value, keys) {
  if (!isRecord(value) || Object.keys(value).some(key => !keys.includes(key)) ||
      keys.some(key => !Object.hasOwn(value, key))) invalid();
}
function word(value) {
  if (typeof value !== 'string' || value.length === 0 || value.length > 128) invalid();
  return value;
}
function oneOf(value, options) {
  if (!options.includes(value)) invalid();
  return value;
}
function subject(value) {
  if (value === null) return null;
  exactKeys(value, ['scope', 'kind', 'id']);
  return {scope: word(value.scope), kind: word(value.kind), id: word(value.id)};
}
function optionalWord(value) { return value == null ? null : word(value); }

/**
 * The caller MUST pass only a previously disclosed public slice. `release.state`
 * is a synthetic fixture marker, not authentication, authorization or a Grant.
 * No raw source, secret data, owner decision or policy evaluation is performed.
 */
export function projectEvidenceView(frame) {
  if (!isRecord(frame) || frame.schema !== EV_R0_FIXTURE_SCHEMA) invalid();
  // Never inspect the payload on a non-publishable frame. The same public
  // fallback is used for unknown, denied, revoked and incomplete decisions.
  if (!isRecord(frame.release) || frame.release.state !== 'prepared-public-fixture') {
    return unavailable();
  }
  exactKeys(frame, ['schema', 'release', 'public']);
  exactKeys(frame.release, ['state', 'purpose']);
  const purpose = oneOf(frame.release.purpose, ['studio-readonly']);

  const p = frame.public;
  exactKeys(p, ['source', 'operation', 'transition', 'observations', 'decision']);
  exactKeys(p.source, ['modelRevision', 'worldFingerprint', 'sourceIndexFingerprint', 'locatorState']);
  const source = {
    modelRevision: word(p.source.modelRevision),
    worldFingerprint: word(p.source.worldFingerprint),
    sourceIndexFingerprint: word(p.source.sourceIndexFingerprint),
    locatorState: oneOf(p.source.locatorState, ['current', 'stale', 'unobserved']),
  };
  exactKeys(p.operation, ['selected', 'target', 'operationId', 'attemptId', 'selectionEpoch']);
  const selected = subject(p.operation.selected);
  if (selected === null) invalid();
  const target = subject(p.operation.target);
  const operationId = word(p.operation.operationId);
  const attemptId = word(p.operation.attemptId);
  if (!Number.isSafeInteger(p.operation.selectionEpoch) || p.operation.selectionEpoch < 0) invalid();
  exactKeys(p.transition, ['actionId', 'actionRevision']);
  const transition = {actionId: word(p.transition.actionId),
    actionRevision: word(p.transition.actionRevision)};

  exactKeys(p.observations, ['command', 'effect', 'delivery']);
  const {command, effect, delivery} = p.observations;
  exactKeys(command, ['executorMode', 'disposition']);
  const executorMode = oneOf(command.executorMode, ['reference', 'runner', 'unobserved']);
  const disposition = oneOf(command.disposition,
    ['committed', 'replayed', 'denied', 'conflict', 'failed', 'unobserved']);
  exactKeys(effect, ['state', 'operationId', 'attemptId']);
  const effectState = oneOf(effect.state, ['unobserved', 'query-hit', 'query-empty', 'read-error']);
  const effectOperation = optionalWord(effect.operationId), effectAttempt = optionalWord(effect.attemptId);
  const effectLink = effectState === 'unobserved' || effectState === 'read-error'
    ? 'not-established'
    : effectOperation === operationId && effectAttempt === attemptId
      ? 'same-id-only-not-causality' : 'mismatch-or-unknown';
  exactKeys(delivery, ['state', 'response']);
  const deliveryState = oneOf(delivery.state, ['unobserved', 'sent', 'received', 'confirmed-not-sent']);
  const response = optionalWord(delivery.response);

  const d = p.decision;
  exactKeys(d, ['outcome', 'reviewBindingStatus', 'decisionActorStatus',
    'decisionAuthorityStatus', 'decisionWitnessStatus', 'witnessDiagnosticCodes']);
  const outcome = oneOf(d.outcome, ['accepted', 'rejected', 'unobserved']);
  const reviewBindingStatus = oneOf(d.reviewBindingStatus,
    ['none', 'recorded', 'verified', 'invalid', 'unobserved']);
  const decisionActorStatus = word(d.decisionActorStatus);
  const decisionAuthorityStatus = word(d.decisionAuthorityStatus);
  const decisionWitnessStatus = oneOf(d.decisionWitnessStatus,
    ['none', 'recorded', 'verified', 'invalid', 'conflicted', 'unobserved']);
  if (!Array.isArray(d.witnessDiagnosticCodes) ||
      !d.witnessDiagnosticCodes.every(x => typeof x === 'string' && /^[A-Z][A-Z0-9_]*$/.test(x))) invalid();

  return {
    evidenceNature: 'synthetic-display-only',
    publication: 'prepared-public-fixture-not-an-authorization',
    purpose,
    lanes: {
      operation: {selected, targetCandidate: target, operationId, attemptId,
        selectionEpoch: p.operation.selectionEpoch, source,
        targetQualification: 'not-established-by-presenter'},
      declaredTransition: {...transition, qualification: 'declaration-only'},
      observedEffect: {
        command: {executorMode, disposition,
          qualification: 'reported-result-only-not-durable-effect'},
        effect: {state: effectState, correlation: effectLink,
          qualification: 'no-independent-causality-proof'},
        delivery: {state: deliveryState, response,
          qualification: 'recipient-status-independent-of-command'},
      },
      ownerAcceptedMeaning: {status: 'unknown', sourceDecisionOutcome: outcome,
        reviewBindingStatus, decisionActorStatus, decisionAuthorityStatus,
        decisionWitnessStatus, witnessDiagnosticCodes: [...d.witnessDiagnosticCodes],
        qualification: 'no-owner-acceptance-in-this-reference'},
    },
    limitations: ['no-current-policy-or-grant-check', 'no-owner-meaning-acceptance',
      'no-provider-effect-or-receipt-verification', 'no-actual-recipient-readback'],
  };
}
