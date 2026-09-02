# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: Ready
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- Gate AでReadyになった関連項目: #67（Application Pattern） / #68（既存認証Compatibility） / #77（Capability Outcome） / #125（業務Validation） / #126（Capability Binding）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手でした。

現行RepositoryにはDeployment executor、process Plugin、runtime reattach等の実行可能Referenceがありますが、それらはApplication設計を開発中に動かして確かめるDriverとは役割が異なります。

Gate A監査 #124により、Driverへ渡すApplication上の意味はReadyになりました。したがって、Driver専用コードで不足する意味を補う危険を避けたまま、主実装へ進めます。

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても主要な業務Scenarioを一続きに実行し、意味上の矛盾や未解決のBindingを早い段階で見つけるための設計検証環境です。

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

Driverは不足するApplication上の意味を埋める場所ではありません。Gate Aで成立した正本を読み、実行・検証することが役割です。

## Gate Aから受け取るもの

### Pattern / Data access

`order_entry`では次を解決できます。

- `browse`: `order_entry.recent_orders`
- `record`: `order_entry.order_record`
- `reference`: `sales.order.customer` + datasource `master`
- `header-detail`: `sales.order.lines -> sales.order_line many`
- `command`: `order_entry.submit_order -> sales.order.submit`
- `sales`: read-write
- `master`: read-only

Driver専用設定へこれらを書き写しません。

### Capability Binding

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決する入口が成立しています。

#68では同じ原則をAuthentication / Authorizationにも適用しました。

```text
order_entry / sales.order.submit
order_entry / security.authenticate
order_entry / security.authorize
```

Driverはfile path、Go hook、列挙順からimplementationを推測しません。

### 業務Validation

Issue #125で、次のCanonical ruleと最小Evaluatorが成立しています。

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

Driverは`quantity > 0`等のrule本文を再実装せず、Canonical ValidationとEvaluatorを利用します。

Validation不成立は`NotSatisfied`、評価そのものを完了できない状態はerrorとして扱います。

### Capability結果境界

ADR-0020 / Issue #77を正とします。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability invocation自体を完了できなかった状態
```

`security.authenticate`の`authenticated` / `denied`、`security.authorize`の`allowed` / `denied`は正常な業務上の結果です。

Binding解決失敗、Database接続失敗、adapter障害、未対応legacy return code等はExecution Errorです。

既存の`sales.order.submit`は従来の`output sales.order`を維持し、同じInvocation入口で扱います。

### Authentication / Authorization

Issue #68 / PR #148 / #149 / #151で、Gate Aに必要な境界が成立しました。

Authentication:

```text
security.authenticate
  ├─ authenticated(subject)
  └─ denied(reason)
```

Authorization:

```text
subject + action + resource
          ↓
 security.authorize
     ├─ allowed
     └─ denied
```

AuthorizationのresourceにはApplication Pattern identityを利用します。

```text
browse  / order_entry.recent_orders
edit    / order_entry.order_record
execute / order_entry.submit_order
```

Driverはlegacy schema、organization、role、legacy return codeを直接読みません。これらは既存認証adapterの内側へ残します。

## 最小Scenario

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

この順序を固定的な汎用workflow engineへ一般化することは、第一段階の目的ではありません。まず一つのReference Scenarioを、同じApplication上の意味から再現できることを確認します。

## 正常な業務上の不成立

次はDriver自体の障害ではありません。

- Authenticationの`denied`
- Validationの`NotSatisfied`
- Authorizationの`denied`
- Capabilityが契約上定義している別Outcome

Scenarioは、その段階で後続処理へ進めない理由をEvidenceとして残せます。

たとえばAuthorizationが`denied`ならcommandを呼び出しませんが、それをDriverのExecution Errorとは扱いません。

## Execution Error

次は業務上の拒否へ丸めません。

- Pattern / Bindingを一意に解決できない
- Databaseへ接続できない
- adapterを起動・実行できない
- legacy実装が未対応return codeを返す
- Capability契約にないOutcomeを返す
- Validation評価そのものを完了できない

Driver独自のerror code体系を作るのではなく、既存の各境界から返されるerrorを、Scenario実行を完了できなかった状態として扱います。

## SecretとEvidence

`security.authenticate`のCanonical契約にcredential型があっても、実際のpassword、token、credential hash、接続資格情報をCanonical Worldや通常Evidenceへ保存しません。

Driver Scenarioでも、入力の再現性を理由にsecretを平文で記録しません。

Scenario Evidenceに残せるのは、たとえば次の情報です。

- credential値そのものではない入力識別情報
- Authentication Outcome
- authenticated subject identity
- 利用したCapability / Bindingのidentity
- secretではない来歴

production用のsecret解決方式は、このvertical sliceの前提にはしません。

## Database境界

最初はReference用adapter / fixtureを利用して構いませんが、`sales_core`と`master_data`という二つのlogical datasourceを維持します。

Driverの都合で一つのin-memory Databaseへ意味上統合しません。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

`master_data`へwriteしないことをScenarioのConformanceとして確認します。

既存認証が別Database、LDAP、既存関数、外部API等を利用しても、DriverはAuthentication / Authorization Capabilityを通して利用し、物理構造へ直接依存しません。

## CommandとObservation

Capability invocationの結果と、外部世界で実際に成立した状態は別の問いです。

`sales.order.submit`では少なくとも次を分けます。

1. Capability呼び出しを開始・完了できたか
2. CapabilityがどのBusiness Resultを返したか
3. 外部世界で期待した状態が実際に成立したか

commandのreturn valueだけを外部世界の最終事実とはみなしません。必要な場合はObservationから確認します。

## Scenario Evidence

Driver Scenarioは、別の開発者や生成AIがRepositoryから同じ設計を再現できる小さなEvidenceとして扱います。

少なくとも次を記録できる形を検討します。

- scenario inputの非secret部分
- resolved Pattern
- resolved logical Datasource
- resolved Capability Binding
- Authentication Result
- Validation Result
- Authorization Result
- invoked Capability
- Business Result
- Execution Errorの有無
- command acknowledgement
- observed result

Scenario logをCanonicalの正本にはしません。Driver固有状態をCanonical Worldへ逆流させません。

## 実装順

最初から大きなDriver frameworkを作らず、次の順でvertical sliceを成立させます。

1. `order_entry`を読み込むDriverの最小入口を置く
2. ScenarioからApplication PatternをCanonical identityで解決する
3. logical Datasource境界を解決する
4. Authenticationを既存Bindingから実行する
5. browse / record / reference / header-detailを一続きに扱う
6. Canonical Validationを既存Evaluatorへ渡す
7. Authorizationを既存Bindingから実行する
8. `sales.order.submit`を既存Bindingから呼び出す
9. Business Result / Execution ErrorをScenario結果へ反映する
10. 必要なObservationを別の結果として扱う
11. Repositoryから再現可能なScenario Evidenceを整える
12. Gate Bを監査する

CLI/APIの具体的な名前やScenario file formatは、最小縦切りを組む段階で既存CLIとの整合を確認して決めます。先に構文を固定しません。

## Gate Bへ進む条件

Driverのコードが存在するだけでは、Gate Bを通過したとはみなしません。

少なくとも次をReference Scenarioから確認します。

- 同じApplication定義から再現可能にScenarioを組み立てられる
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
- Runtime上の汎用implementation選択機構

## 開始条件の評価

- [x] Gate A監査 #124がReady
- [x] Application PatternがReady
- [x] Capability BindingがReady
- [x] 業務ValidationがReady
- [x] Capability Business Result / Execution Error境界がReady
- [x] Authentication / AuthorizationがReady
- [x] legacy認証CompatibilityがReady
- [x] Secret非保存境界がReady

したがって、このIssueは**Ready**です。

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143 / #148 / #149 / #151
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
