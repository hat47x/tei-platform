# 休日Dataの暦識別値の検査の確認（2026-11）

## 対象

- 実装: `internal/businesscalendar/calendar_domain.go`
- 試験: `internal/businesscalendar/calendar_domain_test.go`
- 対象版: `main` の `ba9009bff` から分岐した`feat/holiday-calendar-value-check-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -run 'TestCalendarDomain' -count=1 -v
```

## 観測

- `TestCalendarDomainReadsTheDeclaredValues` 成功。暦識別属性の型が宣言した値（`sales.tokyo`・`sales.osaka`）を、別名の下地まで辿って読む。
- `TestCalendarDomainLeavesAnOpenTypeOpen` 成功。値を宣言しない型は「閉じた集合」と読まず、`Declared=false`になる。
- `TestCalendarDomainReportsAnUnknownIdentity` 成功。宣言外の`sales.kyoto`と空の識別は拒否され、宣言済みは通る。
- `TestCalendarDomainAcceptsAnyIdentityWhenOpen` 成功。開いた型では識別を検査しない（閉じた集合を捏造しない）。
- `TestCalendarDomainIsEmptyForASingleCalendarData` 成功。暦識別を持たないDataは領域なし・エラーなし。
- パッケージ全体は`ok`。`internal/validation`・`cmd/tei`も`ok`。

## 読み取れること

- 休日Dataの暦識別を、**型の値域**で閉じられる。X-TEAは暦コード文字列と`code;date`の組を比較していたため（`Session.java` L786、L1691）、どの暦も宣言していないコードでも一致し得た。
- 値を宣言しない型は開いた集合として扱い、検査を捏造しない。

## 読み取れないこと

- `HolidayRowsForCalendar`が領域検査を自動で行うか。今は`CalendarDomain`と`Validate`を呼出し側が組合わせる形で、行の読み取りに自動では組み込んでいません。
- Workspace例への暦識別値域の追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。暦識別値の検査は参照実装で確認済み、行読み取りへの自動組込みは未実装です。
