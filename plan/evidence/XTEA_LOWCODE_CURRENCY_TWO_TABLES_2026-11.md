# 為替の月次・年次を別Dataに分ける構成の確認（2026-11）

## 対象

- 実装: `internal/currencyrate/tables.go`（`Table`・`Granularity`・`ResolveTable`・`Pair`・`NewPair`・`Pair.RateFor`）、`internal/currencyrate/binding.go`の月順序の検査
- 試験: `internal/currencyrate/tables_test.go`、`internal/currencyrate/binding_test.go`
- 対象版: `main` の `113e1e6f6` から分岐した`feat/currency-two-data-20261110`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -run 'TestResolveTable|TestNewPair|TestPairRateFor' -count=1 -v
go test ./internal/currencyrate -count=1
```

## 観測

### 見つけた不具合と修正

`AnnualMonthSequence = 0`を宣言していましたが、`Bind`は月順序を**1..12に限って**拒否していました。つまり**年次行はどの構成でも結合できず**、`FallbackAnnual`は行を読めませんでした。

- 原因: 月順序の検査が`0`を弾いていた。前ラウンドの試験は`TestBindReportsAnImpossibleMonthSequence`で`0`を**不正値として固定**していたため、矛盾が固定されていました。
- 修正: `AnnualMonthSequence`だけを1..12の外として受理。検査の試験は`-1`と`13`へ変更（`0`は年次標識として意図的に受理）。

### 別Dataの構成

- `Table`（宣言＋粒度＋行）と`ResolveTable`（宣言・結合・粒度判定を1回で）。
- `Granularity`（`monthly`・`annual`）は**行から読む**。1つの表が両方を混ぜる場合は拒否（年がどちらの行を指すか決められない）。
- `Pair`（月次表＋年次表＋宣言された規則）と`NewPair`。**逆順は拒否**（フォールバックの意味は「欠けた月は年を使う」）。
- `Pair.RateFor`:月次行が優先。`annual`宣言時だけ年次行へ。`none`では欠落を報告し、年次行で埋めない。

### 試験（8件）

| 試験 | 内容 |
| --- | --- |
| `TestResolveTableReadsTheGranularity` | 行から`monthly`/`annual`を判定 |
| `TestResolveTableRefusesMixedRows` | 混在・空を拒否 |
| `TestNewPairRefusesTheWrongOrder` | 逆順を拒否 |
| `TestPairRateForUsesTheMonthlyRowFirst` | 月次優先、欠けた月は年次 |
| `TestPairRateForReportsAMissingMonthWithoutFallback` | `none`では欠落を報告 |
| `TestPairRateForKeepsTheSystemCurrencyAtOne` | システム通貨は率1 |
| `TestPairRateForReportsAMissingAnnualRow` | 年次行も無ければ報告 |

## 読み取れること

- X-TEAの月次表・年次表（別名のシステム変数で指す2つの表、`Session.java` L2253–2313）を、**別Dataとして宣言**できる。どちらが月次かは行が、どちらが年次かは行が示します。
- 粒度の混在と表の逆順は拒否されるので、「年が月の行を指す」取り違えが起きません。
- `none`では年次行で埋めないため、X-TEAの「0なら年次へ」という制御値が残りません。

## 読み取れないこと

- `.tei`の宣言から粒度を語彙として導くこと（今は行から判定）。
- 3つ以上の表（日次・四半期など）を組む構成。
- Workspace例への2Data構成の追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。2Data構成の結合と年次標識の受理は参照実装で確認済み、宣言語彙と3表以上の構成は未実装です。
