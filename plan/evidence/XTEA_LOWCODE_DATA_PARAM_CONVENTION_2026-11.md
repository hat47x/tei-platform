# Data型パラメータへ渡す属性の規約の確認（2026-11-10）

## 対象

- 読取り対象: `internal/validation/application_pattern.go`（`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH`）、`internal/hook/calls.go`（`BindInputsFromReads`）、`internal/semantic/model.go`（`Attribute.Role`）
- 試験: `internal/validation/application_pattern_data_param_test.go`
- 対象版: `9037d547a`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

`AC-2`の残る未確認「Data型パラメータへ同一Dataの属性を渡す規約の明文化」に着手しました。

`input <パラメータ名> <属性>` のパラメータ型が**Data**（`sales.order`など）を名指すとき、検証は「その属性が同じDataのものであるか」だけを見ます。規約として何が成り立ち、何が検査されていないかを実測しました。

## 実測した事実

### 1. 検証が要求するのは「同じDataの属性であること」だけ

```text
APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH
  record pattern %s binds command %s input %s of data %s to attribute %s
  that is not an attribute of %s
```

識別子であることは要求しません。

### 2. 識別子でない属性を結んでも検証は通る

`sales.order`型のパラメータ`order`へ`sales.order.status`（`std.text`）を結んだfixtureで、`tei validate`は`ok`。

### 3. 実行すると、パラメータは属性の値をそのまま受け取る

同じfixtureで実行した結果:

```json
"inputs":[{"name":"order","type":"sales.order","attribute":"sales.order.status",
           "label":"status","value":"SHIPPED","provided":true}]
```

`"ready":true`、`"committed":true`。**Data型のパラメータが、状態を表す文字列`SHIPPED`を受け取って実行が成立**しました。Command境界は値の形（在るか・型に合うか）を見ますが、記録の同一性は見ません。

### 4. 既存の例はすべて識別子を結んでいる

| 例 | 結び付け |
| --- | --- |
| `examples/xtea-lowcode/.../order-record.tei` | `input order sales.order.order_id`（パラメータ型`sales.order`） |

`reference/workspace`にもData型パラメータの結び付けは他にありません。

### 5. 「識別子」を宣言する語彙は無い

`semantic.Attribute`は`Role`を持ちますが、値は`holiday-date`・`tax-applies-from`・`tax-value`のような**標準業務の役割**で、識別子を表す役割はありません。識別子は「型が`std.identifier`であること」でしか判別できません。

## 規約として記すこと

**Data型のパラメータは、そのDataの識別子（`std.identifier`型の属性）へ結ぶ。**

理由:

- パラメータがData型でも、**渡るのは値**（1つの文字列）です。Dataそのものは渡りません。したがって「どのレコードを指すか」を表せるのは識別子だけです。
- 状態や金額のような非識別属性を結ぶと、`sales.order`を期待する能力が`SHIPPED`を受け取ります。型の名前が示す意味と、実際に渡る値が食い違います。
- 既存の例はすべてこの形です。

## 検査していないこと（意図的に残す）

**「識別子に限る」検査は入れていません。** 次の理由です。

1. 識別子を宣言する語彙が無く、`std.identifier`型であることへ依存した規則になります。これは型の使い方についての新しい制約で、設計文書が述べていません。
2. 複合キー（2つの属性で1つのレコードを指す）を1つのData型パラメータへ渡す形を、いまの1パラメータ=1属性の対応では表せません。「識別子だけ」と固定すると、この形を先に塞ぎます。
3. `AC-2`の項目は「規約の**明文化**」であり、規則の発明ではありません。

代わりに、現状の挙動を試験で固定しました（`TestDataParameterBoundToAnOrdinaryAttributeIsAccepted`）。試験のコメントに「現在の挙動を述べており、是認ではない」と明記しています。

## 実行

```powershell
go test ./internal/validation -run 'TestDataParameterBoundTo' -count=1
go run ./cmd/tei validate -workspace <fixture>
go run ./cmd/tei view -workspace <fixture> -application order_entry \
  -pattern order_entry.order_record -command sales.submit_order \
  -input order=SHIPPED -input sales.order.status=SHIPPED \
  -execute -allow-capability sales.submit_order -format json
```

## 観測

- `TestDataParameterBoundToTheIdentifierIsAccepted` 成功。識別子を結ぶ形（既存例と同じ）は通る。
- `TestDataParameterBoundToAnOrdinaryAttributeIsAccepted` 成功。非識別属性を結ぶ形も通る（現状の記録）。
- 上記fixtureの`tei validate`は`ok`、実行は`ready:true`・`committed:true`で完了した。
- `internal/validation`の既存試験はすべて`ok`。

## 読み取れること

- Data型パラメータへ結べるのは「同じDataの属性」であり、識別子に限らない。
- 実行時にパラメータが受け取るのは属性の値であり、Dataそのものではない。

## 読み取れないこと

- 非識別属性を結んだときに、能力の側が何をするか（実Pluginが無いため）。参照Executorは値の形だけを見ます。
- 複合キーを1つのData型パラメータへ渡す形の表現方法。
- 実ProviderがData型パラメータをどう解釈するか。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。`AC-2`の「Data型パラメータへ同一Dataの属性を渡す規約の明文化」として、**検査されている範囲と、されていない範囲、実際に渡る値**を記録しました。識別子限定の検査は、意味の設計判断として未着手のまま残します。
