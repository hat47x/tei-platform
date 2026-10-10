# Definition-Centered Memory Separation Audit — 2026-09

Status: **P0-3 baseline audit / no storage architecture adoption**

Baseline: `main` after P0-2 Feedback Routing Catalog

Related:

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`
- `plan/DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md`
- `docs/architecture.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
- `plan/design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`
- `docs/product-quality-architecture.md`

## 1. 目的

Definition-centered feedback loopのP0-3として、TEIで将来AI・非LLM cognition・Studio・Runtime feedbackを高度化しても、**accepted meaning / associative knowledge / experience recordが互いのauthorityを奪わないこと**を監査する。

前段では三種類のmemoryを次のように整理した。

```text
M1 authoritative semantic memory
M2 derived associative / structural memory
M3 episodic / experience memory
```

本auditの中心は、この三分類を単なる保存先分類にしないことである。

```text
memory class
  != storage product
  != database table
  != process boundary
```

同じartifactが複数の役割を持つ場合もある。分類で見るのは、**その情報が何を決める権限を持つか、削除・再生成・stale時に何が起きるべきか**である。

## 2. 三memoryのauthority contract

### M1 — Authoritative accepted memory

前段では`authoritative semantic memory`と呼んでいるが、ここでの`semantic`は狭いCanonical Semantic Layerだけを意味しない。

concernごとにacceptedされたauthorityを含む。

例:

- Semantic Data / Relation / Validation / Capability meaning
- Application intent / Pattern
- Database Contract
- native SQL when SQL is the chosen authoring authority
- Persistence / Query / Capability Realization choice
- Environment Binding / Permission / Resource choice
- accepted Business Procedure meaning when Contractが成立した将来
- promoted Release snapshot
- explicit accepted Decision / policy where it owns behavior

共通点:

```text
これを失うと、accepted systemの意味・実現選択・公開境界・時間的決定を
他のaccepted sourceだけからlosslessに再構築できない
```

### M2 — Derived associative / structural memory

M1やaccepted sourceから再生成でき、探索・局所理解・candidate generationを助ける。

例:

- Source Index
- dependency index
- reverse reference index
- Derived Meaning Graph
- Data / Function / Business cross-view matrix
- lexical search index
- Reference Pattern retrieval index
- embedding
- sparse high-dimensional code
- HDC / VSA representation
- candidate neighborhood / rank
- novelty / familiarity signal
- impact projection cache

共通点:

```text
削除してもaccepted systemは変わらない
staleなら再生成できる
score / edge / activation自体はauthorityではない
```

### M3 — Episodic / experience memory

何を観測し、何を提案し、何を試し、何を判断したかを時点付きで残す。

例:

- Proposal
- Preview / Semantic Diff record
- Accept / Reject record
- review rationale
- AI prompt / model / trajectory when retained as Evidence
- cross-view proof evaluation result
- Quality Envelope
- Runtime execution / Observation
- external actual-state snapshot
- Diagnostic / failure record
- feedback routing record
- past correction trajectory
- benchmark / retrieval evaluation run

共通点:

```text
過去の事実・判断過程を説明する
  != current accepted meaningそのもの
```

## 3. P0-1 A〜Fとの対応は1:1ではない

P0-1ではartifactをA〜Fへ分類した。

```text
A authored irreducible meaning / intent / contract / promoted snapshot
B mechanically derived projection / golden
C native external authoritative asset
D realization / target / environment choice
E observed / runtime-only information
F proposal / evidence / episodic / non-canonical Reference
```

memory分類との概略対応:

| P0-1 class | 主memory | 注意 |
| --- | --- | --- |
| A | M1 | Release等のtemporal authorityもM1 |
| B | M2 | goldenの一部はM3的なEvidence役割も持つ |
| C | M1 | native SQL等。nativeであることとderivedは別 |
| D | M1 | Canonicalではなくてもaccepted realization authority |
| E | M3 | Actualは重要だがDesired authorityではない |
| F | M3 | Reference pattern indexへ変換したものはM2にも派生可能 |

重要なのは、artifactのfile typeではなくauthority roleである。

## 4. M1は「一つの巨大な正本Store」ではない

M1を一つのKnowledge Graphやdatabaseへ集約する必要はない。

TEIの既存原則ではconcern ownershipが異なる。

```text
Semantic meaning       -> TEI-native source
native query meaning   -> SQL
Database Contract      -> Contract source
Realization choice     -> Mapping / Binding source
Environment authority  -> Environment source
Release authority      -> promoted Release source
```

これらをまとめてM1と呼ぶ理由は、保存形式が同じだからではない。

**accepted systemを再構築するときにauthorityとして読む必要がある**からである。

したがって:

```text
M1 != one canonical database
M1 != one graph
M1 != only ir.World
```

## 5. M1 current inventory

### 5.1 Semantic source

代表:

```text
reference/workspace/semantic/sales/order.tei
reference/workspace/semantic/sales/validation.tei
```

Authority:

- identity
- attribute / relation
- validation
- capability signature

Memory rule:

**M1.**

M2のgraph / embedding / indexから逆生成しない。

### 5.2 Application intent

代表:

```text
reference/workspace/applications/order-entry/application.tei
reference/workspace/applications/order-entry/patterns/*.tei
```

Authority:

- Semantic scope
- Datasource dependency / access
- Application interaction intent
- explicit Pattern selection

Memory rule:

**M1.**

Pattern retrievalで類似caseを見つけても、M2 candidateから暗黙採用しない。

### 5.3 Database Contract

代表:

```text
reference/workspace/databases/sales-core/contracts/*.tei
```

Authority:

- published read / write / query surface

Memory rule:

**M1.**

Data modelやphysical DBから自動復元した結果をContract authorityへしない。

### 5.4 Native SQL

代表:

```text
applications/order-entry/queries/recent_orders.sql
databases/sales-core/query-realizations/order-line-find-by-order.sql
```

Authority:

選択したauthoring modeではquery implementation / native definitionのsource。

Memory rule:

**M1.**

SQL analyzer outputやquery embeddingはM2。

query execution historyはM3。

### 5.5 Persistence / Query / Capability realization

代表:

```text
databases/sales-core/persistence/*.tei
databases/sales-core/query-realizations/*.tei
applications/order-entry/realizations/*.tei
```

Authority:

- Semantic ↔ physical mapping
- Contract ↔ SQL realization
- Capability / Pattern ↔ implementation choice

Memory rule:

**M1.**

provider-specificであることはderivedであることを意味しない。

### 5.6 Environment / Permission / Resource

代表:

```text
reference/workspace/environments/development/environment.tei
```

Authority:

- provider selection
- datasource binding
- executor / observer provider
- Resource reference
- Permission grant
- lease policy

Memory rule:

**M1.**

Runtime telemetryから暗黙に再authorしない。

### 5.7 Release snapshot

代表:

```text
reference/workspace/releases/applications/order-entry/3.8.0.tei
```

Releaseは過去時点のrecordなのでM3に見えるが、accepted promoted snapshotとしてdeployment / compatibility判断のauthorityを持つ。

Memory rule:

**M1 temporal authority.**

```text
past != non-authoritative
```

current Application definitionへ追従して書き換えない。

## 6. M2 current / planned inventory

### 6.1 Source Index

Architecture上、Source情報はCanonical Worldへ埋め込まずderived Source Indexへ分離されている。

Memory rule:

**M2.**

- path / fingerprint / range / roleを再生成
- Canonical meaningがSource Indexへ依存しない
- index削除でaccepted meaningが消えない

Stale rule:

source revision変更後、古いindexをcurrent locator authorityとして使わない。

### 6.2 dependency / cross-view projection

- dependency index
- Data / Function / Business matrix
- Derived Meaning Graph
- impact graph

Memory rule:

**M2.**

Edgeには可能な限りM1 ref / derivation pathを持たせる。

```text
edge absent
  != semantic relation absent
```

### 6.3 Reference Pattern Memory

X-TEA / XEAD / TEI Referenceから抽出したpattern:

- header-detail
- conditional subtype
- self-reference
- plan / actual / consequence / ledger
- composition
- derived summary

等を検索可能にするrepresentation。

Memory rule:

**M2.**

patternの原観測 / provenanceはM3的Evidenceとして保存できるが、retrieval indexやrankはM2。

```text
pattern frequency != requirement
pattern match != applicability
```

### 6.4 Cognitive representation

- weighted feature set
- sparse code
- FlyHash-like representation
- HDC / VSA
- embedding

Memory rule:

**M2.**

```text
Sparse Code != Canonical Identity
Activation != Semantic Fact
Similarity != Applicability
Novelty != Unknown
Association != Dependency
```

削除 / rebuild可能であることが採用条件。

### 6.5 Learned association

将来:

- false-positive suppression
- task-local association weight
- frequently useful candidate hint

を保持する場合。

Memory rule:

**M2 learned derivative.**

Decision historyそのものではない。

M3のaccept / reject episodeを入力にしても、M1 ruleへ昇格しない。

## 7. M3 current / planned inventory

### 7.1 AI / human Proposal

Proposalはminimum semantic deltaを持つcandidate。

Memory rule:

**M3 until accepted.**

Accept後:

```text
accepted meaning -> M1 owner source
Proposal record  -> M3 provenance / history if retained
```

Proposal recordだけをaccepted meaningの唯一の保存先にしない。

### 7.2 Accept / Reject / review rationale

Memory rule:

**M3.**

過去にrejectされたことは、現在revisionで永久にinvalidという意味ではない。

過去にacceptされた理由も、current Sourceと矛盾するなら再評価対象。

### 7.3 Cross-view Proof Result

`supported / contradiction / unresolved`のsnapshot-bound評価。

Memory rule:

**M3 as evaluation evidence; optionally recomputable derived result.**

ここはM2/M3境界である。

- 計算結果だけならM2として捨てて再計算可能
- 「このrevisionでこの判断を行った」Evidenceとして保持するならM3

どちらの場合もM1ではない。

### 7.4 Quality Envelope

subject + revision / context + probe + Evidence + status。

Memory rule:

**M3.**

品質主張の履歴・EvidenceでありBusiness meaningのauthorではない。

### 7.5 Observed external state

代表:

```text
reference/observed/production.deployment-state.json
reference/observed/production.canary.deployment-state.json
```

Memory rule:

**M3 actuality snapshot.**

Actual stateが重要でもDesired / M1へ自動promoteしない。

### 7.6 Runtime execution / Observation

- Business Result
- Named Outcome
- Execution Error
- Observation
- Developer Diagnostic
- Operational Telemetry

Memory rule:

**M3.**

種類を畳まない。

### 7.7 Feedback routing record

P0-2で整理したFeedback Envelopeを将来保存する場合。

Memory rule:

**M3.**

owner decisionの過程を説明できるが、owner sourceの代替にはならない。

## 8. Golden artifactの境界

### `reference/world.json`

accepted Workspaceから生成されるgolden。

主役割:

**M2 derived snapshot.**

一方、Repositoryで「このrevisionの期待生成結果」として保存することはtest Evidenceの役割も持つ。

つまり:

```text
semantic authority -> no
reproducibility evidence -> yes
```

### `.tei/plugin-lock.json`

resolved Plugin / Bundle result。

主役割:

**M2 derived golden.**

Runtime planがそのfingerprintへbindされる時点では実行入力として重要でも、Plugin selection intentの元sourceを置き換えない。

### Durable Run start golden

Workspace + Plugin Lock + observed fixtureから生成される。

主役割:

**M3 runtime episode fixture + derived generation.**

M2/M3 boundary。

手編集してM1へ近づけない。

## 9. Memory transition rules

### T1 — M1 -> M2

許可:

```text
accepted source
  -> normalize / extract / derive
  -> index / graph / embedding / sparse code
```

必須:

- input revision / fingerprint
- rebuildability
- provenance back to M1 identity

禁止:

M2生成のためにM1 meaningを変えること。

### T2 — M1 / runtime -> M3

許可:

```text
accepted snapshot + execution
  -> Observation / Evidence / Quality / proof episode
```

必須:

- evaluated snapshot / context
- status / uncertaintyを保持

禁止:

Observation不足をsuccessに丸めること。

### T3 — M2 -> Proposal in M3

許可:

```text
retrieval / similarity / pattern
  -> candidate references
  -> Proposal
```

必須:

- M1 refsへ戻る
- deterministic qualification
- applicable / incompatible / unknown

禁止:

rank / activationをaccepted factへ変換。

### T4 — M3 -> M1

唯一重要なpromotion path。

```text
Proposal / feedback / Observation / proof
       ↓
owner resolution
       ↓
minimum candidate delta
       ↓
Shared Acceptance Boundary
       ↓ accepted only
M1 authoritative owner source
```

禁止:

- Observation auto-backfill
- proof auto-fix
- AI direct write
- learned association auto-rule

### T5 — M3 -> M2 learning

将来限定的に許可し得る。

```text
past episodes
  -> task-local / outcome-local association
  -> retrieval hint
```

必須:

- episode provenance
- invalidation
- disable / rebuild capability

禁止:

Decisionをweightだけへ保存。

## 10. Invalidation matrix

### M1 changes

M1のaccepted sourceが変わった場合:

```text
M2
  -> affected indexes / graphs / embeddingsをstaleにする
  -> rebuild / re-evaluate

M3
  -> historical episodeは保持可能
  -> current applicabilityは再評価
  -> past resultをcurrent resultとして再利用しない
```

### M2 changes

retrieval algorithm / index versionが変わった場合:

```text
M1 -> unchanged
M3 -> historical retrieval run remains historical evidence
```

accepted systemのmeaning changeを発生させない。

### M3 changes / deletion

chat履歴、Proposal history、benchmark result等を削除しても:

```text
M1 accepted system -> reconstructable
M2 -> M1から再生成可能な範囲は再生成
```

でなければならない。

### Environment / Release special case

M1には時点やenvironmentにより複数のaccepted authorityがあり得る。

```text
current Application
production Release 3.8.0
staging Release candidate
```

を一つの最新版へ畳まない。

## 11. Stalenessはmemoryごとに意味が違う

### M1 staleness

原則として「staleだから捨てる」ではない。

Release等は過去時点authorityとして有効。

必要なのはscope / time identity。

### M2 staleness

input M1 revisionと合わなければcurrent candidate generationに使わない。

再生成が基本。

### M3 staleness

過去episodeとしては古くならない。

ただしcurrent Decisionの根拠として再利用するときは再qualificationが必要。

```text
historically true
  != currently applicable
```

## 12. Identity rules

### M1 identity

qualified logical key / Canonical identity / concern-specific accepted identity。

file pathだけへ依存しない。

### M2 identity

M1 subject ref + representation version / input fingerprint等。

M2独自IDをCanonical identityへ昇格しない。

### M3 identity

Proposal / run / observation / review等のepisode identityを持てる。

そのIDをBusiness object IDとして流用しない。

## 13. Provenance requirements

### M2 minimum provenance

少なくとも:

- source M1 subject(s)
- input revision / fingerprint
- extraction / representation version where relevant

### M3 minimum provenance

少なくとも:

- base / evaluated snapshot
- referenced M1 subject(s)
- source mechanism / actor where relevant
- Evidence refs

AI固有:

- model / prompt / trajectoryは再現・評価に必要な範囲だけ
- secret / credential / unnecessary private dataを保存しない

## 14. Reference Pattern Memoryの二層化

Reference Patternを一つのstoreへ畳むとauthorityが曖昧になる。

望ましい論理分離:

```text
Pattern Observation / Provenance
  Reference Xでこの構造を観測した
  nature = observed / candidate
  evidence / caveat
        -> M3

Pattern Retrieval Representation
  structural signature
  index / embedding / sparse code
  rank
        -> M2
```

Current requirementへの採用:

```text
M2 candidate
  + M3 provenance
  -> current M1 qualification
  -> Proposal M3
  -> Acceptance
  -> M1 if accepted
```

これによりReference Modelをanswer libraryにしない。

## 15. Design Synthesis Memory separation

Design Synthesisでも同じ境界を使う。

### Accepted Semantic Snapshot

current accepted authoritiesを参照するinput。

**M1-backed view.**

### Derived Meaning Graph

cross-view relationの作業graph。

**M2.**

### Proof Result

snapshot-bound evaluation。

**M3 Evidence / recomputable derivative.**

### Candidate designs

まだacceptedでない。

**M3 Proposal.**

### DesignContextPacket

M1 facts + M2 derived context + M3 candidates / proofをLLMへboundedに渡すtransient packet。

**No new authority.**

Packetを保存してもM1へ昇格しない。

## 16. Feedback routing memory separation

P0-2と接続すると:

```text
M3 feedback / proof / quality / observation
        ↓
M2 exact neighborhood / retrieval assistance
        ↓
owner resolution against M1
        ↓
M3 candidate
        ↓
Shared Acceptance
        ↓
M1 owner source
        ↓
M2 invalidate / rebuild
        ↓
M3 re-observe / close
```

重要:

```text
owner resolution against M1
```

であり、過去trajectoryだけでownerを決定しない。

## 17. Current risk audit

### Risk 1 — Source Index becomes hidden authority

症状:

Editor / AIがSource Indexにある情報だけを信じ、Canonical validationよりIndexを優先する。

Guard:

- Source IndexはM2
- deletion / rebuild test
- Semantic validationはIndex非依存

Current design:

**境界は明示済み。**

### Risk 2 — AI Proposal store becomes second truth

症状:

Accept後もsourceではなくProposal JSONを次のAIが正本として読む。

Guard:

- accepted meaningはM1へ書く
- ProposalはM3
- AIなしreloadを完了条件にする

Current design:

**ADR-0022 / ISSUE-000027で防止方針あり。**

### Risk 3 — Learned memory hides Decision

症状:

「前回こうした」がweightへ入り、current contextでも自動適用される。

Guard:

- learned stateはM2
- Decision historyはM3
- accepted ruleはM1
- M2 disable時もM1不変

Current design:

**future risk / implementation前。**

### Risk 4 — Proof result persists past its snapshot

症状:

過去revisionの`supported`をcurrent accepted factとして使う。

Guard:

- proofはM3 snapshot-bound
- M1 changeで再評価

Current design:

**Design Synthesis文書でsnapshot binding方針あり。**

### Risk 5 — Observation becomes Desired

症状:

actual DB / deployment stateを「現実だから正しい」とcurrent definitionへコピー。

Guard:

- ActualはM3
- P0-2 owner routing
- Proposal / Acceptance経由

Current design:

**Desired / Actual分離あり。**

### Risk 6 — Release demoted to mere history

症状:

ReleaseをM3 episodeとみなし、current Applicationから自動再生成して過去内容を失う。

Guard:

- promoted ReleaseはM1 temporal authority
- creation assistanceとpost-promotion authorityを分ける

Current design:

**P0-1で明示。**

### Risk 7 — Native asset representation replaces native authority

症状:

SQL embedding / analyzer IR / normalized queryを正本にし、native SQLとの関係を失う。

Guard:

- SQL-authoring modeではSQLはM1
- analysis / embedding / IRはM2
- source traceを保持

Current design:

**Architecture方針と整合。**

### Risk 8 — Chat transcript becomes only requirement record

症状:

AIとの会話に重要DecisionがあるがM1 sourceへ反映されず、次回はchat memoryがないとsystem meaningを説明できない。

Guard:

- chat / trajectoryはM3
- accepted DecisionはM1 owner sourceへ反映
- chat削除後reconstructability

Current design:

**principleあり、Reference EvidenceはGate C後。**

### Risk 9 — Reference Pattern frequency becomes authority

症状:

多くのReferenceにある構造をcurrent Requirementとして自動採用。

Guard:

- provenance M3
- retrieval M2
- current applicabilityはM1照合

Current design:

**P0-2 / ISSUE-000047方針と整合。**

### Risk 10 — Quality / benchmark score becomes meaning

症状:

retrieval accuracyやlatency targetがBusiness meaningを自動変更する。

Guard:

- Quality EnvelopeはM3
- score / benchmarkはM3/M2 evaluation
- Requirement変更は別Decision / Acceptance

Current design:

**Product Quality Architectureと整合。**

## 18. Memory deletion tests — conceptual

将来Referenceで少なくとも次をpressureする価値がある。

### Test A — delete M2

削除:

- Source Index cache
- cognitive index
- embedding / sparse code
- Derived Meaning Graph

期待:

```text
M1 accepted system unchanged
M2 rebuild possible
```

### Test B — delete AI / Studio M3

削除:

- Proposal cache
- chat trajectory
- Preview history

期待:

```text
accepted source reload succeeds
accepted meaning unchanged
```

### Test C — stale M2

M1変更後、古いindexを利用。

期待:

```text
stale detected / regenerated
not silently used as current truth
```

### Test D — replay old M3

過去Proposalを新revisionへapplyしようとする。

期待:

```text
base mismatch
-> reject / rebase / regenerate / manual resolution
```

### Test E — observed state changes

Actualだけ変更。

期待:

```text
M1 unchanged
feedback / diff M3 changes
```

### Test F — promote candidate

Proposal accepted。

期待:

```text
M1 owner source changes
Proposal may remain M3 history
M2 invalidated / regenerated
```

## 19. Experience reuse policy

M3を価値ある学習資源として使うことは否定しない。

順序:

```text
Episode corpus
  ↓
extract bounded association
  ↓
M2 candidate-memory update
  ↓
current retrieval
  ↓
M1 qualification
```

直接:

```text
Episode corpus
  -> M1 rule learning
```

を標準にしない。

### Good initial learning targets

- false-positive suppression
- task-local candidate ranking
- likely relevant owner candidates
- Reference pattern recall priority

### Bad initial learning targets

- Business Rule generation
- Permission grant generation
- Handoff authoring
- Release dependency promotion
- migration acceptance

## 20. Human memoryも例外ではない

memory separationはAIだけの問題ではない。

人が:

```text
「前回こうだったはず」
```

と覚えていることもM3的経験である。

accepted sourceと矛盾する場合、記憶をauthorityへしない。

Studio UXは過去Decision rationaleを提示できても、current M1 refを同時に見せる方が望ましい。

## 21. Durable Decisionの置き場所

Decision record自体がM3でも、Decisionがsystem behaviorを定めるなら、その**結果**はM1 owner sourceへ反映される必要がある。

例:

```text
Decision:
  order_entryからcustomerはread-onlyにする
        -> rationale record M3

Application source:
  datasource master access read
        -> M1
```

rationaleを失ってもbehaviorは再構築できる。

rationaleがあると「なぜ」を説明できる。

この分離を目指す。

## 22. Unknown / unresolvedの保存

`unknown`をM1 factへしない。

current reviewで:

```text
actor unknown
Handoff unresolved
external actuality not observed
```

なら、M3のunresolved stateとして保持できる。

後にEvidenceが増えたときM1 candidateを作る。

空欄を埋めるためだけにM1へplaceholder factをauthorしない。

## 23. Security / privacy boundary

M2 / M3は大量のderived / historical dataを持ち得るため、M1より情報漏洩リスクが低いとは限らない。

保存しない / 最小化するもの:

- credential
- password
- raw secret
- unnecessary DSN
- unnecessary personal / business payload
- promptへ不要なprivate data

Resource handleやlogical identityで足りる場合はmaterialを複製しない。

memory分類はsecurity classificationの代替ではない。

## 24. Storage architectureを今は固定しない

三memoryは論理境界であり、次を意味しない。

- three databases
- three services
- vector DB必須
- event store必須
- graph DB必須
- memory server必須

小規模Referenceではfile / in-memory derived index / test fixtureで十分な可能性がある。

storage選定は:

- size
- latency
- rebuild cost
- retention
- privacy
- concurrency
- deployment

Evidenceから決める。

## 25. Memory ownership matrix

| Information | M1 | M2 | M3 | Rule |
| --- | ---: | ---: | ---: | --- |
| Semantic Relation | ✓ | projection可 | change history可 | M1がauthority |
| Application Pattern | ✓ | index可 | proposal history可 | accepted PatternはM1 |
| Database Contract | ✓ | reverse index可 | version history可 | published boundaryはM1 |
| native SQL | ✓ | analyzer IR / embedding | execution history | authoring modeがSQLならM1 |
| Persistence Mapping | ✓ | impact projection | migration history | mapping choiceはM1 |
| Environment Binding | ✓ | dependency index | runtime observation | environment authorityはM1 |
| Release snapshot | ✓ | compatibility index | deployment episode | promoted releaseはM1 |
| Source Index |  | ✓ |  | rebuildable |
| Derived Meaning Graph |  | ✓ | proof run may reference | no authority |
| Reference Pattern index |  | ✓ | provenance observation | candidate only |
| embedding / sparse code |  | ✓ | evaluation run | removable |
| Proposal | accepted result only | candidate refs | ✓ | pre-accept M3 |
| proof result |  | recomputable view | ✓ | snapshot-bound |
| Quality Envelope |  |  | ✓ | claim Evidence |
| actual DB / deployment state |  |  | ✓ | Actual != Desired |
| chat / trajectory |  | optional retrieval index | ✓ | never sole truth |
| learned association |  | ✓ | training provenance | Decisionではない |

## 26. P0-3 findings

### Finding 1 — Current Architecture already has the right fault line

Source IndexをCanonical Worldから分離し、AI ProposalをShared Acceptance前へ置き、cognitive indexをderived / rebuildableにする設計は三memory分離と整合する。

### Finding 2 — Biggest risk is not data duplication but authority duplication

同じ文字列が二箇所にあることより:

```text
どちらを変更すれば意味が変わるのか不明
```

な状態が危険である。

### Finding 3 — Temporal authority needs special treatment

Releaseのように「過去のaccepted state」はM3 episodeではなくM1 authorityになり得る。

### Finding 4 — Derived and episodic are orthogonal dimensions in some artifacts

Proof resultやgoldenは:

- recomputable derivative
- historical Evidence

の両方になり得る。

したがって一つのstorage classへ無理に畳まない。

### Finding 5 — Learning should first improve M2, not mutate M1

脳科学由来方式やexperience learningの最初の適用先はcandidate recall / ranking / attentionであり、accepted Business Ruleではない。

### Finding 6 — Reconstructability is the strongest practical guard

```text
AI / cognitive index / Proposal historyを消しても
accepted systemを再構築できるか
```

をReferenceで繰り返し確認することが、第二正本化を防ぐ実用的なtestになる。

## 27. P0-3で固定しないもの

- memory store product
- memory service
- retention period
- vector database
- event sourcing
- graph database
- universal episode schema
- global revision
- learned-memory file format
- model-specific memory API
- automatic consolidation
- online learning

## 28. P0-3結論

生成AI・脳科学時代にTEIが長期的に保つべき構造は、知識を全部一箇所へ集めることではない。

```text
M1 accepted authority
  何を意味し、何を選び、何を公開し、何をreleaseしたか

M2 derived association / structure
  何が近く、何を参照し、次に何を見るべきか

M3 episode / experience
  何を観測し、何を試し、なぜaccept/rejectしたか
```

を分離しながら接続することである。

最も重要なpromotion ruleは一つだけでよい。

```text
M2 / M3
  -> candidate / feedback
  -> current M1 qualification
  -> Shared Acceptance
  -> M1 owner source
```

それ以外の暗黙write pathを増やさない。

次のP0-4では、X-TEA / XEAD / TEI Referenceから得たpattern evidenceを:

```text
M3 provenance observation
  -> M2 retrieval representation
  -> current M1 qualification
```

として引き渡すための最小contractを整理する。
