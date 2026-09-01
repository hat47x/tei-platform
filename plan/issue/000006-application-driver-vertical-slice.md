# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: 提案中
- GitHub Issue: #69
- 依存: Gate A、ADR-0020 / `ISSUE-000009`のCapability結果境界

## 背景

Product Requirements Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattachなどの実行可能Referenceがありますが、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Driverは`ISSUE-000005`のApplication Patternだけを受け取って先行実装するのではなく、`ISSUE-000003`のAuthentication / Authorization境界も含め、Gate A「Application上の意味をDriverへ渡せる」を確認してから主実装へ進みます。

また、既存認証Compatibilityの設計から、Capabilityの呼び出し結果について「正常に返された業務上の結果」と「実行そのものを完了できなかった状態」を分ける必要があることが分かりました。この共通境界は[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)で定め、実装前設計は[`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)に置きます。Driverはこの区別を失わずScenarioへ反映する必要があります。

## 関連するProduct Requirement

- RQ-005 Application ↔ Database N:M
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可互換
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても次を一続きに実行し、意味上の矛盾を早い段階で発見するための環境とします。

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

Applicationの不足をDriver専用コードで補うのではなく、Gate Aで揃えたApplication上の意味を実行することがDriverの役割です。

## Gate Aから受け取るもの

Driverの主実装へ進む前に、少なくとも次がApplication側から説明できることを前提とします。

- browse対象
- record対象
- reference対象とDatasource
- header-detailの関係
- Validation
- commandとして呼ぶCapability
- Authentication / Authorization Capabilityと対象action
- read / read-write Datasource境界
- Capability Bindingのidentity
- CapabilityのBusiness ResultとExecution Errorを区別する契約

これらをWeb画面やDriver専用設定から初めて知る状態では、着手条件を満たしません。

Application Pattern側の詳細は`plan/APPLICATION_PATTERN_V1_DESIGN.md`、既存認証側の詳細は`plan/LEGACY_AUTH_V1_DESIGN.md`、Capability結果境界の詳細は`plan/CAPABILITY_OUTCOME_V1_DESIGN.md`を参照します。

## 最小Scenario

`order_entry`で次を再現します。

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を`master_data`から解決する
4. header-detailでOrderLineを編集する
5. Validationを実行する
6. Authorizationを評価する
7. commandで`sales.order.submit`を呼び出す
8. Capabilityが返したBusiness ResultとExecution Errorを区別する
9. 必要な場合は、結果として得られた外部状態を観測する

ValidationまたはAuthorizationが業務上成立しなかった場合は、commandへ進みません。これはDriver自体の障害ではなく、Scenarioが正常に確認できる結果として扱います。

一方、Database接続失敗やCapability adapterの異常など、実行そのものを完了できなかった場合は、Business Resultとは別のExecution Errorとして扱います。

## Business ResultとExecution Error

ADR-0020の境界をDriverへそのまま引き継ぎます。

Capabilityが正常に実行された場合、Driverが受け取るBusiness Resultには次の二形態があります。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability呼び出しそのものを完了できなかった状態
```

既存の単一`output` CapabilityはOutput valueを返せます。名前付きOutcomeを持つCapabilityは、正常な一回のInvocationにつき契約上のOutcomeを一つ返します。

Authenticationであれば、`authenticated`と`denied`はいずれも正常に実行されたBusiness Resultです。credential不一致やaccount lockによる`denied`は、認証Databaseへの接続失敗とは意味が異なります。

`sales.order.submit`については、現行`output sales.order`を直ちにOutcomeへ書き換えません。Commandを実行して初めて判明する業務上の分岐がReferenceで必要になった場合に、ADR-0020の名前付きOutcomeが自然に使えるかを圧力試験します。ValidationやAuthorizationで事前に判定できる不成立をCommand Outcomeへ重複させません。

Driverは、この区別をTarget向けのHTTP statusやGUI messageへ先回りして変換しません。Target固有表現は後続Projectionの責務です。

## 実行境界

DriverはApplication上の意味を直接Goのif文へ埋め込む場所にはしません。

少なくとも次を明示的に解決します。

- Application / Pattern identity
- Datasource alias → Database Binding
- Query identity
- Capability identity → implementation Binding
- Validation
- Authentication / Authorization Capability
- Command
- Capability Business Result / Execution Error

一方、初期Driverのためだけにproduction-grade workflow engineやdistributed schedulerを作りません。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

actual PostgreSQL integrationを利用する場合も、Driver CoreがPostgreSQL固有APIへ依存しない境界を保ちます。

既存認証Compatibilityが別Databaseを利用する場合も、DriverはAuthentication / Authorization Capabilityを通して利用し、legacy schemaへ直接依存しません。

## CommandとObservation

Capability invocationの結果を区別できるようになっても、RQ-018のCommand / Observation境界は置き換えません。

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
  - output value、またはnamed Outcomeとpayload
- Execution Errorの有無と必要なcategory
- command acknowledgement
- observed result

ただし、Scenario logをCanonicalの正本にはしません。secretやcredential materialを通常Evidenceへ保存しない原則も維持します。

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
- Capability Bindingを明示する
- legacy authentication schemaへDriverが直接依存しない
- validation failureではcommandへ進まない
- authorization denialではcommandへ進まない
- expectedなnamed OutcomeをDriver障害として扱わない
- Execution Errorを業務上のOutcomeやresult codeへ押し込まない
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

## 完了条件

UIを用意しなくても、`order_entry`のbrowse → record → reference → header-detail → validation → authorization → command → observationという業務シナリオを、一つのDriver ScenarioとしてRepository内で再現できることを完了条件とします。

その過程で、既存`output`または名前付きOutcomeとして返るBusiness Result、Execution Error、command acknowledgement、observed resultを必要に応じて区別できることもEvidenceとして残します。

後続の第一Web TargetとEditorが、同じApplication上の意味と同じ結果境界を利用でき、Gate Bを評価できる状態にします。

## 関連文書

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
