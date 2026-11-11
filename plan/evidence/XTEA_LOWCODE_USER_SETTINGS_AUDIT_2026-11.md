# 利用者設定の変更監査の確認（2026-11）

## 対象

- 実装: `internal/usersetting/audit.go`（`AuditEntry`・`AuditedStore`・`Table.ApplyAudited`）
- 試験: `internal/usersetting/audit_test.go`
- 対象版: `main` の `528d5eac6` から分岐した`feat/usersetting-audit-20261111`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

利用者設定は「宣言された鍵だけを更新して記録する」ところまで実装していましたが、**誰が変えたかの記録がありませんでした**。X-TEAのスクリプトAPIは値をセッションの変数へ直接書き（`setVariant`・`setUserValueOf`: `D:Session.java` L2112、`D:XFScriptable.java` L56–60）、誰が何を変えたかを残していませんでした。「先週と画面の挙動が違う」という問いに答える手段がありません。

設計文書の残課題にも「利用者設定の監査と権限」として挙がっていました。

## 変更

- `AuditEntry`（時刻・Data・鍵・利用者・変更前・**存在したか**・変更後・**実行者**）。
- `AuditedStore`（`Store`を包み、受理された変更の履歴を追記専用で持つ。`Entries`で読む。時計は差し替え可能）。
- `Table.ApplyAudited`（検査・記録・監査を1回で行う）。

## 設計上の判断

- **実行者が空の変更は拒否**します。実行者を書かない記録は監査記録ではないためです。
- **拒否された変更は記録しません**。設定に何も起きていないからです。
- 「変更前」に**存在したか（`Had`）**を持たせます。「初回の書き込み」と「空からの変更」は別の事実だからです。
- 「変更前」は**検査に使った表**から読むので、置き換えた値と記録が食い違いません。

## 観測

- `TestApplyAuditedRecordsWhoChangedWhat` 成功。`alice`の変更が1件記録され、`before=50`・`after=100`・時刻が固定時計から取られる。
- `TestApplyAuditedDistinguishesAFirstWrite` 成功。宣言の無い鍵への変更は拒否され、**記録も残らない**。
- `TestApplyAuditedRefusesAnUnattributedChange` 成功。実行者なしは拒否され、履歴も現在値も変わらない。
- `TestApplyAuditedKeepsTheLogAcrossChanges` 成功。2件が**順序を保って**残り、2件目の`before=100`が読める。
- パッケージ全体は`ok`。

## 読み取れること

- 設定の変更が**誰によって・いつ・何から何へ**変わったかとして残る。
- 拒否された変更は履歴に現れない（記録と実状態が一致する）。
- 実行者なしの変更は通らない。

## 読み取れないこと

- **権限**（誰がどの設定を変えられるか）。今は実行者を記録するだけで、許可の判断はしていません。
- 監査記録の**永続化**（`AuditedStore`は記憶域）。
- 監査記録の**照会・改ざん検出**（追記専用ですが、ハッシュ鎖などはありません）。
- CLI/Studioからの設定変更経路。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。変更監査の記録は参照実装で確認済み、権限・永続化・照会は未実装です。
