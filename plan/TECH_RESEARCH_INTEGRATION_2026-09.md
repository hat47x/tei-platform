# TEI 技術リサーチ統合方針 — 2026-09

Status: **current research-integration plan**  
Last updated: **2026-09-18**

## 1. 位置づけ

この文書は、`TEI 技術リサーチ短報`で確認した外部研究・技術動向を、TEIのProduct Value、Product Requirements、Milestone、Reference / Evidenceへどう翻訳するかを整理する月次の計画正本です。

外部研究やframeworkはTEIの仕様根拠そのものではありません。外部研究からは、失敗しやすい境界、比較可能なarchitecture、評価方法を取り込みます。最終的なProduct Requirementやarchitectureの採否は、TEI自身のReference / Conformance / actual Evidenceで決めます。

したがって順序は次です。

```text
external research / engineering evidence
        ↓
TEI Product Valueとの照合
        ↓
既存Requirementで表現できるか
        ↓
不足するならRequirement候補
        ↓
small Reference pressure
        ↓
TEI自身のEvidence
        ↓
Requirement / ADR / Milestoneへ昇格または棄却
```

技術名や論文固有の構造をCanonical Semantic Layerへ写すことより、技術が変わっても残る境界を抽出することを優先します。

## 2. Product Valueから見た評価軸

TEIの中心価値は、業務システムの意味を画面・コード・DB・特定Runtimeより一段上に置き、その意味がどの実装へ落ち、どの権限で実行され、実際に何が起きたかまで追跡できることです。

今回の研究は、次の既存原則へ戻して評価します。

- meaningはsurface / implementationから独立させる
- implementation / policy / Targetは交換可能にする
- native assetはnativeのまま接続できる
- runtimeの暗黙推測と権限拡大を減らす
- derived knowledge / generated artifactを第二の正本にしない
- Command / acknowledgementとObservation / Evidenceを分離する
- AIのautonomyとauthorityを分離する
- deterministicに検証できる境界はLLM reasoningへ委ねない
- 一人でも全体を追える維持可能性を保つ

この軸から外部研究を読むと、2026-09-11短報の共通点は「より賢いagentを作ること」ではありません。

**AI / Rule / Humanの判断を交換可能なProposal / Policy層へ留め、その下にsemantic action、current authority enforcement、actual Observationを置くこと**が共通して重要です。

## 3. 9月前半ですでに取り込んだ知見

### 3.1 LLM × Model-Driven Engineering

2026年のsystematic mapping studyでは、LLMのMDE利用はModel Generationへ偏り、DSL Engineering、Metamodeling、既存toolchainとの深い統合は相対的に少ないことが報告されています。

TEIはこの偏りを追わず、AIの主要価値を大量生成へ固定しません。既存知識の探索、変更候補、構造検査、Validation / Conformance、Source-to-Canonical Traceabilityを中心に置きます。

### 3.2 Structured Intentと決定論的Generator

自然言語からWorkflowを構築する研究からは、LLMによるIntent抽出と、検証可能なStructured Intent、決定論的Generatorを分離する境界を取り込みました。

```text
Natural Language / AI / Editor
            ↓
Proposal / Structured Intent candidate
            ↓
Shared Acceptance Boundary
            ↓
authoritative source
            ↓ reload
Canonical TEI Meaning
            ↓
Deterministic Mapping / Driver / Projection
            ↓
Execution / Observation
```

ProposalはCanonicalの別名ではありません。

### 3.3 Experience reuse

成功Proposal、修正trajectory、利用例等はDerived Experience Knowledgeとして再利用できます。ただし、元のCanonical ID / Source / Evidenceへ戻れ、staleなdependencyを無条件に再利用せず、共通Acceptance Boundaryを迂回しないことを要求します。

### 3.4 AI autonomyとauthority

AI4SE等の整理から、purpose、target activity / artifact、AI mechanism、autonomy levelを比較可能にし、autonomyとwrite / invoke / Resource authorityを別軸にする方針を固定しました。

### 3.5 Semantic / Functional / Visual Evidence

Vision2Web等から、Target評価を少なくとも次へ分けています。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

VLM / screenshot similarityだけでSemantic / Functional Conformanceを合格にしません。

### 3.6 Wails v3

WailsはStudio / HostやTarget Adapterのarchitecture probeとして有用ですが、Wails APIをTEIのCanonical Presentationへ写しません。

特に次を分離します。

- Target Capability
- Execution / Trust Profile
- Host inspection / control plane
- Application Presentation / Interaction Target

## 4. 9月中旬短報から新たに深める知見

### 4.1 Beyond Agent Harnesses — authorityはplanner-visible stateだけには存在しない

*Beyond Agent Harnesses: Cross-Substrate Authority for Multi-Agent Systems* は、workspaceやagent memoryが同じに見えても、runtime、registry、approval service等の外部authority stateによって安全なactionが逆になる **cross-substrate authority gap** を示します。

研究では、plannerへ情報を追加するだけではunsafe decisionが残る一方、effect直前のdeterministic authority guardが固定されたmodel intentに対してunsafe effectを遮断できました。

TEIへの含意は明確です。

```text
planner / modelが知っている
        !=
現在そのeffectを起こすauthorityがある
```

また、

```text
Proposal accepted once
        !=
future mutation authorized forever
```

です。

TEIではProposal / Decision / Canonical meaningだけでなく、effectful Capability invocationやResource accessの直前に、**current authority stateを決定論的に強制できる境界**が必要です。

ここで重要なのはagent専用securityを作ることではありません。Human、AI、automation、generated codeを同じCapability / Permission / Resource authorityへ収束させることです。

### 4.2 Avatar — reasoning policyとvalidated action catalogを分離する

*Avatar: Toward Autonomous End-to-End Orchestration of Scientific Workflows using LLMs* は、Orchestrator / Executor / Provenance Monitorを分離し、rule-based policyとLLM policyを同じadapter-validated action catalogへ載せています。

TEIではAvatarのactor architecture自体を採用しません。取り込むのは次の境界です。

```text
Rule / AI / Human policy
        ↓ proposal / selection
Validated Semantic Action Catalog
        ↓
Driver / Capability / Resource authority
        ↓
Execution
        ↓
Observation / Provenance
```

policyがRuleからLLMへ変わっても、実行可能actionの意味、Validation、Permission、Observationまで変わってはいけません。

したがって、AI providerやagent frameworkをRuntime semanticsの所有者にしません。

### 4.3 Procedural Graphs — procedural knowledgeはBusiness Procedure authorityではない

*Procedural Graphs: Self-Evolving Execution Structures for LLM Agents* は、手続き知識を `(procedure, relation, procedure)` のgraphとして外在化し、局所近傍をguidanceへ利用し、成功 / 失敗trajectoryからgraphを更新します。更新はheld-out validationで改善を確認したものだけを採用します。

これはTEIのRQ-025 Business Procedureと名前が近いですが、同一視してはいけません。

```text
Business Procedure
  = 業務上の意味 / accepted business source

Procedural Graph
  = agent / developer支援のderived procedural knowledge
```

Procedural Graphが「この順序でうまくいった」と学習しても、それだけで業務上の正しい手順、Application Function、Authorization、Capability authorityにはなりません。

TEIでは次の形なら価値があります。

```text
Canonical / accepted sources
        ↓
execution / proposal trajectories
        ↓
Derived Procedural Knowledge candidate
        ↓
validation / comparison
        ↓
retrieval / guidance only
        ↓
Proposal
        ↓
Shared Acceptance Boundary
```

Derived Procedural Knowledgeを削除・再生成してもCanonical Worldが変わらないことを守ります。

### 4.4 ASIL — screenshot-and-clickをsemantic interactionの正本にしない

*ASIL: Replacing Screenshot-and-Click with Structured State and Semantic Actions* は、pixel screenshotとmotor actionではなく、structured software stateとschema-constrained semantic actionをagent interfaceに利用します。

この結果をTEIの固定性能値にはしませんが、architecture上の圧力は強いです。

TEIはHuman UI、Agent、API / MCP、VUI等を別々の業務意味として作るのではなく、同じApplication meaningからTargetごとのinteractionへProjectionすることを目指しています。

したがってagent integrationでも、可能な範囲で次を優先します。

```text
Canonical / Application meaning
        ↓
Semantic Interaction / Action
        ↓              ↓
Human Target       Agent / API Target
        ↓              ↓
surface events     structured invocation
```

DOM selector、screen coordinate、click sequence等はTarget固有のrealization / test evidenceにはなり得ますが、Canonical action identityにはしません。

ただしGUI automation自体を禁止しません。semantic interfaceを持たないlegacy applicationではpixel / accessibility / DOM等のsurface automationが必要です。その場合も「meaning」と「surface motor event」を区別します。

### 4.5 GraphDroid — すべての操作でLLMを呼ばない

*GraphDroid: Asynchronous LLM-Based Mobile App GUI Testing via History-Aware Exploration and Hybrid Intent Fulfillment* は、complex intentをLLMへ、simple operationを軽量heuristicへ分け、history-aware explorationと非同期intent生成を組み合わせています。

TEIへの含意は、特定のGUI testing algorithm採用ではありません。

```text
known / deterministic operation
  -> deterministic path

ambiguous / compositional intent
  -> AI reasoning candidate
```

というroutingをProduct原則へ沿って評価できることです。

AIを呼ばない経路を「低機能」とみなさず、同じsemantic action contractをより安価・安定に満たせるなら優先します。

### 4.6 Wails v3 beta.19 — Host transportのtrust境界をTarget意味へ混ぜない

Wails v3.0.0-beta.19では、MCP origin / remote accessのhardeningとtoken authentication、runtime request size制約等が追加されています。

これはWails採用理由ではなく、既存RQ-013 / RQ-019の圧力です。

Host control interfaceをnetwork越しに公開する場合、origin、token、transport、request limit等はExecution / Trust Profileです。これらをApplicationの業務意味やMCP TargetのCanonical contractへ混ぜません。

### 4.7 The Router Within — Knowledge available と Knowledge injected を分ける

*The Router Within: Eliciting Native Skill Routing from a Frozen LLM* は、Skill library全体のmetadataを毎回contextへ載せるのではなく、frozen LLM内部の中間状態からrouting signalを読み出し、候補Skillだけを後段へ渡す方式を示しています。

TEIが取り込むべき点はmodel-native routerそのものではなく、次の非同一性です。

```text
Knowledge available
        !=
Knowledge selected
        !=
Knowledge injected
        !=
Semantic applicability
```

Knowledge / Skill / Reference patternが存在することだけで、毎回AI contextへ全量投入する必要はありません。まずtask / intentに対して候補を絞り、その後にCanonical identity、dependency、Source、Evidence、current revisionから適用可能性を判定します。

routing実装はexact lookup、graph、lexical、embedding、Derived Cognitive Retrieval、model-native routing等を交換可能にします。routing scoreやmodel内部signalは候補生成Evidenceであり、Semantic authorityではありません。

### 4.8 ProgramDistill — Targetの振る舞いをreplay可能なEvidenceへ分解する

*ProgramDistill: From Interactive Web Apps to Verifiable Reference-Guided SWE Tasks* は、完成済みWeb applicationを操作してfeatureを発見し、featureごとにreplay可能なbehaviorを構成して検証する方法を示しています。

TEIへの圧力は、Target評価をscreen / DOM snapshotへ閉じないことです。

```text
Canonical Scenario / Semantic Operation
        ↓
Target-specific interaction realization
        ↓
Replayable Behavior
        ↓
Result / Observation
        ↓
Functional Evidence
```

replay behaviorはTarget Evidenceとして有用ですが、Canonical business meaningの正本にはしません。DOM selector、coordinate、route、component identity等はTarget realizationとして保持し、behaviorから上位meaningを逆定義しません。

複数featureをまたぐ深いworkflowほど復元が難しくなるという結果も、単一画面の一致だけではTarget Conformanceを評価できないことを補強します。

### 4.9 RSIAgent — experienceをcondition / action / consequenceへ圧縮する余地

*RSIAgent: Autonomous Exploration for Recursive Self-improvement in New Environments* は、環境探索からaction、condition、consequence、failure、boundary condition等の関係を検証し、model parameterを更新せず再利用できるmemoryへまとめます。

TEIではこれをself-improving Canonical Knowledgeとして採用しません。一方、Derived Experience Knowledgeを単なるtrajectory logより一段構造化する候補になります。

```text
Execution / Observation
        ↓
verified experience candidate
  condition / action / outcome / boundary
        ↓
Derived Experience Knowledge
        ↓
future exploration / Proposal guidance
```

experienceは作成時のCanonical / Target / Driver / Resource / revisionへtraceでき、current authority / dependencyが変われば再評価できる必要があります。過去に成功した因果候補を現在のBusiness Ruleへ自動昇格しません。

### 4.10 HazardAuditor — 異種Runtimeのevent正規化はEvidence側で試す

*HazardAuditor: From Executable Threats to Safer Computer-Use Agents* は、複数のcomputer-use agent frameworkをcontrolled environmentで実行し、異種interactionをcanonical event representationへ正規化して横断評価します。

TEIで有用なのは、各Target / Agent固有eventを比較可能なEvidenceへ変換する発想です。ただし、ここでいうevent normalizationをCanonical Semantic Modelへ直結しません。

```text
Target / Agent native events
        ↓
derived normalized evidence
        ↓
cross-target / cross-runtime comparison
```

raw Evidenceへ戻れるprovenanceを保持し、正規化で失われたTarget固有差をunresolvedとして残せることを優先します。Universal Event Modelを先に設計しません。

### 4.11 Agora — Git lineageはDerived Knowledgeの再現性に使える

*Agora: Git as Shared Memory for Collective AutoResearch* は、result、hypothesis、verification等をappend-onlyなGit DAGとして記録し、親commitを知識の系譜として利用します。

TEIではGit commitをSemantic Knowledge nodeへ一対一対応させる必要はありません。一方、AI / Humanが作るDerived Knowledgeや検証成果について、immutable lineage、再現可能なparent relation、derived indexを分ける考え方は既存のGit活用方針と整合します。

重要なのは次です。

- authoritative sourceのversion historyとDerived Knowledgeを混同しない
- derived indexを削除しても履歴・正本を復元できる
- claim / experienceのprovenanceを保存しても、それだけでCanonical authorityにしない
- 複数agentの共有memoryを中央の不可視stateだけへ閉じ込めない

### 4.12 Wails v3 beta.20〜23 — Host lifecycleとbinding dependencyをTarget meaningから分ける

Wails v3はbeta.20〜23で、Windows / Linuxのaborted request処理、pure Go WebView2 loaderへの整理、single-instance application ID、generic model creatorの依存順序や相互依存によるstack overflow等を修正しています。

TEIの上位設計を変える内容ではありませんが、二つのnegative pressureとして保持します。

1. WebView request cancellation / navigation lifecycleはHost / Adapter lifecycleで扱い、Application business outcomeへ短絡しない
2. generated bindingは個々の型だけでなく、型依存graph、宣言順、cycleを含めて検証する

Wails固有のbuild / binding事情をCanonical Presentation vocabularyへ持ち込みません。


## 5. 今回の統合仮説

最新短報をTEIへ翻訳すると、次の5層が最も安定した仮説です。

```text
1. Authoritative Meaning / Current State
   Canonical / Source / Binding / Permission / Resource state

2. Observation Surface
   structured state / Evidence / Target observation

3. Decision / Proposal Policy
   Human / Rule / AI / derived procedural guidance

4. Validated Semantic Action
   Application Pattern / Capability / accepted edit / semantic operation

5. Effect Boundary
   current authority check -> execution -> Observation
```

重要な非同一性は次です。

```text
Observation      != Authority
Proposal         != Decision
Decision         != current execution authority
Policy           != Action Contract
Action Contract  != Target motor event
Trajectory       != Business Procedure
Derived Procedure != Canonical source
Command accepted != Observed outcome
```

この分離は「agent architecture」をTEIへ追加するためではありません。Human / AI / automation / Targetの違いを越えて、同じ意味とauthorityを維持するためです。

## 6. Product Requirement採否判断

### 6.1 新しい番号付きRequirementは追加しない

2026-09-11時点ではRQ-027等の新しいProduct Requirementは追加しません。

理由は、今回の主要論点が既存Requirementの間をつなぐ境界であり、新番号を追加すると同じ責務を複数RQが所有しやすいためです。

- semantic action / multi-surface interaction → RQ-013 / RQ-014
- derived procedural knowledge / hybrid AI → RQ-015
- replaceable reasoning policy / Proposal history → RQ-016
- effect-time authority → RQ-019
- reproducible evaluation → RQ-021 / RQ-022
- Targetごとのcost / latency / action count → RQ-024
- Business Procedure authorityとの分離 → RQ-025 / RQ-026

### 6.2 RQ-019は最優先のRequirement refinement候補

現行RQ-019は、AI / Editor / generated code / automationがCapability / Permission / Resource境界を迂回しないことを要求しています。

最新研究からは、さらに次を明文化する価値があります。

> planner-visible state、過去のapproval、Proposal acceptanceだけをexecution authorityとみなさず、effectful operationはeffect直前のcurrent authority stateに対してdeterministicに許可されなければならない。missing / stale / ambiguous authorityではfail closedする。

ただし外部論文だけを理由にRequirement正本へ即時追記しません。

まず既存Resource Broker / Permission / fencing等で小さいcross-substrate Referenceを作り、この条件がTEI内部Evidenceとして成立することを確認します。成立後にRQ-019へ昇格します。

### 6.3 RQ-013はGate C / Milestone 5の解釈を深める

RQ-013にはすでにTarget-neutral InteractionとTarget Capability / Execution-Trust分離があります。

現時点では新しいRequirement文言を追加せず、Reference pressureとして次を確認します。

- Human TargetとAgent / API Targetが同じsemantic operationへ戻れるか
- DOM / coordinate / click列をCanonical action identityにせずに済むか
- semantic interfaceがないlegacy Targetではsurface automationを明示Realizationとして隔離できるか

### 6.4 RQ-015 / RQ-016は現行文言で主要論点を所有できる

RQ-015はDerived Experience Knowledge、deterministic generation優先、AI Proposal非authorityをすでに要求しています。

Procedural Graphs / GraphDroidからはRequirement追加より、評価caseを増やす方が適切です。

RQ-016もautonomyとauthorityを分けています。Avatarからは「Rule / AIが同じvalidated action catalogを使えるか」をReference pressureへ追加します。

### 6.5 RQ-025 / RQ-026は変更しない

Procedural Graph研究を理由にBusiness Procedure schemaやWorkflow Engineを追加しません。

RQ-025 / RQ-026はaccepted business meaningとcross-view reviewを所有します。agentが学習したprocedural graphはderived knowledgeとして別責務に留めます。

## 7. Milestoneへの反映

### Gate B / Milestone 3

**変更しません。**

今回の研究を理由にGate B Required Evidence lane、PAS、Gateを追加しません。Gate Bは現在の#310 / #255 / #308 same-R Evidenceを閉じることが主線です。

Agent interface、Procedural Graph、hybrid policyをGate Bへ前倒ししません。

### Milestone 4 / First Web Target / Gate C

Gate Cの既存Semantic / Functional / Visual分離を維持します。

追加のplanning pressureとして、一つの利用者操作について次のtraceを観測可能にします。

```text
Application meaning
  -> semantic operation
  -> Web-specific interaction realization
  -> Driver / Capability
  -> Result / Observation
```

これは新しいGate C Required Evidenceを直ちに追加するものではありません。First Targetで「semantic operationとsurface eventを分けられるか」を見られるだけのtraceを残すことが目的です。

### Milestone 5 / Second Target

第二Targetは、第一Targetで得たTarget-neutral Interaction仮説を圧力試験します。

選定時には、GUI形状が違うTargetだけでなく、API / MCP / agent-facing interactionのようにsurface性が異なるTargetも候補に含めます。ただし研究追随だけで第二Targetを今固定しません。

比較軸へ次を追加候補とします。

- same semantic operationをTargetごとにどうrealizeするか
- structured observationの範囲
- surface motor eventへの依存量
- Target / Host固有Execution-Trust
- AIを使わないdeterministic pathの成立範囲

### Milestone 6 / Studio

Studio authoringは従来どおりShared Acceptance Boundaryを通します。

Agent researchを理由にStudio独自action authorityを作りません。Studio command、AI Proposal、CLI editが同じaccepted operation / authorityへ収束できるかを確認します。

### Milestone 7 / knowledge-first / AI

今回の主な実証先です。

Milestone 7では既存`ISSUE-000047` knowledge-first explorationと`ISSUE-000027` AI Proposal / Acceptanceへ、次のReference pressureを段階的に追加します。

1. Rule policyとAI policyが同じvalidated semantic action contractを利用できる
2. policyがraw tool callやResource authorityを独自に増やせない
3. Proposal acceptance後にauthorityが変化した場合、effect直前でfail closedできる
4. successful / failed trajectoryからderived procedural candidateを作れてもCanonicalを自動変更しない
5. derived procedureの更新をvalidationでき、削除・再生成してもCanonical Worldが変わらない
6. known operationはdeterministic pathを利用し、AI reasoningは曖昧・compositionalなcaseへ限定できる
7. routing理由、policy種別、action、authority decision、Observationを後から追跡できる

### Milestone 9 / Operational Boundary

Wails beta.19のMCP hardeningは、Host control planeのproduction trustを考える際の具体的pressureとして保持します。

origin、token、remote exposure、request size、session、credential等をApplication meaningへ混ぜずExecution / Trust Profileとして扱えることを確認します。

## 8. Reference / Evidence計画

詳細schemaやpackageを先に固定せず、次の順で価値を確認します。

### R1 — Semantic Action Trace

目的:

- Web Targetの一つの操作を、surface eventより上のsemantic operationへtraceできるか確認する

最低条件:

```text
Canonical/Application identity
  -> semantic operation
  -> Target realization
  -> Driver / Capability
  -> Result / Observation
```

ここで新しいUniversal Action Schemaを作ることは目的ではありません。

### R2 — Cross-Substrate Authority

目的:

- Proposal / planner-visible stateとcurrent execution authorityが食い違うcaseを再現する

pressure case:

```text
t0: Proposal accepted / action candidate created
t1: Permission / Resource authority changes
t2: old candidate attempts effect
```

期待:

- t0のapprovalだけではeffectしない
- t2のcurrent authorityでdeterministicに再評価する
- revoked / stale / ambiguousなら副作用前にfail closedする
- reasonとcurrent authority referenceをEvidenceとして追える
- modelへ再質問しなくてもguardが成立する

このReferenceが成立した場合、RQ-019 refinementをRequirement正本へ昇格する候補とします。

### R3 — Replaceable Decision Policy

目的:

- RuleとAIがRuntime意味を二重化しないことを確認する

期待:

```text
Rule policy ----┐
                ├-> same action catalog -> same validation/authority
AI policy ------┘
```

policyごとに別Capability、別Permission semantics、別Observation modelを作らないことを確認します。

### R4 — Derived Procedural Knowledge

目的:

- trajectory reuseが第二正本にならないことを確認する

最低条件:

- source trajectory / Canonical / Evidenceへ戻れる
- successだけでなくfailed / rejected trajectoryも区別できる
- proposed graph/updateはvalidation前にauthoritativeにしない
- validation悪化なら採用しない
- graph/index全削除後もCanonical / authoritative sourceは不変
- Business Procedureへ自動昇格しない

FlyHash等のDerived Cognitive Explorationと接続する場合も、候補生成の下位方式として比較し、Procedural Graphを新Architecture Layerへ自動昇格しません。

### R5 — Hybrid Intent Fulfillment

目的:

- 全operationをLLMへ送る必要があるかをTEI自身で測る

比較:

```text
A: deterministic / exact path
B: AI-assisted path
C: hybrid routing
```

測定候補:

- success / semantic correctness
- action count
- LLM call count
- latency
- cost
- retry / failure classification
- authority rejection count

外部研究の36.4%等の数値をTEIの閾値にはしません。

### R6 — Selective Knowledge Routing

目的:

- availableなKnowledge / Skill / Reference patternを毎回全量contextへ投入せず、必要候補だけを選べるか比較する

比較候補:

```text
A: exact / deterministic selection
B: lexical / graph / embedding retrieval
C: cognitive / model-native routing
D: hybrid routing
```

最低条件:

- routing mechanismを交換してもCanonical / accepted meaningが変わらない
- selected knowledge identity / provenanceを追跡できる
- routing scoreをSemantic applicabilityやDecisionへ自動昇格しない
- selection後にdeterministic qualificationを通す
- all-skill metadata preloadを必須architectureにしない

#### R6 first Reference pressure — 2026-09-18

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6_selective_knowledge_routing_pressure_test.go`

current Business Procedure ReferenceのBP-02を使い、次を最小Pressureする。

```text
available Application Pattern knowledge
  -> router-specific selected candidates
  -> deterministic qualification against current task / Source Index
  -> task-scoped disclosed knowledge
```

最初の比較では、exact deterministic routerと、意図的に候補を広く返すtest routerを使う。routing方式が異なっても、deterministic qualification後のqualified identityが同じになることを要求する。

このReferenceで固定する非同一性:

```text
available != selected
selected != qualified
qualified != execution authority
routing mechanism != semantic authority
```

routing score / confidence / winnerは導入しない。embedding、model-native routing、Derived Cognitive Retrievalもこの段階では実装せず、後からrouterを交換しても同じqualification boundaryを通せる形だけをPressureする。

R7 / R8はここで擬似実装しない。R7はFirst Web Targetのactual interaction Evidence、R8はactual Execution / Observationを入力として必要とするため、R6のstatic Source Evidenceから成功扱いへ昇格させない。

#### R6B heterogeneous knowledge pressure — 2026-09-19

R6の候補集合をApplication Patternだけに閉じず、用途ごとに異なるknowledge kindを同じrouting / qualification境界へ載せる。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6b_heterogeneous_knowledge_routing_pressure_test.go`

候補poolには少なくとも次を含める。

- Business Procedure Reference
- Application Pattern
- Semantic Validation
- Semantic Capability
- task-specific probe Evidence

Pressure対象:

```text
BP-02 read-only
  -> Procedure + customer_reference Pattern
  -> customer-edit conflict Evidenceはinjectしない

BP-05 validation
  -> Procedure + exact Validation subjects

BP-06 authorization boundary
  -> Procedure + security.authorize Capability

R1 customer-edit review
  -> Procedure + customer_reference Pattern
  -> explicit customer-edit conflict probe Evidence
```

exact task routerと、全候補を返すbroad heterogeneous routerを比較し、deterministic qualification後のidentity集合とdisclosure集合が一致することを要求する。

重要なのは全knowledge kindを一つのauthorityへflattenしないことである。

```text
Procedure Reference != Semantic Capability
Semantic subject    != probe Evidence
probe Evidence      != runtime authority
qualified context   != executable context
```

Evidenceはtask-specific provenanceとして選択できるが、Operational taskへ無関係なreview Evidenceを注入しない。Capabilityがselected / qualifiedでもAuthorization成立やinvoke authorityを意味しない。

この段階でもuniversal Context Packet、共通score、knowledge-kind横断confidence、production disclosure schemaは追加しない。

#### R6C projection dependency handoff — 2026-09-19

R6Bでqualifiedになったknowledgeを、R3のbounded projectionへ渡すときの境界をPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6c_projection_dependency_handoff_pressure_test.go`

R6Cはfreshness判定を新設しない。R3R〜R3Xが所有するdependency / reason / change / context identityを再実装せず、routing側では次だけを扱う。

```text
qualified knowledge
  -> consumer-specific projection selection
  -> existing R3 dependency key / reason
```

current BP-06ではR6Bから:

```text
Business Procedure Reference
Semantic Capability: security.authorize
```

がqualifiedになるが、これだけで両方をcurrent stop projection dependencyへ昇格させない。

Business Procedure Referenceは既存R3Sの:

```text
reference:business-procedure-reference:<locator>
reason = authorization-requirement-source
```

へ接続できる。

一方 `security.authorize` がR6Bでqualifiedであることだけから、新しいsemantic dependency edgeを発明しない。current R3 projectionがそのCapability sourceをdependencyとして選択していなければ:

```text
qualified-not-selected-by-consumer
```

として保持する。

Human / Agentでprojection category集合が異なっても、同じBusiness Procedure dependencyを利用する場合はdependency key / reason provenanceを変えない。

```text
consumer-specific category
  != new dependency identity

qualified
  != projection dependency
  != fresh
  != authorized
```

R6CはR3S/R3Vのreason identity、R3Wのvalue-change判定、R3XのAuthorization freshness authorityを所有しない。

#### R6D dependency origin classification — 2026-09-19

R6Cでqualified knowledgeからR3 dependencyへlinkできるようになったが、projection dependencyの全てがrouting由来とは限らない。

current BP-06では少なくとも:

```text
Business Procedure Reference
  -> routed-qualified-knowledge

protected-target overlay
  -> consumer-reference-input

runtime Authorization Observation
  -> consumer-runtime-observation
```

を分離する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6d_dependency_origin_classification_pressure_test.go`

R6Dは各R3 dependency reasonについて、出所を:

- routed qualified knowledge
- consumer-local Reference input
- consumer-local Runtime Observation
- consumer-local Semantic subject

のいずれかとして明示する。

同じdependencyをroutingとconsumer-localの両方がclaimした場合、またはいずれにも分類できない場合はfail closedする。

```text
routing available
  != routing owns every dependency

consumer-local input
  != routed knowledge

origin class
  != authority
  != freshness
```

これによりR6を万能input registryへ拡張せず、R3側が直接必要とするReference / Runtime / Semantic dependencyを保持できる。

#### R6E router substitution invariance — 2026-09-19

R6の最初の目的であるrouter交換可能性を、qualificationだけでなくR6Dのprojection dependency originまで下流へ延長してPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6e_router_substitution_origin_invariance_pressure_test.go`

current BP-06について:

```text
exact task router
  -> narrow selected set

broad heterogeneous router
  -> deliberately over-selected set
  -> deterministic qualification rejects unrelated knowledge
```

とした上で、両者が最終的に同じ:

- qualified knowledge identity
- R3 dependency key / reason
- routed / consumer-local origin classification

へ到達することを要求する。

特に:

```text
router choice
  != dependency identity
  != dependency reason
  != origin class
```

である。

broad routerが余計なValidation / Pattern / Capability等をselectedに含んでも、qualification後のover-selectionをprojectionへ漏らさない。

またconsumer-localなprotected-target overlayやruntime Authorization Observationはrouter交換の影響を受けず、broad routerがそれらをrouted knowledgeとして取り込まない。

```text
routing mechanism
  = replaceable discovery/selection implementation detail

deterministic qualification
  = semantic/task boundary

R3 dependency provenance
  = consumer projection boundary
```

を保持する。

PreferredRouter / Winner / Score / Confidenceを追加せず、router比較結果をSemantic authorityやproduction routing policyへ昇格しない。

#### R6F dependency-driven materialization — 2026-09-19

R6Eでrouter交換後もdependency originが一致することを確認した次段として、実際にcontextへmaterializeするknowledge量をprojection dependencyから導く。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6f_dependency_driven_materialization_pressure_test.go`

materialization対象は:

```text
qualified knowledge
  + R6D dependency origin
  -> routed-qualified-knowledgeとして実際に消費されたknowledgeだけ
```

とする。

current BP-06 Human orientationでは:

```text
qualified:
  Business Procedure Reference
  security.authorize Capability

materialized:
  Business Procedure Reference

qualified-but-not-materialized:
  security.authorize Capability

consumer-local:
  protected-target overlay
```

となる。

runtime Observationだけをconsumer-local dependencyとして扱うprojection sliceでは、qualified knowledgeが存在していてもrouted materializationを0にできる。

```text
qualified
  != materialized

consumer-local dependency
  != reason to preload routed candidate pool

materialized
  != authorized
```

materialization manifestはderived / rebuildableなtest-only一覧であり、Universal Context Packet、Prompt payload、AllKnowledge dumpにはしない。

exact / broad routerの違いはmaterialized resultへ残さず、R6Eと同じくrouter実装差をcontext identityへ混ぜない。

#### R6G explainable non-materialization — 2026-09-19

R6Fでmaterialize対象を最小化した後、materializeされなかったqualified knowledgeを「欠落」として扱わず、identity / provenanceを保持した説明可能な状態としてPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6g_materialization_explanation_pressure_test.go`

R6Gではqualified knowledgeごとに:

```text
materialized
  reason = consumed-routed-projection-dependency
  dependency keys = exact R3 dependency identities

qualified-not-materialized
  reason = no-consumed-routed-projection-dependency
  dependency keys = none
```

をderived witnessとして保持する。

current BP-06では:

```text
Business Procedure Reference
  -> materialized
  -> routed R3 dependencyへ逆追跡可能

security.authorize Capability
  -> qualified-not-materialized
  -> SourceRefsは保持
  -> knowledge contentはcontextへ戻さない
```

となる。

これにより:

```text
not materialized
  != missing source
  != qualification failure
  != irrelevant globally
```

を区別する。

omission witnessはknowledge identity / kind / SourceRefs / reason codeだけを持ち、Body / Content / Prompt / Payload等を持たない。したがってexplainabilityのために未使用knowledge本文を再注入しない。

exact / broad routerで同じqualified / dependency consumptionへ収束する限り、materialization explanationも同一でなければならない。

consumer-local dependencyはknowledge omission理由へ変換せず、別dependency identityとして保持する。

R6G reasonはfreshness / authority / next action / routing preferenceを決定しない。

#### R6H materialization revision provenance — 2026-09-19

R6Gのmaterialization explanationに、qualified knowledgeがどのsource revisionを見ていたかをfingerprint付きprovenanceとして接続する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6h_materialization_revision_provenance_pressure_test.go`

R6Hでは各knowledge witnessに:

- materialization disposition
- materialization reason
- exact routed dependency keys
- RequiredInputFingerprints
- SourceRefs

を併置する。

ただし:

```text
input fingerprint changed
  != materialization disposition changed
  != freshness decided
  != reevaluation required
```

である。

current BP-06で`security.authorize` Capabilityのsource fingerprintだけをtest-onlyに変更しても、current bounded projectionがそのCapabilityをrouted dependencyとして消費していない限り:

```text
qualified-not-materialized
reason = no-consumed-routed-projection-dependency
```

を維持する。

同様にBusiness Procedure Referenceのfingerprint変更を観測しても、current dependency consumption identityが同じならR6H自身はmaterialized dispositionを反転しない。

fingerprint差分は「どのrevisionを見ていたか」を説明するprovenanceであり、Fresh / Stale / ChangeClass / ShouldReevaluate / AutoRefreshへ変換しない。これらはR3R以降のfreshness / reevaluation責務に残す。

exact / broad routerがcurrent qualificationへ収束する場合、revision provenanceも同一でなければならない。

#### R6I shared knowledge materialization / bounded use edges — 2026-09-19

R6Hでknowledge revision provenanceを保持した上で、同じqualified knowledgeをHuman / Agentのbounded consumerが異なるprojection categoryで利用する場合も、knowledge本文自体をconsumerごと・categoryごとに複製しない。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6i_shared_materialization_use_edges_pressure_test.go`

R6Iでは次を分離する。

```text
shared materialized knowledge
  = knowledge identity
  + kind
  + RequiredInputFingerprints
  + SourceRefs

bounded use edge
  = knowledge identity
  + projection category
  + dependency key
  + dependency reason
```

current BP-06ではBusiness Procedure Referenceを:

- Human orientation: `machine-reasons` から1回利用
- Agent resolution-inspection: `machine-reasons` と `known-authority-facts` から2回利用

するが、materialized knowledge bodyは双方とも同じ1件である。

```text
one shared knowledge body
  -> Human use edge x1
  -> Agent use edge x2

use edge count
  != knowledge copy count
  != stronger authority
```

`security.authorize` Capabilityは両consumerでqualified-not-materializedのまま保持し、Agent側のdetail増加だけで自動materializeしない。

shared materialized knowledge shapeにはAudience / Purpose / ProjectionCategory / DependencyKeyを持たせず、consumer固有情報はuse edgeへ限定する。これによりconsumer presentation差をknowledge identity / revision provenanceへ混ぜない。

R3AG〜R3AIが扱うoutput側のpurpose-bound explanation / drill-down / read-only reasoningは再実装せず、R6Iはその手前のknowledge input materializationだけをPressureする。

#### R6J bounded input plane assembly — 2026-09-19

R6D〜R6Iの責務を一つのbounded input assemblyへ接続する。ただし全入力を一つのgeneric packetへ平坦化せず、input originごとのplaneを保持する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6j_bounded_input_plane_assembly_pressure_test.go`

assemblyは少なくとも次の3 planeを分離する。

```text
routed knowledge body
  knowledge identity
  kind
  RequiredInputFingerprints
  SourceRefs

routed use edge
  knowledge identity
  projection category
  dependency key
  dependency reason

consumer-local use edge
  origin class
  projection category
  dependency key
  dependency reason
```

current BP-06 Human orientationでは:

- Business Procedure Reference body: 1件
- Procedureへのrouted use: 3 edge
- protected-target overlayへのconsumer-local Reference use: 3 edge
- `security.authorize` Capability: qualified-not-materialized

となる。

consumer-local planeにはKnowledgeKey / KnowledgeKind / SourceRefsを持たせず、overlayやruntime Observationをrouter knowledgeへ変換しない。

runtime Observationだけを使うsliceでは:

```text
routed knowledge = 0
routed use = 0
consumer-local runtime use = 1
qualified-not-materialized = Procedure + Capability
```

を維持する。

これにより:

```text
bounded input assembly
  != universal context packet

consumer-local input
  != missing routed knowledge
  != routed knowledge

multiple use edges
  != multiple source bodies
```

を固定する。

R6JはPrompt serialization、token budget、production packet schema、freshness decision、execution authorityを導入しない。

#### R6K explicit bounded input scope — 2026-09-19

R6Jでbounded inputを複数planeへ分離した次段として、planeが空である理由を暗黙推論しない。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6k_explicit_input_scope_expectation_pressure_test.go`

bounded consumerは既知planeごとに:

```text
required
out-of-scope
```

のどちらかを明示する。

current plane classes:

- routed-knowledge-body
- routed-use-edge
- consumer-reference-use-edge
- consumer-runtime-use-edge
- consumer-semantic-use-edge

BP-06 Human orientationではrouted knowledge / routed use / consumer Reference useをrequiredとし、runtime / semantic consumer-local planeはout-of-scopeとする。

runtime Observationだけのsliceでは逆にconsumer-runtime-use-edgeだけをrequiredとし、routed knowledge / routed use / consumer Reference / consumer Semanticをout-of-scopeとする。

これにより:

```text
required + observed 0
  -> fail closed

out-of-scope + observed > 0
  -> fail closed

out-of-scope + observed 0
  -> intentional absence
  != missing input
```

を区別する。

scope declarationはknown planeをexactly onceで覆う必要があり、未宣言planeや重複宣言を許さない。

ただしglobal `Complete` / `Ready` / `Valid` booleanを導入せず、各planeのbounded expectation / observationだけを保持する。

`qualified-not-materialized`はplane scopeとは別軸に残す。runtime-only sliceでrouted planeをout-of-scopeにしても、Procedure / Capabilityのqualified identityを「存在しない」へ変換しない。

R6KはR3AJ〜R3ALのmissing authority information request / response correlation / evidence qualificationを再実装せず、input assembly scopeだけをPressureする。

#### R6L scope identity / cardinality separation — 2026-09-19

R6Kのexplicit scope declarationについて、scopeの意味とobserved input countを分離する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6l_scope_identity_cardinality_pressure_test.go`

scope identityは:

```text
plane class
+ required / out-of-scope disposition
```

だけから導出し、Audience / Purpose / observed countを含めない。

current BP-06のProcedure-only bounded useでは:

```text
Human orientation
  routed knowledge body = 1
  routed use edge       = 1

Agent resolution-inspection
  routed knowledge body = 1
  routed use edge       = 2
```

となるが、両者のsemantic input scope identityは同じである。

```text
same scope declaration
  + different use cardinality
  -> same scope identity

different scope declaration
  -> different scope identity
```

とする。

これによりAgent側のdetail増加やcategory追加がinput scopeそのものの意味変更へ誤変換されることを防ぐ。

scope identityはexpectation declarationの順序にも依存させず、known planeをexactly once宣言したnormalized setから導出する。

ただしscope identityはauthority / completeness / freshnessではない。observed countはR6K witnessに残し、identityへ埋め込まない。

R6LはR3AG以降のpresentation差・outcome identityを再実装せず、knowledge input scopeのsemantic identityだけをPressureする。

#### R6M same-scope observation delta — 2026-09-19

R6Lでscope identityとobserved cardinalityを分離した次段として、同一scope identity内の観測件数差だけを比較する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6m_same_scope_observation_delta_pressure_test.go`

comparisonは:

```text
same ScopeIdentity
  + before plane observations
  + after plane observations
  -> plane-local cardinality delta
```

として行う。

current BP-06 Procedure-only bounded useでは:

```text
Human:
  routed knowledge body = 1
  routed use edge       = 1

Agent:
  routed knowledge body = 1
  routed use edge       = 2
```

なので同一scope identityのまま:

```text
routed knowledge body delta = 0
routed use edge delta       = +1
```

と観測できる。

これはconsumer-specific category利用数の差であり、scope meaning changeではない。

```text
cardinality delta
  != scope declaration change
  != freshness change
  != authority change
```

ScopeIdentityが異なるbefore / afterはR6Mでは比較しない。scope declaration変更はR6Lの別scopeとして扱い、cardinality deltaへ丸めない。

同じScopeIdentityを主張しながらplane dispositionが変わっている場合もfail closedする。

plane observationの入力順序は比較結果へ影響させない。

R6MはR3AM以降のassertion / acceptance boundaryを再実装せず、bounded input observationの差分だけを扱う。

#### R6N same-scope membership delta — 2026-09-19

R6Mでcardinality deltaを観測できるようになった後、countが同じままmemberだけが入れ替わるケースを見落とさないため、同一scope内のmember identity差分を追加する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6n_same_scope_membership_delta_pressure_test.go`

plane member identityは内容本文ではなく、既存のbounded identityだけから構成する。

```text
routed knowledge member
  = KnowledgeKey

routed use member
  = KnowledgeKey
  + ProjectionCategory
  + DependencyKey
  + DependencyReason

consumer-local use member
  = OriginClass
  + ProjectionCategory
  + DependencyKey
  + DependencyReason
```

current BP-06 Human -> Agentではshared Procedure knowledge membershipは変わらず、Agent-only `known-authority-facts` use memberだけがaddedになる。

また:

```text
before use count = 1
after use count  = 1
```

でもmemberがmachine-reasonsからknown-authority-factsへ置換された場合:

```text
added   = 1
removed = 1
```

として観測できる。

一方、同じKnowledgeKeyのsource fingerprintだけが変わった場合はmembership changeへ変換しない。revision provenanceはR6Hの責務に残す。

```text
source revision changed
  != knowledge membership changed

member substitution
  != semantic meaning changed
  != freshness changed
```

R6NもScopeIdentity一致を必須とし、異なるscope間のmember集合を同一意味上のdeltaとして比較しない。

R6NはR3AM以降のauthority assertion / acceptance処理を再実装しない。

#### R6O retained-member provenance delta — 2026-09-19

R6Nでmembership identityとsource revisionを分離した次段として、同じScopeIdentityかつ同じKnowledgeKeyがbefore / afterの両方に残る場合だけ、knowledge provenanceのbefore / afterを観測する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6o_retained_member_provenance_delta_pressure_test.go`

対象はretained routed knowledgeだけとし、snapshotは:

```text
KnowledgeKey
KnowledgeKind
RequiredInputFingerprints
SourceRefs
```

を保持する。

```text
same ScopeIdentity
  + same KnowledgeKey retained
  + provenance differs
  -> retained-member provenance delta
```

とする。

同じKnowledgeKeyのfingerprintだけが変わる場合:

```text
membership delta = none
provenance delta = before / after retained
```

となる。

SourceRefsだけが変わる場合もprovenance differenceとして観測するが、それをsemantic change / locator refresh requiredへ分類しない。分類責務はR3R以降へ残す。

一方:

```text
knowledge member added / removed
  -> R6N membership delta
  != R6O revision delta
```

である。

またrouted use edgeだけが増減してknowledge body provenanceが同一ならR6O deltaは生成しない。

R6Oは`Changed` / `ChangeClass` / `Fresh` / `Stale` / `ShouldReevaluate` / `Authorized`等を持たず、before / after provenanceを観測するだけに留める。

R3AO / R3APのAccepted Decision / write-authority Evidence Gateは再実装しない。

#### R6P orthogonal observation decomposition — 2026-09-19

R6M / R6N / R6Oで分離したsame-scope observationを、一つの比較結果へ束ねても意味を平坦化しないことをPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6p_orthogonal_observation_decomposition_pressure_test.go`

R6Pは同一ScopeIdentity上で:

```text
cardinality delta
membership delta
retained-member provenance delta
```

を別axisとして保持する。

整合条件は:

```text
CountDelta
  = len(Added)
  - len(Removed)
```

である。

これによりduplicate use observationのように:

```text
count +1
membership +0
```

となる入力をfail closedできる。

一方:

```text
fingerprint-only change
  cardinality delta = 0
  membership delta  = 0
  provenance delta  = 1

same-count member substitution
  cardinality delta = 0
  membership        = Added 1 / Removed 1
  provenance delta  = 0
```

として、差分の種類を混ぜない。

knowledge memberのAdded / Removedはretained provenance deltaへ現れてはならない。

R6Pはこれらを`SemanticChanged` / `Fresh` / `ShouldReevaluate`等へ集約せず、input observationの直交分解に留める。

Accepted Decision / authority application / execution responseはR3側の責務のままとする。

#### R6Q router-substitution observation invariance — 2026-09-19

R6Eで確認したrouter substitution invarianceを、R6J〜R6Pのbounded input / observation decompositionまで貫通させる。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6q_router_substitution_observation_invariance_pressure_test.go`

exact task routerとbroad heterogeneous routerはselection段階では異なる。

```text
exact router
  -> narrow selected set

broad router
  -> over-selected set
  -> deterministic rejection
```

しかしqualification後に同じknowledge identityへ収束するなら:

```text
bounded input assembly
scope identity
cardinality delta
membership delta
retained provenance delta
```

もrouter mechanismに依存してはならない。

current BP-06ではexact / broad router双方から同じHuman bounded assemblyを導出し、router substitutionそのものをbefore / afterとしてR6Pへ渡した場合、全観測deltaは0でなければならない。

またBusiness Procedure Referenceへ同じtest-only fingerprint revisionを与えた場合、exact / broad双方のR6P decompositionは完全一致し:

```text
cardinality = 0
membership  = 0
retained provenance = 1
```

となることをPressureする。

consumer-local runtime Observationについても、broad routerがcandidate poolを広く見たことを理由にrouted knowledgeへ捕捉してはならず、runtime-only bounded assembly / observationはrouter invariantである。

```text
router mechanism
  != bounded input identity
  != scope identity
  != observation delta identity
  != authority
```

RoutingMechanism / over-selection / rejected candidateは診断情報に留め、R6P observation shapeへ持ち込まない。

R6QはR3AQ以降のAccepted Decision commit candidate / authority applicationを再実装しない。

#### R6R candidate-pool augmentation invariance — 2026-09-19

R6Qでrouter substitution後のdownstream invarianceを確認した次段として、repository / candidate pool自体が増えた場合でも、current taskに無関係なknowledgeがbounded inputへ侵入しないことをPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6r_candidate_pool_augmentation_invariance_pressure_test.go`

test-only unrelated candidate:

```text
kind = application-pattern
id   = test-only.unrelated_pattern
task hint = BP-02
```

をcurrent BP-06 candidate poolへ追加する。

exact task routerではtask hintが異なるためselected set自体が変化してはならない。

broad heterogeneous routerではcandidateを一旦selectedしてよいが、deterministic qualificationで:

```text
not-in-deterministic-task-scope
```

としてrejectし、qualified / disclosed knowledgeを変えてはならない。

その結果:

```text
candidate pool grows
  -> available diagnostics may grow

qualified knowledge
bounded input assembly
scope identity
qualified-not-materialized identities
R6P observation decomposition
  -> unchanged
```

を要求する。

broad routerについてもbaseline / augmented bounded assemblyをR6Pで比較し:

```text
cardinality delta = 0
membership delta  = 0
provenance delta  = 0
```

でなければならない。

CandidatePoolSize / RepositoryKnowledgeCount / RejectedCandidateCount等をbounded input / scope / observation identityへ持ち込まない。

```text
repository richness
  != working-context size
  != scope meaning
  != authority
```

R6RはKnowledge Portability relation、R3 authority acceptance、Gate C Evidence acquisitionを再実装しない。


#### R6 Source-pressure exit assessment — 2026-09-19

R6A〜R6RでSelective Knowledge RoutingのReference pressureは、Source / Design Evidenceとして一巡したと判断する。R6Rは無関係なcandidate pool growthがtask-scoped meaningへ漏れないことを確認する、具体的な最終closure pressureとして扱う。

成立した境界は次のとおり。

```text
available
  != selected
  != qualified
  != materialized

router mechanism
  != qualified meaning
  != bounded input identity

routed knowledge
  != consumer-local input

shared knowledge body
  != consumer-specific use edge

scope identity
  != observed cardinality

cardinality
  != membership
  != retained provenance

router diagnostics
  != downstream observation meaning
```

この段階でR6へさらにproduction router、ranking、embedding、model-native routing、Prompt payload、Universal Context Packet、freshness policy、Authorization authorityを追加しない。

またR6は、actual runtimeでのlatency / token / retrieval quality改善、AI Builderでのtask success、Jev等の非LLM認知層、model-native routerの優位性を実証していない。これらはSource pressureとは別Evidenceである。

R6からRequirementへ新しいnormative contractを自動昇格させない。現在の成果は既存の責務分離・traceability・deterministic qualification原則を具体化するReference Evidenceとして扱う。

次の再開条件は次のいずれかとする。

1. actual AI Builder / Agent contextでR6境界を破る具体的defectが観測された
2. First Web Target / Gate Cのactual EvidenceからR7 Replayable Behavior Evidenceへ進める
3. production context service / router候補が生まれ、R6 invariantsへの適合Pressureが必要になった

それまではR6を増殖させず、R3のfreshness / acceptance / execution authority、R7のbehavior Evidence、Gate Cのactual acquisitionへ責務を戻す。

### R7 — Replayable Behavior Evidence

目的:

- First Web TargetのFunctional Evidenceを、再実行可能な小さいbehavior単位でも説明できるか確認する

最低trace:

```text
Scenario / Semantic Operation
  -> behavior precondition
  -> Target interaction realization
  -> observed Result / Observation
  -> conformance result
```

期待:

- screenshot / DOM snapshotだけではFunctional successにしない
- replay用selector / coordinate等をCanonical identityにしない
- behaviorを複数回再実行して同じ意味上の結果を確認できる
- raw Target evidenceとnormalized evidenceの両方へ戻れる
- deeper cumulative workflowを単一behavior成功で代替しない

### R8 — Verified Experience Memory / Evidence Normalization

目的:

- Runtime経験を再利用しても第二正本やUniversal Event Modelを作らずに済むか確認する

最低条件:

- condition / action / outcome / failure / boundary candidateを元Execution / Observationへtraceできる
- build-input / Target / Driver / authority revisionを保持または再評価できる
- stale experienceを現在のruleへ無条件適用しない
- heterogeneous event normalizationをderived Evidenceとして扱う
- normalized eventからraw evidenceへ戻れる
- memory / normalized indexを削除してもCanonical Worldが変わらない


## 9. Requirementへ昇格する条件

研究の面白さだけでRequirementを増やしません。

新しいRequirement文または既存RQへのnormative追記は、少なくとも次を満たした場合に行います。

1. TEIのProduct Valueから必要性を説明できる
2. 既存RQの解釈だけではdefectを防げないことをReferenceで示せる
3. そのdefectがframework固有ではない
4. small Referenceで期待するfailure / successを再現できる
5. Requirementを追加した方が責務所有が明確になり、重複が減る
6. Gate / PAS / Evidence laneを不要に増殖させない
7. 一人で維持できる複雑性に収まる

現時点ではR2 Cross-Substrate Authorityが、RQ-019へのnormative追記に最も近い候補です。

## 10. 採用しない短絡

- AIが自然言語からCanonicalやRuntimeへ直接書き込むこと
- planner / modelが「許可されている」と判断したことをexecution authorityにすること
- Proposal acceptanceを将来のmutation authorityへ永久変換すること
- autonomous agentをTEIのCapability / Permission modelの代わりにすること
- agent frameworkごとに別action semanticsを作ること
- learned Procedural GraphをBusiness Procedure authorityへ昇格すること
- successful trajectoryを業務上の正しい手順とみなすこと
- screenshot / DOM / coordinate click列をCanonical interaction identityにすること
- AI向けKnowledgeをCanonical meaningの複製正本にすること
- simple / deterministic operationまで常時LLMへ委ねること
- Wails MCP control planeを自動的にApplication MCP Targetとみなすこと
- Wailsのorigin / token等をCanonical Application meaningへ持ち込むこと
- 第一Web TargetだけでTarget-neutral interaction taxonomyを完成扱いすること
- 新しいAgent Runtime、Workflow Engine、Universal Action Schemaを研究だけで先行採用すること
- 外部研究のsuccess rate、coverage、cost削減率をTEIの固定KPIへ転記すること

## 11. 開発順

現在のProduct主線は変えません。

```text
Gate B actual Evidence
  ↓
Handover B / R
  ↓
First Web Target / Gate C
  ↓
┌────────────────────┬────────────────────┐
↓                    ↓                    ↓
Studio minimal       knowledge-first /    Second Target
Authoring Loop       AI References         pressure
                     ↓
              R2 / R3 / R4 / R5
                     ↓
              Requirement再監査
                     ↓
              必要ならRQ refinement
```

Gate B環境待ちを理由にagent / cognitive subsystemを前倒し実装しません。一方、Referenceで何を検証するか、Requirementへ何を昇格しないかは先に明文化します。

## 12. 参考にした外部資料

9月前半:

- Weixing Zhang et al., *Large language models in model-driven engineering: a systematic mapping study*, Empirical Software Engineering, 2026
- Bartosz Balis et al., *From Research Question to Scientific Workflow: Leveraging Agentic AI for Science Automation*, 2026
- WorkflowGen: experience-driven adaptive workflow generation, 2026
- Vision2Web: functional workflow verification and visual fidelity evaluation for web generation, 2026
- *Augmenting software engineering with AI* / AI4SE taxonomy, 2026

2026-09-11短報で重点再評価:

- Yang Li et al., *Beyond Agent Harnesses: Cross-Substrate Authority for Multi-Agent Systems*, arXiv:2609.08472, 2026
- Suman Raj et al., *Avatar: Toward Autonomous End-to-End Orchestration of Scientific Workflows using LLMs*, arXiv:2609.10509, 2026
- Yuxing Lu et al., *Procedural Graphs: Self-Evolving Execution Structures for LLM Agents*, arXiv:2609.09153, 2026
- Xiaolei Li et al., *GraphDroid: Asynchronous LLM-Based Mobile App GUI Testing via History-Aware Exploration and Hybrid Intent Fulfillment*, arXiv:2609.10031, 2026
- Rui Xie, Lu Chen, *ASIL: Replacing Screenshot-and-Click with Structured State and Semantic Actions*, arXiv:2608.26991, Findings of EMNLP 2026
- Wails v3.0.0-beta.19 changelog, 2026-09-09


2026-09-18短報で重点再評価:

- Ruishuo Chen et al., *The Router Within: Eliciting Native Skill Routing from a Frozen LLM*, arXiv:2609.15982, 2026
- Jeonghye Kim et al., *ProgramDistill: From Interactive Web Apps to Verifiable Reference-Guided SWE Tasks*, arXiv:2609.18805, 2026
- Sibo Zhu et al., *RSIAgent: Autonomous Exploration for Recursive Self-improvement in New Environments*, arXiv:2609.15364, 2026
- Yunhao Feng et al., *HazardAuditor: From Executable Threats to Safer Computer-Use Agents*, arXiv:2609.15134, 2026
- Yifan Zhang et al., *Agora: Git as Shared Memory for Collective AutoResearch*, arXiv:2609.18094, 2026
- Wails v3.0.0-beta.20〜beta.23 changelog, 2026-09-10〜2026-09-16

外部資料の更新によって実装候補は変わり得ますが、この文書で抽出したTEI固有の境界はTEI自身のReference / Conformanceから再評価します。
