# 実行時の値変換が要らないことの確認（2026-11-10）

## 対象

- 読取り対象: `internal/validation/application_pattern.go`（`sameType`・`underlyingType`・`bothPrimitive`・`isPrimitive`、`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`）、`internal/hook/calls.go`（`BindInputsFromReads`）
- 試験: `internal/validation/application_pattern_conversion_test.go`
- 対象版: `a1b818ab9`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

`AC-2`の残る未確認「実行時の値変換」を調べました。「型が違う属性とパラメータの間で、値をどう変換するか」という項目です。

## 実測した事実

### 1. 原始型同士の結び付けは、同じ下位原始型のときだけ受理される

`sameType`は両側を`underlyingType`で別名解決してから比較します。

| 結び付け | 結果 | 両側の下位型 |
| --- | --- | --- |
| `std.decimal` → `std.decimal` | 受理 | `std.decimal` / `std.decimal` |
| `sales.order_amount`（`std.decimal`の別名）→ `std.decimal` | 受理 | `std.decimal` / `std.decimal` |
| `std.decimal` → `sales.order_amount`（同上） | 受理 | `std.decimal` / `std.decimal` |
| `std.decimal` → `std.text` | 拒否 | — |
| `std.date` → `std.datetime` | 拒否 | — |
| `std.identifier` → `std.decimal` | 拒否 | — |

**受理される組は、必ず両側が同じ`std.*`原始型に解決されます。** 別名が違っても、最終的に同じ型です。

### 2. 受理された組では、値の字面が両側で同じ

`BindInputsFromReads`は属性の値をそのままパラメータへ渡します。両側が同じ原始型なら、その値の字面はその型の字面であり、Command境界は同じ型に対して値を検証します。**変換すべき差がありません。**

### 3. 変換が必要になる組は、そもそも受理されない

`std.decimal`の値を`std.text`のパラメータへ渡す、`std.date`を`std.datetime`へ渡す、といった組は`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`で拒否されます。**変換を必要とする結び付けが検証を通りません。**

### 4. Data型パラメータは変換ではなく素通し

前段（[Data型パラメータへ渡す属性の規約](XTEA_LOWCODE_DATA_PARAM_CONVENTION_2026-11.md)）で記録したとおり、パラメータ型がDataを名指す場合は「同じDataの属性か」だけを見て、値はそのまま渡ります。ここには変換の規則がありません（記録の同一性を見ない、という既知の限界です）。

## 結論

**「実行時の値変換」は、実装されていない機能ではなく、受理される結び付けに存在しない処理です。**

- 原始型同士: 受理条件が「同じ下位原始型」なので、変換すべき差が生じない。
- Data型: 変換の規則を定義しておらず、値は素通し（別項で記録済み）。

変換が必要になるのは**異なる原始型を結んだ場合**ですが、その場合は検証が先に拒否します。したがって変換層を足しても、通る経路がありません。足せば「宣言は在るが実行では何もしない」種類の実装になります。

## 変換層を足さなかった判断

入れていません。理由:

1. 通る経路が無い（上記）。
2. 異なる原始型の結び付けを許すかどうかは**意味の設計判断**です（例: `std.date`を`std.datetime`として受けてよいか）。実装の都合で先に決めません。
3. 将来許す場合に必要になるのは「変換規則の宣言」であり、暗黙の変換ではありません。`AC-3`（項目の意味の役割と表示の指定を分ける）と関わります。

## 実行

```powershell
go test ./internal/validation -run 'TestAcceptedPrimitiveBindingsShareTheSameUnderlyingType|TestDifferentPrimitivesAreRefusedSoNoConversionIsReachable' -count=1
go test ./internal/validation ./internal/hook -count=1
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestAcceptedPrimitiveBindingsShareTheSameUnderlyingType` 成功（同じ原始型・別名→原始型・原始型→別名の3件）。
- `TestDifferentPrimitivesAreRefusedSoNoConversionIsReachable` 成功（`decimal→text`・`date→datetime`・`identifier→decimal`の3件を拒否）。
- `internal/validation`・`internal/hook`の既存試験は`ok`。`reference/workspace`は`ok`。

## 読み取れること

- 受理される原始型の結び付けは、両側が同じ下位原始型であり、値の変換を要さない。
- 変換を要する異なる原始型の結び付けは、検証が拒否する。

## 読み取れないこと

- 異なる原始型の結び付けを**許すべきか**（意味の設計判断）。未決定です。
- 許す場合に必要な変換規則の宣言方法。
- 実Providerが独自の型解釈を持つ場合の扱い。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。`AC-2`の「実行時の値変換」を、**実装漏れではなく受理条件から生じない処理**として記録したものです。異なる原始型を許すかどうかは、意味の設計判断として残します。
