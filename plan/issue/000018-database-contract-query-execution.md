# ISSUE-000018: Database Contract Queryのinput・result・実行境界を明示する

- 状態: Done
- GitHub Issue: #179（完了）
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / GitHub Issue #178（完了）
- 後続Issue: `ISSUE-000019` / GitHub Issue #185（完了）
- runtime identity: `ISSUE-000020` / GitHub Issue #222（完了）
- Oracle Runner: `ISSUE-000021` / GitHub Issue #224（完了）
- actual Oracle Evidence: `ISSUE-000022` / GitHub Issue #214
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 目的

Database Contract Queryについて、名前から実行方法を推測せず、次を別々の明示契約として保持できる境界を定めます。

```text
Database Contract Query signature
        ↓
Application側の明示 invocation
        ↓
provider-independent SQL realization
        ↓
Environment / Database instance / executor Capability
        ↓
provider Runner
```

Semantic Relation、Database read契約、物理SQL、runtime接続先を一つのmodelへ混ぜません。

## 成立した境界

### Query signature

Database Contract Queryは少なくとも次を明示します。

- Query identity
- input identity
- inputのSemantic type
- result Semantic Data
- cardinality

第一Referenceでは`master_data.customer / find_by_id`を用います。

```text
input id
  type customer.customer_id
result customer.customer
cardinality zero-or-one
```

Driverが`find_by_id`という名前や型一致だけからinput / resultを補いません。

### Application invocationとの分離

どのApplication PatternからどのContract Queryを利用し、Relation側の値をどのQuery inputへ渡すかは`ISSUE-000019`の明示invocationが所有します。

Database Contract Query自身は、Application固有のPatternやRelationを知りません。

### SQL realizationとの分離

Contract Queryの意味契約と、SQL / parameter / raw result columnは別realizationに分離します。

```text
Contract input
    ↓ explicit mapping
physical parameter
    ↓
native SQL source
    ↓
raw result columns
    ↓ explicit mapping
Semantic result
```

次を禁止します。

- Contract input名を物理parameter名として暗黙流用すること
- SELECT列順からSemantic resultを推測すること
- raw column名の類似からSemantic attributeを推測すること
- Persistence MappingからQuery result mappingを補完すること

### Validation

通常のWorkspace Validationから少なくとも次を検査できます。

- input identity / type
- result Data
- cardinality
- SQL realization target
- input → physical parameter mapping
- raw result → Semantic attribute mapping
- duplicate / missing mapping
- provider / target整合性

Validation Errorをruntimeで暗黙補完しません。

### runtime composition

Environment上で、次のidentityを明示したままexecutorを解決します。

```text
Environment
  + Database instance
  + executor Capability
```

Capability IDだけへ縮退せず、同じCapabilityを複数Database instanceで利用する場合も接続先を取り違えません。

この境界は`ISSUE-000020` / #222で完了しています。

### provider Runner

provider RunnerはContractの業務意味やApplicationの意味を所有しません。

- 上位から渡されたnative sourceを実行する
- explicit physical parameterをdriver引数へ渡す
- raw columns / row valuesを返す
- 0 rowとExecution Errorを区別する

Oracle向け最小`database/sql` Runnerは`ISSUE-000021` / #224で完了しています。

## 0件とExecution Error

第一Referenceでは`zero-or-one`を使います。

- 0 row: 正常なnot-found
- 1 row: 正常なSemantic result
- 複数row: cardinality違反
- query実行失敗: Execution Error
- row scan失敗: Execution Error

0件をエラーに変えず、Execution Errorを0件へ畳みません。

## external-observed Databaseとの関係

`master_data`はTEIが物理Schemaを所有するmanaged-here Databaseではありません。

したがって本Issueで成立したContract Query model / runtime境界を理由に、実Oracle object、column、parameter、SQL、権限を推測しません。

actual physical facts / provenance / read-only強制 / Oracle driver integrationは`ISSUE-000022` / #214のEvidence責務です。

PR #288 / #300 / #306では、actual external Evidenceを非Canonicalのまま受け取り、明示authoring済みrealizationと照合するfail-closed境界も成立しています。

Evidenceが不足していることを、本IssueのContract Query設計未完了とは扱いません。

## 完了条件

- [x] Query inputを名前から推測しない
- [x] input identity / Semantic typeを追跡できる
- [x] result Semantic Dataを契約から追跡できる
- [x] 必要なcardinalityを明示できる
- [x] Database Contract Queryとphysical realizationを分離した
- [x] input → physical parameter mappingを明示した
- [x] raw result → Semantic attribute mappingを明示した
- [x] 0件とExecution Errorを区別した
- [x] provider固有SQLをDriver Coreへ持ち込まない
- [x] Environment / Database instance / executor Capabilityを保持してruntimeへ接続できる
- [x] Oracle向け最小Runnerを別責務で実装した
- [x] authoring / realization sourceへ追跡できる
- [x] actual Oracle Evidenceを#214へ分離した

## 非目標

- 汎用ORM
- 全Database Contract Queryの完全な型体系
- production Connection Pool / HA
- external Database SchemaのTEI管理化
- Database readをすべてSemantic Capabilityへ統一すること
- Target UI向けData API
- 未観測Oracle物理情報の推測

## 関連

- `ISSUE-000006` / #69
- `ISSUE-000017` / #178
- `ISSUE-000019` / #185
- `ISSUE-000020` / #222
- `ISSUE-000021` / #224
- `ISSUE-000022` / #214
- `ISSUE-000029` / #273
- `internal/realization/contract_query_sql.go`
- `internal/driver/queryruntime/sql/`
- `internal/driver/queryruntime/oracle/`
