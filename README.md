# TEI（鼎／テイ）

> **開発状況・レビューに関する重要な注意**  
> TEIは生成AIを用いて開発中の実験的なプロジェクトです。現段階では、コード・設計・セキュリティに対する体系的な**人的レビューは未実施**です。  
> READMEや各種文書に記載するテスト結果・Gate判定は、対象・時点・検証範囲を限定したものであり、人的レビューや品質・安全性の保証を意味しません。現時点で本番利用を前提としないでください。

TEIは、業務上の意味を長く保ちながら、実装・データベース・実行先・実行環境を必要以上に一体化しない業務アプリケーション基盤を目指す実験的なプロジェクトです。

![TEI — Traceable Execution of Intent](docs/assets/tei-branding-image.jpg)

業務の意味を一つの正本として保ち、SQL・Go・データベース・画面・AIの間を追跡可能につなぐことで、担当者や技術が替わっても理解し、確かめ、変え続けられる業務システムを目指します。現在の価値と将来像は[TEI Product Value](product/value/TEI_PRODUCT_VALUE.md)と[TEI Product Vision](product/vision/TEI_PRODUCT_VISION.md)に記しています。

画面や表を素早く作ることだけでなく、技術を変更しても次の点を追跡・検証できる状態を重視します。

- この業務概念は何を意味するか
- どの設計元情報から来たか
- どの実装・提供元・実行先へ対応付けられたか
- 実行した操作が何を返したか
- 外部世界で実際に何が観測されたか
- 何が未検証・未解決か

> **現在の開発段階**  
> CLIとReference Workspaceを通じて、業務定義、実装、実行結果、観測のつながりを検証しています。Gate A・B・Cの過去のReady判定は、対象を固定した実証結果であり、最新のmainや製品全体の品質保証ではありません。実装済みの範囲と未完成の機構は[現在の実装状況](docs/status.md)を参照してください。

完成したlow-code製品、production-ready Editor、production-ready multi-Target Runtimeを提供する段階ではありません。

## どこから読むか

### 製品価値と開発計画

- **最上位の価値:** [TEI Product Value](product/value/TEI_PRODUCT_VALUE.md)
- **成熟した姿:** [TEI Product Vision](product/vision/TEI_PRODUCT_VISION.md)
- **判断原則:** [TEI Product Principles](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- **Product Requirements:** [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- **Requirement → PAS → Evidence:** [Upstream Acceptance Scenario Traceability](plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- **開発全体:** [TEI Master Plan](plan/MASTER_PLAN.md)
- **v1主線:** [v1 Reference Application 実証計画](plan/V1_REFERENCE_APPLICATION_PLAN.md)
- **Requirement Gap:** [Product Requirement Gap Audit](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- **Milestones:** [TEI Milestones](plan/MILESTONES.md)

### 現在の判定段階

- **Gate A:** [Gate A Readiness Audit](plan/GATE_A_READINESS_AUDIT.md) — Ready
- **Gate B:** [Gate B Readiness Audit](plan/GATE_B_READINESS_AUDIT.md) — Ready
- **Gate B Evidence:** [Acquisition](plan/GATE_B_EVIDENCE_ACQUISITION.md) / [Runbook](plan/GATE_B_EVIDENCE_RUNBOOK.md)
- **Gate B → C:** [Handover Contract](plan/GATE_B_TO_C_HANDOVER_CONTRACT.md) / [Baseline B](plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)
- **Gate C:** [Gate C Readiness Audit](plan/GATE_C_READINESS_AUDIT.md) — Ready / [Evidence Contract](plan/GATE_C_EVIDENCE_CONTRACT.md) / [First Web Target Runbook](plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)

### 技術を読む / 動かす

- [設計文書の案内](docs/README.md)
- [はじめてTEIを知る方へ](docs/for-readers.md)
- [TEIの考え方](docs/overview.md)
- [使い始める](docs/getting-started.md)
- [全体構成](docs/architecture.md)
- [現在の実装状況](docs/status.md)

## 開発状況と検証範囲

Go CLI、Driver、Reference Workspace、および第一Web Targetの一部について、固定Source・環境での実証を積み重ねています。一方、統合モデラー、汎用Target Runtime、多様な外部Provider、本番環境での安全性は未完成または未検証です。

過去の実証結果と現在の実装は、次の資料で区別しています。

- [現在の実装状況](docs/status.md)：実装範囲と未完成事項の概要
- [Gate A](plan/GATE_A_READINESS_AUDIT.md)・[Gate B](plan/GATE_B_READINESS_AUDIT.md)・[Gate C](plan/GATE_C_READINESS_AUDIT.md)：それぞれの固定revisionとEvidenceに対する判定
- [TEI Master Plan](plan/MASTER_PLAN.md)・[Milestones](plan/MILESTONES.md)：今後の開発方針と優先順位

## 設計上の主要原則

### Meaningと実装を分ける

```text
Authoring Source
      ↓
Canonical Meaning
      ↓
Binding / Realization
      ↓
Provider / Target / Runtime
      ↓
Execution
      ↓
Observation / Evidence
```

「何で実装するか」を「何を意味するか」へ暗黙に混ぜません。

### file pathをSemantic identityにしない

Source-to-Canonical Traceabilityを持ちつつ、file pathはsemantic identityではありません。

file moveだけでlogical identityが変わらない場合はsemantic no-opになり得ます。logical rename / Canonical identity変更はSemantic changeとして扱います。

### SQLやGoを敵にしない

`.tei`へ全面移植する方針ではありません。

```text
*.tei   semantic / application / lifecycle definitions
*.sql   native SQL
*.go    custom implementation
*.json  observed / derived artifacts
*.yaml  workspace / external configuration
*.md    documentation
```

native assetはauthoritativeなままSource-to-Canonical traceへ接続します。

### CommandとObservationを分ける

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation / Execution Error
```

Command acknowledgementだけでexternal state成立とはみなしません。

Business Result、Execution Error、Observation、Developer Diagnosticを分離します。

### Studio / AIを第二正本にしない

Gate C後のauthoringは次へ収束させます。

```text
file-native / Studio / CLI / AI
          ↓ candidate
Shared Acceptance Boundary
          ↓
authoritative source
          ↓ reload
Canonical meaning
```

Preview / Rejectはnon-mutating、stale `baseModelRevision`は明示処理します。

Studio / AI都合でmandatory authored Stable ID / UUIDやPatch-only authoringを前提にしません。

### Studio Wails HostとApplication Targetを分ける

Studioがdeveloper-facing desktop hostとしてWailsを使っても、Generated Wails Desktop Application Targetとは別物です。

Studio GUIの成功を第二Application TargetのSemantic / Functional / Visual Evidenceへ数えません。

## まず動かす

clone後、基本的なRepository確認はGo toolchainだけで行えます。

```bash
git clone https://github.com/hat47x/tei-platform.git
cd tei-platform

go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Gate B Product Evidenceの正規取得手順は通常の開発確認とは別です。actual Gate B Evidenceを取得するときは必ず [Gate B Evidence Runbook](plan/GATE_B_EVIDENCE_RUNBOOK.md) を参照してください。

Source Index例:

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

## Repository layout

```text
product/                 Product Value / Vision / Principles
requirements/            Product Requirements
plan/                    Master / Milestone / PAS / Gate / ADR / Plan Issue
cmd/tei/                 CLI
internal/                executable architecture / validation / runtime
reference/workspace/     end-to-end Reference Workspace
reference/observed/      observed fixtures
tools/                   Evidence / Reference support tools
examples/                focused executable examples
docs/                    reader / architecture / conformance documentation
```

## 制約と未完成の機能

TEIは実験的な開発基盤であり、現時点では本番利用やAPI互換性を保証しません。特に、Studio / GUI Editor、Web以外も含む汎用Target、外部Provider、AIを用いた設計支援、運用時の耐障害性・セキュリティは、設計内容と実証済みの範囲を区別する必要があります。

詳しい現在地と根拠は[現在の実装状況](docs/status.md)、開発計画は[TEI Master Plan](plan/MASTER_PLAN.md)を参照してください。設計契約や局所テストの存在を、製品全体の検証済み状態と同一視しません。
