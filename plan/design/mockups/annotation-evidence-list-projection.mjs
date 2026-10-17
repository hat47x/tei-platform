// Read-only UI projection of two existing Reference results. Not a source/mapping join.
export const AUXILIARY_SCHEMA = 'tei.reference.authoring-auxiliary/v0';
export const EPISTEMIC_CONTRACT = 'tei.epistemic-projection/v0';
export const EPISTEMIC_SCHEMA = 'tei.reference.epistemic-assessment/v0';

const noteKind = { issue: '課題', question: '質問', note: 'メモ' };
const noteState = { open: '注記：未対応', done: '注記：対応済み' };
const confirmation = { unreviewed: '内容：人による確認なし', confirmed: '内容：人が確認', rejected: '内容：人が否認' };
const binding = {
  exact: '対象位置：一意', reanchored: '対象位置：再特定',
  ambiguous: '対象位置：曖昧', detached: '対象位置：参照切れ',
  'not-required': '対象位置：位置指定なし',
};
const use = {
  premise: '利用：作業前提', 'candidate-only': '利用：候補のみ',
  'review-required': '利用：要確認', blocked: '利用：不可',
};

const shown = (dictionary, value) => Object.hasOwn(dictionary, value) ? dictionary[value] : `未対応の値（${String(value ?? '')}）`;
const text = value => typeof value === 'string' ? value : '';
const array = value => Array.isArray(value) ? value : [];

// This transient display projection never infers an X-TEA↔TEI mapping, Owner acceptance,
// Canonical Acceptance, or a join between note.subject and assessment.assertionId.
export function projectAnnotationEvidence(auxiliary, epistemic) {
  if (!auxiliary || auxiliary.schema !== AUXILIARY_SCHEMA ||
      !Array.isArray(auxiliary.annotations) || !Array.isArray(auxiliary.findings)) {
    throw new Error('未対応の補助情報Reference結果です。schemaとannotations/findingsを確認してください。');
  }
  if (!epistemic || epistemic.contract !== EPISTEMIC_CONTRACT ||
      epistemic.schema !== EPISTEMIC_SCHEMA || !Array.isArray(epistemic.assessments)) {
    throw new Error('未対応の知識評価Reference結果です。contract/schemaとassessmentsを確認してください。');
  }

  const findingsByID = new Map();
  for (const f of auxiliary.findings) {
    const key = text(f.subject);
    if (!findingsByID.has(key)) findingsByID.set(key, []);
    findingsByID.get(key).push(text(f.code));
  }

  const notes = auxiliary.annotations.map(a => ({
    id: text(a.id), subject: text(a.subject), kind: shown(noteKind, a.kind),
    workState: shown(noteState, a.state), text: text(a.text),
    findings: array(findingsByID.get(text(a.id))),
    // Missing subject is also apparent when an unvalidated report is supplied.
    locator: a.subject ? '意味上の参照あり（同一性・存続は未検証）' : '意味上の参照なし',
    acceptance: '対応受理：このReferenceからは判定不可',
  })).sort((a, b) => a.id.localeCompare(b.id, 'ja'));

  const assessments = epistemic.assessments.map(a => ({
    id: text(a.assertionId), statementKind: text(a.statementKind),
    confirmation: shown(confirmation, a.confirmationState),
    targetBinding: shown(binding, a.targetBinding),
    useState: shown(use, a.useState),
    reviewBinding: text(a.reviewBinding),
    needsLocatorReview: a.targetBinding === 'ambiguous' || a.targetBinding === 'detached',
    acceptance: '対応受理・Canonical受理：このReferenceからは判定不可',
  })).sort((a, b) => a.id.localeCompare(b.id, 'ja'));

  return { notes, assessments,
    limitations: [
      '注記のopen/doneは原典↔TEI対応の受理ではありません。',
      '内容のhuman confirmedはOwnerの対応受理やCanonical Acceptanceではありません。',
      '二つのReference結果をsubjectや文字列の一致で結合しません。',
      '原典確認、source/model mapping、production接続、Windows GUI実行はこの表示から判定しません。',
    ],
  };
}
