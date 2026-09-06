# Product Requirement Gap Audit

- 監査日: 2026-09-05
- 対象: RQ-001〜RQ-024
- 上流Trace: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`

## 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`の各Requirementについて、現在のRepositoryにどのEvidenceがあり、どの不確実性が残っているかを逆マッピングする監査です。

実装量やIssue close数ではなく、次を見ます。

- semantic boundaryが成立しているか
- Reference / Conformanceで再現できるか
- actual execution Evidenceが必要な場合に取得済みか
- synthetic / generated / real-provider / actual-externalを混同していないか
- Provider / Target / Host条件を説明できるか
- unresolvedを成功へ読み替えていないか

本書は現在断面だけを保持します。旧評価はGit historyから追跡します。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装・Reference・Conformance / Evidenceが一続きで存在し、主要意味境界を説明できる |
| **Partial Evidence** | 重要部分は成立しているが、Requirement全体またはactual / multi-context Evidenceが不足 |
| **Design Only** | 要求・契約・Planは具体化されているが、Reference実証はまだない |
| **Not Started** | 対応する設計・Referenceがほぼない |

## 現在のProduct主線

現在最大の直列blockerは**Gate B actual Evidence**です。

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
   ├─ knowledge-first exploration — PAS-06
   ├─ AI Proposal / Acceptance — PAS-01 / PAS-06
   └─ Second Target — PAS-04 / RQ-024
```

Gate C後の4本は並行可能です。

actual Oracle #214 / #371 / #376は重要なPAS-03 / external asset Evidenceとして継続しますが、Gate Bの第四Required laneではありません。

## Requirement別監査

| RQ | 評価 | 主PAS | 現在成立していること | 主な残Gap |
| --- | --- | --- | --- | --- |
| RQ-001 | **Strong Evidence** | PAS-01 | Canonical ID / source path分離、Semantic Data / Relation / Capability、Application-owned Pattern、Provider / Target固有実装の隔離 | 第一・第二Targetを通したmeaning安定性、複数Application再利用 |
| RQ-002 | **Strong Evidence** | PAS-01 | Source Index、Source Diagnostic、native source trace、file move / source trace基盤 | Studio / AIからauthoritative sourceへ戻るE2E |
| RQ-003 | **Strong Evidence** | PAS-01 | native source / derived IR、Source Index、Evidence / Handover非Canonical、AI / Studio非authority方針 | Studio / AI authoring E2E |
| RQ-004 | **Partial Evidence** | PAS-03 | native SQL、CustomGoBinding、Plugin、legacy auth、external-observed Contract Query | existing batch / API / config、actual external migration Evidence |
| RQ-005 | **Partial Evidence** | PAS-02 | Application DatabaseRequirement、logical Datasource、`sales_core` / `master_data`、Driver data access | multi-DB transaction / failure、actual external E2E |
| RQ-006 | **Strong Evidence** | PAS-03 | Desired / Actual / Diff / Unresolved、PostgreSQL observation、ProviderExpression | Database外で同原則が必要な領域の実証 |
| RQ-007 | **Partial Evidence** | PAS-03 | Provider SPI、driver assembly分離、provider-specific normalization / Runner | actual second Database Provider圧力試験 |
| RQ-008 | **Partial Evidence** | PAS-02 | Semantic Capability、Binding、Environment、Plugin、legacy auth compatibility | builtin / generated / plugin / existingの同一Capability比較 |
| RQ-009 | **Partial Evidence** | PAS-02 | legacy Authentication / Authorization Reference、Gate A、Driver subject継承 | actual migration target、段階置換Evidence |
| RQ-010 | **Design Only** | PAS-01 | Studio非Canonical、high-density方針、Shared Acceptance、`ISSUE-000046` | Gate C後のminimal Authoring Loop actual Reference |
| RQ-011 | **Partial Evidence** | PAS-04 | Source / Diagnostic / Workbench / Driver / Handover等の複数Projection | First Web Target、Studio authoring、second Target |
| RQ-012 | **Partial Evidence** | PAS-02 | Driver主要Scenario、Validation / Auth / Command / Observation、#310 runner | Gate B actual #310/#255/#308 same-R Evidence |
| RQ-013 | **Design Only** | PAS-04 | Gate C contract、baseline `R` / Target build `T`分離、Target Capability / Execution-Trust仮説 | First Web Target、second Target比較 |
| RQ-014 | **Partial Evidence** | PAS-02 | browse / record / reference / header-detail / commandをApplication intentとして実証 | 複数Application / Targetでの再利用Evidence |
| RQ-015 | **Design Only** | PAS-06 | knowledge-first原則、`ISSUE-000047`、derived Knowledge非authority | Gate C後のread-only exploration Reference |
| RQ-016 | **Design Only** | PAS-06 | Proposal / Decision分離、suppression方針、autonomy / authority分離、`ISSUE-000027` | Proposal / Decision / suppressionの実UX / Evidence |
| RQ-017 | **Partial Evidence** | PAS-01 | deterministic generated artifact、native source authority、Shared Acceptance設計 | Studio / AI / Target間で二重管理を作らないE2E |
| RQ-018 | **Strong Evidence** | PAS-02 | ADR-0023、Business Result / Execution Error / separate Observation read | Target Projectionで同境界を維持するEvidence |
| RQ-019 | **Strong Evidence** | PAS-05 | Permission Grant、Resource Broker / Lease、Sandbox Profile、Host material分離 | production-grade enforcementの拡張 |
| RQ-020 | **Strong Evidence** | PAS-05 | ownership / heartbeat / fencing / durable handle / no-auto-replay / unknown保持 | 複数production Runtimeでの圧力試験 |
| RQ-021 | **Strong Evidence** | cross | Reference Workspace、Conformance、golden、source-fixed runners、Provider Evidence tools | Gate B actual run、Gate C Target Evidence |
| RQ-022 | **Strong Evidence** | cross | architecture probe / actual Evidence / production claimを分離 | v1以後のsupport / compatibility level整理 |
| RQ-023 | **Partial Evidence** | cross | single repo、Go中心、ADR / Plan / Reference、Provider / Target分離 | Studio / Target / AI追加後の保守量、文書 / Evidence tool増加 |
| RQ-024 | **Design Only** | PAS-04 | First / Second Targetで測る方針、Gate C測定点、R/T trace | actual Target measurement、比較、contract |

## 評価集約

### Strong Evidence

- RQ-001
- RQ-002
- RQ-003
- RQ-006
- RQ-018
- RQ-019
- RQ-020
- RQ-021
- RQ-022

### Partial Evidence

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

### Design Only

- RQ-010
- RQ-013
- RQ-015
- RQ-016
- RQ-024

### Not Started

なし。

Design Onlyを実装済み・実証済みとは読み替えません。

## Product Gapの再統合

24 RQを24本の独立workstreamにしません。現在の不確実性は次へ束ねます。

### Gap A — Gate B actual Evidenceを閉じる

主に:

- RQ-005
- RQ-009
- RQ-012
- RQ-014
- RQ-021

現在必要なのは新しいDriver abstractionではありません。

```text
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

actual TEI checkout + Evidence専用PostgreSQL実行環境が必要です。

### Gap B — Application meaningをFirst Targetへ出す

主に:

- RQ-011
- RQ-013
- RQ-018
- RQ-024

Gate B Ready後、Handover Baseline `B` / revision `R`を固定し、First Web Target build `T1`を別identityとして追跡します。

Gate Cでは:

- Semantic Conformance
- Interaction / Functional Evidence
- Visual Fidelity Evidence
- performance measurement

を分けます。

Target実装中にbaseline meaningを変更する必要が生じた場合はsuccessor Handoverを発行し、Web固有field / ruleで不足を隠しません。

### Gap C — 安全なdeveloper authoring

主に:

- RQ-002
- RQ-003
- RQ-010
- RQ-011
- RQ-017
- RQ-023

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

Studioは第二正本ではありません。

file moveとlogical renameを区別し、Studio都合でmandatory authored Stable ID / UUIDを導入しません。

### Gap D — knowledge-first reuse / AI Proposal

主に:

- RQ-014
- RQ-015
- RQ-016
- RQ-017
- RQ-023

Gate C後:

1. `ISSUE-000047` — Human / AI共通read-only Semantic exploration
2. `ISSUE-000027` — AI Proposal / Shared Acceptance

へ分けます。

検索順位、similarity、embedding、model confidenceはSemantic Evidenceではありません。

AI Knowledge / index / trajectoryはderivedであり第二正本にはしません。

### Gap E — multi-Target pressure test / performance

主に:

- RQ-011
- RQ-013
- RQ-024

`ISSUE-000026`で第二Targetを選び、第一Target `T1`と第二Target `T2`を明示baselineに対して比較します。

- Target Capability
- Runtime Binding
- Execution / Trust
- Host control plane
- Semantic / Functional / Visual Evidence
- performance

を分離します。

Studioのdeveloper-facing Wails HostをApplication Target Evidenceへ数えません。

### Gap F — actual external asset / Provider breadth

主に:

- RQ-004
- RQ-005
- RQ-007
- RQ-008
- RQ-009

actual Oracle #214 / #371 / #376を含む既存外部資産Evidenceを継続します。

ただし、これをGate B第四Required laneへ戻しません。未完了ならHandover unresolvedとして追跡し、Provider作業だけで他Product laneを停止しません。

## 現在の優先順位

1. **Gate B actual final Evidenceを取得する**
   - #310 / #255 / #308
   - same candidate `R`
   - #387 final convergence
2. **Gate B ReadyならHandover Baseline `B` / `R`を固定する**
3. **First Web Target `T1`を実装しGate C Evidenceを取得する**
4. **Gate C Ready後、次を並行する**
   - Studio minimal Authoring Loop — `ISSUE-000046`
   - knowledge-first exploration — `ISSUE-000047`
   - AI Proposal / Acceptance — `ISSUE-000027`
   - second Target pressure test — `ISSUE-000026`
5. **各laneのEvidenceを相互に戻して境界を再評価する**
6. **actual external asset / Provider / production境界を必要な範囲で継続する**

## 維持コストの監査

RQ-023の観点から、上流整理そのものが過剰な管理負担にならないようにします。

- current snapshot文書へ履歴を無制限追記しない
- 実行手順の正本を複数作らない
- Gate / PASを安易に増やさない
- planning-onlyの将来作業を不要にGitHub Issueへ二重起票しない
- 同じsemantic boundaryを複数planで別表現の正本にしない
- Evidence toolを「安心のため」だけに増やさず、異なるEvidence nature / failure modeがある場合だけ分離する

今回の上流再基線化では、`MASTER_PLAN`、`V1_REFERENCE_APPLICATION_PLAN`、`MILESTONES`、PAS Trace、Gate B / Gate C契約をcurrent-snapshotとして整理し、旧判断はGit historyへ戻しました。

## 結論

現在のTEIは、Application / Driverの中心意味を作る段階から、**その意味をactual Evidenceで閉じ、明示Handoverを経てFirst TargetへProjectionする段階**へ移っています。

最大の直列blockerはGate B actual runです。

その後の最大不確実性はTarget実証ですが、Gate C後はStudio、knowledge-first / AI、第二Targetを一つの巨大な直列計画にはせず、小さいReferenceを並行して互いに圧力試験します。
