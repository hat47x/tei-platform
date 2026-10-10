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

## Transform拡張の検討結果：現時点では追加しない

当初は、二つの項目参照から固定長の規則配列を構築するために、`construct-array(values: Expr[])`相当の演算を候補にした。現行Transform IRには同じ形のprimitiveが無いため、純粋な表現能力だけを見れば自然な追加候補である。

その後、実際のTEI業務検証経路を確認すると、受注アプリケーションでは別の検証計画を生成する必要がなかった。

```text
Application uses validation
        ↓
driver.ResolveApplicationValidations
        ↓
semantic.Validation
        ↓
validationeval.Evaluate
        ↓
既存Target / Web応答
```

正本の規則を既存Driver/Evaluatorが直接消費している。したがって、現在の受注Consumerを理由に規則配列を新しく生成したり、そのためにTransform IRを拡張したりしない。

既存`map-array`へ受理済み規則配列を渡せば派生JSONへ投影できることも検討したが、その派生JSONを必要とするConsumer自体が無い。使われない生成物のためのSource・契約・試験を保持すると、第二正本化と保守費を増やす。

このため、#1069で一時的に置いたTransform能力試験は削除し、#979へ渡した`construct-array`要求も**このConsumerからの要求としては取り下げる**。別の具体的Consumerが、既存primitiveでは表現できない固定長配列構築を本当に必要とした場合に、そのConsumerの受入条件を根拠として再提案する。

判断規則は次のとおり。

1. 正本を既存機構が直接消費できるなら、派生生成器を追加しない。
2. 派生形式が必要なら、まず具体的Consumerとその受入条件を固定する。
3. 既存Transformで表現できる場合は新primitiveを増やさない。
4. 表現できない場合だけ、最小primitiveをConsumerの負例・resource budgetとともに追加する。
5. 「生成できる」ことを「採用価値がある」ことの証拠にしない。

## なぜCapabilityへ直接接続しないか

PR #1058のGeneric Invocationは、Source AdapterやContract Queryのような**実行サービス**を扱います。Contract / Provider / artifact / Authority scopeを解決し、opaque requestを実装へ渡す境界です。

今回の共有生成器をそこへ載せると、純粋変換にもProvider解決が必要になり、案件ごとの生成処理を小さく保つという#1067の目的に反します。

将来、生成処理が外部schema取得、DB introspection、remote service参照などを必要とする場合は、純粋Transformからその処理を分離し、必要なreadをCapabilityとして明示的に呼びます。外部readの結果を固定入力としてTransformへ渡す構造にします。

## 次の実証条件

1. 同じ派生表現を実際に必要とする独立Consumerを二つ以上固定する。
2. 各Consumerの正本、期待値、Effect、Authority、版を比較し、同じ意味契約かを確認する。
3. 既存Driver／Evaluator／Transformで直接満たせる場合は、新しい共有生成器や演算子を追加しない。
4. 追加が必要な場合だけ、共有前後の定義量・試験量・レビュー負担を測り、共有側の固定費を含めて採算を判定する。
5. 外部read/writeを必要としない限りCapability Invocationを必須依存にしない。

この停止線により、#1069は「共通化できそうだから抽象化する」のではなく、「具体的Consumerがあり、意味と費用対効果が成立したときだけ共通化する」方式を実証対象とする。
