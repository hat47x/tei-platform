# 標準業務の属性役割語彙の確認（2026-11）

## 対象

- 実装: `internal/validation/validation.go`の`attributeRoles`
- 試験: `internal/validation/semantic_standard_business_role_test.go`
- 対象版: `main` の `12b991ee6` から分岐した`docs/standard-business-role-vocabulary-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/validation -run 'TestValidateStandardBusinessRole' -count=1 -v
```

## 観測

- `TestValidateStandardBusinessRolesAreAccepted` 成功。17個の役割（税5・通貨6・採番5・休日1）すべてが`.tei`で書け、検証で`SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED`が出ない。
- `TestValidateStandardBusinessRoleTypoIsReported` 成功。`tax-valuee`のような綴りの誤りは引き続き拒否される。
- パッケージ全体は`ok`。

## 読み取れること

- 標準業務のDataは、列名ではなく宣言された役割で結合できる。語彙を検証が受理するので、宣言したDataを結合器が読める。
- 語彙の追加は「何でも受理する」ことではない。綴りの誤りは報告される。

## 読み取れないこと

- Workspace例（`examples/xtea-lowcode`）への役割付きDataの追加。
- 役割の型（`std.date`・`std.decimal`・`std.integer`）と役割の対応検査。今は役割名の受理だけで、型の一致は見ていない。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。役割語彙の受理は確認済み、型の対応検査と例への追加は未実装です。
