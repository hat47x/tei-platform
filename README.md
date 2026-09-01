# TEI（鼎／テイ）

TEIは、**業務上の意味を長く保ちながら、実装・データベース・配備先を必要以上に一体化しないローコード基盤**を目指す実験的なプロジェクトです。

画面やテーブルを素早く作ることだけでなく、「この業務概念は何を意味するのか」「どの実装へ落ちたのか」「本番で実際に何が起きたのか」を、技術が変わっても追跡できる状態を重視しています。

> **現在地:** M0.32 / architecture & product validation / research implementation  
> Source-to-Canonical Traceability、Application / Database、actual PostgreSQL、Deployment、Plugin / Runtime等の設計境界は実行可能なReferenceで検証していますが、完成したEditorや本番用Runtimeを提供する段階ではありません。

## どこから読むか

目的に応じて入口を分けています。

### TEIが何を目指すか

- **最も上位の価値を知る:** [TEI Product Value](product/value/TEI_PRODUCT_VALUE.md)
- **成熟した姿を知る:** [TEI Product Vision](product/vision/TEI_PRODUCT_VISION.md)
- **設計判断の基準を知る:** [TEI Product Principles](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- **プロダクトとして必要なことを確認する:** [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- **期限を置かない開発の到達点を確認する:** [TEI Milestones](plan/MILESTONES.md)
- **開発全体の進め方を確認する:** [TEI Master Plan](plan/MASTER_PLAN.md)

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

現在は、このAI支援に先立って、AIが探索する価値のあるApplication Patternや設計知識をv1 Referenceで実証する段階へ進んでいます。

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

さらにM0.31以降では、Database Schemaについて「差がある」と「比較に必要な情報が足りず、まだ確定できない」を分離しています。不明なものを都合よく一致や差分へ丸めません。

### 5. Pluginは意味モデルではなく配布単位

Plugin / BundleはExecutor、Database Provider、Migration Provider、Editor Intelligence等を配布するための仕組みです。Business Canonical IDとは分離しています。

Applicationから見たCapability identityと、その実現方法も分離する方向で設計しています。

### 6. 実行時権限を最小化する

process Pluginには、必要なCapabilityへ必要なPermissionだけを明示Grantします。Secret値はCanonical DefinitionやDurable Runへ保存せず、Host Resource Brokerがopaque handleとして解決します。

Lease renewal、refresh、revoke、Supervisor ownership、fencing、Host takeover、runtime reattachまで、外部Runtimeを安全に扱う境界を実験実装しています。

## 現在の重点

M0.32までに、Source / Meaning、Database、Execution / Observation、Plugin / Runtimeの下部構造にはかなり強いEvidenceが蓄積しました。

[Product Requirement Gap Audit](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)では、次の領域が相対的に未実証であることを確認しています。

- 既存認証・認可を保持した移行
- standard Application Pattern
- Driverによる設計シミュレーション
- target-neutral Presentation / Interaction
- 開発者向けEditor
- AIによる知識探索、確認提案、判断履歴

そのため、v1へ向けて既存`order_entry`をReference Applicationとして育て、**Application Pattern → Driver → 第一Web Target**を主線として検証します。既存認証Compatibility Referenceは並行して進めます。

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
plan/                    Milestones / Master Plan / ADR / Issue
cmd/tei/                 CLI
internal/                executable architecture models / validation
reference/workspace/     small end-to-end TEI workspace
reference/observed/      deployment observation fixtures
reference/world.json     golden canonical world snapshot
examples/                focused executable examples
docs/                    public and technical documentation
.github/workflows/       repository verification
```

上位・計画文書:

- [Product文書の入口](product/README.md)
- [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [TEI Planning](plan/README.md)
- [Product Requirement Gap Audit](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)

技術文書:

- [Architecture](docs/architecture.md)
- [現在の実装状況](docs/status.md)
- [既存の設計判断記録（ADR）](docs/adr/README.md)
- [Plugin / Bundle](docs/plugins.md)
- [Security / Trust Boundary](docs/security.md)
- [命名とWorkspace](docs/naming.md)
- [Community](docs/community.md)

新しい上位・横断的ADRは`plan/adr/`へ置きます。既存の`docs/adr/0001`〜`0012`は参照互換を保つため、段階的に移行します。

## 現在の実装範囲

このRepositoryでは、Semantic / WorkspaceからApplication・Database・Query、Release / Deployment、Plugin / Resource authority、Host takeover / runtime reattachまでを、一本の縦切りとしてコードとテストで検証しています。

M0.23以降はSource-to-Canonical Traceability、actual process Plugin、actual PostgreSQL Desired / Actual / Diff、Foreign Key、Unique Constraint、Advanced Index、比較未確定、ProviderExpression representationまで、個々の境界を段階的に深めています。

個々の実装項目をREADMEへ重複して列挙せず、最新の一覧は [現在の実装状況](docs/status.md) に集約します。

## まだ提供していないもの

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
