# TEI（鼎／テイ）

TEI is an experimental low-code platform architecture for building and operating business applications with **semantic intent, deterministic realization, and traceable execution**.

日本語では、**業務の意味を先に定義し、実装・データベース・配備先を必要以上に一体化しないローコード基盤**を目指しています。生成AIを大量のコードを吐く装置としてだけ使うのではなく、既存の意味部品・契約・制約・実現方式を理解して再利用できる土台を作ることが主眼です。

> **Current status:** architecture probe / research implementation.  
> CLI、モデル、検証、参照Workspace、Deployment/Pluginの安全境界は実行可能ですが、完成したEditorや本番用Runtimeを提供する段階ではありません。

## TEIが解こうとしていること

業務システムは、画面やテーブルだけでは表せない多くの意味を持っています。

- 「受注」「顧客」「請求」のような業務概念
- 有効期間、履歴、承認、状態遷移などの共通パターン
- Applicationが複数Databaseを利用する現実
- Database SchemaとApplication Releaseの独立したライフサイクル
- SQL、Go、外部ツール、クラウド等の実装方式
- 本番で何が実際に起きたかというObservation / Evidence

TEIでは、これらを一つの巨大な独自DSLへ押し込めず、**意味・実現・実行・観測を分けたまま追跡可能にする**ことを重視しています。

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

AIが毎回ゼロからコードを生成するのではなく、まず既存のDomain Component、Semantic Pattern、Policy、Capabilityを探索し、それでも足りない部分だけを実装へ落とす方向を想定しています。

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

Canaryのhealth gate、drain、Database migration、rollback/forward-fixも同じ考え方で扱います。

### 5. Pluginは意味モデルではなく配布単位

Plugin / BundleはExecutor、Database Provider、Migration Provider、Editor Intelligence等を配布するための仕組みです。Business Canonical IDとは分離されています。

### 6. 実行時権限を最小化する

process Pluginには、必要なCapabilityへ必要なPermissionだけを明示Grantします。Secret値はCanonical DefinitionやDurable Runへ保存せず、Host Resource Brokerがopaque handleとして解決します。

現在はさらに、Lease renewal、refresh、revoke、Supervisor ownership、fencing、Host takeover、runtime reattachの境界まで実験実装しています。

## まず動かす

正規ソースツリー展開後に必要なのはGo toolchainです。

```bash
git clone https://github.com/hat47x/tei-platform.git
cd tei-platform

go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Validationでは、参照用のhistorical contractについてWarningが出ますが、最後に`ok`となればReference Workspaceは整合しています。

Plugin解決を見る例:

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace > /tmp/tei-plugin-lock.json

go run ./cmd/tei plugin-runtime-plan \
  -workspace reference/workspace \
  -environment production \
  -plugin-lock /tmp/tei-plugin-lock.json
```

Host takeover後のruntime reattachの最小例:

```bash
go run ./examples/runtime-reattach
```

詳しくは [Getting Started](docs/getting-started.md) を参照してください。

## Repository layout

```text
cmd/tei/                 CLI
internal/                executable architecture models / validation
reference/workspace/     small end-to-end TEI workspace
examples/                focused executable examples
docs/                    public and technical documentation
```

主要なドキュメント:

- [はじめてTEIを知る方へ](docs/for-readers.md)
- [TEIの考え方](docs/overview.md)
- [Getting Started](docs/getting-started.md)
- [アーキテクチャ](docs/architecture.md)
- [現在の実装範囲](docs/status.md)
- [Plugin / Bundle](docs/plugins.md)
- [Security / Trust Boundary](docs/security.md)
- [命名とWorkspace](docs/naming.md)
- [Community posture](docs/community.md)

## 現在できること

このRepositoryでは、少なくとも次をコードとテストで検証しています。

- Semantic / Application / Database / Environmentの基本IR
- Applicationから複数DatabaseへのRequirement / Binding
- Database Contract / Persistence Mapping / Desired Schema
- Schema Diff / versioned DDL+DML Migration
- Database Release / Application Release / Solution Release
- SQL analysis boundaryとDerived Query IR
- PostgreSQL schema / EXPLAIN normalization
- Database Workbench向けderived view model
- Deployment Release Graph
- Canary / traffic shift / drain / retire
- Durable Deployment Run / Evidence / Retry / Recovery
- Executor / Observation / Evidence provider routing
- Plugin / Bundle / Plugin Lock / capability negotiation
- Plugin permission / Resource Broker / Sandbox Profile
- Lease renew / refresh / revoke / Host-side supervisor
- Supervisor ownership / heartbeat / fencing
- Durable runtime invocation handle / takeover後のreattach判定

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
