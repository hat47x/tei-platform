# TEI（鼎／テイ）

TEIは、**業務上の意味を長く保ちながら、実装・データベース・配備先を必要以上に一体化しないローコード基盤**を目指す実験的なプロジェクトです。

画面やテーブルを素早く作ることだけでなく、「この業務概念は何を意味するのか」「どの実装へ落ちたのか」「本番で実際に何が起きたのか」を、技術が変わっても追跡できる状態を重視しています。

> **現在地:** M0.35 / architecture & product validation / research implementation  
> Database / Provider側ではM0.35まで設計境界を検証し、その後のv1主線では`order_entry`を使ったApplication Pattern Stage 1を完了しました。現在はApplication Driverへ進む前のGate Aを監査し、既存認証Compatibility、Capability Outcome、業務Validation、Capability Bindingの残条件を実証している段階です。完成したEditorや本番用Runtimeを提供する段階ではありません。

## どこから読むか

目的に応じて入口を分けています。

### TEIが何を目指すか

- **最も上位の価値を知る:** [TEI Product Value](product/value/TEI_PRODUCT_VALUE.md)
- **成熟した姿を知る:** [TEI Product Vision](product/vision/TEI_PRODUCT_VISION.md)
- **設計判断の基準を知る:** [TEI Product Principles](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- **プロダクトとして必要なことを確認する:** [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- **期限を置かない開発の到達点を確認する:** [TEI Milestones](plan/MILESTONES.md)
- **開発全体の進め方を確認する:** [TEI Master Plan](plan/MASTER_PLAN.md)
- **Application Driverへ進めるかを確認する:** [Gate A 到達性監査](plan/GATE_A_READINESS_AUDIT.md)

### TEIを読み、動かす

- **TEIを初めて知る:** [はじめてTEIを知る方へ](docs/for-readers.md)
- **考え方をもう少し知る:** [TEIの考え方](docs/overview.md)
- **実際に動かす:** [Getting Started](docs/getting-started.md)
- **技術構造を追う:** [Architecture](docs/architecture.md)
- **実装済み／未実装を確認する:** [現在の実装状況](docs/status.md)

最初からコードを読む必要はありません。Product Value、一般向けの説明、技術構造、現在の実装状況を分け、必要な深さから読めるようにしています。

## TEIが解こうとしていること

業務システムは、画面やテーブルだけでは表せない多くの意味を持っています。

- 「受注」「顧客」「請求」のような業務概念
- 有効期間、履歴、承認、状態遷移などの共通パターン
- 一つのApplicationが複数Databaseを利用する現実
- Database SchemaとApplication Releaseの独立したライフサイクル
- SQL、Go、外部ツール、クラウド等の実装方式
- 本番で何が実際に起きたかというObservation / Evidence

従来は、こうした意味が画面定義、SQL、プログラム、DB制約、運用手順書、担当者の知識へ分散しがちでした。

TEIでは、それらを一つの巨大な独自DSLへ押し込めるのではなく、**意味・実現・実行・観測を分けたまま相互に追跡できる状態**を目指します。

```text
Intent / Semantic Definition
        ↓
Canonical Model
        ↓
Realization / Provider Binding
        ↓
Application / Database / Query / Deployment
        ↓
Observed State + Evidence
        ↓
Reconciliation / Recovery
```

ここでCanonical Modelは「特定の画面やDB製品に依存しない、基準となる意味表現」、Realization / Bindingは「その意味を具体的な技術や環境へ結び付ける段階」を指します。

より上位の定義では、TEIの中心価値を次のように置いています。

> 業務の意味を技術から独立して保ち、現実の複雑さを削らずに実装へ結び付け、その関係を人と生成AIの双方が追跡・検証しながら選び直せる開発基盤を目指す。

詳しくは [TEI Product Value](product/value/TEI_PRODUCT_VALUE.md) を参照してください。

## 設計上の特徴

### 1. ApplicationとDatabaseを一対一にしない

一つのApplicationが複数Databaseを利用することを通常形として扱います。

```text
order_entry
├─ sales  → sales_core   (read-write)
└─ master → master_data  (read-only)
```

Application Releaseは必要なDatabase Contractの集合を保持し、Database Releaseは独立してversion管理されます。

Reference Workspaceの`order_entry`も、この2 Database構成を実際に利用しています。

### 2. SQLやGoを敵にしない

TEI-native DSLは存在しますが、すべての資産を`.tei`へ変換する方針ではありません。

```text
*.tei   semantic / lifecycle definitions
*.sql   SQL
*.go    custom implementation
*.json  observed / derived artifacts
*.yaml  workspace / external configuration
*.md    documentation
```

M0.23では、native `.sql`をauthoritative sourceのままQueryへ接続し、physical pathをCanonical QueryやDerived Query IRへ混ぜないことをReferenceとCIで確認しました。

Workspaceは`tei.workspace.yaml`と管理ディレクトリで識別します。

### 3. 生成AIに「意味の部品」を渡せる構造

AIが毎回ゼロからコードを生成するのではなく、既存の業務概念、意味パターン、Policy、Capabilityを先に探索し、それでも足りない部分だけを実装へ落とす方向を想定しています。

AIが常時本番判断を行うことや、自動的に強い実行権限を持つことを前提にはしていません。採用した規則、Binding、権限、実現方法は、明示的で追跡可能な状態へ固定することを重視します。

現在は、このAI支援より先に、AIやEditorが利用できるApplication上の意味をv1 Referenceから実証しています。Application Pattern Stage 1では`browse`、`record`、`reference`、`command`、`header-detail`の5種類が実Referenceまで成立しました。次は、業務Validation、Authentication / Authorization、Capability BindingなどをGate Aで接続し、Driverへ渡せる状態を確認します。

### 4. 「命令した」と「実際に変わった」を分ける

Deployment Executorが成功を返しても、それだけではAction完了にしません。

```text
command accepted
    ≠
external state changed

accepted
  ↓
Observation Provider
  ↓
actual state confirmed
  ↓
completed
```

Canaryのhealth gate、drain、Database migration、rollback / forward-fixも同じ考え方で扱います。

M0.31以降では、Database Schemaについて「差がある」と「比較に必要な情報が足りず、まだ確定できない」を分離しています。不明なものを都合よく一致や差分へ丸めません。

M0.35では、この安全境界を保ったままAdvanced Indexの主要な意味をPostgreSQLのマイグレーションSQLへ戻し、生成したDDLを実DBへ適用して再観測後にDesiredへ収束するところまでCIで確認しています。

### 5. Pluginは意味モデルではなく配布単位

Plugin / BundleはExecutor、Database Provider、Migration Provider、Editor Intelligence等を配布するための仕組みです。Business Canonical IDとは分離しています。

Applicationから見たCapability identityと、その実現方法も分離します。現在のGate Aでは、`sales.order.submit`を例に、Capability identityからimplementationへ暗黙推測せず接続できるBinding境界を次の実証対象の一つとしています。

### 6. 実行時権限を最小化する

process Pluginには、必要なCapabilityへ必要なPermissionだけを明示Grantします。Secret値はCanonical DefinitionやDurable Runへ保存せず、Host Resource Brokerがopaque handleとして解決します。

Lease renewal、refresh、revoke、Supervisor ownership、fencing、Host takeover、runtime reattachまで、外部Runtimeを安全に扱う境界を実験実装しています。

Application利用者のAuthentication / Authorizationは、このPlugin / Runtime Permissionとは別の意味として扱います。

## 現在の重点

M0.35までに、Source / Meaning、Database、Execution / Observation、Plugin / Runtimeの下部構造にはかなり強いEvidenceが蓄積しました。Database領域では、Actual / Desired / Diff、比較未確定、ProviderExpression、Advanced Index、プロバイダー固有のマイグレーションSQL生成まで一連の境界を実PostgreSQLで検証しています。

その後、v1 Reference Application `order_entry`でApplication Pattern Stage 1を進め、次の5種類を実Referenceまで成立させました。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

この結果により、「standard Application Patternが未実証」という当初の大きなGapは、少なくともv1 Referenceの具体例では一段縮小しました。ただし、この5分類をTEI全体の永久的な標準語彙として確定したわけではありません。

現在の主線はGate Aです。[Gate A 到達性監査](plan/GATE_A_READINESS_AUDIT.md)では、Application Driverへ進む前の主な残条件として次を特定しています。

- 既存認証・認可を保持したCompatibility Reference
- Capabilityの名前付きOutcomeとExecution Errorの実Reference
- 業務Validationの正本とidentity
- `sales.order.submit`の明示的なCapability Binding

これらをDriver専用GoコードやTarget固有設定へ埋め込まず、Application上の意味として成立させたうえで、**Gate A再監査 → Application Driver → 第一Web Target**へ進みます。

Database / Provider側の検証を止めるのではなく、プロダクト全体の空白と比較しながら着手順を判断します。

## まず動かす

ここから先は開発者向けです。clone後に必要なのはGo toolchainです。

```bash
git clone https://github.com/hat47x/tei-platform.git
cd tei-platform

go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Validationでは、参照用のhistorical contractについてWarningが出る場合がありますが、最後に`ok`となればReference Workspaceは整合しています。

Source Indexは次のように確認できます。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

Host takeover後のruntime reattachの最小例は次で実行できます。

```bash
go run ./examples/runtime-reattach
```

Plugin解決やReference artifactの再生成を含む手順は [Getting Started](docs/getting-started.md) を参照してください。

## Repository layout

```text
product/                 Product Value / Vision / Principles
requirements/            Product Requirements
plan/                    Milestones / Master Plan / Gate
plan/design/             機能ごとの設計文書
plan/issue/              作業定義と要件詳細化
plan/adr/                設計判断
cmd/tei/                 CLI
internal/                executable architecture models / validation
reference/workspace/     small end-to-end TEI workspace
reference/observed/      deployment observation fixtures
reference/world.json     golden canonical world snapshot
examples/                focused executable examples
docs/                    public and technical documentation
.github/workflows/       repository verification
AGENTS.md                文書構成と管理ルール
```

上位・計画文書:

- [Product文書の入口](product/README.md)
- [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [TEI Planning](plan/README.md)
- [Product Requirement Gap Audit](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- [Gate A 到達性監査](plan/GATE_A_READINESS_AUDIT.md)

技術文書:

- [Architecture](docs/architecture.md)
- [現在の実装状況](docs/status.md)
- [既存の設計判断記録（ADR）](docs/adr/README.md)
- [Plugin / Bundle](docs/plugins.md)
- [Security / Trust Boundary](docs/security.md)
- [命名とWorkspace](docs/naming.md)
- [Community](docs/community.md)

新しい上位・横断的ADRは`plan/adr/`へ置きます。既存の`docs/adr/0001`〜`0015`は参照互換を保ちながら、番号を変えず段階的に移行します。新しい`plan/adr/`の番号は0016以降から継続します。

## 現在の実装範囲

このRepositoryでは、Semantic / WorkspaceからApplication・Database・Query、Release / Deployment、Plugin / Resource authority、Host takeover / runtime reattachまでを、一本の縦切りとしてコードとテストで検証しています。

M0.23以降はSource-to-Canonical Traceability、actual process Plugin、actual PostgreSQL Desired / Actual / Diff、Foreign Key、Unique Constraint、Advanced Index、比較未確定、ProviderExpression representation、オフラインのprovider policy、プロバイダー固有識別子の比較、Advanced IndexのマイグレーションSQL生成まで、個々の境界を段階的に深めています。

M0.35以後はApplication側へ重点を移し、5種類のApplication Patternを`order_entry`で一続きに追跡できるところまで進みました。現在は、その意味をDriverへ渡すためのGate A残条件を実証しています。

個々の実装項目をREADMEへ重複して列挙せず、最新の一覧は [現在の実装状況](docs/status.md) に集約します。

## まだ提供していないもの

- Gate Aを通過したApplication vertical slice
- 既存認証仕様を保持したAuthentication / Authorization Reference
- 業務Validationの実Reference
- `sales.order.submit`の明示的なCapability Binding Reference
- 完成したGUI Editor / Wails application
- Application設計を通しで扱う完成Driver
- production-readyなWeb / Desktop / Mobile Target
- 本番運用向けの永続Store実装
- Kubernetes / cloud / Vault / etcd等のproduction adapter一式
- 完全なSQL dialect対応
- 本番用Workflow Runtime
- 署名済み配布物の完全なsupply-chain attestation
- 万能なOS sandbox
- production supportや互換性保証

この区別は意図的です。現段階では、完成製品のように見せることより、後から大きく壊れやすい意味境界を実行可能なコードで確かめることを優先しています。

M0.22.1を最初の公開設計チェックポイントとして扱った理由は、[ADR-0002](docs/adr/0002-m0221-public-checkpoint.md) に記録しています。M0.22.1は現在地ではなく、公開設計検証を始めた基準点です。

## 名前について

主名称は **TEI（鼎／テイ）**、CLIは`tei`です。TEIを一つの英語展開へ固定せず、設計の異なる面を説明する読みとして次を併存させています。

- Traceable Execution of Intent
- Target-neutral Executable Infrastructure
- Three-Element Integration

詳しくは [docs/naming.md](docs/naming.md) を参照してください。

## License / participation

MIT Licenseです。

積極的なコード貢献だけを参加とは考えていません。実際に試した結果、設計上の違和感、既存業務システムとの適合性、他ツールとの比較、運用上の知見などを公開してもらうことも重要な貢献です。

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/community.md](docs/community.md).
