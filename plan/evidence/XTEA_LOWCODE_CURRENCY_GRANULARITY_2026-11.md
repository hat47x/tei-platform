# 為替の粒度の宣言語彙の確認（2026-11）

## 対象

- 実装: `internal/currencyrate/granularity.go`、`internal/validation`の`currency-rate-granularity`役割
- 試験: `internal/currencyrate/granularity_test.go`
- 対象版: `main` の `ddcef7414` から分岐した`feat/currency-granularity-declaration-20261110`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -run 'TestResolveGranularity|TestGranularityRows|TestCheckDeclared|TestDeclaredGranularity' -count=1 -v
go test ./internal/validation ./internal/currencyrate -count=1
```

## 観測

- `TestResolveGranularityReadsTheOptionalRole` 成功。`currency-rate-granularity`の役割がある場合だけ属性を解決し、無い場合は空で成功する。未宣言のDataは拒否。
- `TestGranularityRowsReadsTheDeclaredValue` 成功。行の`monthly`を読み、役割の無いDataは宣言なしを返す。
- `TestGranularityRowsRejectsAnUnknownOrDisagreeingValue` 成功。`weekly`のような未知の値、1つの表で`monthly`と`annual`が混ざる行、役割を宣言したDataの空の値はすべて拒否される。
- `TestCheckDeclaredComparesTheDeclarationWithTheRows` 成功。宣言と行が矛盾する場合（宣言`annual`・行`monthly`）は報告される。
- `TestDeclaredGranularityMakesAnEmptyTableReadable` 成功。行が無い表でも役割は宣言から読める。
- パッケージ全体は`ok`。`internal/validation`も`ok`。

## 読み取れること

- 粒度を**宣言**として持てる。行からの推定だけでは、空の表（まだ行が無い）と、行がすべて年次の表を区別できませんでした。
- 宣言と行が矛盾する場合は報告される。宣言を`annual`、行を月順序1..12にすると、行を信じる読み手と宣言を信じる読み手で解釈が分かれるためです。
- X-TEAは2つの表をシステム変数の名前で指していた（`Session.java` L2253–2313）ため、名前が「どちらを読むか」は示しても「何が入っているか」は示しませんでした。

## 読み取れないこと

- `ResolveTable`が役割の宣言を自動で読むこと（今は`ResolveGranularity`と`CheckDeclared`を呼出し側が組む）。
- 3表以上（日次・四半期など）の語彙。
- Workspace例への粒度宣言の追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。粒度の宣言語彙は参照実装で確認済み、`ResolveTable`への自動組込みと例への追加は未実装です。
