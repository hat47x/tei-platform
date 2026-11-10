# Command → effectcommit の四つの接合・結果資格反例（限定Go Characterization）

- **状態**：テスト専用・合成の現行挙動記録。安全な実装の受入や正式業務意味の証明ではない。
- **対象**：`internal/security/effectcommit/commandadapter/executor_characterization_test.go`・`executor_result_characterization_test.go`。既存の `adapterWorld()`／`request()`／`grant()` と本物の `command.Invoker`・`commandadapter.Executor`・`effectcommit.Runner` を通す。Providerと外側UnitOfWorkだけを合成とする。
- **設計との接点**：[Macro設計PR #2602](https://github.com/hat47x/tei-platform/pull/2602) 第16章／M3-B・M3-C・M3-D。[#2598](https://github.com/hat47x/tei-platform/issues/2598)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)のOwnerとP0は変えない。

## 対象・Tx境界についての二つの独立Expectedと現在の観測

| ID | 独立Expected／入力 | テストが現在記録するもの | 証明しないもの |
| --- | --- | --- | --- |
| **MC-B01** 操作対象不一致 | Command Input `order=ORD-1`、呼出側が明示したPayload `{"order":"ORD-2"}`。両者は別の対象 | 明示Payloadが現Adapterを通じてRunnerの合成Transactionへ渡り、合成Commitに至ること。**不一致を事前拒否した成功試験ではない** | OwnerがORD-2を実際に更新したこと、正しいResourceレベルの認可、意図→実Targetの適合 |
| **MC-C01** 二つのCommit | 内側`effectcommit`合成Transactionは成功し、外側`command.UnitOfWork.Commit`だけを失敗させる | Commandは`COMMAND_COMMIT_FAILED`を返す一方、合成内側TransactionのCommit済み記録は残る。外側Rollbackで内側Commitをなかったことにはできない | 実PostgreSQLで二つのTxが分裂すること、内側の永続化、実Receipt/ACK欠落の再現 |

ここでの「PASS」は**反例が現行配線で観測できる**ことを意味する。特にMC-B01では、対象不一致の受入を製品の望ましい動作として固定してはいけない。Owner側で安全な検証が実装されたら、**不一致を拒否する安全性試験に置換**する。

## 再実行

~~~sh
go test ./internal/security/effectcommit/commandadapter -run TestCharacterization -count=1 -v
~~~

限定CI `.github/workflows/macro-command-boundary-characterization.yml` はPRのexact HEADをcheckoutし、`go.mod`のGo版、`gofmt -l`、上記Go試験、worktree不変を別々に確認する。**PR作成やテストSource保存だけで、CIが成功したとは記録しない。**

## Replay/unknownの別資格（MC-D01/D02）

[submit側の既存共通Caller](https://github.com/hat47x/tei-platform/blob/main/internal/security/effectcommit/submit/submit.go)は `Committed`／`Replayed`／`NotCommitted`／`Uncertain`／`NotRecorded` 等を区別する**未採用A2参照**を持つ。他方、現在の `commandadapter.Executor` は `committed/replayed` を両方 `command.Result{}` と `nil` に写す。これらは別の既存Caller入口であり、既存 `submit.Outcome` をただちに製品の正式 `Command.Result` に昇格させない。

| ID | 独立Expectedと合成観測 | 合格と呼ばない範囲 |
| --- | --- | --- |
| **MC-D01** 初回対Replay | 同じOperationIDと同じInputsを2回実行。Fake Provider内では最初のEffect＋Receiptが1回保存され、2回目はReceipt一致によるReplayで追加Writeしない。CommandのResultは**両方空値**。 | UI/CLIに初回確定・再送・Receiptが既に伝達されたとは言えない。Providerの実DB冪等性・現Grantの実適合は証明外 |
| **MC-D02** Commit uncertain | Fake TransactionのCommitがエラー。Commandの外側では `COMMAND_EXECUTION_FAILED` に包まれるが、`errors.As` で `CommitUncertainError` は保持される。 | DBに書き込まれた／Rollbackされた／Receiptがない、といった実結果は分からない。新しいOperationIDを発行した再Submitは許されない |

**設計上の停止線**：内部Goの型付きCauseを保持することと、外部Studio/API/CLIへ正しい非実行Dispositionが表現・伝達されることを分ける。未知Commit時は**同OperationID・同PayloadのRequestで権限適格性を確認してLookup**し、Lookupが `not-recorded` でも対象Ownerの実Effect状態の保証がなければ新Effectを自動実行しない。実Host/Providerと独立Owner read/Receipt照合は[#1174](https://github.com/hat47x/tei-platform/issues/1174)／[#1119](https://github.com/hat47x/tei-platform/issues/1119)へ委譲する。

この追加は**Test Characterizationのみ**であり、初回/Replay/unknownを伝達する正式なResult schemaを採用・実装したものではない。[Focused Actions #41632111078](https://github.com/hat47x/tei-platform/actions/runs/41632111078)では直前head `c9927f247839106c28811e14abba16a53bd38e98` のGo1.26.10/Linux実試験 **4/4 PASS**、`gofmt`・worktree cleanを確認。ただし、その後のWorkflow/README変更HEADに同じ合格を自動転記しない。

## 次の実装所有と中止線

- MC-B01に対しては、呼出側で`OperationID`・`Grant`・`ExpectedVersion`・`Payload`を同一Operationとして確定させた根拠が必要。合法な関連明細更新もあるので、単純な文字列一致で全件を拒否せず、Ownerが受理したTarget変換とProvider側の実書込み対象を結ぶ。
- MC-C01では、外側UnitOfWorkとRunner内側Transactionが**同一物理Tx**に参加している根拠がなければ一つのAtomic Commitと主張しない。Commit uncertainでは同じOperationIDの既存Receipt照会を優先し、異なるIDで無確認再実行しない。
- 本PRの4件はすべて合成Providerであり、現在Grant発行/失効、**Command Adapterに接続した実PostgreSQL**、BusinessResult、独立SQL Observer、Effectlog Journal/補償、Studio/SHO/EKI Consumer、OS隔離、Owner意味受理は未実行。別Callerでの実PostgreSQL Reference（#1174）は既存の限定証拠として区別する。P0-a／P0-b・Gate A/B/Cは変更しない。
