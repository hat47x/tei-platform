# ADR-0041: 生成物の来歴・選抜・artifact・auditを分離し、実行権限へ接続しない

- 状態: 提案中
- 日付: 2026-10-24
- 関連ADR: ADR-0022 / ADR-0034 / ADR-0035 / ADR-0037 / ADR-0040
- 関連Issue: #2016 / #943 / #1104 / #1370
- 関連実装: [`reference-generator-selection/`](../../tools/reference-generator-selection/README.md)、[`reference-generator-attestation/`](../../tools/reference-generator-attestation/README.md)、[`reference-generator-artifact-store/`](../../tools/reference-generator-artifact-store/README.md)

## 背景

#2016の参照実装は、生成器候補の選抜に加えて、次の記録を持つようになった。

- 固定fixtureの選抜receiptとbuild/execution evidence
- 候補statementの`ssh-keygen`署名
- source/binaryのcontent-addressed artifact store
- source path非依存の反復build
- 明示keepによるpruneと、hash-chain audit
- audit checkpointの署名・再検証

これらは別々の証拠だが、一つの「信頼済み」flagや「合格」にまとめると、何を確認したのか分からなくなる。特に、署名が成立したこと、artifactが保存されていること、prune auditが正しいことを、実行権限や意味の受理と混同してはならない。

## 判断

### 1. 選抜receiptはfixture適合の証拠であり、正しさや権限ではない

選抜receiptは、受理済み契約・固定fixtureに対して候補が一致したことを示す。候補の一般的な正しさ、実Provider・実Targetでの動作、意味の受理、Hostの実行許可を証明しない。`grantsRuntimeAuthority`と`provesRealProvider`はfalseのままにする。

### 2. statement署名はbyte列の束縛であり、署名者のAuthorityではない

`ssh-keygen`の署名は、statementのbyte列が署名後に改変されていないこと、caller指定のallowed signersで署名が成立することを示す。署名者のrole・Authority、trust root、鍵の失効、候補の正しさは検証しない。署名の成立を実行許可にしない。

### 3. artifact storeはbyteの同一性を確認するが、実行を承認しない

artifact storeは、source digestとbinary digestを再計算し、content-addressed pathへ保存・取得する。保存後に改変されていないこと、receiptと実byteが一致することを確認する。候補の実行承認、実Targetへの配備、意味の受理は扱わない。

### 4. retentionとauditは明示操作の記録であり、削除権限ではない

pruneは明示したkeep setだけを残し、空keep・未知digest・検証失敗objectでは削除しない。成功したpruneはhash-chain auditへ記録し、checkpointでheadとaudit file digestを参照できる。checkpointは署名・再検証できるが、署名者のAuthority・削除承認・保持期間を決めない。auditのhash chainはローカルのtamper-evident記録であり、chain全体の置換や作り直しを防がない。

### 5. 再現性の主張は環境と範囲を明示する

反復buildの一致は、同じ環境・同じtoolchain・同じsourceでの範囲を示す。source path非依存の一致は確認できるが、別machine・別toolchain・clean cache・外部依存での再現性を証明しない。artifact storeの保存や署名を、再現性の証明に使わない。

### 6. 単一の「信頼済み」flagを作らない

選抜、署名、artifact保存、audit、再現性は、それぞれ別のreceipt・evidenceとして残す。利用者は、必要な証拠を明示的に組み合わせる。どれか一つの`passed=true`を、他の証拠の代替や実行許可にしない。

## 代替案

- **選抜合格を候補の正しさ・実行許可とする**: fixture範囲を一般の保証へ拡大するため採用しない。
- **署名済みstatementを信頼の正本とする**: 署名者のAuthority・trust rootを検証しないため採用しない。
- **artifact storeの存在を配備承認とする**: byteの同一性と実行承認を混同するため採用しない。
- **audit checkpointの署名を削除承認とする**: 署名はbyte束縛であり、削除の権限・保持期間を決めないため採用しない。
- **反復build一致を別環境の再現性とする**: 環境差を無視するため採用しない。

## 影響

- 生成物の来歴は複数のreceipt・fileへ分かれる。単一の判定にまとめたい利用者は、必要な証拠を自分で選択する必要がある。
- 署名・artifact・auditの検証toolは、それぞれの範囲だけを報告する。
- Authority、削除承認、本番配備の受理は、ADR-0022 / ADR-0035のA4として人へ戻す。
- 将来、Build attestationやartifact bindingの正本を追加する場合も、この分離を維持する。

## 未決

- 選抜・署名・artifact・auditを、運用上どの単位で一つの変更票へ束ねるか。
- 署名者のAuthority・trust rootを既存のAuthority Evidenceへどう接続するか。
- 保持期間・削除承認・監査アクセス権を誰が決めるか。
- 別machine・別toolchainでの再現性をどこまで確認するか。
- artifact storeの保持対象を、Go以外の候補へ広げるか。

## 検証

- 選抜receipt、署名receipt、artifact storeのput/get、prune audit、checkpoint、署名済みcheckpoint検証の試験が、それぞれの範囲だけを報告する。
- 改変されたsource・binary・statement・audit・checkpointを拒否する。
- `grantsAuthority`、`provesRealProvider`、`grantsRuntimeAuthority`、`provesRealTarget`がfalseである。
- 実Provider・実DB・実Target・本番配備を証明しないことをREADMEとPRに明記する。

このADRは提案中であり、採用判断は人間の受理境界に残す。効果が確認できない場合は、証拠の分離を参照実装の範囲へ戻す。
