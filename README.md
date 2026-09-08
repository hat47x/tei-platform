# TEI（鼎／テイ）

- **想定仕様（HTML）:** [仕様ガイド](docs/specification/index.html) — 採用済み・評価待ち・X-TEA原典の対応
- **期限なしロードマップ（HTML）:** [プロダクト開発ロードマップ](plan/product-roadmap.html) — 依存関係と完了条件

TEIは、**業務上の意味を長く保ちながら、実装・Database・Target・Runtimeを必要以上に一体化しない業務Application基盤**を目指す実験的なプロジェクトです。

画面やtableを素早く作ることだけでなく、次を技術変更後も追跡・検証できる状態を重視します。

- この業務概念は何を意味するか
- どのauthoring sourceから来たか
- どの実装 / Provider / TargetへBindingされたか
- Commandは何を返したか
- 外部世界で実際に何が観測されたか
- 何が未検証 / unresolvedか

> **現在地 — 2026-09-05:** architecture & product validation / research implementation  
> `order_entry`のApplication Pattern Stage 1とGate Aは完了しています。Application Driverの主要意味境界も成立しています。現在の正式判定は **Gate B: Not Ready** で、#310 deterministic Repository、#255 generated artifact reproducibility、#308 real PostgreSQL Providerのactual Evidenceを同じclean candidate revisionへ収束する段階です。第一Web Target、Studio / AI、第二TargetはまだProduct Evidence取得前です。

完成したlow-code製品、production-ready Editor、production-ready multi-Target Runtimeを提供する段階ではありません。

## どこから読むか

### Product / planning

- **最上位の価値:** [TEI Product Value](product/value/TEI_PRODUCT_VALUE.md)
- **成熟した姿:** [TEI Product Vision](product/vision/TEI_PRODUCT_VISION.md)
- **判断原則:** [TEI Product Principles](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- **Product Requirements:** [TEI Product Requirements](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- **Requirement → PAS → Evidence:** [Upstream Acceptance Scenario Traceability](plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- **開発全体:** [TEI Master Plan](plan/MASTER_PLAN.md)
- **v1主線:** [v1 Reference Application 実証計画](plan/V1_REFERENCE_APPLICATION_PLAN.md)
- **Requirement Gap:** [Product Requirement Gap Audit](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- **Milestones:** [TEI Milestones](plan/MILESTONES.md)

### 現在のGate

- **Gate A:** [Gate A Readiness Audit](plan/GATE_A_READINESS_AUDIT.md) — Ready
- **Gate B:** [Gate B Readiness Audit](plan/GATE_B_READINESS_AUDIT.md) — Not Ready
- **Gate B Evidence:** [Acquisition](plan/GATE_B_EVIDENCE_ACQUISITION.md) / [Runbook](plan/GATE_B_EVIDENCE_RUNBOOK.md)
- **Gate B → C:** [Handover Contract](plan/GATE_B_TO_C_HANDOVER_CONTRACT.md)
- **Gate C:** [Gate C Evidence Contract](plan/GATE_C_EVIDENCE_CONTRACT.md)

### 技術を読む / 動かす

- [設計文書の案内](docs/README.md)
- [はじめてTEIを知る方へ](docs/for-readers.md)
- [TEIの考え方](docs/overview.md)
- [Getting Started](docs/getting-started.md)
- [Architecture](docs/architecture.md)
- [現在の実装状況](docs/status.md)

## 現在のv1主線

```text
Application Pattern Stage 1
        Done
          ↓
Gate A
        Ready
          ↓
Application Driver
  major semantics ready
          ↓
Gate B
  Not Ready / actual Evidence待ち
  #310 + #255 + #308 same-R
          ↓
Handover Baseline B / revision R
          ↓
First Web Target / build T1
          ↓
Gate C / PAS-04
          ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

Gate C後のStudio / knowledge-first / AI Proposalと第二Targetは並行可能です。

## Gate Bの現在地

Gate Bで新しいDriver抽象を増やすことが現在の主課題ではありません。

Ready判定へ必要なsource-dependent Evidenceは次の三つです。

| Evidence | Repository / GitHub Issue | 現在地 |
| --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | runner Ready、actual clean checkout run待ち |
| generated artifact reproducibility | `ISSUE-000025` / #255 | check / repair境界 Ready、actual check-only待ち |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | runner / safety境界 Ready、専用空DB actual run待ち |

三つを同じclean candidate repository revision `R`へ揃え、`ISSUE-000037` / #387のfinal convergenceで確認します。

actual Oracle #214 / #371 / #376は重要なactual external asset / legacy compatibility / second-provider Evidenceとして継続しますが、Gate Bの第四Required laneではありません。未完了ならHandoverへ`unresolved`として渡します。

## Gate B → Gate Cでrevisionを分ける

Gate B Ready revisionを`R`、Handover Baselineを`B`、第一Web Targetの実装revisionを`T1`とします。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
        ↓ meaning baseline
First Web Target build T1
        ↓
Gate C Evidence
```

通常`T1 != R`です。

Target実装中にbaseline Application meaningを変更する必要が生じた場合、Web固有field / ruleで不足を埋めません。authoritative sourceへ戻り、必要なDriver / Gate B Evidenceを再確認してsuccessor Handoverを発行します。

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

## 現時点で完成済みとは言わないもの

- Gate B actual Evidence / Ready判定
- First Web Target / Gate C
- production-ready Web / Desktop / Mobile Target
- Studio minimal Authoring Loopのactual Reference
- production-ready GUI Editor
- knowledge-first Semantic explorationのactual Reference
- AI Proposal / Decision UXのactual Reference
- second Application Target
- complete Target Capability taxonomy
- complete Workflow Runtime
- production durability / HA guarantee
- generic OS sandbox
- API compatibility guarantee

設計契約やisolated testが存在することを、actual Product Evidence取得済みと読み替えません。

