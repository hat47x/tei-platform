# Product Requirement Gap Audit

- 監査日: 2026-09-11
- 対象: RQ-001〜RQ-030
- 上流Trace: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- Quality設計: `../docs/product-quality-architecture.md`
- Quality実証計画: `PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`

## 1. 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`の各Requirementについて、現在のRepositoryにどのEvidenceがあり、どの不確実性が残っているかを逆マッピングするcurrent auditです。

実装量、Issue close数、文書数ではなく、次を見ます。

- semantic boundaryが成立しているか
- authoritative sourceへ戻れるか
- Reference / Conformanceで再現できるか
- actual execution Evidenceが必要な場合に取得済みか
- synthetic / generated / real-provider / actual-external / Target Evidenceを混同していないか
- Provider / Target / Host / build条件を説明できるか
- failure / degraded / unknownを成功へ読み替えていないか
- quality claimがContext / Measure / Evidenceへ戻れるか
- unresolvedを成功へ丸めていないか

本書は現在断面だけを保持します。旧評価はGit historyから追跡します。

## 2. 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装・Reference・Conformance / Evidenceが一続きで存在し、主要意味境界を説明できる |
| **Partial Evidence** | 重要部分は成立しているが、Requirement全体またはactual / multi-context Evidenceが不足 |
| **Design Only** | Requirement / Architecture / Planは具体化されているが、対応するReference実証はまだない |
| **Not Started** | 対応する設計・Referenceがほぼない |

`Design Only`を実装済み、`Partial Evidence`をProduct claim成立済みとは読み替えません。

## 3. 現在のProduct主線

現在最大の直列blockerは引き続き**Gate B actual Evidence**です。

```text
Gate A — Ready
   ↓
Gate B — Not Ready
  #310 / #255 / #308 same-R actual Evidence待ち
   ↓
Handover Baseline B / revision R
   ↓
First Web Target build T1
   ↓
Gate C — PAS-04
   ↓
   ├─ Studio minimal loop — PAS-01
   ├─ knowledge-first / derived cognition — PAS-06
   ├─ AI Proposal / Acceptance — PAS-01 / PAS-06
   └─ Second Target — PAS-04 / RQ-024
```

Cross-cutting Product Qualityは、この主線へ新しいGateを挿入しません。

```text
Requirements / Quality Architecture
      ├─ Gate B既存Evidenceを再利用
      ├─ First Targetでinteraction / accessibility / performanceを具体化
      ├─ Studio / AI / Second TargetをQuality Envelopeで比較
      └─ productionでoperational / compatibility / supply-chainへ拡張
```

actual Oracle #214 / #371 / #376は重要なPAS-03 / external asset Evidenceとして継続しますが、Gate Bの第四Required laneではありません。

## 4. Requirement別監査

| RQ | 評価 | 主PAS / scope | 現在成立していること | 主な残Gap |
| --- | --- | --- | --- | --- |
| RQ-001 | **Strong Evidence** | PAS-01 | Canonical ID / source path分離、Semantic Data / Relation / Capability、Application-owned Pattern、Provider / Target固有実装の隔離 | 第一・第二Targetを通したmeaning安定性、複数Application再利用 |
| RQ-002 | **Strong Evidence** | PAS-01 | Source Index、Source Diagnostic、native source trace、file move / source trace基盤 | Studio / AIからauthoritative sourceへ戻るE2E |
| RQ-003 | **Strong Evidence** | PAS-01 | native source / derived IR、Evidence / Handover非Canonical、AI / Studio非authority方針 | Studio / AI authoring E2E |
| RQ-004 | **Partial Evidence** | PAS-03 | native SQL、CustomGoBinding、Plugin、legacy auth、external-observed Contract Query | existing batch / API / config、actual external migration Evidence |
| RQ-005 | **Partial Evidence** | PAS-02 | Application DatabaseRequirement、logical Datasource、`sales_core` / `master_data`、Driver data access | multi-DB transaction / failure、actual external E2E |
| RQ-006 | **Strong Evidence** | PAS-03 | Desired / Actual / Diff / Unresolved、PostgreSQL observation、ProviderExpression | Database外のchange / actual-state領域での実証 |
| RQ-007 | **Partial Evidence** | PAS-03 | Provider SPI、driver assembly分離、provider-specific normalization / Runner | actual second Database Provider圧力試験 |
| RQ-008 | **Partial Evidence** | PAS-02 | Semantic Capability、Binding、Environment、Plugin、legacy auth compatibility | builtin / generated / plugin / existingの同一Capability比較 |
| RQ-009 | **Partial Evidence** | PAS-02 | legacy Authentication / Authorization Reference、Gate A、Driver subject継承 | actual migration target、段階置換Evidence |
| RQ-010 | **Design Only** | PAS-01 | Studio非Canonical、high-density方針、Shared Acceptance、`ISSUE-000046` | Gate C後のStudio actual Reference、operability / accessibility |
| RQ-011 | **Partial Evidence** | PAS-04 | Source / Diagnostic / Workbench / Driver / Handover等の複数Projection | First Web Target、Studio authoring、second Target |
| RQ-012 | **Partial Evidence** | PAS-02 | Driver主要Scenario、Validation / Auth / Command / Observation、#310 runner | Gate B actual #310/#255/#308 same-R Evidence |
| RQ-013 | **Design Only** | PAS-04 | Gate C contract、baseline `R` / Target build `T`分離、Target Capability / Execution-Trust仮説 | First Web Target、second Target比較 |
| RQ-014 | **Partial Evidence** | PAS-02 | browse / record / reference / header-detail / commandをApplication intentとして実証 | 複数Application / Targetでの再利用Evidence |
| RQ-015 | **Design Only** | PAS-06 | knowledge-first原則、read-only exploration計画、derived cognition research境界 | Gate C後のexploration actual Reference |
| RQ-016 | **Design Only** | PAS-06 | Proposal / Decision分離、suppression方針、autonomy / authority分離 | Proposal / Decision / suppressionの実UX / Evidence |
| RQ-017 | **Partial Evidence** | PAS-01 | deterministic generated artifact、native source authority、Shared Acceptance設計 | Studio / AI / Target間で二重管理を作らないE2E |
| RQ-018 | **Strong Evidence** | PAS-02 | Business Result / Execution Error / separate Observation read | Target Projectionで同境界を維持するEvidence |
| RQ-019 | **Strong Evidence** | PAS-05 | Permission Grant、Resource Broker / Lease、Sandbox Profile、Host material分離 | production-grade enforcementの拡張 |
| RQ-020 | **Strong Evidence** | PAS-05 | ownership / heartbeat / fencing / durable handle / no-auto-replay / unknown保持 | 複数production Runtimeでの圧力試験 |
| RQ-021 | **Strong Evidence** | cross | Reference Workspace、Conformance、golden、source-fixed runners、Provider Evidence tools | Gate B actual run、Gate C Target Evidence |
| RQ-022 | **Strong Evidence** | cross | architecture probe / actual Evidence / production claimを分離 | v1以後のsupport / compatibility level整理 |
| RQ-023 | **Partial Evidence** | cross | monorepo、Go中心、ADR / Plan / Reference、Provider / Target分離 | Studio / Target / AI追加後の保守量、文書 / Evidence tool増加 |
| RQ-024 | **Design Only** | PAS-04 | First / Second Targetで測る方針、Gate C測定点、R/T trace | actual Target measurement、context-specific threshold |
| RQ-025 | **Partial Evidence** | PAS-07 | Business Action != Application Function、manual / external Action、actor / authority分離、Business Procedure Reference / Pressure | Lane C actual execution、独立second corpus、Canonical identity最終判断 |
| RQ-026 | **Partial Evidence** | PAS-07 | Data / Function / Business cross-view Reference、Handoff identity Pressure、derived Projection原則 | realistic second-case actual Evidence、diagnostic severityの確定 |
| RQ-027 | **Partial Evidence** | cross | Result / Error / Observation分離、source Diagnostic、secret-safe provider failure、runtime unknown / recovery境界 | Target / Runtime横断diagnostic correlation、degraded-mode actual Evidence |
| RQ-028 | **Partial Evidence** | PAS-01 / PAS-03 | Desired/Actual/Diff、generated check/repair分離、migration readiness、fresh-observation destructive apply、stale-base設計 | Database外を含むimpact / compatibility / recoveryの一貫したEvidence |
| RQ-029 | **Design Only** | PAS-04 / PAS-01 | high-density原則、Target-neutral interaction、Semantic/Functional/Visual分離、quality architecture | First Web Targetのkeyboard / focus / recovery / accessibility actual Evidence |
| RQ-030 | **Design Only** | cross | source-fixed Evidence、B/R/T identity、RQ-024 context方針、Quality Envelope設計 | actual Quality Envelope artifact、threshold rationale、cross-Target比較 |

## 5. 評価集約

### Strong Evidence — 9

- RQ-001
- RQ-002
- RQ-003
- RQ-006
- RQ-018
- RQ-019
- RQ-020
- RQ-021
- RQ-022

### Partial Evidence — 14

- RQ-004
- RQ-005
- RQ-007
- RQ-008
- RQ-009
- RQ-011
- RQ-012
- RQ-014
- RQ-017
- RQ-023
- RQ-025
- RQ-026
- RQ-027
- RQ-028

### Design Only — 7

- RQ-010
- RQ-013
- RQ-015
- RQ-016
- RQ-024
- RQ-029
- RQ-030

### Not Started — 0

新Requirementを追加したことで「Not Startedを0に保つ」ことが目的ではありません。RQ-029 / RQ-030は設計正本があるためDesign Onlyですが、actual Product Evidenceはまだありません。

## 6. Product Gapの再統合

30 RQを30本の独立workstreamにしません。現在の不確実性を次へ束ねます。

### Gap A — Gate B actual Evidenceを閉じる

主に:

- RQ-005
- RQ-009
- RQ-012
- RQ-014
- RQ-021
- RQ-027 / RQ-028の既存境界の一部

現在必要なのは新しいDriver abstractionやquality laneではありません。

```text
actual full checkout preflight
        ↓
#310 deterministic Repository
#255 generated artifact reproducibility
#308 real PostgreSQL Provider
        ↓
same clean candidate R
        ↓
#387 final convergence
        ↓
#273 Gate B audit
```

RQ-027〜030を理由に第四Required laneを追加しません。

### Gap B — Application meaningをFirst Targetへ出す

主に:

- RQ-011
- RQ-013
- RQ-018
- RQ-024
- RQ-027
- RQ-029
- RQ-030

Gate B Ready後、Handover Baseline `B / R`を固定し、First Web Target build `T1`を別identityとして追跡します。

Gate Cでは既存のSemantic / Functional / Visual分離を維持しつつ、次をEvidence計画へ加えます。

- Interaction / Operability
- applicable accessibility checks
- failure / degraded / recovery
- performance / resource context
- Quality Envelope provenance

一つの総合scoreへ畳みません。

### Gap C — 安全なdeveloper authoring

主に:

- RQ-002
- RQ-003
- RQ-010
- RQ-011
- RQ-017
- RQ-023
- RQ-028
- RQ-029
- RQ-030

Gate C後の最初のReferenceは`ISSUE-000046`です。

```text
Select
 -> Propose
 -> Preview Semantic Diff / impact
 -> Accept / Reject
 -> Shared Acceptance Boundary
 -> authoritative source
 -> reload / Conformance
```

追加quality Pressure:

- Preview / Reject non-mutating
- stale base conflict
- partial write / recovery
- high-density + keyboard / navigation
- failureからsource / subjectへ戻れる診断

### Gap D — knowledge-first reuse / AI / derived cognition

主に:

- RQ-014
- RQ-015
- RQ-016
- RQ-017
- RQ-023
- RQ-027
- RQ-030

Gate C後のread-only exploration / AI Proposalへ、derived cognition researchを必要な場合だけ接続します。

```text
retrieval candidate quality
  != semantic applicability
  != accepted decision
```

candidate recall、irrelevant candidate、latency、memory / index size等を評価しても、ranking scoreやmodel confidenceをSemantic Evidenceへ格上げしません。

### Gap E — multi-Target pressure / performance / interaction quality

主に:

- RQ-011
- RQ-013
- RQ-024
- RQ-029
- RQ-030

`ISSUE-000026`で第一Target `T1`と第二Target `T2`を明示baselineに対して比較します。

- Target Capability
- Runtime Binding
- Execution / Trust
- Host control plane
- Semantic / Functional / Visual / Interaction Evidence
- performance / resource
- failure presentation

T1とT2へ同じ数値を強制するのではなく、同じApplication meaningに必要なTarget別quality conditionを説明します。

### Gap F — actual external asset / Provider breadth

主に:

- RQ-004
- RQ-005
- RQ-007
- RQ-008
- RQ-009
- RQ-027
- RQ-028

actual Oracle #214 / #371 / #376を含む既存外部資産Evidenceを継続します。

Gate B第四Required laneへ戻しません。未完了ならHandover unresolvedとして追跡し、Provider作業だけで他Product laneを停止しません。

### Gap G — Cross-cutting Product Quality Envelope

主に:

- RQ-022
- RQ-024
- RQ-027
- RQ-028
- RQ-029
- RQ-030

最大の不足は「quality subsystemがない」ことではありません。

各領域に散在する品質主張を、次へ戻せる形でEvidence化することです。

```text
subject
+ revision / build
+ Target / Host / Provider / Runtime context
+ Scenario / precondition
+ property
+ measure / probe
+ acceptance rationale
+ artifact
+ status
```

最初のactual Quality EnvelopeはFirst Targetで具体化し、その後Studio / AI / Second Target / productionへ広げます。

## 7. 外部quality modelの利用

ISO/IEC 25010:2023、WCAG 2.2、OpenTelemetry semantic conventions、NIST SSDF 1.1等は、TEIのRequirementを置き換える正本にはしません。

用途を限定します。

- ISO/IEC 25010: product quality characteristicの抜け漏れ監査
- WCAG 2.2: Web Target accessibilityのtestable external reference
- OpenTelemetry: Operational Telemetry Projectionのinteroperability候補
- NIST SSDF: secure development / supply-chain processのchecklist

明示採用とactual Evidenceなしに準拠claimへ格上げしません。

## 8. 現在の優先順位

1. **Gate B actual final Evidenceを取得する**
   - #310 / #255 / #308
   - same candidate `R`
   - #387 final convergence
2. **Gate B ReadyならHandover Baseline `B / R`を固定する**
3. **First Web Target `T1`を実装しGate C Evidenceを取得する**
   - Semantic / Functional / Visualを維持
   - Interaction / accessibility / performance / failure qualityをcontext付きで観測
4. **Gate C Ready後、次を並行する**
   - Studio minimal Authoring Loop
   - knowledge-first / derived cognition exploration
   - AI Proposal / Acceptance
   - second Target pressure test
5. **各laneのQuality Envelopeを比較し、Requirement / Architectureへ反証を戻す**
6. **actual external asset / Provider / production qualityを必要な範囲で継続する**

quality planningを理由にGate B actual executionを後回しにしません。逆に、Gate B環境blockerを理由にFirst Targetを先行実装せず、上流quality contractだけを先に整備します。

## 9. 維持コストの監査

RQ-023 / RQ-030の観点から、品質管理そのものが負債にならないようにします。

- current snapshot文書へ履歴を無制限追記しない
- 実行手順の正本を複数作らない
- quality characteristicごとにGate / PASを増やさない
- planning-onlyの将来作業を不要にIssueへ二重起票しない
- 同じsemantic boundaryを複数planで別表現の正本にしない
- metricが増えたことを品質向上とみなさない
- Evidence toolを「安心のため」だけに増やさない
- 一つの総合scoreでunresolvedを隠さない

## 10. 結論

現在のTEIは、Application / Driverの中心meaningを作る段階から、**そのmeaningをactual Evidenceで閉じ、Handoverを経てFirst TargetへProjectionする段階**にあります。

今回の再監査で、次のquality gapを上流Requirementとして明示しました。

```text
failureを説明できる
changeを適用前に検査できる
interactionをTarget条件で操作・accessibility検証できる
quality claimをContext / Measure / Evidenceへ戻せる
```

これらは新しい巨大subsystemではありません。既存のSource / Canonical / Binding / Runtime / Target / Evidence境界を横断して、TEIのProduct claimをより厳密にするための設計基線です。