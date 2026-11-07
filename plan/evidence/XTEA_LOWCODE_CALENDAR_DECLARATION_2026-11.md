# 暦の宣言と期間への結合の確認（2026-11-07）

## 対象

- 実装: `internal/semantic/model.go`（`Calendar`、`Period.Calendar`）、`internal/syntax/semantic_calendar.go`、`internal/validation/calendar.go`、`internal/workspace/loader.go`
- 試験: `internal/syntax/semantic_calendar_test.go`、`internal/validation/calendar_test.go`
- 対象版: `99a3367d6`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/syntax -run 'TestSemanticCalendar|TestSemanticPeriodCarriesCalendar' -count=1 -v
go test ./internal/validation -run 'TestValidateCalendar|TestValidatePeriodRejectsA' -count=1 -v
go run ./cmd/tei validate -workspace examples/xtea-lowcode
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

構文（3件）:

- `TestSemanticCalendarParses` 成功。`fiscal-year starts month 4`・`week starts monday`・`business-day excludes sales.holiday`が読める。
- `TestSemanticPeriodCarriesCalendar` 成功。`period`が`calendar sales.fiscal`を持つ。
- `TestSemanticCalendarRejectsUnknownMember` 成功。未知の項目（`fiscal-year ends month 3`）は拒否される。

検証（8件）:

- `TestValidateCalendarAcceptsACompleteDeclaration` 成功。完全な宣言は診断を出さない。
- `TestValidateCalendarRequiresTheFiscalYearStart` 成功。開始月の欠落は`CALENDAR_FISCAL_YEAR_START_REQUIRED`。
- `TestValidateCalendarRejectsAnImpossibleStartMonth` 成功。13月は`CALENDAR_FISCAL_YEAR_START_RANGE`。
- `TestValidateCalendarRequiresTheWeekStart` 成功。週開始の欠落は`CALENDAR_WEEK_START_REQUIRED`。
- `TestValidateCalendarRejectsAnUnknownWeekStart` 成功。水曜は`CALENDAR_WEEK_START_UNSUPPORTED`。
- `TestValidateCalendarRejectsAnUnknownHolidayData` 成功。未宣言のDataは`CALENDAR_UNKNOWN_EXCLUDES_DATA`。
- `TestValidatePeriodRejectsAnUnknownCalendar` 成功。`PERIOD_UNKNOWN_CALENDAR`。
- `TestValidatePeriodRejectsACalendarFromAnotherNamespace` 成功。`PERIOD_CALENDAR_NAMESPACE_MISMATCH`。

Workspace:

- `examples/xtea-lowcode`は`ok`（既存の`UNBOUND_DATASOURCE`警告のみ）。
- `reference/workspace`は`ok`（既存の契約警告のみ）。

## 読み取れること

- 年度開始月・週開始・休日Dataは、システム変数ではなく宣言された意味になった。
- 期間は、従う暦を名前で指し、名前空間をまたぐ参照は拒否される。
- 欠落・範囲外・未対応値は、既定値で埋めずに診断になる。

## 読み取れないこと

- 暦からの日付計算（営業日数・年度の範囲・期間キーの導出）は未実装。
- 税・通貨・採番の宣言は未実装（設計のみ）。
- 実際の業務カレンダーのデータ（休日の実体）は例に含めていない。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`のうち暦の宣言は参照実装で確認済み、日付計算と税・通貨・採番は未実装です。
