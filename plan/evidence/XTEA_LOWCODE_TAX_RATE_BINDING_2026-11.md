# 税率Dataの宣言と行の結合の確認（2026-11）

## 対象

- 実装: `internal/taxrate/binding.go`
- 試験: `internal/taxrate/binding_test.go`
- 対象版: `main` の `87d41e72f` から分岐した`feat/tax-rate-data-binding-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/taxrate -count=1 -v
```

## 観測

- `TestResolveReadsTheDeclaredRoles` 成功。`tax-applies-from`・`tax-applies-until`・`tax-category`・`tax-value`・`tax-rounding`の役割から属性を解決する。列名には依存しない。
- `TestResolveRefusesAnIncompleteDeclaration` 成功。必須の役割が欠けたDataと、未宣言のDataは拒否され、欠けた役割名が報告される。
- `TestBindBuildsRatesFromRows` 成功。保存された行（属性ID→値）からRateを作り、終了日の無い行は開いた端として残り、行自身の端数処理が既定より優先される。
- `TestBindReportsAnUnparsableRow` 成功。数値でない率・解釈できない日付は、行番号と属性名つきで報告され、飛ばされない。
- `TestBindReportsAnUnknownRounding` 成功。未知の端数処理は既定へ丸めず拒否する。
- `TestBindRequiresALocale` 成功。時間帯の無い結合は拒否される。
- パッケージ全体は`ok`（13件）。

## 読み取れること

- 税率表の属性は、列名ではなく**宣言された役割**から決まる。X-TEAは`DTSTART`・`VLTAXRATE`という列名を直接読み（`Session.java` L2315–2334）、表名もシステム変数から取っていた。
- 必須の役割が欠けたDataは、率の0や既定の端数処理で埋めずに拒否される。
- 行の読み取り失敗は行番号つきで報告され、欠番として後段へ流れない。

## 読み取れないこと

- 実際のWorkspaceに税率Dataを置いた場合の検証（新規の役割名を含む`.tei`は未追加）。
- 端数処理の行ごとの上書きが、`RateFor`の適用選択とどう結び付くかの網羅。
- 実DB・旧Runtimeでの税率表の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。税のData宣言との結合は参照実装で確認済み、Workspace例への追加と実DBの実測は未実施です。
