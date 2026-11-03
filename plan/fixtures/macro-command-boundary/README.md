# Command → effectcommit の二つの接合負例（限定Go Characterization）

- **状態**：テスト専用・合成の現行挙動記録。安全な実装の受入や正式業務意味の証明ではない。
- **対象**：`internal/security/effectcommit/commandadapter/executor_characterization_test.go`。既存の `adapterWorld()`／`request()`／`grant()` と本物の `command.Invoker`・`commandadapter.Executor`・`effectcommit.Runner` を通す。Providerと外側UnitOfWorkだけを合成とする。
- **設計との接点**：[Macro設計PR #2602](https://github.com/hat47x/tei-platform/pull/2602) 第16章／M3-B・M3-C・M3-D。[#2598](https://github.com/hat47x/tei-platform/issues/2598)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)のOwnerとP0は変えない。

## 二つの独立Expectedと現在の観測

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

## 次の実装所有と中止線

- MC-B01に対しては、呼出側で`OperationID`・`Grant`・`ExpectedVersion`・`Payload`を同一Operationとして確定させた根拠が必要。合法な関連明細更新もあるので、単純な文字列一致で全件を拒否せず、Ownerが受理したTarget変換とProvider側の実書込み対象を結ぶ。
- MC-C01では、外側UnitOfWorkとRunner内側Transactionが**同一物理Tx**に参加している根拠がなければ一つのAtomic Commitと主張しない。Commit uncertainでは同じOperationIDの既存Receipt照会を優先し、異なるIDで無確認再実行しない。
- この2件は合成Providerのみ。現在Grant発行/失効、実PostgreSQL、BusinessResult、独立SQL Observer、Effectlog Journal/補償、Studio/SHO/EKI Consumer、OS隔離、Owner意味受理は未実行。P0-a／P0-b・Gate A/B/Cは変更しない。
