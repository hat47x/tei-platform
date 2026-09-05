# TEI Upstream Acceptance Scenario Traceability

Status: current planning baseline

## 1. 目的

この文書は、TEIのProduct Value / Product Requirementsを、利用者レベルのProduct Acceptance Scenario、設計判断、Plan Issue、Reference、Evidence、Gate / Product claimへ接続する追跡基線です。

新しいProduct Requirement、Canonical schema、Gate、coverage率を追加する文書ではありません。

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

Scenarioが通ったことだけで関連Requirement全体を自動的に`verified`へ引き上げません。Evidence nature、source revision、Provider / Target / Host、未解決事項を分離して扱います。

## 2. 追跡原則

1. Product Requirementを実装都合へ言い換えない。
2. 一つのRequirementは複数PASへ関係してよい。
3. 一つのPASは複数Requirementを束ねてよい。
4. PASは利用者またはmaintainerが観測できる結果で記述する。
5. ADRはPASを満たすための設計判断でありRequirementそのものではない。
6. Issue close、test pass、CI green、visual scoreだけでPAS成立とは扱わない。
7. synthetic / deterministic / generated-artifact / real-provider / actual-external等のEvidence natureを混同しない。
8. source-dependent Evidenceは取得したrevisionへ固定し、後続revisionへ黙ってcarry-forwardしない。
9. Target Evidenceではmeaning baseline revisionとTarget build revisionを分ける。
10. 不足情報は`unresolved`として残し、成功へ丸めない。
11. Gateを増やすより、既存Gateが何を証明するかを明確にする。
12. Human / Studio / CLI / AI等の入口ごとに第二のsemantic authorityを作らない。

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

### identityの注意

file pathそのものをSemantic identityにはしません。

- file moveだけでqualified logical identityが変わらない場合: semantic no-opとして扱える
- logical rename / Canonical identity変更: Semantic changeとして扱う

したがって、すべての変更後に「同じIDである」ことを要求するのではなく、**変更intentに対してidentity変化が説明可能であること**を要求します。

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
- ADR-0022 — AI ProposalとCanonical受入境界の分離

StudioはProjection / Authoring Clientであり第二正本ではありません。

Studioのdeveloper-facing Wails HostをGenerated Wails Desktop Application TargetのEvidenceへ読み替えません。

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

### 主なRequirement

- RQ-005 Application / Database N:M
- RQ-008 Capability単位の実装置換
- RQ-009 既存認証・認可Compatibility
- RQ-012 Driverによる設計検証
- RQ-014 Application Pattern再利用
- RQ-018 Command / Observation分離
- RQ-021 Reference / Conformance

### 現在のEvidence位置

Gate AはReadyです。Driverの主要意味境界も成立しています。

Gate BはNot Readyで、理由は新しいDriver抽象不足ではなくactual execution Evidence未取得です。

Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃え、`ISSUE-000037` / #387でfinal convergenceを確認します。

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

Target buildでbaseline subjectのmeaningを変更する必要が生じた場合、Target固有field / ruleで埋めません。authoritative sourceへ戻り、必要なDriver / Gate B Evidenceを再確認し、successor Handover Baseline `B'` / `R'`を発行します。

旧baseline Evidenceを新baselineへ黙ってcarry-forwardしません。

### Target評価

少なくとも次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

### 主なRequirement

- RQ-001 業務意味の実装技術非依存
- RQ-010 EditorとのProjection境界
- RQ-011 複数Projection
- RQ-013 Target-neutral Presentation / Interaction
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance
- RQ-024 Target別性能要求

### 現在のPlan / Evidence位置

Gate B Ready + Handover固定後:

- `ISSUE-000007` — First Web Target Projection
- `ISSUE-000028` — Gate C Conformance Evidence
- `GATE_C_EVIDENCE_CONTRACT.md`

Gate CではすべてのrunからHandover `B` / baseline `R`とWeb build `T1`へ追跡します。

Gate C Ready後:

- `ISSUE-000026` — second Target pressure test

第一Targetだけで完全なPresentation IR / Target Capability taxonomyを固定しません。

StudioがWailsを使うことと、Generated Wails Desktop Application Targetは別です。Studio HostをPAS-04のTarget成功Evidenceへ数えません。

## 7. PAS-05 — 外部状態が不確かな場合でも安全に実行・再開できる

### 利用者価値

外部Runtime / Resourceの状態が不明な場合に、「成功したはず」と推測したり、副作用を無条件にreplayしたりせず、Observation、ownership、fencing、durable handle等を使って安全に判断できます。

### 主なRequirement

- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-020 再実行 / takeover / unknown
- RQ-021 Reference / Conformance

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

検索順位、文字列類似度、embedding similarity、AI confidenceだけをSemantic Evidenceにしません。

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

### 現在のPlan / Evidence位置

Gate C Ready後:

- `ISSUE-000047` — Human / AI共通read-only Semantic exploration（PAS-06 primary）
- `ISSUE-000027` — exploration結果からAI Proposal / Acceptanceへ接続

最初から巨大Knowledge Graph、vector database、AI専用Knowledge Storeを作りません。

## 9. Cross-cutting Requirements

### RQ-021 — Reference / Conformance

すべてのPASに横断します。

「設計上できる」ではなく、小さいReference、fixture、Conformance、Evidenceから境界を再現できることを要求します。

### RQ-022 — 完成製品と設計検証段階を区別

すべてのPASに横断します。

architecture probe、local fixture、synthetic adapter、beta framework、actual provider run、production claimを同じ`done`へ畳みません。

### RQ-023 — 一人でも維持可能な開発構造

主にPAS-01 / PAS-06へ関係しますが全体の作業分割へ横断します。

同じRepository内でRequirement → PAS → Decision → Reference → Evidenceを追えることを優先し、不要な第二正本 /巨大subsystemを増やしません。

### RQ-024 — Target別性能

PAS-04で具体化します。

第一Target `T1`と第二Target `T2`の実測を、Handover baseline、Target / Host、Runtime / Provider条件と一緒に比較します。

根拠のない一律thresholdを第一Targetだけから固定しません。

## 10. Requirement → Primary PAS mapping

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

Primaryは最初に具体化する代表PASであり、coverage scoreではありません。

## 11. 現行Gate / Stageとの関係

```text
Gate A
  -> PAS-02のApplication meaningをDriverへ渡せることを確認

Gate B
  -> PAS-02中心 + PAS-03のreal-provider境界を含む3 Required Evidenceをsame-Rへ収束

Gate B → C Handover
  -> PAS-02 / PAS-03で確認したbaseline revision RとunresolvedをBとして固定

Gate C
  -> PAS-04をFirst Web Target build T1で実証
  -> baseline RとTarget build T1を分離

Gate C後 — 並行可能
  -> PAS-01: ISSUE-000046 Studio minimal authoring loop
  -> PAS-06: ISSUE-000047 knowledge-first exploration
  -> PAS-01 / PAS-06: ISSUE-000027 AI Proposal / Acceptance
  -> PAS-04: ISSUE-000026 second Target pressure test
```

PAS-05はruntime / deployment workstreamへ接続します。現行Gate B必須scopeを本書から拡張しません。

## 12. 新しいADR / Plan Issueへ要求する最小Trace

新しい上流設計・機能提案では、必要に応じて次を明示します。

- 関連Product Requirement
- Primary PAS
- Scenarioで失敗すると利用者 / maintainerに何が見えるか
- 保存するsemantic boundary
- Provider / Target / Hostへ残す差
- baseline / source / build identity
- Reference / fixture
- 必要なEvidence nature
- 既存Gateへ影響するか否か
- unresolved
- reopen condition

実装が先に存在する場合でも、この欄を埋めるためだけに架空Requirementを作りません。

## 13. 現行Authoring基線

PAS-01 / PAS-06では次を現行前提とします。

- authoritative sourceはfile-nativeで保持可能
- qualified logical key / Canonical identityをfile pathと分離
- Studio / CLI / AI等はcandidate入口
- Preview / Rejectはnon-mutating
- AcceptはShared Acceptance Boundaryへ収束
- accepted sourceは入口toolなしでreload可能
- stale base revisionは明示処理
- derived index / AI Knowledge / trajectoryは第二正本ではない

次を現行前提にはしません。

- Semantic Patchだけを唯一のauthoring経路にすること
- Base Semantic Digestを全candidateの必須基準にすること
- mandatory authored Stable ID / UUID
- Studio独自semantic authority
- AI独自Canonical Knowledge Store

optional Patchを利用する場合も、file-native / Studio / CLI / AIと同じAcceptance原則へ接続します。

## 14. 非目標

- Product Requirementsの置換
- Canonical object `ProductAcceptanceScenario`の追加
- runtime Scenario Registryの実装
- Requirement coverage率 / 完成率の算出
- Gate B / Gate Cのscopeを本書だけで変更すること
- 第一Targetだけから完全なPresentation IRを固定すること
- Gate C前にStudio / AIの詳細interfaceを固定すること
