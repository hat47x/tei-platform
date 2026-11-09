# 暦からの日付導出の確認（2026-11-07）

## 対象

- 実装: `internal/businesscalendar/calendar.go`、`internal/businesscalendar/resolve.go`
- 試験: `internal/businesscalendar/calendar_test.go`、`internal/businesscalendar/resolve_test.go`
- 対象版: `027be7752`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -count=1 -v
```

## 観測

導出（9件）:

- `TestDaysBetweenNamesTheRule` 成功。`elapsed`・`calendar`は11日、`business`は9日（4/29・5/5の休日を除く）。X-TEAの`countType` 0/2/1に対応する。
- `TestDaysBetweenCountsNegativeSpans` 成功。向きを入れ替えると`-11`。
- `TestDaysBetweenIgnoresTheClockTime` 成功。時刻を足しても日付差は変わらない。
- `TestAddBusinessDaysStopsOnABusinessDay` 成功。休日を飛ばして営業日へ着地し、0は営業日へ寄せる。
- `TestAddBusinessDaysCountsBackwards` 成功。負の向きも同じ規則。
- `TestFiscalYearFollowsTheDeclaredStart` 成功。3月は2026年度、4月は2027年度、年度開始は4/1。
- `TestCalendarWithoutABoundaryIsRefused` 成功。開始月0・13、週開始欠落、時間帯欠落は`ErrInvalidCalendar`。
- `TestDaysBetweenRefusesAnUnnamedRule` 成功。`"1"`のような名前でない規則は拒否。
- `TestHolidaySetReportsTheSetUsed` 成功。休日集合が日付順で読める。

宣言からの解決（4件）:

- `TestResolveReadsTheDeclaration` 成功。開始月・週開始・休日Dataが宣言から読める。
- `TestResolveRefusesAnUndeclaredCalendar` 成功。未宣言の暦は拒否。
- `TestBuildCombinesDeclarationAndHolidays` 成功。宣言と休日から暦を作り、休日が除外される。
- `TestPeriodCalendarRequiresADeclaration` 成功。期間が暦を宣言していなければ拒否。

## 読み取れること

- 日付の数え方は、整数ではなく名前で選ぶ。
- 年度・営業日・休日の扱いは、宣言と休日集合から決まり、実行環境の時計・zone・設定ファイルに依存しない。
- 境界（開始月・週開始・時間帯）が無い暦は、既定値で埋めずに拒否する。

## 読み取れないこと

- 休日の実体（Dataの行）の読み取り。呼出し側が与える範囲のみ確認。
- 週開始の日付計算への効果。X-TEAは休日だけを除外し、週末を除外していなかったため、営業日判定に週開始を使っていない。
- 期間キー（年月）の導出、税・通貨・採番。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`のうち暦の宣言と日付導出は参照実装で確認済み、期間キー・税・通貨・採番は未実装です。
