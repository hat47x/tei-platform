# 能力入力の結び付けの型検査の確認（2026-11-10）

## 対象

- 実装: `internal/validation/application_pattern.go`（`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`、`sameType`・`underlyingType`・`bothPrimitive`・`isPrimitive`）
- 試験: `internal/validation/application_pattern_binding_type_test.go`
- 対象版: `8cdec3056`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 見つけた欠陥

interactionの`command <capability> { input <パラメータ名> <属性> }`は、能力のパラメータと属性を結び付けます。検証は

- パラメータ名が能力に存在するか（`UNKNOWN_..._COMMAND_INPUT`）
- 属性がPatternのDataの属性か（`UNKNOWN_..._BINDING_ATTRIBUTE`）
- 同じパラメータを二度結んでいないか（`..._INPUT_DUPLICATE`）

を見ていましたが、**両者の型が合うかは見ていませんでした**。型の情報（パラメータ型・属性型・型エイリアス）はすべて手元にあったのに、比較する場所が無かったためです。

前段（#3887）でフックの`reads`がこの結び付けを通って能力のパラメータへ渡るようになったため、型の食い違いが**実行時の値**として能力へ届く経路ができていました。

## 実測（修正前）

`std.datetime`型の属性`sales.order.ordered_at`を、`sales.order`型のパラメータ`order`へ結んだfixtureで:

- `tei validate` は `ok`（何も報告しない）
- `view -command -execute` は `ok:true`、フックは`executed`、能力は`called:true`

型の食い違いが、宣言の段階でも実行の段階でも止まっていませんでした。

## 変更の要点

- 検証の入口で宣言済みの型（`semantic.Type`）を集め、interactionの検証へ渡します。
- 結び付けごとに、属性型とパラメータ型を比較します。エイリアスは最終的な型までたどって比べるので、`type OrderId from std.identifier`と`std.identifier`は一致します。
- **両側が`std.`の原始型のときだけ拒否します。** 片側でもData（`sales.order`など）を名指す場合は比較しません。`sales.order`型のパラメータをそのDataの識別子属性へ結ぶ形は既存の例（`examples/xtea-lowcode`）にあり、これは型の同一性ではなく「そのDataを指す識別子」という別の対応だからです。
- 型の循環や未定義では、たどれるところまでで比較し、型の層が報告する重複を増やしません。

## 実行

```powershell
go test ./internal/validation ./internal/hook ./internal/command ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/validation
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestValidateBindingAcceptsTheSamePrimitiveType` 成功。同じ原始型同士は通る。
- `TestValidateBindingAcceptsAnAliasOfTheSameType` 成功。エイリアスは最終型で比較される。
- `TestValidateBindingRefusesDifferentPrimitiveTypes` 成功。`std.date`のパラメータと`std.datetime`の属性が`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`で拒否され、診断文が両方の型と属性名を示す。
- `TestValidateBindingLeavesADataParameterAlone`・`TestValidateBindingLeavesADataAttributeAlone` 成功。Dataを名指す側があるものは比較しない。
- **実測（修正後）**: 前掲のfixtureが`tei validate`で`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`を返し、実行前に止まる。
- `examples/xtea-lowcode`は引き続き`ok`（`UNBOUND_DATASOURCE`の警告のみ、終了コード0）。`reference/workspace`も`ok`。
- `internal/validation`・`internal/hook`・`internal/command`・`internal/studiohost`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。

## 読み取れること

- 原始型同士の食い違いは、宣言の段階で拒否される。
- エイリアスは最終型で比較されるので、別名の宣言が不当に拒否されない。
- Dataを名指す結び付けは、型の同一性として比較されない。

## 読み取れないこと

- Dataを名指すパラメータと属性の対応が正しいか（`sales.order`のパラメータに`std.identifier`を渡す形を、どう検証するか）。現在は未検査として残しています。
- 実行時の値変換。型が合っていても、値の正規化・丸めは行いません。
- 実Providerの型体系との対応。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「属性型とパラメータ型の変換規則」のうち、**比較できる場合の拒否規則**を入れた記録です。Data対応の規則は未着手のまま残ります。
