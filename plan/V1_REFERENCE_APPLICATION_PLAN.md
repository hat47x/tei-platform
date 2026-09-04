# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、Reference Application `order_entry`を中心に一つの主線として整理するものです。

個別Issueを別々に完了させることではなく、同じ業務上の意味をSemanticからApplication、Driver、第一Targetへ一続きに追跡できることを重視します。

期限は置きません。次へ進む条件は日付やIssue close数ではなく、Reference / Conformance / Evidenceから判断します。

## v1 Reference Application

v1では、新しい理想化サンプルを別に作らず、`reference/workspace/applications/order-entry`を育てます。

主な資産は次です。

- `sales` / `customer` / `security` Semantic Namespace
- `sales.Order` / `sales.OrderLine`
- `sales.order.customer` Relation
- `sales.order.lines` 1:N Relation
- `sales.order.submit` Capability
- `security.authenticate` / `security.authorize` Capability
- `sales_core`へのread-write Datasource
- `master_data`へのread-only Datasource
- Database Contract / native SQL Query / Query realization
- 5種類のApplication Pattern
- Capability Binding
- Canonical業務Validation
- legacy Authentication / Authorization Compatibility Reference
- CapabilityのBusiness Result / Execution Error境界
- header-detail Contract Query realization
- Database-owned Relation Storage
- Source-to-Canonical Traceability

M0.xxで積み上げたDatabase / Provider / RuntimeのEvidenceを捨てず、Application vertical sliceへ接続します。

## v1で確認する業務シナリオ

最初の業務Scenarioは次です。

```text
利用者を認証する
    ↓
受注一覧を探す
    ↓
受注を開く
    ↓
顧客を参照する
    ↓
受注明細を扱う
    ↓
入力内容を検証する
    ↓
利用者が操作できるか確認する
    ↓
受注を確定する
    ↓
業務結果と実行エラーを区別する
    ↓ separate explicit read
外部状態を必要に応じて観測する
```

この意味をDriver、Web Target、Editor、AIごとに作り直しません。同じCanonicalな意味を共有し、それぞれの層では必要なProjection / Binding / Evidenceだけを追加します。

## 全体の進行

```text
Stage 0  Product Baseline / Gap Audit         完了
               ↓
Stage 1  Application Pattern                  完了
               ↓
Gate A   Application上の意味をDriverへ渡せる   Ready
               ↓
Stage 2  Application Driver #69               In Progress
               ↓
Gate B   UIなしで主要Scenarioを再現できる       Not Ready / 監査中
               ↓
Stage 3  第一Web Target #70 + Evidence #270
               ↓
Gate C   Semantic / Functional / Visual Evidence
               ↓
Stage 4  Editor / AI支援 #267
               ↓
Stage 5  第二Target #259
```

Gate Dという独立した番号付きGateを先に固定するより、第一Web TargetのGate C EvidenceをEditor / AIと第二Targetの入力として明示的に受け渡します。第二Targetで追加のGateが必要かは、実際のTarget差から判断します。

## Stage 0 — Product Baseline / Gap Audit

完了しています。

Product Value / Vision / Principles / Requirements / Master Plan / Milestones / Gap Auditを基線としました。

この監査から、Application Pattern、既存認証Compatibility、Driver、Target、Editor、AI支援の実証を厚くする必要があると判断し、v1の主線をApplication vertical sliceへ移しました。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

完了しています。

第一Referenceでは次の5種類が成立しました。

```text
browse
record
reference
command
header-detail
```

確認したことは次です。

- PatternはApplicationが所有する意図として成立する
- Semantic / Query / Relation / Capabilityの意味をPatternへ複製しない
- Pattern identityをfile pathから分離できる
- Source Index / Diagnosticからauthoring sourceへ戻れる
- Semantic RelationとDatabase Foreign Keyを混同しない
- PatternをTarget固有UIへ固定しない

5分類をTEI全体の永久的な標準一覧とはみなしません。Driverと複数TargetのEvidenceから再評価します。

## Gate A — Application上の意味をDriverへ渡せる

関連: `ISSUE-000013` / GitHub Issue #124  
監査: [`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md)

**Readyです。**

Gate Aでは、Driverへ必要な意味をWeb画面やDriver専用設定なしで説明できる状態を確認しました。

主な成立領域は次です。

- Application Pattern
- Datasource read / write境界
- Capability Binding
- Canonical業務Validation
- Capability Business Result / Execution Error
- Authentication / Authorization
- legacy認証方式の交換
- Source / Binding traceability

Gate A Readyは「Driver実装が完了した」という意味ではありません。Stage 2を開始できる意味契約が揃ったという判定です。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69  
監査: [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) / `ISSUE-000029` / GitHub Issue #273

### 現在の状態

**In Progressです。Gate BはNot Readyです。**

以前の文書ではGate A Readyを受けてStage 2を「Ready」と表現していましたが、これは「着手可能」と「Gate B通過可能」を混同しやすいため廃止します。

現在はDriverの中心設計が大きく成立し、Gate Bを止める要因が外部Evidence、Observation、生成物再現性、最終Scenario統合へ収束しています。

### 成立している主な境界

#### Application / Capability

- 5 PatternのCanonical resolution
- Capability Binding
- Authentication / Authorization
- Business Result / Execution Error

#### browse

- logical Datasource resolution
- native Query execution
- provider-side parameter binding
- raw result
- explicit Query result mapping
- Semantic result

#### reference — `Order -> Customer`

構造・runtime境界は成立しています。

- Relation key
- Reference Contract Query invocation
- typed Database Contract Query
- SQL realization model
- Environment / Database instance / executor Capability
- instance-aware Registry
- Oracle SQL Runner

ただし`master_data`はexternal-observedで、実物理object / SQL / raw columns / provenanceが未観測です。#214でEvidenceを得るまでOracle物理情報を推測しません。

#### header-detail — `Order -> OrderLine[]`

keylessなSemantic Relationを維持したまま次まで成立しています。

- ADR-0021 Pattern外traversal realization
- HeaderDetailContractQueryBinding
- Database-owned Relation Storage
- support columnのDesired Physical Schema / migration
- PostgreSQL SQL realization
- Environment composition
- 0件 / 複数件 / Execution Error
- 実PostgreSQL Evidence tool

実DB成功Evidenceはまだ未取得です。

#### Validation -> Authorization -> Command

PR #265で次をReference Scenarioとして固定しました。

```text
Authentication
  ↓
Canonical Validation
  ↓
Authorization
  ↓
Command
```

Validation `NotSatisfied`、Validation評価Error、Authorization `denied`、Command Business Resultを区別し、不成立時にはCommandへ進みません。

### Command / Observation

関連: GitHub Issue #266 / ADR-0023

Command Business ResultをObserved stateへ変換しません。

v1では次を採用します。

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation Evidence
```

Query / Contract Query / read adapter等の既存の明示read境界を別操作として利用します。

Command payload、HTTP status、Capability名からObservation対象や結果を推測しません。

Observation専用Application Pattern、Semantic Capability、Workflow Engine、Event Sourcing基盤は第一Referenceだけから先行導入しません。

### Gate B blocker

2026-09-04時点の主なblockerは次です。

1. #214 external `master_data`実物理Evidence
2. header-detail実PostgreSQL成功Evidence
3. #266 Command / Observation最小Reference
4. #255 World / Plugin Lock等の正規再生成
5. 最終UI-less Scenario Evidence
6. CI再開後の通常runner再検証

CI停止そのものをSemantic design gapとは扱いませんが、未実行をgreenとも扱いません。

## Gate B — UIなしで主要Scenarioを再現できる

現在の判定は**Not Ready**です。

Gate Bは画面がなくても次をEvidenceから説明できることを要求します。

- 同じApplication定義から主要Scenarioを一続きに再現できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- external assetの未観測事実を推測していない
- read-only / secret等の安全境界を保持できる
- Validation / Authorization不成立時にCommandへ進まない
- Business Result / Execution Error / Observationを区別できる
- Query 0件とExecution Errorを区別できる
- 最新authoringからReference artifactを再現できる
- 第一Web TargetへWeb側の意味補完なしに同じbaselineを渡せる

最新判定は`GATE_B_READINESS_AUDIT.md`を正とします。

## Gate Bから第一Web Targetへ渡すbaseline

Gate B通過時には少なくとも次を渡します。

- browse
- record
- reference
- header-detail
- Canonical Validation
- Authentication / Authorization
- command
- Business Result / named Outcome
- Execution Error
- Observation
- Datasource read / read-write境界
- Source / Binding traceability
- Driver Scenario Evidence

Web側で不足を独自実装して隠しません。不足が見つかった場合はGate B baselineまたは上位の意味境界へ戻します。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70  
Evidence実証: `ISSUE-000028` / GitHub Issue #270  
契約: [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)

### 着手条件

Gate B通過後です。

Gate B前にframework、画面構造、Presentation IRを先に固定しません。

### 目的

Driverで成立した同じApplication上の意味を第一Web TargetへProjectionし、次を実例から分離します。

- Application上の意味
- Target-neutral Presentation候補
- Target Capability候補
- Runtime Binding差
- Execution / Trust差
- Web固有実装
- Host control plane固有差
- Canonicalへ持ち上げてはいけない差
- unresolved

第一Targetだけで完全なTarget-neutral taxonomyを確定しません。

### 対象

- browse
- record
- reference
- header-detail
- Validation feedback
- Authorizationに応じた利用可否
- command
- Business Result
- Execution Error
- Observation
- standard Projection
- Custom UI

## Gate C — 第一Targetの境界をEvidenceで説明できる

第一Web Targetが表示できただけでは通過しません。

`GATE_C_EVIDENCE_CONTRACT.md`に従い、次の三種類を別Evidenceとして扱います。

1. **Semantic Conformance**
2. **Interaction / Functional Evidence**
3. **Visual Fidelity Evidence**

Visual evaluator、VLM、screenshot diff、GUI agent等は利用できますが、Visual scoreだけでSemantic / Functionalを合格にしません。

Gate Cではさらに、Webで見えたTarget Capability / Execution / Trust / Web固有差を第二Targetへ渡せることを要求します。

Gate Cの合否は#70や#270のclose状態だけではなくEvidenceから判断します。

## Stage 4 — Editor / AI支援

関連: `ISSUE-000008` / `ISSUE-000027` / GitHub Issue #267  
設計判断: ADR-0022

### 着手条件

Gate C通過後です。

### Editor

EditorはCanonicalを所有しません。

file-native / CLI / AIと同じAcceptance Boundaryへauthoring candidateを渡すProjection / design environmentとして扱います。

高密度に確認したい対象は次です。

- Application / Datasource
- Data / Relation
- Application Pattern
- Validation / Command
- Authorization
- Source / Diagnostic
- Driver result
- Target差
- Decision / Proposal history

Go + Wails/WebViewは有力なHost候補ですが、Wails APIをCanonical Presentationへ持ち上げません。

### AI支援

AI出力をCanonicalへ直接書きません。

```text
AI / Editor / CLI / file-native
        ↓
Proposal / authoring candidate
        ↓
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical World
```

AIでは生成量より次を重視します。

- 既存知識の探索
- 変更候補
- Validation / Conformance
- Source-to-Canonical Traceability
- stale proposal検出
- autonomyとauthorityの分離
- Derived Experience Knowledgeを第二の正本にしない

## Stage 5 — 第二Target

関連: `ISSUE-000026` / GitHub Issue #259

第二Targetは対応Target数を増やすことより、第一Web Targetで得た境界の圧力試験を目的とします。

候補はWails Desktop / Mobile / Server、API、MCP等ですが、Gate C Evidenceから不確実性を最も減らせるものを選びます。

比較する主な軸は次です。

- Targetが変わっても残るCanonical Application上の意味
- Target Capability差
- Execution / Trust Profile差
- Runtime Binding差
- Target固有実装
- Host control plane差
- Canonicalへ持ち上げてはいけない差

第二Targetで境界修正が必要になっても、それ自体を失敗とはみなしません。

## Evidenceの残し方

各Stageは「実装した」だけでは完了にしません。

Evidence候補は次です。

- authoritative Reference source
- deterministic fixture
- Conformance test
- generated / golden artifact
- Driver Scenario
- real external / managed-here observation
- Target comparison
- ADR
- readiness audit
- unresolvedを明示したDiagnostic / Evidence

Evidence artifactをCanonicalの第二正本にはしません。

実験結果が予想と異なる場合、その差を隠して計画へ合わせません。Product Valueに照らし、Requirement / Architecture / Planを修正します。

## まだ固定しないこと

- Application Patternの永久的な標準一覧
- 第一Web Targetのframework
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- Editor最終UI framework
- AI provider / Proposal最終schema
- 第二Targetの種類
- B2C向け一律性能目標
- Authentication実装の単一方式
- `std.auth.*`標準契約
- 汎用Workflow Engine
- Event Sourcing標準
- 汎用retry / polling framework
- production HA / Connection Pool一般化

ReferenceからEvidenceを得る前に固定しません。

## v1へ進んだと判断する条件

少なくとも次をRepositoryのEvidenceから説明できることを目安とします。

- 一つの現実的なApplicationをSemanticから第一Targetまで一続きに追跡できる
- 複数Databaseを自然に利用できる
- native SQLとCustom implementationを正本のまま利用できる
- 既存認証仕様を保持したままCapabilityへ接続できる
- Application PatternをDriverとTargetで共有できる
- UIなしでもDriverで主要Scenarioを検証できる
- 第一Targetでstandard ProjectionとCustom UIを同じ上位意味から利用できる
- Business Result / Execution Error / Observationを必要な箇所で分離できる
- Semantic / Functional / Visual Evidenceを混同しない
- Reference / Conformanceから同じ意味を再現できる
- 何が未保証かを監査文書 / statusから判断できる

Editor / AI支援の完成、第二Target、production HAは第一Web Targetまでのv1成立条件にはしません。ただし、それらを後続検証できる境界は残します。

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/adr/0021-header-detail-relation-traversal-realization.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/adr/0023-command-observation-explicit-read-boundary.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
