# Definition-Centered Feedback Loop Plan — 2026-09

Status: **planning / existing RQ-PAS-Gate composition / no new Gate / no new production schema**

## 1. 目的

`docs/definition-centered-feedback-loop.md`で定めた設計方向を、既存のStudio、knowledge-first exploration、AI Proposal、derived cognition、Reference Model、Driver / Observationへ段階的に接続する。

目的は「AI機能を一式実装すること」ではない。

```text
端的なauthoritative definition
  -> deterministic derivation / execution
  -> human / runtime / external feedback
  -> correct ownership routing
  -> candidate / preview
  -> shared Acceptance
  -> minimum accepted semantic delta
  -> reload / regeneration
```

この循環をReferenceから成立させることが目的である。

## 2. 上位Requirement / PASとの関係

新しい番号付きRequirementやPASを追加しない。

主に既存Requirementを組み合わせる。

- `RQ-001` — 業務意味と実装技術の分離
- `RQ-002` — Source-to-Canonical traceability
- `RQ-003` — 暗黙の複数正本を作らない
- `RQ-010` — 高密度Editor
- `RQ-011` — 複数Projection
- `RQ-012` — Driverによる設計検証
- `RQ-014` — Application Pattern再利用
- `RQ-015` — knowledge-first AI workflow
- `RQ-016` — Proposal / Decision分離
- `RQ-017` — 二重管理最小化
- `RQ-018` — Command / Observation分離
- `RQ-021` — Reference / Conformance
- `RQ-022` — 検証段階の明示

主なProduct Acceptance Scenario:

```text
PAS-01
  authoring / correction / Shared Acceptance

PAS-06
  existing knowledge exploration / reuse

PAS-07
  Data / Function / Business cross-view review
```

既存Gate B / Gate C Required Evidenceは変更しない。

## 3. 既存Issueの責務を一つのloopへ接続する

新しい巨大Issueを作らず、既存Issueを一つのReference flowで合流させる。

| Concern | Primary owner | Loopでの役割 |
| --- | --- | --- |
| local authoring UX | `ISSUE-000046` | Select / Preview / Accept-Reject |
| read-only knowledge exploration | `ISSUE-000047` | candidate discovery / qualification |
| AI-generated change candidate | `ISSUE-000027` | explanation / hypothesis / minimum semantic delta proposal |
| non-LLM candidate cognition | `ISSUE-000051` | associative recall / ranking / novelty signal |
| Business cross-view | `ISSUE-000042` / PAS-07 | Data-Function-Business contradiction pressure |
| actual Observation / Runtime boundary | existing Driver / RQ-018 work | actuality evidence |
| final authority | Shared Acceptance + authoritative source | accepted change only |

各Issueへ別々のKnowledge Store、Acceptance engine、definition formatを作らない。

## 4. 現行Referenceから確認するdefinition baseline

現在の`reference/workspace`には、すでに「短いauthoring + derived structure」の方向がある。

例としてSemantic definitionは、namespace / type / data / relation / capability等を必要な範囲で直接表し、Application、Database、Environment等は別concernのsourceへ分離されている。

このincrementでは新syntaxを設計しない。

まず次をinventoryする。

```text
A. human-authored irreducible meaning
B. mechanically derived meaning / projection
C. native external authoritative asset
D. realization / target / environment-specific detail
E. observed / runtime-only information
F. proposal / evidence / episodic information
```

目的は「すべてをもっと短いDSLへ押し込む」ことではなく、**現在どこに重複authoringやderived情報の手書きが残っているか**を把握することにある。

## 5. Reference Modelから継承する開発方法

過去Reference Model Gateでは、現実的な業務contentがPlatformへ具体修正を返しつつ、新しいtop-level Surface conceptを要求しない結果が得られていた。

今後も同じ方向を採る。

```text
Platform hypothesis
   ↓
realistic Reference content
   ↓ execute / inspect / cross-check
observed friction / contradiction / missing meaning
   ↓
ownership routing
   ├─ definition correction
   ├─ derived projection correction
   ├─ realization correction
   ├─ cognition correction
   └─ no change needed
   ↓
Platform / Reference both improve
```

### ReferenceにPlatform都合を押し付けない

Referenceに必要なmeaningが既存Surfaceで表せない場合、fixtureを不自然に変形してPASSさせない。

### Platformを一つのReferenceへ過適合させない

一つのReferenceだけが要求するdetailは、まずlocal authoring / Projection / Providerで表現できるか確認する。複数domainでdurable semantic needが観測されるまでCore kindを増やさない。

## 6. Phase 0 — Gate C前に行う設計準備

Product Gate C Readyまではproduction AI / Studio implementationへ進まない。

### P0-1 Definition inventory

現行`order_entry`を対象に、authoritative sourceとderived artifactを分類する。

Minimum output:

- Data / Relation / Validation / Capabilityのauthoring source
- Application Pattern / Query / Datasource source
- Binding / Realization source
- Business Procedure reference fixture
- Source Index / dependency / matrix等のderived artifact
- Runtime / Observation artifact

確認点:

```text
same meaningの手入力重複があるか
mechanically derived可能な情報をauthorしていないか
逆にderivedへ追い出すとmeaning lossになる情報は何か
```

このinventoryはschema変更を伴わない。

### P0-2 Feedback routing catalog

少なくとも次のfeedback classを例で整理する。

- semantic/business-definition defect
- cross-view contradiction
- realization/provider mismatch
- target/presentation problem
- actual-state drift
- runtime/resource failure
- retrieval/cognition miss
- unsupported inference / false completeness

category enumをproduction採用せず、Reference review vocabularyとして使う。

### P0-3 Memory separation audit

現在のSource Index、AI Proposal、Evidence、future cognitive indexを次へ分類する。

```text
authoritative semantic memory
derived associative / structural memory
episodic / experience memory
```

同じmeaningが複数memoryへauthorityとして重複していないか確認する。

### P0-4 X-TEA / XEAD pattern evidence handoff

過去に抽出したReference patternを、現行TEIの自動ruleへ変換せず、次の最低情報を持つderived candidate corpusとして扱う方法を整理する。

- pattern id / label
- provenance models / source
- observed vs candidate nature
- structural signature
- examples
- applicability question
- known non-proof

この段階ではpattern importerやvector indexを作らない。

## 7. Phase 1 — Gate C後の最初のclosed loop

最初に一つの小さい変更だけを通す。

### Scenario shape

Gate C後の`order_entry`から、実際にReference上価値のある小さいsemantic changeを一件選ぶ。

変更内容を先に固定しない。以下は候補にすぎない。

- missing Relation / requiredness
- Validationの不足
- existing Capability / Pattern参照の明示
- Business ProcedureとApplicationのcross-view contradiction修正

### Flow

```text
1. user / Reference task intent
       ↓
2. ISSUE-000047 read-only exploration
       ↓
3. exact / graph candidate + existing pattern knowledge
       ↓
4. applicable / incompatible / unknown
       ↓
5. Studio local context (ISSUE-000046)
       ↓
6. human or AI minimum semantic delta proposal (ISSUE-000027)
       ↓
7. Preview
     Semantic Diff
     impact / diagnostics
     persistence / target effect
       ↓
8. Accept / Reject
       ↓ Accept
9. authoritative source update
       ↓
10. reload / validate / conformance
       ↓
11. regenerate derived projections
       ↓
12. compare before/after Reference behavior
```

### P1 acceptance

- AIなしでも同じReference loopを実施できる
- Studioなしでもaccepted sourceをreloadできる
- ProposalやPreviewを削除してもaccepted meaningは残る
- one changeをwhole-model rewriteにせずreviewできる
- derived impactがexact dependencyへ追跡できる
- stale proposalを黙って適用しない
- correctionがvisible symptomではなくownership layerへ戻る

## 8. Phase 2 — Reference Pattern Memoryをcandidate sourceへ加える

Phase 1が成立した後、過去X-TEA/XEAD Reference Modelから抽出したpattern knowledgeをcandidate generationへ加える。

### 目的

LLMのpretraining記憶だけに依存せず、**どのReferenceでどの構造が観測されたかをprovenance付きで提示できる**ようにする。

### Flow

```text
current intent / local model context
  ↓
exact / graph search
  +
Reference Pattern Memory
  ↓
multiple candidates
  ├─ candidate A: exact existing asset
  ├─ candidate B: observed Reference pattern
  └─ candidate C: no safe match / unknown
  ↓
qualification / simulation / review
```

### 必須negative case

名称や構造が似ていてもBusiness requirementに合わないReference patternを用意し、候補上位でも`incompatible`へ落とせることを確認する。

### authority boundary

```text
Reference frequency
  != requirement

Reference pattern
  != accepted design

pattern match
  != semantic applicability
```

## 9. Phase 3 — Derived cognitionをcandidate generationへ比較導入

`ISSUE-000051`でexact / graph / lexical / weighted set / sparse expansive representation / HDC-VSA / embedding等を比較する。

このPhaseではfeedback loopのauthority構造を変えない。

### 評価対象

- candidate recall
- incompatible near-matchの混入
- novel caseでのforced nearest reuse
- latency / cost / locality
- explanation / provenance availability
- index rebuildability
- revision staleness

### Reference patternとの関係

Reference patternはlabelとして正解を与えるのではなく、構造類似探索のcorpus / evaluation caseとして使える。

例えば:

- header-detail構造
- conditional subtype / role
- self-reference hierarchy
- plan / actual / consequence / ledger
- composition / BOM
- derived summary

を、名前を変えたTEI-native caseでretrievalできるか比較する。

## 10. Phase 4 — Actual Observationをfeedbackへ接続する

Design-time closed loopが成立した後、Runtime / external actual stateからのfeedbackを接続する。

```text
accepted definition
  ↓ expected behavior/state
Runtime / external world
  ↓ Observation
compare
  ├─ supported
  ├─ contradiction / drift
  └─ unresolved
  ↓
route / proposal
```

Observationをsemantic sourceへ自動backfillしない。

### candidate cases

- Database desired vs actual drift
- Command accepted vs external completion
- generated UI interaction friction
- Capability execution error / resource denial
- Business Procedureで必要なactualityが観測不能

### feedback ownership

各caseで「どのsource/layerが直るべきか」をまず判定する。

## 11. Phase 5 — Derived Experience Knowledge / local learning

Phase 1〜4で十分なactual correction historyが得られた場合だけ、経験再利用を比較する。

### first candidates

- repeated accepted pattern candidate
- known false-positive suppression
- task-local association
- outcome-local retrieval weight
- context-specific ranking preference

### invariants

- Canonical / accepted Decisionはmodel weightだけへ保存しない
- source revision / dependency fingerprintへtraceできる
- learned stateは無効化・再評価できる
- stale experienceを現在revisionへ無条件適用しない
- learned rankingをAcceptanceへ昇格しない

Global online-learning architectureはこのPhaseでも必須ではない。

## 12. Editor / UX方向

将来Studioを高密度にする際、「項目数の多いform」ではなくlocal semantic workspaceを目指す。

一つのselected definitionについて:

```text
Authoritative definition
Exact neighborhood
Data / Function / Business lenses
Source / provenance
Derived impact
Observed actuality
Candidate patterns
AI explanation / proposal
Unresolved / contradiction
Semantic Diff
```

を同じcontextから辿れることが理想である。

ただし一画面へ全情報を常時表示することは要求しない。progressive disclosureとlocalityを保つ。

### AI UX

AI chatを主画面にしない。

AIは選択中のdefinition / local contextに対して:

- explain
- find analogues
- identify unresolved
- compare candidates
- propose minimum delta
- summarize impact

を行える。

利用者は最終的にsource / diff / evidenceを確認できる。

## 13. Definition authoringの改善仮説

Reference実験を通して次を評価する。

### H1 — derived defaultを減らさず、authored boilerplateを減らせるか

既知のstandard patternを自動導出し、明示override / business-specific meaningだけを書く。

### H2 — full file generationよりminimum deltaがreviewしやすいか

AIがwhole artifactを再生成した場合と、targeted semantic proposalをした場合を比較する。

### H3 — local contextでlarge modelを扱えるか

whole-model graphなしに、検索 + selected neighborhood + cross-view lensesで現実規模Referenceを扱えるか。

### H4 — Reference patternがLLM hallucinationを減らせるか

「過去Referenceに類例がある」ときだけprovenance付きcandidateを提示し、類例がない場合はunknownを維持できるか。

### H5 — cognitionがLLM callを代替すべき部分を特定できるか

exact/graph/associative retrievalで十分な処理と、LLM reasoningが必要な処理をReferenceから分けられるか。

これらは採用前提ではなくevaluation hypothesisである。

## 14. Feedback record

全feedbackを一つの巨大Event Storeへ保存することを要求しない。

Referenceで最低限追いたいrecord:

```text
feedback source
  human / reference review / simulator / runtime / external observation

base revision
selected / affected identities
observed problem or intent
candidate mechanisms used
candidate result
qualification result
semantic / realization / target / cognition routing
preview / diff
accept / reject
result revision
post-reload conformance
```

AI prompt / model / trajectoryは必要な評価Evidenceとして別管理できるが、Business meaningではない。

## 15. Reopen conditions

次の場合だけArchitecture / Requirementをreopenする。

- concise definitionでは必要meaningをlosslessに保持できない
- current Shared Acceptanceでは複数authoring sourceのcorrectionを安全に扱えない
- local neighborhoodだけではrealistic scaleで重要依存を見落とす
- derived pattern/cognitionのprovenanceを既存Source/Evidence構造で追えない
- actual Observationからcorrect ownership layerを決めるのにdurable semantic conceptが不足する
- learned associationを削除するとaccepted meaningまで失われる設計しか成立しない

最後のcaseは採用理由ではなく、derived memory設計が誤っている可能性を先に疑う。

## 16. 非目標

- Product Gate C前の本格AI / Studio implementation
- 新しいGate / PAS / Product Requirement
- `Cognitive Layer`の恒久Architecture化
- autonomous self-modifying system
- AIによるautomatic Accept
- AI chat中心IDE
- 全資産のTEI DSL化
- Reference Modelのcopy / compatibility implementation
- one universal vector space
- one global learned memory
- production feedback enum / event schemaの先行固定

## 17. 現在の作業順

```text
NOW / Gate C前
  1. design docsを正規化
  2. current definition / derived artifact inventory
  3. feedback routing examples
  4. X-TEA/XEAD pattern evidenceをderived candidateとして整理

Product Gate C Ready後
  5. ISSUE-000046 + 000047 + 000027でone-change closed loop
  6. Reference Pattern Memoryをcandidate sourceへ追加
  7. ISSUE-000051でnon-LLM cognition比較
  8. actual Observation feedback
  9. sufficient experience後のみlocal learning比較
```

順序は期限ではなくdependencyを表す。

## 18. 関連

- `docs/definition-centered-feedback-loop.md`
- `docs/architecture.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
- `plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`
