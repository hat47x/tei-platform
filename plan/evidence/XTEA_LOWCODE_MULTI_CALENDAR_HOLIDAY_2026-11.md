# 複数暦の休日Dataの確認（2026-11）

## 対象

- 実装: `internal/businesscalendar/holiday_binding.go`（`RoleHolidayCalendar`・`HolidayRowsForCalendar`）、`internal/validation`の`holiday-calendar`役割
- 試験: `internal/businesscalendar/holiday_multi_calendar_test.go`
- 対象版: `main` の `e1b17c24a` から分岐した`feat/calendar-multiple-calendars-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -run 'TestBindHolidaysFindsTheCalendarIdentity|TestHolidayRowsForCalendar|TestHolidayRowsRefusesAnUnnamedCalendar' -count=1 -v
```

## 観測

- `TestBindHolidaysFindsTheCalendarIdentity` 成功。休日Dataが`holiday-calendar`の役割を宣言しているとき、結合が暦の識別属性を解決する。
- `TestHolidayRowsForCalendarTakesOnlyItsOwnHolidays` 成功。2つの暦が1つのDataを共有し、`sales.tokyo`は2件、`sales.osaka`は1件だけを取る。
- `TestHolidayRowsRefusesAnUnnamedCalendar` 成功。共有Dataを暦名なしで読むと拒否され、空の暦名も拒否される。
- `TestHolidayRowsForCalendarIgnoresTheFilterForASingleCalendar` 成功。暦の識別を持たないDataは従来どおり全行が1つの暦に属し、`HolidayRows`も使える。
- パッケージ全体は`ok`。`internal/validation`・`internal/applicationview`・`cmd/tei`も`ok`。

## 読み取れること

- 複数の暦が1つの休日Dataを共有できる。X-TEAは`KBCALENDAR`の文字列で同じ表を切り替えていた（`Session.java` L786、L1691）。
- 暦名を渡さない読み取りは拒否されるので、別の暦の休日を黙って取り込まない。
- 暦の識別を持たないDataは、従来の単一暦の意味のまま動く。

## 読み取れないこと

- 暦名と暦宣言の照合（`sales.tokyo`という行の値が宣言済みの暦か）。今は行の値をそのまま比較する。
- Workspace例への複数暦の追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。複数暦の休日結合は参照実装で確認済み、行の値と暦宣言の照合は未実装です。
