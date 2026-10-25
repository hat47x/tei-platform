# ADR-0040: 生成器の失効承認をregistryの編集と分離し、承認recordの対象一致だけを検証する

- 状態: 提案中
- 日付: 2026-10-21
- 関連ADR: ADR-0022 / ADR-0034 / ADR-0037 / ADR-0035
- 関連Issue: #2076 / #2016 / #951 / #1104 / #1370
- 関連実装: [`tools/reference-generator-revocation-review/`](../../tools/reference-generator-revocation-review/README.md)、[設計案](../design/GENERATOR_REVOCATION_APPROVAL_BOUNDARY_2026-10.md)

## 背景

ADR-0037で、生成器の失効・置換を意味・生成物・実行受理の版から分離した。registryは`revoked`を宣言できるが、次の問いは残る。

- 誰が失効を承認したか。
- 承認時と現在で、対象id・version・source digestが一致しているか。
- registry編集と承認recordを同じ差分に混ぜてよいか。
- 承認recordが揃っただけで、Hostの実行許可や意味の受理になったとみなしてよいか。

registryの編集だけで失効を成立させると、承認と権限の境界を迂回する。

## 判断

### 承認recordをregistryから分ける

失効の承認は、registryとは別のrecordとして固定する。承認recordは少なくとも次を持つ。

- 契約のid・version・digest
- 対象生成器のid・version・source digest
- authority・role
- evidence digest
- approved revision
- 理由

registryは`revoked`と対象digestを宣言し、承認recordを参照する。registry編集だけで承認を代替しない。

### 対象一致だけを機械検証する

参照reviewは、revoked entryと承認recordのid・version・source digest・契約identityが一致すること、必須fieldとdigest形式が揃っていること、重複・未知対象・非revoked対象がないことを検査する。

これは**承認recordの存在と対象一致**の検証であり、承認者のAuthority・roleの妥当性、evidenceの署名・issuer trust、失効判断の正しさを検証しない。`grantsAuthority`と`verifiesIssuerTrust`は常にfalseとする。

### 失効・置換・再生成を別の完了条件にする

- 失効宣言: registryの`revoked`。
- 承認: 対象一致した承認record。
- 置換: 新しい生成器版の`active`。
- 再生成・再選抜: 新候補の固定fixture選抜receiptと`reconcile`の`current`。
- 実行受理: 既存Host Authority。

承認recordが揃っても、再生成・再選抜が未完了なら、失効した生成物を現在の証拠として使わない。失効の完了をregistry編集やreviewの`passed=true`だけで宣言しない。

### 過去の生成物を監査記録として残す

失効した生成器が過去に出した生成物・適合結果は、現在の証拠から外すが、監査・再現のために履歴として保持する。削除・保持期間・アクセス権は別の運用判断とする。

### Authorityを複製しない

承認recordは既存のAuthority Evidence / Build Evidenceへ接続できるかを評価する参照であり、新しいAuthority・常駐サービス・承認UIを作らない。意味の受理・権限付与はADR-0022 / ADR-0035のA4として人へ戻す。

## 代替案

- **registryに承認者名を同居させる**: 承認recordとregistry編集が同じ差分になり、分離できないため採用しない。
- **承認者名の自己申告だけで失効を成立させる**: 宣言とAuthorityを混同するため採用しない。
- **reviewのpassed=trueを実行許可にする**: 承認recordの存在と実行権限を混同するため採用しない。
- **失効と同時に過去の生成物を削除する**: 監査・再現の証拠を失うため採用しない。
- **失効後に自動で再生成・再選抜する**: 新しい候補の適合と受理を迂回するため採用しない。

## 影響

- registryとは別に承認recordの形式と保存場所が必要になる。
- 失効reviewは、対象一致と必須fieldの検査に限定される。
- 承認の取り消し・有効期間・trust rootは未決として残る。
- 失効から再生成・再選抜までの作業単位を、既存receiptで追う必要がある。
- 監査記録と現在の証拠を分けるため、候補の履歴保持が必要になる。

## 未決

- 承認recordを既存のAuthority Evidence / Build Evidenceのどの形式へ寄せるか。
- 承認の有効期間・取り消し・再承認。
- 承認者・roleのtrust rootの管理者。
- 失効した生成物の保持期間・監査アクセス権。
- 再生成・再選抜の完了を機械確認するreceipt。

## 検証

#2076の参照実装で、次を確認する。

- 正しい対象の承認recordを`approved-syntax-only`として受理する。
- 承認record欠落、source digest不一致をfail closedにする。
- 非revoked対象・registryにない対象への承認をinvalidにする。
- 重複承認、authority欠落を拒否する。
- `grantsAuthority`と`verifiesIssuerTrust`が常にfalseである。

この検証は、承認者のAuthority、署名、trust root、Hostの実行受理、実Provider・実Target、本番配備を証明しない。効果が確認できない場合は、承認recordの形式を参照実装の範囲へ戻す。
