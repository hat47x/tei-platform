# Definition-Centered Feedback Routing Catalog — 2026-09

Status: **P0-2 baseline audit / review vocabulary / no production enum or schema**

Baseline: `main` after P0-1 Definition / Derived Artifact Inventory

Related:

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`
- `plan/design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`
- `docs/product-quality-architecture.md`
- `docs/business-system-design.md`
- `plan/design/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`

## 1. 目的

Definition-centered feedback loopのP0-2として、Reference / Studio / Driver / Runtime / Observation / AI / cognitionから得たfeedbackを、**最も目につく場所ではなく、その意味を所有する最小concernへ戻す**ためのreview catalogを整理する。

中心原則:

```text
visible symptom
    !=
source to edit
```

例:

```text
UIで顧客を編集できない
  -> Dataに編集可能flagを足す
```

とは限らない。

ApplicationのDatasource authorityがread-onlyなら、問題はUIではなくApplication / Pattern candidateの矛盾かもしれない。逆にSemantic relationが欠けているのにUIだけ修正するのも誤りである。

したがってfeedback loopを次へ分ける。

```text
Observation / Review / Proof / Diagnostic
        ↓
Feedback Evidence
        ↓
owner resolution
  semantic / business
  application
  contract
  realization
  target
  environment / authority
  release
  derived cognition
  actual-state handling
  unknown
        ↓
Candidate
        ↓
Preview / deterministic qualification
        ↓
Shared Acceptance Boundary
        ↓
owner source update if accepted
        ↓
reload / regenerate / re-observe
```

このcatalog自体はruntime routing engine、persisted enum、Canonical kindではない。

## 2. FeedbackとAuthorityを分ける

feedback sourceは、そのfeedbackが指すauthoritative sourceと同一とは限らない。

```text
feedback source
  human review
  cross-view proof
  Target test
  Runtime error
  external Observation
  Quality Envelope
  retrieval benchmark
  Reference Model pressure
  AI hypothesis

        !=

authoritative owner
```

例えばDatabase Actualからdriftを観測できても、Actual DBがSemantic meaningのauthorにはならない。

AIが矛盾を発見しても、AI Proposal storeが新しい正本にはならない。

cross-view proofが`contradiction`を返しても、proof resultからsourceへ直接patchしない。

## 3. Routingの三段階

### 3.1 Stage A — What happened?

まずfeedbackを観測事実として記述する。

悪い例:

```text
Data modelが間違っている
```

良い例:

```text
order_entry.customer_referenceからcustomer editを要求したが、
order_entry.master datasourceはread-onlyである
```

結論を先に入れない。

### 3.2 Stage B — Who owns the disputed meaning?

次を順に確認する。

```text
Business / Semantic meaningか
Application intentか
published Contractか
native assetか
Realization / Mappingか
Target-specific projectionか
Environment / Permission / Resourceか
Release-time snapshotか
Actual / Observation処理か
derived retrieval / cognitionか
```

一意に決められなければ`unknown owner`を許容する。

### 3.3 Stage C — What is the minimum accepted change?

ownerが確定しても、直ちにsourceを書かない。

```text
Feedback Evidence
  -> candidate
  -> semantic / realization / target / release diff
  -> impact
  -> Accept / Reject
```

変更はMinimum Semantic Delta / minimum owner deltaを基本とする。

## 4. Feedback Envelope — review-only concept

routing reviewで必要な情報を、概念上次のようにまとめる。

```text
Feedback Envelope
  source
    human | proof | runtime | observation | quality | cognition | reference
  observed subject / scope
  accepted snapshot / revision
  observation or contradiction
  evidence refs
  known authority limits
  owner candidate(s)
  owner status
    resolved | ambiguous | unknown
  prohibited shortcut(s)
  proposed next step
```

これはproduction schemaではない。

### 4.1 Quality Envelopeとの違い

Quality Envelopeは:

```text
subject
+ revision / context
+ quality property
+ probe / measure
+ Evidence
+ status
```

を保つ。

Feedback Envelopeは、その観測から**どのconcernを再検討するか**を扱う。

```text
Quality Envelope
  -> claim under conditions

Feedback Envelope
  -> review / owner-routing decision
```

どちらもCanonical authorityではなく、どちらからもsourceへ暗黙backfillしない。

### 4.2 Cross-view Proof Resultとの違い

Proof resultは限定された問いについて:

```text
supported
contradiction
unresolved
```

を返す。

Feedback routingはその後段である。

```text
proof contradiction
  -> Feedback Evidence
  -> owner resolution
  -> candidate
```

`contradiction`だけでは、どちらのaccepted sourceを変更すべきか決まらない場合がある。

`unresolved`なら、source変更ではなくEvidence acquisitionが正しいnext stepになり得る。

## 5. Routing class R1 — Semantic / Business definition defect

### Trigger

- Business factが表現できない
- Relation / cardinality / constraintがaccepted requirementと不一致
- Business responsibility / manual work / external continuationが欠落
- cross-view proofでBusiness meaningとData / Functionが両立しない

### Owner

対象に応じて:

```text
Semantic authoring source
Business authoring source / accepted Procedure source
```

### Candidate

- Relation追加 /修正
- Validation追加 /修正
- Business Action / responsibility meaning修正
- accepted Handoff source修正

### Prohibited shortcut

- UIだけを調整して欠落したData meaningを隠す
- DB columnがあるからSemantic Attributeを自動authorする
- Function sequenceからBusiness Handoffを発明する
- actor labelからAuthorization Roleを発明する

### Post-accept verification

- Canonical reload
- cross-view proof再評価
- affected Application / Persistence / Target Projection再生成
- relevant Reference / Conformance

### Concrete Reference case A — missing Relation

過去Prototype型のcase:

```text
ContactがCustomerを必ず参照すべき
but Semantic Relationが無い
```

ownerはUIではなくSemantic Relation。

Relationをaccepted sourceへ追加した後に:

- Action input
- Reference Selector
- Persistence impact
- Runtime referential integrity

がderived effectとして変わる。

### Concrete Reference case B — Business manual work

`order-entry.order-handling`のBP-04 / BP-09のようなmanual / external unresolved workは、Application Patternが無いから削除する対象ではない。

Business meaningがacceptedされるならBusiness owner側へ置く。

## 6. Routing class R2 — Application intent / access contradiction

### Trigger

Application interaction candidateが、declared Datasource / Contract / Capability intentと矛盾する。

### Owner

```text
Application definition
Application Pattern
Application-to-Contract dependency
```

必要に応じてDatabase Contract ownerとjoint review。

### Concrete Reference — customer edit conflict

Reference pressure:

```text
procedure step BP-02
  uses order_entry.customer_reference

experimental candidate
  requests read-write customer edit

application order_entry
  datasource master
  access read
```

このcaseでやってはいけないのは:

```text
customer Dataをread-writeに変更
master_data Contractを自動拡張
UIだけedit formを生やす
```

である。

まず問いを分ける。

```text
Q1. order_entryはcustomer edit責任を本当に持つか
Q2. 持つならmaster datasource authorityを変えるべきか
Q3. 別Application / commandへ委譲すべきか
Q4. Reference candidate自体が不適合か
```

現行accepted Applicationがread-onlyなら、candidate interactionを`contradiction`として落とすことも正しい。

### Prohibited shortcut

physical DBがwrite可能だからApplication write authorityを増やさない。

## 7. Routing class R3 — Database Contract / ownership mismatch

### Trigger

Applicationが必要とするoperationが、published Database Contractに存在しない、またはContractのwrite/read scopeと矛盾する。

### Owner

```text
Database Contract owner
```

ただしApplication requirementが誤っている可能性もあるため、joint proofが必要。

### Key distinction

```text
Data has attribute
  != Contract publishes attribute

physical table writable
  != Contract permits mutation
```

### Candidate

- Contract version更新
- Application dependency更新
- alternative existing Contractの利用
- requirement撤回

### Prohibited shortcut

Semantic DataからContract surfaceを全自動公開しない。

## 8. Routing class R4 — Realization / Provider mismatch

### Trigger

Canonical meaningは成立しているが:

- physical mappingが不適切
- provider-specific typeで意味保持できない
- SQL realizationがContractを満たさない
- Capability Bindingが欠落 / incompatible

### Owner

```text
Persistence Mapping
Query Realization
Capability Binding / Realization
Provider Adapter
```

### Concrete case — persistence

```text
sales.order.total : std.money
-> public.orders.total_amount : decimal
```

ここでphysical column名やstorage typeの問題が出ても、`std.money`のBusiness meaningをprovider都合で変えない。

### Concrete case — query realization

Contract Query `find_by_order`に対しSQL result mappingが誤っているなら、native SQL / realization mappingをreviewする。

Data AttributeをSQL resultに合わせてrenameするのは最後の選択である。

### Prohibited shortcut

Provider limitationをCanonical meaningへ昇格しない。

## 9. Routing class R5 — Target / Presentation-specific problem

### Trigger

- Webだけでinteractionが成立しない
- touch terminalで操作密度が高すぎる
- visual layout / accessibility issue
- Target Capability差によるprojection failure

### Owner

```text
Presentation / Interaction Projection
Target Adapter
Target-specific realization / hint where explicitly allowed
```

### Routing question

まず:

```text
同じCanonical meaningが別Targetでは成立するか
```

を確認する。

yesならTarget ownerが有力。

全Targetで同じmeaning lossが出るならSemantic / Applicationへ戻る可能性がある。

### Prohibited shortcut

- UI convenienceのためData cardinalityを変えない
- Target制約からBusiness Ruleを推論しない
- screenshot差分をSemantic defectとみなさない

### Post-accept verification

Target-specific Functional / Visual / Performance Evidenceを再取得する。

## 10. Routing class R6 — Security / Authority contradiction

### Trigger

- interactionを実行するPrincipalにauthorityが無い
- Business actorとSecurity identityを混同している
- AI / Editor / generated codeがPermission / Resource境界を迂回しようとしている

### Owner

状況により:

```text
Authorization Policy / Provider
Capability Binding
Environment Permission / Resource grant
Application entry authorization
```

### Concrete Reference — BP-06 / submit target

Business Procedure ReferenceではBP-06は:

```text
determine whether authenticated subject may execute submit action
supporting capability = security.authorize
protected target = order_entry.submit_order
```

という境界を圧力試験している。

ここから:

```text
actor = sales operator
-> role = sales
```

を生成してはいけない。

### Prohibited shortcut

- actor labelをSecurity Roleへ変換
- AI autonomyからwrite permissionを付与
- Capabilityが存在することからinvoke authorityを付与

## 11. Routing class R7 — Actual-state drift

### Trigger

Desired / accepted definitionとactual external stateが異なる。

例:

- DB schema drift
- deployment state divergence
- external service actuality不一致

### Owner

まず`Actual / Observation`として保持する。

次にdiffの意味から:

```text
Mapping / Migration candidate
Release / Deployment candidate
External reconciliation
or Definition review
```

へrouteする。

### Important rule

```text
Actual != Desired authority
```

driftが見つかったからといって、ActualをSemantic sourceへcopyしない。

### Concrete Reference

`reference/observed/production.deployment-state.json`等は固定actual-state fixtureであり、Canonical authoring変更だけを理由に更新しない。

### Prohibited shortcut

- actual column追加 -> Semantic Attribute自動追加
- running release -> desired release自動変更
- telemetry value -> Business Result自動確定

## 12. Routing class R8 — Runtime / Resource failure

### Trigger

- Capability execution failure
- dependency unavailable
- timeout
- Resource denial
- lease loss
- unknown external outcome

### Owner

failure topologyを先に解く。

```text
Business Result?
Named Outcome?
Execution Error?
Observation?
Resource / Permission?
Provider failure?
```

その後:

- Capability implementation
- Runtime / Provider
- Resource Broker / Environment
- Retry / takeover policy
- Business handling rule

のどこへ戻るか決める。

### Prohibited shortcut

- process exit 0をBusiness successにする
- HTTP 200をexternal completionにする
- timeoutを自動retryして二重副作用を起こす
- Resource denialをSemantic Capability欠落に変換する

### Post-action

Runtime fixでもSemantic source変更が無いことは正常である。

## 13. Routing class R9 — System result / external actuality gap

### Trigger

system上のcommand acceptance / resultはあるが、Business continuationに必要な外部・物理状態が未確認。

### Owner

まずObservation / Evidence acquisition。

必要なら:

- Observation Provider
- external correlation
- Business Procedure continuation condition

をreviewする。

### Concrete Reference — BP-07 / BP-08

```text
BP-07 request accepted application command
BP-08 verify relevant external state when continuation depends on actual state
```

この分離が示す通り:

```text
submit accepted
  != external actuality verified
```

### Prohibited shortcut

Command successからphysical / external completionを推論しない。

## 14. Routing class R10 — Responsibility Handoff contradiction

### Trigger

- Step sequenceはあるが責任継続条件が説明できない
- Handoff endpoint / conditionがaccepted sourceと矛盾
- Observationやactor changeからHandoffを推測している

### Owner

```text
accepted Business Handoff source
```

ただしContract ADR前のReferenceではcandidate / Pressure planeに留める。

### Prohibited shortcut

Handoffを次から生成しない。

- adjacency
- Function sequence
- actor difference
- lifecycle transition
- Observation alone

### Unresolved handling

explicit Business sourceが無ければ、`no handoff`ではなく`unresolved`になり得る。

## 15. Routing class R11 — Release / current-definition divergence

### Trigger

current Application / Database definitionと、既存Release snapshotが異なる。

### Owner

時間軸を確認する。

```text
current authoring
  != promoted historical release
```

past Releaseが当時のaccepted snapshotを表すなら、差異はdefectではない。

### Candidate

- new Release candidateを作る
- current definitionを修正する
- stale Reference artifactなら正規生成経路からrefreshする

### Prohibited shortcut

current definitionからpast Releaseを再導出して上書きしない。

### Important feedback type

これは「矛盾を直す」のではなく:

```text
expected temporal divergence
```

としてrouting不要 / no changeになる場合がある。

## 16. Routing class R12 — Derived retrieval / cognition miss

### Trigger

- exact existing assetが検索結果に出ない
- structurally similar Reference patternを拾えない
- rankingが不適切
- false positiveが多い
- novelty signalが誤る

### Owner

```text
Derived index
Feature extraction
Retrieval algorithm
Candidate ranking / cognition
```

### Prohibited shortcut

検索に出なかったからCanonical relationを追加しない。

```text
retrieval miss
  != missing semantic fact
```

### Post-change verification

- Retrieval oracle
- Qualification結果
- invariant that Canonical World is unchanged

## 17. Routing class R13 — Reference Pattern mismatch

### Trigger

X-TEA / XEAD / TEI Referenceから抽出したpatternが、current intentへ合わない。

### Owner

```text
Pattern candidate applicability
Reference Pattern Memory
routing / ranking
```

### Prohibited shortcut

- 頻出patternをRequirementへ昇格
- Reference modelにあるfieldをTEI Canonical kindへ追加
- pattern mismatchをcurrent Business requirementの誤りとみなす

### Correct outcome

`incompatible`または`unknown`でよい。

Reference patternは回答ではなく候補構造である。

## 18. Routing class R14 — Unsupported inference / false completeness

### Trigger

modelやAIが、空欄をもっともらしいfactで埋めている。

例:

```text
actor不明 -> nearby roleを採用
trigger不明 -> preceding eventを採用
Data trace無し -> dependency無し
Handoff無し -> adjacent stepをHandoff
```

### Owner

多くの場合、**source変更ではない**。

まずinferenceを撤回し:

```text
unknown / unresolved
```

へ戻す。

必要ならEvidence acquisition taskへrouteする。

### Prohibited shortcut

「完全なmodelを作る」ことを優先してcandidate factをauthoritative sourceへ書かない。

## 19. Proof ObligationからRoutingへの橋

Design Synthesis cross-view proofをroutingへ接続する基本表:

| Proof result | Routing default |
| --- | --- |
| supported | 通常はno change。candidateを残す根拠の一部 |
| contradiction | owner resolutionへ。どちらのsourceを変えるかは別問題 |
| unresolved | 原則Evidence acquisition / identity resolution。source changeを急がない |

### Example — P3 Function / Mutation Authority

```text
candidate Pattern requests customer write
Application datasource = read
```

proof result:

```text
contradiction
```

routing:

```text
Application intent / candidate interaction review
```

であり、Databaseをwrite-enableする命令ではない。

### Example — P5 System Result / Actuality

Observationが無い場合:

```text
unresolved
```

routing:

```text
Observation / Evidence acquisition
```

であり、Business Procedureを「complete」に書き換えない。

## 20. Quality EnvelopeからRoutingへの橋

Quality Envelope statusも修正命令ではない。

| Quality status | Routing interpretation |
| --- | --- |
| observed / passed | 通常no change |
| observed / failed | owner resolutionへ |
| unresolved | additional Evidence / scope clarificationへ |
| not-applicable | no change。適用条件を維持 |
| not-yet-observed | Evidence acquisitionへ。failure扱いしない |

### Example — Target performance

Web Targetでlatency failure:

```text
Target / Host / workload conditioned failure
```

ならまずTarget / Runtime ownerへrouteする。

Semantic modelを軽くするためにBusiness fieldを削除する、といった短絡をしない。

## 21. Owner Resolution Rules

### Rule 1 — 最小のmeaning ownerへ戻す

```text
symptom layer <= correction layer
```

とは限らない。

原因のmeaningを所有する最小concernを選ぶ。

### Rule 2 — cross-concernなら一方へ押し込まない

例えばApplication requirementとDatabase Contractが矛盾する場合:

```text
Application owner
vs
Contract owner
```

のjoint reviewが必要。

### Rule 3 — Actualはauthorではない

外部世界は重要なEvidenceだが、Observed shapeがDesired designを自動決定しない。

### Rule 4 — Derived cognitionはauthorではない

index / embedding / sparse code / LLM trajectoryは、candidate sourceにはなってもaccepted meaningのownerにはならない。

### Rule 5 — 時間的authorityを尊重する

past Release、migration、Decision等はcurrent sourceとの差だけで「古いから修正」としない。

### Rule 6 — unknown ownerを許容する

ownerが決められない場合:

```text
owner = unknown
next = evidence / authority resolution
```

とする。

誤ったownerへ高速にpatchするよりよい。

## 22. Candidate typeをownerに合わせる

feedbackを一種類の`SemanticPatch`へ畳まない。

概念上:

```text
Semantic candidate
Business candidate
Application candidate
Contract candidate
Realization candidate
Target candidate
Environment / Authority candidate
Release candidate
Derived-cognition candidate
Evidence acquisition task
No-change / expected divergence
```

があり得る。

これらをproduction union schemaへ固定することは本P0-2の目的ではない。

重要なのは、**candidate typeがowner concernを隠さないこと**である。

## 23. Previewの責務

Routing後のPreviewでは、最低限:

```text
feedback evidence
selected owner
proposed minimum delta
unchanged concerns
prohibited shortcut avoided
expected derived impacts
remaining unresolved
```

を確認する。

### Example

Customer edit contradictionを解くために、仮に新しいdedicated customer-update Applicationへ委譲するcandidateを選ぶなら:

```text
changed:
  Application interaction / navigation

unchanged:
  customer Semantic Data
  master_data read Contract used by order_entry
  physical table

remaining:
  new update Application authority / Contract requirement
```

のように、何を変えないかも明示する。

## 24. Post-Accept closure

feedback loopはsource updateで終わらない。

ownerごとにclosureを確認する。

### Semantic / Business

- reload
- Validation
- cross-view proof
- affected Projection

### Application / Contract

- dependency / authority validation
- Driver / Reference scenario

### Realization / Environment

- provider / binding validation
- actual execution where required

### Target

- Functional / Visual / Performance Evidence

### Release

- promoted snapshot consistency
- deployment / compatibility checks

### Cognition

- benchmark / retrieval oracle
- Canonical invariance

### Actual / Observation

- re-observe
- distinguish resolved / still divergent / unknown

## 25. Reference routing matrix

| Reference symptom / pressure | Correct first owner | Do not mutate first |
| --- | --- | --- |
| missing required Semantic Relation | Semantic source | UI / DB only |
| `order_entry` customer edit vs read-only `master` datasource | Application intent / candidate interaction | customer Data / DB authority |
| missing Database Contract operation | Contract + Application joint review | physical DB capability alone |
| `std.money` cannot map losslessly to provider | Realization / Provider | Semantic money meaning |
| Web-only layout/accessibility issue | Target / Projection | Data cardinality |
| submit actor lacks Authorization | Security / Policy / Binding | Business actor label |
| Desired DB != Actual DB | Observation / Mapping / Migration review | Semantic source auto-backfill |
| command accepted but external state unknown | Observation / Evidence | Business completion flag |
| Step adjacency suggests Handoff only | unresolved Business Evidence | Handoff auto-authoring |
| current Application differs from past Release | temporal review / often no change | historical Release overwrite |
| existing component not retrieved | derived cognition / index | Canonical relation |
| Reference pattern near-match fails applicability | pattern qualification | Requirement / Canonical kind |
| proof result unresolved | Evidence acquisition | speculative source completion |

## 26. Automation boundaries

将来自動化できるのは、まず**routing assistance**までとする。

安全な候補:

- feedback subjectから関連owner candidatesを列挙
- exact Source / Canonical / Binding refsを集める
- forbidden shortcutを警告
- proof / Quality Evidenceを束ねる
- candidate typeを提案

慎重に扱う:

- ownerの自動確定
- sourceの自動変更

禁止する標準経路:

```text
Observation -> automatic semantic write
Proof contradiction -> automatic patch
Quality fail -> automatic requirement change
Retrieval miss -> automatic model expansion
Reference pattern -> automatic adoption
```

## 27. AI / cognitionの役割

AIはroutingで有用だがauthorityではない。

適した役割:

- symptomとpossible ownerの比較説明
- multiple owner candidatesの列挙
- exact refsを使ったimpact説明
- Minimum Delta候補作成
- unresolved questionの言語化

非LLM cognitionは:

- relevant local neighborhood recall
- similar past feedback / Reference retrieval
- candidate owner attention

に利用できる。

どちらも:

```text
routing suggestion
  != owner Decision
```

である。

## 28. Feedback routingと学習

将来experience memoryからroutingを学習する場合も、最初は局所的なassociationに限定する。

例:

```text
"read-only datasource + edit candidate"
  -> Application authority reviewが過去に頻出
```

というranking hintは持てる。

しかし:

```text
このpatternだから必ずApplicationが誤り
```

というDecision ruleへ自動昇格しない。

Decisionはcurrent Source / Evidenceで再評価する。

## 29. P0-2で固定しないもの

- `FeedbackEnvelope` production schema
- feedback category enum
- owner enum
- generic routing engine
- automatic fixer
- universal Patch type
- Design Synthesis専用authority path
- LLM judgeによるowner確定
- graph DB
- event sourcing
- global correlation ID
- one global semantic revision

Referenceでdurable needが出るまで固定しない。

## 30. P0-2結論

Definition-centered feedback loopを高度化するうえで重要なのは、feedback量を増やすことではなく、**誤った層へfeedbackを戻さないこと**である。

現行TEIの分離を使うと、少なくとも次を維持できる。

```text
Semantic defect        -> Semantic owner
Business gap           -> Business owner
Application conflict   -> Application / Contract owner
Provider mismatch      -> Realization owner
Target issue           -> Target owner
Authority issue        -> Security / Environment owner
Actual drift           -> Observation / reconciliation path
Release divergence     -> temporal authority review
Retrieval miss         -> derived cognition owner
Unsupported inference  -> unresolved / Evidence acquisition
```

そして:

```text
feedback source != author
contradiction != patch instruction
unresolved != false
actual != desired
similarity != applicability
quality failure != requirement rewrite
```

を不変条件とする。

次のP0-3では、このroutingで参照される情報を三種類のmemory:

```text
authoritative semantic memory
derived associative / structural memory
episodic / experience memory
```

へ具体的に割り当て、同じmeaningが複数memoryでauthority化していないかを監査する。
