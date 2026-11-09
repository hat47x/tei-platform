# 月内締めのCLI観察と状態源の確認（2026-11）

## 対象

- 実装: `cmd/tei/cli_view.go`の`commandFailure`、`internal/command/period.go`の`ClosedScope`（`MapPeriodState`・`MemoryPeriodState`）
- 試験: `cmd/tei/cli_view_failure_test.go`、`internal/command/period_test.go`、`internal/command/period_scope_test.go`、`internal/periodstate/file_test.go`
- 対象版: `main` の `cd45e05ed` から分岐した`feat/cli-period-scope-codes-20261109`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./cmd/tei -run TestCommandFailure -count=1 -v
go test ./internal/command -run 'TestMapPeriodState|TestMemoryPeriodState|TestInvokeRefusesASplitPeriodWithoutHalfState' -count=1 -v
go test ./internal/periodstate -run TestFileReadsHalfMonthScopes -count=1 -v
```

## 観測

- `TestCommandFailureNamesTheScopeCodes` 成功。`COMMAND_PERIOD_SCOPE_UNAVAILABLE`と`COMMAND_PERIOD_SCOPE_UNKNOWN`がそれぞれ専用コードで返る。従来は`COMMAND_FAILED`に入っていた。
- `TestMapPeriodStateAnswersForHalves` 成功。`sales.period#1`を閉じた状態で、`#1`は閉、`#2`は開。**期間全体の鍵`sales.period`を半分の答えとして読まない**。
- `TestMemoryPeriodStateAnswersForHalfMonths` 成功。参照状態で半分の閉塞を設定でき、閉じた半分だけが拒否される。
- `TestInvokeRefusesASplitPeriodWithoutHalfState` 成功。`ClosedScope`を持たない状態は`COMMAND_PERIOD_SCOPE_UNAVAILABLE`になる。
- `TestFileReadsHalfMonthScopes` 成功。`period-state`のJSONに`sales.month#1`を書けば、同じファイルで半分を閉じられる。
- `go test ./internal/command ./internal/periodstate ./internal/applicationview` 成功。

## 読み取れること

- CLIの利用者が、半分の閉塞を「状態源が答えられない」のか「業務日付が読めない」のか「一般の実行失敗」なのかで区別できる。
- 半分の閉塞は`period-state`のJSONへ`sales.month#1`として書け、既存の期間全体の書き方と混在できる。
- 状態源が半分を答えられない場合は、月の閉塞で代用せずに拒否する。

## 読み取れないこと

- 閉め操作そのもの（状態を進めるCommand・利用者操作）。
- `cmd/tei`の**全件**試験は既定の10分を超えるため、`-timeout 1800s`を明示しないと完走しません（今回の変更とは別の、試験量の増加によるものです）。
- Workspace例での半分の状態ファイルの利用。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。半分ごとの閉塞の宣言とCLIの区別は参照実装で確認済み、閉め操作の実行は未実装です。
