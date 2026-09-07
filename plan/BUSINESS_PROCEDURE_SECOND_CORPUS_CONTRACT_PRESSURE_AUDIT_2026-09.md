# Business Procedure Second Corpus Contract Pressure Audit — 2026-09

Status: **Lane C / outline-level comparison + test-only Observation pressure / no Reference fixture / no production schema adoption**

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

関連:

- `BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`
- `BUSINESS_PROCEDURE_SECOND_CORPUS_DATA_AUTHORITY_DEFERRAL_2026-09.md`

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

### Data authorityは意図的に保留する

Second corpus outlineにはTEI Canonical Data / Relation / Database / Pattern identityがまだない。

したがって:

```text
direct Data ref is required
direct Relation ref is required
physical stock is external-observed
manufacturing completion is managed-here
```

とは言えない。

Current `order_entry`ではexact World identityを通して`managed-here + read-write`と`external-observed + read`をcross-view検査できるが、そのshapeをmanufacturing / inventoryへsyntheticに移植しない。

D4のcurrent minimum:

```text
businessAction
applicationPatternRefs[]
validationRefs[]
capabilityRefs[]
```

をこの段階で拡張しない。

将来TEI-original sourceでBusiness meaningを保持するためにdirect Data / Relation identityが不可欠になった場合だけfail closedしてreopenする。詳細は`BUSINESS_PROCEDURE_SECOND_CORPUS_DATA_AUTHORITY_DEFERRAL_2026-09.md`を正とする。

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

### まだ証明しないもの

しかしexact Procedure / Step / conditionは未確定である。

したがって:

```text
Procedure A / Step X
  -> Procedure B / Step Y
  when condition Z
```

というaccepted Handoff relationをまだ作らない。

さらに:

```text
physical completion observed
  != accepted responsibility handoff
```

とする。Observation pressureやscreen adjacencyからH1/R3を自動生成しない。

Current H1/R3を拡張も縮小もしない。

## 6. Command / Observation separation

ここはsecond corpusで最も強く再現し、今回test-only mechanical pressureまで進めた。

```text
application command accepted
  != material physically received
  != manufacturing physically completed
  != stock physically present
  != shipment physically completed
```

`physical_world_observation_boundary_pressure_test.go`では:

```text
Command accepted + no explicit observation
  -> BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION

Observation read success + observed=false
  -> BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED

Observation read success + external-observed + observed=true
  -> test-local fact may be treated as observed

Observation read error
  -> no physical completion result
```

をPressureする。

これはcurrent `order_entry` BP-08で置いた:

```text
Business Result != actual external Observation
```

の境界を別domainから支持する。

ただし、この再現から:

```text
Observation Canonical identity
physical-world state engine
workflow completion state
```

を導入しない。

既存Driver `ExecuteObservationRead` / `ObservationReadExecution`を利用し、runtime ReadIdentityはEvidence locatorのままとする。詳細は`BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`を正とする。

## 7. Business Flow / DFD

Manufacturing / inventoryでは物・document・情報のmovementが重要になる可能性が高い。

しかしoutline-level factだけでは、Procedure/Handoffでは失われるaccepted flow meaningをexact relationとしてまだ示せない。

したがってT-GAP-06は:

```text
relevant pressure exists
but contract insufficiency is not yet demonstrated
```

とする。

full Business Flow / DFD schema、DataflowNode / DataflowLine相当、generic Edgeを導入しない。

## 8. Cross-view contradiction候補

Second corpusがfixtureへ進む場合、最初に価値があるのはfeature追加ではなく矛盾検出である。

候補:

```text
C-01
Business says physical completion is required
but Function only records command acknowledgement
  -> test-only pressureへ進展

C-02
Business depends on external-observed stock
but Application claims managed-here write authority
  -> exact TEI Data / Pattern authority待ち

C-03
Business action is manual / physical
but projection invents a Capability

C-04
Business accepted meaning requires direct authority identity
but D4 silently drops it

C-05
Business continuation depends on a real-world condition
but Handoff is inferred from screen adjacency
```

C-01以外はfixture作成時にbaseline / intentional defectを分離して検査する。

## 9. Current candidate stackへの判定

| Candidate | Second corpusで再現したもの | 現時点で不足と証明したもの | 判断 |
| --- | --- | --- | --- |
| D4 | manual / physical / external Business Action | direct Data / Relation ref necessity | keep |
| S4-A | future exact step addressability question | global Step ID necessity | keep |
| H1/R3 | responsibility continuation / condition pressure | exact accepted Handoff relation | keep narrow |
| O1 | role-separationを問う必要性 | second cross-domain protected-target example | do not generalize |
| Observation boundary | command != physical actuality | new Canonical Observation identity | test-only mechanical pressureへ進展、schema expansionなし |
| Data authority | physical state authority relevance | exact TEI Data / ownership mapping | defer; synthetic ownership禁止 |
| Business Flow / DFD | material/info movement relevance | Procedure/Handoffではlossyというexact case | defer |

## 10. Gate 15c

本監査ではcurrent stackのproduction contract insufficiencyをまだ発見していない。

したがって:

```text
Gate 15c = not activated
```

を維持する。

これはsecond corpusの価値がないという意味ではない。

現在の価値は:

1. physical-world caseでもBusiness ActionをFunctionへ還元しない境界が再現した
2. Command / Observation separationを別domainでtest-only mechanical pressureまで進めた
3. O1 generic role taxonomyの根拠がまだ一例に留まることを明確にした
4. Data authorityをexact identityなしにsynthetic化しない境界を固定した
5. physical ObservationからHandoff / Business Flowを先走って一般化しないnegative Evidenceを得た

ことにある。

## 11. 次にfixture化できる条件

actual Repository pressure testsがgreenになった後、B-F19bを進める場合だけTEI-original fixtureへ進む。

その際の最初の目的はcandidate stackを飾ることではなく、次のいずれかを反証することである。

```text
D4 is sufficient
O1 remains one-domain special pressure
H1/R3 remains handoff-only sufficient
Procedure/Handoff can preserve enough meaning without Business Flow
current Data authority derivation remains sufficient
```

具体的不足が出ない限りproduction schemaを増やさない。

## 12. 現時点の結論

Second corpusから得られた最も重要な結論は、新しいnounやrole enumではない。

```text
physical-world pressure reinforces existing boundaries
Command/Observation can be mechanically pressured without new Canonical identity
Data authority must remain unresolved until exact TEI identities exist
responsibility continuation does not itself create Handoff
```

特にO1について:

```text
order_entry protected-target pressure = real
cross-domain repeated role taxonomy   = unproven
```

である。

したがってO1は引き続きnarrow / test-only / source-persistence deferredとする。
