# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: Blocked
- GitHub Issue: #69
- Blocker: `ISSUE-000013` / GitHub Issue #124（Gate A到達性監査）
- Gate Aの現在の残条件: #68
- Gate AでReadyになった関連項目: #77（Capability Outcome） / #125（業務Validation） / #126（Capability Binding）

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattachなどの実行可能Referenceがあります。しかし、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Application Pattern Stageは完了し、`order_entry`では`browse`、`record`、`reference`、`command`、`header-detail`の5種類が実Referenceまで成立しています。

Capability Binding #126、業務Validation #125、Capability Outcome / Execution Error #77もGate Aで`Ready`になりました。

現在残っている独立blockerは#68の既存認証Compatibilityです。Authentication契約と結果境界は成立していますが、legacy認証仕様のMapping / BindingとAuthorizationがまだ揃っていないため、Driverの主実装はGate Aを通過するまで開始しません。

## なぜ主実装を止めるか

Driverを先に実装すると、未解決のAuthentication / Authorization条件をDriver専用Goコードや設定へ埋め込むおそれがあります。

Driverは不足するApplication上の意味を補う場所ではなく、Gate Aで成立した意味を実行・検証する場所とします。

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

### Capability結果境界 — Ready

ADR-0020 / GitHub Issue #77を正とします。

PR #132で名前付きOutcomeのCanonical契約を実装し、PR #141で実行時境界を実証しました。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability invocation自体を完了できなかった状態
```

`security.authenticate`では、`authenticated` / `denied`を正常なBusiness Resultとして扱い、Database / adapter障害や未定義OutcomeをExecution Errorとして分離できます。

既存の`sales.order.submit`も従来の`output sales.order`を保ったまま、同じInvocation入口で扱えます。

Driverはこの境界を再利用し、独自のOutcome体系を作りません。

### Authentication / Authorization — Blocked

`security.authenticate`のSemantic契約は存在しますが、Gate Aにはさらに次が必要です。

- authenticated subject
- Authentication Capabilityとlegacy実装のMapping / Binding
- Authorization Capability identity
- Application上のaction / resource
- legacy field / functionとのMapping
- secretを通常Evidenceへ残さない境界

この部分は`ISSUE-000003` / GitHub Issue #68で実証します。

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
- named Outcome / Execution ErrorのInvocation境界
- `security.authenticate`のnamed Outcome契約
- Command / Observation分離の基本境界

これらをDriver専用設定へ再記述しません。

## Gate A通過後の最小Scenario

`order_entry`で次を再現します。

1. Authenticationを実行してsubjectを得る
2. `browse`で受注一覧を取得する
3. `record`で一件を扱う
4. `reference`で顧客を`master_data`から解決する
5. `header-detail`でOrderLineを扱う
6. Canonical業務Validationを実行する
7. Authorizationを評価する
8. `command`で`sales.order.submit`を呼び出す
9. CapabilityのBusiness ResultとExecution Errorを区別する
10. 必要な場合は外部状態をObservationから確認する

Authenticationの`denied`、Validationの`NotSatisfied`、Authorizationの業務上の拒否は、Driver自体の障害とは扱いません。

一方、Binding解決失敗、Database接続失敗、adapter異常等はExecution Errorです。

## SecretとEvidence

`security.authenticate`のCanonical契約にcredential型があっても、実際のpassword、token、credential hash、接続資格情報をCanonical Worldや通常Evidenceへ保存しません。

Driver Scenarioでも、入力の再現性を理由にsecretを平文で記録しません。必要な場合は、実行時に安全な境界から受け取るか、Reference専用の非secret fixture値へ置き換えます。

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

Authentication / Authorizationも同じ原則に従い、legacy schemaや実装名をDriverのCanonicalな意味にしません。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

既存認証Compatibilityが別の保存先や既存関数を利用する場合も、DriverはAuthentication / Authorization Capabilityを通して利用し、legacy schemaへ直接依存しません。

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

- scenario inputの非secret部分
- resolved Pattern
- resolved Datasource Binding
- resolved Capability Binding
- authentication result
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
- Driver独自のAuthentication / Authorization rule

## Block解除条件

`ISSUE-000013` / GitHub Issue #124で、Gate Aの必須項目がすべてReadyと判定されたときに、このIssueを`Ready / In Progress`へ移します。

現時点の独立blockerは#68のGate A必須範囲です。

Issueが単にcloseしたかではなく、Driverへ必要な意味をReference / Conformanceから再現できることを条件とします。

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- GitHub Issue #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
