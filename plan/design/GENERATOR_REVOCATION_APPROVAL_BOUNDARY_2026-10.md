# 生成器の失効承認・監査・再生成の境界（2026-10）

- 状態: **評価待ち（未採用）**
- 日付: 2026-10-21
- 担当: [Issue #2076](https://github.com/hat47x/tei-platform/issues/2076)
- 関連: #2016 / #951 / #1104 / #1370 / ADR-0037 / ADR-0040
- 参照実装: [`tools/reference-generator-revocation-review/`](../../tools/reference-generator-revocation-review/README.md)

## 1. 問題

#2016のregistryは`active`・`superseded`・`revoked`を区別しますが、`revoked`はプロジェクト内の宣言にすぎません。次の問いが残ります。

- 誰が、どの根拠で失効を承認したか。
- 失効の対象id・version・source digestが、承認時と現在で一致しているか。
- 承認recordとregistry編集を同じファイルへ混ぜてよいか。
- 失効した生成器の過去の生成物を、現在の証拠と監査記録のどちらとして扱うか。
- 置換・再生成・再選抜がどこまで進めば、失効を完了とみなすか。

registryの編集だけで失効を成立させると、意味の受理・Provider適合・Hostの実行許可を迂回します。

## 2. 検証仮説

失効を、少なくとも4つの別recordへ分ければ、承認と実行受理を混同せずに扱える。

1. **失効提案**: 対象id・version・source digest・理由。
2. **承認record**: authority・role・evidence digest・approved revision。
3. **registry宣言**: `revoked`と対象digest。承認recordを参照するが、置き換えない。
4. **再生成・再選抜の証拠**: 新しい候補の選抜receiptと`reconcile`のcurrent判定。

参照実装は2と3の**対象一致**だけを検査する。1と4の妥当性、2のAuthority・trustは別の境界へ残す。

## 3. 承認recordの候補形式

```json
{
  "schema": "tei.reference.generator-revocation-approvals/v0",
  "contractId": "sales.line-amount",
  "contractVersion": "1",
  "contractDigest": "sha256:...",
  "approvals": [
    {
      "id": "dangerous",
      "version": "1",
      "sourceDigest": "sha256:...",
      "authority": "release-board",
      "role": "generator-owner",
      "evidenceDigest": "sha256:...",
      "approvedRevision": "rev-2026-10-20",
      "reason": "float64と欠落0化のため使用しない"
    }
  ]
}
```

- 契約id・version・digestを承認record側にも固定し、別契約の承認を混ぜない。
- 対象の`id@version`と`sourceDigest`をregistryと照合する。
- `authority`・`role`・`evidenceDigest`・`approvedRevision`を必須にする。
- 同一対象の重複承認、registryにない対象、revokedでない対象への承認を拒否する。
- 承認record自体をregistryへ埋め込まず、registry編集との差分を残す。

## 4. 検査するもの・しないもの

| 検査する | 検査しない |
| --- | --- |
| 承認recordのschemaと契約identity | 承認者が実在するか |
| revoked entryとのid・version・source digest一致 | authority・roleの妥当性 |
| 必須fieldとdigest形式 | evidenceの署名・issuer trust |
| 重複・未知対象・非revoked対象 | 失効判断の業務上・安全上の正しさ |
| registryと承認recordのdigest | Hostの現在受理・実行権限 |

`passed=true`は「承認recordが揃っている」ことだけを示す。`grantsAuthority`と`verifiesIssuerTrust`は常にfalseとする。

## 5. 失効から再生成まで

1. 失効提案を作る。
2. 承認recordを別fileで固定する。
3. registryの対象entryを`revoked`へ変更する。registry編集だけでは承認を代替しない。
4. 承認recordとregistryの対象一致をreviewする。
5. 置換先の生成器候補を生成し、固定fixtureで再選抜する。
6. `reconcile`で旧候補が`revoked`/`superseded`、新候補が`current`であることを確認する。
7. 旧候補の適合結果は履歴として保持し、現在の証拠に再利用しない。

5〜7が未完了の場合、失効は「宣言済みだが再生成未完了」であり、成功として扱わない。完了条件の正本は#2016と#1104 / #1370の範囲へ戻す。

## 6. 証拠の境界

- 参照実装は、承認recordの存在と対象一致だけを確認する。
- 承認者・role・authorityの実在性、署名、trust root、承認の取り消しは未検証。
- registryの`revoked`はHostの実行受理ではない。
- 失効した生成物の再生成・再選抜は未実装。
- 実運用の承認workflow・監査ログ・UIは対象外。

## 7. 代替案

- **registryに`revoked`と承認者名を同居させる**: 承認recordの改変とregistry編集が同じ差分になり、分離できないため採用しない。
- **承認者名の自己申告だけで失効を成立させる**: Authorityと宣言を混同するため採用しない。
- **失効と同時に過去の生成物を削除する**: 監査・再現の証拠を失うため採用しない。
- **失効後に自動で再生成・再選抜する**: 新しい候補の適合と受理を迂回するため採用しない。

## 8. 未決

- 承認recordを既存のAuthority Evidence / Build Evidenceのどの形式へ寄せるか。
- 承認の有効期間・取り消し・再承認をどう表すか。
- 承認者・roleのtrust rootを誰が管理するか。
- 失効した生成物の保持期間と監査アクセス権。
- 再生成・再選抜の完了を、どのreceiptで機械確認するか。

## 9. 非目標

- 失効の判断を完全自動化すること。
- 署名・attestation・artifact store自体の実装（#1104 / #1370）。
- 実Provider・実Target・本番配備の受理。
- 新しい常駐サービスや独自の承認UI。
