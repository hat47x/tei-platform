# TEI Milestones

## 方針

TEI（鼎／テイ）のMilestoneには期限を置きません。

Milestoneは「いつまでに終えるか」ではなく、**どのCapabilityとEvidenceが揃えば、その能力を成立したとみなせるか**を示します。

Milestone番号は、特にGate C後では厳密な直列依存を意味しません。Milestone 5〜7は同じGate C Evidenceを入力として並行して進められます。

個別のM0.xxは細かなarchitecture checkpointとして継続利用できますが、本書ではProduct全体の能力到達点を扱います。

Product Qualityは独立した番号付きMilestoneにはせず、各MilestoneのclaimをContext / Measure / Evidenceへ戻せる横断条件として扱います。

関連:

- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `docs/product-quality-architecture.md`
- `plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`

## v1 Reference Applicationとの対応

`order_entry`を使い、Milestone 1〜4を一続きに実証します。

```text
Milestone 1 / 2
Meaning / Application / Legacy
        ↓
Gate A — Ready
        ↓
Milestone 3
Driver
        ↓
Gate B — 3 Required Evidence / Not Ready
        ↓
Handover Baseline B / revision R
        ↓
Milestone 4
First Web Target / build T1
        ↓
Gate C — PAS-04 Evidence
        + Product Quality Evidence
        ↓
   ┌────┴──────────────┐
   ↓                   ↓
Milestone 6 / 7     Milestone 5
Studio / AI         Second Target
PAS-01 / PAS-06     PAS-04 / RQ-024 / 029 / 030
```

Handoverは番号付きGateではありません。Gate B Readyの意味baselineをGate Cへ固定して渡す契約です。

## Milestone 0 — 上位基線がある

### 到達条件

- Product Value / Vision / PrinciplesがRepository上で参照できる
- Product Requirementsが固定IDで参照できる
- Product Acceptance ScenarioからRequirement → Plan → Evidenceへ追跡できる
- Master Plan / Milestones / ADR / Plan Issueの役割が分離されている
- Product Quality claimをContext / Measure / Evidenceへ束縛する設計基線がある
- unresolvedを成功へ読み替えない

### 意味

今後の作業が「何を確かめるためか」「どの条件で品質を主張できるか」を判断できる地図がある状態です。

## Milestone 1 — Meaningを作り、追跡し、検査できる

### 到達条件

- authoring sourceからCanonical meaningへ追跡できる
- file pathとSemantic identityが分離されている
- native sourceをauthoritativeなまま接続できる
- Diagnosticからsourceへ戻れる
- Application PatternをApplication所有の意図として扱える
- file moveとlogical rename等の意味差を区別できる
- ambiguity / insufficient informationをsilent meaning driftへ変換しない

### 意味

「定義を読み込める」だけでなく、意味とsourceの関係を保持して変更・検査できる状態です。

v1の5 PatternをTEI全体の永久標準Catalogにはしません。

## Milestone 2 — 現実の既存業務を受け止められる

### 到達条件

- Application ↔ Databaseを実例で扱える
- Database Contract / native SQL / external assetを明示Bindingできる
- 既存Authentication / Authorization仕様を保持したまま接続できる
- Capability単位でstandard / existing implementationを交換できる
- external-observed assetの未観測事実を推測しない
- actual external Evidenceとsynthetic / fixture Evidenceを区別できる
- provider / external failureをsuccessへ丸めず、必要なdiagnosticへ戻れる

### 意味

理想化sampleだけでなく、実際の既存業務資産を段階移行できる状態です。

actual Oracle #214 / #371 / #376はこの能力とMilestone 8のProvider圧力試験へつながる重要Evidenceですが、Gate Bの第四Required laneにはしません。

## Milestone 3 — 設計をDriverで動かして確認できる

Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

### 到達条件

- Application PatternをDriverで解決できる
- browse / record / reference / header-detailをUIなしで扱える
- Validation / Authentication / Authorizationを実行できる
- Capability invocationのBusiness ResultとExecution Errorを区別できる
- Command acknowledgementとObservationを分けられる
- failure / degraded / unknownをsuccessへ丸めない
- same-source Reference Scenarioを再現できる
- generated artifact / real-provider / deterministic Evidenceを相互代替しない

### Gate B

Gate B Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃え、#387 final convergenceで確認します。

Gate B Ready後、その`R`をHandover Baselineへ固定します。

RQ-027〜RQ-030を理由にGate Bへ第四Required laneを追加しません。

## Milestone 4 — 一つの実用TargetでApplicationを通せる

Product Acceptance Scenario: PAS-04 primary

### 到達条件

- Handover Baseline `B / R`から第一Web TargetへProjectionできる
- Web build revision `T1`をbaseline `R`と別に追跡できる
- browse / record / reference / header-detail / commandを同じApplication meaningへ接続できる
- Validation / Authorization / Capability / Result / ObservationをTarget側で再定義しない
- standard ProjectionとCustom UIを同じ上位意味へ接続できる
- Web Adapterを除去してもCanonical Worldが変わらない
- Reference Scenarioを必要なinput modalityで操作でき、validation / errorから復旧できる
- applicable accessibility checkをVisual Fidelityと分けて確認できる
- performance / resource measurementをTarget / Host / Scenario条件と一緒に記録できる
- failure / degraded / unknownのTarget表現をsuccessと区別できる
- quality claimを`B / R / T1`とEvidenceへ戻せる

### Gate C

Gate CのCore Evidenceでは次を別々に扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

VLM / screenshot scoreだけでSemantic / Functionalを合格にしません。

Product Quality Evidenceはこれらを置き換えず、同じScenarioへ次を重ねます。

- Interaction / Operability
- applicable Accessibility
- Performance / Resource
- Failure / degraded / recovery
- Quality Envelope provenance

Gate CではTarget Capability / Execution / Trust / Web固有差に加え、quality差も第二Targetへ渡せることを確認します。

## Milestone 5 — 同じ意味を複数Application Targetへ展開できる

Product Acceptance Scenario: PAS-04 primary

### 到達条件

- 第二Application Targetを第一Targetとは別build identityで実証できる
- 同じHandover Baseline、または明示successor baselineを比較基準にできる
- Target Capability差を説明できる
- Runtime Binding差を説明できる
- Execution / Trust Profile差を説明できる
- Target固有実装をCanonicalへ逆流させない
- RQ-024のperformance measurementをTarget / Host条件付きで比較できる
- RQ-027のfailure / degraded表現をTarget間で比較できる
- RQ-029のoperability / accessibilityをTarget固有interactionとして説明できる
- RQ-030のQuality EnvelopeからT1 / T2の条件差へ戻れる

### 意味

Target-neutralが第一Web Targetだけからの仮説ではなく、性質の異なる第二Targetで圧力試験された状態です。

T1とT2へ同じ性能値やUI規則を強制することが目的ではありません。同じApplication meaningを満たすためにTargetごとに必要なquality conditionを説明できることが目的です。

Studio / Editorのdeveloper-facing Wails HostはApplication Target Evidenceへ数えません。

## Milestone 6 — Studioが安全なAuthoring環境として成立する

Product Acceptance Scenario: PAS-01 primary

### 到達条件

- StudioがCanonicalの第二正本にならない
- high-density read / inspectができる
- Select → Propose → Preview → Accept / Rejectの最小authoring loopを実証できる
- file-native / CLI / AIとShared Acceptance Boundaryを共有できる
- Preview / Rejectでauthoritative sourceを変更しない
- Accept後にsourceをStudioなしでreloadできる
- stale `baseModelRevision`を黙って適用しない
- change impact / incompatible / unresolvedを必要な粒度で表示できる
- failure / partial writeが起きた場合にauthority状態を説明できる
- high-densityでもkeyboard / focus / navigation / diagnostic locationを失わない
- Studio都合でmandatory authored Stable IDを導入しない

### 意味

厳格なmeaning / source境界を壊さず、日常的なdeveloper-facing authoringへ接続できる状態です。

最初のReferenceは`ISSUE-000046`で扱います。

## Milestone 7 — knowledge-first / AI支援が厳格さを軽いUXへ変える

Product Acceptance Scenario: PAS-06 primary / PAS-01 secondary

### 到達条件

- Human / AIが既存Semantic資産をread-only探索できる
- applicable / incompatible / unknownを区別できる
- search ranking / similarity / derived activationをSemantic Evidenceへ格上げしない
- AI Proposalとaccepted Decision / source changeを区別できる
- ProposalがCanonical / Source / Evidenceへtraceできる
- AI ProposalもShared Acceptance Boundaryを通る
- stale Proposalを黙って適用しない
- autonomyとauthorityを分ける
- Derived Knowledgeを削除してもCanonical Worldが変わらない
- candidate recall / irrelevant candidate / latency / resource等をbenchmark contextへ戻せる
- candidate qualityをsemantic correctnessへ読み替えない

### 意味

AI機能が追加されたというだけでなく、既存知識探索・変更候補・検証を通して利用者負担を減らせる状態です。

read-only探索は`ISSUE-000047`、AI Proposal / Acceptanceは`ISSUE-000027`で扱います。algorithm-neutral derived cognitionは必要な範囲で別research incrementとして評価します。

## Gate C後のMilestone 5〜7の関係

Milestone 5、6、7は厳密な直列ではありません。

```text
Gate C Ready
   ├─→ Milestone 5 第二Target pressure test
   ├─→ Milestone 6 Studio minimal loop
   └─→ Milestone 7 knowledge-first / AI Proposal
```

それぞれのEvidenceを相互に戻します。

- 第二Target差 → Studioが表示すべきTarget差 / AIが参照すべきEvidence
- Studio authoring Evidence → Target-neutral source / acceptance境界
- knowledge-first / AI Evidence → reusable meaning / traceability不足
- Quality Envelope差 → Requirement / threshold / failure handlingの見直し

一つのlaneの完成を他laneの開始条件にしません。

## Milestone 8 — Provider / Runtimeの交換可能性が実証される

### 到達条件

- PostgreSQL以外のDatabase Providerを小さい実例で接続できる
- Provider固有knowledgeがCoreへ漏れない
- actual external asset / Provider Evidenceを取得できる
- external Runtime / orchestratorをCapabilityとして接続できる
- Plugin / Bundle / trust boundaryを説明できる
- unknown / takeover / fencing / recoveryをProvider差を超えて扱える
- provider-specific failure / change-safety Evidenceをcommon meaningへ無理に畳まず比較できる

### 意味

「交換可能」という主張が一つのReference Providerだけに依存しない状態です。

## Milestone 9 — 実利用へ向けた運用境界が整う

### 到達条件

- production永続Store / HA境界が確立される
- credential / secret / Resource authorityが運用可能に分離される
- signed artifact / registry / supply-chain Evidenceを扱える
- compatibility / upgrade policyがある
- observability / audit / failure analysisに必要なEvidenceを取得できる
- operational telemetryから安全なlogical identityへ相関できる
- telemetry data modelがCanonical business meaningのauthorityにならない
- production quality claimが対象version / Host / Provider / known limit / Evidenceへ戻れる
- production adapterとConformanceを持つ

### 意味

architecture validationを越え、限定された実利用へ責任を持って進められる状態です。

OpenTelemetry、NIST SSDF等の外部標準は必要に応じてinteroperability / process checklistとして利用できますが、Evidenceなしに準拠claimへしません。

## Version Horizon

### v0 — Architecture / Product Validation

Product Valueと境界を実行可能Referenceで検証する期間です。

現在のM0.xx系列はここに含みます。

### v1 — Practical Developer Platform vertical slice

少なくともMilestone 0〜4を満たし、一つの現実的Applicationを設計・Driver・Database接続・第一Targetまで一続きに扱える状態を目安とします。

v1の第一Targetではfull production maturityを要求しませんが、少なくとも主要Scenarioについてoperability / applicable accessibility / performance context / failure behaviorをEvidenceから説明できる必要があります。

Studio / AIの完成、第二Target、production HAはv1 first-target成立の前提にはしません。

ただしGate C後にMilestone 5〜7を検証できるtrace / plan / acceptance境界を残します。

### v2 — Target / Authoring / Provider Expansion

Milestone 5〜8を中心に、複数Target、Studio、knowledge-first / AI、Provider交換可能性を広げます。

Milestone 5〜7は必要に応じて並行します。

### v3以降 — Operational Maturity

Milestone 9を中心に、production運用、互換性、供給網、組織規模での再利用を深めます。

## 先に固定しないもの

- 完全なTarget Capability taxonomy
- 第一Web Target framework
- Studio最終UI
- AI provider / vector database
- Proposal最終schema
- mandatory authored Stable ID / UUID
- Patch-only authoring model
- 第二Targetの種類
- 一律performance threshold
- 一律availability / reliability threshold
- Quality Envelopeの固定schema
- universal quality score
- OpenTelemetry必須dependency
- production HA一般化

Reference / Evidenceが必要性を示す前に固定しません。