# 月内締めの区分ごとの閉めの確認（2026-11）

## 対象

- 実装: `internal/applicationview/period_scope.go`（`PeriodScope`・`ScopeFor`）、`internal/command/period_scope.go`（`ScopePeriodState`・`scopeClosed`）、`internal/command/invoke.go`の`checkPeriod`、`CommandPeriodRef.ClosingDay`
- 試験: `internal/applicationview/period_scope_test.go`、`internal/command/period_scope_test.go`
- 対象版: `main` の `68cf2a019` から分岐した`feat/period-closing-operations-20261109`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/applicationview -run TestScopeFor -count=1 -v
go test ./internal/command -run 'TestInvokeClosesOneHalf|TestInvokeRefusesASplit|TestInvokeKeepsWholeMonth' -count=1 -v
```

## 観測

- `TestScopeForAWholeMonthPeriod` 成功。締め日の無い期間は期間IDだけを範囲にし、既存の状態格納はそのまま動く。
- `TestScopeForAClosingDayUsesTheBusinessDate` 成功。20日締めで20日は`sales.month#1`、21日は`sales.month#2`になる。
- `TestScopeForAClosingDayRequiresADate` 成功。業務日付が無い・形式違いは拒否される。
- `TestInvokeClosesOneHalfOfASplitMonth` 成功。前半を閉じると前半の日付は`COMMAND_PERIOD_CLOSED`、同じ月の後半は通る。
- `TestInvokeRefusesASplitPeriodWithoutHalfState` 成功。月単位しか知らない状態では`COMMAND_PERIOD_SCOPE_UNAVAILABLE`。月の閉塞を半分の閉塞として読まない。
- `TestInvokeRefusesASplitPeriodWhoseDateIsNotADate` 成功。業務日付の宣言が日付でない場合は`COMMAND_PERIOD_SCOPE_UNKNOWN`。
- `TestInvokeKeepsWholeMonthPeriodsUnchanged` 成功。締め日の無い期間は従来どおり期間IDで問い合わせる。
- パッケージ全体は`ok`（`internal/command`・`internal/applicationview`・`internal/businesscalendar`・`internal/validation`）。

## 読み取れること

- 閉めの単位が**月または月の半分**になり、閉じた半分だけが Command を拒否する。X-TEAは月順序を締め日で動かしていたため、閉じた単位と日付の月が食い違った（`Session.java` L2471–2484）。
- 半分の閉塞を答えられない状態は、月の閉塞で代用せずに拒否する。誤った半分を通したり止めたりしないため。
- 締め日の無い期間は範囲が期間IDのままで、既存の動作を変えない。

## 読み取れないこと

- 閉め操作そのもの（状態を`open`→`closing`→`closed`へ進める Command や利用者操作）。今は状態の問い合わせだけ。
- CLIの`view -execute`で`COMMAND_PERIOD_SCOPE_UNAVAILABLE`を専用コードとして出すこと（今は`COMMAND_FAILED`に入る）。
- Workspace例への締め日宣言と半分の状態ファイル。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。半分ごとの閉塞の判定は参照実装で確認済み、閉め操作の実行と例への追加は未実装です。
