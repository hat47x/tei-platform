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
Stage 2  Application Driver #69               主要境界は成立
               ↓
Gate B   UIなしで主要Scenarioを再現できる       Not Ready / Evidence取得中
               ↓
Handover Gate B → C baseline #291
               ↓
Stage 3  第一Web Target #70 + Evidence #270
               ↓
Gate C   Semantic / Functional / Visual Evidence
               ↓
Stage 4  Editor / AI支援 #267
               ↓
Stage 5  第二Target #259
```

Handoverは新しい番号付きGateではありません。Gate Bの出力をGate Cの入力へ曖昧なく引き渡すための契約です。

Gate Dのような次の番号付きGateを先に固定するより、第一Web TargetのGate C EvidenceをEditor / AIと第二Targetの入力として明示的に受け渡します。第二Targetで追加のGateが必要かは、実際のTarget差から判断します。

## Stage 0 — Product Baseline / Gap Audit

完了しています。

Product Value / Vision / Principles / Requirements / Master Plan / Milestones / Gap Auditを基線としました。

この監査から、Application Pattern、既存認証Compatibility、Driver、Target、Editor、AI支援の実証を厚くする必要があると判断し、v1の主線をApplication vertical sliceへ移しました。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](design/APPLICATION_PATTERN_V1_DESIGN.md)

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
Evidence取得: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)

### 現在の状態

**主要な意味境界とScenario構造は成立しています。Gate BはNot Readyで、現在はEvidence取得段階です。**

### 2026-09-04 現行Gate B scope（ISSUE-000035 / #380）

Gate B Ready判定へ必須とするsource-dependent実行Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider Evidence

最終判定では、この三つの完了Evidenceを同じclean candidate revision `R`へ揃えます。

#214 / #371 / #376のactual Oracle EvidenceはGate B必須blockerではありません。`master_data`のactual physical facts / provenance / effective read-only / driver integration / actual readは引き続き未実証であり、完了扱いにはしません。actual external asset / legacy compatibility / 第二Database Provider圧力試験として継続します。

#214が未完了のままGate Bを通過する場合は、#291 Handover Baselineへ未検証範囲を`unresolved`として明示します。第一Web Target側でOracle physical factsやrealizationを補完せず、#310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしません。

Gate Bではexternal-observed assetを推測で埋めない境界を維持し、#310のexternal Evidence intake Conformanceで確認します。

**Gate BはまだNot Readyです。** #310 / #255 / #308のactual run Evidenceは未取得です。Gate B ReadyとHandover固定前にStage 3へ進みません。

以下の「Gate Bを直接止める四系統」等の記述は`ISSUE-000035`以前の計画履歴として保持します。現行必須scopeと矛盾する場合は本節を優先し、各Evidence種別の非代替性・安全条件は維持します。

PR #285でContract Query、header-detail、Relation Storage、Observation、最終UI-less Scenario、Reference authoringを`main`へ収束しました。

したがって、Stage 2の中心課題は新しいDriver抽象を増やすことではなく、成立済み境界へ実外部・実DB・生成物再現・通常runnerのEvidenceを揃えることです。

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

raw列名、列順、Persistence MappingからSemantic attributeを推測しません。

#### reference — `Order -> Customer`

構造・runtime境界は成立しています。

- Relation key
- Reference Contract Query invocation
- typed Database Contract Query
- Contract Query SQL realization model
- explicit input / result mapping
- Environment / Database instance / executor Capability
- instance-aware Registry
- Oracle向け最小`database/sql` Runner
- 0件 / Execution Errorの分離

ただし`master_data`はexternal-observedで、実物理object / parameter / raw columns / native realization / provenanceが未観測です。#214でEvidenceを得るまでOracle物理情報を推測しません。

read-onlyについても`ownership external-observed`という宣言だけで済ませず、credential / role / session / view / API等の実効的な強制境界を確認します。

#### header-detail — `Order -> OrderLine[]`

keylessなSemantic Relationを維持したまま次まで成立しています。

- ADR-0021 Pattern外traversal realization
- `HeaderDetailContractQueryBinding`
- Database-owned Relation Storage
- `sales.order.order_id -> order_id` support column
- Desired Physical Schema / migration
- PostgreSQL SQL realization
- Environment composition
- 0件 / 複数件 / Execution Error
- 実PostgreSQL Evidence tool

実DB成功Evidenceはまだ未取得です。

#### Validation -> Authorization -> Command

次を区別し、不成立時にはCommandへ進みません。

- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `denied`
- Command Business Result

最終UI-less Scenarioではheader-detailのSemantic OrderLine resultをValidation inputへ渡し、Authenticationで得たsubjectをAuthorizationへ、browseで得た同じ`order_id`をCommandへ引き継ぎます。

### Command / Observation

関連: ADR-0023 / 完了済みGitHub Issue #266

Command Business ResultをObserved stateへ変換しません。

v1では次を採用済みです。

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation read result / Execution Error
```

Query / Contract Query / read adapter等の既存の明示read境界を別操作として利用します。

- Observation対象をCommand payload、HTTP status、Capability名から推測しない
- read resultと期待状態評価を分ける
- Observation Execution Errorを成功済みCommand結果へ逆流させない
- synthetic Evidenceと実外部Evidenceを区別する
- runtime parameter値やcredentialを通常Evidenceへ保存しない

Observation専用Application Pattern、Semantic Capability、Workflow Engine、Event Sourcing基盤は第一Referenceだけから先行導入しません。

#266は境界設計として完了しており、Gate Bの未設計blockerではありません。

### 最終UI-less Scenario

`internal/driver/final_gate_b_scenario_test.go`で次を一つのtest-local Scenarioへ構造統合しています。

```text
Authentication
  -> browse
  -> record Pattern resolution
  -> reference Customer
  -> header-detail OrderLine[]
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

browseでSemantic mapping済みの同じOrder rowから`customer_id` / `order_id`を後続境界へ引き継ぎます。

このScenarioはsynthetic adapterを含むため、実Oracle / 実PostgreSQL / 実外部Command Evidenceの代替にはしません。

### Gate Bを直接止めているEvidence

現在の直接残件は四系統です。

1. **#214 external `master_data`実物理 / read-only Evidence**
   - 実object / parameter / raw result identity
   - native realization source
   - explicit Semantic mapping
   - provenance
   - 実効的なread-only境界
   - 1件 / 0件 / Execution Error
2. **header-detail実PostgreSQL Evidence**
   - `tools/reference-postgres-header-detail-runtime`
   - 破棄可能な専用DBで2件 / 0件 / Execution Errorを確認
3. **#255 Reference golden正規再生成Evidence**
   - World / Plugin Lock / 必要ならDurable Run
   - 正規コマンドから再生成し、fingerprintや生成物を手編集しない
4. **最終UI-less Scenarioの通常Go runner Evidence**
   - `TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`
   - repository revisionを記録して実行する

具体的な実行手順と安全条件は`GATE_B_EVIDENCE_ACQUISITION.md`を正とします。

### GitHub Actionsの扱い

GitHub Actionsによる継続的な再検証は重要ですが、上記四系統のProduct Evidenceと同一ではありません。

Actions停止そのものをSemantic design gapへ読み替えず、逆に未実行をgreenとも扱いません。再開後は同じConformanceをregression検知として継続実行します。

## Gate B — UIなしで主要Scenarioを再現できる

現在の判定は**Not Ready**です。

Gate Bは画面がなくても次をEvidenceから説明できることを要求します。

- 同じApplication定義から主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- external assetの未観測事実を推測していない
- read-only / secret等の安全境界を保持できる
- Validation / Authorization不成立時にCommandへ進まない
- Business Result / Execution Error / Observationを区別できる
- synthetic Evidenceと実外部Evidenceを区別できる
- Query 0件とExecution Errorを区別できる
- 最新authoringからReference artifactを再現できる
- 第一Web TargetへWeb側の意味補完なしに同じbaselineを渡せる

最新判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

## Gate BからGate CへのHandover Baseline

関連: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / GitHub Issue #291

Gate BがReadyになった後、第一Web Targetへ渡す比較baselineを曖昧な「その時点の最新Driver」にはしません。

Gate B通過時のrevisionを一つのHandover Baselineとして固定し、少なくとも次へ追跡できるようにします。

- repository revision
- Reference Workspace identity
- Application: `order_entry`
- browse / record / reference / header-detail / command
- Data / Relation / Query / Datasource
- Canonical Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Observationに利用する明示read境界
- Source / Binding traceability
- Driver Scenario identity
- Gate B Evidence
- unresolved

Handover recordは第二の正本ではありません。authoritative source、Canonical ID、Gate B Evidenceへの参照を束ねます。

Web側で不足が見つかった場合は独自fieldやruleで隠さずApplication / Driverへ戻ります。Application / Driverの意味を変更した場合は、必要なGate B Evidenceを再確認し、新しいHandover Baselineを発行します。既存baselineを黙って「最新」へ読み替えません。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70  
Evidence実証: `ISSUE-000028` / GitHub Issue #270  
Gate B→C Handover: `ISSUE-000030` / GitHub Issue #291  
契約: [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md) / [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

### 着手条件

**Gate B Readyかつ第一Handover Baselineを固定した後**です。

Gate B前にframework、画面構造、Presentation IRを先に固定しません。

### 目的

Handover Baselineで固定したApplication / Driver上の意味を第一Web TargetへProjectionし、次を実例から分離します。

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

各Gate C Evidence runから、どのHandover Baselineと比較したかを辿れる必要があります。

## Gate C — 第一Targetの境界をEvidenceで説明できる

第一Web Targetが表示できただけでは通過しません。

[`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)に従い、次の三種類を別Evidenceとして扱います。

1. **Semantic Conformance**
2. **Interaction / Functional Evidence**
3. **Visual Fidelity Evidence**

Semantic Conformanceでは、Gate Bから固定したHandover Baselineと同じCanonical Application上の意味をWebが保持しているかを確認します。

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
- Handover Baseline
- Target comparison
- ADR
- readiness audit
- unresolvedを明示したDiagnostic / Evidence

Evidence artifactやHandover recordをCanonicalの第二正本にはしません。

実験結果が予想と異なる場合、その差を隠して計画へ合わせません。Product Valueに照らし、Requirement / Architecture / Planを修正します。

## まだ固定しないこと

- Application Patternの永久的な標準一覧
- 第一Web Targetのframework
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- Editor最終UI framework
- AI provider / Proposal最終schema
- Handover Baselineの汎用machine-readable schema
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
- Gate B通過時のDriver baselineを固定し、第一TargetのEvidenceからそのrevisionへ戻れる
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
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/design/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
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
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
