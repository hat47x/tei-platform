# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: Blocked
- GitHub Issue: #69
- Blocker: `ISSUE-000013` / GitHub Issue #124（Gate A到達性監査）
- Gate Aの現在の残条件: #68 / #77
- Gate AでReadyになった関連項目: #125（業務Validation） / #126（Capability Binding）

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattachなどの実行可能Referenceがあります。しかし、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Application Pattern Stageは完了し、`order_entry`では`browse`、`record`、`reference`、`command`、`header-detail`の5種類が実Referenceまで成立しています。

さらに、Capability Binding #126と業務Validation #125もGate Aで`Ready`になりました。

一方、Authentication / AuthorizationとCapabilityのnamed Outcome実行境界はまだ揃っていません。このため、Driverの主実装はGate Aを通過するまで開始しません。

## なぜ主実装を止めるか

Driverを先に実装すると、未解決の認証・認可条件やCapabilityの結果境界をDriver専用Goコードや設定へ埋め込むおそれがあります。

現在の残条件は次の2つです。

- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #77: named OutcomeをBusiness Resultとして扱い、Execution Errorと分離する実行時Evidence

これらは単なる実装詳細ではありません。Driverが「誰が実行できるか」「正常な業務上の拒否とは何か」「実行障害とは何か」を判断するための上位の意味です。

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても次を一続きに実行し、意味上の矛盾を早い段階で見つけるための設計検証環境です。

```text
Application定義
       ↓
Pattern / Binding / Authorizationの解決
       ↓
Driver Scenario
       ↓
Query / Load / Edit / Validate / Authorize / Command
       ↓
Business Result / Execution Error
       ↓
Observed Result / Evidence
```

Application上の不足をDriver専用コードで補うのではなく、Gate Aで成立した意味を実行することがDriverの役割です。

## Gate Aから受け取るもの

### Pattern / Data access — Ready

- browse対象
- record対象
- reference対象RelationとDatasource
- header-detail対象Relation
- commandとして呼ぶCapability
- read / read-write Datasource境界

この部分はApplication Pattern StageでReadyです。

### Capability Binding — Ready

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決できるBinding Referenceが成立しました。

Driverはfile path、Go hook、列挙順をBinding discoveryに利用しません。

0件または複数件で一意に解決できない場合は、Business Resultへ変換せずerrorとして扱います。

### 業務Validation — Ready

Issue #125 / PR #136 / #139 / #140で、業務ValidationはGate Aに必要な範囲を満たしました。

- Validation identityとauthoring sourceを追跡できる
- `sales.order_line.quantity_positive`をCanonical ruleとして保持できる
- `sales.order.lines.line_number_unique`をCanonical ruleとして保持できる
- 同じruleを最小Evaluatorへ渡せる
- Validation不成立を`NotSatisfied`として扱える
- 評価そのものを完了できない状態を`NotSatisfied`へ畳まずerrorとして扱える

Driverはrule本文を専用Goコードへ複製せず、このEvaluator境界を利用します。

### Authentication / Authorization — Blocked

Gate Aへは少なくとも次が必要です。

- authenticated subject
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resource
- legacy field / functionとのMapping
- secret解決境界

この部分は`ISSUE-000003` / GitHub Issue #68で実証します。

### Capability結果境界 — Partial

ADR-0020 / `ISSUE-000009` / GitHub Issue #77を正とします。

PR #132で、名前付きOutcomeのCanonical model / Parser / Validation / Normalizationまでは実装済みです。

Driver開始前には、少なくとも次を実Referenceから確認する必要があります。

- named Outcomeを正常なBusiness Resultとして扱える
- expected denialをExecution Errorへ変換しない
- Binding / adapter / Database障害を業務Outcomeへ押し込まない
- DriverへBusiness ResultとExecution Errorの区別を渡せる

#68と#77は`security.authenticate`のReferenceで合流させます。

## 現時点でReadyな入力

Gate A監査では、次を既にReadyと評価しています。

- `browse`: `order_entry.recent_orders`
- `record`: `order_entry.order_record`
- `reference`: `sales.order.customer` + datasource `master`
- `header-detail`: `sales.order.lines -> sales.order_line many`
- `command`: `sales.order.submit`
- Pattern identity / Source Index
- `sales` read-write / `master` readのDatasource境界
- `sales.order.submit`の既存単一`output sales.order`
- `sales.order.submit`の明示的なCapability Binding
- Binding authoring source / provenance
- 業務ValidationのCanonical rule / Source / Evaluator
- Command / Observation分離の基本境界

これらをDriver専用設定へ再記述しません。

## Gate A通過後の最小Scenario

`order_entry`で次を再現します。

1. `browse`で受注一覧を取得する
2. `record`で一件を扱う
3. `reference`で顧客を`master_data`から解決する
4. `header-detail`でOrderLineを扱う
5. Canonical業務Validationを実行する
6. Authorizationを評価する
7. `command`で`sales.order.submit`を呼び出す
8. CapabilityのBusiness ResultとExecution Errorを区別する
9. 必要な場合は外部状態をObservationから確認する

ValidationまたはAuthorizationが業務上成立しなかった場合はcommandへ進みません。これはDriver自体の障害ではなく、正常に評価できた業務上の結果として扱います。

一方、Database接続失敗、Binding解決失敗、adapter異常など、実行そのものを完了できなかった状態はExecution Errorです。

## Business ResultとExecution Error

ADR-0020の境界をDriverへそのまま引き継ぎます。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability invocation自体を完了できなかった状態
```

既存の単一`output` CapabilityはOutput valueを返せます。

名前付きOutcomeを持つCapabilityでは、expected denial等をDriver障害へ変換しません。

Authenticationでは、`authenticated`と`denied`はいずれも正常に実行されたBusiness Resultです。credential不一致やaccount lockによる`denied`と、認証Databaseへの接続失敗は意味が異なります。

`sales.order.submit`の既存`output sales.order`は不用意にOutcomeへ書き換えません。

## Validationとの境界

Driverが実行する業務ValidationはSemantic Validationを正本とします。

Driverは、PR #140で成立したEvaluatorへ実Scenarioの値を渡すAdapterを持つことはできますが、`quantity > 0`等のrule本文を再実装しません。

Evaluatorの`Source`境界は、Driverより先に汎用的な業務データObject modelを固定しないための最小境界です。Driver Scenarioから自然な値表現が得られた時点で、必要なら圧力試験します。

## Capability Bindingとの境界

`command` PatternはCapability identityを保持します。

Driverは、そのidentityとApplication scopeから既存のBinding解決入口を利用します。

```text
order_entry.submit_order
        ↓ capability
sales.order.submit
        ↓ Application + Capability
Custom Go Binding
        ↓
implementation
```

file pathやGo symbolを暗黙に推測してimplementationへ接続しません。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

既存認証Compatibilityが別Databaseを利用する場合も、DriverはAuthentication / Authorization Capabilityを通して利用し、legacy schemaへ直接依存しません。

## CommandとObservation

Capability invocationの結果を扱えるようになっても、RQ-018のCommand / Observation境界は置き換えません。

`sales.order.submit`では、少なくとも次を別の問いとして扱います。

1. Capability呼び出し自体を実行できたか
2. CapabilityがどのBusiness Resultを返したか
3. 外部世界で期待した状態が実際に成立したか

commandのreturn valueだけを外部世界の最終事実とはみなしません。

## ScenarioとEvidence

Driver Scenarioは、別の開発者や生成AIが同じ設計を再現できる小さなfixtureとして扱います。

少なくとも次を記録できる形を検討します。

- scenario input
- resolved Pattern
- resolved Datasource Binding
- resolved Capability Binding
- validation result
- authorization result
- invoked Capability
- Business Result
- Execution Errorの有無
- command acknowledgement
- observed result

Scenario logをCanonicalの正本にはしません。secretやcredential materialも通常Evidenceへ保存しません。

## Gate Bへ進む条件

Driverの実装が存在するだけでは、Gate B「UIなしで業務シナリオを説明できる」を通過したとはみなしません。

少なくとも次をReference Scenarioから確認します。

- 同じApplication定義からdeterministicにScenarioを組み立てられる
- logical datasource aliasを保持する
- read-only `master_data`へwriteしない
- native SQLの正本をDriverが複製しない
- Capability Bindingを明示的に解決する
- legacy authentication schemaへDriverが直接依存しない
- validation failureではcommandへ進まない
- authorization denialではcommandへ進まない
- expectedなnamed OutcomeをDriver障害として扱わない
- Execution Errorを業務Outcomeやresult codeへ押し込まない
- 既存`output`と名前付きOutcomeの両方をBusiness Resultとして扱える
- command acknowledgementとobserved resultを混同しない
- Driver固有状態をCanonical Worldへ逆流させない
- secretをScenario Evidenceへ混入させない
- 同じApplication上の意味を第一Web Targetへ渡せる
- Reference ScenarioをRepositoryだけから再現できる

## 非目標

- production-grade Workflow Runtime
- distributed execution
- production HA
- completed Web / Desktop UI
- arbitrary user scripting environment
- full transaction coordinator
- external orchestratorの内製
- HTTP statusやGUI messageをDriver Coreの意味として固定すること
- Driver独自のOutcome体系
- Driver独自のValidation rule

## Block解除条件

`ISSUE-000013` / GitHub Issue #124で、Gate Aの必須項目がすべてReadyと判定されたときに、このIssueを`Ready / In Progress`へ移します。

現時点では#68と#77のGate A必須範囲が残っています。

Issueが単にcloseしたかではなく、Driverへ必要な意味をReference / Conformanceから再現できることを条件とします。

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- GitHub Issue #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
