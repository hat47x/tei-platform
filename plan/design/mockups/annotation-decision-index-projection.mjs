// Read-only display of the EXISTING Shared Acceptance "decisions" CLI output.
// This is a third, independent list: it does not join an annotation, an assertion,
// a source-model mapping, or a Host-accepted Source to a Decision record.
export const DECISION_INDEX_SCHEMA = 'tei.reference.decision-index/v0';

const named = (value, known) => typeof value === 'string' && Object.hasOwn(known, value)
  ? known[value]
  : value == null || value === '' ? 'Referenceに値なし' : `未対応の値（${String(value)}）`;
const codeList = list => Array.isArray(list) && list.every(a => a && typeof a.code === 'string')
  ? (list.map(a => a.code).join('、') || 'なし')
  : 'Referenceに値なし・形式未確認';

const decisionName = { accepted: '参照実装の受理結果', rejected: '参照実装の却下結果' };
const revisionName = {
  current: '記録時のSource版と現在のWorkspace版が一致',
  superseded: '記録時のSource版と現在のWorkspace版が不一致',
};
const editName = {
  intact: '記録されたSource編集は現在も一致',
  diverged: '記録されたSource編集に不一致',
  none: '比較対象のSource編集なし',
};
const evidenceName = {
  current: 'Source根拠が現在のWorkspaceと一致',
  recorded: '記録済みArtifact（現在の外部原典は未検証）',
  'current-with-recorded-artifacts': 'Source根拠は一致、Artifactは記録のみ',
  diverged: '根拠が不一致または不正',
  none: '根拠記録なし',
};
const reviewName = {
  none: 'Review拘束の記録なし', recorded: 'Review拘束を記録',
  verified: 'Review投影の指紋整合（人の承認の証明ではない）',
  invalid: 'Review拘束が不正',
};
const coverageName = { none: 'Review範囲の記録なし', complete: 'Review A1〜A8の範囲が完全', incomplete: 'Review範囲に未評価・一部あり', invalid: 'Review範囲が不正' };
const authorityName = { unresolved: '承認資格未確認' };

// Do not turn a Decision record, a review fingerprint, or a current revision
// into a human-authorized or production-Host accepted meaning.
export function projectDecisionIndex(index) {
  if (!index || index.schema !== DECISION_INDEX_SCHEMA ||
      typeof index.modelRevision !== 'string' || !Array.isArray(index.records)) {
    throw new Error('未対応のDecision Indexです。schema、modelRevision、recordsを確認してください。');
  }
  const records = index.records.map((entry, i) => {
    if (!entry || typeof entry !== 'object' || !entry.decision || typeof entry.decision !== 'object') {
      throw new Error(`Decision Indexのrecords[${i}]にdecisionがありません。`);
    }
    const d = entry.decision;
    const details = [
      { label: 'Source編集', value: named(entry.sourceEdits, editName) },
      { label: '根拠全体', value: named(entry.evidenceStatus, evidenceName) },
      { label: 'Source根拠', value: named(entry.sourceEvidenceStatus, evidenceName) },
      { label: 'Artifact根拠', value: named(entry.artifactEvidenceStatus, evidenceName) },
      { label: 'Review拘束', value: named(entry.reviewBindingStatus, reviewName) },
      { label: 'Review範囲', value: named(entry.reviewCoverageStatus, coverageName) },
      { label: '判断者の確認', value: named(entry.decisionActorStatus, authorityName) },
      { label: '判断権限の確認', value: named(entry.decisionAuthorityStatus, authorityName) },
      { label: 'Source変更フラグ', value: typeof d.sourceChanged === 'boolean' ? String(d.sourceChanged) : 'Referenceに値なし' },
      { label: '記録後のSource版', value: typeof d.modelRevisionAfter === 'string' ? d.modelRevisionAfter : 'Referenceに値なし' },
      { label: 'Decision履歴の位置（表示のみ）', value: typeof entry.path === 'string' ? entry.path : 'Referenceに値なし' },
      { label: '拒否診断コード', value: codeList(d.refusals) },
      { label: '記録時の警告コード', value: codeList(d.warnings) },
      { label: '根拠診断コード', value: codeList(entry.evidenceDiagnostics) },
      { label: 'Review診断コード', value: codeList(entry.reviewDiagnostics) },
    ];
    const warnings = [];
    if (entry.status !== 'current') warnings.push('現在のWorkspace版との一致が確認できない');
    if (entry.sourceEdits === 'diverged') warnings.push('記録済みSource編集と現在のSourceが不一致');
    if (entry.evidenceStatus === 'diverged') warnings.push('根拠が不一致または不正');
    if (entry.reviewBindingStatus === 'invalid' || entry.reviewCoverageStatus === 'incomplete' || entry.reviewCoverageStatus === 'invalid') {
      warnings.push('Reviewの拘束または範囲に注意');
    }
    if (entry.decisionActorStatus !== 'unresolved' || entry.decisionAuthorityStatus !== 'unresolved') {
      // Only unresolved is emitted/understood by this Reference. A newer
      // status requires an actual authority consumer, not a UI guess.
      warnings.push('判断者または承認権限に未対応の状態値');
    }
    if (d.decision !== 'accepted' && d.decision !== 'rejected') warnings.push('Decision結果に未対応の値');
    if (d.decision === 'rejected') warnings.push('記録は却下結果であり、Source受理ではない');
    return {
      row: i + 1, outcome: named(d.decision, decisionName),
      sourceRevision: named(entry.status, revisionName),
      sourceEdit: named(entry.sourceEdits, editName),
      review: named(entry.reviewBindingStatus, reviewName),
      authority: named(entry.decisionAuthorityStatus, authorityName),
      warnings: warnings.join('／') || '追加注意なし（人的承認や本番受入を意味しない）',
      hasWarnings: warnings.length > 0,
      details,
      acceptedMeaning: '権限あるOwner／正式Hostの意味受理：このReferenceからは判定不可',
    };
  });
  return {
    records,
    summary: records.length === 0
      ? 'Decision履歴は0件。このledgerに記録がないことは、過去の受理操作がなかった証明ではありません。'
      : `Decision履歴：${records.length}件。これは保存済み補助記録の一覧であり、正式な承認履歴の完全性を保証しません。`,
    limitation: '操作直後の未保存Decision出力、Source復元失敗、Ownerの意味対応、人的承認資格、正式Host受理、実DB／Windows GUIはこの一覧から判定しません。',
  };
}
