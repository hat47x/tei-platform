# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- Status: Proposed

## 背景

Product Requirements Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手であることを確認しました。

現行RepositoryにはDeployment executor、process Plugin、runtime reattach等の実行可能Referenceがありますが、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

`ISSUE-000005`でApplication Patternの意味境界が成立した後、その同じ`order_entry`をprogrammaticに実行する最小Driverを作ります。

## 関連するProduct Requirement

- RQ-005 Application ↔ Database N:M
- RQ-008 Capability substitution
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても次を一続きに実行し、意味上の矛盾を早い段階で発見するための環境とします。

```text
Application Definition
       ↓
Pattern / Binding resolution
       ↓
Driver Scenario
       ↓
Query / Load / Edit / Validate / Command
       ↓
Observed Result / Evidence
```

## 最小Scenario

`order_entry`で次を再現します。

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を`master_data`から解決する
4. header-detailでOrderLineを編集する
5. Validationを実行する
6. commandで`sales.order.submit`を呼び出す
7. resulting stateを観測する

## 実行境界

DriverはApplication meaningを直接Goのif文へ埋め込む場所にはしません。

少なくとも次を明示的に解決します。

- Application / Pattern identity
- Datasource alias → Database Binding
- Query identity
- Capability identity → implementation Binding
- validation / command

一方、初期Driverのためだけにproduction-grade workflow engineやdistributed schedulerを作りません。

## Database

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

actual PostgreSQL integrationを利用する場合も、Driver CoreがPostgreSQL固有APIへ依存しない境界を保ちます。

## CommandとObservation

`sales.order.submit`では、commandを呼び出したこととresulting stateを必要に応じて分けます。

第一段階でDeploymentほど重いDurable Runを要求する必要はありませんが、RQ-018の原則を崩さず、command return valueだけを外部世界の最終事実とみなさない設計を選びます。

## ScenarioとEvidence

Driver Scenarioは、別の開発者や生成AIが同じ設計を再現できる小さいfixtureとして扱います。

少なくとも次を記録できる形を検討します。

- scenario input
- resolved Binding
- validation result
- invoked Capability
- observed result

ただし、Scenario logをCanonical source of truthにはしません。

## CLI / API

最初はGUIを要求しません。

Repositoryから次のような形で実行できる、小さいCLIまたはGo APIを第一候補とします。

```text
tei driver run ...
```

実際のcommand名は既存CLI構成を監査してから決めます。

## Conformance

- 同じApplication definitionからdeterministicにScenarioを組み立てられる
- logical datasource aliasを保持する
- read-only `master_data`へwriteしない
- native SQL source-of-truthをDriverが複製しない
- Capability Bindingを明示する
- validation failureではcommandへ進まない
- command acknowledgementとobserved resultを混同しない
- Driver固有状態をCanonical Worldへ逆流させない

## 非目標

- production-grade Workflow Runtime
- distributed execution
- production HA
- completed Web / Desktop UI
- arbitrary user scripting environment
- full transaction coordinator
- external orchestratorの内製

## 完了条件

UIを用意しなくても、`order_entry`のbrowse → record → reference → header-detail → validation → commandという業務シナリオを、一つのDriver ScenarioとしてRepository内で再現できること。

その結果を、後続のWeb TargetとEditorが同じApplication meaningのEvidenceとして利用できる状態にします。
