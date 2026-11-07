# 休日Dataの宣言と行の結合の確認（2026-11）

## 対象

- 実装: `internal/businesscalendar/holiday_binding.go`
- 試験: `internal/businesscalendar/holiday_binding_test.go`
- 対象版: `main` の `86ad44558` から分岐した`feat/holiday-data-binding-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -count=1 -v
```

## 観測

- `TestBindHolidaysFindsTheDeclaredDate` 成功。暦が休日Data（`sales.holiday`）を名指し、Dataが`holiday-date`の役割で日付属性を名指す。
- `TestBindHolidaysRefusesAnUnroledDate` 成功。日付の役割が無いDataは拒否される（列を推測しない）。
- `TestBindHolidaysRefusesAnUndeclaredData` 成功。暦が名指すDataが存在しなければ拒否される。
- `TestHolidayRowsReadsTheDates` 成功。行から日付を作り、暦に渡すと`HolidaySet`が日付順になり、その日が`Excludes`で真、非休日は`IsBusinessDay`で真になる。
- `TestHolidayRowsReportABadDate` 成功。空の日付・別形式（`02/05/2026`）は行番号と属性名つきで拒否され、時間帯の無い結合も拒否される。
- `TestCalendarWithoutHolidayDataHasNoHolidays` 成功。休日Dataを宣言しない暦は休日を持たない（エラーではない）。
- パッケージ全体は`ok`（24件）。

## 読み取れること

- 休日の日付は、列名ではなく**宣言された役割**から決まる。X-TEAは暦コード＋日付文字列の組を起動時に読み込み（`Session.java` L786、L1691、L1882）、日付の書式は表に任されていた。
- 解釈できない日付は、行と属性を名指して報告される。X-TEAはそのままリストへ入れ、除外されない休日が黙って残り得た。
- 休日Dataを宣言しない暦は、休日なしとして読める。既定の表を勝手に使わない。

## 読み取れないこと

- Workspace例（`.tei`）への`holiday-date`役割の追加。
- 暦ごとの休日（`KBCALENDAR`の複数暦）を1つのDataで表す場合の、暦の識別属性との結合。
- 実DB・旧Runtimeでの休日表の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。休日のData宣言との結合は参照実装で確認済み、例への追加と複数暦の識別は未実装です。
