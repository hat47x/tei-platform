/**
 * Synthetic evidence for the Organization View -> Intent -> Request boundary.
 * This is a read-only handoff oracle, not a TEI Action, Host or Tx implementation.
 * Run with: node --test reference/org-semantic-views/verify-handoff.mjs
 */
// Reuse the existing synthetic-node CI entrypoint for commercial negative controls.
import './verify-commercial-probes.mjs';
import { scenario } from './source-fixture.mjs';
import test from 'node:test';
import assert from 'node:assert/strict';

const read = Object.freeze({ ...scenario.readEvidence });
const intent = Object.freeze({ action: 'unlink', targetKind: 'Membership', targetId: 'A3' });

// Returns a *read-only* suitability claim; it does not authorize or construct a business Request.
function assess(readEvidence, selection, candidate, targetRequest = undefined) {
  for (const k of ['instance', 'anchor', 'epoch', 'meaning', 'query', 'readVersion', 'occurrence']) {
    if (readEvidence[k] !== selection[k]) return { verdict: 'unresolved', reason: `changed-${k}` };
  }
  if (candidate.targetKind !== readEvidence.edgeKind || candidate.targetId !== readEvidence.edge) {
    return { verdict: 'unresolved', reason: 'wrong-relation-occurrence' };
  }
  if (candidate.action !== 'unlink') return { verdict: 'unsupported', reason: 'operation-meaning' };
  if (selection.requiresAll && readEvidence.complete !== true) {
    return { verdict: 'unresolved', reason: 'incomplete-required-scope' };
  }
  if (targetRequest && (targetRequest.targetKind !== candidate.targetKind ||
      targetRequest.targetId !== candidate.targetId || targetRequest.action !== candidate.action)) {
    return { verdict: 'unresolved', reason: 'request-target-or-operation-mismatch' };
  }
  return { verdict: 'candidate', relation: candidate.targetId };
}
const selection = Object.freeze({ ...read, requiresAll: false });

test('same evidence and exact edge target yields read-only candidate', () => {
  assert.deepEqual(assess(read, selection, intent), { verdict: 'candidate', relation: 'A3' });
});
test('a serialized intent alone does not preserve a changed selection epoch', () => {
  assert.deepEqual(assess(read, { ...selection, epoch: 8 }, intent),
    { verdict: 'unresolved', reason: 'changed-epoch' });
});
test('accepted meaning revision is not replaceable by the same displayed edge', () => {
  assert.deepEqual(assess(read, { ...selection, meaning: 'org-r8' }, intent),
    { verdict: 'unresolved', reason: 'changed-meaning' });
});
test('a changed selected parent cannot silently retarget the edge', () => {
  assert.deepEqual(assess(read, { ...selection, anchor: 'D1' }, intent),
    { verdict: 'unresolved', reason: 'changed-anchor' });
});
test('the same row appearance in a separate Function Instance is not the same session', () => {
  assert.deepEqual(assess(read, { ...selection, instance: 'I2' }, intent),
    { verdict: 'unresolved', reason: 'changed-instance' });
});
test('a new provider snapshot requires read-time qualification', () => {
  assert.deepEqual(assess(read, { ...selection, readVersion: 'snap-4' }, intent),
    { verdict: 'unresolved', reason: 'changed-readVersion' });
});
test('a different occurrence of the same Person does not identify A3', () => {
  assert.deepEqual(assess(read, selection, { ...intent, targetId: 'A2' }),
    { verdict: 'unresolved', reason: 'wrong-relation-occurrence' });
});
test('unlink of Membership A3 is not deletion of Person P2', () => {
  assert.deepEqual(assess(read, selection, { action: 'delete', targetKind: 'Person', targetId: 'P2' }),
    { verdict: 'unresolved', reason: 'wrong-relation-occurrence' });
});
test('an all-scope action may not use an incomplete page as a whole collection', () => {
  assert.deepEqual(assess({ ...read, complete: false }, { ...selection, requiresAll: true }, intent),
    { verdict: 'unresolved', reason: 'incomplete-required-scope' });
});
test('different request payload target fails even when intent is unchanged', () => {
  assert.deepEqual(assess(read, selection, intent,
    { action: 'unlink', targetKind: 'Membership', targetId: 'A2' }),
    { verdict: 'unresolved', reason: 'request-target-or-operation-mismatch' });
});
test('different request operation fails even for the same relation', () => {
  assert.deepEqual(assess(read, selection, intent,
    { action: 'delete', targetKind: 'Membership', targetId: 'A3' }),
    { verdict: 'unresolved', reason: 'request-target-or-operation-mismatch' });
});
test('a display-only department rename does not itself change the identity binding', () => {
  const renamedView = { ...selection, deptLabel: 'D2 new label' };
  assert.deepEqual(assess(read, renamedView, intent), { verdict: 'candidate', relation: 'A3' });
});
test('two distinct instance dirty patches do not share a storage identity', () => {
  const dirty1 = { instance: 'I1', anchor: 'D2', relation: 'A3', version: 'snap-3' };
  const dirty2 = { ...dirty1, instance: 'I2' };
  assert.notDeepEqual(dirty1, dirty2);
  assert.equal(dirty1.relation, dirty2.relation); // business target can be the same
});
test('a two-provider save is not atomic just because the view issued one Save Intent', () => {
  const save = { requests: ['remove-A2', 'remove-A3'], atomicityEvidence: null };
  assert.equal(save.requests.length, 2);
  assert.equal(save.atomicityEvidence, null);
});
test('one successful effect and one uncertain result do not prove business completion', () => {
  const results = [{ target: 'A2', effect: 'confirmed' }, { target: 'A3', effect: 'unknown' }];
  assert.equal(results.every(x => x.effect === 'confirmed'), false);
  assert.deepEqual(results.map(x => x.target), ['A2', 'A3']);
});
test('a confirmed request for A2 never serves as a receipt for intended A3', () => {
  const actualEffect = { operationId: 'op-77', target: 'A2', confirmed: true };
  const requested = { operationId: 'op-77', target: 'A3' };
  assert.notEqual(actualEffect.target, requested.target);
});
