# 役割と型の対応検査の確認（2026-11）

## 対象

- 実装: `internal/validation/attribute_role_type.go`、`internal/validation/validation.go`の呼出し
- 試験: `internal/validation/attribute_role_type_test.go`
- 対象版: `main` の `392af6ea9` から分岐した`feat/role-type-check-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/validation -run 'TestValidateRoleType' -count=1 -v
go run ./cmd/tei validate -workspace examples/xtea-lowcode
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestValidateRoleTypeAcceptsMatchingPrimitive` 成功。`tax-applies-from`＋`std.date`、`tax-value`＋`std.decimal`は診断なし。
- `TestValidateRoleTypeReportsADateOnADecimal` 成功。`tax-value`＋`std.date`は`SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH`になり、メッセージとSubjectが残る。
- `TestValidateRoleTypeFollowsAnAlias` 成功。`tax.rate_value`（`from std.decimal`）は通る。
- `TestValidateRoleTypeRefusesAnAliasToTheWrongPrimitive` 成功。`tax.rate_date`（`from std.date`）を`tax-value`に使うと拒否される。
- `TestValidateRoleTypeAcceptsAnAnyTypeRole` 成功。`tax-category`・`tax-rounding`は型を問わない。
- `TestValidateRoleTypeLeavesAnUnknownTypeToItsOwnCheck` 成功。未知の型は`UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`だけになり、二重報告しない。
- `TestValidateRoleTypeAcceptsEveryDeclaredRolePair` 成功。役割語彙と要求プリミティブの対応表が矛盾しない。
- パッケージ全体は`ok`。`examples/xtea-lowcode`・`reference/workspace`の`validate`は既存警告のみで`ok`。

## 読み取れること

- 役割が要求する型（日付・小数・整数）を、宣言の段階で検査できる。X-TEAは列名で読み（`Session.java` L2315–2334等）、型の食い違いは数値がおかしいときまで分からなかった。
- 別名（`type X from Y`）は下地のプリミティブまで辿って判定するので、宣言済みの別名を使うDataは不当に拒否されない。
- 型が解決できない場合は、型の未定義だけで報告し、役割の不一致を重ねて報告しない。

## 読み取れないこと

- Workspace例（`examples/xtea-lowcode`）へ役割付きDataを追加する作業。
- 役割の要求が「どのプリミティブか」の妥当性（`tax-value`に`std.money`を許すか等）。今は`decimal`だけを要求する。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。役割と型の対応検査は参照実装で確認済み、例への追加と要求プリミティブの拡張は未実装です。
