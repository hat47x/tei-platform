import {test} from 'node:test';
import assert from 'node:assert/strict';
import {EV_R0_FIXTURE_SCHEMA, projectEvidenceView} from './project.mjs';

// All material is synthetic/public. This is deliberately NOT a real Decision
// Index, Provider, Policy, Grant, Source, SHO or AI Consumer fixture.
function fixture() {
  return {
    schema: EV_R0_FIXTURE_SCHEMA,
    release: {state: 'prepared-public-fixture', purpose: 'studio-readonly'},
    public: {
      source: {modelRevision: 'R10', worldFingerprint: 'W10',
        sourceIndexFingerprint: 'I10', locatorState: 'current'},
      operation: {selected: {scope: 'sales', kind: 'line', id: 'L2'},
        target: {scope: 'sales', kind: 'order', id: 'H1'},
        operationId: 'op-1', attemptId: 'try-1', selectionEpoch: 7},
      transition: {actionId: 'edit-line', actionRevision: 'A1'},
      observations: {
        command: {executorMode: 'reference', disposition: 'committed'},
        effect: {state: 'unobserved', operationId: null, attemptId: null},
        delivery: {state: 'unobserved', response: null},
      },
      decision: {outcome: 'accepted', reviewBindingStatus: 'verified',
        decisionActorStatus: 'unresolved', decisionAuthorityStatus: 'unresolved',
        decisionWitnessStatus: 'verified', witnessDiagnosticCodes: []},
    },
  };
}

// EV-R0-01: no in-place mutation; unknown schema/secret-bearing fields rejected.
test('EV-R0-01 strict read-only public fixture projection', () => {
  const f = fixture(), before = structuredClone(f);
  const view = projectEvidenceView(f);
  assert.deepEqual(f, before);
  assert.equal(view.evidenceNature, 'synthetic-display-only');
  assert.equal(view.lanes.operation.targetCandidate.id, 'H1');
  assert.notEqual(view.lanes.operation.selected.id, view.lanes.operation.targetCandidate.id);
  assert.throws(() => projectEvidenceView({...f, schema: 'unknown'}), /INVALID_EV_R0_PUBLIC_SLICE/);
  assert.throws(() => projectEvidenceView({...f, privatePath: '/secret'}), /INVALID_EV_R0_PUBLIC_SLICE/);
});

// EV-R0-02: reference Committed=true is not an external DB commit.
test('EV-R0-02 reference result does not prove external effect', () => {
  const view = projectEvidenceView(fixture());
  assert.equal(view.lanes.observedEffect.command.disposition, 'committed');
  assert.equal(view.lanes.observedEffect.command.executorMode, 'reference');
  assert.equal(view.lanes.observedEffect.effect.state, 'unobserved');
  assert.match(view.lanes.observedEffect.command.qualification, /not-durable-effect/);
  assert.equal(view.lanes.ownerAcceptedMeaning.status, 'unknown');
});

// EV-R0-03: different operation/attempt cannot be silently joined; even exact
// string IDs alone do not establish effect causality.
test('EV-R0-03 independent effect read is not causal proof', () => {
  const f = fixture();
  f.public.observations.effect = {state: 'query-hit', operationId: 'other-op', attemptId: 'try-1'};
  assert.equal(projectEvidenceView(f).lanes.observedEffect.effect.correlation, 'mismatch-or-unknown');
  f.public.observations.effect.operationId = 'op-1';
  assert.equal(projectEvidenceView(f).lanes.observedEffect.effect.correlation, 'same-id-only-not-causality');
});

// EV-R0-04: valid-looking witness and separate invalid witness diagnostic may
// coexist. Neither is an Owner meaning acceptance.
test('EV-R0-04 witness verification and invalid diagnostic coexist', () => {
  const f = fixture();
  f.public.decision.decisionAuthorityStatus = 'authority-qualified';
  f.public.decision.witnessDiagnosticCodes = ['WITNESS_SIGNATURE_DIGEST_MISMATCH'];
  const owner = projectEvidenceView(f).lanes.ownerAcceptedMeaning;
  assert.equal(owner.decisionWitnessStatus, 'verified');
  assert.deepEqual(owner.witnessDiagnosticCodes, ['WITNESS_SIGNATURE_DIGEST_MISMATCH']);
  assert.equal(owner.status, 'unknown');
});

// EV-R0-05: Source identity and locator must not be collapsed.
test('EV-R0-05 unchanged WorldFP does not revive stale Source locator', () => {
  const f = fixture();
  f.public.source.sourceIndexFingerprint = 'I11';
  f.public.source.locatorState = 'stale';
  const op = projectEvidenceView(f).lanes.operation;
  assert.equal(op.source.worldFingerprint, 'W10');
  assert.equal(op.source.sourceIndexFingerprint, 'I11');
  assert.equal(op.source.locatorState, 'stale');
  assert.equal(op.targetQualification, 'not-established-by-presenter');
});

// EV-R0-06: failure/read-only reply cannot undo observed recipient delivery.
test('EV-R0-06 received request remains received after HTTP 503', () => {
  const f = fixture();
  f.public.observations.command.disposition = 'failed';
  f.public.observations.delivery = {state: 'received', response: 'http-503-model-failed'};
  const observed = projectEvidenceView(f).lanes.observedEffect;
  assert.equal(observed.command.disposition, 'failed');
  assert.equal(observed.effect.state, 'unobserved');
  assert.equal(observed.delivery.state, 'received');
  assert.equal(observed.delivery.response, 'http-503-model-failed');
});

// EV-R0-07: the upstream disclosure boundary, not this presenter, must
// prepare the slice. When it has not done so, there is no visible secret
// existence/count/diagnostic/ID difference between two worlds.
test('EV-R0-07 two-world unavailable fallback is non-revealing', () => {
  const w0 = fixture(), w1 = fixture();
  w0.release.state = 'unknown';
  w1.release.state = 'withheld';
  w1.public.operation.selected.id = 'SECRET-S';
  w1.public.decision.witnessDiagnosticCodes = ['SECRET_CHANGED'];
  w1.privateCount = 42;
  assert.deepEqual(projectEvidenceView(w0), projectEvidenceView(w1));
  assert.doesNotMatch(JSON.stringify(projectEvidenceView(w1)), /SECRET|42|withheld/);
});

// EV-R0-08: a malformed `prepared` slice cannot be used for disclosure.
test('EV-R0-08 unprepared or malformed public data is fail-closed', () => {
  const f = fixture();
  delete f.release;
  assert.equal(projectEvidenceView(f).publication, 'unavailable');
  const bad = fixture();
  bad.public.decision.witnessDiagnosticCodes = ['<secret>'];
  assert.throws(() => projectEvidenceView(bad), /INVALID_EV_R0_PUBLIC_SLICE/);
  const withInternal = fixture();
  withInternal.public.operation.secret = 'forbidden';
  assert.throws(() => projectEvidenceView(withInternal), /INVALID_EV_R0_PUBLIC_SLICE/);
});
