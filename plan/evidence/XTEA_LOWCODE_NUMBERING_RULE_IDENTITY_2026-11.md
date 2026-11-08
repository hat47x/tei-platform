# 採番規則の行識別の確認（2026-11）

## 対象

- 実装: `internal/numbering/rule_binding.go`、`internal/validation`の`numbering-rule`役割
- 試験: `internal/numbering/rule_binding_test.go`
- 対象版: `main` の `bb51bb570` から分岐した`feat/numbering-row-identity-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -run 'TestBindRule|TestRowsFor' -count=1 -v
```

## 観測

- `TestBindRuleReadsTheOptionalRole` 成功。`numbering-rule`の役割がある場合だけ属性を解決し、無い場合は空で成功する。
- `TestRowsForSeparatesRulesInOneData` 成功。1つのDataの`order`行（`ORD-`）と`invoice`行（`INV-`）を、規則名で分けて読む。
- `TestRowsForRefusesAnUnknownOrEmptyRule` 成功。規則名の不在・行の無い規則・規則名の空の行は拒否される。
- `TestRowsForKeepsASingleRuleDataWhole` 成功。役割を持たないDataは全行を返し、`RuleFor`は規則名を報告しない。
- パッケージ全体は`ok`。`internal/validation`も`ok`。

## 読み取れること

- 1つの採番表に複数の規則を置ける。X-TEAは`IDNUMBER`列で規則を識別し、1つの表に全規則を置いていた（`Session.java` L1985–1994）。
- 規則名が無い行は、どの規則の番号か推測せずに拒否される（誤った規則の番号を発行しない）。
- 役割を持たないDataは単一規則のまま動く。

## 読み取れないこと

- `Reserve`の`ruleID`と規則名の照合（`RowsFor`で絞った行を`Bind`に渡す形は呼出し側が組む）。
- Workspace例への複数規則Dataの追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。複数規則の行識別は参照実装で確認済み、`Reserve`との一続きの経路と例への追加は未実装です。
