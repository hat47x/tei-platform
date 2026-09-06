# TEI Milestones

## 方針

TEI（鼎／テイ）のMilestoneには期限を置きません。

Milestoneは「いつまでに終えるか」ではなく、**どのCapabilityとEvidenceが揃えば、その能力を成立したとみなせるか**を示します。

Milestone番号は、特にGate C後では厳密な直列依存を意味しません。Milestone 5〜7は同じGate C Evidenceを入力として並行して進められます。

個別のM0.xxは細かなarchitecture checkpointとして継続利用できますが、本書ではProduct全体の能力到達点を扱います。

関連:

- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`

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
        ↓
   ┌────┴──────────────┐
   ↓                   ↓
Milestone 6 / 7     Milestone 5
Studio / AI         Second Target
PAS-01 / PAS-06     PAS-04 / RQ-024
```

Handoverは番号付きGateではありません。Gate B Readyの意味baselineをGate Cへ固定して渡す契約です。

## Milestone 0 — 上位基線がある

### 到達条件

- Product Value / Vision / PrinciplesがRepository上で参照できる
- Product Requirementsが固定IDで参照できる
- Product Acceptance ScenarioからRequirement → Plan → Evidenceへ追跡できる
- Master Plan / Milestones / ADR / Plan Issueの役割が分離されている
- unresolvedを成功へ読み替えない

### 意味

今後の作業が「何を確かめるためか」を判断できる地図がある状態です。

## Milestone 1 — Meaningを作り、追跡し、検査できる

### 到達条件

- authoring sourceからCanonical meaningへ追跡できる
- file pathとSemantic identityが分離されている
- native sourceをauthoritativeなまま接続できる
- Diagnosticからsourceへ戻れる
- Application PatternをApplication所有の意図として扱える
- file moveとlogical rename等の意味差を区別できる

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
- same-source Reference Scenarioを再現できる
- generated artifact / real-provider / deterministic Evidenceを相互代替しない

### Gate B

Gate B Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃え、#387 final convergenceで確認します。

Gate B Ready後、その`R`をHandover Baselineへ固定します。

## Milestone 4 — 一つの実用TargetでApplicationを通せる

Product Acceptance Scenario: PAS-04 primary

### 到達条件

- Handover Baseline `B` / revision `R`から第一Web TargetへProjectionできる
- Web build revision `T1`をbaseline `R`と別に追跡できる
- browse / record / reference / header-detail / commandを同じApplication meaningへ接続できる
- Validation / Authorization / Capability / Result / ObservationをTarget側で再定義しない
- standard ProjectionとCustom UIを同じ上位意味へ接続できる
- Web Adapterを除去してもCanonical Worldが変わらない

### Gate C

Gate Cでは次を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

VLM / screenshot scoreだけでSemantic / Functionalを合格にしません。

Gate CではTarget Capability / Execution / Trust / Web固有差を第二Targetへ渡せることも確認します。

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

### 意味

Target-neutralが第一Web Targetだけからの仮説ではなく、性質の異なる第二Targetで圧力試験された状態です。

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
- Studio都合でmandatory authored Stable IDを導入しない

### 意味

厳格なmeaning / source境界を壊さず、日常的なdeveloper-facing authoringへ接続できる状態です。

最初のReferenceは`ISSUE-000046`で扱います。

## Milestone 7 — knowledge-first / AI支援が厳格さを軽いUXへ変える

Product Acceptance Scenario: PAS-06 primary / PAS-01 secondary

### 到達条件

- Human / AIが既存Semantic資産をread-only探索できる
- applicable / incompatible / unknownを区別できる
- search ranking / similarityをSemantic Evidenceへ格上げしない
- AI Proposalとaccepted Decision / source changeを区別できる
- ProposalがCanonical / Source / Evidenceへtraceできる
- AI ProposalもShared Acceptance Boundaryを通る
- stale Proposalを黙って適用しない
- autonomyとauthorityを分ける
- Derived Knowledgeを削除してもCanonical Worldが変わらない

### 意味

AI機能が追加されたというだけでなく、既存知識探索・変更候補・検証を通して利用者負担を減らせる状態です。

read-only探索は`ISSUE-000047`、AI Proposal / Acceptanceは`ISSUE-000027`で扱います。

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

一つのlaneの完成を他laneの開始条件にしません。

## Milestone 8 — Provider / Runtimeの交換可能性が実証される

### 到達条件

- PostgreSQL以外のDatabase Providerを小さい実例で接続できる
- Provider固有knowledgeがCoreへ漏れない
- actual external asset / Provider Evidenceを取得できる
- external Runtime / orchestratorをCapabilityとして接続できる
- Plugin / Bundle / trust boundaryを説明できる
- unknown / takeover / fencing / recoveryをProvider差を超えて扱える

### 意味

「交換可能」という主張が一つのReference Providerだけに依存しない状態です。

## Milestone 9 — 実利用へ向けた運用境界が整う

### 到達条件

- production永続Store / HA境界が確立される
- credential / secret / Resource authorityが運用可能に分離される
- signed artifact / registry / supply-chain Evidenceを扱える
- compatibility / upgrade policyがある
- observability / audit / failure analysisに必要なEvidenceを取得できる
- production adapterとConformanceを持つ

### 意味

architecture validationを越え、限定された実利用へ責任を持って進められる状態です。

## Version Horizon

### v0 — Architecture / Product Validation

Product Valueと境界を実行可能Referenceで検証する期間です。

現在のM0.xx系列はここに含みます。

### v1 — Practical Developer Platform vertical slice

少なくともMilestone 0〜4を満たし、一つの現実的Applicationを設計・Driver・Database接続・第一Targetまで一続きに扱える状態を目安とします。

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
- production HA一般化

Reference / Evidenceが必要性を示す前に固定しません。
