# 例の標準業務Dataの宣言と結合の確認（2026-11）

## 対象

- 例: `examples/xtea-lowcode/semantic/master/standard.tei`、`semantic/sales/order.tei`の`sales.fiscal`
- 試験: `cmd/tei/xtea_standard_business_test.go`
- 対象版: `main` の `9fec9fa38` から分岐した`feat/example-standard-business-data-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./cmd/tei -run 'TestXTEAExampleDeclaresStandardBusinessRoles' -count=1 -v
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestXTEAExampleDeclaresStandardBusinessRoles` 成功。
  - 税: `master.tax_rate`の`applies_from`・`applies_until`・`category`・`rate`・`rounding`を役割から解決。
  - 為替: `master.exchange_rate`の`currency`・`fiscal_year`・`month_sequence`・`middle`・`buying`・`selling`を解決。
  - 採番: `master.numbering_rule`の`counter`・`digits`・`prefix`・`check`・`overflow`を解決し、`ORD-00042`相当の番号と次の状態42を得る。
  - 暦: `sales.fiscal`が`master.holiday`を名指し、`holiday_on`を役割から解決。年度2026の判定が通る。
- 例の`validate`は`ok`（既存の`UNBOUND_DATASOURCE`警告のみ）。
- `go test ./cmd/tei ./internal/businesscalendar ./internal/taxrate ./internal/currencyrate ./internal/numbering ./internal/validation` 成功。

## 読み取れること

- 例のWorkspaceで、税・為替・採番・休日のDataが**役割**で宣言され、各`Resolve`がそれを読める。列名に依存しない宣言が、例の受理済み意味として成立する。
- 暦が休日Dataを名指すことで、休日の結合が例でも動く。

## 読み取れないこと

- 税率・為替・採番の**行データ**（例には含まない）。参照実装の試験で行を直接与えて確認している。
- Studio画面での標準業務Dataの表示。今の`view`はApplication投影に期間の暦を出すが、税・為替・採番のDataは投影しない。
- 実DB・旧Runtimeでの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。例での宣言と結合は確認済み、行データ・画面投影・実DBは未確認です。
