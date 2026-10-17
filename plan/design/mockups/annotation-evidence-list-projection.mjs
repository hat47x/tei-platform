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

const reviewBinding = { none: '確認履歴：なし', current: '確認履歴：現在の対象に有効', stale: '確認履歴：現在の対象には無効' };
const metadata = { complete: 'メタデータ：完全', partial: 'メタデータ：一部欠落', absent: 'メタデータ：欠落' };
const origin = {
  user: '内容の由来：利用者', ai: '内容の由来：AI', system: '内容の由来：システム',
  imported: '内容の由来：取込み情報', external: '内容の由来：外部情報',
  unknown: '内容の由来：不明',
};
const ingestion = {
  user: '取込み主体：利用者', ai: '取込み主体：AI', system: '取込み主体：システム',
  connector: '取込み主体：コネクター', import: '取込み主体：インポート',
};
const compatibility = { compatible: '文脈：適合', incompatible: '文脈：不適合', unknown: '文脈：判断不可' };
const freshness = {
  current: '鮮度：現在有効', stale: '鮮度：古い',
  'not-yet-valid': '鮮度：有効期間前', unknown: '鮮度：判断不可',
};
const lifecycle = { active: 'ライフサイクル：有効', superseded: 'ライフサイクル：後続情報あり' };
const conflict = { none: '競合：なし', inactive: '競合：現在は非活性', present: '競合：あり' };

// All labels are display-only readings of fields from the existing Assessment.
// No one-line "approved" or "healthy" status is synthesized from the axes.
function inspection(a) {
  const axes = [
    ['確認履歴の現在性', 'reviewBinding', reviewBinding],
    ['メタデータの完全性', 'metadataState', metadata],
    ['内容の由来', 'contentOrigin', origin],
    ['取込み主体', 'ingestedBy', ingestion],
    ['文脈への適合性', 'contextCompatibility', compatibility],
    ['情報の鮮度', 'freshness', freshness],
    ['情報の存続状態', 'lifecycleState', lifecycle],
    ['競合の状態', 'conflict', conflict],
  ];
  const detailRows = axes.map(([label, key, dictionary]) => ({
    label, value: a[key] == null || a[key] === '' ? 'Referenceに値なし' : shown(dictionary, a[key]),
  }));
  detailRows.push({
    label: '推定した文脈の利用',
    value: typeof a.inferredContextUsed === 'boolean'
      ? (a.inferredContextUsed ? 'あり' : 'なし') : 'Referenceに値なし',
  });
  detailRows.push({
    label: '評価理由（Reference出力）',
    value: Array.isArray(a.reasons)
      ? (a.reasons.filter(value => typeof value === 'string').join('／') || '理由の記載なし')
      : 'Referenceに値なし',
  });

  const attention = [];
  const caution = (predicate, message) => { if (predicate) attention.push(message); };
  caution(a.targetBinding === 'ambiguous' || a.targetBinding === 'detached', '対象位置の確認が必要');
  caution(a.reviewBinding === 'stale', '過去の内容確認は現在の対象に適用できない');
  caution(a.metadataState === 'partial' || a.metadataState === 'absent', 'メタデータが不足');
  caution(a.contentOrigin === 'unknown', '情報の由来が不明');
  caution(a.contentOrigin === 'ai' && a.ingestedBy === 'user', '利用者の取込みはAI生成情報の内容確認ではない');
  caution(a.contextCompatibility === 'unknown' || a.contextCompatibility === 'incompatible', '文脈の適合性に注意');
  caution(a.freshness === 'stale' || a.freshness === 'not-yet-valid' || a.freshness === 'unknown', '情報の鮮度に注意');
  caution(a.lifecycleState === 'superseded', '後続情報による置換あり');
  caution(a.conflict === 'present', '競合が存在');
  caution(a.inferredContextUsed === true, '推定した文脈を使用');

  // If a caller pasted a partial/older payload, show that fact rather than
  // supplying default-positive values. This is NOT a new schema validator.
  const missing = axes
    .filter(([, key]) => a[key] == null || a[key] === '')
    .map(([label]) => label);
  if (typeof a.inferredContextUsed !== 'boolean') missing.push('推定した文脈の利用');
  if (!Array.isArray(a.reasons)) missing.push('評価理由');
  if (missing.length) attention.push('詳細情報の未提供：' + missing.join('、'));
  for (const [label, key, dictionary] of axes) {
    if (a[key] != null && a[key] !== '' && !Object.hasOwn(dictionary, a[key])) {
      attention.push(label + 'に未対応の状態値');
    }
  }
  if (a.confirmationState != null && !Object.hasOwn(confirmation, a.confirmationState)) {
    attention.push('人の内容確認に未対応の状態値');
  }
  if (a.useState != null && !Object.hasOwn(use, a.useState)) {
    attention.push('利用状態に未対応の状態値');
  }
  if (a.targetBinding != null && !Object.hasOwn(binding, a.targetBinding)) {
    attention.push('対象位置に未対応の状態値');
  }
  return {
    detailRows,
    attention,
    hasAttention: attention.length > 0,
    attentionSummary: attention.length ? attention.join('／') : '追加の注意表示なし（受理・全体健全性を意味しない）',
  };
}


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
    ...inspection(a),
    acceptance: '対応受理・Canonical受理：このReferenceからは判定不可',
  })).sort((a, b) => a.id.localeCompare(b.id, 'ja'));

  return { notes, assessments,
    limitations: [
      '注記のopen/doneは原典↔TEI対応の受理ではありません。',
      '内容のhuman confirmedはOwnerの対応受理やCanonical Acceptanceではありません。',
      '二つのReference結果をsubjectや文字列の一致で結合しません。',
      '個別Assessmentはproject-wide Healthやconsumer固有の鮮度適合を証明しません。',
      '原典確認、source/model mapping、production接続、Windows GUI実行はこの表示から判定しません。',
    ],
  };
}
