# 現在の実装状況

TEIはまだ完成したローコード製品ではなく、設計境界を実行可能なコードで検証している段階です。

**現在の設計検証チェックポイントはM0.32です。**

この文書では「現在、Repository上で何を実装・検証できているか」と、「まだ何を実装済みとは言えないか」を分けて整理します。

TEIを初めて知る場合は [はじめてTEIを知る方へ](for-readers.md)、上位の価値や今後の方向を確認する場合は [TEI Product Value](../product/value/TEI_PRODUCT_VALUE.md) と [TEI Master Plan](../plan/MASTER_PLAN.md) を先に読む方が自然です。

## 実装済みの主要縦切り

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
- Database Contract
- Persistence Mapping
- provider-neutral Desired Physical Schema
- Actual Physical Schema
- Schema Diff
- 確定したChangeと比較未確定`Unresolved`の分離
- versioned DDL / data / verify Migration
- Migration Provider / applied-history reconciliation
- Database Release / Application Release / Solution Release

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

次はProduct Visionに含まれますが、M0.32時点で完成済みとは主張しません。

- production-ready GUI Editor / Wails application
- standard Application Patternの実装
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

このGapの全体像は [Product Requirement Gap Audit](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md) に整理しています。

# 設計検証チェックポイント

## M0.22.1 — 最初の公開設計チェックポイント

M0.22.1では、Semantic / Application / DatabaseからRelease、Deployment、Plugin、Resource authority、Host takeover、外部runtime再接続までを一本の設計思想で接続できることを確認しました。

公開前のhardeningとして、次も整えました。

- Lease Supervisorのoperational clockをHostから注入可能にする
- Reference Workspace、golden World、observed state fixtureをRepositoryだけから再現可能にする
- Plugin LockとDurable Run fixtureの再生成経路を固定する
- Plugin / Runtimeの安全条件をConformanceで追跡する

詳細は [ADR-0002](adr/0002-m0221-public-checkpoint.md) と [M0.22.1 Plugin / Runtime Conformance](conformance/plugin-runtime-m0.22.1.md) を参照してください。

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

Reference Workspaceの`order_entry.recent_orders`はnative SQLへ移行済みですが、normalized `reference/world.json`は変更していません。

詳細は [M0.23 Conformance](conformance/source-traceability-m0.23.md) を参照してください。

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

詳細は [M0.24 Conformance](conformance/process-runtime-m0.24.md) を参照してください。

## M0.25 — process executable / argv

process Pluginの実行指定をshell-likeなcommand lineではなく、単一executableと順序付きargvへ分離しました。

- `Runtime.Command`は単一executable
- `Runtime.Args`は順序付きargv
- legacyな空白入りcommandを暗黙splitしない
- executable / argvをHost境界まで構造化したまま維持
- args差分をDefinition / Plugin Lock / Runtime Policy fingerprintへ反映
- 空白を含む一つのargv elementがactual child processまで保存されることを確認

詳細は [M0.25 Conformance](conformance/process-command-argv-m0.25.md) を参照してください。

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

詳細は [M0.26 Conformance](conformance/postgres-actual-schema-m0.26.md) を参照してください。

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

詳細は [ADR-0008](adr/0008-persistence-foreign-key-identity.md) と [M0.27 Conformance](conformance/foreign-key-drift-m0.27.md) を参照してください。

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

詳細は [ADR-0009](adr/0009-unique-constraint-and-unique-index.md) と [M0.28 Conformance](conformance/unique-constraint-m0.28.md) を参照してください。

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

詳細は [ADR-0010](adr/0010-advanced-index-semantics.md) と [M0.29 Conformance](conformance/advanced-index-actual-diff-m0.29.md) を参照してください。

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

詳細は [ADR-0011](adr/0011-advanced-index-authoring.md) と [M0.30 Conformance](conformance/advanced-index-authoring-m0.30.md) を参照してください。

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

詳細は [M0.31 Conformance](conformance/schema-diff-unresolved-m0.31.md) を参照してください。

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

詳細は [ADR-0012](adr/0012-provider-expression-representation.md) と [M0.32 Conformance](conformance/provider-expression-representation-m0.32.md) を参照してください。

# 継続的な検証

常設CIでは、少なくとも次を確認します。

### 基本検証

- `gofmt`
- `go vet ./...`
- `go test ./...`
- Reference Workspace validation

### Reference再現性

- Reference QueryのSource Index projection
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
- ProviderExpression / Unresolvedの安全な比較
- connection dataをPhysical Snapshot / Diffへ保存しないこと

CIはproduction-gradeな外部環境を保証するものではありません。Reference実装の意味境界と再現性がRepository内で崩れていないことを継続して確認するものです。

# 現在残っている主な境界

## Product側の大きなGap

M0.32までの下部構造に比べ、次はまだ実証が薄い領域です。

- standard Application Pattern
- 既存認証・認可を保持したCompatibility
- Application Driver
- target-neutral Presentation / Interaction
- 第一Web Target
- 開発者向け高密度Editor
- AIによる知識探索、確認提案、判断履歴

今後は既存`order_entry`をv1 Reference Applicationとして育て、Application Pattern → Driver → 第一Web Targetを主線として進めます。

詳細は [Product Requirement Gap Audit](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md) と [Master Plan](../plan/MASTER_PLAN.md) を参照してください。

## Database / Provider側の継続課題

- Partitioned table / child partitionのSchema Drift semantics
- provider default / identifier normalization
- Desired Physical realizationのprovider-owned offline policy
- Advanced Index migration SQL rendering
- PostgreSQL以外のProviderによる抽象境界の圧力試験

これらは重要ですが、PostgreSQL固有機能を網羅すること自体を目的にはしません。Product全体のGapと比較しながら着手順を決めます。

## Production境界

- migration proposalからactual DB schema changeへ至る明示的execution integration
- durable process manager / container runtimeとのcross-host reattach integration
- production-grade Host sandbox / process isolation adapter
- Deploymentの外部実環境executor integration
- production durability / HA / compatibility policy

これらを進める場合も、外部製品固有adapterの都合を先にCanonical modelへ持ち込まず、TEI固有のidentity / meaning / execution / observation境界を保つ方針を維持します。
