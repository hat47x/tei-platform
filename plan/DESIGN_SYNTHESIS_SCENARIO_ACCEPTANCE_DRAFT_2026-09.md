# Design Synthesis Scenario Acceptance Draft — 2026-09

Status: **Lane D / Scenario Acceptance draft / pre-Gate-C design contract / 未採用**

## 1. この文書の役割

本書は、`DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`で定義したScenario S1〜S5について、何をもって「そのScenarioが成立した」と評価するかを具体化するAcceptance素案である。

対象は次の五つである。

```text
S1  上流モデリングの複数案提案
S2  他設計要素による縮約
S3  local LLM synthesis
S4  implementation / test derivation
S5  feedback closure
```

本書はProduct Requirement、Gate、PAS、production schema、AI runtime contractではない。

また、現時点でS1〜S5をすべて実装可能とみなす文書でもない。

TEIの現在地ではProduct Gate BはNot Ready、Gate CはBlockedであり、Studio / knowledge / AIのproduction implementationはGate C後の主線である。したがって本書では、**Gate C前にread-onlyでPressureできるAcceptanceと、Gate C後の実Referenceで初めて評価すべきAcceptanceを分離する。**

## 2. 上位原則

S1〜S5を通して、次を不変とする。

```text
retrieved candidate
  != accepted meaning

proof supported
  != whole candidate accepted

proof unresolved
  != false

proof contradiction
  != unrelated candidate contradiction

candidate survived contraction
  != candidate supported

supported existing candidate
  != preferred / winner / accepted candidate

missing Evidence / authority
  != contradiction

provider / runtime failure surface
  != semantic contradiction

provider realization supported
  != semantic Acceptance

operational recovery required
  != structural provider incompatibility

Reference Pattern frequency
  != current applicability

past successful composition / experience memory
  != current applicability

derived procedural knowledge
  != Business Procedure authority

retrieval similarity
  != semantic evidence

LLM output
  != authority

Proposal / semantic change accepted
  != current execution authority

decision policy / routing choice
  != semantic action contract
  != current execution authority

knowledge available
  != selected candidate
  != current applicability
  != disclosed context

generated implementation
  != test oracle

runtime symptom
  != semantic owner
```

Design Synthesis Loopの価値は、一つのAIが一つの答えを返すことではない。

```text
曖昧なpurpose
  -> meaningの異なるcandidate
  -> exact authorityによる縮約
  -> unresolvedの保存
  -> 最小限の人間判断
  -> accepted meaning
  -> derived implementation / test
  -> observation
  -> meaning ownerへfeedback
```

という閉じた流れを、authorityを混同せず成立させることである。

## 3. Scenarioごとの実行可能時期

現時点では次の二層へ分ける。

| Scenario | Gate C前 | Gate C後 |
| --- | --- | --- |
| S1 Multiple Design Proposal | read-only / synthetic candidate pressure可 | realistic intentで本評価 |
| S2 Cross-view Contraction | R1〜R6 + frontier / provider pressure可 | realistic candidate frontierで本評価 |
| S3 local LLM synthesis | packet / output contractの設計まで | local modelを使ったactual evaluation |
| S4 implementation / test derivation | derivation source / oracle境界の設計まで | accepted candidateからactual生成 |
| S5 feedback closure | routing rule / provider outcome owner boundaryの設計まで | generated artifact / runtime observationでactual loop |

この区分は、Gate C前の思考停止を意味しない。

S1 / S2については、production AIなしでもcandidate / proof / provenance / freshness / provider scope境界をPressureできる。実際にR1〜R6、S1 semantic candidate diversityのtest-only pressure、`order_entry` accepted sourceを使ったS1 / S2 frontier pressure、provider feasibility scope / outcome classification pressureで、その基礎Evidenceを得ている。

一方S3〜S5は、actual local LLM、accepted candidate、generated artifact、runtime observationが存在しないまま詳細schemaを先に固定すると、架空の実装契約を増やす危険が大きい。

## 4. Acceptance Evidenceの層

各ScenarioのAcceptanceを一つの`pass=true`へ畳まない。

最低でも次を区別する。

```text
A. Input provenance
B. Semantic result
C. Authority discipline
D. Reproducibility / traceability
E. Human reviewability
F. Negative case
G. Actual execution evidence
```

Gate C前のread-only pressureではGを満たしたと主張しない。

### A. Input provenance

- どのaccepted snapshotを基礎にしたか
- どのpartial intent / Business factを入力したか
- Reference Pattern / M2 retrieval / M3 Evidenceを使った場合、そのprovenanceへ戻れるか

### B. Semantic result

- 候補同士の意味差が説明できるか
- supported / contradiction / unresolvedを保持できるか
- exact identityへ戻れるか
- semantic candidateとrealization / binding variantを必要なcaseで区別できるか

### C. Authority discipline

- M2 / M3 / LLM / runtime artifactをM1へ暗黙昇格していないか
- missing authorityを補完していないか
- provider / runtime limitationからBusiness / Canonical meaningを逆authorしていないか
- Shared Acceptanceを飛ばしていないか

### D. Reproducibility / traceability

- candidateや除外理由をaccepted source / Evidenceまで辿れるか
- candidate / proofが実際に依存するrequired accepted authority fingerprintが変わった場合にstaleを検知または再評価できるか。無関係なaccepted changeだけで一律staleにしない
- provider / runtime outcomeのreason / owner / scopeへ戻れるか

### E. Human reviewability

- 人間が「何を判断する必要があるか」を読み取れるか
- Repository全体の探索やAI内部推論を再現しなくても、残余判断を行えるか

### F. Negative case

- もっともらしい誤候補を落とせるか
- unresolvedを誤ってsafe / trueへ丸めないか
- generic failure surfaceだけで意味を誤縮約しないか

### G. Actual execution evidence

- actual process / local model / generator / runtimeで実行したか
- source-level testや設計文書をactual executionへ読み替えていないか

## 5. S1 — 上流モデリングの複数案提案

### 5.1 目的

S1は、partial business intentから一つの即答を出すことではない。

**意味の異なる複数candidateを、既存asset再利用候補・Pattern由来proposal・新規proposalを混同せず保持できること**を確認する。

### 5.2 入力契約

最低限:

```text
purpose / partial business intent
known accepted facts
explicit unknown / unresolved
current accepted snapshot
allowed candidate scope
```

を持つ。

入力不足をLLMやPatternで埋めない。

### 5.3 Candidateの最低条件

候補ごとにconceptually次を復元できること。

```text
candidate key
semantic claim / proposed meaning
existing target identity, if any
proposal-only meaning, if new
assumptions
unresolved
current qualification state
discovery traces
provenance refs
```

production schemaとして固定しない。

### 5.4 意味差を持つ複数案

Acceptanceでは単なる表現違いを候補数として数えない。

次のようなものは同一案の言い換えである。

```text
A. OrderとOrderLineを1:Nで持つ
B. Order headerとOrder detailを1:Nで持つ
```

一方:

```text
A. Order / OrderLineを同一aggregateとして持つ
B. lineを独立Business objectとして持つ
C. line ownershipを外部Capabilityへ委譲する
```

はmeaningが異なるcandidateになり得る。

### 5.5 Existing assetとproposalを分ける

```text
existing candidate
  -> current M1 identityへexact qualification

Pattern-inspired candidate
  -> M3 provenance付きproposal

associative discovery
  -> M2 discovery trace
```

同じcandidateが複数経路で見つかった場合、candidateを複製せずdiscovery traceを複数保持する。

### 5.6 S1 Acceptance

S1を成立とみなす最低条件:

1. independentなpartial intentを一つ用意する。
2. 意味の異なるcandidateを最低2案保持する。最大数を固定KPIにはしない。
3. existing asset candidateはcurrent M1へexactに戻る。
4. proposal candidateは存在しないCanonical IDをaccepted refとして持たない。
5. Reference Pattern / associative retrieval / experience memoryのprovenanceをcurrent semantic Evidenceへ変換しない。
6. past successful composition / experience candidateを利用する場合、元Execution / Observation / Evidenceへ戻れ、current accepted authority / dependency / Target条件で再qualificationされる。
7. 各candidateでassumption / unresolvedが区別される。
8. 「候補が一つしか残らない」場合も、生成器が一つしか出さなかったのか、constraintで縮約されたのかを区別できる。

### 5.7 S1 Negative Acceptance

次のいずれかが起きたらS1は不成立。

- 言い換えだけを複数案として数える
- Pattern名をcurrent requirementへ自動適用する
- retrieval scoreをapplicabilityと呼ぶ
- 過去の成功trajectory / compositionをcurrent applicabilityやaccepted meaningとして扱う
- stale experienceをcurrent accepted authority / Target条件の再確認なしに適用する
- missing identityをLLMが補完する
- unresolvedをcandidateから隠す
- existing assetとnew proposalを同じauthorityとして表示する

### 5.8 Gate C前にできること

R5のcandidate discovery / provenance pressureに加え、`tools/reference-business-procedure-check/design_synthesis_s1_candidate_diversity_pressure_test.go`で、synthetic candidate setについて次をtest-onlyにPressureしている。

- surface wordingの違いだけを独立candidateとして水増ししない
- surface variantを一つのsemantic alternativeへ畳んでもprovenanceを失わない
- ownership / identity等の意味差があれば別candidateとして保持する
- `1案しか生成されなかった`と`複数案がconstraintで1案へ縮約された`を区別する

さらに`DESIGN_SYNTHESIS_ORDER_ENTRY_S1_S2_FRONTIER_PRESSURE_2026-09.md`では、current `order_entry`のaccepted sourceをauthorityとして、existing read-only reuse / new snapshot proposal / master mutation proposalというmeaning-different frontierをread-onlyでPressureしている。

ただしこれらはcandidate generatorのactual能力試験ではない。「実際の上流設計で有用な複数案を生成できた」というProduct Evidenceにはしない。

## 6. S2 — 他設計要素による縮約

### 6.1 目的

S2はcandidateをAI scoreで順位付けするScenarioではない。

Business / Data / Function / Interaction / Security / Actuality / Handoff / Target / Provider等から得られるexact constraintを使い、**説明可能にcandidate spaceを縮約し、本当に残るunresolvedを保持すること**を確認する。

### 6.2 R1〜R6と後続Pressureから得ている境界

```text
same proof family
  != same candidate scope

contradiction
  -> exact scopeならdeterministic elimination可能

unresolved
  -> elimination reasonではない

supported
  -> whole candidate Acceptanceではない

missing required proof / Evidence
  -> safeではない
  -> falseでもない

fresh proof
  != stale proof

stale contradiction
  -> current candidate eliminationには使わない

semantic candidate
  != realization / binding variant

provider failure surface
  != one generic contradiction
```

### 6.3 Incremental input

S2では追加情報を段階的に与える。

例:

```text
step 1
  Business intent only

step 2
  datasource access

step 3
  Validation / Capability

step 4
  Business Procedure / Handoff

step 5
  Target / Provider feasibility
```

追加情報のたびにRepository全candidateを自由再生成するのではなく、影響scopeへ閉じて再評価する。

Target / Provider feasibilityでは、少なくとも次を区別する。

```text
missing provider Evidence
  -> unresolved

observed unsafe realization
  -> matching realization contradiction

current provider structurally unsupported
  -> current provider capability claimへのcontradiction

operational recovery required
  -> runtime / operations unresolved

supported realization
  -> feasibility support
  != semantic Acceptance
```

### 6.4 Candidate-local proof selection

candidate claimへ関係するproofだけを適用する。

```text
candidate C-auth
  protected targetを検討

relevant
  P4

irrelevant
  unrelated P3 mutation contradiction
```

無関係なcontradictionでcandidateを除外してはならない。

同様に、provider-specific contradictionを上位semantic candidate全体へ無条件に伝播させない。accepted hard Target / Provider constraintがある場合だけ、そのsemantic meaningとmatching realizationを組み合わせたcomposite candidateの除外へ拡張できる。

### 6.5 Explainable elimination

candidateを除外する場合、最低限:

```text
candidate / candidate claim
contradicted claim
decisive proof
accepted authority / evidence
accepted snapshot / required authority fingerprints
reason
owner
scope
```

へ戻れること。

full proof logを保持すること自体は要求しない。

missing Evidenceやoperational recoveryは、除外certificateではなく`unresolved` / owner-routingとして保持する。

### 6.6 Unresolved frontier

候補が残る場合、unresolvedを意味dimensionへ整理できること。

ただし:

```text
aggregate
  != flatten
```

であり、member proofとproofごとのmissing authorityを保持する。

S2後のfrontierは、一つのqualification stateへ揃っている必要はない。

```text
supported existing candidate
  +
unresolved proposal candidate
  +
semantic candidate with unresolved realization Evidence
```

が同時に残ることは正しい結果になり得る。

ここで:

```text
survived
  != supported

supported
  != preferred / winner / accepted
```

を維持する。

S2はfrontierを必ず1 candidateへ縮約するScenarioではない。根拠不足のproposalを「1案化」のためだけに落とさない。

### 6.7 S2 Acceptance

1. S1の候補に対し、少なくとも一つの追加cross-view constraintを与える。
2. 一つ以上のcandidateまたはcandidate claimを説明可能に縮約できるcaseを含む。
3. 一つ以上の`unresolved`を正しく残すcaseを含む。
4. unrelated contradictionがcandidate eliminationへ流入しない。
5. required proof / Evidence欠落を`no contradiction`としてsafe扱いせず、同時に`false` / contradictionへ変換しない。
6. eliminated candidateはdecisive contradictionへ戻れる。
7. unresolved aggregation後もmissing authority traceを失わない。
8. proofが依存するrequired accepted authority fingerprintが変わった場合、旧proofをcurrent resultとして使わない。無関係なaccepted changeだけでは当該proofをstaleにしない。
9. contraction後に複数candidateが残るcaseを許容し、1 candidateへの強制縮約をAcceptance条件にしない。
10. `supported existing`と`unresolved proposal`が同時に残る場合、そのqualification差を保持する。
11. existing reuse / minimum semantic delta等のpreferenceを、accepted constraintでない限りcontradictionとして扱わない。
12. semantic candidateとrealization / binding variantを必要なcaseで区別する。
13. missing provider Evidenceはrealization contradictionにせず`unresolved`として残す。
14. observed / structural provider incompatibilityはmatching realization scopeへ閉じる。
15. accepted hard Target / Provider constraintがある場合だけcomposite candidate eliminationへ拡張し、semantic subject自体を消さない。
16. provider / runtime outcomeをsurface `error`だけで分類せず、reason / owner / scopeへ戻れる。
17. operational recovery requiredをstructural provider incompatibilityへ変換しない。
18. provider realization supportをsemantic Acceptanceへ短絡しない。

### 6.8 S2 Negative Acceptance

- candidate全体へ単一`valid` flagだけを付ける
- proof inventory全体から一件でもcontradictionがあれば全candidateを落とす
- no Data traceをno dependencyへ変換する
- DB writeからCapability mutationをauthorする
- policyからBusiness protected targetをauthorする
- Command successからphysical completionをauthorする
- positive ObservationからHandoffをauthorする
- stale contradictionをcurrent contradictionとして再利用する
- proofのsupported数でwinnerを決める
- unresolved candidateをfrontierの1案化だけを目的として捨てる
- existing assetが存在するという理由だけでnew proposalをcontradiction扱いする
- missing provider observation / Evidenceをprovider contradictionとしてcandidateを落とす
- current provider limitationからBusiness / Canonical meaningを否定する
- generic `error`だけを見てmissing Evidence / observed unsafe / structural unsupported / operational recoveryを同じ結果へ畳む
- hard provider requirementを理由にoperational recoveryまでpermanent incompatibilityへ変換する
- provider supportをsemantic Acceptanceとして扱う

### 6.9 Gate C前にできること

R1〜R6とtest-only pressureにより、S2のauthority / contraction / freshness boundaryはかなりPressureできている。

特にR6では、proofが実際に依存するaccepted authority fingerprintへfreshnessを束縛し、unrelated changeで全proofをstale化せず、missing / ambiguous authorityではfail closedし、stale contradictionをcandidate eliminationへ流さない境界を確認している。

加えて`DESIGN_SYNTHESIS_ORDER_ENTRY_S1_S2_FRONTIER_PRESSURE_2026-09.md`では、customer確認のsynthetic partial intentへcurrent accepted sourceを当て、read-only existing candidateをretainし、master mutation proposalをP3-02 contradictionで除外し、order-owned snapshot proposalを`unresolved`のまま残すcaseを確認している。これはS2がwinner selectionではなく、異なるepistemic statusを保ったfrontier contractionであることを具体化する。

`DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_SCOPE_PRESSURE_2026-09.md`では、current PostgreSQL fail-closed sourceを使い、semantic candidateとrealization / binding variantを分離し、missing provider Evidenceをsemantic contradictionへ変換せず、provider-specific incompatibilityをmatching realization scopeへ閉じることをPressureした。

さらに`DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_OUTCOME_CLASSIFICATION_PRESSURE_2026-09.md`では、同じfailure surfaceへ現れるmissing Evidence、observed unsafe、structural unsupported、operational recovery、supported realizationをreason / owner / scopeで分離し、generic provider failureだけからcontractionを決めないことをPressureしている。

しかしactual Design Synthesis engineの性能、人間のreview工数削減、actual incremental re-evaluation cost、actual provider evaluation loopまでは未評価である。

## 7. S3 — local LLM synthesis

### 7.1 目的

S3はLLMにRepository全体を読ませて「最善案」を選ばせるScenarioではない。

S2後の小さいfrontierとunresolvedだけをbounded contextとして渡し、**意味上の残余を比較・説明し、minimum semantic deltaをProposalとして返せるか**を評価する。

### 7.2 Precondition

S3をactual評価する前に次を要求する。

```text
S1 candidate provenance boundary established
S2 contraction boundary established
bounded context packet available
current accepted snapshot identified
Shared Acceptance boundary available
```

Gate C前にはproduction local LLM integrationを開始しない。

### 7.3 LLMへ渡す内容

概念上:

```text
purpose
candidate frontier
exact accepted facts
selected supported proof
unresolved dimensions
missing authorities
realization / provider unresolved where relevant
representative elimination reasons
authority limits
requested structured output
```

を渡す。

Repository全dumpやhidden exploration historyは標準入力にしない。

routing / retrievalを使う場合、LLMへ渡すKnowledgeは次の段階を区別する。

```text
available
  -> selected by routing
  -> deterministically qualified
  -> intentionally disclosed
```

selectedであることだけを`applicable`とせず、routing score / retrieval rank / model内部signalをSemantic Evidenceへ変換しない。qualification済みでもcurrent taskに不要ならdiscloseしなくてよい。一方、current判断に必要なunknown / missing authority / unresolvedは削らない。

### 7.4 LLM出力

少なくとも次を区別できること。

```text
comparison
assumptions
proposed semantic delta
required question / missing authority
remaining unresolved
supporting refs
```

LLM outputはProposalである。

### 7.5 Minimum semantic delta

S3ではwhole model再生成より、current accepted snapshotとの差分を小さく返せることを優先評価する。

例:

```text
bad
  order-entry modelを全面再生成

good
  protected target authorityをauthorする必要あり
  candidate value = order_entry.submit_order
  supporting corroboration = legacy policy
  current status = proposal, not accepted
```

minimum semantic deltaはS3の比較上のpreferenceであり、S2のproof resultではない。

```text
existing reuseの方が変更量が小さい
  != accepted authorityがnew proposalをcontradictした
```

### 7.6 S3 Acceptance

actual local LLM評価では少なくとも:

1. bounded packetだけでtaskを実行する。
2. exact refsを捏造しない。
3. `unresolved`をtrue / falseへ勝手に丸めない。
4. accepted factとproposalを区別する。
5. missing authorityを明示できる。
6. realization / provider unresolvedをsemantic factへ書き換えない。
7. whole-model rewriteではなくminimum deltaを返せるcaseを含む。
8. eliminated candidateを根拠なく復活させない。
9. model差し替えでM1 schema / identityが変わらない。
10. outputがShared Acceptance前にowner sourceへwrite-backされない。
11. routing / retrievalを使う場合、disclosed itemをselected candidate / qualification / Source・Evidenceへtraceできる。
12. routing scoreやselected statusだけをapplicabilityとしてLLMへ提示しない。
13. context削減のためにcurrent decisionに必要なunknown / missing authorityを隠さない。

### 7.7 比較Evidence

whole-repository prompt等との比較を行う場合も、一つの「品質score」だけにしない。

見る候補:

- fabricated exact ref数
- unresolved preservation
- unsupported assumption数
- humanが確認するsemantic choice数
- context size
- latency / memory
- minimum delta率
- review correction回数

固定閾値は実Reference前に決めない。

## 8. S4 — implementation / test derivation

### 8.1 目的

S4はLLM code generationの能力試験ではない。

**accepted semantic meaningを人が再入力せず、standard implementation / testをどこまでdeterministically deriveできるか**を評価する。

加えて、Product Valueでいう`structure-to-application leverage`を、単なる生成率ではなく「一つのstructured meaningから複数の下流surfaceへどこまで一貫して波及でき、change amplificationと重複authoringを減らせるか」というReference Evidenceとして扱う。

### 8.2 Precondition

```text
accepted candidate exists
owner source updated through Shared Acceptance
accepted snapshot reproducible
derivation target identified
```

Proposalのままimplementationをproduction authorityへしない。

### 8.3 Derivation priority

```text
existing reusable component / Pattern
  -> declarative rule / binding
  -> deterministic generator / compiler
  -> constrained LLM generation
  -> custom handwritten residual
```

下位手段へ進んだ理由を追跡できること。

### 8.4 Test oracle

```text
accepted semantics / independent Evidence
  -> test oracle

implementation output
  -X-> sole test oracle
```

Validation、Relation、Capability Outcome、Authorization、Pattern、Handoff等からderiveできるtestはmeaning ownerへ戻る。

### 8.5 S4 Acceptance

1. 一つのaccepted semantic changeを起点にする。
2. implementation artifactのどの部分がderivedかを示せる。
3. test / fixture / conformanceの根拠をaccepted meaningへ戻せる。
4. implementationとtest oracleの独立性を少なくとも一つのnegative caseで示す。
5. custom residualを明示する。
6. generated artifactをCanonicalへ逆書きしない。
7. accepted meaning変更時にaffected artifactを再生成またはstale判定できる。
8. Target固有差をCanonical business meaningへ持ち上げない。
9. 同じaccepted meaningからderiveできるProjection / Target configuration / documentation / structured AI context等がある場合、別のauthoritative meaningを作らず再生成できる。
10. 同一change challengeで、manual duplicate authoring / manual synchronizationがどこに残るかを説明できる。
11. native / custom residualが特殊Interaction・algorithm・provider固有処理等へ局所化されているかを確認できる。

### 8.6 Structure-to-application leverageの比較Evidence

S4のReference比較では、単一の総合scoreを先に固定しない。

同じsemantic change challengeについて、少なくとも次を観測候補にする。

- authoritative meaningを変更する箇所数
- derived / regenerated surfaceの種類
- 手動で同期する必要があるsurface数
- 同じsemantic factの重複記述箇所
- stale判定 / regenerationで追随できるartifact
- handwritten / custom residualのscope
- before / afterで触るlayer・file・test・documentationの広がり

```text
high leverage
  != generated file count is large
  != TEI-owned code percentage is high

high leverage
  = one accepted meaning change
    -> multiple downstream surfaces follow without duplicate authority
```

SUI等のReference Adoptionでは既存Applicationのhistorical change challengeと比較できるが、その数値をRepository全体の固定KPIや採用閾値にはしない。

### 8.7 Gate Cとの関係

第一Web Target / Gate Cでは、baseline meaningとTarget projectionの分離、Semantic / Functional / Visual Evidence分離、Command / Observation境界等を実証する。

S4はそのEvidenceを踏まえ、Gate C後に「設計からTarget artifactをどこまでderivedへ寄せられるか」を評価する。

Gate C前にWeb実装生成器をDesign Synthesis専用として先行実装しない。

## 9. S5 — feedback closure

### 9.1 目的

S5はruntime errorをLLMへ投げて自動修正するScenarioではない。

実行・Simulation・Target evaluationから得た症状を、**正しいmeaning-owning layerへroutingし、minimum semantic delta Proposalへ戻せること**を評価する。

### 9.2 Feedbackの分類

最低限:

```text
semantic inconsistency
implementation defect
Target projection defect
provider realization incompatibility
provider / runtime Evidence不足
operational recovery required
physical actuality mismatch
visual / interaction concern
performance concern
```

を同じ「失敗」へ潰さない。

### 9.3 Routing

例:

```text
read-only datasourceでwrite UIが出た
  -> Application / Business semantic inconsistency候補

Command成功だが外部状態未成立
  -> Observation / actuality concern

CSS groupingが読みにくい
  -> Target visual concern
  != Data model defect

providerがcurrent operationを構造的にrealizeできない
  -> Binding / provider feasibility concern
  != Business intent rewrite

provider observationが不足している
  -> Evidence acquisition / unresolved
  != realization contradiction

runtimeがoperational recoveryを要求する
  -> Runtime / Operations owner
  != permanent provider incompatibility
```

### 9.4 S5 Acceptance

1. generated artifactまたはruntime / simulationから一つのobservable feedbackを得る。
2. feedbackのEvidence artifactを保持する。
3. symptomからowner layerを決定論的またはreviewableに特定する。
4. provider / runtime failureではreason / owner / scopeを保持する。
5. derived artifactへ直接patchしてauthorityを分岐させない。
6. semantic changeが必要ならminimum semantic delta Proposalへ戻す。
7. implementation-only / operational-only defectならsemantic owner sourceを偽変更しない。
8. Proposalは再びShared Acceptanceを通る。
9. Accept後にaffected implementation / testを再deriveし、loopを一周できる。

### 9.5 S5 Negative Acceptance

- browser symptomからCanonical fieldを自動追加する
- runtime stateをBusiness factへ自動昇格する
- generated codeのmanual patchを新しい正本にする
- LLM patch成功をAcceptanceとみなす
- visual problemをData schema変更で隠す
- provider limitationをBusiness requirementとして固定する
- missing provider Evidenceをsemantic contradictionへ変換する
- operational recovery requiredをstructural incompatibilityとして固定する
- surface `error`だけでownerを決める

## 10. Scenario間の依存

S1〜S5は独立benchmarkではない。

```text
S1
  candidate meaning / provenance
   ↓
S2
  exact contraction / unresolved / realization scope
   ↓
S3
  bounded residual synthesis
   ↓
Shared Acceptance
   ↓
S4
  deterministic derivation
   ↓
S5
  observation / owner routing / next proposal
   ↺
```

S3だけ高性能でも、S1/S2のauthority境界が壊れていればDesign Synthesis Loopとしては不成立である。

同様に、S4の生成率が高くてもtest oracleがgenerated implementation自身なら不十分である。

## 11. Acceptanceの成熟度

Scenario acceptanceには成熟度を持たせるが、production enumとして固定しない。

概念上:

```text
A0  design-only acceptance drafted
A1  source/test-only pressure exists
A2  actual execution evidence exists
A3  realistic Reference evidence exists
A4  cross-scenario closed loop exists
```

現在のLane Dはおおむね:

```text
S1  A1相当 — provenance / semantic candidate diversity / order-entry frontier pressureあり
S2  A1相当 — cross-view / candidate-local contraction / selective proof freshness / order-entry frontier / provider feasibility scope + outcome pressureあり
S3  A0 — bounded packet / output boundaryまで
S4  A0 — derivation / oracle boundaryまで
S5  A1未満 — routing boundary + provider outcome test-only pressureあり、actual feedbackなし
```

と評価する。

S1 / S2およびS5の一部にsource/test-only Evidenceを含むが、actual `go test` greenを意味しない。GitHub-hosted runnerのpre-run allocation blockerが継続している期間はA2へ進めない。

これはGateや正式なmaturity levelではない。

## 12. R1〜R6と後続Pressureとの対応

| Evidence | 主に支えるScenario | 内容 |
| --- | --- | --- |
| R1 order-entry proof | S2 | P1〜P4、supported / contradiction / unresolved |
| R2 actuality / Handoff | S2 / S5 | Command / Observation / Business continuation非短絡 |
| R3 Context Packet boundary | S2 / S3 | bounded packet、authority limit、minimum residual |
| R4 candidate-local proof packet | S2 / S3 | scope-local proof、elimination certificate、unresolved aggregation |
| R5 candidate discovery provenance | S1 / S2 | exact / Pattern / associative discoveryのprovenance分離 |
| R6 snapshot invalidation | S2 / S3 | proof freshness、selective invalidation、stale contradictionの除外 |
| S1 semantic candidate diversity pressure | S1 | surface variantとsemantic alternativeの分離、生成1案と縮約後1案の区別 |
| order-entry S1 / S2 frontier pressure | S1 / S2 / S3 | existing supported candidateとunresolved proposalの共存、exact contradictionによる縮約、winner selectionとの分離 |
| mixed-frontier context refinement | S2 / S3 | semantic subject / discovery trace / proof qualificationをflattenせずS3へ渡す |
| provider feasibility scope pressure | S2 / S5 | semantic candidateとrealization variantを分離し、provider contradictionをmatching scopeへ閉じる |
| provider feasibility outcome classification pressure | S2 / S5 | missing Evidence / observed unsafe / structural unsupported / operational recovery / supportのreason・owner・scope分離 |

これらはS1〜S5そのもののProduct Acceptanceではない。

とくにS3〜S5のactual executionを証明していない。またS1 / S2についてもactual candidate generator / contraction engineの性能を証明していない。

## 13. Gate C前にこれ以上固定しないもの

現時点で次をproduction contractへしない。

- `DesignCandidate` Canonical kind
- `DesignContextPacket` production JSON schema
- candidate origin enum
- persisted proof result enum
- `EliminationCertificate` production schema
- `ProviderFeasibility` production enum / generic provider outcome taxonomy
- Pattern Memory production schema
- LLM provider / model family
- prompt contractの細部
- vector DB / graph DB
- generic solver
- AI score / acceptance probability
- automatic Shared Acceptance
- semantic-delta persistence model
- feedback event ontology

actual Referenceでlossが観測された箇所だけreopenする。

## 14. Gate C後の最初の実証順

Gate C Ready後、いきなりS1〜S5全部を実装しない。

推奨順:

```text
1. S1 realistic intent / candidate frontier
2. S2 incremental cross-view contraction
3. Target / Provider constraintをrealization scopeへ当て、failure reason / owner / scopeを保持する
4. bounded packet固定ではなくactual information-loss測定
5. S3 local LLM comparison / minimum delta
6. Shared Acceptance経由で一つのsmall semantic changeをAccept
7. S4 standard artifact / test derivation
8. S5 one observable feedbackをmeaning ownerへ返す
9. 1つのclosed loopを完走
10. そのEvidenceからproduction schemaを決める
```

最初の価値は「高度なAI」ではなく、**一つの意味変更を、候補化・縮約・受理・生成・観測・再提案までauthorityを崩さず一周できること**に置く。

## 15. 現時点の判断

Lane Dのpre-Gate-C read-only workとしては、S1〜S5のAcceptance素案をここまで定義できる。

特にS1 / S2はR1〜R6、S1 semantic candidate diversity pressure、current `order_entry` sourceを使ったfrontier pressure、provider feasibility scope / outcome classification pressureにより、production implementation前でも誤った短絡をかなり除外できた。

現在のS2 frontierは、epistemic statusだけでなくrealization statusも一つへ揃える必要がない。

```text
supported existing candidate
  +
unresolved proposal candidate
  +
semantic candidate with unresolved / incompatible realization variant
```

が残る場合、S2はその差を保持してS3 / human reviewへ渡す。minimum semantic deltaやexisting reuse preference、provider failure surfaceを、accepted constraintなしにproofへ昇格しない。

S5でも同じ境界を守り、runtime / provider symptomを正しいownerへ戻す。とくにoperational recovery requiredをsemantic defectやpermanent provider incompatibilityへ固定しない。

次の境界は明確である。

```text
Gate C前
  -> read-only / test-only authority pressure
  -> scenario acceptance refinement
  -> actual runnerが得られればsource-level qualification

Gate C後
  -> realistic S1 / S2
  -> actual local LLM S3
  -> accepted semantic change
  -> actual derivation S4
  -> actual feedback closure S5
```

この時点で新しいAI abstractionを増やすより、Gate B / Gate C主線を進め、実際のHandover BaselineとFirst Targetから得られるmeaning / Target / Evidenceを次のDesign Synthesis Referenceへ持ち込む方がよい。