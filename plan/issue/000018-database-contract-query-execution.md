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

`database.QueryContract`には、既に次の最小契約を追加しています。

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

`zero-or-one`は、Semantic Relationの`one`をoptionalへ変更するものではありません。`master_data`がexternal-observedなread境界であるため、外部データの欠落をExecution Errorへ変換せず、正常な「見つからない」として表現できるようにするためです。

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

## 現在露出しているGap: Query realization

Contract Queryの「何を受け、何を返すか」は明示できましたが、`master_data.customer / find_by_id`を**どの実装で実行するか**はまだCanonicalに存在しません。

Repository内を再確認した結果、次が分かりました。

- `internal/driver/queryruntime`の実Query runtimeは現時点でPostgreSQLのみ
- 第一Referenceの`master_data`はOracle provider
- `internal/database/realization`のOracle対応はPersistence Mappingから物理Schemaを構成するための`PhysicalRealizationPolicy`であり、Contract Query read runtimeではない
- 既存`CustomGoBinding`は`Application + Capability`をidentityとするため、Database-owned Queryをそのまま流用すると意味階層が混ざる

したがって、Contract Query realizationは少なくとも`Database / Contract / Query`を起点とする独立したidentityとして扱う必要があります。

## 次に追加する最小実行境界

まずDriver Coreには、Oracle SQLや接続ライブラリを直接持ち込まず、provider非依存の実行境界だけを置きます。

概念上は次の責務です。

```text
PreparedReferenceContractQuery
        ↓
ContractQueryExecutor
  - resolved Database / Contract / Query
  - runtime parameters
        ↓
ContractQueryResult
  - zero rows
  - one row
  - execution error
```

第一Referenceの`zero-or-one`では、次を明確に区別します。

- 0件: 正常な「見つからない」
- 1件: 正常な取得結果
- 2件以上: Contract違反
- 接続失敗・SQL/API失敗等: Execution Error

この境界ではまだOracle SQL自体を固定しません。どのprovider implementationを選ぶか、どのSQL/API/stored procedureへ接続するかは、別のrealization identityへ分離します。

## Query realization identityの要件

第一段階では、少なくとも次を満たす必要があります。

- Database IDを含む
- Contract IDを含む
- Query名を含む
- Application IDやCapability IDへ変換しない
- 同じContract Queryに複数realizationが存在する場合も、環境・provider等の明示情報から選択できる余地を残す
- SQL/API/stored procedure等の詳細をDatabase Contract signatureへ混ぜない
- authoring sourceとrealization sourceを追跡できる

## 実証順

1. provider非依存の`ContractQueryExecutor`境界を追加する
2. `zero-or-one`のruntime result表現を追加する
3. 0件 / 1件 / 複数件 / Execution ErrorをConformanceで固定する
4. realization identityの最小modelを追加する
5. 第一Referenceに`master_data.customer / find_by_id`のrealization sourceを明示する
6. Oracle read implementationを追加する、またはOracle依存を避けたReference adapterで実行境界を先に実証する
7. authoring source / realization sourceの来歴を確認する
8. `reference` Patternを一続きに実行する

Oracle driverや実DB接続がCI環境で安定して用意できない場合でも、Coreの境界をOracle固有実装へ引きずらないため、まずprovider非依存Conformanceを成立させます。

## 完了条件

- [x] `find_by_id`のinputを名前から推測しない
- [x] inputがどのSemantic typeを表すか追跡できる
- [x] resultが`customer.customer`であることを契約から追跡できる
- [x] result cardinalityを`zero-or-one`として明示できる
- [ ] Database Contract Queryとrealization方式を分離できる
- [ ] `master_data`をread-onlyのまま利用する
- [ ] Database固有SQL / API / stored procedureをDriver Coreへ持ち込まない
- [ ] 0件とExecution Errorをruntimeで区別する
- [ ] 1件とcardinality違反をruntimeで区別する
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
- PR #176 / #183 / #184 / #193 / #195 / #197 / #201 / #205
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `internal/database/model.go`
- `internal/driver/reference_contract_query.go`
- `internal/driver/reference_contract_query_parameters.go`
- `internal/driver/queryruntime/postgres`
- `internal/database/realization`
