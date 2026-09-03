# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Query実行: `ISSUE-000016` / GitHub Issue #167
- Relation traversal: `ISSUE-000017` / GitHub Issue #178
- Database Contract Query: `ISSUE-000018` / GitHub Issue #179
- Reference invocation: `ISSUE-000019` / GitHub Issue #185

## 背景

Application Driverは、production Runtimeの簡易版ではありません。設計途中のApplicationをUIなしで実際に動かし、意味の不足、曖昧な参照、実現境界の混同を早い段階で露出させるための設計検証環境です。

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

Driverは不足する意味を独自規則で補いません。実行によってGapが見つかった場合は、その意味を所有すべきApplication / Semantic / Database Contract / Realization / Binding等の上位層へ戻します。

## 現在までに成立したvertical slice

### Application PatternとCapability

`internal/driver`を正式なDriver入口として、Application IDとPattern IDからCanonical資産を解決します。

- `browse`: Semantic Data / Application Query / logical Datasource
- `record`: Semantic Data
- `reference`: Relation / source・target Data / logical Datasource / Contract Query invocation
- `header-detail`: `many` Relation / source・target Data
- `command`: Semantic Capability

Capabilityについては、Application + Capability identityからBindingを一意に解決し、既存Invocation境界から実行できます。Authentication / Authorizationも同じ共通入口を利用し、Driver本体へlegacy実装固有の規則を持ち込みません。

Business ResultとExecution Errorは分離します。AuthenticationやAuthorizationの`denied`、Validationの`NotSatisfied`をDriver障害へ変換しません。

### browse Queryと実PostgreSQL

`order_entry.recent_orders`では、logical Datasource `sales`から`sales_core`を解決し、native QueryをProvider側のExecutorへ渡す経路が成立しています。

named parameterはProvider側で安全にbindingし、Driver CoreへPostgreSQL固有placeholder処理を置きません。実PostgreSQLからraw resultを観測し、raw column labelとSemantic attributeの対応はQuery authoringで明示します。

```text
total_amount -> sales.order.total
```

runtimeではraw resultとSemantic mapping後のresultを区別し、列名の類似、SELECT順、Persistence Mappingから対応を推測しません。実行時parameter値も通常Evidenceへ保存しません。

### Semantic Relation traversal key

`reference`を実値まで進める過程で、Relationの概念的なsource / targetだけではinstance間を辿れないことが分かりました。

第一Referenceでは次を明示しています。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

Relation keyはSemantic Relation自身が所有します。物理Foreign Key、Database Query、Application Datasource、Query input名は持ち込みません。詳細は`ISSUE-000017`を正とします。

### Database Contract Query

`master_data.customer / find_by_id`にはtyped signatureを持たせています。

```tei
query find_by_id {
    input id type customer.customer_id
    result customer.customer zero-or-one
}
```

Database Contract Queryはinput / result / cardinalityを所有し、Database固有の実現方法とは分離します。Queryを実現する対象identity、専用の`contract-query-executor` Capability、Environment + Database instance + target + executorの明示Bindingも導入しました。

詳細は`ISSUE-000018`を正とします。

### Reference Contract Query invocation

Application Pattern側では、利用するContract / QueryとRelation target keyからQuery inputへのbindingを明示します。

```tei
contract-query {
    contract master_data.customer
    query find_by_id
    input customer.customer.customer_id -> id
}
```

これにより、Relation key、Reference invocation、Database Contract Query signatureを次の三層へ分離しています。

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

三層は横断Validationされ、target attribute、Query input、Semantic型、result Data、cardinalityの不整合をauthoring段階で検出できます。

runtimeでは、`browse`でmapping済みのsource rowからRelation keyと明示bindingだけを使ってContract Query parameterを構築できます。

### Environment runtime composition

第一Referenceはlogical DatasourceからEnvironment上のDatabase instanceを解決し、明示Bindingを通してexecutor Capabilityとruntime実装へ接続できます。

```text
order_entry.customer_reference
        ↓ datasource master
order_entry.master
        ↓ Environment binding
development / master_data.dev
        ↓ ContractQueryExecutorBinding
master_data/master_data.customer/find_by_id
        ↓
database.query.oracle
        ↓ runtime registry
ContractQueryExecutor
```

Environment、Database instance、executor、runtime implementationをprovider名や配置から推測しません。Binding authoring sourceはWorkspace Loaderから読み込まれ、Source Indexへ来歴を残します。

## Driver入口の一本化

旧`internal/applicationdriver`は、現在の`internal/driver`とPattern解決・Scenario組み立ての責務が重複していました。

現行Driverへ必要な機能が集約されたため、互換wrapperを追加せず旧packageを削除し、Driver入口を`internal/driver`へ一本化します。今後、新しいScenario実装が旧入口へ分岐する状態を残しません。

## 現在の主要Gap

### 1. Contract Query realization source

Reference invocationからexecutor Capabilityまでは明示的に解決できますが、`database.query.oracle`が利用する具体的なQuery realization sourceと、そのauthoring / provenance境界はまだ完了していません。

次に必要なのは、SQLやDatabase APIをDriver Coreへ持ち込まず、Contract Query targetと物理的な実現sourceを明示的に対応付けることです。

### 2. physical resultからSemantic rowへのmapping

Contract Query Executorの結果境界はSemantic rowを受け取る形まで定義しています。実DBのphysical column / valueからSemantic rowへ変換する明示mappingを、実現層に追加する必要があります。

列名や型からの推測は行いません。

### 3. `Order -> Customer` Referenceのend-to-end実証

Workspace読込み、Relation key、Reference invocation、parameter準備、Environment composition、executor解決を、実read結果まで一続きに通します。

0件は正常な「見つからない」、接続・Query実行失敗はExecution Errorとして区別します。`master_data`のread-only境界も維持します。

### 4. header-detail

`sales.order.lines`は`many` Relationとして定義されていますが、現行`sales.order_line`には親Order identityに相当するSemantic attributeがありません。

Driverを動かすためだけに物理Foreign KeyをSemanticへコピーせず、Reference業務意味として必要性を確認したうえでRelation keyを確定し、runtime traversalへ接続します。

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

この順序を汎用Workflow Engineへ一般化することは第一段階の目的ではありません。

## 次の作業

1. Contract Query realization sourceのidentityとauthoring境界を追加する
2. `master_data.customer / find_by_id`へ具体的なrealization sourceを接続する
3. physical resultからSemantic rowへの明示mappingを追加する
4. realization sourceの来歴をSource Indexへ接続する
5. `Order -> Customer` Referenceを実readまで一続きに実証する
6. `sales.order.lines`のSemantic key要件をReferenceから確定する
7. header-detailをruntime traversalへ接続する
8. Canonical Validation / Authorization / Commandを一つのScenarioへ接続する
9. Scenario EvidenceをRepositoryだけから再現できる形へ整える
10. Gate Bを監査する

## Gate Bへ進む条件

Driverのコードが存在するだけではGate B Readyとはみなしません。少なくとも次をReference Scenarioから確認します。

- 同じApplication定義からScenarioを再現できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingの解決を追跡できる
- native QueryやContract Queryの物理実現をDriver Coreへ複製しない
- raw columnとSemantic attributeの対応を明示mappingから追跡できる
- Relation traversal keyを名前・型・物理Foreign Keyから推測しない
- Reference Contract QueryをQuery名・型・列挙順から推測しない
- Relation keyからQuery inputへのbindingを明示できる
- Query 0件とQuery Execution Errorを混同しない
- logical Datasource aliasとEnvironment上のinstance境界を保持する
- `master_data`へwriteしない
- Validation不成立時にcommandへ進まない
- Authorization `denied`時にcommandへ進まない
- expectedなNamed OutcomeをDriver障害として扱わない
- Execution Errorを業務Outcomeへ押し込まない
- command acknowledgementとobserved resultを混同しない
- secretやruntime parameter値を通常Evidenceへ混入させない
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
- Driver独自のOutcome体系
- Driver独自のValidation / Authentication / Authorization rule
- Runtime上の汎用implementation選択機構
- DB固有placeholder処理をDriver Coreへ固定すること
- 未定義のQuery / Relation / BindingをDriverが推測すること

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000016-native-query-runtime-boundary.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- GitHub Issue #69 / #77 / #124 / #167 / #178 / #179 / #185
- ADR-0019 / ADR-0020
- RQ-005 / RQ-008 / RQ-009 / RQ-011 / RQ-012 / RQ-014 / RQ-017 / RQ-018 / RQ-021
