# reference-generator-revocation-review（#2076）

生成器registryの`revoked` entryと、registry編集とは**別の承認record**を照合する参照です。失効の対象・版・source digestが一致し、承認者のauthority・role・evidence digest・revisionが揃っているかを検査します。

ただし、承認者のAuthority、issuer trust、署名、失効判断の妥当性は検証しません。`passed=true`は承認recordが揃っていることだけを示し、実行権限や失効の受理を与えません。

```text
go run ./tools/reference-generator-revocation-review/cmd/tei-generator-revocation-review \
  -registry tools/reference-generator-selection/testdata/registry-lifecycle.json \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -approvals tools/reference-generator-revocation-review/testdata/approvals.json
```

## 承認record

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

- `id`・`version`・`sourceDigest`は、registryのrevoked entryと一致しなければなりません。
- `authority`・`role`・`evidenceDigest`・`approvedRevision`は空を許しません。
- `sourceDigest`と`evidenceDigest`は`sha256:<64 hex>`形式です。
- 同じ`id@version`の承認recordを重複させません。
- registryに存在しない対象、revokedでない対象への承認は`invalid`です。

## 状態

| 状態 | 意味 |
| --- | --- |
| `approved-syntax-only` | 対象一致と必須fieldが揃っている。Authority検証ではない |
| `unapproved` | revoked entryに承認recordがない、またはsource digestが一致しない |
| `invalid` | 対象がregistryにない、またはrevokedでないentryへの承認 |

`passed`は`unapproved`と`invalid`が0件の場合だけtrueです。

## 検査しないこと

- 承認者が本当にそのrole・authorityを持つか。
- evidenceの署名・発行元・失効・trust root。
- 失効判断が業務上・安全上正しいか。
- registryの`revoked`宣言がHostの実行受理へ反映されているか。
- 失効後の生成物が再生成・再選抜されたか。

これらは#951（Contract/Provider Authority）、#1104 / #1370（Build attestation / artifact binding）、#2016（再生成・再選抜）の境界へ戻します。

## 失効から再生成までの分離

参照実装では、次の記録を混ぜません。

1. **失効提案**: 対象id・version・source digest・理由。
2. **承認record**: authority・role・evidence digest・revision。
3. **registry宣言**: `revoked`と対象digest。registry編集だけでは承認を代替しない。
4. **再生成・再選抜**: 新しい候補の選抜receiptと`reconcile`のcurrent判定。

承認recordが揃っても、再生成・再選抜が未完了なら、失効した生成物を現在の証拠として使いません。

## 検証

`go test ./tools/reference-generator-revocation-review/... -count=1`で、次を確認します。

- 正しい対象の承認recordを`approved-syntax-only`として受理。
- 承認record欠落をfail closed。
- source digest不一致をfail closed。
- revokedでないentryへの承認をinvalid。
- registryにないentryへの承認をinvalid。
- 重複承認、authority欠落を拒否。
- `grantsAuthority`と`verifiesIssuerTrust`が常にfalse。

## 未実装・限界

- 承認者・role・authorityの実在性、署名、trust rootの検証。
- 承認の取り消し・失効・有効期間。
- 監査ログ・Build Evidence・Host Authorityへの接続。
- 失効後の自動再生成・再選抜。
- 実運用の承認workflow、UI、常駐サービス。
