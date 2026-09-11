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
- `docs/product-quality-architecture.md`
- `plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`

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
Gate B — 3 Required Evidence / Not Ready
        ↓
Handover Baseline B at revision R
        ↓
First Web Target build T1
        ↓
Gate C — PAS-04 Evidence
        ↓
        ├──────── Authoring / Knowledge lane
        │           ├─ Studio minimal loop — PAS-01
        │           ├─ knowledge-first exploration — PAS-06
        │           └─ AI Proposal / Acceptance — PAS-01 / PAS-06
        │
        └──────── Second Target pressure-test lane — PAS-04 / RQ-024
```

Gate C後のAuthoring / Knowledgeと第二Targetは、互いの完成待ちにはしません。第一Targetから得た差とEvidenceを共有しながら並行できます。

Product Qualityはこの主線へ別の直列Stageを挿入しません。Meaning / Change / Execution / Interaction / Performance / Evidence qualityを各Stageへ横断適用します。

## 現在のGate / Handover

### Gate A — Application上の意味をDriverへ渡せる

**Ready**です。

Application Pattern、Datasource、Validation、Authentication / Authorization、Capability Binding、Business Result / Execution Error等を、UI固有定義なしでDriverへ接続できる意味基線が成立しています。

### Gate B — UIなしで主要Scenarioを再現できる

**Not Ready**です。

中心設計の不足ではなく、actual execution Evidenceが未取得だからです。

Required Evidenceは次の三つだけです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃え、`ISSUE-000037` / #387のfinal convergenceで確認します。

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、Gate Bの第四Required laneにはしません。未完了ならHandoverへ`unresolved`として渡します。

RQ-027〜RQ-030のProduct Quality Requirementを理由に、Gate Bへ第四Required laneや新しいquality Gateを追加しません。既存Gate B Evidenceが示すdiagnostic / change-safety境界はQuality Evidenceとして再利用しますが、Gate B成功を待たずStrong Evidenceへ格上げしません。

### Handover Baseline — Gate B → Gate C

Handoverは新しい番号付きGateではありません。

Gate B Readyのcandidate revision `R`を、第一Web Targetの意味baselineとして固定します。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

Handover recordはCanonicalの第二正本ではありません。

### Gate C — 第一Targetとの差をEvidenceで説明できる

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

First Targetでは、既存3区分を壊さず、Product Quality Evidence Planに従って次も条件付きで観測します。

- Interaction / Operability
- applicable accessibility checks
- failure / degraded / recovery behavior
- performance / resource context
- Quality Envelope provenance

これらを一つの総合scoreへ畳みません。

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

quality claimについても、単独の数値や形容詞ではなく、subject / context / measure / acceptance / Evidenceへ戻れることを要求します。

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

Derived cognition researchも、candidate generation qualityとSemantic applicability / authorityを分離します。

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

compatibility / degraded behavior / external failureも、成功Scenarioだけに隠さずQuality Envelopeへ戻せる形で扱います。

## Workstream D: Database / Provider Lifecycle

### 目的

Desired / Actual / Diff / Migration / Provider境界を、Provider固有知識をCoreへ漏らさず成立させます。

PostgreSQLは深い実証Providerとして利用します。

Oracle等の異なるProviderは、既存資産互換とprovider-neutral境界の圧力試験に利用します。

actual Oracle Evidence未完了をGate B blockerへ戻しません。

change safetyでは、planning時の観測をdestructive apply時へ無条件carry-forwardせず、必要なfresh observation / preconditionをProvider境界で実証します。PostgreSQL実装をそのままgeneric Core protocolへ昇格しません。

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
- failure / degraded / unknownの分類とdiagnostic trace

Driverは本番Runtimeの簡易版ではなく、設計検証の基準線です。

## Workstream F: Target-neutral Presentation

### 目的

同じCanonical Applicationを複数TargetへProjectionし、共通意味とTarget差をEvidenceから分離します。

### 第一Target

第一Web TargetはPAS-04の最初の実証です。

Gate C EvidenceではHandover baseline `R`とWeb build `T1`を分けて追跡します。

Semantic / Functional / Visualに加え、Target条件付きのoperability / accessibility / performance / failure behaviorをQuality Envelopeへ接続します。

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
- Semantic / Functional / Visual / Interaction Evidence
- RQ-024 performance measurement
- RQ-027 / 029 / 030 Quality Envelope差

第一TargetだけでTarget-neutral taxonomyやquality thresholdを完成扱いしません。

### Studio HostとApplication Targetを分ける

Studio / Editorがdeveloper-facing desktop hostとしてWailsを利用しても、それを第二Application Target Evidenceへ数えません。

Generated Wails Desktop Application Targetを採用する場合は、別のTargetとしてSemantic / Functional / Visual / Interaction Evidenceを取得します。

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

Studio qualityでは、Preview / Reject non-mutating、stale conflict、failure recovery、high-densityとkeyboard / navigation / state visibilityの両立を実証します。

## Workstream H: AI-assisted Engineering

### 目的

AIを大量生成器ではなく、既存知識探索、変更候補、検査、Decision支援へ接続します。

### 二段階

1. `ISSUE-000047` — Human / AI共通のknowledge-first Semantic exploration（PAS-06）
2. `ISSUE-000027` — AI Proposal / Shared Acceptance Boundary（PAS-01 / PAS-06）

AI ProposalはCanonical authorityではありません。

search ranking、similarity、model confidenceもSemantic Evidenceではありません。

autonomyとauthorityを分け、raw AI tool callをCapability / Permission modelの代替にはしません。

AI / derived cognitionのqualityはcandidate recall、irrelevant candidate、unknown、latency、resource / external cost等として測定できますが、それをSemantic correctnessやaccepted Decisionへ自動変換しません。

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

Operational Telemetryは必要に応じてOpenTelemetry等の外部semantic conventionsへProjectionできますが、telemetry data modelをCanonical business meaningへしません。

## Cross-cutting: Product Quality Envelope

Product QualityはWorkstream Jではありません。A〜Iを横断して、各claimを再確認可能にする設計軸です。

### Quality axes

```text
Meaning quality
  -> silent meaning driftを防ぐ

Change quality
  -> impact / compatibility / destructive precondition / recovery

Execution quality
  -> failure / degraded / unknown / diagnostic

Interaction quality
  -> operability / accessibility / recovery

Performance / resource quality
  -> Target / Host / Provider条件付きmeasurement

Evidence quality
  -> claimをContext / Measure / Evidenceへ戻す
```

### Quality Envelope

概念上、quality claimを次へ結びます。

```text
subject
+ revision / build identity
+ Target / Host / Provider / Runtime context
+ Scenario / precondition
+ quality property
+ measure / probe
+ acceptance rule / rationale
+ Evidence artifact
+ status
```

現時点で固定schema、新しいCanonical kind、universal quality scoreを導入しません。

### 外部quality model

- ISO/IEC 25010:2023 — quality characteristicの抜け漏れ監査
- WCAG 2.2 — Web Target accessibilityのtestable reference
- OpenTelemetry semantic conventions — operational telemetry Projection候補
- NIST SSDF 1.1 — secure development / supply-chain process checklist

これらをTEI Requirementの第二正本や準拠claimへ自動昇格しません。

詳細は`docs/product-quality-architecture.md`、実証順は`plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`を正とします。

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

Cross-cutting Product Quality
  ── A〜Iすべてのclaim / change / execution / interaction / Evidenceへ適用
```

完全な直列ではありません。Gate C後はStudio / AI / knowledge-firstと第二Targetを並行できます。

## 現在の優先順

1. Gate B actual final Evidenceを同一revisionへ収束する
2. Gate B ReadyならHandover Baseline `B / R`を固定する
3. 第一Web Targetをbuild revision `T1`として実装し、Gate C Evidenceを取得する
   - Semantic / Functional / Visualを維持
   - operability / accessibility / performance / failure qualityをcontext付きで観測する
4. Gate C Ready後、次を並行開始する
   - `ISSUE-000046` Studio minimal Authoring Loop
   - `ISSUE-000047` knowledge-first Semantic exploration
   - `ISSUE-000027` AI Proposal / Acceptance
   - `ISSUE-000026` 第二Target pressure test
5. 第二TargetとAuthoring / AIのEvidenceを相互に戻し、Target-neutral / authoring / quality境界を再評価する
6. production / ecosystem境界を必要なEvidenceとともに広げる

現在、actual Gate B final runは実TEI checkout + Evidence専用PostgreSQL環境待ちです。環境待ちを理由にGate C以後の実装を前倒ししませんが、qualityを含むplanning / traceabilityは先に矛盾なく固定します。

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
- 一律availability / reliability threshold
- Quality Envelopeの固定schema
- 全品質を一つへ畳むuniversal score
- OpenTelemetryの必須Runtime dependency
- 汎用Workflow Engine
- production HA一般化

Reference / Evidenceから必要性が確認される前に固定しません。