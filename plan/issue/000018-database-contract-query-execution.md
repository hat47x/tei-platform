# ISSUE-000018: Database Contract Queryのinput・result・実行境界を明示する

- 状態: In Progress
- GitHub Issue: #179
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / GitHub Issue #178
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから`master_data`のCustomerを取得する経路を実際に通した結果、Semantic Relationのkey対応とは別に、Database Contract Query自身の契約と実現境界が必要であることが明確になりました。

第一Referenceでは、次の三層を分離して扱います。

```text
Semantic Relation
  sales.order.customer_id
      -> customer.customer.customer_id
        ↓
Reference invocation
  customer.customer.customer_id
      -> Query input id
        ↓
Database Contract Query
  master_data.customer / find_by_id
  input id : customer.customer_id
  result customer.customer zero-or-one
```

Relationはinstanceを辿るSemantic key対応を所有し、Applicationのreference Patternは利用するContract Queryとinput bindingを所有します。Database Contract Queryは、受け取るinput、返すSemantic Data、result cardinalityを所有します。

この分離により、RelationへSQLやDatabase APIを持ち込まず、Database Contract Queryへ`Order.customer`という業務Relationの意味を重複させず、Driverにも名前・型・列順による推測を持たせません。

## ここまでに確定したこと

### Query signature

`database.QueryContract`には、次の最小契約を追加しています。

```text
QueryContract
  ├─ name
  ├─ inputs[]
  │    ├─ name
  │    └─ Semantic type
  ├─ result Semantic Data
  └─ result cardinality
```

第一Referenceの`find_by_id`は次の契約です。

```tei
query find_by_id {
    input id type customer.customer_id
    result customer.customer zero-or-one
}
```

`zero-or-one`は、Semantic Relationの`one`をoptionalへ変更するものではありません。`master_data`がexternal-observedなread境界であるため、外部データの欠落をExecution Errorへ変換せず、正常な「見つからない」として表現するためです。

### Reference invocation

Applicationのreference Patternには、利用するContract QueryとRelation target keyからQuery inputへの対応を明示しました。

```tei
contract-query {
    contract master_data.customer
    query find_by_id
    input customer.customer.customer_id -> id
}
```

DriverはSemantic型が一致するだけのinputや、`find_by_id`という名前からこの対応を推測しません。

### Runtime parameter準備

`browse`でSemantic attributeへ明示mappingした一行から、次の経路だけを使ってruntime parameterを作る境界まで実装しました。

```text
MappedResult
  sales.order.customer_id = 42
        ↓ Relation key
  customer.customer.customer_id
        ↓ explicit invocation binding
  Query input id
        ↓
  Parameters["id"] = 42
```

値はこの境界では型変換せず、そのまま渡します。またruntime parameter値はCanonical Worldや通常Evidenceへ保存しません。

### Provider非依存の実行結果境界

Driver Coreには`ContractQueryExecutor`を置き、Oracle SQL、API、stored procedure等の実装詳細を持ち込まないようにしました。

```text
PreparedReferenceContractQuery
        ↓
ContractQueryExecutor
        ↓
ContractQueryResult
  ├─ Semantic rows
  └─ provider固有の非secret Payload
```

第一Referenceの`zero-or-one`では、次を区別します。

- 0件: 正常な「見つからない」
- 1件: 正常な取得結果
- 2件以上: Contract違反
- 接続失敗・SQL/API失敗等: Execution Error

返されたrowについては、attributeがQuery result Dataに属するSemantic attributeであることもDriver側で確認します。物理column名やprovider固有row表現は、この境界より内側へ持ち込みません。

### Query realization target identity

Database-ownedなQueryを既存の`Application + Capability` identityへ変換せず、次の三つ組を独立したtarget identityとして追加しました。

```text
ContractQueryTarget
  Database = master_data
  Contract = master_data.customer
  Query    = find_by_id
```

安定identityは次です。

```text
master_data/master_data.customer/find_by_id
```

このidentityにはApplication ID、Capability ID、provider、environment、file pathを含めません。実装方式や配置が変わっても、何を実現しているかというtarget identityを維持します。

### Executor Capabilityと明示Binding

既存`database-provider` CapabilityをContract Query実行能力として暗黙に扱わないため、Plugin Capability kindとして`contract-query-executor`を独立させました。

第一ReferenceのOracle pluginは、概念上次の二つの能力を別々に宣言します。

```tei
provides database-provider database.oracle {
    provider oracle
}

provides contract-query-executor database.query.oracle {
    provider oracle
}
```

さらに、実行時選択のmodelとして`ContractQueryExecutorBinding`を追加しました。

```text
Database instance
  master_data.dev
        +
ContractQueryTarget
  master_data/master_data.customer/find_by_id
        ↓ explicit binding
Executor Capability
  database.query.oracle
```

解決時には次を検査します。

- Bindingが0件ならprovider一致から推測せず未解決とする
- 同じinstance + targetへのBindingが複数なら曖昧として拒否する
- Database instanceのdefinitionとtarget Databaseが一致すること
- executorが`contract-query-executor` kindとして一意に解決できること
- executor CapabilityがinstanceのDatabase providerを明示的にサポートすること

Environment authoring構文は、このmodel境界を実Referenceで確認してから固定します。巨大な`ParseEnvironment`へ先回りの構文を追加しないことを優先しています。

## 現在露出しているGap

Contract Queryのsignature、reference invocation、runtime parameter、result cardinality、target identity、executor capability、明示Binding modelまでは分離できました。

残っている主要Gapは次です。

1. `ContractQueryExecutorBinding`のauthoring source
2. authoring sourceからEnvironment runtime compositionへ読み込む経路
3. `database.query.oracle`を実際のQuery realization sourceへ接続する方法
4. physical resultからSemantic rowへ変換する明示mapping
5. Oracle実接続、またはOracle依存を避けたReference adapterによるend-to-end Evidence
6. realization sourceのSource Index / provenance

ここでも、Oracleの物理列名やSQLをSemantic Relation、Database Contract signature、Driver Coreへ持ち込まないことを維持します。

## 次の実証順

1. `ContractQueryExecutorBinding` modelと選択Conformanceを固定する
2. 第一ReferenceのOracle pluginへ`contract-query-executor` Capabilityを明示する
3. Binding authoring sourceの最小構文を、既存Environment authoring体系との整合を確認して追加する
4. `database.query.oracle`の最小realization sourceを追加する
5. physical result → Semantic rowの明示mapping境界を追加する
6. Oracle read implementationを追加する、またはOracle依存を避けたReference adapterで実行境界を先に実証する
7. authoring source / realization sourceの来歴を確認する
8. `reference` Patternを一続きに実行する

Oracle driverや実DB接続がCI環境で安定して用意できない場合でも、Coreの境界をOracle固有実装へ引きずらないため、まずprovider非依存Conformanceを成立させます。

## 完了条件

- [x] `find_by_id`のinputを名前から推測しない
- [x] inputがどのSemantic typeを表すか追跡できる
- [x] resultが`customer.customer`であることを契約から追跡できる
- [x] result cardinalityを`zero-or-one`として明示できる
- [x] Database Contract Queryとrealization target identityを分離できる
- [x] Database固有SQL / API / stored procedureをDriver Coreへ持ち込まない
- [x] 0件とExecution Errorをruntimeで区別する
- [x] 1件とcardinality違反をruntimeで区別する
- [x] Database Contract Query executorを`database-provider`とは別Capabilityとして表現できる
- [x] provider一致だけではexecutorを自動選択しない
- [x] instance + target + executorの明示Bindingをmodelとして表現し、一意性とprovider整合を検査できる
- [ ] Bindingをauthoring sourceから読み込める
- [ ] `master_data`をread-onlyのまま実実行できる
- [ ] physical resultを明示mappingからSemantic rowへ変換できる
- [ ] authoring source / realization sourceの来歴を追跡できる
- [x] Relation keyを暗黙変換なしでquery inputへ渡せる

## 非目標

- 汎用ORM
- 全Database Contract Queryを覆う完全な型体系
- production Connection Pool / HA
- external Database schemaのTEI管理化
- Database queryをすべてSemantic Capabilityへ統一すること
- Application Queryとの自動統合
- Target UI向けデータ取得API
- Oracle固有の高度な接続・HA機能をこのIssueだけで完成させること

## 関連

- GitHub Issue #69 / #167 / #178 / #179 / #185 / #192
- PR #176 / #183 / #184 / #193 / #195 / #197 / #201 / #205 / #206
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/plugins/tei-database-oracle/1.0.0/plugin.tei`
- `internal/database/model.go`
- `internal/driver/contract_query_execution.go`
- `internal/driver/reference_contract_query.go`
- `internal/driver/reference_contract_query_parameters.go`
- `internal/realization/contract_query.go`
- `internal/realization/contract_query_binding.go`
