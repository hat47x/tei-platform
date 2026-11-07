# 休日の結合と暦識別検査を一続きにする確認（2026-11）

## 対象

- 実装: `internal/businesscalendar/holiday_resolve.go`
- 試験: `internal/businesscalendar/holiday_resolve_test.go`
- 対象版: `main` の `6cf510d63` から分岐した`feat/holiday-domain-integration-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -run 'TestHolidayResolve|TestCalendarForPeriod' -count=1 -v
```

## 観測

- `TestHolidayResolveJoinsTheSteps` 成功。暦宣言の解決、休日Dataの結合、暦名による絞り込み、識別の検査を1回の呼出しで行い、`sales.tokyo`の1件だけを返す。
- `TestHolidayResolveReportsAMisspeltIdentity` 成功。**別の暦の行**でも識別が宣言外なら報告される。X-TEAはコード文字列の比較で、その行を黙って飛ばした（`Session.java` L786）。
- `TestHolidayResolveLeavesASingleCalendarDataAlone` 成功。暦識別を持たないDataは従来どおり全行を1暦として読む。
- `TestCalendarForPeriodBuildsTheCalendarThePeriodFollows` 成功。期間の暦から作った暦は自分の休日を除外し、別の暦の休日は除外しない。
- `TestCalendarForPeriodRefusesAPeriodWithoutACalendar` 成功。暦を宣言しない期間は拒否される。
- パッケージ全体は`ok`。

## 読み取れること

- 「期間 → 暦 → 休日Data → 行」の経路が、1つの関数で閉じる。
- 識別の検査は**行全体**に適用されるので、別の暦の行の綴り誤りも報告される。黙って飛ばさない。
- 期間が暦を宣言していない場合は、別の期間の境界を借りずに拒否される。

## 読み取れないこと

- Workspace例での複数暦の利用（例は単一暦のまま）。
- 休日の重複・同一日の複数行の扱い。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。期間から暦・休日を一続きに解決する経路は参照実装で確認済み、例での複数暦利用と実DBは未確認です。
