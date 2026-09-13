# Reference Pattern Evidence Handoff — 2026-09

Status: **P0-4 baseline / research-to-derived-memory handoff / no Pattern authority adoption**

Baseline: `main` after P0-3 Memory Separation Audit

Related:

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/DEFINITION_MEMORY_SEPARATION_AUDIT_2026-09.md`
- `plan/DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
- `plan/design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`

Historical research inputs reviewed for this handoff:

- `xtea_pattern_mining_report_v0_1.md`
- `xtea_research_checkpoint_v0_2.md`
- `XTEA_REFERENCE_MODEL_VITALITY_AUDIT_2026-08-21.md`
- `REFERENCE_MODEL_GATE_PLAN_v1.md`

These historical research artifacts are evidence inputs, not current TEI authority.

## 1. 目的

Definition-centered feedback loopのP0-4として、X-TEA / XEAD / TEI Referenceから得た反復構造や設計上の観測を、生成AI・非LLM cognition・Studioが再利用できる**provenance-bound candidate knowledge**へ引き渡すための境界を整理する。

目的はPattern Libraryを新しい正本として作ることではない。

```text
Reference observation
      ↓
M3 provenance evidence
      ↓ abstract / curate
Pattern evidence card
      ↓ derive
M2 retrieval representation
      ↓ retrieve
current-context candidate
      ↓
M1 exact qualification
      ↓
M3 Proposal / comparison
      ↓
Shared Acceptance
      ↓ accepted only
M1 owner source
```

Reference patternがこの経路を飛び越えてM1へ入ることを禁止する。

## 2. Pattern evidenceとPattern authorityを分ける

Reference Modelに同じ構造が複数回現れることは有益なEvidenceである。

しかし:

```text
observed repeatedly
  != current requirement

frequent
  != universally good

structurally similar
  != semantically applicable

historically successful
  != currently accepted
```

したがってPattern evidenceは次の役割に限定する。

- 候補構造を思い出す
- 比較対象を増やす
- applicability questionを提示する
- simulator / Referenceで検証する出発点を作る

自動採用、schema生成、Core kind追加の根拠にはしない。

## 3. 研究Evidenceの種類を分ける

過去研究には性質の異なる情報が混在する。

### E0 — Corpus metadata

例:

- 12 XEA[D] Reference Models
- Relationship counts
- lexical indicator counts

役割:

corpusの規模・抽出背景を説明する。

Patternの正しさそのものではない。

### E1 — Observed structural pattern

複数Referenceで、研究者の高次解釈を最小化して直接的に反復構造を確認したもの。

例:

- header-detail transaction
- conditional subtype / role
- self-reference hierarchy / chain

### E2 — Candidate synthesized pattern

複数観測をまとめて、再利用可能な上位構造として研究者が仮説化したもの。

例:

- plan / actual / consequence / ledger
- derived summary projection
- composition / BOM / bundle

出現モデル数が多くてもE1へ自動昇格しない。

### E3 — Architecture / boundary evidence

Pattern候補ではなく、TEIの設計境界を支持・圧力試験するEvidence。

例:

- Business ActionとApplication Functionは一致しない
- 三要素を別視点としてcross-checkする
- Data-firstはData-onlyではない
- Reference ModelはPlatformも改善する

これをPattern retrieval corpusへ無条件に混ぜない。

### E4 — Example / simulation evidence

具体instance、snapshot、scenario、state transition等。

Pattern applicabilityを試すcaseには使えるが、例そのものを一般ruleへしない。

### E5 — Limitation / provenance constraint

- source access条件
- extraction方式
- parser / database未読範囲
- lexical heuristic
- license / redistribution制約
- copied contentではなく独自抽象化であること

これを欠いたPattern Cardをhigh-confidenceに見せない。

## 4. 過去Pattern Miningの確認結果

一次Pattern索引は12個のXEA[D] Reference Modelを対象に機械抽出されている。

Corpus-level indicators:

```text
Relationship
  FAMILY   348
  REFFER   438
  SUBTYPE   60

name-based primary indicators
  header_or_detail          117
  summary                    90
  instruction_or_application 62
  history                    23
  plan                       22
```

これらは**coarse indicator**であり、個別PatternのOccurrence数と同じではない。

たとえば`summary = 90`という名称指標から`derived_summary_projection`が90件存在すると解釈してはいけない。

## 5. 初期Pattern Evidence Set

過去一次索引から、最初のcurated setとして7件を引き渡せる。

### P-01 `header_detail_transaction`

Nature: **observed**

Historical statement:

```text
一つの業務取引・申請・指示を、見出しと複数明細に分けて保持する
```

Evidence:

- observed source models: 8 / 12

TEI-local structural signature候補:

```text
transaction-root Data
  -> one-to-many relation
  -> detail Data

root owns transaction-level facts
detail owns repeated line-level facts
```

Applicability questions:

- 子要素は親取引の一部として同じlifecycle / ownershipに属するか
- line identity / orderingは何を意味するか
- detailが独立したBusiness aggregateではないか

Contraindication:

子が独立した責任・release・lifecycleを持つなら、単純header-detailへ圧縮しない。

TEI mapping:

既存Data / Relation / Application Patternでまず表現し、新しい`HeaderDetail` Canonical kindを要求しない。

### P-02 `conditional_subtype_role`

Nature: **observed**

Historical statement:

```text
基本主体に対し、資格・役割・取引区分などの条件成立時だけ追加属性を持たせる
```

Evidence:

- observed source models: 8 / 12

Structural signature候補:

```text
base subject
  + conditional role / subtype-specific facts
```

Applicability questions:

- 差分は恒久的typeか、一時的roleか
- lifecycle stateをsubtypeと誤認していないか
- nullable fieldsだけで十分か、別Data / Relationが必要か

Contraindication:

単なる状態遷移やauthorization roleをData subtypeへしない。

### P-03 `self_referential_hierarchy_or_chain`

Nature: **observed**

Historical statement:

```text
同種record間の階層または連鎖
```

Evidence:

- observed source models: 4 / 12

Structural signature候補:

```text
Data X
  -> relation
  -> Data X
```

Applicability questions:

- hierarchyかpredecessor chainか
- cycleを許すか
- ordering / depthがBusiness meaningか

Contraindication:

似た名称の別概念を無理にself relationへしない。

### P-04 `plan_actual_consequence_ledger`

Nature: **candidate**

Historical statement:

```text
予定・実績・残高 / 履歴を分離し、業務事実の成立による結果を追跡する
```

Evidence:

- supporting source models: 10 / 12

重要:

**10 / 12でもcandidateである。**

個別構造の反復から高次に統合された研究仮説なので、frequencyで`observed`へ昇格しない。

Applicability questions:

- planとactualは同一identityのstate差か、別のBusiness factか
- consequenceは再計算可能か、immutable historyとして保持すべきか
- ledgerは集計projectionか、監査上の事実か
- cancellation / correctionをどう表すか

TEI mapping:

まずData / lifecycle / Observation / derived projectionで分解して検討する。

`Ledger`や`PlanActual`を新Canonical kindにしない。

### P-05 `derived_summary_projection`

Nature: **candidate**

Historical statement:

```text
明細事実から期間・主体・品目等の軸で派生集計を持つ
```

Evidence:

- supporting source models: 10 / 12

Applicability questions:

- summaryはcurrent factsから再生成可能か
- performance cacheか、業務上保存すべきsnapshotか
- historical closing / auditのためfreezeが必要か

Default TEI pressure:

まずQuery / View / derived projectionを検討する。

```text
can derive
  -> do not automatically author duplicate Data
```

ただしclosing snapshot等の独立Business factならM1候補になり得る。

### P-06 `composition_bom_bundle`

Nature: **candidate**

Historical statement:

```text
製品・サービス・組織などを構成要素の集合として表し、展開や影響範囲を扱う
```

Evidence:

- supporting source models: 6 / 12

Applicability questions:

- compositionか単なるassociationか
- quantity / effective period / versionを持つか
- recursive expansionを許すか
- shared componentを許すか

Contraindication:

containmentを所有権・lifecycleまで意味するものとして自動解釈しない。

### P-07 `example_instance_snapshot`

Nature: **observed**

Historical statement:

```text
具体的な業務世界の例をmodel上へ置き、複数の表示状態を使って理解を助ける
```

Evidence:

- observed source models: 4 / 12

後続研究による補正:

旧`Instance / Slide / SortKey`は、あるmodelでは業務状態変化を表した一方、別modelでは単なる段階的説明順にも使われていた。

したがってPatternを次へ分解して扱う。

```text
Dataset
Snapshot
Scenario
Expectation
Presentation Order
```

`SortKey = time`とはみなさない。

この補正は、Pattern Cardに**negative / ambiguity evidence**も必要であることを示す。

## 6. FrequencyとNatureは別軸

初期setから明確に分かる。

| Pattern | Nature | supporting models |
| --- | --- | ---: |
| header_detail_transaction | observed | 8 |
| conditional_subtype_role | observed | 8 |
| self_referential_hierarchy_or_chain | observed | 4 |
| plan_actual_consequence_ledger | candidate | 10 |
| derived_summary_projection | candidate | 10 |
| composition_bom_bundle | candidate | 6 |
| example_instance_snapshot | observed | 4 |

したがって:

```text
model_count != confidence
model_count != applicability
model_count != authority
```

`Nature`は抽出・解釈の性質、`model_count`はcorpus内のsupport breadthを表す別情報とする。

## 7. Corpus denominatorも過信しない

`8 / 12`等は、このhistorical corpus内でのsupport breadthである。

それを:

```text
67%の業務システムに適用できる
```

のように一般化しない。

理由:

- corpusが無作為標本ではない
- model間に思想・lineageの共通性がある可能性
- extraction粒度がmodel単位
- business domainの偏り
- source completeness差

分母12はprovenance contextであり、統計的一般化の母集団ではない。

## 8. Pattern Handoff Record — conceptual minimum

production schemaではなく、P0-4で情報損失を避けるためのconceptual recordを整理する。

```text
Pattern Evidence Handoff
  pattern_key                 TEI-local stable research key
  nature                      observed | candidate
  concise_statement
  structural_signature

  provenance
    source_corpus
    source_count
    corpus_size
    extraction_method
    research_artifact_refs
    evidence_date / version
    license_or_access_note

  applicability_questions[]
  contraindications[]
  ambiguity_or_negative_evidence[]
  known_non_proof[]
  unresolved[]

  example_strategy
    TEI-native synthetic / independently designed example preferred

  TEI_translation
    existing concepts first
    possible derived projection
    what must not become a new Canonical kind by default
```

これをそのままJSON schemaとして採用しない。

Gate C後のReferenceで、どのfieldが実際に必要かpressureする。

## 9. `pattern_key`のauthority

`header_detail_transaction`等のkeyはresearch / retrieval identityとして使える。

しかし:

```text
Pattern key != Canonical business identity
Pattern key != Application Pattern kind
Pattern key != Requirement ID
```

Pattern keyを使用しても、current modelへ自動的に同名conceptを作らない。

## 10. Provenanceの最小要求

Pattern candidateをAI / cognitionへ渡すなら、少なくとも次へ戻れる必要がある。

```text
どのresearch corpusか
どのresearch artifact / versionか
observedかcandidateか
何モデルがsupportしたか
どの抽出方法か
どの limitationがあるか
```

source modelのraw contentを再配布する必要はない。

むしろlicense / accessが不確実な場合:

```text
raw model content
  -X-> Pattern Memoryへcopy

abstract research observation
  -> provenance note付きでhandoff
```

を基本にする。

## 11. License / source boundary

過去Reference Gateでも、原modelの利用条件が明確になるまでcontentをcopy / redistributeせず、公開資料から一般的構造を参考に独自Referenceを作る方針だった。

P0-4でも維持する。

### Store / distributeしてよい方向

- abstract structural statement
- aggregate counts
- extraction method note
- public source locator where appropriate
- independently designed TEI-native example
- synthetic Simulation Dataset

### 慎重に扱う方向

- original model file
- large raw table/field list
- proprietary sample data
- exact source textの大量転記

Pattern Memoryをsource archiveの代替にしない。

## 12. Pattern ObservationとRetrieval Representationを分ける

P0-3のmemory separationを適用する。

### M3 — Pattern Observation / Provenance

保持する:

- nature
- concise finding
- support breadth
- source / method / limitation
- negative evidence
- research decision history

### M2 — Retrieval Representation

M3 + current accepted M1から再生成できる:

- lexical tokens
- structural features
- graph signature
- weighted set
- sparse code
- HDC representation
- embedding
- candidate rank

### M1 — Current Design

Patternを採用する場合でも、current requirementに必要なmeaningだけをcurrent owner sourceへ書く。

Pattern Card自体をM1へpromoteしない。

## 13. Structural signatureはraw schema copyにしない

例えばheader-detailなら:

```text
root
  relation one-to-many
  detail
```

程度のrole-aware abstractionを持てばよい。

原Referenceのtable名・field名をそのままfeature vectorへ入れる必要はない。

利点:

- license / source boundaryを守りやすい
- naming差を越えて構造類似を探せる
- current TEI identityと混同しない

## 14. Reference PatternからTEIへ翻訳するときの順序

```text
1. current Business intentを確認
2. exact existing TEI assetを探索
3. Reference Patternを複数候補として取得
4. applicability questionsをcurrent M1へ照合
5. incompatible / unknownを落とさず保持
6. existing TEI conceptsで表現できるか確認
7. TEI-native Reference / Simulationでpressure
8. 必要ならProposal
9. Shared Acceptance
```

Pattern retrievalをstep 1へ置かない。

Current requirementがPatternへ従属するのを避ける。

## 15. Pattern Cardはsolution recipeではない

Pattern Cardへprocedure codeや完全schemaを詰め込みすぎない。

最低限:

- structural insight
- applicability conditions
- tradeoff / contra-indication
- provenance
- example / simulation strategy

を持つ。

実装recipeが必要なら、Patternとは別のReference / Component / Capability knowledgeとして扱う。

## 16. Architecture evidenceをPattern Cardへ混ぜない

後続研究では:

- 12 XEA[D]のTaskAction 1606件
- Application Function realizationを持たないAction 202件

という観測があり、

```text
Business Action != Application Function
```

を強く支持する。

これは重要だが、`business_action_without_function`という自動適用Patternを作る必要はない。

**E3 Architecture evidence**として:

- Business Actionをfake Functionへ変換しない
- Business ProcedureをApplication flowへ縮約しない

という境界の根拠に使う。

## 17. State research evidenceも別枠で扱う

後続研究では9モデル / 902 Table Scriptの再走査から:

- 状態 / 状況名称のbusiness VIRTUAL field: 30
- その30すべてに算定Script
- high-confidence lifecycle Script: 31
- empty ↔ setを直接判定するtransition shape: 8

等が観測された。

これはState / Transition / Consequenceを検討する強いresearch seedだが、P0-4の初期7 Patternと同じ成熟度とは限らない。

当面:

```text
State research observation
  -> E3/E4 research evidence
  -> dedicated TEI Reference pressure
  -> only then Pattern candidate or Canonical decision
```

とする。

`StateMachine` Canonical kindをこの数字だけで追加しない。

## 18. Example Instance研究の教訓

旧Instanceの`SortKey`が:

- 時系列的な業務状態変化
- 人間向けの段階的説明順

の両方に使われていたことは重要である。

Pattern miningでは、同じfield / mechanismが複数意味に使われている可能性を記録する必要がある。

Handoff recordに:

```text
ambiguity_or_negative_evidence
```

を持たせる理由である。

AIにpositive exampleだけを渡すと、`SortKey = chronology`のようなfalse generalizationが起きる。

## 19. Example / Simulation Dataset

Pattern applicabilityを確認する際、原Referenceのsample dataをcopyすることを標準にしない。

current requirementに合わせた小さいTEI-native datasetを独立作成する。

例:

### header-detail

```text
Order O1
  Line 1
  Line 2
```

だけでは不十分。

negative caseとして:

```text
child has independent lifecycle
```

も用意し、header-detailへ潰すべきでないことを確認する。

### plan / actual / consequence

- partial execution
- cancellation
- correction
- derived balance

を含むscenarioで、単一status / single recordへ圧縮してよいか検証する。

## 20. Negative exampleを第一級にする

Pattern Memoryの価値は「適用例」を増やすことだけではない。

```text
when NOT to use
```

を持つことで、LLM / cognitionのnear-match誤用を抑える。

初期Referenceでは各Patternに少なくとも一つ:

- compatible case
- incompatible near-match

を用意するのが望ましい。

## 21. Retrieval feature family

P0-4ではalgorithmを決めず、Pattern Cardからderive可能なfeature familyだけ整理する。

### F-P1 structure

- relation shape
- same-kind / cross-kind
- cardinality
- repetition / composition role

### F-P2 lifecycle / temporal

- plan / actual separation
- historical / current
- transition / consequence presence

### F-P3 responsibility / ownership

- same aggregate / independent aggregate
- owner boundary
- Business responsibility

### F-P4 projection nature

- authored fact
- derived summary
- snapshot
- presentation-only order

### F-P5 applicability questions

boolean featureとして答えを埋めず、current M1 qualificationへ必要なquestion setとして保持する。

M2 encoderが質問への回答を勝手に推論しない。

## 22. Cognitive retrievalへのhandoff

`ISSUE-000051`ではPattern Cardを次のように利用できる。

```text
current local TEI structure
       ↓ feature extraction
candidate field
       ↓
Pattern retrieval representation M2
       ↓
P-01 / P-03 / P-06 ... candidates
       ↓
M3 provenance cardをattach
       ↓
ISSUE-000047 deterministic qualification
```

評価する:

- exact / graphだけでは拾えない構造candidateを回収できるか
- incompatible near-matchを後段で落とせるか
- novel caseでforced nearestを避けられるか

## 23. LLMへのhandoff

LLMへはPattern名だけを渡さない。

bounded context:

```text
pattern key
nature
concise structural statement
provenance summary
applicability questions
contraindications
negative evidence
current exact M1 facts
```

を渡す。

LLMの仕事:

- candidate比較
- current requirementとの差の説明
- unresolved questionの抽出
- TEI-native minimum delta候補

LLMの仕事ではない:

- frequencyから採用決定
- source不足の補完
- PatternをCanonical kindへ昇格

## 24. Design Synthesisへのhandoff

Design SynthesisではPatternをcandidate generationへ使える。

```text
intent
  + current accepted snapshot M1
  + Reference Pattern candidates M2/M3
      ↓
candidate designs M3
      ↓
cross-view proof
      ↓
supported / contradiction / unresolved
```

Pattern matchはproof resultではない。

候補生成とproofを分離する。

## 25. Pattern更新 / versioning

研究が進むとPattern statementが修正される可能性がある。

例:

`example_instance_snapshot`が後続研究でDataset / Snapshot / Scenario / Presentation Orderへ分解された。

したがってPattern evidenceにはresearch version / lineageを残す。

ただしglobal Semantic revisionへはしない。

概念上:

```text
P-07 research revision 1
  observed instance slides

P-07 research revision 2
  ambiguity found: chronology vs presentation order
```

のように、evidence interpretationの変化を追えることが望ましい。

## 26. Deprecation / split / merge

Pattern Cardも永続taxonomyではない。

将来:

- split
- merge
- rename
- deprecate
- reject

できる。

M1 current designはPattern keyへ依存しないため、Pattern taxonomy変更でaccepted systemが壊れない。

## 27. Pattern Evidence Quality Review

Patternをretrieval corpusへ入れる前に確認する。

### Provenance

- corpus / research artifactへ戻れるか
- raw content copyではなく抽象化か

### Nature

- observed / candidateを区別したか

### Breadth

- source_count / corpus_sizeを保持したか
- frequencyをprobabilityへ変換していないか

### Applicability

- applicability questionがあるか
- contraindicationがあるか

### Negative evidence

- ambiguity / counterexampleがあれば保持したか

### TEI translation

- existing conceptで表現する道を先に検討したか
- Pattern名をCanonical kindへしていないか

### Reconstructability

- retrieval indexを消してもprovenance card / current M1が残るか

## 28. Initial Handoff Table

| Key | Nature | Breadth | Primary TEI use | Main caution |
| --- | --- | ---: | --- | --- |
| P-01 header_detail_transaction | observed | 8/12 | Data relation / interaction candidate | independent child lifecycle |
| P-02 conditional_subtype_role | observed | 8/12 | Data modeling alternatives | role/state/type混同 |
| P-03 self_referential_hierarchy_or_chain | observed | 4/12 | relation candidate | hierarchy vs chain / cycle |
| P-04 plan_actual_consequence_ledger | candidate | 10/12 | lifecycle / fact separation candidate | frequency != observation nature |
| P-05 derived_summary_projection | candidate | 10/12 | Query/View/summary candidate | derived vs frozen Business fact |
| P-06 composition_bom_bundle | candidate | 6/12 | composition candidate | containment semantics overreach |
| P-07 example_instance_snapshot | observed | 4/12 | Dataset/Scenario/Simulation | chronology vs presentation order |

この表はsearch seedであり、Pattern catalogの完成形ではない。

## 29. P0-4で固定しないもの

- production Pattern schema
- Pattern API
- Pattern database
- vector store
- Pattern score
- confidence probability
- automatic recommendation
- Pattern-to-Code generator
- X-TEA importer
- original Reference Model redistribution
- StateMachine Canonical kind
- new Product Requirement / Gate / PAS

## 30. P0-4結論

X-TEA / XEAD Reference Modelから得た知識を生成AI・脳科学時代へ引き継ぐ際、最も重要なのは大量のmodel contentsをAIへ覚えさせることではない。

必要なのは:

```text
M3
Referenceで何を観測したか
+ どう抽出したか
+ 何が曖昧か
+ どこまでしか言えないか

        ↓ derive

M2
構造類似を思い出すための軽量representation

        ↓ retrieve

current M1
現在のBusiness / Semantic / Application authorityへ照合

        ↓

M3 Proposal
比較可能な候補として提示

        ↓ explicit Acceptance only

M1 owner source
```

という流れである。

この構造なら、Reference Modelを「知的社会資本」として利用しつつ、過去モデルの答えをcurrent systemへ押し付けない。

P0-1〜P0-4でGate C前のdefinition-centered design preparationは一巡する。

次の実証段階では、Gate C Ready後に一つのsmall semantic changeを使い:

```text
ISSUE-000047 knowledge-first exploration
        +
Reference Pattern candidates
        +
ISSUE-000046 local Studio context
        +
ISSUE-000027 minimum semantic delta Proposal
        ↓
Shared Acceptance
        ↓
accepted source / reload / regeneration
```

を一つのclosed loopとして通す。
