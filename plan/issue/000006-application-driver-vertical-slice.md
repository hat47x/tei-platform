# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Query実行: `ISSUE-000016` / GitHub Issue #167
- Relation traversal: `ISSUE-000017` / GitHub Issue #178
- Database Contract Query: `ISSUE-000018` / GitHub Issue #179
- Reference invocation: `ISSUE-000019` / GitHub Issue #185
- CI基盤: GitHub Issue #192

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手でした。

現行RepositoryにはDeployment executorやprocess Plugin等の実行基盤がありますが、それらはApplication設計を開発途中に動かし、意味の不足や矛盾を見つけるDriverとは役割が異なります。

Gate A監査 #124により、Driverへ渡すApplication上の意味はReadyになりました。現在は、その意味をDriver専用コードへ複製せず、実Referenceから必要になった境界だけを小さなvertical sliceとして積み上げています。

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても主要な業務Scenarioを一続きに実行し、次のような問題を早い段階で見つけるための設計検証環境です。

- Patternが参照する意味を一意に解決できない
- logical DatasourceやBindingが不足している
- 業務Validationが成立しない
- Authentication / Authorizationで後続処理へ進めない
- CapabilityのBusiness ResultとExecution Errorが混同されている
- native SQLや既存実装とCanonicalな意味の間に対応関係が不足している
- Relationは定義されているが、instance間を辿るSemantic keyがない
- Database Contract Queryは存在するが、input / resultの契約が機械可読でない
- RelationとContract Queryは定義されているが、そのPatternで利用するQueryやinput bindingが明示されていない

```text
Application定義
       ↓
Pattern / Datasource / Bindingの解決
       ↓
Driver Scenario
       ↓
Authentication / Query / Load / Edit / Validate / Authorize / Command
       ↓
Business Result / Execution Error
       ↓
Observed Result / Evidence
```

Driverは不足する意味を埋める場所ではありません。実行中にGapを発見した場合は、意味を所有すべきApplication / Semantic / Query / Database Contract / Binding等の上位層へ戻します。

## mainへ統合済みのvertical slice

### 1. Application PatternのCanonical解決 — PR #161

`internal/driver.Resolver`から、Application IDとPattern IDを起点にPattern Kindごとの既存Canonical資産を解決できるようにしました。

- `browse`: Semantic Data / Query / Queryが保持するlogical Datasource
- `record`: Semantic Data
- `reference`: Relation / source・target Data / Patternに明示されたDatasource
- `header-detail`: `many` Relation / source・target Data
- `command`: Semantic Capability

Driverは`record`の保存先、`header-detail`のPersistence Mapping、`command`のBinding等を推測で補いません。

### 2. Capability Bindingの明示解決 — PR #162

Application + Capability identityから既存Bindingを一意に解決する入口を追加しました。

```text
order_entry / sales.order.submit
order_entry / security.authenticate
order_entry / security.authorize
```

Bindingが0件または複数件なら、列挙順等で一件を選ばず実行準備上のerrorにします。

### 3. Binding経由のCapability実行 — PR #164

解決済みBindingに対応する`capabilityinvoke.Adapter`を受け取り、既存Invocation境界からCapabilityを実行できるようにしました。

Driver本体は`legacyauth`等の具体実装を知りません。

Authentication Referenceでは次を確認しています。

- `authenticated(subject)`はBusiness Result
- credential不一致の`denied`もBusiness Result
- Adapter解決失敗やAdapter障害はExecution Error
- Capability契約にないOutcomeもExecution Error
- 実行時credentialをDriver結果へ保持しない

### 4. AuthenticationからAuthorizationへの接続 — PR #165

PR #164の共通実行入口を再利用し、Security専用の実行系をDriverへ増やさずAuthorizationまで接続しました。

```text
security.authenticate
        ↓ authenticated subject
security.authorize
        ↓
allowed / denied
```

legacy organization / role / ruleはAuthorization Adapterの内側に残します。

### 5. browse Queryの実行境界 — PR #166

`browse` Patternから解決したCanonical Queryとlogical Datasourceを、そのまま`BrowseQueryExecutor`へ渡す最小境界を追加しました。

```text
order_entry.recent_orders
        ↓
Query order_entry.recent_orders
        ↓ datasource sales
Application database sales
        ↓
sales_core / read-write
        ↓
BrowseQueryExecutor
```

native SQLをDriverへ複製せず、SQL本文から物理Databaseやtableを推測しません。実行時parameterもDriver結果へ保持しません。

### 6. PostgreSQL named parameter binding — PR #169

`:from`等のnamed parameterをPostgreSQLの`$1`, `$2`, ...へ変換する責務をProvider側へ置きました。

文字列、comment、quoted identifier、dollar-quoted body、`::` cast等をparameterと誤認しない境界を持ち、Driver CoreへDB固有placeholder処理を入れていません。

### 7. 実PostgreSQLでraw resultを観測 — PR #172

Reference Workspaceから`sales -> sales_core`を解決し、`order_entry.recent_orders`を実PostgreSQLで実行しました。

観測したraw列は次です。

```text
order_id
customer_id
ordered_at
status
total_amount
```

実行parameter値は通常Evidenceへ保存していません。

### 8. Query result mappingの明示authoring — PR #174

raw column labelとSemantic attributeを、名前の類似やPersistence Mappingから推測せずQuery authoringで明示する方式を採用しました。

```text
total_amount -> sales.order.total
```

物理列名とSemantic attributeが実際に異なるEvidenceが得られたことで、Query固有の結果射影はQuery自身が正本を持つ境界が成立しました。

### 9. Query result mappingの共通Validation — PR #175

mapping先Semantic attributeがWorkspace上に実在することを、共通`validation.Validate`から確認できるようにしました。

## CI待ちのnative Query Runtime

### runtime result mapping — PR #176

PR #176では、Executorが返すraw resultと、Canonical Queryの明示mappingを適用した結果を分けます。

```text
BrowseQueryResult
  ├─ Raw
  ├─ Mapped
  └─ Payload  // Executor固有の非secret Evidence
```

Driverは次を暗黙に補いません。

- mappingのないQuery
- 欠けたraw列
- mappingにない余分なraw列
- 重複raw列
- 一つのSemantic attributeへの重複mapping
- row幅の不一致

値のSemantic型変換やPresentation向け整形は、この段階では行いません。

実装はPRとして成立していますが、GitHub Actionsがrunner割当前に終了するため、通常CIによる検証はまだ完了していません。したがってmainへは統合していません。

### Driver入口の一本化 — PR #177

旧`internal/applicationdriver`と現行`internal/driver`の責務重複を解消し、Driver入口を`internal/driver`へ一本化する整理もPR #177で進めています。

互換wrapperや新しい意味modelは追加せず、重複した入口だけを除去します。このPRも通常CI待ちです。

## browseから先へ進んで露出した三つの意味境界

raw resultをSemantic attributeへ対応付けた後、`record / reference / header-detail`へ実値を渡そうとしたことで、設計上の不足が三つ具体化しました。

### ISSUE-000017 — Semantic Relation traversal key

Relationにはsource / target Dataとcardinalityがありますが、source instanceとtarget instanceをどのSemantic attributeで対応付けるかが不足しています。

第一Referenceでは次が必要です。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

これは`sales_core`と`master_data`を跨ぐため、物理Foreign Keyを正本にはできません。

PR #181で`semantic.RelationKey`のmodel境界、PR #182で参照整合性Validationを進めています。

Relationが所有するのはSemanticなsource / target key対応までです。どのDatabase Contract Queryを利用するか、どのQuery inputへ値を渡すかはRelationへ持ち込みません。

### ISSUE-000018 — Database Contract Query signature / realization

`master_data.customer`には`find_by_id`がありますが、従来のQuery Contractは名前しか持たず、何を受け何を返すかを機械可読に追えません。

第一Referenceでは概念上、次の契約が必要です。

```text
query find_by_id
  input id type customer.customer_id
  result customer.customer zero-or-one
```

PR #183でtyped signatureのmodel境界、PR #184で参照整合性Validationを進めています。

この層はinput / result / cardinalityとDatabase固有realizationとの境界を所有し、Pattern固有のRelation key → input bindingは所有しません。

### ISSUE-000019 — Reference Contract Query invocation

Relation keyとContract Query signatureを別々に定義しても、そのreference PatternでどのQueryを利用し、どのtarget keyをどのinputへ渡すかは一意に決まりません。

そこでApplication Pattern側に明示的な呼び出し境界を置きます。

```text
reference Pattern
  relation sales.order.customer
  datasource master
  contract master_data.customer
  query find_by_id
  bind customer.customer.customer_id -> id
```

責務は次の三層に分けます。

```text
Semantic Relation
  source-key / target-key
        ↓
Reference invocation
  Contract / Query選択
  target-key -> Query input binding
        ↓
Database Contract Query
  input / result / cardinality
```

現在の実装stackは次です。

- PR #186: Reference Contract Query invocationのCanonical model
- PR #187: Application内で閉じる基本Validation
- PR #188: `reference` Patternの`contract-query` authoring
- PR #189: Driverによるtarget Database / Contract / Queryの明示解決
- PR #193: 同じ解決経路をauthoring段階で確認する静的Validation

型一致は明示bindingの整合性を検査するためには利用しますが、bindingを自動生成する正本にはしません。

## 計画文書への反映

PR #180では、Driver実行から露出した`ISSUE-000017`と`ISSUE-000018`を計画文書へ追加しています。

PR #191ではその上に`ISSUE-000019`を追加し、三つの責務が重複しないよう既存Issueも読み直しています。

```text
ISSUE-000017
Semantic Relation key
        ↓
ISSUE-000019
Reference invocation
        ↓
ISSUE-000018
Contract Query signature / realization
```

## Gate Aから受け取っている意味

### Pattern / Data access

`order_entry`では次を解決できます。

- `browse`: `order_entry.recent_orders`
- `record`: `order_entry.order_record`
- `reference`: `sales.order.customer` + datasource `master`
- `header-detail`: `sales.order.lines -> sales.order_line many`
- `command`: `order_entry.submit_order -> sales.order.submit`
- `sales`: `sales_core` / read-write
- `master`: `master_data` / read

### 業務Validation

既存のCanonical ruleとEvaluatorを利用します。

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

Validation不成立は`NotSatisfied`、評価自体を完了できない状態はerrorです。

### Capability結果境界

ADR-0020 / Issue #77を正とします。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Invocation自体を完了できなかった状態
```

Authenticationの`denied`、Authorizationの`denied`、Validationの`NotSatisfied`はDriver自体の障害ではありません。

## 最小Scenario

`order_entry`で最終的に次を一続きに再現します。

1. Authenticationを実行してsubjectを得る
2. `browse`で受注一覧を取得する
3. `record`で一件を扱う
4. `reference`で顧客を`master_data`から解決する
5. `header-detail`でOrderLineを扱う
6. Canonical業務Validationを評価する
7. Authorizationを評価する
8. `command`で`sales.order.submit`を呼び出す
9. Business ResultとExecution Errorを区別する
10. 必要な場合はObservationから外部状態を確認する

この順序を汎用Workflow Engineへ一般化することは、第一段階の目的ではありません。

## Database境界

`sales_core`と`master_data`を意味上ひとつへ統合しません。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

`master_data`へwriteしないことをScenarioのConformanceとして確認します。

接続資格情報やConnection PoolはDriver CoreのCanonicalな意味ではありません。Reference composition / Provider側から実行環境へ与えます。

## SecretとEvidence

再現性を理由にpassword、token、credential hash、接続資格情報を通常Evidenceへ保存しません。

Evidenceへ残せる候補は次です。

- secretではない入力識別情報
- resolved Pattern
- resolved logical Datasource
- resolved Capability Binding
- resolved Database Contract Query
- Query identityとnative sourceの来歴
- Authentication / Authorization / Validationの結果
- Business Result
- Execution Errorの有無
- command acknowledgement
- observed result

Scenario EvidenceはCanonicalの正本ではありません。

## CommandとObservation

Capability invocationの結果と、外部世界で実際に成立した状態を分けます。

`sales.order.submit`では少なくとも次の三つを区別します。

1. Capability呼び出しを開始・完了できたか
2. CapabilityがどのBusiness Resultを返したか
3. 外部世界で期待した状態が実際に成立したか

必要な場合は3をObservationから確認します。

## 現在のCI制約

2026-09-03時点で、複数PRのGitHub Actionsがrunnerへ割り当てられる前に終了しています。

典型的な状態は次です。

```text
runner_id = 0
steps = [] または null
conclusion = failure
```

公開GitHub StatusではActionsはOperationalですが、Repositoryではfailed jobsの再実行でも同じ状態が再現しています。詳細はGitHub Issue #192で追跡します。

この状態をコードfailureとは扱いませんが、CI成功とも扱いません。通常runnerで`verify`と`postgres-integration`が実際に実行・成功するまで、影響を受けるPRはmainへ統合しません。

## 次の作業

CIの復旧確認と並行して、実装は互いの責務を混ぜない小さなstackとして進めます。

### 1. CIが通常runnerへ到達することを確認する

Issue #192の復旧条件に従い、基底PRから順に`verify`と`postgres-integration`を再実行します。

### 2. native Query Runtimeを完了する

PR #176でraw result / Semantic mappingのruntime境界を通常CIで確認し、`ISSUE-000016`の完了条件を再監査します。

### 3. Relation keyをauthoringまで通す

PR #181 / #182のmodel・Validationを基礎に、実Referenceへ必要なRelation keyをauthoringし、source / target attributeの来歴を追跡できる状態にします。

### 4. Contract Query signatureをauthoringまで通す

PR #183 / #184のmodel・Validationを基礎に、`master_data.customer / find_by_id`へtyped input / result / cardinalityを与えます。

### 5. Reference invocationを実Referenceへ適用する

PR #186〜#193の境界を`customer_reference`へ適用し、PatternからContract Queryまでを名前や型の推測なしで追跡できるようにします。

### 6. 三層を横断Validationする

三つのmodelが共通baseへ合流した後、少なくとも次を確認します。

- invocationの`TargetAttribute`がRelation keyのtarget側に存在する
- 必要なRelation target keyをbindingが過不足なく扱う
- binding先inputがContract Query signatureに存在する
- target attributeとinputのSemantic型が整合する
- Query result DataがRelation target Dataと一致する
- result cardinalityがreference Patternの意味と矛盾しない

型一致は明示bindingの検査に使いますが、bindingを自動生成するためには使いません。

### 7. referenceをread-onlyで実行する

Relation source instanceから明示bindingに従ってparameterを組み立て、Database Contract Query realizationへ渡します。

`master_data`のread-only境界を維持し、0件を正常な「見つからない」として扱いながら、Query Execution Errorとは分離します。

### 8. header-detailへ進む

`Order -> OrderLine`のmany Relationについて、業務上必要なSemantic keyを実Referenceから確定し、物理Foreign KeyをそのままSemanticへ逆流させずに実証します。

### 9. Canonical Validation / Authorization / CommandをScenarioへ接続する

Validation不成立やAuthorization `denied`でcommandへ進まないことを確認したうえで、`sales.order.submit`を既存Bindingから呼び出します。

### 10. Scenario Evidenceを整えGate Bを監査する

Business Result / Execution Error / acknowledgement / Observationを混同せず、Repositoryだけから同じScenarioを再現できるEvidenceを残します。

## Gate Bへ進む条件

Driverのコードが存在するだけではGate B Readyとはみなしません。

少なくとも次をReference Scenarioから確認します。

- 同じApplication定義からScenarioを再現できる
- Pattern / Query / Relation / Capability / Datasource / Binding / Database Contract Queryの解決を追跡できる
- native SQLの正本をDriverへ複製しない
- raw columnとSemantic attributeの対応を明示mappingから追跡できる
- Relation traversal keyを名前・型・物理Foreign Keyから推測しない
- Reference Contract Queryを名前・型・列挙順から推測しない
- Relation keyからQuery inputへのbindingを明示できる
- logical Datasource aliasを保持する
- `master_data`へwriteしない
- legacy authentication schemaへDriverが直接依存しない
- Validation不成立時にcommandへ進まない
- Authorization `denied`時にcommandへ進まない
- expectedなnamed OutcomeをDriver障害として扱わない
- Execution Errorを業務Outcomeへ押し込まない
- query 0件とQuery Execution Errorを混同しない
- command acknowledgementとobserved resultを混同しない
- secretを通常Evidenceへ混入させない
- Driver固有状態をCanonical Worldへ逆流させない
- 同じApplication上の意味を第一Web Targetへ渡せる
- Reference ScenarioをRepositoryだけから再現できる

## 非目標

- production-grade Workflow Runtime
- distributed execution / production HA
- 完成済みWeb / Desktop UI
- arbitrary user scripting environment
- full transaction coordinator
- external orchestratorの内製
- 汎用ORM
- generic join engine
- GraphQL風resolver framework
- Driver独自のOutcome体系
- Driver独自のValidation / Authentication / Authorization rule
- Runtime上の汎用implementation選択機構
- DB固有placeholder処理をDriver Coreへ固定すること
- 未定義のQuery result mappingをDriverが推測すること
- Relation keyやContract Query input bindingを名前・型から自動生成すること

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000016-native-query-runtime-boundary.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126 / #167 / #178 / #179 / #185 / #192
- GitHub PR #161 / #162 / #164 / #165 / #166 / #169 / #172 / #174 / #175 / #176 / #177 / #180 / #181 / #182 / #183 / #184 / #186 / #187 / #188 / #189 / #191 / #193
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
