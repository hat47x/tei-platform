# TEI Upstream Acceptance Scenario Traceability

Status: current planning baseline

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/product-quality-architecture.md`
- `PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`

## 1. 目的

この文書は、TEIのProduct Value / Product Requirementsを、利用者レベルのProduct Acceptance Scenario、設計判断、Plan Issue、Reference、Evidence、Gate / Product claimへ接続する追跡基線です。

新しいCanonical schema、Gate、coverage率を追加する文書ではありません。

```text
Product Value / Vision / Principles
        ↓
Product Requirement (RQ-xxx)
        ↓
Product Acceptance Scenario (PAS-xx)
        ↓
ADR / Plan Issue / Design
        ↓
Reference Scenario / Fixture
        ↓
Evidence
        ↓
Gate / Product claim
```

Requirementだけから実装を直接導出せず、利用者やmaintainerにとって何が成立すれば要求の意味を確認できるかをScenarioで束ねます。

Scenarioが通ったことだけで関連Requirement全体を自動的に`verified`へ引き上げません。Evidence nature、source revision、Provider / Target / Host、quality context、未解決事項を分離して扱います。

## 2. 追跡原則

1. Product Requirementを実装都合へ言い換えない。
2. 一つのRequirementは複数PASへ関係してよい。
3. 一つのPASは複数Requirementを束ねてよい。
4. PASは利用者またはmaintainerが観測できる結果で記述する。
5. ADRはPASを満たすための設計判断でありRequirementそのものではない。
6. Issue close、test pass、CI green、visual score、quality scoreだけでPAS成立とは扱わない。
7. synthetic / deterministic / generated-artifact / real-provider / actual-external / Target等のEvidence natureを混同しない。
8. source-dependent Evidenceは取得したrevisionへ固定し、後続revisionへ黙ってcarry-forwardしない。
9. Target Evidenceではmeaning baseline revisionとTarget build revisionを分ける。
10. 不足情報は`unresolved`として残し、成功へ丸めない。
11. Gateを増やすより、既存Gateが何を証明するかを明確にする。
12. Human / Studio / CLI / AI等の入口ごとに第二のsemantic authorityを作らない。
13. performance / reliability / accessibility等のquality claimは、対象・条件・measure・Evidenceから切り離さない。
14. operational telemetryやexternal quality modelをCanonical business meaningの第二正本にしない。

## 3. PAS-01 — 意味を壊さず設計を変更・再読込できる

### 利用者価値

開発者は、file-native authoring、Studio、CLI / refactor、AI Proposal等の入口が異なっても、変更前後の意味差と影響を確認し、同じAcceptance原則を通して安全にauthoritative sourceを変更・再読込できます。

### 代表Scenario

```text
Canonical Source / Source Indexを読む
    ↓
qualified logical identityを選ぶ
    ↓
変更candidateを作る
    ↓
Preview Semantic Diff / Impact
    ↓
Accept / Reject
    ↓ Accept only
Shared Acceptance Boundary
    ↓
authoritative file-native sourceへ反映
    ↓
reload
    ↓
意図したCanonical meaning / identity / Conformanceを再構成
```

Preview / Rejectはauthoritative sourceを変更しません。

candidateは必要に応じて`baseModelRevision`を持ち、staleならreject / re-read / explicit rebase等へ進みます。最新版へ黙って適用しません。

### identity / change safety

file pathそのものをSemantic identityにはしません。

- file moveだけでqualified logical identityが変わらない場合: semantic no-opとして扱える
- logical rename / Canonical identity変更: Semantic changeとして扱う
- destructive / incompatible / unresolvedな変更: impact / precondition / recoveryを必要な粒度で明示する

すべての変更後に「同じIDである」ことを要求するのではなく、**変更intentに対してidentity変化とimpactが説明可能であること**を要求します。

Studio / AI都合でmandatory authored Stable ID / UUIDを導入しません。

### 主なRequirement

- RQ-001 業務上の意味を実装技術から独立して保持
- RQ-002 Authoring sourceとCanonical meaningの追跡
- RQ-003 暗黙の複数正本を作らない
- RQ-010 Editor
- RQ-011 複数Projection
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision history
- RQ-017 二重管理の最小化
- RQ-019 Runtime / Resource authority
- RQ-021 Reference / Conformance
- RQ-028 change impact / compatibility / destructiveness
- RQ-029 Studio interaction quality（secondary）
- RQ-030 quality claim provenance

### shared Acceptance Boundary

```text
file edit -------┐
Studio ----------|
CLI / refactor --|--> candidate --> Shared Acceptance Boundary --> authoritative source
AI Proposal -----|
Matrix ----------|
optional Patch --┘
```

対象に応じて次を組み合わせます。

```text
Parse / Decode
 -> Normalize
 -> Validate
 -> Architecture / Policy Check
 -> Semantic Diff
 -> Impact Analysis
 -> Acceptance Policy
 -> authoritative source update
 -> reload
 -> Conformance
```

一つの固定API / processを要求するのではなく、入口依存でsemantic validationやauthorityが変わらないことを要求します。

### 現在のPlan / Evidence位置

Gate C前にStudio / AIの詳細interfaceを固定しません。

Gate C Ready後:

- `ISSUE-000046` — Studio minimal Authoring Loop（PAS-01 primary）
- `ISSUE-000027` — AI Proposal / Shared Acceptance（PAS-01 primary / PAS-06 secondary）

StudioはProjection / Authoring Clientであり第二正本ではありません。

## 4. PAS-02 — Application上の意味をDriverで実行・検証できる

### 利用者価値

開発者は画面やWeb frameworkを作る前でも、Application meaningをDriverで実行し、Validation、Authorization、Business Result、Execution Error、Observationを区別して確認できます。

### 代表Scenario

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Validation
 -> Authorization
 -> Command
 -> Business Result
 -> separate explicit read
 -> Observation
```

Command acknowledgementだけでObserved state成立とは扱いません。

failure、dependency unavailable、degraded、unknown等もsuccessへ丸めず、意味・実行境界へ戻れるdiagnosticを保持します。

### 主なRequirement

- RQ-005 Application / Database N:M
- RQ-008 Capability単位の実装置換
- RQ-009 既存認証・認可Compatibility
- RQ-012 Driverによる設計検証
- RQ-014 Application Pattern再利用
- RQ-018 Command / Observation分離
- RQ-021 Reference / Conformance
- RQ-027 failure / degraded / unknown diagnosability
- RQ-028 change safety（secondary）
- RQ-030 quality claim provenance（secondary）

### 現在のEvidence位置

Gate AはReadyです。Driverの主要意味境界も成立しています。

Gate BはNot Readyで、理由は新しいDriver抽象不足ではなくactual execution Evidence未取得です。

Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃え、`ISSUE-000037` / #387でfinal convergenceを確認します。

RQ-027〜RQ-030を理由に第四Required laneを追加しません。

## 5. PAS-03 — Native / external assetを壊さず段階移行・置換できる

### 利用者価値

既存SQL、Database、認証、Plugin、外部Provider等をTEI独自形式へ全面移植せず、意味・契約・realization・外部観測を分離したまま段階的に利用できます。

### 代表Scenario

```text
Canonical Contract / Application intent
        ↓
explicit Binding / native realization
        ↓
Provider / external asset
        ↓
Command or Query execution
        ↓
Evidence / Observation
```

未観測のexternal physical factsを生成・推測しません。Provider固有知識はProvider / Adapter側に残し、Canonical Contractをprovider都合で変更しません。

change / migrationではplanning時観測をapply-time safetyへ無条件carry-forwardせず、必要なfresh observation / preconditionでfail closedします。

### 主なRequirement

- RQ-004 Native asset利用
- RQ-005 Application / Database N:M
- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-007 Provider固有知識の隔離
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可Compatibility
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-021 Reference / Conformance
- RQ-027 provider failure diagnosability
- RQ-028 change / migration safety
- RQ-030 context-bound quality claim（secondary）

### 現在のEvidence位置

Gate Bの#308 real PostgreSQL fixtureはPAS-03の一部を圧力試験しますが、actual external Oracle Evidenceの代替ではありません。

actual Oracle #214 / #371 / #376はGate B第四Required laneではなく、actual external asset / legacy compatibility / second-provider Evidenceとして継続します。

Gate B Ready時に未完了ならHandover Baselineへ`unresolved`として明示し、第一Target側でphysical facts / SQL / read-only方式等を補完しません。

## 6. PAS-04 — 同じApplication meaningを複数TargetへProjectionできる

### 利用者価値

同じCanonical ApplicationをWeb、Desktop、Mobile、API、MCP等へ接続しても、Target固有UI / transport / Host事情から業務意味が逆定義されません。

### baselineとTarget buildを分ける

Gate B Ready candidate revisionを`R`、Handover Baselineを`B`、Target実装revisionを`T`とします。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
        ↓ meaning baseline
Target build T
        ↓
Target execution / Evidence
```

通常`T != R`です。

Target buildでbaseline subjectのmeaningを変更する必要が生じた場合、Target固有field / ruleで埋めません。authoritative sourceへ戻り、必要なDriver / Gate B Evidenceを再確認し、successor Handover Baseline `B' / R'`を発行します。

旧baseline Evidenceを新baselineへ黙ってcarry-forwardしません。

### Target評価

少なくとも次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. Interaction / Operability / applicable Accessibility Evidence
5. Performance / Resource Evidence
6. Failure / degraded / recovery Evidence

4〜6を既存1〜3へ置き換える一つの総合scoreにはしません。

### 主なRequirement

- RQ-001 業務意味の実装技術非依存
- RQ-010 EditorとのProjection境界
- RQ-011 複数Projection
- RQ-013 Target-neutral Presentation / Interaction
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance
- RQ-024 Target別性能要求
- RQ-027 failure / degraded / unknown diagnosability
- RQ-029 Target-conditioned operability / accessibility
- RQ-030 Context / Measure / Evidence-bound quality claim

### 現在のPlan / Evidence位置

Gate B Ready + Handover固定後:

- `ISSUE-000007` — First Web Target Projection
- `ISSUE-000028` — Gate C Conformance Evidence
- `GATE_C_EVIDENCE_CONTRACT.md`
- `PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md` — quality Evidence progression

Gate CではすべてのrunからHandover `B / R`とWeb build `T1`へ追跡します。

Web Targetのaccessibilityでは適用可能なWCAG 2.2 criteriaをexternal testable referenceとして利用できますが、actual Evidenceなしに準拠claimを行いません。

Gate C Ready後:

- `ISSUE-000026` — second Target pressure test

第一Targetだけで完全なPresentation IR / Target Capability taxonomy / universal quality thresholdを固定しません。

Studio HostをPAS-04のTarget成功Evidenceへ数えません。

## 7. PAS-05 — 外部状態が不確かな場合でも安全に実行・再開できる

### 利用者価値

外部Runtime / Resourceの状態が不明な場合に、「成功したはず」と推測したり、副作用を無条件にreplayしたりせず、Observation、ownership、fencing、durable handle等を使って安全に判断できます。

### 主なRequirement

- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-020 再実行 / takeover / unknown
- RQ-021 Reference / Conformance
- RQ-027 failure / degraded / unknown diagnosability
- RQ-028 change / recovery safety（secondary）
- RQ-030 quality claim provenance（secondary）

### Evidence位置

Deployment / Plugin / Resource lifecycle等のReferenceで継続します。

PAS-05を現行Gate Bの新しいblockerには追加しません。現在Gate Bに必要な範囲は既存3 Required Evidenceで確認し、より広いrecovery能力は対応Milestone / Referenceで別Evidenceとして扱います。

## 8. PAS-06 — 既存の意味知識を探索してから新規実装へ進める

### 利用者価値

開発者またはAIは毎回ゼロから実装せず、既存Data、Relation、Validation、Capability、Application Pattern、利用例、Evidenceを探索し、適用可能性と不足条件を確認してから変更・構成候補へ進めます。

### 代表Scenario

```text
purpose / intent
   ↓
read-only Semantic exploration
   ↓
identity / dependency / source / evidenceを確認
   ↓
applicable / incompatible / unknown
   ↓
explicit-reference candidate
   ↓
Shared Acceptance Boundary
```

検索順位、文字列類似度、embedding similarity、derived cognition activation、AI confidenceだけをSemantic Evidenceにしません。

AI Knowledge、index、cache、過去trajectoryはderivedでありCanonicalの第二正本ではありません。

### 主なRequirement

- RQ-002 Source traceability
- RQ-003 単一権威
- RQ-014 Application Pattern再利用
- RQ-015 knowledge-first AI
- RQ-016 AI suggestion / history
- RQ-017 二重管理最小化
- RQ-021 Reference / Conformance
- RQ-023 一人でも維持可能な構造
- RQ-027 derived mechanism failureのdiagnosability（secondary）
- RQ-030 benchmark / resource claim provenance

### 現在のPlan / Evidence位置

Gate C Ready後:

- `ISSUE-000047` — Human / AI共通read-only Semantic exploration（PAS-06 primary）
- `ISSUE-000027` — exploration結果からAI Proposal / Acceptanceへ接続
- `ISSUE-000051` — algorithm-neutral derived cognition evaluation

最初から巨大Knowledge Graph、vector database、AI専用Knowledge Storeを作りません。

candidate qualityとsemantic applicability / accepted decisionを分離して評価します。

## 9. PAS-07 — 手作業を失わずData / Function / Businessを相互検証して設計修正できる

### 利用者価値

開発者は、業務をApplication内部の実行手順へ還元せず、人・外部作業を含むBusiness Procedureを確認しながらData / Function-Interaction / Businessの三視点から矛盾・不足・過剰な仮定を見つけ、Projectionだけを補修するのではなくauthoritative sourceへ戻って設計を修正できます。

### 代表Scenario

```text
accepted Data / Relation / Application meaningを読む
        ↓
Business固有meaningをaccepted authoring sourceとして保持する
        ↓
manual / external Actionをfake Functionへ変換しない
        ↓
existing Application / Semantic identityをexplicitに参照する
        ↓
derived cross-view Projectionで相互検証する
        ↓
contradiction / omission / unknown / impactを発見する
        ↓
correction candidateをauthoritative source側へ返す
        ↓
Shared Acceptance Boundary
        ↓
reload / regenerate / reviewする
```

### 壊してはいけない境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Validation PASS != whole-business approval
Application Business Result != external-world Observation
Projection != second source of truth
```

Application Patternの並びやPAS-02のDriver validation sequenceを、accepted Business Procedureなしに利用者の業務手順へ読み替えません。

### 主なRequirement

- RQ-025 業務上の行為・手順をApplication Functionから独立して設計
- RQ-026 Data / Function-Interaction / Business Procedureの相互検証
- RQ-021 Reference / Conformance
- RQ-022 設計検証段階と完成製品の区別
- RQ-028 cross-view impact / change safety（secondary）
- RQ-030 Evidence-bound quality claim（secondary）

RQ-025はPAS-07 primary / PAS-02・PAS-04 secondaryです。

RQ-026はPAS-07 primary / PAS-01・PAS-02・PAS-04 secondaryです。

### 現在のPlan / Evidence位置

Reference / Evidence実装はLane C `ISSUE-000042` / PR #482系のmain統合実装が所有します。Lane BはReference fixture / checkerを複製せず、EvidenceをPAS-07 claimへ対応付けます。

現時点では次を分離します。

```text
Lane C fixture / checker / pressure implementation exists
        !=
actual Repository runner Evidence accepted
        !=
PAS-07 whole-product claim accepted
```

Business-side Representation EvidenceとPAS-02 Runtime Evidenceも分けます。

```text
Lane C Business Procedure Reference
  -> manual / external Action
  -> Business judgment / Validation separation
  -> actor / authority separation
  -> Command / Observation separation
  -> cross-view reference / diagnostic

PAS-02 Driver Evidence
  -> Validation execution
  -> Authentication / Authorization decision
  -> Capability execution
  -> Business Result
  -> separate explicit read / Observation
```

片方をもう片方の代替にしません。

Reference-local stable Procedure keyはreview / handoff / stale detection / file move分離に価値がありますが、それだけでCanonical `Procedure` / `Step`を採用しません。

### Gateへの影響

PAS-07は新Gateではありません。

- Gate B Required Evidence 3 laneへ追加しない
- Gate C First Web Target Required Evidenceへ自動追加しない
- Lane C actual runner greenはPAS-07 Evidence statusを更新するが、それだけでPAS-07全体を`accepted`へ上げない
- end-to-end correction / acceptance / reloadはPAS-01 Shared Acceptance Evidenceとcross-checkする

詳細なE7 mappingは `RQ025_RQ026_ACCEPTANCE_TRACE.md` をsupporting Evidence contractとして参照します。

## 10. Cross-cutting Requirements

### RQ-021 — Reference / Conformance

すべてのPASに横断します。

「設計上できる」ではなく、小さいReference、fixture、Conformance、Evidenceから境界を再現できることを要求します。

### RQ-022 — 完成製品と設計検証段階を区別

すべてのPASに横断します。

architecture probe、local fixture、synthetic adapter、beta framework、actual provider run、production claimを同じ`done`へ畳みません。

### RQ-023 — 一人でも維持可能な開発構造

主にPAS-01 / PAS-06へ関係しますが全体の作業分割へ横断します。

同じRepository内でRequirement → PAS → Decision → Reference → Evidenceを追えることを優先し、不要な第二正本 / 巨大subsystemを増やしません。

### RQ-024 — Target別性能

PAS-04で具体化します。

第一Target `T1`と第二Target `T2`の実測を、Handover baseline、Target / Host、Runtime / Provider条件と一緒に比較します。

根拠のない一律thresholdを第一Targetだけから固定しません。

### RQ-027 — failure / degraded / unknown diagnosability

PAS-02 / PAS-03 / PAS-05を中心に、全実行surfaceへ横断します。

Business Result、Execution Error、Observation、Developer Diagnostic、Operational Telemetryを同一視しません。

### RQ-028 — change impact / compatibility / destructiveness

PAS-01 / PAS-03をprimaryな実証面とし、PAS-07 cross-view impactへも接続します。

planning時の観測をapply時へ無条件carry-forwardせず、必要なfresh observation / precondition / stale detectionを要求します。

### RQ-029 — Target-conditioned operability / accessibility

PAS-04をprimary、PAS-01 Studioをsecondaryとします。

high-densityとoperability / accessibilityを対立させず、Target固有input modalityでReference Scenarioを完遂できることをEvidence化します。

### RQ-030 — Context / Measure / Evidence-bound quality claim

すべてのPASに横断します。

quality claimを概念上次へ戻します。

```text
subject
+ revision / build
+ Target / Host / Provider / Runtime context
+ Scenario / precondition
+ property
+ measure / probe
+ acceptance rationale
+ Evidence
+ status
```

Quality Envelopeはderivedな監査単位でありCanonical authorityではありません。

## 11. Requirement → Primary PAS mapping

| Requirement | Primary PAS | Secondary / cross-cutting |
| --- | --- | --- |
| RQ-001 | PAS-01 | PAS-04 |
| RQ-002 | PAS-01 | PAS-06 |
| RQ-003 | PAS-01 | PAS-06 |
| RQ-004 | PAS-03 | - |
| RQ-005 | PAS-02 | PAS-03 |
| RQ-006 | PAS-03 | PAS-05 |
| RQ-007 | PAS-03 | PAS-04 |
| RQ-008 | PAS-02 | PAS-03 |
| RQ-009 | PAS-02 | PAS-03 |
| RQ-010 | PAS-01 | PAS-04 |
| RQ-011 | PAS-04 | PAS-01 |
| RQ-012 | PAS-02 | PAS-04 |
| RQ-013 | PAS-04 | - |
| RQ-014 | PAS-02 | PAS-06 |
| RQ-015 | PAS-06 | PAS-01 |
| RQ-016 | PAS-06 | PAS-01 |
| RQ-017 | PAS-01 | PAS-04 / PAS-06 |
| RQ-018 | PAS-02 | PAS-03 / PAS-05 |
| RQ-019 | PAS-05 | PAS-01 / PAS-03 |
| RQ-020 | PAS-05 | - |
| RQ-021 | cross-cutting | all PAS |
| RQ-022 | cross-cutting | all PAS |
| RQ-023 | cross-cutting | PAS-01 / PAS-06 |
| RQ-024 | PAS-04 | - |
| RQ-025 | PAS-07 | PAS-02 / PAS-04 |
| RQ-026 | PAS-07 | PAS-01 / PAS-02 / PAS-04 |
| RQ-027 | cross-cutting | PAS-02 / PAS-03 / PAS-05 / PAS-04 |
| RQ-028 | PAS-01 / PAS-03 | PAS-07 / PAS-05 |
| RQ-029 | PAS-04 | PAS-01 |
| RQ-030 | cross-cutting | all PAS |

Primaryは最初に具体化する代表PASであり、coverage scoreではありません。

## 12. 現行Gate / Stageとの関係

```text
Gate A
  -> PAS-02のApplication meaningをDriverへ渡せることを確認

Gate B
  -> PAS-02中心 + PAS-03のreal-provider境界を含む3 Required Evidenceをsame-Rへ収束
  -> RQ-027〜030を理由にlaneを追加しない

Gate B → C Handover
  -> PAS-02 / PAS-03で確認したbaseline revision RとunresolvedをBとして固定

Gate C
  -> PAS-04をFirst Web Target build T1で実証
  -> baseline RとTarget build T1を分離
  -> Semantic / Functional / Visualを維持しつつquality contextを観測

Gate C後 — 並行可能
  -> PAS-01: ISSUE-000046 Studio minimal authoring loop
  -> PAS-06: ISSUE-000047 knowledge-first exploration
  -> PAS-01 / PAS-06: ISSUE-000027 AI Proposal / Acceptance
  -> PAS-04: ISSUE-000026 second Target pressure test

Gate非依存のparallel Reference pressure
  -> PAS-07: Business Procedure cross-view Reference
  -> Gate B / Gate C Required Evidenceへ追加しない
```

PAS-05はruntime / deployment workstreamへ接続します。現行Gate B必須scopeを本書から拡張しません。

## 13. 新しいADR / Plan Issueへ要求する最小Trace

新しい上流設計・機能提案では、必要に応じて次を明示します。

- 関連Product Requirement
- Primary PAS
- Scenarioで失敗すると利用者 / maintainerに何が見えるか
- 保存するsemantic boundary
- Provider / Target / Hostへ残す差
- baseline / source / build identity
- Reference / fixture
- 必要なEvidence nature
- quality claimがある場合のContext / Measure / acceptance rationale
- 既存Gateへ影響するか否か
- unresolved
- reopen condition

実装が先に存在する場合でも、この欄を埋めるためだけに架空Requirementを作りません。

## 14. 現行Authoring基線

PAS-01 / PAS-06では次を現行前提とします。

- authoritative sourceはfile-nativeで保持可能
- qualified logical key / Canonical identityをfile pathと分離
- Studio / CLI / AI等はcandidate入口
- Preview / Rejectはnon-mutating
- AcceptはShared Acceptance Boundaryへ収束
- accepted sourceは入口toolなしでreload可能
- stale base revisionは明示処理
- derived index / AI Knowledge / trajectoryは第二正本ではない

PAS-07のBusiness Procedure pressureでも、Business固有meaningをfile-native authoring sourceへ保持し、Pattern-owned Data / Relation / Query / Capability / DatasourceやMatrix / diagnostic等のderived Projectionと分離します。

現時点でReference-local stable Procedure keyの有用性は確認していますが、accepted Business sourceをRQ-002上どのCanonical trace targetへ接続するかは`unresolved`です。この一点からfull Canonical Procedure modelやRuntime Workflowを先に固定しません。

次を現行前提にはしません。

- Semantic Patchだけを唯一のauthoring経路にすること
- Base Semantic Digestを全candidateの必須基準にすること
- mandatory authored Stable ID / UUID
- Studio独自semantic authority
- AI独自Canonical Knowledge Store
- Canonical `Procedure` / `Step`の先行採用
- Business ProcedureからRuntime Workflow Orchestrationへの自動変換
- Quality Envelopeの固定Canonical schema
- 一つのuniversal quality score

optional Patchを利用する場合も、file-native / Studio / CLI / AIと同じAcceptance原則へ接続します。

## 15. 非目標

- Product Requirementsの置換
- Canonical object `ProductAcceptanceScenario`の追加
- runtime Scenario Registryの実装
- Requirement coverage率 / 完成率の算出
- Gate B / Gate Cのscopeを本書だけで変更すること
- 第一Targetだけから完全なPresentation IRを固定すること
- Gate C前にStudio / AIの詳細interfaceを固定すること
- PAS-07を理由にCanonical `Procedure` / `Step`、Workflow Engine、新Gateを自動導入すること
- RQ-027〜030を理由にquality専用Gateや万能scoreを導入すること
- external standardへの準拠をTrace表だけで宣言すること