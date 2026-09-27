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

> **現在地 — 2026-09-22:** architecture & product validation / research implementation  
> `order_entry`のApplication Pattern Stage 1とGate Aは完了しています。Application Driverの主要意味境界も成立しています。現在の正式判定は **Gate B: Ready**（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）で、Handover Baseline `B / R`も固定済みです。第一Web Targetについてもactual Gate C Evidenceを取得し、**Gate C: Ready**と判定しました（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`。Visual Fidelityは単一のAI評価者による評価で、人による確認ではありません）。  
> Gate C後には、第二Application `repair_intake`のsourceとEnvironment bindingを、利用者の明示的なAcceptance decisionを経て`reference/workspace`へ受理し、通常のDriver経路で実行できることも確認しました（S5〜S8、[`ISSUE-000040`](plan/issue/000040-semantic-component-composition.md)）。これは第二Application Targetや実行権限の成立を意味しません。Studio / AI、第二TargetはまだProduct Evidence取得前です。詳細な現状は[現在の実装状況](docs/status.md)を正本とします。

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
- **Gate B:** [Gate B Readiness Audit](plan/GATE_B_READINESS_AUDIT.md) — Ready
- **Gate B Evidence:** [Acquisition](plan/GATE_B_EVIDENCE_ACQUISITION.md) / [Runbook](plan/GATE_B_EVIDENCE_RUNBOOK.md)
- **Gate B → C:** [Handover Contract](plan/GATE_B_TO_C_HANDOVER_CONTRACT.md) / [Baseline B](plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)
- **Gate C:** [Gate C Readiness Audit](plan/GATE_C_READINESS_AUDIT.md) — Ready / [Evidence Contract](plan/GATE_C_EVIDENCE_CONTRACT.md) / [First Web Target Runbook](plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)

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
  Ready / R = e5231839a5dcc62cce1181a2731fcb7c70d16736
          ↓
Handover Baseline B / revision R
  fixed
          ↓
First Web Target / build T
  implemented with Evidence / T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
          ↓
Gate C / PAS-04
  Ready
          ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

Gate Cを通過したので、Studio / knowledge-first・AI Proposal / 第二Application・第二Targetは並行して進められます。いま必要なのは、どのlaneから始めるかをEvidenceに基づいて判断することです。各laneの進み具合は[現在の実装状況](docs/status.md)を参照してください。

## Gate Bの現在地

Gate Bは **Ready** です。candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`について、#310 deterministic Repository、#255 generated artifact reproducibility、#308 real PostgreSQL Provider + ephemeral fixtureをsame-runで揃え、#387 final convergenceと独立artifact auditまで成立しています。

Handover Baseline `B / R`も固定済みです。判定の正本は[Gate B Readiness Audit](plan/GATE_B_READINESS_AUDIT.md)、第一Web Targetへの引き渡しは[Baseline B](plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)を参照してください。

この`B / R`を引き継いだ`ISSUE-000007`の第一Web Targetは、baseline `R`とTarget build revision `T`を分離したままGate CのSemantic / Functional / Visual Evidenceを取得し、Gate C Readyと判定済みです（[Gate C Readiness Audit](plan/GATE_C_READINESS_AUDIT.md)）。

actual Oracle #214 / #371 / #376はactual external asset / legacy compatibility / second-provider Evidenceとして継続しますが、Gate B Readyを差し戻す第四Required laneではありません。

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

通常`T1 != R`です。Gate C Readyと判定したbuild `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`が、この`T1`にあたります。

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

- Visual Fidelityの人による承認（Gate C Visualは単一のAI評価者による評価）
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

