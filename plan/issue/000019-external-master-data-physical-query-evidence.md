# ISSUE-000019: external-observed `master_data`の物理Query Evidenceを確定する

- 状態: Ready
- GitHub Issue: #214
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

第一Referenceの`master_data`は、TEIが所有・生成するDatabaseではなく、既存資産を観測して利用する`external-observed`なDatabaseとして定義しています。Environment上ではOracle providerを指定しています。

一方、現在のRepositoryにある`reference/workspace/databases/master-data`を確認すると、存在するのは`database.tei`とDatabase Contractだけです。Persistence Mapping、物理Schema、table / view名、column名、既存SQLなど、実際のread queryを構成するための物理Evidenceはまだありません。

この状態で`CUSTOMER`、`CUSTOMER_ID`のような名前を仮定してSQLを作ると、TEIが観測されていない外部資産を名前や慣習から補完することになります。これは、Driverやrealization層が不足した意味・物理事実を推測しないというTEIの原則に反します。

したがって、SQL realizationの仕組み自体は先に整備しても、第一Referenceの実SQLは物理Evidenceが得られるまで確定しません。

## 既に準備できる境界

物理Schemaの具体名が未確定でも、意味と物理実装を分離する境界は先に固定できます。

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
provider adapter / raw execution
        ↓ explicit result mapping
ContractQueryResult
  └─ Semantic rows
```

この境界では、Contract input名とSQL parameter名が同じ場合でも対応を省略しません。raw列についても、列名の類似やSELECT順序からSemantic attributeを決めません。

## 必要な物理Evidence

第一Referenceの`master_data.customer / find_by_id`を実際のread executionへ接続する前に、少なくとも次を確定します。

- 実際に参照する物理objectが何か（table / view / API等）
- Contract input `id`をどの物理parameterへ渡すか
- 実行結果として返るraw列は何か
- 各raw列を`customer.customer`のどのSemantic attributeへ対応付けるか
- read-onlyであることを確認できる実行方法
- Oracle SQLを利用する場合、そのnative SQL source
- これらの情報を確認したsource / provenance

## 物理Evidenceを得る方法

Referenceは「既存システムを受け入れる」性質を実証するためのものなので、物理情報をTEI側で都合よく生成するのではなく、次のいずれかの形で外部資産として明示します。

1. 既存Databaseを模した観測済みfixtureをReference assetとして用意する
2. 実際の外部Schema / SQLの匿名化・縮約fixtureを用意する
3. Oracle依存がCIで扱えない場合は、同じphysical contractを返すReference adapterを用意し、SQL固有部分だけをprovider boundaryの外側に残す

どの方法を採る場合も、fixtureやadapterがCanonical business meaningの正本にならないようにします。

## 完了条件

- [ ] 第一Referenceで参照する物理objectがEvidenceとして明示されている
- [ ] Contract input `id`から物理parameterへの対応が明示されている
- [ ] raw result列が明示されている
- [ ] raw result列から`customer.customer.customer_id` / `customer.customer.name`等への対応が明示されている
- [ ] native SQL / API / adapter等のrealization sourceを追跡できる
- [ ] `master_data`をread-onlyのまま実行できる
- [ ] 0件を正常なnot-foundとして保持できる
- [ ] Execution Errorと0件を混同しない
- [ ] physical fixtureやadapterがSemantic meaningの正本として扱われていない

## 非目標

- 外部Database全体をTEI管理下へ移すこと
- 不明な物理SchemaをTEIが生成・推測すること
- 汎用ORMの導入
- Oracle固有の高度なHA / Connection Pool機能をこのIssueだけで完成させること

## 関連

- GitHub Issue #179 / #185 / #214
- `plan/issue/000018-database-contract-query-execution.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `internal/realization/contract_query_sql.go`
- `internal/driver/contract_query_sql_result.go`
