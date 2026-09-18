# Actor Context Projections — AI Builder / Human Review / AI Business Actor

- Status: Phase 0 design baseline / read-only / no production schema adoption
- Date: 2026-09-18
- Parent: `plan/issue/000053-ai-native-development-and-agent-operational-knowledge.md`
- Related: ADR-0022 / ADR-0023 / ADR-0028 / RQ-010 / RQ-015 / RQ-018〜RQ-021 / RQ-025〜RQ-030
- Gate impact: None
- Runtime impact: None

## 1. この文書の役割

TEIを人間・AI Builder・AI Business Actorが共有して利用する場合でも、全主体へ同じ巨大なJSONやRepository dumpを渡すことは望ましくない。

一方、主体ごとに別のKnowledge Storeや第二の正本を作ることも望ましくない。

そこで、共通のauthority rootから用途別の**bounded derived projection**を作る。

```text
Authoritative Source / Canonical Meaning / Accepted Decision
                      │
                      ├── Builder Re-entry Context
                      ├── Human Review Projection
                      └── Operational Projection
```

三者は同じ意味へ戻れるが、purpose、必要情報、disclosure、freshness、authority limitが異なる。

本書はproduction `ActorContext` schemaを採用しない。Phase 0では、current TEI資産だけでどこまで構成できるかを整理する。

## 2. 既存資産からすでに使えるもの

current TEIには、actor-specific projectionを組むための材料が相当量存在する。

### Authoritative / accepted meaning

- Semantic Data / Relation / Validation / Capability
- Application / Application Pattern
- Database Contract / Persistence / Query
- Environment Binding
- Permission / Resource / Lease policy
- Business Procedure ReferenceでPressure中のBusiness Action / Handoff
- accepted authoring sourceへ戻るSource Index

### Derived navigation / impact

- `internal/index` のSemantic / declared dependency
- Application ↔ Database edge
- Capability Binding
- Environment / Plugin resolution
- Permission Grant / Resource Binding
- Query / Persistence / Release index
- Consumer Impact
- Design Synthesis cross-view proof
- DesignContextPacket concept

### Existing authority guard

- Source Index fingerprint
- Proposal / Shared Acceptance Boundary
- Command / Observation separation
- Permission / Resource boundary
- Reference / Conformance
- UNKNOWN / unresolved preservation

したがってPhase 0の中心課題は、新しい知識層を増やすことではなく、**既存の意味をactorのtaskへ必要十分に切り出すこと**である。

## 3. 共通Projection原則

Builder / Human / Business Actorの各Projectionは、少なくとも次の性質を共有する。

### 3.1 Derivedである

```text
projection deleted
  -> accepted meaning remains

projection regenerated
  -> authority root is recoverable
```

Projection自身をCanonical source、Decision authority、Business factにしない。

### 3.2 Purpose-boundである

ContextはRepository全体の一般的要約ではなく、明示されたtask / review / Actionのために選択する。

例:

- 「order status Validationを変更する」
- 「このAI変更を承認できるか確認する」
- 「この注文をsubmitする前に必要条件を確認する」

purposeが変わればcontext選択も変わる。

### 3.3 Snapshot / freshnessへ束縛する

少なくとも、

- authority ref
- source / revision / fingerprint
- derived input revision

へ戻れる必要がある。

staleなcontextをcurrent authorityとして利用しない。

### 3.4 Omissionを隠さない

bounded contextである以上、含まれない情報がある。

「このpacketにない = 存在しない」としない。

最低限、

- selected scope
- omitted / unresolved category
- drill-down path

を説明できる必要がある。

### 3.5 Authority limitを明示する

```text
context says X
  != context authorizes X
```

特にAI Business Actorではreadabilityとinvocation authorityを分離する。

## 4. 三つのProjection family

### 4.1 Builder Re-entry Context

目的:

> 別session / 別model / 別agentがApplication開発へ再入場し、Repository全体の自由探索から始めず安全に変更対象へ到達する。

必要情報の候補:

- task / change intent
- relevant Canonical subjects
- authoritative source refs
- dependencies / consumers
- accepted Decision / Variation
- Validation / Policy
- Capability / Binding
- native asset / custom code touchpoint
- derived / generated artifact
- unresolved concern
- current Conformance
- likely impact surface
- drill-down source

### DesignContextPacketとの関係

既存Design Synthesisの`DesignContextPacket`は、このfamilyの**特定用途の一Projection**として扱う。

```text
Builder Re-entry Context family
  ├── Design Synthesis Context Packet
  ├── migration context
  ├── defect correction context
  └── provider / target change context
```

`DesignContextPacket`を全AI利用の万能schemaへ昇格させない。

### 4.2 Human Review Projection

目的:

> 人間がgenerated code全量を読む前に、変更の意味と見るべき箇所を把握する。

必要情報の候補:

- semantic diff
- affected Canonical subjects
- impacted Application / Database / Procedure
- authority / permission delta
- unresolved / exception
- native code touchpoint
- generated artifact delta
- Validation / Conformance result
- Evidence / provenance
- source drill-down

Human Review Projectionは「AIの説明文」だけではない。

同じCanonical ID / Source / Evidenceへ戻れる必要がある。

### 4.3 Operational Projection

目的:

> AI Business Actorが、現在の業務Actionについて「何が必要で、どこまで自分が実行でき、何を確認して次へ進むべきか」を理解する。

これはWorkflow Engineのexecution planではない。

Business Procedureが人間・外部組織・物理作業・unresolved judgementを含むことを保ったまま、現在ActionのOperational Knowledgeをboundedに提示する。

## 5. Operational Projectionの概念形

production schemaではない。情報損失を検証するためのconceptual decompositionである。

```text
OperationalProjection

  purpose
  environment / execution context

  procedure_context
    procedure ref
    current action / step ref
    purpose / trigger context
    actor context

  action_meaning
    business action
    realization class
    semantic / application refs
    validations / constraints
    policy refs

  capability_candidates[]
    semantic capability ref
    application pattern / action ref
    binding ref
    provider / runtime ref

  authority_requirements[]
    authorization requirement
    permission requirement
    resource requirement

  current_runtime_resolution
    binding resolution
    permission evaluation
    resource availability
    provider availability

  expected_result
    business result shape
    command acknowledgement semantics

  observation_obligations[]
    what must be observed
    evidence requirement
    unknown condition

  continuation
    next known action candidates
    human handoff
    external handoff
    unresolved continuation

  provenance
    canonical refs
    source refs
    evidence refs
    snapshot / fingerprint

  authority_limits
  unresolved[]
```

この形からproduction fieldを確定しない。

## 6. 「実行可能」を一つのbooleanへ潰さない

AI Agentにとって危険なのは、

```text
executable = true
```

という一値で多数の異なる境界を隠すことである。

少なくとも次は分ける。

### L0 — Defined

Business Actionがaccepted knowledge上に定義されている。

```text
action exists
  != machine executable
```

### L1 — Resolvable

ActionがApplication Pattern / Semantic Capability / external realization等へexactに辿れる。

```text
resolved
  != bound
```

### L2 — Bound

current environmentでCapability / implementation / Provider Bindingが解決できる。

```text
bound
  != authorized
```

### L3 — Authorized

current actor / principal / policyについて必要Authorityが成立する。

```text
authorized
  != resources available
```

### L4 — Resource-ready

必要なsecret / endpoint / filesystem / external resource等をResource boundaryから利用可能である。

secret valueをProjectionへ埋め込むことは要求しない。

```text
resource-ready
  != command accepted
```

### L5 — Invokable

Side effect、idempotency、stale base、lease / ownership等のpreconditionを満たし、invokeを試みてよい。

```text
invokable
  != executed
```

### L6 — Command acknowledged

Provider / external systemがcommandを受理した。

```text
command acknowledged
  != actual state observed
```

### L7 — Observed

必要なObservation / Evidenceが得られた。

```text
observed
  != business completed
```

### L8 — Business continuation satisfied

accepted Business meaning上、次へ進んでよい条件が満たされた。

このL0〜L8をproduction enumにすることはまだ決めない。

重要なのは、これらを一つのAgent confidenceや`canExecute`へ畳まないことである。

## 7. order_entryへ当てた場合

current Business Procedure Referenceを使うと、少なくとも次の差を保持できる。

### BP-00〜BP-03

Application Patternを介したinspection / preparation。

Agentが利用できるかは、

- Patternが存在する
- Data / Queryへtraceできる
- current Target / CapabilityがAgentから利用可能
- Authorization / Resourceが成立

まで確認して初めてOperationalに判断する。

### BP-04

`manual-unresolved`

```text
"decide whether the order is ready"
```

を、LLMが賢いという理由だけで自動実行可能Actionへ昇格させない。

将来AI委任可能になる場合は、Decision authority / policyのEvidenceが必要。

### BP-05

accepted Validation authorityへの参照がある。

AgentはValidationを実行・参照できる余地があるが、

```text
validation passed
  != ready to submit in surrounding business context
```

である。

BP-04とBP-05を同一化しない。

### BP-06

`security.authorize` Capabilityへexplicit refがある。

これはAgentにとって重要だが、

```text
authorization capability exists
  != authorization result allow
```

である。

protected target overlayはcurrent Reference pressureであり、production authorityではない点も失わない。

### BP-07

accepted application commandをrequestするAction。

ここが将来のside-effect Agent Reference候補になり得る。

ただしBP-06 authorization、Binding、Resource、command semantics、retry / ownershipを満たす前にinvokeしない。

### BP-08

Observation boundary。

BP-07 command acceptanceをactual completionへ短絡しないため、Agent Operational Projection上で非常に重要である。

### BP-09

`manual-external-unresolved`

Application interaction後の業務継続を、Application successだけでauto-completeしない。

## 8. AI Business ActorのplanningとTEI Authorityを分離する

AgentはOperational Projectionから、次のようなplanを作ってよい。

```text
1. recent orderを特定
2. order / customer / linesを確認
3. constraintsを評価
4. submit readinessの判断を得る
5. authorize
6. submit
7. observe
8. continue / handoff
```

しかしplan自体はTEI authorityではない。

```text
Agent Plan
  != Business Procedure
  != Authorization
  != Capability Binding
  != Business completion
```

Agentがplanを最適化・省略・並べ替える場合、それがaccepted Procedure meaningへ抵触するなら別途Proposal / Acceptanceまたは明示Policyが必要になる。

## 9. SEIとの接続

TEI Operational ProjectionへSEIの認知・判断文脈をoptionalに接続できる。

例:

```text
TEI
  current action = BP-04
  authority = human / policy dependent
  unresolved = submit readiness judgement

SEI
  relevant observations
  unknowns
  evidence
  recommendation
  decision history
```

ただし、

```text
SEI Recommendation
  != TEI Authorization
  != Business completion
```

を維持する。

SEIなしでdeterministic / policy-driven Actionを実行する経路も成立する。

## 10. Human ProjectionとOperational Projectionは同じではない

Human Reviewでは、

- なぜ変更されたか
- 何に影響するか
- 何が例外か
- どこを承認すべきか

が重要である。

Operational Agentでは、

- 今どのActionか
- 何が必要か
- 何が許可されているか
- 何をinvokeできるか
- 何をObservationすべきか
- どこでhandoffすべきか

が重要である。

同じCanonical rootでもselection / disclosureが異なる。

したがって一つの万能`ContextPacket`へ統一しない。

## 11. Knowledge portabilityとの関係

Actor Context Projectionはknowledge portabilityの正本ではない。

ただし、別Applicationへ知識を移植するとき、

- origin authority
- assumptions
- dependencies
- provider / target constraints
- accepted variations
- unresolved

をbounded contextへ出せることは有益である。

これによりAI Builderはcopy-pasteではなく、**何が移植可能で、何がdestination-specificか**を比較できる。

再利用時に採用されたmeaningはdestination側のappropriate authorityへAcceptanceする。

## 12. Phase 0で新設しないもの

- production `ActorContext` schema
- production `OperationalProjection` schema
- Agent-specific Canonical kind
- Agent RoleをAuthorization Roleへ写す規則
- global `canExecute` state
- Agent planner
- workflow scheduler
- Agent memory database
- hidden chain-of-thought archive
- Agent専用Capability system
- Agent専用Permission bypass
- Business Procedureのproduction authority前倒し

## 13. Phase 0 findings

### F1 — AI Builder contextの土台はすでにある

Design Synthesis Context Packet、Source Index、dependency / impact等を組み合わせれば、新Knowledge StoreなしでBuilder contextをPressureできる。

### F2 — DesignContextPacketは万能化しない方がよい

Design Synthesisではproof / candidate / unresolvedが重要だが、Operational AgentではAuthorization / Resource / Observationが中心になる。

用途を分ける方がauthority leakageを防ぎやすい。

### F3 — Agent Operational ProjectionはBusiness Procedureだけでは足りない

Procedureは「何の仕事か」を与えるが、actual invocationには、

- Capability
- Binding
- Permission
- Resource
- Environment
- Observation

が必要である。

逆にこれらRuntime情報だけでもBusiness purpose / handoff / completionは分からない。

したがってOperational Projectionはcross-view derived projectionになる。

### F4 — current Business Procedure Referenceはread-only Agent pressureに使える

BP-00〜BP-03 / BP-05 / BP-06 / BP-08は、side effectを起こさず、

- Procedure理解
- Pattern / Validation / Capability trace
- Authorization requirement
- Observation boundary
- human / unresolved handoff

をPressureできる。

### F5 — side-effect Referenceを急がない

BP-07は将来有力だが、Business Procedure production authority、Capability invocation、Authorization / Resource / retry境界を先に満たす必要がある。

## 14. 次のReference

ISSUE-000053 Phase 1〜3では次の順を推奨する。

### R0 — Static actor context derivation

current `order_entry`から、

- Builder context
- Human Review context
- read-only Operational context

を同じaccepted snapshotからderivedに作り、三者の情報差を比較する。

### R1 — Builder re-entry challenge

小さいsemantic changeを一件選び、Repository free explorationとstructured context先行を比較する。

### R2 — Human review challenge

R1の同じ変更からsemantic diff / impact / Evidenceだけでfirst-pass reviewできるか確認する。

### R3 — read-only Business Actor

BP-00〜BP-06 / BP-08を使い、

- machine-readable Action selection
- exact trace
- unresolved preservation
- unauthorized invocation prevention
- human handoff

をPressureする。

### R4 — side-effect

R3と必要Authorityが成立した後のみBP-07等で検討する。

## 15. 成功の見方

TEIがAgent向けに大量のtextを出せることは成功ではない。

価値があるのは、

1. 同じaccepted meaningから用途別contextを再生成できる
2. contextからexact authorityへ戻れる
3. Agentが「分からない」「権限がない」「人へ渡す」を保持できる
4. readとexecuteを混同しない
5. commandとactual completionを混同しない
6. Human / Builder / Agentで第二の正本を増やさない
7. contextを削除してもApplication / Business meaningが失われない

ことである。
