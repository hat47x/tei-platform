# 月内締めの期間化の確認（2026-11）

## 対象

- 実装: `internal/businesscalendar/closing.go`、`internal/semantic/model.go`の`Period.ClosingDay`、`internal/syntax/parser.go`の`closes on day N`、`internal/validation/calendar.go`の締め日検査、`internal/applicationview`と`view`の投影
- 試験: `internal/businesscalendar/closing_test.go`、`internal/validation/closing_period_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `main` の `5d5d5de6b` から分岐した`feat/monthly-closing-period-20261109`
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 実行

```powershell
go test ./internal/businesscalendar -run 'TestClosingPeriod|TestMonthPeriodKey' -count=1 -v
go test ./internal/validation -run 'TestClosingDay' -count=1 -v
node --test cmd/tei-studio/test/client.test.mjs cmd/tei-studio/test/preview-scope.test.mjs cmd/tei-studio/test/view-format.test.mjs
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestClosingPeriodKeepsTheMonth` 成功。20日締めで20日と21日は**同じ年度・同じ月順序・同じ暦月**に属し、区分だけが変わる。
- `TestClosingPeriodSeparatesTheHalves` 成功。同じ月の前半は`2027-02-1`、後半は`2027-02-2`となり、閉めの単位として区別できる。
- `TestClosingPeriodRefusesAnImpossibleDay` 成功。0日・29〜31日は拒否される。
- `TestMonthPeriodKeyHasNoSegment` 成功。締め日を宣言しない期間は`2027-02`のままで、既存の宣言は変わらない。
- `TestClosingDayParsesAndValidates` 成功。`closes on day 20`が読まれ、暦があれば診断なし。
- `TestClosingDayRequiresACalendar` 成功。暦の無い締め日は`PERIOD_CLOSING_REQUIRES_CALENDAR`。
- `TestClosingDayRefusesADayThatDoesNotExistEveryMonth` 成功。29〜31日は`PERIOD_CLOSING_DAY_RANGE`。
- Studioの`view-format`試験は8件すべて成功。「期間」表に「締め日」列が加わり、`closingDay: 20`が`20日`と表示される。
- `view -application`のテキスト出力は、締め日がある期間に`period <id> closes on day N`を出す。
- `validate`は`examples/xtea-lowcode`・`reference/workspace`とも`ok`（既存警告のみ）。

## 読み取れること

- 月内締めは、月を動かさずに**独立した区分**として宣言できる。X-TEAは締め日を過ぎた日付を翌月の月順序へ移し、年度も変えていた（`getMSeqOfDate` L2471–2484、`getFYearOfDate` L2440–2507）。
- 締め日の無い期間の鍵は従来と同じ形式なので、既存の宣言は影響を受けない。
- 締め日は1..28に限る。29〜31日は2月に存在せず、毎月ある境界にならないため。

## 読み取れないこと

- 締め日ごとの実際の閉め操作（状態遷移・`closed-rejects`との組合せ）。
- 締め日を週・四半期など月以外の単位へ広げる形。
- Workspace例への締め日宣言の追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。月内締めの宣言と区分の導出は参照実装で確認済み、閉め操作の実装と例への追加は未実施です。
