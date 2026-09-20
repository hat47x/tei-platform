# Definition-Centered Feedback Loop Plan — 2026-09

Status: **planning baseline / Phase 0 static baseline complete / Gate C-dependent execution pending / no new Gate / no new production schema**

## 1. 目的

`docs/definition-centered-feedback-loop.md`で定めた設計方向を、既存のStudio、knowledge-first exploration、AI Proposal、derived cognition、Reference Model、Driver / Observationへ段階的に接続する。

目的は「AI機能を一式実装すること」ではない。

```text
端的なauthoritative definition
  -> deterministic derivation / execution
  -> human / runtime / external feedback
  -> correct ownership routing
  -> candidate / preview
  -> Shared Acceptance
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

## 4. 現行Referenceのdefinition baseline

`reference/workspace`には、すでに「短いauthoring + derived structure」の方向がある。

Semantic definitionはnamespace / type / data / relation / capability等を必要な範囲で直接表し、Application、Database、Environment等は別concernのsourceへ分離されている。

Phase 0では新syntaxを設計せず、現行Referenceを次へ分類して監査した。

```text
A. human-authored irreducible meaning
B. mechanically derived meaning / projection
C. native external authoritative asset
D. realization / target / environment-specific detail
E. observed / runtime-only information
F. proposal / evidence / episodic information
```

正本は`DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`とする。

主要結論:

- 広範な重複正本を解消するための大規模DSL再設計は現時点で不要
- 見かけ上の重複には、boundary mapping、Environment choice、promoted Release等、独立authorityとして必要なものがある
- `world.json`、Plugin Lock、Source Index、cross-view matrix等はderivedとして扱う
- native SQLはnative authorityを保つ
- ObservationをSemantic / Desiredへ自動backfillしない
- Business Procedure Reference fixtureをaccepted Canonical authoring量へ数えない
- AI時代の削減余地は、意味を消すよりsafe default / proposal / Previewで入力boilerplateを減らす側にある

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

Referenceに必要なmeaningが既存Surfaceで表せない場合、fixtureを不自然に変形してPASSさせない。

一方、一つのReferenceだけが要求するdetailは、まずlocal authoring / Projection / Providerで表現できるか確認する。複数domainでdurable semantic needが観測されるまでCore kindを増やさない。

## 6. Phase 0 — Gate C前の設計準備

**Status: static / planning baseline complete.**

Product Gate C Readyまではproduction AI / Studio implementationへ進まない。

Phase 0の完了はactual closed-loop Evidenceの取得を意味しない。cold reconstruction、cognitive index削除不変性、AI provider無しE2E、actual Observationからのcorrection loop等は後続Referenceで実証する。

### P0-1 Definition / Derived Artifact Inventory — Complete

正本:

- `DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`
- merged via PR #587

現行`order_entry`を中心に、authoritative source、native asset、realization、derived golden/index、observed fixture、Business Procedure Reference等をA〜Fへ分類した。

残るreduction候補は「今すぐ削除する項目」ではない。safe defaultやmechanical derivationがlosslessであることをReferenceで証明できた場合だけreopenする。

### P0-2 Feedback Routing Catalog — Complete

正本:

- `DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md`
- merged via PR #589

中心原則:

```text
feedback source != author
contradiction != patch instruction
unresolved != false
actual != desired
similarity != applicability
quality failure != requirement rewrite
```

FeedbackはまずEvidenceとして扱い、meaning-owning concernを解決してからcandidate / Preview / Shared Acceptanceへ進む。

### P0-3 Memory Separation Audit — Complete

正本:

- `DEFINITION_MEMORY_SEPARATION_AUDIT_2026-09.md`
- merged via PR #591

三種類のmemoryをauthority roleとして分離した。

```text
M1 authoritative accepted memory
M2 derived associative / structural memory
M3 episodic / experience memory
```

これは三つのDB / serviceを意味しない。

主要guard:

- M2を削除してもaccepted system meaningは変わらない
- M3のpast success / acceptanceをcurrent applicability / authorityへ昇格しない
- Proposal Accept後のmeaningはM1 owner sourceへ残す
- Source Index / embedding / sparse code / learned associationをhidden semantic authorityにしない
- promoted Releaseのようなtemporal authorityを単なるepisodeへ降格しない

### P0-4 X-TEA / XEAD Reference Pattern Handoff — Complete

正本:

- `REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md`
- merged via PR #593

Reference patternを自動ruleではなくprovenance-bound candidate knowledgeとして整理した。

```text
Reference observation
      ↓
M3 provenance evidence
      ↓ curate / abstract
Pattern evidence
      ↓ derive
M2 retrieval representation
      ↓ retrieve
current M1 qualification
      ↓
M3 Proposal / comparison
      ↓ Shared Acceptance only
M1 owner source
```

主要guard:

```text
Reference frequency != Product Requirement
pattern match        != semantic applicability
similarity           != accepted relation
retrieval rank       != Decision
candidate            != authority
```

Function viewはlosslessに可能ならView / read-CUD / Action-Capability / call / input-outputからderiveする。Task / Business ActionをApplication Functionへ畳まない。

## 7. Phase 1 — Gate C後の最初のclosed loop

**Status: In Progress — Gate C Ready済み。2026-09-21に決定論的なShared Acceptance loop（[`tools/reference-shared-acceptance`](../tools/reference-shared-acceptance/README.md)、Validation追加1件）を実装した。`ISSUE-000047`の探索・Studio UI・AI producerとの接続は未着手。**

最初に一つの小さい変更だけを通す。

### Scenario shape

Gate C後の`order_entry`から、実際にReference上価値のある小さいsemantic changeを一件選ぶ。

変更内容を先に固定しない。候補例:

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

**Status: Not Started — Phase 1 dependency.**

Phase 1が成立した後、P0-4でcurateしたpattern evidenceをcandidate generationへ加える。

目的は、LLMのpretraining記憶だけに依存せず、**どのReferenceでどの構造が観測されたかをprovenance付きで提示できる**ようにすることにある。

```text
current intent / local model context
  ↓
exact / graph search
  +
Reference Pattern Memory
  ↓
multiple candidates
  ├─ exact existing asset
  ├─ provenance-bound Reference pattern
  └─ no safe match / unknown
  ↓
qualification / simulation / review
```

名称や構造が似ていてもBusiness requirementに合わないnegative caseを必ず含め、候補上位でも`incompatible`へ落とせることを確認する。

## 9. Phase 3 — Derived cognitionをcandidate generationへ比較導入

**Status: Not Started — Phase 1 / 2 dependency.**

`ISSUE-000051`でexact / graph / lexical / weighted set / sparse expansive representation / HDC-VSA / embedding等を比較する。

このPhaseではfeedback loopのauthority構造を変えない。

評価対象:

- candidate recall
- incompatible near-matchの混入
- novel caseでのforced nearest reuse
- latency / cost / locality
- explanation / provenance availability
- index rebuildability
- revision staleness

Reference patternはlabelとして正解を与えるのではなく、構造類似探索のcorpus / evaluation caseとして使う。

## 10. Phase 4 — Actual Observationをfeedbackへ接続する

**Status: Not Started — design-time closed loop dependency.**

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

Candidate cases:

- Database desired vs actual drift
- Command accepted vs external completion
- generated UI interaction friction
- Capability execution error / resource denial
- Business Procedureで必要なactualityが観測不能

各caseで「どのsource / layerが直るべきか」をまず判定する。

## 11. Phase 5 — Derived Experience Knowledge / local learning

**Status: Not Started — sufficient actual correction history dependency.**

Phase 1〜4で十分なactual correction historyが得られた場合だけ、経験再利用を比較する。

First candidates:

- repeated accepted pattern candidate
- known false-positive suppression
- task-local association
- outcome-local retrieval weight
- context-specific ranking preference

Invariants:

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

AI chatを主画面にはしない。AIは選択中のdefinition / local contextに対してexplain、analogue探索、unresolved検出、candidate比較、minimum delta proposal、impact summary等を支援し、利用者は最終的にsource / diff / evidenceを確認できる。

## 13. Definition authoringの改善仮説

Reference実験を通して次を評価する。いずれも採用前提ではない。

### H1 — authored boilerplateを減らせるか

既知のstandard patternを安全に導出し、明示override / business-specific meaningだけを書く。

### H2 — minimum deltaはwhole artifact generationよりreviewしやすいか

AIによるfull regenerationとtargeted semantic proposalを比較する。

### H3 — local contextでrealistic scaleを扱えるか

whole-model graphなしに、検索 + selected neighborhood + cross-view lensesで重要依存を見落とさず扱えるか。

### H4 — provenance-bound Reference patternがunsupported generationを減らせるか

Reference類例がある場合だけ根拠付きcandidateを提示し、類例がない場合はunknownを維持できるか。

### H5 — LLM reasoningが必要な範囲を限定できるか

exact / graph / associative retrievalで十分な処理と、LLM reasoningが必要な処理をReferenceから分けられるか。

## 14. Feedback record

全feedbackを一つの巨大Event Storeへ保存することは要求しない。

Referenceで最低限追いたい情報:

```text
feedback source
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
- derived pattern / cognitionのprovenanceを既存Source / Evidence構造で追えない
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

Phase 0のstatic / planning baselineは完了した。

現在の直列Product主線は変更しない。

```text
NOW
  Gate B actual Evidence
    ↓ Ready only
  Handover B / R
    ↓
  First Target / Gate C

Gate C Ready後
  Phase 1: ISSUE-000046 + 000047 + 000027 one-change closed loop
    ↓
  Phase 2: provenance-bound Reference Pattern Memory
    ↓
  Phase 3: ISSUE-000051 non-LLM cognition comparison
    ↓
  Phase 4: actual Observation feedback
    ↓
  Phase 5: sufficient experience後のみlocal learning比較
```

Gate C前にDefinition-centered workstreamを理由とする新しいimplementation branchを増やさない。

Phase 0文書はcurrent baselineの保守だけを行い、P1以後の責務は既存owner Issueへ戻す。

## 18. 関連

- `docs/definition-centered-feedback-loop.md`
- `docs/architecture.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`
- `plan/DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md`
- `plan/DEFINITION_MEMORY_SEPARATION_AUDIT_2026-09.md`
- `plan/REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md`
- `plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
- `plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`
