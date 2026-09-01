# 現在の実装状況

TEIは、完成したローコード製品ではなく、設計上の境界を実行可能なコードとReferenceで検証している段階です。

**Database / Provider側の設計検証チェックポイントはM0.35まで進んでいます。** その後は、v1 Reference Applicationの主線としてApplication Patternの実証を進めています。

この文書では、Repository上で実際に実装・検証できていることと、まだ実装済みとは言えないことを分けて整理します。

TEIを初めて知る場合は[はじめてTEIを知る方へ](for-readers.md)、上位の価値や今後の方向を確認する場合は[TEI Product Value](../product/value/TEI_PRODUCT_VALUE.md)と[TEI Master Plan](../plan/MASTER_PLAN.md)を先に読むと、全体の位置づけを把握しやすくなります。

## 実装済みの主要な縦切り

### Semantic / Workspace

- `tei.workspace.yaml`
- Canonical IDとfile pathの分離
- Semantic namespace / type / data / relation / capability
- `std.*` prelude
- managed native filesの列挙
- Canonical Worldから分離したSource Index
- Source Artifact fingerprint / byte range / source role
- Semantic SubjectからSource ReferenceへのDiagnostic projection
- parse / lexer errorのsource-direct diagnostic
- `tei source-index` inspection CLI

### Application / Database

- Application `DatabaseRequirement[]`
- N:M Application ↔ Database
- logical datasource alias
- read / read-write Access
- Applicationが所有するPatternのCanonical model
- `applications/*/patterns/*.tei`のWorkspace Loader
- PatternのCanonical ID順Normalization
- `application-pattern:<Canonical ID>`のSource Indexと識別子range
- Pattern固有Validationの共通`validation.Validate`への統合
- Pattern Diagnosticからauthoring sourceへのprojection
- `browse` / `record` / `reference` PatternのParser・Validation・実Reference
- Database Contract
- Persistence Mapping
- provider-neutral Desired Physical Schema
- Actual Physical Schema
- Schema Diff
- 確定したChangeと比較未確定`Unresolved`の分離
- Foreign Key / Unique Constraint / Advanced IndexのDesired・Actual・Diff
- ProviderExpressionのauthor text / provider-normalized representation
- Desired Physical realizationのprovider-owned offline policy
- PostgreSQL向けAdvanced Index migration SQL rendering
- versioned DDL / data / verify Migration
- Migration Provider / applied-history reconciliation
- Database Release / Application Release / Solution Release

Application Patternでは、現時点で次の3種類を`order_entry`の実Referenceとして扱えます。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

`browse`では、Queryが保持するDatasourceをPatternへ重複記述せず利用できます。`record`はSemantic Data一件という最小の意味だけを保持します。`reference`はSemantic Relationが持つsource / target / cardinalityを複製せず、Application固有のlogical datasourceだけを組み合わせます。

3種類は同じ`order_entry`で共存し、Reference World、Source Index、Diagnostic、Workspace Validationを通して追跡できます。

次のApplication Pattern実証は`command`です。既存`sales.order.submit` Capabilityを、input / outputや実装方式をPatternへ複製せず参照できるかを検証します。その後、`OrderLine`と対応するSemantic Relationを追加し、`header-detail`へ進みます。

### Query / Workbench

- embedded SQL boundary
- native `.sql` authoritative source
- Query declarationとSQL authoring sourceのrole分離
- SQL analyzer stable protocol
- authoring pathを含まないAnalyzer request / derived boundary
- scope / field identity / lineage
- provider-neutral Query IR
- PostgreSQL schema observation
- PostgreSQL EXPLAIN normalization
- Database Workbench derived model

### Deployment

- release graph
- multi-DB compatibility
- active consumer withdrawal gate
- canary / blue-green / traffic shift
- accepting-work / drain / retire separation
- Durable Deployment Run
- evidence gate
- retry / pause / abort / forward-fix / rollback decision
- explicit Executor / Observation / Evidence routing

### Plugin / Security boundary

- Plugin / Bundle / Capability negotiation
- Plugin Lock
- in-process / process runtime boundary
- actual child processとのstdio protocol
- artifact verification contract
- explicit Permission Grant
- Resource Broker
- secret / network / filesystem logical binding
- Sandbox Profile
- Lease renew / refresh / revoke
- Host-side Lease Supervisor
- Supervisor Ownership / heartbeat / monotonic fencing
- Durable Invocation Handle / runtime observe / reattach
- Supervisor generationとruntime-confirmed generationの分離

## 現時点で実装済みとは言わないもの

次はProduct Visionに含まれますが、現時点で完成済みとは主張しません。

- production-ready GUI Editor / Wails application
- 5種類のApplication Patternを`order_entry`で一続きに扱うReference
  - `browse` / `record` / `reference`は実Referenceまで完了
  - `command` / `header-detail`は未完了
- Application設計を通しで実行するDriver
- target-neutral Presentation / Interaction IR
- production-readyなWeb / Desktop / Mobile Target
- 既存認証仕様を保持したCompatibility Reference
- AIによるDomain Component / Pattern探索
- AIの確認提案・判断履歴・提案抑制UX
- complete Workflow Runtime
- complete SQL dialect support
- production-grade distributed ownership store
- Kubernetes / Vault / Cloud provider integrations
- generic OS sandbox implementation
- production durability / HA guarantee
- すべてのDurable artifactに対する暗号学的署名
- API compatibility guarantee
- Go source semantic analysis / refactoring engine
- AIによるauthoritative sourceの無確認自動編集

Gap全体は[Product Requirement Gap Audit](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)に整理しています。

# 設計検証チェックポイント

## M0.22.1 — 最初の公開設計チェックポイント

M0.22.1では、Semantic / Application / DatabaseからRelease、Deployment、Plugin、Resource authority、Host takeover、外部runtime再接続までを一本の設計思想で接続できることを確認しました。

公開前のhardeningとして、次も整えました。

- Lease Supervisorのoperational clockをHostから注入可能にする
- Reference Workspace、golden World、observed state fixtureをRepositoryだけから再現可能にする
- Plugin LockとDurable Run fixtureの再生成経路を固定する
- Plugin / Runtimeの安全条件をConformanceで追跡する

詳細は[ADR-0002](adr/0002-m0221-public-checkpoint.md)と[M0.22.1 Plugin / Runtime Conformance](conformance/plugin-runtime-m0.22.1.md)を参照してください。

M0.22.1は現在地ではなく、公開可能な設計検証を始めた基準点です。

## M0.23 — Source-to-Canonical Traceability

Authoring sourceからCanonical Semantic Layerへ入る側の追跡性をvertical sliceとして実装しました。

- Source IndexをCanonical Worldから分離
- Canonical IDをfile pathから独立したまま維持
- content fingerprintと半開byte rangeを検証
- Parser source anchorをdomain modelから分離
- semantic diagnosticをpath-independentなSubjectからsourceへ投影
- parse diagnosticをsource-directに表示
- native `.sql`をauthoritative sourceのままQueryへ接続
- Query declaration=`definition`、SQL file=`authoring`というroleを区別
- physical pathをCanonical Query / Analyzer protocolへ混ぜない
- file rename / moveだけではnormalized Worldを変更しない
- derived artifactからauthoritative sourceへ暗黙に逆書きしない

Reference Workspaceの`order_entry.recent_orders`はnative SQLへ移行済みですが、この変更だけではnormalized `reference/world.json`を変えていません。

詳細は[M0.23 Conformance](conformance/source-traceability-m0.23.md)を参照してください。

## M0.24 — actual process Plugin runtime

Host注入interfaceだけだったprocess boundaryを、actual OS child processへ接続しました。

- `StdioProcessLauncher`でRequest / Response JSONをstdio transport
- shell semanticsを暗黙に導入しない
- context cancellationでchild processを終了
- malformed / oversized responseを拒否
- untrusted stdoutをbounded capture
- stderr本文をTEI errorへ無条件にコピーしない
- forged Plugin identityを拒否
- Repository内のReference process Pluginを実binaryとしてbuild可能

plain process transportをOS sandboxやDurable cross-host runtimeと同一視しません。

詳細は[M0.24 Conformance](conformance/process-runtime-m0.24.md)を参照してください。

## M0.25 — process executable / argv

process Pluginの実行指定をshell-likeなcommand lineではなく、単一executableと順序付きargvへ分離しました。

- `Runtime.Command`は単一executable
- `Runtime.Args`は順序付きargv
- legacyな空白入りcommandを暗黙splitしない
- executable / argvをHost境界まで構造化したまま維持
- args差分をDefinition / Plugin Lock / Runtime Policy fingerprintへ反映
- 空白を含む一つのargv elementがactual child processまで保存されることを確認

詳細は[M0.25 Conformance](conformance/process-command-argv-m0.25.md)を参照してください。

## M0.26 — actual PostgreSQL Desired / Actual / Diff

既存Database lifecycleをactual PostgreSQLへ接続しました。

```text
Persistence Mapping
    ↓
Desired Physical Schema

actual PostgreSQL
    ↓ introspection
Actual Physical Schema

Desired + Actual
    ↓
Schema Diff
```

- PostgreSQL向けDesiredを生成
- `database/sql`経由でactual catalogを観測
- same-stateで差分0を確認
- actual側の意図的なcolumn driftを検出
- credential / DSNをPhysical SnapshotやDiffへ保存しない
- DB driver assemblyをProvider Coreから分離

M0.26でautomatic schema applyを実装したわけではありません。

詳細は[M0.26 Conformance](conformance/postgres-actual-schema-m0.26.md)を参照してください。

## M0.27 — Foreign Key realization / drift

Foreign KeyをPersistence MappingのDesired realizationとprovider-neutral Schema Diffへ統合しました。

- Semantic RelationからFKを自動生成しない
- source Attribute、referenced Mapping、target Attributeを明示
- Mapping / Attribute identityからphysical FKへ解決
- cross-database FKは拒否
- constraint名をdrift identityにはしない
- 構造差を`drop-foreign-key` + `add-foreign-key`として表現
- composite FKのcolumn順序とmultiplicityを保持
- actual PostgreSQLでbaseline差分0とFK driftを確認

詳細は[ADR-0008](adr/0008-persistence-foreign-key-identity.md)と[M0.27 Conformance](conformance/foreign-key-drift-m0.27.md)を参照してください。

## M0.28 — Unique Constraint realization / drift

UNIQUE constraintをexplicit unique indexとは別のDatabase-owned physical intentとして扱う境界を実装しました。

- Persistence MappingからUnique Constraintを明示
- Attribute IDからphysical columnへDesired realization
- actual PostgreSQLの`UNIQUE` constraintを観測
- constraint backing indexを通常Indexへ二重計上しない
- explicit unique indexはIndexとして保持
- constraint名だけの差はdriftにしない
- ordered columnsとmultiplicityを保持
- actual UNIQUE constraint削除後に`add-unique-constraint`を検出

PostgreSQL内部でUNIQUE constraintがunique indexに支えられていても、constraint intentとaccess-path intentを同一視しません。

詳細は[ADR-0009](adr/0009-unique-constraint-and-unique-index.md)と[M0.28 Conformance](conformance/unique-constraint-m0.28.md)を参照してください。

## M0.29 — Advanced Index Actual / Diff

PostgreSQLのAdvanced Indexを、単純なcolumn listへ潰さずActual Physical Schemaへ保持できるようにしました。

- ordered column / expression key
- INCLUDE column
- partial predicate
- per-key collation / operator class
- ASC / DESC、NULL ordering
- unique indexのNULL treatment
- access method
- actual lifecycle state `valid / ready / live`

expression / predicateはCore共通SQL ASTへ変換せず、provider / dialect provenanceを持つ`ProviderExpression`として扱います。

Schema Diffでは、structural driftと`index-state-drift`を分離しました。

詳細は[ADR-0010](adr/0010-advanced-index-semantics.md)と[M0.29 Conformance](conformance/advanced-index-actual-diff-m0.29.md)を参照してください。

## M0.30 — Advanced Index Desired authoring

M0.29で観測・比較できるようになったAdvanced Indexを、Database-ownedなPersistence authoringからDesiredへ明示できるようにしました。

- simple Index authoringとの互換性を維持
- advanced block form
- ordered Attribute / expression key
- INCLUDE
- partial predicate
- sort / NULL ordering
- collation / operator class
- unique NULL treatment
- access method
- Source Index / Diagnostic projection

Advanced IndexをSemantic modelから自動推論せず、利用者が選んだphysical intentとしてPersistence Mappingに置きます。

PostgreSQL B-treeについてofflineで確定できるeffective defaultはDesired / Actualで同じ表現へ正規化します。一方、一般化にprovider固有知識が必要な部分は推測せず後続Issueへ分離しています。

詳細は[ADR-0011](adr/0011-advanced-index-authoring.md)と[M0.30 Conformance](conformance/advanced-index-authoring-m0.30.md)を参照してください。

## M0.31 — Schema Diffの比較未確定

Schema Diffに、確定したChangeとは別の`Unresolved`を導入しました。

ProviderExpressionのDesired author textとActual decompiled textが異なるとき、信頼できるnormalizationがなければ「同じ」「違う」のどちらにも断定しません。

```text
Diff
├─ Changes[]
└─ Unresolved[]
```

- provider / dialectが明確に違えば構造差
- sourceが同じなら一致
- 同じprovider / dialectで表記だけが異なり意味比較できなければUnresolved
- 明確な別構造があれば通常のChange
- lifecycle state driftは観測事実としてChange
- `Unresolved`が残るDiffからmigration SQLを生成しない

「比較できない」をmigration可能なChangeとして扱わないことが重要な境界です。

詳細は[M0.31 Conformance](conformance/schema-diff-unresolved-m0.31.md)を参照してください。

## M0.32 — ProviderExpression representation / comparison

`ProviderExpression`の文字列が、利用者が記述したものなのか、providerが内部表現から再構成したものなのかを明示しました。

```text
ProviderExpression
├─ authored
└─ provider-normalized
```

provider-normalized表現には、必要に応じてnormalization authorityとprovider versionのprovenanceを持たせます。

主な比較契約は次のとおりです。

- Desired authoringは`authored`
- PostgreSQL catalog由来は`provider-normalized`
- source textが完全一致すればrepresentationが異なっても一致
- authored同士の異なるsourceは異なるauthoring intentとして構造差
- authored / provider-normalizedで異なるsourceは、信頼できるnormalizerを通すまでUnresolved
- provider-normalized同士も、authority / versionが比較可能でなければ異なるtextを即構造差にしない
- unresolved comparisonからmigration SQLを生成しない

詳細は[ADR-0012](adr/0012-provider-expression-representation.md)と[M0.32 Conformance](conformance/provider-expression-representation-m0.32.md)を参照してください。

## M0.33 — Desired Physical realizationのoffline provider policy

Desired Physical Schemaを生成する処理から、DBMS固有の知識をruntime Provider SPIと切り離しました。

- `PhysicalRealizationPolicy`をPersistence側のportとして定義
- PostgreSQL / Oracleがそれぞれoffline policyを所有
- provider enumからpolicyを選ぶcomposition層を分離
- physical type、Index既定値、PostgreSQL B-tree normalizationをprovider側へ移動
- Desired realizationはlive DB接続を要求しない
- generic providerへ暗黙fallbackしない
- CLIと実DBintegrationを同じcomposition境界へ接続

この分離により、ProviderExpressionやIndex既定値の知識を増やしても、Persistence Coreや接続系runtime SPIへ無秩序に流入しにくくしました。

詳細は[ADR-0013](adr/0013-offline-physical-realization-policy.md)と[M0.33 Conformance](conformance/offline-physical-realization-policy-m0.33.md)を参照してください。

## M0.34 — プロバイダー固有Index識別子の比較

Advanced Indexのcollationとoperator classについて、表記差だけからIndex再作成を決めない比較契約を固定しました。

- Desiredで明示したcollationを根拠なく空値へ畳まない
- 両側に明示値があり一致すれば同一、異なれば構造差
- collationの片側だけが空なら比較未確定
- Desiredがdefault operator classを要求し、Actualが明示non-defaultなら構造差
- Desired明示・Actual default表現のoperator classは比較未確定
- `provider-identifier`のUnresolvedにfield pathとDesired / Actual値を保持
- 比較未確定から`DROP INDEX` / `CREATE INDEX`を生成しない

「offlineで確定できる既定値」と「列・型・catalog状態を見なければ確定できない識別子」を分けています。

詳細は[ADR-0014](adr/0014-provider-index-identifier-comparison.md)と[M0.34 Conformance](conformance/provider-index-identifier-m0.34.md)を参照してください。

## M0.35 — Advanced IndexのPostgreSQL migration SQL

Advanced Indexの主要なPhysical semanticsを、PostgreSQL固有のDDLへ意味を失わず戻せるようにしました。同時に、SQL文法の責務を共通migration層からPostgreSQL provider packageへ移しています。

- expression key / ordered key
- `COLLATE` / operator class
- `ASC` / `DESC`、`NULLS FIRST` / `NULLS LAST`
- `INCLUDE`
- partial predicate
- unique indexの`NULLS NOT DISTINCT`
- B-tree以外のaccess method
- schema-qualifiedな`DROP INDEX`
- schemaを確定できない破壊的`DROP INDEX`を拒否
- ProviderExpressionのprovider / dialect / representationを検査
- ProviderExpression本文を推測で書き換えない
- `CONCURRENTLY`を通常migrationで自動選択しない
- `index-state-drift`を自動drop/recreateへ変換しない

CIではPostgreSQL 17を使い、Desired → Diff → SQL生成 → 実DBへ適用 → Introspect → 再比較を一周させ、Advanced Indexを作成した後に`Changes = 0 / Unresolved = 0`へ収束することを確認しています。

詳細は[ADR-0015](adr/0015-provider-owned-migration-sql-rendering.md)と[M0.35 Conformance](conformance/advanced-index-migration-rendering-m0.35.md)を参照してください。

## M0.35以後 — v1 Application Patternの実証

M0.35のDatabase / Provider検証後、v1 Reference Applicationの主線であるApplication Patternへ実装の重点を移しています。この節は新しいM0.xxチェックポイントを宣言するものではなく、M0.35以後の現在地を記録するためのものです。

### 共通基盤

PR #87〜#103で、Application PatternをReferenceとして扱うための基盤を段階的に閉じました。

- Applicationが所有する`Pattern`のCanonical model
- Pattern Parser
- `applications/*/patterns/*.tei` Loader
- Canonical ID順Normalization
- `application-pattern:<Canonical ID>`のSource Index
- Pattern ID tokenを指すsource range
- Pattern identityとfile pathの分離
- Workspace全体の参照整合性Validation
- Pattern固有Diagnosticのsource projection
- 共通`validation.Validate`への統合
- `browse`実ReferenceとGolden World
- 実Referenceのfile rename invariance

### `record`

Issue #104、PR #105 / #106で、`sales.order`を一件として扱う`record`をData-onlyの最小契約として実証しました。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

Query、Datasource、特定`browse`との遷移関係をPatternへ追加しなくてもReferenceが成立したため、現時点ではそれらをCanonical fieldに含めていません。

### `reference`

Issue #107、PR #108 / #109で、Semantic Relation `sales.order.customer`をlogical datasource `master`から参照する`reference`を実証しました。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Relationが持つsource / targetをPatternへ複製せず、Datasource accessはApplication定義を正本とします。

`reference`ではwrite-only Datasourceを拒否し、read / read-writeを読み取り可能として扱います。実Referenceでは、`master_data.customer ^2` ContractがRelation target `customer.customer`をread対象として公開していることもConformanceで追跡しています。

### 次の主線

次は`command`として`sales.order.submit` Capabilityへの接続を実証します。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

第一候補では、Capabilityのinput / output、Datasource、実装方式をPatternへ複製しません。Capability結果はADR-0020のBusiness Result / Execution Error境界を利用し、`command`追加だけを理由に既存`output sales.order`を名前付きOutcomeへ変更しません。

`command`の後は、`OrderLine`と対応Relationを追加して`header-detail`を検証します。

この進捗だけでStage 1、Gate A、Milestone 1の完了とはみなしません。5種類のPattern候補と既存認証Compatibilityを含め、後続Driverへ渡せるApplication上の意味が揃ってからGate Aを評価します。

詳細は[Application Pattern v1 設計・実装基線](../plan/APPLICATION_PATTERN_V1_DESIGN.md)、[ISSUE-000005](../plan/issue/000005-application-pattern-reference.md)、[ISSUE-000010](../plan/issue/000010-command-application-pattern.md)を参照してください。

# 継続的な検証

常設CIでは、少なくとも次を確認します。

### 基本検証

- `gofmt`
- `go vet ./...`
- `go test ./...`
- Reference Workspace validation

### Reference再現性

- Reference Query / Application PatternのSource Index projection
- `reference/world.json`の再生成一致
- Plugin Lockの再生成一致
- Durable Deployment Run fixtureの再生成一致

### 実行境界

- Deployment executor参照例
- Reference process Pluginの実binary build
- DeploymentExecutorからactual child processへのstdio protocol
- executable / argvのend-to-end保持
- runtime reattach参照例

### actual PostgreSQL

PostgreSQL 17 serviceを使い、Reference integrationを実行します。

少なくとも次の経路はunit / integration testを通して継続的に検証されます。

- Desired / Actual / Diff
- column drift
- Foreign Key
- Unique Constraintとconstraint backing indexの分離
- View / Materialized View column observation
- Advanced Indexの観測・比較
- ProviderExpression / provider identifier / Unresolvedの安全な比較
- Advanced Index migration SQLの生成・実行・再観測・収束
- connection dataをPhysical Snapshot / Diffへ保存しないこと

CIはproduction-gradeな外部環境を保証するものではありません。Reference実装の意味境界と再現性がRepository内で崩れていないことを継続して確認するものです。

# 現在残っている主な境界

## Product側の大きなGap

Database / Provider側の設計検証はM0.35まで進み、Application Patternも`browse`、`record`、`reference`の実Referenceまで進みました。

一方、次は引き続き実証が薄い領域です。

- `command` / `header-detail`を含むApplication Pattern Stage 1の完了
- `OrderLine`を含む最小header-detail Semantic
- 既存認証・認可を保持したCompatibility
- Application Driver
- target-neutral Presentation / Interaction
- 第一Web Target
- 開発者向け高密度Editor
- AIによる知識探索、確認提案、判断履歴

今後は`sales.order.submit`を`command` Patternから参照する実証を進め、その後`OrderLine` / `header-detail`へ進みます。Application Pattern側と既存認証Compatibility側のEvidenceが揃ったところでGate Aを評価し、その後Application Driver → 第一Web Targetへ進みます。

詳細は[Product Requirement Gap Audit](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)、[Master Plan](../plan/MASTER_PLAN.md)、[v1 Reference Application実証計画](../plan/V1_REFERENCE_APPLICATION_PLAN.md)を参照してください。

## Database / Provider側の継続課題

- Partitioned table / child partitionのSchema Drift semantics
- PostgreSQL以外のProviderによる抽象境界の圧力試験
- production環境でのonline index build / recovery policy
- provider versionや拡張catalogを含むnormalization authorityの高度化

これらは重要ですが、PostgreSQL固有機能を網羅すること自体を目的にはしません。Product全体のGapと比較しながら着手順を決めます。

## Production境界

- migration proposalからactual DB schema changeへ至る明示的execution integration
- durable process manager / container runtimeとのcross-host reattach integration
- production-grade Host sandbox / process isolation adapter
- Deploymentの外部実環境executor integration
- production durability / HA / compatibility policy

これらを進める場合も、外部製品固有adapterの都合を先にCanonical modelへ持ち込まず、TEI固有のidentity / meaning / execution / observation境界を保つ方針を維持します。
