# 共通生成器の実行境界：Capabilityではなく決定論的Transformへ分離する

この文書は、Issue #1067 / Draft PR #1069で扱う共有生成器候補を、TEIの既存機構へどの境界で接続するかを整理するReferenceです。採用済みArchitectureではありません。

## 結論

現在の`shared/required-fields.mjs`は、外部状態を読まず、書込みも行わず、同じ入力から同じ検証計画を返します。この性質だけを見る限り、最初の接続先は汎用Capability Invocationではなく、`internal/plugin/transform`の決定論的Transformが適切です。

Capabilityは、外部read、Runtime Provider、Commandなど、実行Authorityを伴う操作へ進んだ段階で利用します。純粋な生成処理までCapabilityとして表すと、Provider Resolution、Authority scope、Invocation shape、Conformance Evidenceを不要に背負わせます。

## 現在のJS候補が混ぜている二つの責務

`compileRequiredFields`は、現在は一つの関数で次の二つを行っています。

1. **意味上の入力検査**
   - 契約identityが期待したものか
   - 項目名が許可された形式か
   - identity項目とamount項目が同一でないか
2. **決定論的な実現**
   - 二つの入力項目から検証計画を組み立てる
   - 出力順序を固定する
   - 外部状態に依存せず同じ結果を返す

TEI本体へ接続するときは、この二つを分けます。

```text
Accepted semantic / project configuration
  - exact contract identity
  - typed identity attribute reference
  - typed amount attribute reference
  - distinctness and scope validated
             |
             v
deterministic Transform
  - accepted inputsを検証計画へmaterialize
  - external read/writeなし
  - ambient clock/random/networkなし
             |
             v
derived validation plan
```

項目名の字句検査やdistinctnessをTransform用の正規表現・条件分岐として再実装しません。これらは、入力をtyped referenceとして受理する側の責務です。Transformは受理済み入力から派生物を作るだけにします。

## 既存機構への対応

| #1069の概念 | TEI側の対応候補 |
| --- | --- |
| 共有生成器の論理identity | Transform `SourceIdentity` |
| 共有生成器の版 | `SourceIdentity.Version` |
| 生成契約 | `SourceIdentity.ContractVersion`と上位Semantic contract |
| 生成器Source | `transform.Source`のmaterialized bytes / fingerprint |
| 案件ごとの採用 | Project / Environment側の明示選択と既存Lock |
| Source固定 | `BuildPlan.Sources[].Digest` |
| artifact固定 | Build Artifact Lock / Runtime Plugin Lock |
| 外部副作用 | Transformでは扱わずCapability境界へ送る |

E3で追加した`plugin_lock_integration_test.go`は、採用・Build入力・artifactの固定を確認するため引き続き有効です。ここで扱うTransform分類は、その前段の「共有生成器の実装方式」を明確にするものです。

## 現行Transform IRで不足する最小primitive

現行`deterministic-transform-v1`には、objectの構築、field選択、map、sort、assert、canonical JSON等がありますが、**複数の式から固定長arrayを構築する演算がありません**。

今回必要な出力は概念上、次です。

```json
{
  "format": "tei.reference.validation-plan/v0",
  "rules": [
    { "op": "required-text", "field": "<identityField>" },
    { "op": "nonnegative-safe-integer", "field": "<amountField>" }
  ]
}
```

`OpLiteral`では動的なfield参照を埋め込めず、`OpConstructObject`はarrayを作れません。`OpMapArray`だけで表そうとすると、異なる二つの`op`を割り当てるために条件分岐等を追加する必要があり、今回の要求に対して過剰です。

最小候補は次の一つです。

```text
construct-array(values: Expr[])
```

要求する性質:

- 各要素を宣言順に評価する
- `maxCollection`を超える要素数をSource validationで拒否する
- 各子式を通常のbinding / nesting / step budget検査へ通す
- 出力は既存`ValueArray`と同じ値領域を使う
- clock / random / filesystem / network / process等のAuthorityを追加しない
- 新しい条件式や一般目的制御構文を同時に導入しない

このprimitiveだけで、受理済み二つのfield referenceから検証planを組み立てられます。

## なぜCapabilityへ直接接続しないか

PR #1058のGeneric Invocationは、Source AdapterやContract Queryのような**実行サービス**を扱います。Contract / Provider / artifact / Authority scopeを解決し、opaque requestを実装へ渡す境界です。

今回の共有生成器をそこへ載せると、純粋変換にもProvider解決が必要になり、案件ごとの生成処理を小さく保つという#1067の目的に反します。

将来、生成処理が外部schema取得、DB introspection、remote service参照などを必要とする場合は、純粋Transformからその処理を分離し、必要なreadをCapabilityとして明示的に呼びます。外部readの結果を固定入力としてTransformへ渡す構造にします。

## 次の実証条件

1. #979側で`construct-array`相当の最小primitiveが妥当か確認する。
2. primitiveを追加する場合、既存Transformのresource budgetとSource fingerprint規則を維持する。
3. #1069では、JS共有生成器とTEI Transformの出力をorder / invoice双方でbyteまたは構造的に比較する。
4. 意味検査はTransformへ複製せず、typed field referenceを供給する上位境界のfixtureを用意する。
5. 共有生成器が外部副作用を持たない限り、#951のGeneric Capability Invocationを必須依存にしない。

この分離により、#1069は「共通化できる意味」と「その意味を決定論的に実現する機構」と「外部Authorityを必要とする実行」を混同せずに進められます。
