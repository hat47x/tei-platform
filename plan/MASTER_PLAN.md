# TEI Master Plan

## 目的

この文書は、TEI（鼎／テイ）のProduct Value / Requirementsを開発workstreamへ分解し、現在の主線と並行可能な作業を示すための上位計画です。

期限は置きません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**で見直します。

履歴の蓄積ではなく、現在のplanning baselineを保持します。過去の判断経緯はGit history、ADR、Repository Issueから追跡します。

関連:

- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`

## 現在の主線

v1 Reference Application `order_entry`を中心に、現在は次の順で進めます。

```text
Product Value / Requirements
        ↓
Application Pattern
        ↓
Gate A — Ready
        ↓
Application Driver
        ↓
Gate B — 3 Required Evidence / Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B at revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 Evidence — Ready
        ↓
        ├──────── Authoring / Knowledge lane
        │           ├─ Studio minimal loop — PAS-01
        │           ├─ knowledge-first exploration — PAS-06
        │           └─ AI Proposal / Acceptance — PAS-01 / PAS-06
        │
        └──────── Second Target pressure-test lane — PAS-04 / RQ-024
```

Gate C後のAuthoring / Knowledgeと第二Targetは、互いの完成待ちにはしません。第一Targetから得た差とEvidenceを共有しながら並行できます。

## 現在のGate / Handover

### Gate A — Application上の意味をDriverへ渡せる

**Ready**です。

Application Pattern、Datasource、Validation、Authentication / Authorization、Capability Binding、Business Result / Execution Error等を、UI固有定義なしでDriverへ接続できる意味基線が成立しています。

### Gate B — UIなしで主要Scenarioを再現できる

**Ready**です（2026-09-19再監査、candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

Required Evidenceは次の三つで、同じclean candidate revision `R`のsame-runとして成立しています。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

`#387` final convergenceとsame-run child artifact auditを含むReady判定の正本は `GATE_B_READINESS_AUDIT.md` です。

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、Gate Bの第四Required laneにはしません。未完了範囲はHandoverへ`unresolved`として渡します。

### Handover Baseline — Gate B → Gate C

Handoverは新しい番号付きGateではありません。

Gate B Readyのcandidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`は、Handover Baseline `B`として固定済みです。個別記録の正本は `GATE_B_TO_C_HANDOVER_BASELINE_B.md` です。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

Handover recordはCanonicalの第二正本ではありません。

### Gate C — 第一Targetとの差をEvidenceで説明できる

**Not Ready**です。Gate B / Handoverの着手条件は成立していますが、第一Web Target build `T1`とGate C Evidenceはまだ成立していません。

第一Web TargetはHandover Baseline `B` / revision `R`を意味baselineとして利用します。

Web実装revisionは`T1`として別に追跡します。

```text
Baseline meaning B at R
        ↓
First Web Target build T1
        ↓
Semantic / Functional / Visual Evidence
```

`T1`でbaseline subjectの意味を変更した場合、黙って継続せず新しいHandover Baselineを発行します。

Gate Cでは次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

Visual evaluator / VLM / screenshot scoreだけでSemantic / Functionalを合格にしません。

## Workstream A: Product Baseline / Traceability

### 目的

Product Value、Requirements、PAS、ADR、Plan Issue、Reference、Evidence、Gateを一続きに追跡できる状態を維持します。

### 重点

```text
Product Requirement
        ↓
Product Acceptance Scenario
        ↓
ADR / Plan Issue
        ↓
Reference / Fixture
        ↓
Evidence
        ↓
Gate / Product claim
```

PASはcoverage scoreではなく、Requirementと実証をつなぐtrace単位として扱います。

## Workstream B: Semantic / Authoring / Knowledge

### 目的

業務上の意味をauthoring sourceから再構成し、複数Projectionから同じCanonical meaningを扱える基盤を育てます。

### 対象

- Semantic Kernel
- Source Index / Diagnostic
- Source-to-Canonical Traceability
- Domain Component / Semantic Pattern / Policy候補
- provenance / revision
- reusable semantic knowledge exploration

Application Patternはv1ではApplication所有です。第一ReferenceだけからSemantic Kernelの標準Catalogへ持ち上げません。

AI向けKnowledge、検索index、embedding、trajectory等はderivedとして扱い、Canonicalの第二正本にはしません。

Gate C後のread-only探索は`ISSUE-000047`でPAS-06として実証します。

## Workstream C: Application / Legacy Compatibility

### 目的

新規理想形だけでなく、既存業務資産を保持しながら段階移行できることを確認します。

### 対象

- Application Pattern / intent
- Application ↔ Database
- Database Contract / native Query
- Datasource read / write
- existing authentication / authorization
- existing SQL / external API
- Capability Mapping
- incremental replacement

actual external assetは、観測事実とCanonical contractを分離します。

未観測のphysical object / column / SQL / datatypeを推測しません。

## Workstream D: Database / Provider Lifecycle

### 目的

Desired / Actual / Diff / Migration / Provider境界を、Provider固有知識をCoreへ漏らさず成立させます。

PostgreSQLは深い実証Providerとして利用します。

Oracle等の異なるProviderは、既存資産互換とprovider-neutral境界の圧力試験に利用します。

actual Oracle Evidence未完了をGate B blockerへ戻しません。

## Workstream E: Driver / Execution

### 目的

設計内容をUI実装前に動かし、Application meaningを検証できるDriverを維持します。

### 重点

- browse / record / reference / header-detail / command
- Validation
- Authentication / Authorization
- Capability invocation
- Business Result / Execution Error
- separate Observation read
- deterministic Scenario

Driverは本番Runtimeの簡易版ではなく、設計検証の基準線です。

## Workstream F: Target-neutral Presentation

### 目的

同じCanonical Applicationを複数TargetへProjectionし、共通意味とTarget差をEvidenceから分離します。

### 第一Target

第一Web TargetはPAS-04の最初の実証です。

Gate C EvidenceではHandover baseline `R`とWeb build `T1`を分けて追跡します。

### 第二Target

Gate C後、`ISSUE-000026`で第二Targetを選びます。

比較軸:

- Canonical Application meaning
- Target Capability
- Runtime Binding
- Execution / Trust Profile
- Target固有実装
- Host control plane
- Observation
- Semantic / Functional / Visual Evidence
- RQ-024 performance measurement

第一TargetだけでTarget-neutral taxonomyを完成扱いしません。

### Studio HostとApplication Targetを分ける

Studio / Editorがdeveloper-facing desktop hostとしてWailsを利用しても、それを第二Application Target Evidenceへ数えません。

Generated Wails Desktop Application Targetを採用する場合は、別のTargetとしてSemantic / Functional / Visual Evidenceを取得します。

## Workstream G: Studio / Developer Experience

### 目的

厳格な意味モデルを、高密度かつ安全に扱えるdeveloper-facing Studioへ接続します。

### 原則

StudioはCanonicalを所有しません。

```text
Studio / file-native / CLI / AI
        ↓ candidate
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical meaning
```

Gate C後の最初の実証は`ISSUE-000046`です。

最小loop:

```text
Select
  ↓
Propose
  ↓
Preview Semantic Diff / impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
reload / Conformance
```

mandatory authored Stable IDやStudio専用Acceptance engineを導入しません。

## Workstream H: AI-assisted Engineering

### 目的

AIを大量生成器ではなく、既存知識探索、変更候補、検査、Decision支援へ接続します。

### 二段階

1. `ISSUE-000047` — Human / AI共通のknowledge-first Semantic exploration（PAS-06）
2. `ISSUE-000027` — AI Proposal / Shared Acceptance Boundary（PAS-01 / PAS-06）

AI ProposalはCanonical authorityではありません。

search ranking、similarity、model confidenceもSemantic Evidenceではありません。

autonomyとauthorityを分け、raw AI tool callをCapability / Permission modelの代替にはしません。

## Workstream I: Ecosystem / Production Boundary

### 目的

Coreと外部実行基盤の境界を、実利用に耐える形へ段階的に広げます。

### 対象

- Plugin distribution / trust
- production Provider adapters
- durable store / HA
- observability
- deployment integrations
- supply-chain Evidence
- compatibility policy

外部orchestrator / agent frameworkが提供する一般機能をTEI固有価値として再実装しません。

## Workstream間の依存

```text
A Product Baseline
      ↓
B Semantic / Knowledge ───┬── C Application / Legacy
                          ├── D Database / Provider
                          └── E Driver
                               ↓
                    F First Target / Gate C
                       ┌───────┴────────┐
                       ↓                ↓
                 G Studio          F Second Target
                       ↕                │
                 H AI / Knowledge ──────┘
                       ↓
                 I Production Boundary
```

完全な直列ではありません。Gate C後はStudio / AI / knowledge-firstと第二Targetを並行できます。

## 現在の優先順

1. Gate B actual final Evidenceを同一revisionへ収束する
2. Gate B ReadyならHandover Baseline `B` / revision `R`を固定する
3. 第一Web Targetをbuild revision `T1`として実装し、Gate C Evidenceを取得する
4. Gate C Ready後、次を並行開始する
   - `ISSUE-000046` Studio minimal Authoring Loop
   - `ISSUE-000047` knowledge-first Semantic exploration
   - `ISSUE-000027` AI Proposal / Acceptance
   - `ISSUE-000026` 第二Target pressure test
5. 第二TargetとAuthoring / AIのEvidenceを相互に戻し、Target-neutral / authoring境界を再評価する
6. production / ecosystem境界を必要なEvidenceとともに広げる

現在、actual Gate B final runは実TEI checkout + Evidence専用PostgreSQL環境待ちです。環境待ちを理由にGate C以後の契約を前倒し実装しませんが、planning / traceabilityは先に矛盾なく固定します。

## 先に固定しないもの

- 第一Web Target framework
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- Editor最終UI
- AI provider
- vector database
- Proposal最終schema
- mandatory authored Stable ID / UUID
- Patch-only authoring model
- WailsのTEI標準Runtime化
- 第二Targetの種類
- 一律performance threshold
- 汎用Workflow Engine
- production HA一般化

Reference / Evidenceから必要性が確認される前に固定しません。
