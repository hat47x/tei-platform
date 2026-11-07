# 税率の適用期間と端数処理の確認（2026-11-07）

## 対象

- 実装: `internal/taxrate/taxrate.go`
- 試験: `internal/taxrate/taxrate_test.go`
- 対象版: `main` の `ccda2179f` から分岐した`feat/tax-rate-reference-20261107`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/taxrate -count=1 -v
```

## 観測

- `TestRateForUsesTheDeclaredSpan` 成功。適用期間の両端を含み、`2023-09-30`は10%、`2023-10-01`は8%、`2026-09-30`は8%、`2026-10-01`は5%。
- `TestRateForRefusesADateNoRowCovers` 成功。どの行も覆わない日付は`ErrInvalidRate`で拒否され、0%として扱われない。
- `TestRateForRefusesOverlappingRows` 成功。2行が同じ日付を覆う場合も拒否され、`Overlaps`が1組を返す。
- `TestAmountRoundsByTheDeclaredRule` 成功。`floor`宣言で1,080×8%は86、-1,080×8%は-87（X-TEAの「負は切り上げ」と同じ結果になる）。
- `TestRoundNamesTheRule` 成功。`floor`・`ceiling`・`half-up`・`half-even`の各規則が同じ値に別の結果を返す。
- `TestSetWithoutARoundingRuleIsRefused` 成功。端数処理の欠落・時間帯の欠落は拒否される。
- `TestRateForUsesTheCategory` 成功。別カテゴリの行は使わず、カテゴリ違いの2行は重なりとしない。
- パッケージ全体は`ok`。

## 読み取れること

- 税率の適用は開始日と終了日の両端で決まり、改正日の前後を試験できる。X-TEAは開始日だけで、終了日を持たなかった（`Session.java` L2315–2334）。
- 端数処理は宣言された規則で決まる。X-TEAは金額の符号で切り捨てと切り上げを切り替えていた（同 L2330–2334）。
- 行の欠落・重なりは、もっともらしい数値ではなく報告になる。

## 読み取れないこと

- 税率Data（意味のDataとしての宣言）との結合。参照実装は行データを入力に取るのみ。
- 税区分の多段（軽減税率と標準税率の併用）、免税の扱い。
- 実DB・旧Runtimeでの税額の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`の税の適用期間と端数処理は参照実装で確認済み、Data宣言との結合と税区分の運用は未実装です。
