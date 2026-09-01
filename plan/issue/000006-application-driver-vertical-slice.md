# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: 提案中
- GitHub Issue: #69
- 依存: Gate A

## 背景

Product Requirements Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattachなどの実行可能Referenceがありますが、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Driverは`ISSUE-000005`のApplication Patternだけを受け取って先行実装するのではなく、`ISSUE-000003`のAuthentication / Authorization境界も含め、Gate A「Application上の意味をDriverへ渡せる」を確認してから主実装へ進みます。

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
- AuthorizationのCapabilityと対象action
- read / read-write Datasource境界

これらをWeb画面やDriver専用設定から初めて知る状態では、着手条件を満たしません。

## 最小Scenario

`order_entry`で次を再現します。

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を`master_data`から解決する
4. header-detailでOrderLineを編集する
5. Validationを実行する
6. Authorizationを評価する
7. commandで`sales.order.submit`を呼び出す
8. 結果として得られた状態を観測する

ValidationまたはAuthorizationが失敗した場合はcommandへ進みません。

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

一方、初期Driverのためだけにproduction-grade workflow engineやdistributed schedulerを作りません。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

actual PostgreSQL integrationを利用する場合も、Driver CoreがPostgreSQL固有APIへ依存しない境界を保ちます。

既存認証Compatibilityが第三Databaseを利用する場合も、DriverはAuthentication / Authorization Capabilityを通して利用し、legacy schemaへ直接依存しません。

## CommandとObservation

`sales.order.submit`では、commandを呼び出したことと結果として得られた状態を必要に応じて分けます。

第一段階でDeploymentほど重いDurable Runを要求する必要はありませんが、RQ-018の原則を崩さず、commandのreturn valueだけを外部世界の最終事実とみなさない設計を選びます。

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
- command acknowledgement
- observed result

ただし、Scenario logをCanonicalの正本にはしません。

## CLI / API

最初はGUIを要求しません。

Repositoryから小さなCLIまたはGo APIで実行できる形を第一候補とします。

```text
tei driver run ...
```

実際のcommand名は既存CLI構成を監査してから決めます。

CLI構文を先に固定することより、Driver CoreがApplication上の意味を正しく実行できることを優先します。

## Gate Bへ進む条件

Driverの実装が完了したという事実だけではなく、次をEvidenceから確認できたときにGate B「UIなしで業務シナリオを説明できる」を通過したとみなします。

- 同じApplication定義からdeterministicにScenarioを組み立てられる
- logical datasource aliasを保持する
- read-only `master_data`へwriteしない
- native SQLの正本をDriverが複製しない
- Capability Bindingを明示する
- legacy authentication schemaへDriverが直接依存しない
- validation failureではcommandへ進まない
- authorization failureではcommandへ進まない
- command acknowledgementとobserved resultを混同しない
- Driver固有状態をCanonical Worldへ逆流させない
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

## 完了条件

UIを用意しなくても、`order_entry`のbrowse → record → reference → header-detail → validation → authorization → command → observationという業務シナリオを、一つのDriver ScenarioとしてRepository内で再現できること。

その結果を、後続の第一Web TargetとEditorが同じApplication上の意味を裏付けるEvidenceとして利用でき、Gate Bを評価できる状態にします。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000007-first-web-target-projection.md`
