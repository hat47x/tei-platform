# 為替フォールバックの宣言語彙の確認（2026-11）

## 対象

- 実装: `internal/currencyrate/fallback_declaration.go`、`internal/validation`の`currency-rate-fallback`役割
- 試験: `internal/currencyrate/fallback_declaration_test.go`
- 対象版: `main` の `fc49a70f4` から分岐した`feat/currency-fallback-role-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -run 'TestResolveFallbackReadsTheOptionalRole|TestFallbackRows' -count=1 -v
go test ./internal/validation -run 'TestValidateStandardBusinessRole|TestValidateRoleTypeAcceptsEveryDeclaredRolePair' -count=1
```

## 観測

- `TestResolveFallbackReadsTheOptionalRole` 成功。`currency-rate-fallback`の役割がある場合だけ属性を解決し、無い場合は空で成功する。未宣言のDataは拒否。
- `TestFallbackRowsReadsTheRule` 成功。行の`annual`を読み、役割を持たないDataは既定`none`になる。
- `TestFallbackRowsRejectsAnUnknownRule` 成功。`monthly`のような未知の規則は拒否される。
- `TestFallbackRowsRejectsDisagreeingRows` 成功。1つのDataの行が`annual`と`none`に分かれると拒否される（行順に依存させない）。
- `TestFallbackRowsRequiresAValueWhenTheRoleIsDeclared` 成功。役割を宣言したDataの行が空なら拒否される。
- パッケージ全体は`ok`。`internal/validation`と`cmd/tei`も`ok`。

## 読み取れること

- フォールバック規則は、行の値として宣言される。X-TEAは月次率0の比較で規則を決めていた（`Session.java` L2296–2298）ため、モデルから規則が見えなかった。
- 役割を持たないDataは`none`（フォールバックしない）として読める。宣言の欠落を「暗黙の`annual`」にしない。
- 1つのDataに2つの規則を混ぜることは拒否される。

## 読み取れないこと

- Workspace例の為替Dataへの`currency-rate-fallback`追加。
- 月次と年次を別Dataに分ける場合の結合（`Fallback`は`Set`を組む呼出し側が設定）。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。フォールバック規則の宣言語彙は参照実装で確認済み、例への追加と別Data構成は未実装です。
