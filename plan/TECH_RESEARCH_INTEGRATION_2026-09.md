# TEI 技術リサーチ統合方針 — 2026-09

Status: **current research-integration plan**  
Last updated: **2026-09-11**

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

## 4. 2026-09-11短報から新たに深める知見

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

外部資料の更新によって実装候補は変わり得ますが、この文書で抽出したTEI固有の境界はTEI自身のReference / Conformanceから再評価します。
