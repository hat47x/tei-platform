# Table lifecycle / consequence 最小Reference（#1129）

状態: **未採用の合成Reference・仮実装**。本ディレクトリはGoの実処理と負例を持つが、TEIのCanonical schema／Driver／実DB接続でも旧X-TEAの実行同等性でもない。

## 対象と処理順

受注 `pending` → `submitted` を、入力検証 → 既存認可Capabilityに相当する外部判定口 → 現在行の読取り・期待版確認 → staged write → owner側での在庫予約・履歴作成 → commit → 通知outbox登録、の順序で実行する。通知送信は独立した `SendPending` 操作であり、commit前の外部通知は禁止する。

別の取消業務操作 `ExecuteCancel` は `submitted` → `cancelled` の期待版確認、予約削除、在庫返還、履歴追加を同一staged transactionで行う。業務上の取消と `context.Context` の処理中止は別物である。

- 同一Store内のmutexが、old row/version照合と新row・関連データ・receipt・outboxのcommitを同一境界にする。
- 期待版／from状態不一致は `not_submitted`。認可拒否は `denied`。Provider障害・契約違反はGo error。commit後ack喪失は `unknown` であり、「何もしなかった」を意味しない。
- `OperationID` の同一要求再送は確定済みreceiptを返し、別要求やSubmitとCancelでのID再利用は拒否する。Submitは `Lookup`、業務取消は `LookupCancel` で応答喪失後に確定結果を回復する。いずれもreceipt取得には再認可を必須とし、要求全体の一致がなければ結果を開示しない。対象の正式な業務Action権限やcommit時の認可版保証はこの合成Storeでは成立しない。
- 在庫のEntity跨ぎ変更は `applyReservation` のowner内業務Actionで実行し、外部Scriptを直接任意実行する公開hookを設けない。
- 通知outbox送信はcommit後のみ。同一プロセスでの並行・再入送信は阻止する。**リモート送信に失敗した場合は結果が不明**であり、同一 `OperationID` での再送に対する受信側重複排除が必要。外部通知のexactly-onceは保証しない。
- 派生値は純粋計算としてのみ扱い、このReferenceから保存値更新を自動実行する規則は設けない。

## 検証

```sh
go test -race -count=10 ./reference/table-lifecycle
go vet ./reference/table-lifecycle
```

対象：正常な状態更新と取引内副作用、業務取消に伴う削除、意図的なcommit前障害と処理中断、前版値の競合、認可拒否と障害の区別、ack喪失後のreceipt照会、要求識別子衝突、同時2要求の一意確定、通知失敗と再入防止。

## 所有境界と未完了

- **ADR-0024／0025はいずれも提案中**。本Referenceで採用済みと解釈せず、Semantic Lifecycle構文／状態Canonical ID／CapabilityのOutcomeを変更しない。ADRの評価・正式受理に先立って設計選択を既成事実化しない。
- `#1174` の認可根拠版・実SQLite/PostgreSQL Effect Fenceを代替しない。この参照の `Authorizer` はcommit直前の権限失効を原子的に保証しない。実Provider接続時は既存Effect Fenceの実装と保証契約を利用する。
- `#1640` D2のSELECT集合演算、`#1692` D4のRule Binding、`#1637` D1の純粋導出は別担当の正本。ここに複製しない。
- `#940` のX-TEA Driver実機観測、実DB永続化、再起動後のoutbox、分散配送、認証済み主体と正式Authority、真のPlugin選択・Canonical Source受理、外部Effect保証は**未検証**。
- Provider実装の前に権限版照合、外部通知の宛先、Transaction境界、durable receipt、二重送信防止を契約に明示し、独立した実DB・実ランタイム試験を必要とする。

本書は`#1129`のInsert/Update/Delete・Consequence残存責務を扱う最初の縦切りであり、全ユースケースをカバーした完了宣言ではない。


## X-TEAのsource-derived Evidenceとの対応（2026-10-15、#W1）

根拠: [#940 archive照合](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6181920689)。この節は**旧実行系のソースから追える事実と未検証境界**を記録するだけであり、本Referenceのtransaction phaseが旧製品と同等だという宣言ではない。

- Driverの `XFTableEvaluator.select()` はBRを主SELECTの前に呼び、取得した行へAR/BR/ARを順次適用する（source locator: `XFTableEvaluator.java:434-493`）。Queryでの導出・Filter/Sort/Limit順序を保証する情報ではないため、D2 #1640へ委譲する。
- DriverのInsertはBC→参照値・キー等の確認→SQL実行→row count 1ならAC、UpdateはBU→参照値・入力検査→SQL→AU、DeleteはBD→参照値・検査→SQL→ADのsource path（同 `511-565`, `580-625`, `636-680`）。これらをTEIの公開Script hookとして採用せず、ここではowner側で段階的な書込みを行う合成Referenceに限る。
- **DeleteにはDriver/Serverソース間の条件差がある。** Driverの `delete()` は `delete(false)` を呼ぶ一方、SQL実行条件に `isCheckOnly` が現れ、Serverでは `!isCheckOnly` である。旧実行系で削除される／されないことは未確認。TEI側へその分岐を移植しない。
- `XF100/XF290/XF390` は一部の通常終了で `RETURN_CODE != 99` を条件にcommitする。対して `cancelWithMessage` はRETURN_CODEの設定のみで自動rollbackではない。Script exceptionではrollback経路が重複して呼ばれ得る。実DB・呼出元・Server経路の効果は未観測。ここでのcontext取消／業務取消と旧製品のcancelを同等視しない。
- アーカイブと#940に記載されたinstallerのSHA-256は異なる。ソースのDriver/Server HEADやarchive内jarと、#940で要求する実行baselineは同一とは扱えない。実Provider/旧Runtimeの証拠取得前に対象artifactの固定が必要。

したがって、このReferenceの `PhaseBefore`・`PhaseConsequence`・`PhaseCommit` は*TEI側の合成実行契約候補*であり、X-TEAのBC/AC等イベントを忠実に再現した保証ではない。DB実操作、例外、キャンセル、旧Runtimeの結果コードについてはsourceとactualの比較結果が得られるまでopenとする。
