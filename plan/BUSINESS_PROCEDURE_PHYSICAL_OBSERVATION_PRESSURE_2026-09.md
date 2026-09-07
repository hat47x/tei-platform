# Business Procedure Physical Observation Pressure — 2026-09

Status: **Lane C / test-only pressure / second-corpus outline level / no production schema adoption**

## 1. 目的

manufacturing / inventory second-corpus outlineで強く再現した次の境界を、既存Driver Observation contractへ接続して機械的にPressureする。

```text
Command accepted
  != material physically received
  != manufacturing physically completed
  != finished goods physically present
  != shipment physically completed
```

本Pressureはsecond-corpus Reference fixtureではない。

- external modelのexact Procedure sequenceを作らない
- production `Observation` Canonical kindを追加しない
- physical-world state engineを追加しない
- Workflow completion stateを追加しない

## 2. 既存authority

DriverはすでにCommand invocationとObservation readを分離している。

```text
CapabilityInvocation
  -> Business Result

ExecuteObservationRead
  -> explicit read identity
  -> Evidence nature
  -> read Result
```

`ObservationReadExecution.ReadIdentity`はruntime Evidence locatorでありCanonicalIDを要求しない。

またDriver testはread成功と期待状態一致を分離している。

したがってLane C側で新しいruntime contractを作らず、Business Procedure projectionがこの境界を壊さないことだけをPressureする。

## 3. test-only probe

`physical_world_observation_boundary_pressure_test.go`は次のtest-only shapeだけを持つ。

```text
physicalWorldCompletionPressure
  BusinessFact
  CommandAccepted
  ObservationResult?
```

これはproduction schemaではない。

### P-OBS-01 — Command acceptanceだけではphysical factを成立させない

次のoutline-level fact候補を同じ原則で検査する。

```text
material physically received
manufacturing physically completed
finished goods physically present
shipment physically completed
```

Commandがacceptedでもexplicit Observationがなければtest-only diagnostic:

```text
BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION
```

を出す。

### P-OBS-02 — read success != observed completion

Observation read自体が成功してもResultがfalseなら:

```text
BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED
```

とする。

Transport / Query成功をbusiness fact成立へ丸めない。

### P-OBS-03 — positive external observation

`external-observed` readでResult=trueの場合だけ、そのtest-local factをobservedとして扱える。

ただし:

```text
runtime ReadIdentity
  != Canonical Observation ID
```

を維持する。

### P-OBS-04 — Observation Execution Error

read errorはphysical completionへ変換しない。

Commandが成功済みでもObservation失敗をCommand Business Resultへ畳まない。

### P-OBS-05 — Data authority / Handoff shapeをObservation probeへ混入しない

`physicalWorldCompletionPressure`は構造上も次の三fieldだけを持つ。

```text
BusinessFact
CommandAccepted
ObservationResult
```

Testでは少なくとも次のfieldがprobeへ入らないことを固定する。

```text
Data / Relation / Database / Datasource / Ownership / ApplicationPattern
From / To / Procedure / Step / Condition / Handoff
```

これは:

```text
physical fact observed
  != exact TEI Data authority selected
  != accepted responsibility Handoff created
```

というA75 / A76境界の**representation pressure**である。

ただし、このreflection testだけで「将来どのコード経路からもHandoff推論が絶対に起こり得ない」と主張しない。意味するのは、このphysical Observation pressure surface自身が未証明Data authority / Handoff meaningを所有しないことだけである。

## 4. Data / state authorityとの境界

本Pressureは「physical factがどのData objectを正本に持つか」を決めない。

Second corpus outlineにはまだTEI-original:

```text
Data identity
Database ownership
Application Pattern
Observation Query / Contract Query
```

がないためである。

したがってcurrent testはbooleanのtest-local observed factを使うが、これをproduction state representationとして採用しない。

次の段階でexact TEI-native Data authorityが得られたときだけ:

```text
external-observed state
vs
managed-here write authority
```

のcross-view contradictionへ進む。

## 5. Handoffとの境界

physical factが観測されたことは、そのままProcedure Handoffの存在を意味しない。

```text
physical completion observed
  != accepted responsibility handoff relation
```

H1/R3へ必要なのはexact Procedure/Step endpointsとaccepted conditionである。

Second corpus outlineにはまだそれがないため、Observation pressureからHandoffを自動生成しない。

P-OBS-05は、この非導出境界をtest-only probe shapeでも保持する。Observation probeへHandoff endpoint / condition fieldを追加しない。

## 6. X-TEA fallbackとの整合

X-TEA / Three-Element Analysisへ立ち返ると、Business側の行為・手順とApplication Functionは同一ではなく、Business Modelは社会システムとしての手順を保持する。

したがってphysical-world Business factをApplication commandの成否だけへ還元しないことは、TEI側でBusiness / Function-Interactionを相互批判可能なまま残す方向と整合する。

ただしX-TEA evidenceからTEI固有のObservation Canonical kindやphysical state schemaを導出しない。

## 7. 現時点の判断

```text
Command / Observation separation
  second domainでもmechanical pressureへ進めた

Observation pressure surface
  Data authority / Handoff shapeを持たない

Observation Canonical identity
  still unproven

Data authority contract for manufacturing/inventory
  still unresolved

Responsibility handoff relation
  still unresolved
```

したがってcandidate stackを拡張しない。

Gate 15cも起動しない。

## 8. Verification status

Repository上へtest-only Pressureを追加したが、actual Repository runner greenは未取得である。

`gofmt / go vet / go test / checker` PASSはまだ主張しない。
