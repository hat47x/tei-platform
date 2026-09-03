# ISSUE-000019: external-observed `master_data`の物理Query Evidenceを確定する

- 状態: Ready
- GitHub Issue: #214
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連Issue: `ISSUE-000020` / `ISSUE-000021`
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

第一Referenceの`master_data`は、TEIが所有・生成するDatabaseではなく、既存資産を観測して利用する`external-observed`なDatabaseとして定義しています。Environment上では`master_data.dev`としてBindingされ、providerはOracleです。

一方、現在のRepositoryにある`reference/workspace/databases/master-data`には、Database定義とDatabase Contractはありますが、実際のread queryを確定できる物理Evidenceがありません。

未確定なのは、たとえば次の情報です。

- 実際に読むtable / view / API等
- 実際のcolumn名
- `find_by_id`へ渡すphysical parameter名
- 実際に返るraw result列
- native Oracle SQL
- これらを確認したsource / provenance

この状態で`CUSTOMER`、`CUSTOMER_ID`のような名前を仮定してSQLを作ると、TEIが観測されていない外部資産を命名慣習から補完することになります。これは、Driverやrealization層が不足した意味・物理事実を推測しないというTEIの原則に反します。

したがって、実行機構が整ったことと、第一Referenceの物理事実が確定したことを明確に分けます。

## 既に実装済みの境界

物理Schemaの具体名を知らなくても、意味と物理実装を混ぜないための仕組みは先に整備しています。

```text
Database Contract Query
  master_data.customer / find_by_id
        ↓ explicit realization
ContractQuerySQLRealization
  ├─ target: Database / Contract / Query
  ├─ provider
  ├─ Contract input -> physical parameter
  ├─ native SQL source
  └─ raw result column -> Semantic attribute
        ↓
Environment / Database instance / executor Binding
        ↓
instance-aware runtime Registry
        ↓
provider adapter / raw execution
        ↓ explicit result mapping
ContractQueryResult
  └─ Semantic rows
```

現在までに、次の仕組みはEvidence用convergenceへ実装済みです。

- Relation keyからContract Query inputを準備する
- Contract Query signatureを型付きで表現・Validationする
- Reference invocationを明示する
- Relation / invocation / Query signatureを横断Validationする
- Database / Contract / Queryをrealization target identityとする
- Contract input → physical parameterを明示mappingする
- raw result column → Semantic attributeを明示mappingする
- SQL realizationを通常Workspaceから読み込む
- SQL realizationのauthoring sourceをSource Indexで追跡する
- Database Contract Query DiagnosticをContract内のQuery識別子へ投影する
- Environment / Database instance / executor Capabilityを明示Bindingする
- runtime RegistryでDatabase instance identityを保持する
- 同一Capabilityを共有する別Database instanceへfallbackしない
- Oracle向けの最小`database/sql` Runnerで、明示SQLとnamed parameterをraw executionへ渡す
- 0 rowとExecution Errorを区別する

これらは「物理情報が不明でも何を推測してはいけないか」を固定するための境界です。実際のtable名やSQLの代用品ではありません。

## 必要な物理Evidence

第一Referenceの`master_data.customer / find_by_id`を実際のread executionへ接続する前に、少なくとも次を確定します。

### 物理object

- 実際に参照するのはtable、view、API、stored procedure等のどれか
- そのobjectのidentity
- Referenceで利用してよい範囲

### input

- Contract input `id`をどのphysical parameterへ渡すか
- parameterの実際の型やdriver上の扱い
- NULLや型変換が必要な場合、その扱い

### result

- 実行結果として返るraw列
- 各raw列の型・NULLの扱い
- raw列から`customer.customer`のSemantic attributeへの明示mapping

少なくともCustomer識別子と、第一Referenceで本当に必要な属性だけを対象にします。外部Schema全体を取り込むことは目的ではありません。

### realization source

Oracle SQLを利用する場合は、実際に利用するnative SQL sourceをEvidenceとして用意します。

SQL本文はContractやSemantic Relationから生成しません。既存SQLを利用する場合はその由来を、Reference用に匿名化・縮約する場合は元Evidenceとの対応を記録します。

### provenance

物理情報について、少なくとも次を追跡できるようにします。

- 何を観測したか
- どのsourceから確認したか
- そのまま利用したのか、匿名化・縮約したのか
- どこまでが観測事実で、どこからがReference用の加工か

## read-onlyの実効的な保証

`find_by_id`がQueryであることや、SQLが`SELECT`に見えることだけではread-onlyの保証としません。

第一Referenceでは、実際に書込みを防ぐ境界をEvidenceとして確認します。候補には次があります。

- read-only権限だけを持つDatabase credential
- read-only session / transaction
- Database側の権限制御
- 書込みAPIを公開しないReference adapter

どれを採用するかは実環境に合わせますが、「TEIがSQL文字列を見て安全だと推測した」だけの状態は完了とみなしません。

また、Oracle Runnerが`*sql.DB`を注入される構造になったことを、read-only保証そのものとは扱いません。接続先を明示できることと、その接続が書込み不能であることは別の事実です。

## 実Oracle driverとのintegration

`ISSUE-000021`では、特定Oracle driverに依存しない`database/sql` Runnerまで実装しています。

そのため、このIssueで物理Evidenceが揃った後は、採用するOracle driverについて少なくとも次を確認します。

- named parameterが期待どおりに渡ること
- 対象native SQLが実行できること
- input / resultの型変換
- NULLの扱い
- raw column名
- Query Error / row取得Errorの伝播
- read-only強制との整合

特定driverを先に選んで、そのdriverが扱いやすい形へCanonical contractを変えることはしません。

## 物理Evidenceを得る方法

Referenceは「既存システムを受け入れる」性質を実証するためのものなので、物理情報をTEI側で都合よく生成するのではなく、次のいずれかの形で外部資産として明示します。

1. 既存Databaseを模した、観測根拠のあるfixtureをReference assetとして用意する
2. 実際の外部Schema / SQLを匿名化・縮約したfixtureを用意する
3. Oracle依存がCIで扱えない場合は、同じphysical contractを返すReference adapterを用意し、実Oracle固有部分はintegration Evidenceとして別に残す

どの方法でも、fixtureやadapterをCanonical business meaningの正本にしません。

## Evidence取得後の作業順

1. 観測した物理object / parameter / raw列 / provenanceを記録する
2. 第一Referenceの`ContractQuerySQLRealization` authoringを追加する
3. native SQL sourceを追跡可能な形で配置する
4. read-onlyを強制する接続境界を用意する
5. 採用Oracle driverでnamed parameterとraw resultを確認する
6. `development / master_data.dev / database.query.oracle`向けruntime Executorをinstance-aware Registryへ登録する
7. Workspace読込みからSemantic resultまでのE2Eを実行する
8. 通常CI runnerでも再現可能なEvidenceを残す

## 完了条件

- [ ] 第一Referenceで参照する物理objectがEvidenceとして明示されている
- [ ] Contract input `id`からphysical parameterへの対応が、実物理名で明示されている
- [ ] 実際のraw result列が明示されている
- [ ] raw result列から`customer.customer`の必要なSemantic attributeへの対応が明示されている
- [ ] native SQL / API / adapter等の実realization sourceを追跡できる
- [ ] 物理情報のsource / provenanceを追跡できる
- [ ] `master_data`を実効的にread-onlyのまま実行できる
- [x] 0件を正常なnot-foundとして扱える実行境界がある
- [x] Execution Errorと0件を区別できる実行境界がある
- [x] Contract input → physical parameterの明示mapping機構がある
- [x] raw result → Semantic attributeの明示mapping機構がある
- [x] SQL realization sourceのSource Index / provenance機構がある
- [x] runtime RegistryがDatabase instance identityを保持する
- [x] Oracle向け`database/sql` Runner境界がある
- [ ] 採用Oracle driverとのintegration Evidenceがある
- [ ] 第一ReferenceのWorkspace → 実read → Semantic resultを一続きに確認できる
- [ ] physical fixtureやadapterがSemantic meaningの正本として扱われていないことを確認できる

## 非目標

- 外部Database全体をTEI管理下へ移すこと
- 不明な物理SchemaをTEIが生成・推測すること
- 汎用ORMの導入
- Oracle固有の高度なHA / Connection Pool機能をこのIssueだけで完成させること
- synthetic ConformanceのSQLを第一Referenceの実SQLとして採用すること

## 関連

- GitHub Issue #179 / #185 / #214 / #222 / #224
- PR #223 / #225
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000021-oracle-contract-query-sql-runner.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/environments/development/environment.tei`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
- `internal/realization/contract_query_sql.go`
- `internal/driver/contract_query_sql_result.go`
- `internal/driver/reference_contract_query_environment.go`
- `internal/driver/queryruntime/oracle/runner.go`
