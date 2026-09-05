# TEI Upstream Acceptance Scenario Traceability

Status: proposed planning baseline

## 1. 目的

この文書は、TEIのProduct Value / Product Requirementsを、実際の利用者レベルScenario、設計判断、Plan Issue、Reference、Evidence Gateへ接続するための追跡基線です。

新しいProduct Requirement、Canonical schema、Gate、完成率指標を追加する文書ではありません。

現在の上流構造は次のように扱います。

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

Requirementだけから実装を直接導出せず、利用者にとって何が成立すれば要求の意味が確認できるかをScenarioで束ねます。

一方、Scenarioが通ったことだけで関連Requirement全体を自動的に`verified`へ引き上げません。Evidenceの性質、対象revision、Provider / Target / Host条件、未解決事項を分離して扱います。

## 2. 追跡の原則

1. Product Requirementを実装都合へ言い換えない。
2. 一つのRequirementは複数Scenarioへ関係してよい。
3. 一つのScenarioは複数Requirementを束ねてよい。
4. Scenarioは利用者またはmaintainerが観測できる結果で記述する。
5. ADRはScenarioを満たすための設計判断であり、Requirementそのものではない。
6. Issue close、test pass、CI green、visual scoreだけでProduct Scenario成立とは扱わない。
7. synthetic / deterministic / generated-artifact / real-provider / actual-external等のEvidence natureを混同しない。
8. Evidenceは取得したrepository revisionへ固定し、後続revisionへ黙ってcarry-forwardしない。
9. 不足情報は`unresolved`として残し、成功へ丸めない。
10. Gateを増やすより、既存Gateが何を証明しているかを明確にする。

## 3. PAS-01 — 意味を壊さず設計を変更・再読込できる

### 利用者価値

開発者は、file-native authoring、Studio、CLI/refactor、AI Proposal等の入口が異なっても、同じCanonical meaningを安全に変更し、変更前後の意味差と影響を確認できます。

### 代表Scenario

```text
Canonical Source / Source Indexを読む
    ↓
変更候補を作る
    ↓
Preview Semantic Diff / Impact
    ↓
shared Acceptance Boundary
    ↓
authoritative sourceへ反映
    ↓
reload
    ↓
同じCanonical identity / Conformanceを再構成
```

Studio / AI等の候補は`baseModelRevision`等の読めるrevisionに基づき、staleならreject / explicit rebaseへ進みます。Preview / Rejectはaccepted sourceを変更しません。

### 主なRequirement

- RQ-001 業務上の意味を実装技術から独立して保持
- RQ-002 Authoring sourceとCanonical meaningの追跡
- RQ-003 暗黙の複数正本を作らない
- RQ-010 Editor
- RQ-011 複数Projection
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision history
- RQ-017 二重管理の最小化
- RQ-019 Runtime / Resource authorityの明示境界
- RQ-021 Reference / Conformance

### 既存設計との接続

候補入口を問わず、受理は概念上次へ収束します。

```text
Parse
 -> Normalize
 -> Validate
 -> Architecture Lint
 -> Semantic Diff
 -> Impact Analysis
 -> Acceptance Policy
 -> authoritative source
 -> reload
```

StudioはProjection / Authoring Clientであり、Canonicalの第二権威ではありません。Wails等のHostもこの境界を迂回しません。

### 現在のEvidence位置

Gate C前にStudio / AIの詳細interfaceを固定しません。Gate C後に、第一Web Targetまでで実証したApplication / Presentation境界を用いてStudio / AI Proposalを具体化します。

ただし、shared Acceptance Boundaryそのものの設計原則やfile-native authoringとの同一性は、Host UI実装より先に検証可能です。

## 4. PAS-02 — Application上の意味をDriverで実行・検証できる

### 利用者価値

開発者は、画面やWeb frameworkを作る前でも、Application上の業務意味をDriverで実行し、Validation、Authorization、Business Result、Execution Error、Observationを区別して確認できます。

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

Gate AはReadyです。Application Driverの主要意味境界も成立しています。

Gate Bでは、新しいDriver抽象を増やすのではなく、現在のcandidate revisionについてsource-dependent Evidenceを収束させます。

Gate B Ready判定へ必須の現行三laneは次です。

- deterministic Repository Evidence
- generated artifact reproducibility Evidence
- real PostgreSQL Provider + ephemeral fixture Evidence

この三laneを同じclean candidate revisionへ揃える既存条件を変更しません。

## 5. PAS-03 — Native / external assetを壊さず段階移行・置換できる

### 利用者価値

既存SQL、既存Database、既存認証、Plugin、外部Provider等をTEI独自形式へ全面移植せず、意味・契約・実現・外部観測を分離したまま段階的に利用できます。

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

未観測のexternal physical factsをTEI側から生成・推測しません。Provider固有知識はProvider / Adapter側に残し、Canonical Contractをprovider都合で変更しません。

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

Gate Bのreal PostgreSQL fixture EvidenceはこのScenarioの一部を圧力試験しますが、actual external Oracle Evidenceの代替ではありません。

actual OracleはGate B必須blockerではないまま、既存外部資産と第二Database Providerの継続Evidenceとして扱います。

## 6. PAS-04 — 同じApplication meaningを複数TargetへProjectionできる

### 利用者価値

同じCanonical ApplicationをWeb、Desktop、Mobile、API、MCP等へ接続しても、Target固有のUI / transport / Host事情から業務意味が逆定義されません。

### 代表Scenario

```text
Gate B Handover Baseline
        ↓
Target-neutral candidate meaning
        ↓
Target Adapter / Host
        ↓
Interaction
        ↓
Business Result / Execution Error / Observation
```

Target評価では少なくとも次を分離します。

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

### 現在のEvidence位置

Gate B Ready後、同じrevisionをHandover Baselineとして固定して第一Web Targetへ進みます。

Gate CではSemantic / Functional / Visual Evidenceを分離します。第一Targetだけで完全なPresentation IRを固定せず、第二TargetでTarget CapabilityとExecution / Trust Profileの境界を圧力試験します。

StudioがWailsを使うことと、生成業務ApplicationがWails Desktop Targetを持つことは別Scenarioです。Studio HostをPAS-04の業務Target成功Evidenceへ読み替えません。

## 7. PAS-05 — 外部状態が不確かな場合でも安全に実行・再開できる

### 利用者価値

外部RuntimeやResourceの状態が不明な場合に、TEIは「成功したはず」と推測したり、副作用を無条件にreplayしたりせず、Observation、ownership、fencing、durable handle等を使って安全に判断できます。

### 主なRequirement

- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-020 再実行 / takeover / unknown
- RQ-021 Reference / Conformance

### Evidence上の注意

このScenarioはDeployment / Plugin / Resource lifecycle等の既存Referenceで重要ですが、現行v1 Gate Bへ新しいblockerとして追加しません。

現在のGate B scopeに必要な意味だけを既存三laneで確認し、より広いruntime recovery能力は対応するMilestone / Referenceで別Evidenceとして扱います。

## 8. PAS-06 — 既存の意味知識を探索してから新規実装へ進める

### 利用者価値

開発者またはAIは、毎回ゼロからコードを生成せず、既存Data、Relation、Validation、Capability、Application Pattern、利用例、Evidenceを探索し、適用可能性と不足条件を確認してから新しいApplicationを組み立てられます。

### 代表Scenario

```text
目的 / intent
   ↓
既存意味資産を探索
   ↓
identity / dependency / evidenceを確認
   ↓
applicable / incompatible / unknownを分離
   ↓
明示参照による構成候補
   ↓
shared Acceptance Boundary
```

類似名や検索順位だけを意味適合の根拠にしません。AI Knowledge、過去trajectory、索引はCanonicalの第二正本にしません。

### 主なRequirement

- RQ-002 Source traceability
- RQ-003 単一権威
- RQ-014 Application Pattern再利用
- RQ-015 knowledge-first AI
- RQ-016 AI suggestion / history
- RQ-017 二重管理最小化
- RQ-021 Reference / Conformance
- RQ-023 一人でも維持可能な構造

### 現在のEvidence位置

本格実証はGate C後です。Editor / AI providerを先に固定せず、Repository上の実在資産と同じread-only探索境界を人とAIが共有できることを優先します。

## 9. Cross-cutting Requirement

### RQ-021 — Reference / Conformance

すべてのPASに横断します。

抽象的に「設計上できる」ではなく、小さいReference、fixture、Conformance、Evidenceから境界を再現できることを要求します。

### RQ-022 — 完成製品と設計検証段階を区別

すべてのPASに横断します。

architecture probe、local fixture、synthetic adapter、beta framework、actual provider run、production-ready claimを同じ`done`へ畳みません。

### RQ-023 — 一人でも維持可能な開発構造

主にPAS-01 / PAS-06へ関係しますが、全体の作業分割へ横断します。

Scenarioを増やしすぎず、同じRepository内でRequirement → Decision → Reference → Evidenceを追えることを優先します。

### RQ-024 — Target別性能

PAS-04で具体化します。

単一の固定性能値を全Targetへ適用せず、第一Targetと第二Targetの実測からTarget / Driver / Providerごとの測定境界を確定します。

## 10. Requirement → Primary Scenario mapping

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

この表はcoverage scoreではありません。Primaryは「Requirementを最初に具体化する代表Scenario」を示すだけで、他ScenarioのEvidenceを不要にしません。

## 11. 現行Gateとの関係

```text
Gate A
  -> PAS-02のApplication意味境界をDriverへ渡せることを確認

Gate B
  -> PAS-02を中心に、PAS-03のreal Provider境界を含むsource-fixed Evidenceを収束

Gate B -> C Handover
  -> PAS-02 / PAS-03で固定したrevisionとunresolvedを第一Targetへ渡す

Gate C
  -> PAS-04を第一Web Targetで実証

Gate C後
  -> PAS-01のStudio / AI authoring loopを実Reference化
  -> PAS-06のknowledge-first reuseを実Reference化
  -> 第二TargetでPAS-04を圧力試験
```

PAS-05は既存runtime / deployment workstreamへ接続します。現行Gate Bの必須scopeをこの文書から拡張しません。

## 12. 今後のADR / Plan Issueへ要求する最小Trace

新しい上流設計・機能提案では、可能な範囲で次を明示します。

- 関連Product Requirement
- 主Product Acceptance Scenario
- Scenarioで失敗すると何が利用者に見えるか
- 保存する意味境界
- Provider / Target / Hostへ残す差
- Reference / fixture
- 必要なEvidence nature
- 既存Gateへ影響するか、影響しないか
- unresolved
- reopen condition

実装が先に存在する場合でも、この欄を埋めるためだけに架空のRequirementを作りません。

## 13. shared Acceptance Boundaryの位置づけ

PAS-01とPAS-06の中心にあるAcceptance Boundaryは、Studio専用でもAI専用でもありません。

```text
file edit -------┐
Studio ----------|
CLI / refactor --|--> candidate --> shared Acceptance Boundary --> authoritative source
AI Proposal -----|
Matrix ----------|
optional Patch --┘
```

入口ごとに独自のsemantic validationやauthority modelを持たせません。

`Semantic Patch only`、`Base Semantic Digest`、mandatory authored Stable IDを前提にした旧Authoring設計は、この追跡基線では現行前提にしません。

## 14. この文書の非目標

- Product Requirementsの置換
- 新しいCanonical object `ProductAcceptanceScenario`の追加
- Scenario Registry runtimeの実装
- Requirement coverage率や完成率の算出
- Gate B / Gate Cの合格条件変更
- Gate C前のStudio / AI詳細UI固定
- 第一TargetだけでPresentation IRを完成させること
- WailsをTEI全体のTarget / Runtime標準へ固定すること

## 15. 次の適用順

1. 現行Gate Bは既存三laneのactual Evidence取得へ集中する。
2. Gate B HandoverではPAS-02 / PAS-03の成立範囲と`unresolved`を明示する。
3. 第一Web Target / Gate CではPAS-04のScenario identityをEvidenceへ対応付ける。
4. Gate C後、PAS-01をStudioの最初のrelation-edit loopへ適用する。
5. 同じAcceptance BoundaryをAI Proposal / CLI / file-native editingで再利用する。
6. PAS-06は既存意味資産のread-only探索から始め、AI providerや巨大なKnowledge基盤を先行させない。
7. 第二TargetのEvidenceからPAS-04 / RQ-024を再評価する。

この順序により、上流要求を抽象的に増やすのではなく、既存の実証主線へ段階的に接続します。
