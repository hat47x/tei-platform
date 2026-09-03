# ISSUE-000019: reference PatternからDatabase Contract Queryへの明示的な呼び出し境界を定める

- 状態: In Progress
- GitHub Issue: #185
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連Issue: `ISSUE-000017` / GitHub Issue #178、`ISSUE-000018` / GitHub Issue #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから外部観測DatabaseのSemantic Dataを取得する経路を実際に通すと、Semantic Relationのkey対応とDatabase Contract Queryのsignatureだけでは、両者を安全につなげられません。

第一Referenceでは、次の三層を分離して扱います。

```text
Semantic Relation
  sales.order.customer_id
      -> customer.customer.customer_id
        ↓
Reference invocation
  master_data.customer / find_by_id
  customer.customer.customer_id -> input id
        ↓
Database Contract Query
  input id : customer.customer_id
  result customer.customer zero-or-one
```

Relationはinstance間のSemantic key対応を所有し、Database Contract Queryはinput / result / cardinalityを所有します。このIssueが所有するのは、そのApplication Patternで利用するContract / Queryと、Relation target keyからQuery inputへの明示bindingです。

DriverがQuery名、input名、Semantic型、Contract内のQuery数などから呼び出し先を推測する方針は採りません。

## 現在のReference authoring

`order_entry.customer_reference`では、Relationとlogical Datasourceに加え、利用するContract Queryとinput bindingを明示しています。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
```

既存の`Pattern.Query`はApplication-ownedなQuery identityを表します。Database-ownedなContract Queryは別のidentity体系であるため、同じfieldへ載せません。

```text
Application Query
  order_entry.recent_orders

Database Contract Query
  Database: master_data
  Contract: master_data.customer
  Query: find_by_id
```

## ここまでに実装したこと

### Canonical modelとauthoring

`reference` Patternは、次を明示的に保持できます。

```text
ContractQueryInvocation
  ├─ Contract
  ├─ Query
  └─ Inputs[]
       ├─ TargetAttribute
       └─ Input
```

`contract-query` authoring blockからこのmodelを構築し、未定義の既存Patternには暗黙の呼び出しを追加しません。

### Application境界内のValidation

次を静的に検査します。

- `reference` Pattern以外でContract Query invocationを使わない
- Contract / Query identityを明示する
- Datasourceを明示する
- ApplicationがそのDatasourceで要求しているContractだけを利用する
- 同じtarget attributeを複数inputへ割り当てない
- 同じQuery inputへ複数target attributeを割り当てない
- target Database / Contract / Queryを一意に解決できる
- compatible majorを無視しない

別Databaseの同名Contractや同名Queryへfallbackしません。

### Relation key / Query signatureとの横断Validation

現在は三つのmodelを共通のValidation境界で照合できます。

- bindingの`TargetAttribute`がRelation keyのtarget側に存在する
- 必要なRelation target keyをbindingが過不足なく扱う
- binding先のQuery inputがsignatureに存在する
- Relation target attributeとQuery inputのSemantic型identityが整合する
- Query result DataがRelation target Dataと整合する
- result cardinalityがreference Patternの意味と矛盾しない

型一致は明示bindingの検証に利用しますが、bindingを自動生成する根拠にはしません。

### Runtime parameterの組み立て

`browse`でSemantic attributeへmapping済みのrowから、Relation keyと明示bindingだけを使ってContract Query parameterを組み立てます。

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

値の型や名前を見て別のinputへ補完しません。runtime parameter値はCanonical Worldや通常Evidenceへ保存しません。

### Environment runtime composition

logical DatasourceからEnvironment上のDatabase instanceを解決し、Contract Query targetに対応する明示Bindingを通してexecutor Capabilityとruntime実装へ接続できます。

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

Environment、Database instance、executor Capability、runtime implementationのいずれもprovider名や配置から暗黙選択しません。

### Result境界

provider非依存の`ContractQueryExecutor`からSemantic rowを受け取り、`zero-or-one`では次を区別します。

- 0件: 正常な「見つからない」
- 1件: 正常な取得
- 2件以上: Contract違反
- 接続失敗・Query実行失敗: Execution Error

正常な0件をExecution Errorへ変換しません。

## 残る実証対象

Reference invocation自体の選択・binding・parameter準備・Environment compositionまでは実装できました。残る主要な境界は`ISSUE-000018`と共有します。

1. `database.query.oracle`を具体的なQuery realization sourceへ明示的に接続する
2. physical resultからSemantic rowへのmappingを明示する
3. Query realization sourceの来歴をSource Indexから追跡できるようにする
4. 第一ReferenceをWorkspace読込みから実read結果まで一続きに実証する

これらをApplication PatternやSemantic Relationへ逆流させず、Database固有の実現境界として進めます。

## 完了条件

- [x] reference Patternから利用するDatabase Contract Queryを名前推測なしで特定できる
- [x] Application QueryとDatabase Contract Queryのidentityを混同しない
- [x] Relation target keyとQuery inputの対応を明示できる
- [x] 同じSemantic型のinputが複数あってもbindingを一意に保てる
- [x] Applicationが要求していないContractを呼び出さない
- [x] compatible majorとtarget Database境界を維持する
- [x] Relation key / Query signature / result Data / cardinalityを横断Validationできる
- [x] Relation source instanceの値を明示bindingに従ってQuery parameterへ渡せる
- [x] Environment上のDatabase instanceとexecutorを明示Bindingから解決できる
- [x] 0件とExecution Errorを区別するprovider非依存result境界を持つ
- [ ] concrete Query realization sourceを明示的に解決できる
- [ ] physical resultからSemantic rowへのmappingを明示できる
- [ ] realization sourceの来歴を追跡できる
- [ ] `Order -> Customer` Referenceを実readまで一続きに実証できる

## 非目標

- 汎用ORM
- Query名や型からのContract Query自動選択
- generic join engine
- SQLやDatabase APIをApplication Patternへ埋め込むこと
- Database Contract QueryをすべてSemantic Capabilityへ統一すること
- Target UI固有のnavigation設計
- production向けcache / batching / Connection Pool設計

## 関連

- GitHub Issue #69 / #178 / #179 / #185
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
- `internal/application/model.go`
- `internal/validation/reference_contract_query_cross.go`
- `internal/driver/reference_contract_query.go`
- `internal/driver/reference_contract_query_parameters.go`
- `internal/driver/reference_contract_query_environment.go`
