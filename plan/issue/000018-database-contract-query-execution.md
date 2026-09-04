# ISSUE-000018: Database Contract Queryのinput・result・実行境界を明示する

- 状態: In Progress
- GitHub Issue: #179
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / GitHub Issue #178
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから`master_data`のCustomerを取得する経路を実際に通した結果、Semantic Relationのkey対応とは別に、Database Contract Query自身の契約、Application側の呼び出し、物理realization、Environment固有の実行Bindingを分けて扱う必要があることが明確になりました。

第一Referenceでは、意味の対応を次の三層に分離します。

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

物理SQLやDatabase APIは、この三層とはさらに分離したrealizationとして扱います。これにより、RelationへSQLを持ち込まず、Contract Queryへ`Order.customer`という業務Relationの意味を重複させず、Driverにも名前・型・列順による推測を持たせません。

## 現在の実行経路

現在のEvidence用convergenceでは、第一Referenceの論理的な入力からprovider adapter直前まで、次の経路を明示identityだけで接続できます。

```text
Mapped browse row
  sales.order.customer_id = 42
        ↓ Semantic Relation key
customer.customer.customer_id
        ↓ Reference invocation binding
Contract Query input id = 42
        ↓ Database Contract Query signature
master_data.customer / find_by_id
        ↓ ContractQuerySQLRealization
physical parameter + native SQL + raw column mapping
        ↓ Environment composition
master_data.dev / database.query.oracle
        ↓ instance-aware runtime Registry
Oracle Contract Query Executor
        ↓ database/sql Runner
raw result
        ↓ explicit result mapping
Semantic ContractQueryResult
```

この経路のどこでも、ファイル名、provider名、物理column名、SELECT順序、Semantic型の一致だけを根拠に不足情報を補完しません。

## Query signature

`database.QueryContract`には、次の最小契約があります。

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

## Reference invocationとruntime parameter

Applicationのreference Patternは、利用するContract QueryとRelation target keyからQuery inputへの対応を明示します。

```tei
contract-query master_data.customer find_by_id {
    input customer.customer.customer_id -> id
}
```

Driverは、Semantic型が一致するだけのinputや`find_by_id`という名前から対応を推測しません。

browse結果からruntime parameterを作るときも、次の明示経路だけを使います。

```text
MappedResult
  sales.order.customer_id = 42
        ↓ Relation key
customer.customer.customer_id
        ↓ invocation binding
Query input id
        ↓
Parameters["id"] = 42
```

値はこの境界で型変換せず、そのまま渡します。runtime parameter値はCanonical Worldや通常Evidenceへ保存しません。

## 三層をまたぐValidation

Semantic Relation、Reference invocation、Database Contract Query signatureは、それぞれ独立したmodelとしてValidationします。その上で、三層が単独では妥当な場合にのみ、横断的な対応関係を検査します。

主な検査内容は次です。

- Relation keyのsource / target attributeが正しいSemantic Dataに属すること
- Relation key両端が同じSemantic type identityを持つこと
- invocationがreference Patternにだけ付与されること
- Contract QueryがApplication datasourceの要求Contractに含まれること
- invocationのtarget attributeがRelation target keyであること
- Query inputがsignatureに存在すること
- Relation target keyの型とQuery input型が一致すること
- Query result DataとRelation target Dataが一致すること
- result cardinalityがreference traversalと矛盾しないこと

これらは通常の`validation.Validate`経路へ接続済みです。Workspaceから`tei validate`を実行した場合にも、個別Validationを呼び忘れる構造にはしません。

## Query realization

Database-ownedなQueryを既存の`Application + Capability` identityへ変換せず、次の三つ組を独立したtarget identityとします。

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

このidentityにはApplication ID、Capability ID、provider、Environment、file pathを含めません。実装方式や配置が変わっても、「何を実現しているか」というtarget identityを維持します。

SQL realizationはこのtargetに対して、次を明示します。

```text
ContractQuerySQLRealization
  ├─ target: Database / Contract / Query
  ├─ provider
  ├─ Contract input -> physical parameter
  ├─ native SQL source
  └─ raw result column -> Semantic attribute
```

Contract input名とphysical parameter名が同じ場合でも、対応を省略しません。raw列も、列名の類似やSELECT順序からSemantic attributeへ結び付けません。

SQL realization authoringは通常Workspace読込みへ接続済みで、source identityとSource Indexも持ちます。ただし`external-observed` Databaseについて、authoring sourceが存在しないのに物理SQLを生成することはしません。

## physical resultからSemantic resultへの変換

provider adapterが返したraw resultは、SQL realizationの`result-column` mappingだけを使ってSemantic rowへ変換します。

この境界では次を検査します。

- raw resultに必要な列が存在すること
- 同じraw列を複数Semantic attributeへ曖昧に割り当てないこと
- mapping先attributeがContract Query result Dataに属すること
- result cardinalityに反しないこと

したがって、physical result → Semantic rowの仕組み自体は実装済みです。第一Referenceの実列名が未確定であることと、このmapping機構が未実装であることは区別します。

## Executor CapabilityとEnvironment Binding

既存`database-provider` CapabilityをContract Query実行能力として暗黙に扱わず、Plugin Capability kindとして`contract-query-executor`を独立させています。

第一ReferenceのOracle pluginは、Database providerとQuery executorを別々のCapabilityとして宣言します。

```tei
provides database-provider database.oracle {
    provider oracle
}

provides contract-query-executor database.query.oracle {
    provider oracle
}
```

Environment側では、Database instanceとContract Query targetに対するexecutorを明示Bindingします。

```text
Environment
  development
        +
Database instance
  master_data.dev
        +
ContractQueryTarget
  master_data/master_data.customer/find_by_id
        ↓ explicit binding
Executor Capability
  database.query.oracle
```

第一Referenceのauthoring sourceは次です。

```text
environments/development/contract-query-executors/customer-find-by-id.tei
```

本文にEnvironment、instance、Database、Contract、Query、executor Capabilityを明示します。ファイル名や親ディレクトリ名はCanonical identityを補完しません。

## instance-aware runtime Registry

当初は、Environment側でDatabase instanceまで一意に解決した後、runtime Registryだけがexecutor Capability IDへ縮退していました。この形では、同じCapabilityを複数instanceが共有すると別接続を取り違え得ます。

`ISSUE-000020` / GitHub Issue #222でこの境界を修正し、runtime Registryは次の三要素を保持するようにしました。

```text
Environment + Database instance + executor Capability
```

同じOracle provider、同じ`database.query.oracle` Capabilityを利用する`master_data.dev`と`master_data.shadow`が存在しても、別instance向けExecutorへfallbackしないConformanceを持ちます。

provider一致やDatabase definition一致は、runtime接続先を選ぶ代替規則にはしません。

## Oracle `database/sql` Runner

`ISSUE-000021` / GitHub Issue #224では、Oracle向けの最小`database/sql` Runner境界を追加しました。

Runnerの責務は次だけです。

- 注入済み`*sql.DB`を使う
- 上位から渡されたnative SQLを変更しない
- physical parameterをnamed argumentとして渡す
- Databaseが観測した列名とrow値をraw resultとして返す
- Evidenceにはparameter名だけを残し、runtime値を保存しない

Oracleのtable / view / column、SQL本文、DSN、credential、接続先instanceはRunnerが決めません。

特定Oracle driver packageもCoreへ固定していません。実driverとのnamed parameter互換性は、第一Referenceの物理Evidenceが得られた後にintegrationで確認します。

## 0件とExecution Error

第一Referenceの`zero-or-one`では、次を区別します。

- 0件: 正常な「見つからない」
- 1件: 正常な取得結果
- 2件以上: Contract違反
- 接続失敗・SQL/API失敗: Execution Error
- row取得途中の失敗: Execution Error

0件をDatabase障害のように扱わず、逆にDatabase障害を正常な0件として隠しません。

## Authoring sourceとDiagnosticの追跡

現在は、少なくとも次のsourceへCanonical identityから戻れます。

- Semantic Relation
- Application reference Pattern
- Database Contract Query
- Contract Query executor Binding
- Contract Query SQL realization

Database Contract Query signatureのDiagnosticは、`database-contract-query:<contract>/<query>`というsubject identityを持ち、Contractファイル内の`query <name>`識別子までSource Indexで投影します。

このため、Validation結果を単なるファイル先頭へ戻すのではなく、「どのContract Queryの定義を直すべきか」まで追跡できます。

## 現在露出しているGap

仕組みとして未実装だった主要境界は、Evidence用convergence上ではほぼ埋まりました。現在残っている本質的なGapは、第一Referenceの**観測されていない物理事実を確定すること**です。

### 1. `master_data.customer/find_by_id`の物理Evidence

Repositoryには現在、実際のtable / view / column、native SQL、physical parameter名、raw result列、provenanceがありません。

これらは`ISSUE-000022` / GitHub Issue #214で扱います。synthetic Conformanceや命名慣習から補完しません。

### 2. read-onlyの実効的な保証

Contract Queryという名称やSQL文字列の見た目だけを根拠にread-onlyとは判断しません。

実際のcredential、session、transaction、Database権限等のうち、どの境界が書込みを禁止しているかをEvidenceとして確認します。SQL文字列解析を権限境界の代わりにしません。

### 3. 実Oracle driverとのintegration

`database/sql` Runner自体は実装済みですが、特定Oracle driverはまだ選定・固定していません。

実driverについては、少なくとも次を実接続Evidenceで確認します。

- named parameterの扱い
- 対象native SQLの実行
- 型変換
- NULL
- raw column名
- Error伝播
- read-only強制との整合

### 4. 第一Referenceの最終E2E

物理Evidenceが得られた後、通常のWorkspace読込みから次を一続きに通します。

```text
Workspace authoring
  → validation
  → browse mapped row
  → Relation traversal
  → Contract Query input
  → Environment / instance / executor resolution
  → native Oracle read
  → raw result
  → Semantic result
```

このE2EがGate Bの主要Evidenceになります。

### 5. CI

2026年9月3日現在、`.github/workflows/ci.yml`はRepositoryから明示的に削除され、`work/reference-data-access-convergence`にもその停止が取り込まれています。そのため、現在のPRでworkflow runが作成されないことは、Issue #192で記録した`runner_id=0 / steps=[]`事象の継続を直接意味しません。

CI未実行の成果をgreenとは扱いません。一方、このContract Query作業線から独断でActionsを再有効化することもしません。CIが別の明示的な判断で再開された後、これまでのConformanceを通常runnerで一括実行し、Issue #192のrunner割当事象が再現するかも改めて切り分けます。

## 次の実証順

1. `ISSUE-000022`で第一Referenceの物理object、SQL、parameter、raw列、provenanceを確定する
2. read-onlyを実効的に強制する境界をEvidenceで確認する
3. 採用するOracle driverをintegration境界で接続し、named parameter等の互換性を確認する
4. 第一Referenceの実SQL realization authoringを追加する
5. instance-aware Registryへ`development / master_data.dev / database.query.oracle`の実Executorを登録する
6. Workspace読込みからSemantic resultまでの最終E2Eを通す
7. CIが明示的に再有効化された後、通常runnerで全Conformanceを再実行する

物理Evidenceが揃う前にOracle SQLを仮定して手順4以降を先取りしません。

## 完了条件

- [x] `find_by_id`のinputを名前から推測しない
- [x] inputがどのSemantic typeを表すか追跡できる
- [x] resultが`customer.customer`であることを契約から追跡できる
- [x] result cardinalityを`zero-or-one`として明示できる
- [x] Relation keyからQuery inputへの対応を明示できる
- [x] mapped rowからruntime parameterを暗黙変換なしで作れる
- [x] Database Contract Queryとrealization target identityを分離できる
- [x] Contract inputからphysical parameterへのmappingを明示できる
- [x] physical result列からSemantic attributeへのmappingを明示できる
- [x] physical resultを明示mappingからSemantic rowへ変換できる
- [x] Database固有SQL / API / stored procedureをDriver Coreへ持ち込まない
- [x] 0件とExecution Errorをruntimeで区別する
- [x] 1件とcardinality違反をruntimeで区別する
- [x] Database Contract Query executorを`database-provider`とは別Capabilityとして表現できる
- [x] provider一致だけではexecutorを自動選択しない
- [x] Environment + instance + target + executorの明示Bindingを表現し、一意性とprovider整合を検査できる
- [x] Application datasourceからEnvironment上のDatabase instanceを推測なしで解決できる
- [x] runtime RegistryがEnvironment + Database instance + Capability identityを保持する
- [x] 同一Capabilityを共有する別Database instanceへfallbackしない
- [x] Bindingをauthoring sourceからWorldへ読み込める
- [x] Binding authoring sourceの来歴をSource Indexから追跡できる
- [x] Query realization sourceをWorkspaceから読み込める
- [x] Query realization sourceの来歴を追跡できる
- [x] Database Contract Query DiagnosticをQuery authoring位置へ投影できる
- [x] provider非依存SQL Executorからraw result → Semantic resultを実行できる
- [x] Oracle向け`database/sql` Runner境界を持つ
- [ ] 第一Referenceの実物理object / SQL / parameter / raw列のEvidenceがある
- [ ] `master_data`をread-onlyのまま実際のread実行へ接続できる
- [ ] 採用Oracle driverとのnamed parameter等の互換性を実接続で確認できる
- [ ] 第一ReferenceをWorkspace読込みから実Oracle read、Semantic resultまで一続きに通せる
- [ ] CIが明示的に再有効化された後、通常runnerで関連Conformanceが成功する

## 非目標

- 汎用ORM
- 全Database Contract Queryを覆う完全な型体系
- production Connection Pool / HA
- external Database schemaのTEI管理化
- Database queryをすべてSemantic Capabilityへ統一すること
- Application Queryとの自動統合
- Target UI向けデータ取得API
- 物理Evidenceがない状態でOracle SQLを生成すること

## 関連

- GitHub Issue #69 / #178 / #179 / #185 / #192 / #214 / #222 / #224
- PR #193 / #195 / #197 / #201 / #205 / #206 / #207 / #208 / #216 / #217 / #218 / #219 / #220 / #221 / #223 / #225
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000021-oracle-contract-query-sql-runner.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
- `reference/workspace/plugins/tei-database-oracle/1.0.0/plugin.tei`
- `internal/driver/contract_query_execution.go`
- `internal/driver/reference_contract_query.go`
- `internal/driver/reference_contract_query_parameters.go`
- `internal/driver/reference_contract_query_environment.go`
- `internal/driver/queryruntime/sql/executor.go`
- `internal/driver/queryruntime/oracle/runner.go`
- `internal/realization/contract_query.go`
- `internal/realization/contract_query_binding.go`
- `internal/realization/contract_query_sql.go`
