# 通貨換算の率の種類と方向の確認（2026-11-07）

## 対象

- 実装: `internal/currencyrate/currencyrate.go`
- 試験: `internal/currencyrate/currencyrate_test.go`
- 対象版: `main` の `388412958` から分岐した`feat/currency-rate-reference-20261107`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -count=1 -v
```

## 観測

- `TestRateForSelectsTheDeclaredKind` 成功。中値150.25・買値148.10・売値152.40を、宣言した種類で選ぶ。
- `TestSystemCurrencyConvertsAtOne` 成功。システム通貨は率1。
- `TestRateForRefusesAMissingRow` 成功。行の無い期間は`ErrInvalidRate`で拒否され、年次表への暗黙の切替をしない。
- `TestRateForRefusesTwoRowsForOnePeriod` 成功。同じ期間に2行ある場合は拒否。
- `TestConvertUsesTheDeclaredDirection` 成功。外貨→自国は100×150.25=15025、自国→外貨は15025÷150.25=100。
- `TestConvertRefusesAZeroRate` 成功。率0は、種類がいずれでも拒否される。
- `TestSetWithoutADeclaredRuleIsRefused` 成功。システム通貨・種類・方向・端数処理のいずれの欠落も拒否。
- `TestMissingPeriodsNamesTheGaps` 成功。行の無い期間を列挙する（2026年度の月順序2）。
- パッケージ全体は`ok`。

## 読み取れること

- 率の種類（中値・買値・売値）は宣言で選ぶ。X-TEAは`TTB`/`TTS`以外を中値として扱った（`Session.java` L2265–2277）。
- 換算の方向は宣言である。X-TEAは呼出し側の使い方で決まった。
- 行の欠落・重複・率0は報告になる。X-TEAは月次率が0のとき年次表へ切り替え（L2296–2298）、「行が無い」と「率が0」を区別できなかった。

## 読み取れないこと

- 為替Data（意味のDataとしての宣言）との結合。参照実装は行データを入力に取るのみ。
- 年次表と月次表の優先順位の宣言。参照実装は月次相当の行だけを扱い、呼出し側が期間を渡す。
- 実DB・旧Runtimeでの換算結果の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`の通貨換算の宣言は参照実装で確認済み、Data宣言との結合と年次・月次の優先順位は未実装です。
