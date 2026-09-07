# Business Procedure Second Corpus Contract Pressure Audit — 2026-09

Status: **Lane C / outline-level comparison / no Reference fixture / no production schema adoption**

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
| Data authority | exact authorityが必要という境界 | second-corpus exact ownership | defer |
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
4. Handoff authorityをexplicit sourceへ限定できた
5. current existing refsからderived Data traceがかなり得られ、D4 Data/Relation slot再追加の必要性が弱まった
6. Data authorityはexact identity不足のためsynthetic化すべきでないことを固定した

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

## 13. 現時点の結論

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
