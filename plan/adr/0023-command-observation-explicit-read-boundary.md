# ADR-0023: Command後のObservationは明示的なread境界として分離する

- 状態: 採用
- 日付: 2026-09-04

## 背景

第一Referenceの`order_entry.submit_order`では、PR #265までに次の順序を既存境界から再現できるようになりました。

```text
Authentication
  ↓
Canonical業務Validation
  ↓
Authorization
  ↓
sales.order.submit
  ↓
Command Business Result
```

ここで得られるCommand Business Resultは、Capability invocationが業務契約上の結果を返したことを表します。

しかし、それだけでは外部状態が期待どおり変更されたことを確認したとは言えません。

たとえばCommandが受理・受付済みを返した後に、別Databaseや外部サービスへの反映が遅延・失敗する可能性があります。また、現行Referenceの`sales.order.submit` adapterは入力をBusiness Resultとして返す最小実証であり、外部状態の変更そのものを証明していません。

Command acknowledgementとObserved stateを同一視すると、TargetやDriverが「正常応答」を「状態変更確認済み」へ暗黙変換する余地が生まれます。

一方、第一Referenceだけを根拠に新しいObservation Application Pattern、Observation Semantic Capability、汎用Workflow Engine、Event Sourcing、polling framework等を導入すると、まだ得られていない共通性を先に固定することになります。

## 判断

v1では、**Command後のObservationをCommand結果の派生値として扱わず、既存の明示的なread境界を別操作として実行した結果として扱います。**

概念上は次の構造です。

```text
Command invocation
        ↓
Business Result / acknowledgement
        ↓  別操作
explicit read boundary
  Query / Contract Query / read adapter / other explicit read
        ↓
read result or Execution Error
        ↓
Observation Evidence
```

Observation対象やread方法を、Command payload、Capability名、HTTP status、field名、型の一致等から推測しません。

## Observationは新しいCanonical正本を要求しない

既に業務上意味のあるQuery、Contract Query、Reference read等が存在する場合、Observationという利用目的のためだけに同じread意味を別のCanonical要素へ複製しません。

Observationは、Scenario / Runtime上で「どの明示readを、何の確認として実行したか」という役割として記録できます。

複数のReferenceから独立したObservation意味が必要だと確認された場合は、その時点でApplication PatternやSemantic Capability等への昇格を再検討します。

## Command payloadからObserved stateを作らない

次を禁止します。

- Command outputに`status=done`があるため外部状態も確認済みとする
- HTTP 200 / 202等をObserved stateへ変換する
- Command inputをそのままObservation resultとして複製する
- Capability名から自動的に対応するread Queryを選ぶ
- 同名fieldや同じ型を根拠にObservation parameterをbindingする

Command Business ResultはCommand Business Resultのまま保持します。

## Observation Evidence

Observation EvidenceはCanonicalの正本ではありません。

少なくとも次を追跡できる必要があります。

- 先行Commandのidentity / run reference
- 使用した明示read境界のidentity
- read対象のApplication / Query / Contract Query / Binding等
- read結果が正常に得られたか
- Observationの確認対象
- 確認結果または未確定状態
- Execution Error
- synthetic / external-observed / managed-here等、Evidenceの性質
- secretを含まないprovenance

read resultそのものと「期待状態に一致したか」というScenario上の評価は分離してよいものとします。

一つの固定Observation result schemaは、第一ReferenceだけからCoreへ固定しません。

## synthetic Evidence

第一ReferenceのCommandが実外部状態を書き換える接続をまだ持たない場合、statefulなsynthetic adapterを利用して境界を実証できます。

たとえば次のようにします。

```text
synthetic command adapter
  state: pending -> submitted
        ↓
Command Business Result
        ↓ separate call
synthetic read adapter
  state = submitted
        ↓
Observation Evidence
```

ただしsynthetic Evidenceは、実Oracle / PostgreSQL / 外部API等で状態変更を確認したEvidenceとは明確に区別します。

synthetic実証によって「外部システムで実際に反映された」と主張してはいけません。

## retry / eventual consistency

v1では無制限retryや汎用polling frameworkを導入しません。

readが正常に実行できたが期待状態に達していない場合と、read自体を完了できないExecution Errorは区別します。

再試行回数、間隔、deadline等が業務上必要になった場合は、実Referenceから得たEvidenceをもとに別途設計します。

## Driver / Targetとの関係

DriverはCommandとObservationを別の結果として保持します。

第一Web Targetも同じ境界を受け取り、HTTP responseやUI messageからObservationを作り直しません。

Gate Cでは、少なくとも次を別Evidenceとして追跡できることを要求します。

- Command Business Result
- Observation read result
- Observation evaluation
- Command Execution Error
- Observation Execution Error

## 影響

### 利点

- Command acknowledgementを状態確認へ誤変換しない
- 既存Query / Contract Query資産を再利用できる
- Observationのためだけの第二のread正本を作らない
- Driver / Web / 第二Targetで同じ意味境界を共有できる
- eventual consistencyや外部失敗を後から自然に追加できる

### トレードオフ

Scenarioが「どのreadをObservationとして使うか」を明示する必要があります。

ただしこの明示性は、TEIが外部状態を推測しないために必要なコストとして受け入れます。

## 非目標

- 汎用Workflow Engine
- Event Sourcing標準化
- Observation専用Application Patternの導入
- Observation専用Semantic Capabilityの導入
- CommandとObservationの自動対応付け
- payload / naming conventionからread境界を推測すること
- retry / polling frameworkの一般化

## 実証

GitHub Issue #266で、第一Referenceに対して次を実証します。

1. Validation / Authorization通過時だけCommandを実行する
2. Command Business Resultを得る
3. その結果だけではObserved stateを成立させない
4. 別の明示read境界を実行する
5. read result / 状態評価 / Execution ErrorをCommand結果から分離する
6. synthetic Evidenceを使う場合は実外部Evidenceと区別する

## 関連

- GitHub Issue #69
- GitHub Issue #266
- GitHub Issue #270
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- ADR-0020: Capability Business Outcome / Execution Error
