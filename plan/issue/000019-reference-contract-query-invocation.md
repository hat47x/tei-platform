# ISSUE-000019: reference PatternからDatabase Contract Queryへの明示的な呼び出し境界を定める

- 状態: Done
- GitHub Issue: #185（完了）
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連Issue: `ISSUE-000017` / GitHub Issue #178、`ISSUE-000018` / GitHub Issue #179、`ISSUE-000020` / GitHub Issue #222、`ISSUE-000022` / GitHub Issue #214
- 後続Evidence: GitHub Issue #214 / #310、Gate B総合監査 #273
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから外部観測DatabaseのSemantic Dataを取得するには、Semantic Relationのkey対応とDatabase Contract Queryのsignatureだけでは足りません。

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

Relationはinstance間のSemantic key対応を所有し、Database Contract Queryはinput / result / cardinalityを所有します。このIssueが所有するのは、そのApplication Patternで**どのContract Queryを利用し、Relation target keyの値をどのQuery inputへ渡すか**という明示的な呼び出し境界です。

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

## 実装済みの境界

### Canonical modelとauthoring

`reference` Patternは、次の呼び出し情報を明示的に保持できます。

```text
ContractQueryInvocation
  ├─ Contract
  ├─ Query
  └─ Inputs[]
       ├─ TargetAttribute
       └─ Input
```

`contract-query` authoring blockからこのmodelを構築します。呼び出し情報を持たない既存Patternへ、暗黙のContract Queryを補完することはありません。

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

別Databaseの同名Contractや、同じContract名・Query名を持つ別定義へfallbackしません。

### Relation key / Query signatureとの横断Validation

Semantic Relation、Reference invocation、Database Contract Query signatureを共通のValidation経路で照合します。

- bindingの`TargetAttribute`がRelation keyのtarget側に存在する
- 必要なRelation target keyをbindingが過不足なく扱う
- binding先のQuery inputがsignatureに存在する
- Relation target attributeとQuery inputのSemantic型identityが整合する
- Query result DataがRelation target Dataと整合する
- result cardinalityがreference Patternの意味と矛盾しない

型一致は明示bindingの妥当性確認には使いますが、bindingを自動生成する根拠にはしません。

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

値の型や名前を見て別inputへ補完しません。runtime parameter値はCanonical Worldや通常Evidenceへ保存しません。

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
        ↓ instance-aware runtime registry
ContractQueryExecutor
```

Environment、Database instance、executor Capability、runtime implementationのいずれもprovider名や配置から暗黙選択しません。

runtime Registryは`ISSUE-000020`でEnvironment + Database instance + executor Capability identityを保持するようにし、同一Capabilityを共有する別Database instanceへfallbackしない境界まで実装しています。

### Query realizationとresult mapping

このIssueの初期段階では未実装だったDatabase Contract Queryの物理realizationも、`ISSUE-000018`側で境界を追加しています。

```text
ContractQueryTarget
  Database / Contract / Query
        ↓
ContractQuerySQLRealization
  Contract input -> physical parameter
  native SQL source
  raw result column -> Semantic attribute
        ↓
ContractQueryResult
```

SQL realizationは通常Workspaceから読み込め、authoring sourceをSource Indexで追跡できます。raw resultは明示されたresult-column mappingだけでSemantic rowへ変換し、列名の類似やSELECT順序から意味を補完しません。

### Result境界

`zero-or-one`では次を区別します。

- 0件: 正常な「見つからない」
- 1件: 正常な取得
- 2件以上: Contract違反
- 接続失敗・Query実行失敗・row取得失敗: Execution Error

正常な0件をExecution Errorへ変換せず、Execution Errorを0件として隠しません。

## Invocation境界としての現在地

Reference invocation固有の設計・model・authoring・Validation・parameter準備・Environment compositionは、Evidence用convergence上で一通り接続できました。

当初このIssueに残していた次の項目は、現在は実装済みです。

- concrete Query realizationをDatabase / Contract / Query identityから明示的に解決する仕組み
- physical resultからSemantic rowへの明示mapping
- Query realization authoring sourceのSource Index / provenance
- Environment / Database instance / executor Capabilityを保ったruntime composition

したがって、今後の主な未完了事項はReference invocationの設計不足ではなく、第一Reference固有の**観測された物理Evidence**と最終E2Eです。

## 後続Issueへ委ねる範囲

### `ISSUE-000018`: Database Contract Query実行境界

Database Contract Queryのsignature、realization、executor、result cardinality、provider境界を引き続き統括します。

### `ISSUE-000020`: runtime Database instance identity

同じexecutor Capabilityを複数Database instanceが共有しても、runtime接続先を取り違えない境界を扱います。

### `ISSUE-000021`: Oracle `database/sql` Runner

明示済みnative SQLとphysical parameterを、注入済み`*sql.DB`へ渡すOracle向け最小Runnerを扱います。

### `ISSUE-000022`: external `master_data`の物理Evidence

実際のtable / view / column、native SQL、physical parameter、raw result列、read-only強制、provenance、実Oracle driverとのintegrationを扱います。

物理Evidenceが存在しない状態で、Invocation境界からOracle SQLやobject名を生成しません。

## 完了判定と後続Evidence

Reference invocationの設計・authoring・Validation・runtime compositionは成立済みであり、本Issueは完了とします。

一方、次は本Issueの実装不足ではなくGate Bの後続Evidenceとして別に追跡します。

- #214: actual Oracle physical / read-only / driver integration / `Order -> Customer`実read E2E
- #310: deterministic Repository / normal Go runner Evidence
- #273: Gate B全体のEvidence監査

GitHub Actionsは現在Repositoryの通常検証経路として使用していません。未実行をgreenとは扱いませんが、Actions停止そのものを本Issueの未完了条件にはしません。将来CIを再導入する場合は、その時点のCI検証Issueで継続的なregression確認を扱います。

## 実装境界の完了条件

- [x] reference Patternから利用するDatabase Contract Queryを名前推測なしで特定できる
- [x] Application QueryとDatabase Contract Queryのidentityを混同しない
- [x] Relation target keyとQuery inputの対応を明示できる
- [x] 同じSemantic型のinputが複数あってもbindingを一意に保てる
- [x] Applicationが要求していないContractを呼び出さない
- [x] compatible majorとtarget Database境界を維持する
- [x] Relation key / Query signature / result Data / cardinalityを横断Validationできる
- [x] Relation source instanceの値を明示bindingに従ってQuery parameterへ渡せる
- [x] Environment上のDatabase instanceとexecutorを明示Bindingから解決できる
- [x] runtime接続先でDatabase instance identityを保持できる
- [x] 0件とExecution Errorを区別するprovider非依存result境界を持つ
- [x] Query realizationをDatabase / Contract / Query identityから明示的に解決できる
- [x] physical resultからSemantic rowへのmappingを明示できる
- [x] realization sourceの来歴をSource Indexから追跡できる
- [x] Pattern、Contract、Query、executor Bindingのauthoring sourceを追跡できる

### 後続Evidence（本Issueの完了条件外）

- #310で関連Conformance / fail-closed境界を通常Go runnerから再現する
- #214の物理Evidenceを用いて`Order -> Customer`実read E2Eを確認する

最後の実read E2EはInvocationの選択規則を追加するためのものではなく、既に確定した境界が外部観測資産でも成立することを確認するEvidenceです。

## 非目標

- 汎用ORM
- Query名や型からのContract Query自動選択
- generic join engine
- SQLやDatabase APIをApplication Patternへ埋め込むこと
- Database Contract QueryをすべてSemantic Capabilityへ統一すること
- Target UI固有のnavigation設計
- production向けcache / batching / Connection Pool設計
- 観測されていない外部Databaseの物理情報を推測すること

## 関連

- GitHub Issue #69 / #178 / #179 / #185 / #214 / #222 / #224 / #273 / #310
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000021-oracle-contract-query-sql-runner.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
- `internal/application/model.go`
- `internal/validation/reference_contract_query_cross.go`
- `internal/driver/reference_contract_query.go`
- `internal/driver/reference_contract_query_parameters.go`
- `internal/driver/reference_contract_query_environment.go`
- `internal/realization/contract_query_sql.go`