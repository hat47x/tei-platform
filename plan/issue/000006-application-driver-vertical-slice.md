# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: Blocked
- GitHub Issue: #69
- Blocker: `ISSUE-000013` / GitHub Issue #124（Gate A到達性監査）
- Gate Aの現在の残条件: #68 / #77 / #125
- Gate AでReadyになった関連項目: #126（Capability Binding）

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattachなどの実行可能Referenceがあります。しかし、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Application Pattern Stageは完了し、`order_entry`では次の5種類が実Referenceまで成立しました。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

Capability BindingもIssue #126 / PR #135で実証済みです。

一方、Gate Aの監査では、Driverへ渡すApplication上の意味がまだすべて揃っていないことが分かっています。このため、Driverの主実装はGate Aを通過するまで開始しません。

## なぜ主実装を止めるか

Driverを先に実装すると、未解決の意味をDriver専用のGoコードや設定へ埋め込んでしまうおそれがあります。

現在の残条件は次の3つです。

- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #77: named Outcomeを実行時のBusiness Resultとして扱い、Execution Errorと分離するEvidence
- GitHub Issue #125: 業務Validationの正本と最小Reference

これらは、後から単純にリファクタリングすればよい実装詳細ではありません。Driverが何を実行し、何を業務結果として受け取り、誰に何を許可し、どの条件でcommandへ進むかを決める上位の意味です。

## 関連するProduct Requirement

- RQ-005 Application ↔ Database N:M
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可互換
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance

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

Application上の不足をDriver専用コードで補うのではなく、Gate Aで揃えた意味を実行することがDriverの役割です。

## Gate Aから受け取るもの

Driverの主実装へ進む前に、少なくとも次をRepository上のEvidenceから説明できることを前提とします。

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

Driverはfile path、Go hook、列挙順をBinding discoveryに利用する必要がありません。

0件または複数件で一意に解決できない場合も暗黙選択せずerrorとして扱えます。

Driverはこの既存の解決入口を利用し、Binding定義を別のDriver専用設定へ複製しません。

### 業務Validation — Blocked

Driverが必要とするのは、Pattern定義の構文・参照整合性Validationではなく、業務上のruleです。

- Validation identity
- ruleの正本
- 対象Data / Relation等のidentity
- Validation成立 / 不成立の結果契約

この部分は`ISSUE-000014` / GitHub Issue #125で実証します。

### Authentication / Authorization — Blocked

- authenticated subject
- Authentication Capability identity
- Authorization Capability identity
- 対象action / resource
- legacy field / functionとのMapping
- secret解決境界

この部分は`ISSUE-000003` / GitHub Issue #68で実証します。

### Capability結果境界 — Partial

ADR-0020 / `ISSUE-000009` / GitHub Issue #77を正とします。

PR #132で、名前付きOutcomeのCanonical model / Parser / Validation / Normalizationまでは実装済みです。

Driver開始前には、少なくとも次を実Referenceから確認する必要があります。

- 既存`output` Business Result
- named Outcome Business Result
- expected denialをExecution Errorへ変換しない境界
- Binding / adapter / Database障害を業務Outcomeへ押し込まない境界

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
- Command / Observation分離の基本境界

これらをDriver専用設定へ再記述しません。

## 最小Scenario

Gate A通過後、`order_entry`で次を再現します。

1. `browse`で受注一覧を取得する
2. `record`で一件を扱う
3. `reference`で顧客を`master_data`から解決する
4. `header-detail`でOrderLineを扱う
5. 業務Validationを実行する
6. Authorizationを評価する
7. `command`で`sales.order.submit`を呼び出す
8. CapabilityのBusiness ResultとExecution Errorを区別する
9. 必要な場合は、外部状態をObservationから確認する

ValidationまたはAuthorizationが業務上成立しなかった場合はcommandへ進みません。

これはDriver自体の障害ではなく、Scenarioが正常に確認できる業務結果として扱います。

一方、Database接続失敗、Binding解決失敗、adapter異常など、実行そのものを完了できなかった場合はExecution Errorとして扱います。

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

Authenticationであれば、`authenticated`と`denied`はいずれも正常に実行されたBusiness Resultです。credential不一致やaccount lockによる`denied`と、認証Databaseへの接続失敗は意味が異なります。

`sales.order.submit`については、現行`output sales.order`を不用意にOutcomeへ書き換えません。

## Validationとの境界

Pattern定義の構文・参照整合性Validationと、業務Validationを混同しません。

Driverが実行する業務Validationは`ISSUE-000014`で正本とidentityを実証します。

Driver専用Goコードにだけruleを書いたり、Web form validationを正本にしたりしません。

## Capability Bindingとの境界

`command` PatternはCapability identityだけを保持します。

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

Bindingが見つからない場合、または複数あって一意に決められない場合は、Business Resultではなく実行準備上のerrorとして扱います。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

actual PostgreSQL integrationを利用する場合も、Driver CoreがPostgreSQL固有APIへ依存しない境界を保ちます。

既存認証Compatibilityが別Databaseを利用する場合も、DriverはAuthentication / Authorization Capabilityを通して利用し、legacy schemaへ直接依存しません。

## CommandとObservation

Capability invocationの結果を扱えるようになっても、RQ-018のCommand / Observation境界は置き換えません。

`sales.order.submit`では、少なくとも次を別の問いとして扱います。

1. Capability呼び出し自体を実行できたか
2. CapabilityがどのBusiness Resultを返したか
3. 外部世界で期待した状態が実際に成立したか

第一段階でDeploymentほど重いDurable Runを要求する必要はありませんが、commandのreturn valueだけを外部世界の最終事実とみなさない設計を維持します。

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
- Execution Errorの有無と必要なcategory
- command acknowledgement
- observed result

Scenario logをCanonicalの正本にはしません。

secretやcredential materialを通常Evidenceへ保存しない原則も維持します。

## CLI / API

最初はGUIを要求しません。

Repositoryから小さなCLIまたはGo APIで実行できる形を第一候補とします。

```text
tei driver run ...
```

実際のcommand名は既存CLI構成を監査してから決めます。

CLI構文を先に固定することより、Driver CoreがApplication上の意味と結果境界を正しく扱えることを優先します。

## Gate Bへ進む条件

Driverの実装が完了したという事実だけではなく、次をEvidenceから確認できたときにGate B「UIなしで業務シナリオを説明できる」を通過したとみなします。

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
- Web実装を成立させるためのTarget固有処理
- HTTP statusやGUI messageをDriver Coreの意味として固定すること
- Driver独自のOutcome体系を作ること

## Block解除条件

`ISSUE-000013` / GitHub Issue #124で、Gate Aの必須項目がすべてReadyと判定されたときに、このIssueの状態をReady / In Progressへ移します。

現時点では、少なくとも#68、#77、#125のGate A必須範囲が成立していることを確認する必要があります。

Issueが単にcloseしたかではなく、Driverへ必要な意味をRepositoryから再現できるかを基準にします。

Capability Binding #126は既にReadyです。

## 完了条件

UIを用意しなくても、`order_entry`のbrowse → record → reference → header-detail → validation → authorization → command → observationという業務シナリオを、一つのDriver ScenarioとしてRepository内で再現できることを完了条件とします。

その過程で、既存`output`または名前付きOutcomeとして返るBusiness Result、Execution Error、command acknowledgement、observed resultを必要に応じて区別できることもEvidenceとして残します。

後続の第一Web TargetとEditorが、同じApplication上の意味と同じ結果境界を利用でき、Gate Bを評価できる状態にします。

## 関連文書

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
