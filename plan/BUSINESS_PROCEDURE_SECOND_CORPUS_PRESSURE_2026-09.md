# Business Procedure Second-Corpus Pressure — 2026-09

Status: **Lane C / outline-level comparison + test-only negative pressure / no Reference fixture / no production schema adoption**

## この文書について

本書は、manufacturing / inventory second-corpus outlineをcurrent candidate stack（D4 / S4-A / H1・R3 / O1）へ当てた調査を1本へ統合したものである。

1. `BUSINESS_PROCEDURE_SECOND_CORPUS_CONTRACT_PRESSURE_AUDIT_2026-09.md`（元・全candidateを横断する監査、他2文書が掘り下げる論点をここで提起）
2. `BUSINESS_PROCEDURE_SECOND_CORPUS_DATA_AUTHORITY_DEFERRAL_2026-09.md`（元・Data authorityの先走り禁止を掘り下げ）
3. `BUSINESS_PROCEDURE_SECOND_CORPUS_HANDOFF_INFERENCE_PRESSURE_2026-09.md`（元・H1/R3のadjacency/Observation推論禁止を掘り下げ）

Part 1（監査）がData authorityとHandoff inferenceそれぞれの論点を提起し、Part 2・3がその詳細pressureを行うという親子関係にあるため統合した。個別ファイルの履歴はGit historyから追跡できる。

外部から旧ファイル名を参照していた文書（`plan/issue/000042-business-procedure-cross-view-reference.md`、`plan/BUSINESS_ACTION_DATA_PROJECTION_PRESSURE_2026-09.md`、`plan/LANE_B_A70_A72_SECOND_CORPUS_OUTLINE_AUDIT_2026-09.md`、`plan/LANE_B_A73_A79_PHYSICAL_OBSERVATION_HANDOFF_AUTHORITY_AUDIT_2026-09.md`、`plan/DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`）は本統合と同じPRで本ファイル名へ参照を更新した。

---

# Part 1 — Second Corpus Contract Pressure Audit（横断監査）

## 1. 目的

`BUSINESS_PROCEDURE_MANUFACTURING_INVENTORY_CORPUS_PRESSURE_OUTLINE_2026-09.md`を、current candidate stack:

```text
D4
S4-A
H1 / R3
O1
```

へ当て、`order_entry`とは異なるmanufacturing / inventory / physical-world pressureから、現時点で本当にcontract不足が見つかるかを監査する。

本監査はsecond corpus fixtureではない。

Outline自身が定める:

```text
actual green前にfixture schemaへ昇格しない
fact / candidate / unresolvedを混ぜない
external artifactのexact sequenceを再現しない
```

を維持する。

## 2. 比較単位

公開provenanceから利用できるhigh-level pressureは少なくとも次である。

```text
purchase / receipt
material issue
manufacturing instruction / completion
warehouse stock / transfer
stocktaking
shipment
accounting continuation
payment approval / receipt
```

ただしexact actor、trigger、sequence、approval policy、Application Function realization、TEI Data ownership、Observation contractはunresolvedである。

したがって本監査では「ありそうなFunction / Data identity」を発明しない。

## 3. D4 — residual Business meaning

### 再現するpressure

Physical-world workでは、Application FunctionがなくてもBusiness Actionが成立し得る。

例としてoutline-levelには:

```text
material physically received
material issued to work
manufacturing physically completed
finished goods physically present
shipment physically completed
```

という意味差がある。

これはcurrent D4方針:

```text
businessAction is primary Business residual
manual / physical / external action does not require fake Function
```

を支持する。

### まだ証明しないもの

Second corpus outlineにはTEI Canonical Data / Relation identityがまだない。

したがって:

```text
direct Data ref is required
direct Relation ref is required
```

とは言えない。

D4のcurrent minimum:

```text
businessAction
applicationPatternRefs[]
validationRefs[]
capabilityRefs[]
```

をこの段階で拡張しない。

さらにcurrent `order_entry`では、Application Pattern / Validation / Capabilityの既存accepted refsからBusiness Action × Data / Relation traceをderived projectionできることをtest-onlyで確認した。

```text
current refs
  -> useful Data/Relation trace

no current trace
  -> unknown / no-trace
  != no Data dependency
```

したがってData review traceを得るためだけにD4へdirect Data / Relation slotを戻す必要性は弱まった。

将来TEI-original fixtureでBusiness meaningを保持するためにdirect Data / Relation identityが不可欠になった場合だけfail closedしてreopenする。

## 4. O1 — protected target role

### 問い

`order_entry` BP-06では:

```text
security.authorize        = supporting semantic authority
order_entry.submit_order  = protected Application target
direct Function realization = none
```

というsame authority kind / different role pressureが生じた。

Second corpusでも同種の:

```text
Application Pattern X
  is explicitly the subject / target / concern of Business Action
  but does not realize that Business Action
```

が反復するかを確認する。

### 現在の結果

**反復はまだ証明されない。**

理由:

1. manufacturing / inventory outlineはexact Application Pattern identityをまだ持たない。
2. approvalやphysical completionにAuthorization / Observation的な境界はあり得るが、exact policy / Function realizationはunresolvedとして明示されている。
3. `shipment`や`manufacturing completion`をもっともらしいApplication Patternへ結び、target roleを作るとsource以上の意味を発明する。

したがってcurrent Evidenceは:

```text
O1 is a real order_entry pressure
O1 is not yet a repeated cross-domain role pattern
```

に留まる。

BP-06 Data projectionは`security.authorize`からsecurity-side Dataをtraceするが、O1 targetがauthoritative Business sourceへpersistしていないため`sales.order`を隣接BP-07から補わない。

これはO1 persistenceがimpact projectionへ意味を持つ可能性を示すが、projectionを豊かにするためだけにO1を採用してはならない。

### 設計判断

- generic `ExplicitSemanticRef { Kind, Role, Target }`は採用しない。
- generic Role enumは採用しない。
- manufacturing/inventory向けにsynthetic protected targetを作らない。
- O1 production source fieldもまだ追加しない。

## 5. H1 / R3 — Business Handoff

### 再現し得るpressure

Outlineには:

```text
material available
physical manufacturing completion confirmed
stock available
shipment continuation
```

のようにresponsibility continuationの前提となるBusiness factがある。

これはH1/R3の問いを別domainでも自然に生じさせる。

### authority direction

Test-only pressureでは次を固定した。

```text
Procedure bodies + ordered steps
explicit Handoff source absent
  -> normalized Handoffs = 0

positive physical Observation
explicit Handoff source absent
  -> normalized Handoffs = 0

explicit Handoff source present
  -> normalized Handoff relation
```

したがってStep adjacency、physical Observation、Application Function sequenceからHandoff endpoint / conditionを生成しない。

### まだ証明しないもの

Second corpus outlineにはexact accepted Procedure / Step / condition relationがない。

Current H1/R3のauthority separationは補強されたが、production `BusinessHandoff` Canonical kind necessityは未証明である。

## 6. Command / Observation separation

ここはsecond corpusで最も強く再現し、test-only mechanical pressureまで進んだ。

```text
application command accepted
  != material physically received
  != manufacturing physically completed
  != stock physically present
  != shipment physically completed
```

`physical_world_observation_boundary_pressure_test.go`では:

```text
Command accepted + no Observation
  -> BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION

Observation read success + observed=false
  -> BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED

positive external Observation
  -> test-local physical fact observed

Observation read error
  -> no completion result
```

をPressureする。

既存Driver `ExecuteObservationRead` / `ObservationReadExecution`を再利用し、Observation専用Canonical identityやphysical-world state engineを追加しない。

## 7. Data authority

Physical state authorityのrelevanceは強いが、second-corpus outlineにはexact TEI-original Data / Pattern / Database identityがない。

したがって:

```text
inventory = external-observed
production completion = managed-here
shipment state = external-observed
```

等をsyntheticに割り当てない。

Current `order_entry`でmanaged-here/read-writeとexternal-observed/readをcross-viewできるのは、exact Application / Database / Pattern identityが既に存在するからである。

Business Action × Data derived projectionもownershipを所有しない。Data identityがtraceできてもAccess / Ownership / WritePolicyをprojectionへ移さない。

Data authorityの詳細な扱いは Part 2 を参照する。

## 8. Business Flow / DFD

Manufacturing / inventoryでは物・document・情報のmovementが重要になる可能性が高い。

しかしoutline-level factだけでは、Procedure/Handoffでは失われるaccepted flow meaningをexact relationとしてまだ示せない。

したがってT-GAP-06は:

```text
relevant pressure exists
but contract insufficiency is not yet demonstrated
```

とする。

full Business Flow / DFD schema、DataflowNode / DataflowLine相当、generic Edgeを導入しない。

## 9. Cross-view contradiction候補

Second corpusがfixtureへ進む場合、最初に価値があるのはfeature追加ではなく矛盾検出である。

候補:

```text
C-01
Business says physical completion is required
but Function only records command acknowledgement

C-02
Business depends on external-observed stock
but Application claims managed-here write authority

C-03
Business action is manual / physical
but projection invents a Capability

C-04
Business accepted meaning requires direct authority identity
but D4 silently drops it

C-05
Business continuation depends on a real-world condition
but Handoff is inferred from screen adjacency / Observation

C-06
Business Action has no current Data trace
but projection reports "no Data dependency"

C-07
Derived Data trace is treated as complete / authoritative ownership
```

これらはfixture作成時にbaseline / intentional defectを分離して検査する。

## 10. Current candidate stackへの判定

| Candidate | Second corpus / current projectionで再現したもの | 現時点で不足と証明したもの | 判断 |
| --- | --- | --- | --- |
| D4 | manual / physical / external Business Action、existing refsからのderived Data trace | direct Data / Relation ref necessity | keep; smaller shape reinforced |
| S4-A | future exact step addressability question | global Step ID necessity | keep |
| H1/R3 | responsibility continuation / explicit relation authority | production Canonical Handoff kind necessity | keep narrow |
| O1 | role-separationを問う必要性、partial Data projectionへの影響 | second cross-domain protected-target example | do not generalize |
| Observation boundary | command != physical actuality、mechanical negative pressure | new Canonical Observation identity | strongly reinforced, no schema expansion |
| Data authority | exact authorityが必要という境界 | second-corpus exact ownership | defer（→ Part 2） |
| Business Flow / DFD | material/info movement relevance | Procedure/Handoffではlossyというexact case | defer |

## 11. Gate 15c

本監査ではcurrent stackの具体的production contract insufficiencyをまだ発見していない。

したがって:

```text
Gate 15c = not activated
```

を維持する。

これはsecond corpusの価値がないという意味ではない。

現在の価値は:

1. physical-world caseでもBusiness ActionをFunctionへ還元しない境界が再現した
2. Command / Observation separationが別domainでも強く再現しtest-onlyで機械化した
3. O1 generic role taxonomyの根拠がまだ一例に留まることを明確にした
4. Handoff authorityをexplicit sourceへ限定できた（→ Part 3）
5. current existing refsからderived Data traceがかなり得られ、D4 Data/Relation slot再追加の必要性が弱まった
6. Data authorityはexact identity不足のためsynthetic化すべきでないことを固定した（→ Part 2）

ことにある。

## 12. 次にfixture化できる条件

actual Repository pressure testsがgreenになった後、B-F19bを進める場合だけTEI-original fixtureへ進む。

その際の最初の目的はcandidate stackを飾ることではなく、次のいずれかを反証することである。

```text
D4 is sufficient
O1 remains one-domain special pressure
H1/R3 remains handoff-only sufficient
Procedure/Handoff can preserve enough meaning without Business Flow
existing refs can derive enough Data trace without authored Data duplication
```

具体的不足が出ない限りproduction schemaを増やさない。

## Part 1 現時点の結論

Second corpusとcurrent Data projectionから得られた結論は、新しいnounやrole enumではない。

```text
physical-world pressure reinforces existing boundaries
current refs can derive useful Data trace
unknown can remain unknown
but neither requires a wider production contract yet
```

特に:

```text
order_entry protected-target pressure = real
cross-domain repeated role taxonomy   = unproven

Business Action x Data projection     = structurally useful
projection completeness               = unproven
D4 direct Data/Relation slots         = still unproven
```

である。

したがってO1はnarrow / test-only / source-persistence deferred、D4はsmaller shapeを維持する。

---

# Part 2 — Second-Corpus Data Authority Deferral（Data authorityの先走り禁止）

## 1. 目的

manufacturing / inventory second-corpus outlineには、physical-world stateを扱う必要性が強く現れる。

一方で、現時点のprovenance-backed outlineにはTEI-originalのexact identityとして次がまだない。

```text
Data
Relation
Database
Datasource
Application Pattern
Query / Contract Query
Capability
ownership boundary
```

この状態で「在庫はexternal-observed」「生産完了はmanaged-here」等をTEIのaccepted authorityとして作ると、公開sourceから確認できる高水準factを超えて設計者側が正本を発明する。

本書は、その先走りを禁止するためのdeferral recordである。

## 2. Current order_entryから既に証明されていること

Current Referenceではexact normalized World identityがあるため、次をcross-viewで検査できる。

```text
sales_core
  ownership = managed-here

order_entry / datasource sales
  access = read-write

master_data
  ownership = external-observed

order_entry / datasource master
  access = read

order_entry.customer_reference
  -> master datasource
  -> Contract Query
```

このためcustomer masterへwrite intentを持ち込めば、Business / Function probeとaccepted datasource policyの矛盾として検出できる。

ここで重要なのは、Business Procedure sourceがownershipやdatasource accessを複製していないことである。

## 3. Second corpusへそのまま移植しないもの

Manufacturing / inventory outlineに次を機械的にコピーしない。

```text
physical stock -> external-observed
manufacturing completion -> managed-here
shipment state -> external-observed
warehouse -> one Database
subsystem -> ownership boundary
```

これらはもっともらしくても、current outlineからaccepted TEI authorityとしては導けない。

X-TEAのSubsystemもdeployment / Databaseと一対一ではないため、歴史的namingからownershipを推測しない。

## 4. Reopen条件

Second corpus Data authority pressureをcode / fixtureへ進めるのは、TEI-original source上で少なくとも次がexactに与えられた場合だけとする。

1. Business Actionが依存するbusiness factが明示される
2. そのfactを読む／変更するApplication Patternまたはexplicit semantic authorityがある
3. Pattern owner Applicationをnormalized Worldからexact解決できる
4. datasource / target Databaseをtraceできる
5. Database ownershipとApplication accessがaccepted sourceから得られる
6. Business-side要求とaccepted authorityを比較できる

そのとき初めて次のようなdiagnostic pressureを作る。

```text
Business depends on externally observed state
  but Function attempts managed-here write

or

Business requires managed-here mutation
  but Application only has external-observed/read authority
```

## 5. D4との関係

Second corpus outlineだけではdirect Data / Relation refのminimum必要性も証明されない。

したがってD4は現在のまま:

```text
businessAction
applicationPatternRefs[]
validationRefs[]
capabilityRefs[]
```

を維持する。

将来、PatternやCapabilityからlosslessにderiveできないaccepted Business dependencyとしてData / Relation identityが必要になった場合は、D4が黙ってdropせずfail closedしexplicit-ref designをreopenする。

## 6. Physical Observationとの関係

`BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`は、Data authorityを決めずに:

```text
Command accepted != physical fact observed
```

だけをtest-onlyでPressureする。

これは意図的な段階分離である。

```text
Observation boundaryを守る
  !=
physical factのCanonical Data authorityを決める
```

## 7. Handoffとの関係

同様に、physical factが存在／観測されたことだけからH1/R3 relationを作らない。

```text
physical fact observed
  != responsibility transfer accepted
```

Handoffにはexact Procedure / Step endpointとaccepted conditionが別途必要である。詳細は Part 3 を参照する。

## Part 2 現時点の判断

```text
Data authority relevance
  strong

exact second-corpus TEI authority
  absent

synthetic ownership assignment
  prohibited

D4 Data/Relation expansion
  not justified

Gate 15c
  not activated
```

これはEvidence不足を埋めるための保留ではなく、P11の「unresolved remains unresolved」とX-TEA fallbackの「historical nounをTEI nounへ一対一変換しない」を守る設計判断である。

---

# Part 3 — Second-Corpus Handoff Inference Pressure（adjacency/Observation推論の禁止）

## 1. 目的

manufacturing / inventory second-corpus outlineでは、physical completionやstock availabilityがresponsibility continuationの前提になり得る。

しかし次を区別する必要がある。

```text
Business fact observed
  != responsibility transferred

Procedure Step adjacency
  != accepted Handoff
```

本Pressureは、H1/R3をscreen / Step順 / Observationから推測せず、explicit Business Handoff authorityだけからnormalized relationを作ることを固定する。

## 2. Current H1/R3 derivation

Current test-only normalized Business modelはHandoffを次からだけ構成する。

```text
referenceSet.Handoffs[]
  -> From { Procedure, Step }
  -> To   { Procedure, Step }
  -> Condition
```

Procedure body、Step adjacency、Application Pattern、Observation resultはHandoff generation inputではない。

これはcurrent implementation detailとして偶然残すのではなく、authority directionとしてPressureする。

## 3. H-INF-01 — adjacency does not author Handoff

Primary ProcedureではBP-08がBP-09より前にある。

Explicit Handoff sourceを除去してもProcedure bodyとStep orderは残る。

期待:

```text
ordered steps exist
BP-08 before BP-09
explicit Handoff source absent
  -> normalized Handoffs = 0
```

Step隣接からresponsibility transferを推測しない。

## 4. H-INF-02 — positive physical Observation does not author Handoff

Second-corpus test-local external Observationがpositiveでも:

```text
physical fact observed = true
explicit Handoff source absent
  -> normalized Handoffs = 0
```

とする。

ObservationはBusiness continuationの前提事実を確認できても、誰から誰へ、どのStepからどのStepへ、どのconditionでresponsibilityが移るかを所有しない。

## 5. H-INF-03 — explicit relation is required

同じProcedure fixturesで:

```text
referenceSet.Handoffs = []
  -> normalized Handoffs = 0

referenceSet.Handoffs = [explicit relation]
  -> normalized Handoffs = 1
  -> exact endpoint + condition validation
```

をPressureする。

したがってH1/R3 relationのauthority directionは:

```text
explicit accepted Handoff source
  -> normalized H1/R3 relation

Step order / adjacency
  -X-> Handoff

physical Observation
  -X-> Handoff

Application Function sequence
  -X-> Handoff
```

である。

## 6. Business Flowとの境界

この結果はH1/R3がBusiness Flow全体を表すことを意味しない。

H1/R3は依然として:

```text
Procedure/Step -> Procedure/Step + condition
```

だけを扱う。

material / document / information movementやexternal participantが必要になれば、T-GAP-06を別pressureとしてreopenする。

## 7. X-TEA fallbackとの整合

X-TEAではBusiness Model / Business Flow / Function利用が一つのruntime sequenceへ潰されていない。

TEIでも、Application操作順や観測結果からBusiness責任移管を自動推定せず、Business側でacceptされたrelationを独立して保持する方が三視点の相互批判性を保つ。

ただしこのEvidenceはproduction `BusinessHandoff` Canonical kindの採用を証明しない。

## Part 3 現時点の判断

```text
H1/R3 explicit authority direction
  test-onlyで明確化

adjacency inference
  reject

Observation inference
  reject

production Handoff Canonical kind
  still unproven

Gate 15c
  not activated
```

H1/R3は引き続きnarrow relation pressureとして維持する。

## Verification status

Repository上へ`second_corpus_handoff_inference_pressure_test.go`を追加したがactual runner greenは未取得であった（2026-09-20更新: Gate A actual executionは実行済みでPASS。R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、判定の正本は[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）。

`gofmt / go vet / go test / checker` PASSはまだ主張しない。
