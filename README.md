# TEI（鼎／テイ）

TEIは、**業務上の意味を長く保ちながら、実装・データベース・配備先を必要以上に一体化しないローコード基盤**を目指す実験的なプロジェクトです。

画面やテーブルを素早く作ることだけでなく、「この業務概念は何を意味するのか」「どの実装へ落ちたのか」「本番で実際に何が起きたのか」を、技術が変わっても追跡できる状態を重視しています。

> **現在地:** M0.22.1 / architecture probe / research implementation  
> CLI、モデル、検証、参照Workspace、Deployment / Pluginの安全境界は実行可能ですが、完成したEditorや本番用Runtimeを提供する段階ではありません。

## どこから読むか

目的に応じて入口を分けています。

- **TEIを初めて知る:** [はじめてTEIを知る方へ](docs/for-readers.md)
- **考え方をもう少し知る:** [TEIの考え方](docs/overview.md)
- **実際に動かす:** [Getting Started](docs/getting-started.md)
- **技術構造を追う:** [Architecture](docs/architecture.md)
- **実装済み／未実装を確認する:** [現在の実装状況](docs/status.md)

最初からコードを読む必要はありません。一般向けの説明と、実装を追うための技術文書を分けています。

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

## 設計上の特徴

### 1. ApplicationとDatabaseを一対一にしない

一つのApplicationが複数Databaseを利用することを通常形として扱います。

```text
order_entry
├─ sales  → sales_core   (PostgreSQL, read/write)
└─ master → master_data  (Oracle, read-only)
```

Application Releaseは必要なDatabase Contractの集合を保持し、Database Releaseは独立してversion管理されます。

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

Workspaceは`tei.workspace.yaml`と管理ディレクトリで識別します。

### 3. 生成AIに「意味の部品」を渡せる構造

AIが毎回ゼロからコードを生成するのではなく、既存の業務概念、意味パターン、Policy、Capabilityを先に探索し、それでも足りない部分だけを実装へ落とす方向を想定しています。

AIが常時本番判断を行うことや、自動的に強い実行権限を持つことを前提にはしていません。実行時の規則や権限は、明示的で追跡可能な形へ固定することを重視します。

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

### 5. Pluginは意味モデルではなく配布単位

Plugin / BundleはExecutor、Database Provider、Migration Provider、Editor Intelligence等を配布するための仕組みです。Business Canonical IDとは分離しています。

### 6. 実行時権限を最小化する

process Pluginには、必要なCapabilityへ必要なPermissionだけを明示Grantします。Secret値はCanonical DefinitionやDurable Runへ保存せず、Host Resource Brokerがopaque handleとして解決します。

現在はさらに、Lease renewal、refresh、revoke、Supervisor ownership、fencing、Host takeover、runtime reattachの境界まで実験実装しています。

## まず動かす

ここから先は開発者向けです。clone後に必要なのはGo toolchainだけです。

```bash
git clone https://github.com/hat47x/tei-platform.git
cd tei-platform

go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Validationでは、参照用のhistorical contractについてWarningが出ますが、最後に`ok`となればReference Workspaceは整合しています。

Host takeover後のruntime reattachの最小例は次で実行できます。

```bash
go run ./examples/runtime-reattach
```

Plugin解決やReference artifactの再生成を含む手順は [Getting Started](docs/getting-started.md) を参照してください。

## Repository layout

```text
cmd/tei/                 CLI
internal/                executable architecture models / validation
reference/workspace/     small end-to-end TEI workspace
reference/observed/      deployment observation fixtures
reference/world.json     golden canonical world snapshot
examples/                focused executable examples
docs/                    public and technical documentation
.github/workflows/       repository verification
```

技術文書:

- [Architecture](docs/architecture.md)
- [現在の実装状況](docs/status.md)
- [設計判断記録（ADR）](docs/adr/README.md)
- [Plugin / Bundle](docs/plugins.md)
- [Security / Trust Boundary](docs/security.md)
- [命名とWorkspace](docs/naming.md)
- [Community](docs/community.md)

## 現在の実装範囲

このRepositoryでは、Semantic / WorkspaceからApplication・Database・Query、Release / Deployment、Plugin / Resource authority、Host takeover / runtime reattachまでを、一本の縦切りとしてコードとテストで検証しています。

個々の実装項目をREADMEへ重複して列挙せず、最新の一覧は [現在の実装状況](docs/status.md) に集約しています。

## まだ提供していないもの

- 完成したGUI Editor / Wails application
- 本番運用向けの永続Store実装
- Kubernetes / cloud / Vault / etcd等のproduction adapter一式
- 完全なSQL IR / 全SQL dialect対応
- 本番用Workflow Runtime
- 署名済み配布物の完全なsupply-chain attestation
- 万能なOS sandbox
- production supportや互換性保証

この区別は意図的です。現段階では、後から大きく壊れやすい境界を先に実行可能なコードで検証しています。

M0.22.1を最初の公開設計チェックポイントとして扱う理由は、[ADR-0002](docs/adr/0002-m0221-public-checkpoint.md) に記録しています。

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
