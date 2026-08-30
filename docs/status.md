# 現在の実装状況

TEIはまだ完成したローコード製品ではなく、設計境界を実行可能なコードで検証している段階です。

この文書は「現在、Repository上で何を実装・検証できているか」を確認するための一覧です。TEIを初めて知る場合は、先に [はじめてTEIを知る方へ](for-readers.md) を読む方が自然です。

## 実装済みの主要縦切り

### Semantic / Workspace
- `tei.workspace.yaml`
- Canonical IDとpathの分離
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
- Database Contract
- Persistence Mapping
- provider-neutral desired schema
- Schema Diff
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
- artifact verification contract
- explicit Permission Grant
- Resource Broker
- secret / network / filesystem logical binding
- Sandbox Profile
- Lease renew / refresh / revoke
- Host-side Lease Supervisor
- Supervisor Ownership / heartbeat / monotonic fencing
- Durable Invocation Handle / runtime observe / reattach

M0.22.1で直接確認しているPlugin / Runtime境界の保証は、[Conformance — M0.22.1](conformance/plugin-runtime-m0.22.1.md) に整理しています。

M0.23で直接確認しているSource / meaning境界の保証は、[Conformance — M0.23 Source-to-Canonical Traceability](conformance/source-traceability-m0.23.md) に整理しています。

## 現時点での重要な非目標

以下を実装済みとは主張しません。

- production-ready GUI Editor
- complete Workflow Runtime
- complete SQL dialect support
- production-grade distributed ownership store
- Kubernetes / Vault / Cloud provider integrations
- generic OS sandbox implementation
- production durability / HA guarantee
- cryptographic signing of all Durable Run artifacts
- API compatibility guarantee
- Go source semantic analysis / refactoring engine
- AIによるauthoritative sourceの自動編集

## 最初の公開設計チェックポイント

最初の公開設計チェックポイントは、**M0.22.1: Durable Invocation Handle / Runtime Reattach + publication hardening**です。

M0.22でHost takeover後のruntime reattachまで実装した後、公開前のhardeningとしてLease Supervisorのoperational clockをHostから注入可能にしました。これにより、Resource Brokerと同じ時間基準を使ってdeterministicに検証できます。

さらに、Repository自体も一時的なsource transportではなく、通常のGoソースツリーへ正規化しました。Reference Workspace、golden world、observed state fixtureを含め、cloneした環境だけで検証できる状態を基準にします。

M0.22.1では、Semantic / Application / Databaseのモデルから、Release、Deployment、Plugin、Resource authority、Host takeover、外部runtime再接続まで、一つの設計思想で縦に接続できることを確認しています。

この位置づけの詳細は [ADR-0002](adr/0002-m0221-public-checkpoint.md) に記録しています。

## M0.22.1公開後のhardening

公開チェックポイントを成立させた後も、M0.22.1の意味を広げるのではなく、再現性と安全境界の監査を進めています。

### Reference artifactの再現性

Issue #3 / PR #8で、Reference Workspaceから導出されるartifactとfixtureの関係を監査しました。

- `reference/world.json` をWorkspaceから再生成可能
- `reference/workspace/.tei/plugin-lock.json` をWorkspaceから再生成可能
- baseline observed stateからDurable Deployment Run fixtureを再生成可能
- `reference/README.md` でfixtureの役割と再生成方法を説明
- cloneしたRepositoryだけでDeployment executorの参照例まで実行可能

### Plugin / Runtime Conformance

Issue #4 / PR #10で、Plugin / Runtime境界の主要な安全条件を「設計意図・実装箇所・検証箇所」で追跡できるようにしました。

監査中には、正常なSupervisor shutdown由来の`context canceled`をheartbeat failureと誤認するtiming依存の問題も見つかりました。これはIssue #11 / PR #12で、正常shutdownと実際のownership store failureを区別するよう修正済みです。

`RuntimeUnknown`時のfencing authorityについてはIssue #9 / ADR-0005で整理しました。Durable control plane上のSupervisor generationと、外部runtimeが実際に受理済みのgenerationを分離し、`FencingToken`と`RuntimeFencingToken`として記録します。unknownでは前者だけを進め、runtime-confirmed generationはlive reattach成功時にだけ追いつかせます。

## M0.23 Source-to-Canonical Traceability

M0.23では、実行側adapterをさらに増やす前に、Authoring sourceからCanonical Semantic Layerへ入る側の追跡性をvertical sliceとして実装しました。

設計判断は [ADR-0004](adr/0004-source-to-canonical-traceability.md) に記録し、Issue #16〜#19で段階的に検証しています。

### 到達した境界

- Source IndexをCanonical Worldから分離
- Canonical IDをfile pathから独立したまま維持
- source content fingerprintと半開byte rangeを検証
- Parser source anchorをdomain modelから分離
- semantic diagnosticをpath-independentなSubjectからsourceへ投影
- parse diagnosticをCanonical subjectなしでsource-directに表示
- `tei validate -workspace`で解決可能なfindingを`path:line:column`へ復元
- native `.sql`をauthoritative sourceのままQueryへ接続
- Query declaration=`definition`、SQL file=`authoring`というroleをSource Indexで区別
- physical `.sql` pathをCanonical Query / Analyzer protocolへ混入させない
- file rename / moveだけではnormalized Worldを変更しない
- source move後はDiagnostic projectionが新locatorへ追随
- derived artifactからauthoritative sourceへ暗黙に逆書きするAPIを設けない
- `tei source-index`でsubjectからsource reference / fingerprintをinspection可能

Reference Workspaceの`order_entry.recent_orders`はnative SQLへ移行済みですが、`reference/world.json`は変更していません。旧embedded SQLと同じnormalized meaningを再生成できることをCIで確認しています。

Source Index全体のgolden snapshotは保存せず、deterministic generation、range / fingerprint整合、重要subjectのrole / locator、rename invarianceをコードとCIで固定します。

詳細な対応表は [M0.23 Conformance](conformance/source-traceability-m0.23.md) を参照してください。

## M0.24 実process Plugin runtime

M0.24では、`BoundaryProcess`をHost注入interfaceだけに留めず、actual OS child processへ接続する最小vertical sliceを実装しました。

- `StdioProcessLauncher`でRequest / Response JSONをstdio transport
- commandは単一executable名/pathとして実行し、shell semanticsを暗黙に導入しない
- context cancellationで直接child processを終了
- malformed / oversized responseを拒否
- untrusted stdoutをReference default 8 MiBでbounded capture
- stderr本文をTEI errorへコピーせず、異常終了時はbyte数とSHA-256だけを保持
- forged Plugin identityを既存Registry validationで拒否
- `tools/reference-process-plugin`を実binaryとしてbuild可能
- `examples/process-plugin`でDeploymentExecutor → Registry → actual child processまで再現
- Plugin Invocation provenanceで`boundary=process`を検証

`StdioProcessLauncher`はplain process transportでありOS sandboxそのものではありません。`SandboxedProcessRunner`として再利用する場合も、policy適用は`HostSandboxLauncher`側の責務です。

また、このone-shot local processのPIDをDurable Runtime Handleとはみなしません。Host restart / cross-host reattachはM0.22 / ADR-0005のopaque durable handle契約を満たす別runtime adapterの責務として残します。詳細は [M0.24実process Conformance](conformance/process-runtime-m0.24.md) を参照してください。

## M0.25 process executable / argv

M0.25では、process Pluginの実行指定をshell-likeなcommand lineではなく、単一executableと順序付きargvへ分離しました。設計判断は [ADR-0006](adr/0006-process-command-and-argv.md) に記録しています。

- authoring grammarで `runtime process "python3" args "-m" "worker"` を表現
- `Runtime.Command`は単一executable、`Runtime.Args`は順序付きargv
- legacyな空白入りcommandを自動splitせず、明示argsへの移行を要求
- `ProcessCommand{Executable, Args}`をProcess / Sandbox境界で維持
- args差分をDefinition / Plugin Lock / Runtime Policy fingerprintへ反映
- Reference Workspaceのprocess descriptorをargs利用へ移行
- 空白を含む一つのargv elementがactual child processの`os.Args`まで保存されることをReference実process例で検証

詳細は [M0.25 Conformance](conformance/process-command-argv-m0.25.md) を参照してください。

## M0.26 actual PostgreSQL Desired / Actual / Diff

M0.26では、既存Database lifecycleをactual PostgreSQLへ接続しました。新しいCanonical DB modelを増やさず、既存のPersistence Mapping、PostgreSQL provider、Physical Schema、Schema Diffを一続きで検証しています。

- `persistence.RealizeDesired`でPostgreSQL向けDesired Physical Schemaを生成
- `database/sql` + `SQLCatalogReader`でactual PostgreSQL catalogを観測
- `provider.Introspect`を通してDatabaseInstance / Provider整合を検証
- same-state fixtureで`schemadiff.Compare`のchangesが0件
- actual側だけに追加した`drift_note` columnをdriftとして検出
- actual foreign keyをSchemaSnapshotへ観測
- Primary Key backing indexを通常Indexとして二重計上しないようPostgreSQL providerをnormalize
- DSN / credentialをPhysical SnapshotやDiffへ保存しない
- Reference driver選択はtool assemblyに留め、PostgreSQL provider SPIは`database/sql`依存のまま維持

fixture setup DDLはintegration harnessの責務であり、M0.26でautomatic schema applyやmigration executionを実装したわけではありません。詳細は [M0.26 Conformance](conformance/postgres-actual-schema-m0.26.md) を参照してください。

## M0.27 Foreign Key realization / drift

M0.27では、M0.26でactual PostgreSQLから観測できることまで確認したForeign Keyを、Persistence MappingのDesired realizationとprovider-neutral Schema Diffへ統合しました。設計判断は [ADR-0008](adr/0008-persistence-foreign-key-identity.md) に記録しています。

- Foreign KeyをSemantic Relationから自動生成せず、Database所有のPersistence Mappingで明示
- authoringではsource Attribute、referenced Mapping、target Attributeをidentityで指定
- `RealizeDesired`がMapping / Attribute identityからphysical table / columnへ解決
- cross-database FKをM0.27では拒否
- constraint名はtrace / DDL metadataとして保持するが、Schema Driftの意味identityにはしない
- FKの構造差を`drop-foreign-key` + `add-foreign-key`として表現
- composite FKのcolumn順序を保持
- 同一構造constraintのmultiplicityを保持
- actual PostgreSQLでFK込みbaseline差分ゼロを確認
- actual constraint削除後に`add-foreign-key` driftを検出

詳細は [M0.27 Conformance](conformance/foreign-key-drift-m0.27.md) を参照してください。

## 継続的な検証

常設CIでは、少なくとも次を確認します。

- Goコードに`gofmt`が不要なこと
- `go vet ./...`
- `go test ./...`
- Reference Workspaceのvalidation
- Reference QueryのSource Index projectionで`definition` / `authoring` sourceとfingerprintを追跡できること
- `reference/world.json` がWorkspaceから再生成して一致すること
- Plugin LockがWorkspaceから再生成して一致すること
- Durable Deployment Run fixtureがReference Workspaceとobserved stateから再生成して一致すること
- Deployment executorの参照例が動作すること
- Reference process Pluginを実binaryへbuildできること
- DeploymentExecutorからactual child processへstdio protocolで到達できること
- executable / argvをshell-like stringへ畳まず、空白を含む一つのargv elementがactual child processまで保存されること
- runtime reattachの参照例が動作すること
- actual PostgreSQLからPhysical Schemaをintrospectし、Foreign Key込みbaseline差分ゼロ、column drift、Foreign Key driftを確認できること

CIはproduction-gradeな外部環境を検証するものではありません。Reference実装の意味境界と再現性がRepository内で崩れていないことを継続して確認するものです。

## 次に残る境界

M0.23でSource / meaning境界を厚くした後も、次の候補が残ります。

- Foreign Key option（ON DELETE / ON UPDATE等）をprovider-neutral modelへ持ち上げる必要性の検討
- migration proposalからactual DB schema changeへ至る明示的execution integration
- durable process manager / container runtimeとのcross-host reattach integration
- production-grade Host sandbox / process isolation adapter
- Deploymentの外部実環境executor integration
- Source Indexを利用するGUI / LSP authoring experience
- Go等、SQL以外のnative source semantic analysis

これらを進める場合も、外部製品固有adapterやGUIの都合を先にCanonical modelへ持ち込まず、TEI固有のidentity / source / execution境界を先に固定する方針を維持します。
