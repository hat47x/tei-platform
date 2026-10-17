import { test } from 'node:test';
import assert from 'node:assert/strict';
import { projectAnnotationEvidence, AUXILIARY_SCHEMA, EPISTEMIC_CONTRACT, EPISTEMIC_SCHEMA } from './annotation-evidence-list-projection.mjs';

const fixture = () => ({
  auxiliary: { schema: AUXILIARY_SCHEMA, annotations: [
    { id: 'n2', subject: '', kind: 'issue', state: 'open', text: '<img src=x onerror=alert(1)>' },
    { id: 'n1', subject: 'Data.Order', kind: 'question', state: 'done', text: 'Reviewed note; not an accepted mapping' },
  ], findings: [{ code: 'annotation-without-subject', subject: 'n2' }] },
  epistemic: { contract: EPISTEMIC_CONTRACT, schema: EPISTEMIC_SCHEMA, assessments: [
    { assertionId: 'e3', statementKind: 'hypothesis', confirmationState: 'confirmed', targetBinding: 'exact', useState: 'candidate-only' },
    { assertionId: 'e2', statementKind: 'fact', confirmationState: 'unreviewed', targetBinding: 'detached', useState: 'review-required' },
    { assertionId: 'e1', statementKind: 'fact', confirmationState: 'rejected', targetBinding: 'ambiguous', useState: 'blocked' },
  ] },
});

test('注記の完了・人の確認・対応受理を混同せず、独立した一覧へ投影する', () => {
  const { auxiliary, epistemic } = fixture();
  const before = structuredClone({ auxiliary, epistemic });
  const got = projectAnnotationEvidence(auxiliary, epistemic);
  assert.deepEqual({ auxiliary, epistemic }, before, 'read-only');
  assert.deepEqual(got.notes.map(x => x.id), ['n1', 'n2']);
  assert.match(got.notes[0].workState, /対応済み/);
  assert.match(got.notes[0].acceptance, /判定不可/);
  assert.match(got.assessments.find(x => x.id === 'e3').confirmation, /人が確認/);
  assert.match(got.assessments.find(x => x.id === 'e3').useState, /候補のみ/);
  assert.match(got.assessments.find(x => x.id === 'e3').acceptance, /判定不可/);
  assert.equal(got.notes[0].subject, 'Data.Order');
  assert.equal(got.assessments.length, 3, '注記のsubjectを勝手にassertionと結合しない');
});

test('対象位置が曖昧・参照切れでも別要素へ付け替えない', () => {
  const { auxiliary, epistemic } = fixture();
  const got = projectAnnotationEvidence(auxiliary, epistemic);
  assert.deepEqual(got.assessments.filter(a => a.needsLocatorReview).map(a => a.id), ['e1', 'e2']);
  assert.match(got.assessments[0].targetBinding, /曖昧/);
  assert.match(got.assessments[1].targetBinding, /参照切れ/);
  assert.match(got.notes[1].locator, /参照なし/);
  assert.deepEqual(got.notes[1].findings, ['annotation-without-subject']);
});

test('未知のcontract/schemaはfail closed、未知の状態値は受理へ昇格しない', () => {
  const { auxiliary, epistemic } = fixture();
  assert.throws(() => projectAnnotationEvidence({ ...auxiliary, schema: 'unknown' }, epistemic), /未対応/);
  assert.throws(() => projectAnnotationEvidence(auxiliary, { ...epistemic, contract: 'unknown' }), /未対応/);
  assert.throws(() => projectAnnotationEvidence(auxiliary, { ...epistemic, schema: 'unknown' }), /未対応/);
  assert.throws(() => projectAnnotationEvidence({ ...auxiliary, findings: null }, epistemic), /未対応/);
  const unknown = projectAnnotationEvidence({ ...auxiliary, annotations: [{ id: 'unknown', subject: '', kind: '??', state: 'accepted' }] }, epistemic);
  assert.match(unknown.notes[0].workState, /未対応の値/);
  assert.doesNotMatch(unknown.notes[0].workState, /注記：対応済み/);
});
