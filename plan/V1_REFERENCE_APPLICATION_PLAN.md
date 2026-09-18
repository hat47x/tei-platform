# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、Reference Application `order_entry`を中心に一つの主線として整理するものです。

個別Issueのclose数ではなく、同じ業務上の意味をSemantic → Application → Driver → Handover → Target → Evidenceへ追跡できることを重視します。

期限は置きません。次へ進む条件はReference / Conformance / Evidenceで判断します。

過去のscopeや旧Evidence laneは本文へ履歴として重ねず、Git history / ADR / Issueから追跡します。

## v1 Reference Application

v1では、新しい理想化sampleを別に作らず、`reference/workspace/applications/order-entry`を育てます。

主な資産:

- `sales` / `customer` / `security` Semantic Namespace
- `sales.Order` / `sales.OrderLine`
- `sales.order.customer` Relation
- `sales.order.lines` 1:N Relation
- `sales.order.submit` Capability
- `security.authenticate` / `security.authorize` Capability
- `sales_core` read-write Datasource
- `master_data` read-only / external-observed Datasource
- Database Contract / native Query / Query realization
- Application Pattern
- Capability Binding
- Canonical Validation
- legacy Authentication / Authorization Compatibility Reference
- Business Result / Execution Error境界
- header-detail Contract Query realization
- Database-owned Relation Storage
- Source-to-Canonical Traceability

## v1業務Scenario

```text
Authentication
    ↓
browse Orders
    ↓
record open
    ↓
reference Customer
    ↓
header-detail OrderLine[]
    ↓
Canonical Validation
    ↓
Authorization
    ↓
Command Business Result
    ↓ separate explicit read
Observation
```

この意味をDriver、Web、Studio、AIごとに作り直しません。同じCanonical meaningを共有し、それぞれの層ではProjection / Binding / Evidenceだけを追加します。

## PASとの対応

| PAS | v1での主な実証 |
| --- | --- |
| PAS-01 | Gate C後のStudio minimal Authoring Loop / AI Proposal Acceptance |
| PAS-02 | Application Driver / Gate B / Handover |
| PAS-03 | external asset / Provider / Handover unresolved |
| PAS-04 | 第一Web Target / Gate C / 第二Target |
| PAS-05 | Runtime / deployment recovery系で継続。Gate B blockerへ追加しない |
| PAS-06 | Gate C後のknowledge-first Semantic exploration |

## 全体の進行

```text
Stage 0  Product Baseline / Gap Audit                 Done
               ↓
Stage 1  Application Pattern                          Done
               ↓
Gate A   Application meaning → Driver                 Ready
               ↓
Stage 2  Application Driver                           major semantics ready
               ↓
Gate B   3 Required Evidence same-R convergence       Not Ready
               ↓
Handover Baseline B / revision R
               ↓
Stage 3  First Web Target / build revision T1
               ↓
Gate C   Semantic / Functional / Visual Evidence
               ↓
        ┌───────────────┴────────────────┐
        ↓                                ↓
Stage 4A Authoring / Knowledge        Stage 4B Second Target
  - Studio PAS-01                       - PAS-04
  - knowledge PAS-06                    - RQ-024
  - AI Proposal PAS-01/06
```

Stage 4Aと4Bは並行できます。Studio / AI完成を第二Targetの前提にはしません。

Handoverは新しい番号付きGateではありません。Gate Bの出力をGate Cの入力へ固定する契約です。

## Stage 0 — Product Baseline / Gap Audit

**Done**です。

Product Value / Vision / Principles / Requirements / PAS / Master Plan / Gap AuditをRepository基線として扱います。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67

**Done**です。

第一Referenceでは次の5種類をApplication所有の意図として実証しました。

```text
browse
record
reference
header-detail
command
```

この5分類をTEI全体の永久的な標準Catalogとはみなしません。

## Gate A — Application上の意味をDriverへ渡せる

関連: `ISSUE-000013` / GitHub Issue #124  
監査: `GATE_A_READINESS_AUDIT.md`

**Ready**です。

主な成立領域:

- Application Pattern
- Datasource read / write
- Capability Binding
- Canonical Validation
- Business Result / Execution Error
- Authentication / Authorization
- legacy認証Compatibility
- Source / Binding traceability

Gate A ReadyはDriver完成ではなく、Driverへ渡す意味契約が成立したことを示します。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69  
Gate B監査: `GATE_B_READINESS_AUDIT.md` / `ISSUE-000029` / #273  
Evidence要求: `GATE_B_EVIDENCE_ACQUISITION.md`  
実行手順: `GATE_B_EVIDENCE_RUNBOOK.md`  
最終収束: `ISSUE-000037` / #387

### 現在の状態

主要な意味境界とScenario構造は成立しています。

Gate Bは**Not Ready**で、actual Evidence取得待ちです。

### Gate B Required Evidence

Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。

actual physical facts、provenance、effective read-only、driver integration、actual readは重要なPAS-03 / external asset Evidenceとして継続します。未完了ならHandoverへ`unresolved`として渡します。

### Driverで成立している主な境界

- Application Pattern resolution
- Datasource resolution
- Query / Contract Query
- explicit input / result mapping
- Relation traversal
- header-detail
- Validation
- Authentication / Authorization
- Capability Binding
- Command Business Result
- Execution Error
- separate Observation read

### Command / Observation

Command acknowledgementをObserved external stateへ変換しません。

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation / Execution Error
```

Business Result、Execution Error、Observation、Diagnosticを別々に扱います。

### Gate B current blocker

現在の直接blockerは新しい抽象設計ではなく、actual executionです。

- actual TEI checkoutで#310
- clean check-only #255
- Evidence専用空PostgreSQLで#308
- #387でsame-R final convergence
- #273でReady再監査

## Gate B — UIなしで主要Scenarioを再現できる

現在: **Not Ready**

Readyには少なくとも次を要求します。

- #310 / #255 / #308が同じcandidate `R`
- 各Evidenceの`reproducibleRevision = true`
- source drift / dirty treeを採用しない
- generated artifactを手編集しない
- PostgreSQL fixture前後の安全境界が成立する
- synthetic / generated / real-provider / actual external Evidenceを相互代替しない
- Application meaning不足をWeb側へ先送りしない

Ready判定の正本は`GATE_B_READINESS_AUDIT.md`です。

## Gate B → Gate C Handover Baseline

関連: `GATE_B_TO_C_HANDOVER_CONTRACT.md` / `ISSUE-000030` / #291

Gate B Ready時、candidate revision `R`をHandover Baseline `B`へ固定します。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

少なくとも次へ追跡します。

- Handover Baseline ID
- baseline repository revision `R`
- Reference Workspace
- Application `order_entry`
- Pattern / Data / Relation / Query / Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / Execution Error / Observation
- Driver Scenario
- Gate B Evidence
- unresolved

Handover recordはCanonicalの第二正本ではありません。

Oracle未完了範囲をWeb側で推測・補完しません。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / #70  
Evidence: `ISSUE-000028` / #270  
契約: `GATE_C_EVIDENCE_CONTRACT.md`

### 着手条件

**Gate B Ready + Handover Baseline固定後**です。

### baseline RとTarget build T1

Web Targetの実装revisionを`T1`として別に追跡します。

```text
Handover Baseline B at R
        ↓ meaning baseline
First Web Target build T1
        ↓
Gate C Evidence
```

通常`T1 != R`です。

`T1`でbaseline subjectの意味変更が必要になった場合、Web固有fieldやruleで埋めません。authoritative sourceへ戻り、必要なDriver / Gate B Evidenceを再確認して新Handover Baselineを発行します。

### 対象Scenario

- browse
- record open
- reference lookup
- header-detail read
- header-detail edit（baselineにwrite意味がある場合）
- Validation satisfied / not satisfied
- Authorization allowed / denied
- command
- Business Result
- Execution Error
- Observation
- standard Projection
- Custom UI

### 観測する差

1. Application meaning
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

## Gate C — 第一Targetの境界をEvidenceで説明できる

第一Web Targetが表示できただけでは通過しません。

Evidenceを三種類へ分けます。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

すべてのrunからHandover Baseline `B` / revision `R`とTarget build `T1`へ戻れる必要があります。

Gate Cで確認すること:

- DriverとWebが同じApplication meaningを共有する
- baseline `R`とTarget build `T1`を混同しない
- Semantic Conformance成立
- Reference Scenario Functional Evidence成立
- replayable behaviorを利用する場合、semantic operation / raw Target Evidenceへtrace可能
- Visual Evidenceを別に保持
- Business Result / Execution Error / Observationを維持
- Web Adapter除去でCanonicalが変わらない
- standard Projection / Custom UIが同じ上位意味へ接続
- Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を第二Targetへ渡せる
- unresolvedを成功扱いしない

visual score / VLM judgeだけでSemantic / Functionalを合格にしません。

## Stage 4A — Authoring / Knowledge / AI

Gate C Ready後に開始します。

### Studio minimal Authoring Loop — `ISSUE-000046`

Product Acceptance Scenario: PAS-01

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
authoritative source
  ↓ reload / Conformance
```

StudioはProjection / Authoring Clientであり第二正本ではありません。

file-native / CLI / AIと同じAcceptance原則へ収束します。

Studioのdeveloper-facing Wails HostをGenerated Wails Desktop Application Targetと混同しません。

### knowledge-first Semantic exploration — `ISSUE-000047`

Product Acceptance Scenario: PAS-06

Human / AIが新規authoring前に既存Data / Relation / Validation / Capability / Application Pattern / Evidenceをread-only探索します。

探索候補を次へ分けます。

- applicable
- incompatible
- unknown

search ranking、similarity、embedding、AI confidenceはSemantic Evidenceではありません。

### AI Proposal / Acceptance — `ISSUE-000027` / #267

Product Acceptance Scenario: PAS-01 / PAS-06

AI ProposalはCanonical authorityではありません。

- explicit Canonical / Source / Evidence references
- `baseModelRevision`
- Preview Semantic Diff
- stale conflict
- shared Acceptance
- accepted source reload
- Validation / Conformance
- autonomy / authority分離

AI都合でmandatory authored Stable IDやPatch-only authoring modelを導入しません。

## Stage 4B — 第二Target pressure test

関連: `ISSUE-000026` / #259  
Product Acceptance Scenario: PAS-04

Stage 4Aの完成を待たず、Gate C Ready後に並行開始できます。

第二Targetは対応数を増やすためではなく、第一Web Targetで得た境界を圧力試験するために選びます。

比較軸:

- Handover Baseline / Canonical meaning
- Target build `T1` / `T2`
- Target Capability
- Projection差
- Runtime Binding
- execution artifact / ABI / Host
- Execution / Trust Profile
- Authorization / Resource
- Observation
- Semantic / Functional / Visual Evidence
- RQ-024 performance measurement

Generated Wails Desktop、Mobile、Server、API、MCP Application Target等が候補です。

Studio Wails Hostは第二Application Target Evidenceへ数えません。

第二Targetで境界修正が必要になっても、それ自体を失敗とはみなしません。第一Targetの都合を共通意味へ誤昇格していなかったかを確認します。

## Evidenceの原則

各Stageは「実装した」だけでは完了にしません。

Evidence候補:

- authoritative source
- Source Index / Diagnostic
- deterministic fixture
- Conformance test
- generated / golden artifact
- Driver Scenario
- real-provider observation
- actual external asset observation
- Handover Baseline
- Target build / run identity
- Semantic / Functional / Visual Evidence
- Target comparison
- ADR
- readiness audit
- unresolved Diagnostic

Evidence artifactやHandover recordをCanonicalの第二正本にはしません。

実験結果が予想と異なる場合、その差を隠して計画へ合わせずRequirement / Architecture / Planを再評価します。

## まだ固定しないこと

- Application Patternの永久標準一覧
- 第一Web Target framework
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- Studio最終UI framework
- AI provider
- vector database
- Proposal最終schema
- mandatory authored Stable ID / UUID
- Patch-only authoring model
- Handover汎用schema
- 第二Target種類
- B2C向け一律performance threshold
- Authentication単一方式
- 汎用Workflow Engine
- Event Sourcing標準
- production HA一般化

## v1へ進んだと判断する条件

v1のApplication / first-target成立判断では少なくとも次をRepository Evidenceから説明できることを目安とします。

- `order_entry`をSemanticから第一Targetまで追跡できる
- Gate B Ready revision `R`をHandover Baselineとして固定できる
- 第一Target Evidenceからbaseline `R`とTarget build `T1`へ戻れる
- 複数Databaseを自然に利用できる
- native SQL / Custom implementationを正本のまま利用できる
- 既存認証仕様を保持したままCapabilityへ接続できる
- Application PatternをDriver / Targetで共有できる
- UIなしでもDriverで主要Scenarioを検証できる
- standard Projection / Custom UIを同じ上位意味から利用できる
- Business Result / Execution Error / Observationを分離できる
- Semantic / Functional / Visual Evidenceを混同しない
- 何が未保証かをaudit / unresolvedから判断できる

Studio / AI支援の完成、第二Target、production HAは第一Web Targetまでのv1成立条件にはしません。ただし、それらをGate C後に検証できる境界とplan issueは保持します。

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/adr/0023-command-observation-explicit-read-boundary.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
