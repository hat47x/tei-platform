# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Query実行: `ISSUE-000016` / GitHub Issue #167
- Relation traversal key: `ISSUE-000017` / GitHub Issue #178
- Database Contract Query: `ISSUE-000018` / GitHub Issue #179
- Reference invocation: `ISSUE-000019` / GitHub Issue #185
- Contract Query runtime instance identity: `ISSUE-000020` / GitHub Issue #222
- Oracle Contract Query Runner: `ISSUE-000021` / GitHub Issue #224
- external `master_data`物理Evidence: `ISSUE-000022` / GitHub Issue #214
- header-detail traversal境界: `ISSUE-000023` / GitHub Issue #233（Done / ADR-0021）
- keyless Relationの物理保存境界: `ISSUE-000024` / GitHub Issue #244

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
- `header-detail`: `many` Relation / source・target Data / traversal realization
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

`reference`を実値まで進める過程で、Relationの概念的なsource / targetだけではinstance間を辿れないケースがあることが分かりました。

第一Referenceの`Order -> Customer`では、次を明示しています。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

Relation keyはSemantic Relation自身が所有します。物理Foreign Key、Database Query、Application Datasource、Query input名は持ち込みません。詳細は`ISSUE-000017`を正とします。

一方、RelationKeyをすべてのRelationへ一般化しません。target側に業務上意味のある対応attributeがないkeylessなRelationは、別の実現境界から実行できます。`Order -> OrderLine[]`でそのことを実証しています。

### Database Contract Query

`master_data.customer / find_by_id`にはtyped signatureを持たせています。

```tei
query find_by_id {
    input id type customer.customer_id
    result customer.customer zero-or-one
}
```

Database Contract Queryはinput / result / cardinalityを所有し、Database固有の実現方法とは分離します。

Queryを実現する対象identityには、ApplicationやCapabilityを流用せず、次の三つ組を使います。

```text
Database / Contract / Query
master_data / master_data.customer / find_by_id
```

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

runtimeでは、`browse`でmapping済みのsource rowからRelation keyと明示bindingだけを使ってContract Query parameterを構築できます。詳細は`ISSUE-000019`を正とします。

### Contract Query SQL realizationとSemantic result mapping

Reference invocationの次に必要だった、Database Contract Queryと物理Queryの間の境界も実装しています。

```text
ContractQueryTarget
  Database / Contract / Query
        ↓
ContractQuerySQLRealization
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

Contract input名とphysical parameter名が同じ場合でも、対応を省略しません。raw列についても、列名の類似やSELECT順からSemantic attributeを推測しません。

SQL realizationは通常Workspaceから読み込め、authoring sourceをSource Indexで追跡できます。Database Contract Query DiagnosticもContract内のQuery識別子へ投影できます。

したがって、「Query realization sourceの仕組み」や「physical resultからSemantic rowへのmapping機構」は、現在の主要Gapではありません。第一Referenceの**実際の物理構造やSQLが未確定であること**とは区別します。

### Environment runtime compositionとinstance identity

第一Referenceはlogical DatasourceからEnvironment上のDatabase instanceを解決し、明示Bindingを通してexecutor Capabilityとruntime実装へ接続できます。

`Order -> Customer`では次です。

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

Environment、Database instance、executor、runtime implementationをprovider名や配置から推測しません。Binding authoring sourceはWorkspace Loaderから読み込まれ、Source Indexへ来歴を残します。

当初runtime RegistryはCapability IDだけへ縮退していましたが、`ISSUE-000020`でEnvironment + Database instance + executor Capabilityをkeyとして保持する形へ変更しました。同じproviderと同じexecutor Capabilityを使う別Database instanceへfallbackしません。

### Oracle `database/sql` Runner

`ISSUE-000021`では、上位のSQL realizationが確定したnative SQLとphysical parameterを、注入済み`*sql.DB`へ渡すOracle向け最小Runnerを追加しています。

Runner自身は、次を決めません。

- table / view / column
- Contract inputとphysical parameterの対応
- raw columnとSemantic attributeの対応
- Environment / Database instance
- DSN / credential
- Oracle driver library

0 rowは正常なraw resultとして保持し、Query実行失敗やrow取得失敗はExecution Errorとして返します。runtime parameter値はEvidenceへ保存しません。

### keylessなheader-detail traversal

`ISSUE-000023` / ADR-0021で、`sales.order.lines`をkeylessな`many` Relationのままruntime traversalへ接続しました。

Application PatternはRelation-onlyのまま維持します。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

取得方法は、Pattern外のApplication固有realizationへ置きます。

```tei
header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}
```

Database Contract Queryは次のtyped contractです。

```tei
query find_by_order {
    input order_id type sales.order_id
    result sales.order_line many
}
```

この経路ではRelationKeyを要求せず、`sales.order_line.order_id`も追加していません。mapped Order rowの`sales.order.order_id`を明示Bindingから`order_id`へ渡します。列順、raw列名、Persistence Mapping、physical Foreign Keyから値を推測しません。

既存`ContractQueryExecutor`を再利用し、次を区別できることを実Referenceで確認しています。

```text
0 row          -> 正常な空collection
複数row        -> 正常なsales.order_line[]
Executor error -> Execution Error
```

### header-detailのPostgreSQL Environment composition

PostgreSQL側にも共通`sqlquery.Executor`から利用できる最小SQL Runnerを追加し、既存の`:name -> $1, $2, ...` binderを再利用しています。

Referenceの`development`では、次を明示compositionします。

```text
order_entry.sales
  -> sales_core.dev
  -> database.query.postgres
```

providerがPostgreSQLであることだけを根拠にCapabilityを選びません。Database instance Binding、Contract Query executor Binding、Plugin Capability、instance-aware Registryを順に解決します。

さらに、ここまでのcompositionが成立していても、`sales_core/sales_core.order_line/find_by_order`のPostgreSQL SQL realizationが存在しなければ、physical Runnerを呼ぶ前に明示的に停止します。

つまり、「runtime接続先を解決できた」ことと「物理SQLが確定した」ことを別の状態として保持します。

### Driver入口の一本化

旧`internal/applicationdriver`は、現在の`internal/driver`とPattern解決・Scenario組み立ての責務が重複していました。

現行Driverへ必要な機能が集約されたため、互換wrapperを追加せず旧packageを削除し、Driver入口を`internal/driver`へ一本化しています。今後、新しいScenario実装を旧入口へ分岐させません。

## 現在の主要Gap

### 1. external `master_data`の物理Evidence

第一Referenceの`master_data`は`external-observed`です。Repositoryには現在、`master_data.customer / find_by_id`について実際のtable / view / column、native SQL、physical parameter名、raw result列、provenanceがありません。

そのため、`CUSTOMER`や`CUSTOMER_ID`のような名前を仮定してReference実SQLを作りません。

必要な物理Evidenceは`ISSUE-000022` / GitHub Issue #214で扱います。

### 2. read-onlyの実効的な保証と実Oracle integration

`find_by_id`という名称や、SQLが`SELECT`に見えることだけではread-onlyの保証としません。

実際のcredential、session、transaction、Database権限等のどの境界が書込みを防ぐかをEvidenceで確認します。また、採用するOracle driverについてnamed parameter、型変換、NULL、raw column名、Error伝播を実接続で確認します。

これらも`ISSUE-000022`の物理Evidenceと一緒に進めます。

### 3. `Order -> Customer` Referenceのend-to-end実証

Workspace読込み、Relation key、Reference invocation、parameter準備、Environment composition、instance-aware Registry、Oracle Runner、raw result mappingを、観測済み物理Evidenceを使った実readまで一続きに通します。

0件は正常な「見つからない」、接続・Query実行失敗はExecution Errorとして区別します。`master_data`のread-only境界も維持します。

このE2Eは新しい推測規則を追加する作業ではなく、既に確定した境界がexternal-observed資産でも成立することを確認するEvidenceです。

### 4. keyless header-detail Relationの物理保存

`Order -> OrderLine[]`の論理traversalとruntime compositionは成立しました。残っているのは、managed-hereな`sales_core`で`sales.order.lines`をどのphysical storageへ写すかです。

現行Persistence MappingはSemantic attributeとphysical columnの対応を基本としているため、target Dataに親Order attributeを追加しないkeyless Relationには、Database-ownedなrelationship storageを表す境界が不足しています。

この課題は`ISSUE-000024` / GitHub Issue #244で扱います。

ここが確定する前に、`order_lines.order_id`、physical Foreign Key、`find_by_order` native SQLをDriver側から発明しません。

### 5. 最終Scenario統合

Reference data accessが成立した後、Authentication、Canonical Validation、Authorization、Command、Observationまでを一つのUI-less Scenarioへ接続します。

個別機能が存在するだけではGate B Readyとはせず、同じApplication定義から一続きに再現できることを確認します。

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

1. `ISSUE-000022`で`master_data.customer / find_by_id`の物理object、SQL、parameter、raw列、provenanceを確定する
2. read-onlyを実効的に強制する境界をEvidenceで確認する
3. 採用Oracle driverでnamed parameter等のintegrationを確認する
4. 観測済みEvidenceから第一Referenceの実SQL realization authoringを追加する
5. `Order -> Customer` ReferenceをWorkspace読込みから実read、Semantic resultまで一続きに実証する
6. `ISSUE-000024`でkeylessな`sales.order.lines`のDatabase-owned physical storage境界を決める
7. 確定したstorage authoringからOrderLine persistence / migration / `find_by_order` SQL realizationを追加する
8. 実PostgreSQLで`Order -> OrderLine[]`の0件・複数件・Execution Errorを確認する
9. Canonical Validation / Authorization / Commandを一つのScenarioへ接続する
10. Scenario EvidenceをRepositoryだけから再現できる形へ整える
11. CIが明示的に再有効化された後、関連Conformanceを通常runnerで再検証する
12. Gate Bを監査する

`master_data`の物理Evidenceが揃う前に実Oracle SQLを仮定しません。また、`ISSUE-000024`が決まる前にOrderLineのphysical support columnやForeign Keyを仮定しません。

## Gate Bへ進む条件

Driverのコードが存在するだけではGate B Readyとはみなしません。少なくとも次をReference Scenarioから確認します。

- 同じApplication定義からScenarioを再現できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingの解決を追跡できる
- native QueryやContract Queryの物理実現をDriver Coreへ複製しない
- raw columnとSemantic attributeの対応を明示mappingから追跡できる
- Relation traversal keyを名前・型・physical Foreign Keyから推測しない
- keylessな`many` Relationにtarget-side RelationKeyを強制しない
- header-detailのsource attribute → Query input bindingを明示できる
- Reference Contract QueryをQuery名・型・列挙順から推測しない
- Query 0件とQuery Execution Errorを混同しない
- `many`の0件を正常な空collectionとして扱える
- logical Datasource aliasとEnvironment上のinstance境界を保持する
- runtime Executorが別Database instanceへfallbackしない
- SQL realization欠落時にproviderやschemaから物理SQLを補完しない
- `master_data`へwriteしないことを実効的な境界で保証する
- Validation不成立時にcommandへ進まない
- Authorization `denied`時にcommandへ進まない
- expectedなNamed OutcomeをDriver障害として扱わない
- Execution Errorを業務Outcomeへ押し込まない
- command acknowledgementとobserved resultを混同しない
- secretやruntime parameter値を通常Evidenceへ混入させない
- Driver固有状態をCanonical Worldへ逆流させない
- 同じApplication上の意味を第一Web Targetへ渡せる
- Reference ScenarioをRepositoryだけから再現できる

## CIの扱い

2026年9月3日現在、`.github/workflows/ci.yml`はRepositoryから明示的に削除され、Reference data access収束線にもその停止が取り込まれています。

現在workflow runが作成されないことを、Issue #192で記録した過去の`runner_id=0 / steps=[]`事象の継続と同一視しません。一方、CI未実行の成果をgreenとも扱いません。

このApplication Driver作業線から独断でActionsを再有効化せず、別の明示的な判断でCIが再開された後、Gate B監査前に関連Conformanceを通常runnerで再実行します。

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
- 観測されていないexternal Databaseの物理情報をDriverが補完すること
- keyless Relationのphysical storageをDriver Coreが決めること

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000016-native-query-runtime-boundary.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000021-oracle-contract-query-sql-runner.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- GitHub Issue #69 / #77 / #124 / #167 / #178 / #179 / #185 / #192 / #214 / #222 / #224 / #233 / #244
- ADR-0019 / ADR-0020 / ADR-0021
- RQ-005 / RQ-008 / RQ-009 / RQ-011 / RQ-012 / RQ-014 / RQ-017 / RQ-018 / RQ-021
