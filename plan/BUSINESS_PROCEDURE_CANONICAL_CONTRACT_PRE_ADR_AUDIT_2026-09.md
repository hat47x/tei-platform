# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / Pre-ADR audit / Not Ready / no production Core adoption**

## 1. 目的

Business Procedure Reference pressureからproduction Canonical contractを判断する前に、accepted Business meaning、derived meaning、Authoring/Source metadata、Projection/Diagnostic、Runtime/Provider、別Reference pressureを分離する。

本監査はADRではない。Production `ir.World`変更、`BusinessProcedure` / `BusinessHandoff` kind採用、Business Flow schema、Workflow Engine導入を決定しない。

## 2. 主要な反証

- identity-onlyではBusiness-only accepted changeを失う。
- full Workflow modelは不要。
- Business Procedureをsingle Application childにしない。
- trigger / actor uncertaintyはmeaningだがgeneric certainty frameworkは不要。
- `realization`はauthoring/review labelとして有用でもminimum Core必須とは証明されていない。
- Step body meaningとStep referential addressは分離できる。
- authoring-only Handoff H0はcondition / endpoint changeをnormalized meaningから失う。
- Data Relation / generic Core EdgeはBusiness Handoffの器ではない。
- Business Handoffはfull Business Flow / DFDではない。

## 3. Explicit ref authority

Current `order_entry`では、既存authorityがあるだけではdirect refを保持しない。Business Action自身がそのauthorityを直接必要とし、既存Pattern refからlosslessにderiveできない場合だけBusiness sourceへ置く。

### Application Pattern

```text
BP-00 -> order_entry.recent_orders
BP-01 -> order_entry.order_record
BP-02 -> order_entry.customer_reference
BP-03 -> order_entry.order_lines
BP-07 -> order_entry.submit_order
```

### Validation

```text
BP-05
  -> sales.order_line.quantity_positive
  -> sales.order.lines.line_number_unique
```

BP-03 line reviewへ同じValidation authorityを重複しない。

### Capability

```text
BP-06 -> security.authorize
```

BP-07の`sales.order.submit`はPattern-owned meaningなのでBusiness sourceへコピーしない。

### Observation

BP-08はBusiness intentionを保持するが、runtime `ExecuteObservationRead` / `ReadIdentity`をSemantic identityへ昇格しない。

原則:

```text
explicit refs are evidence-driven, not symmetry-driven
```

## 4. Candidate stack

```text
D3
  broader thin Business Procedure Step body candidate
  - realization review label
  + explicit typed refs including Data/Relation slots

D4
  smaller evidence-backed Step body candidate
  - current未証明のdirect Data ref slot
  - current未証明のdirect Relation ref slot
  + fail-closed reopen rule

S4-A
  Procedure-scoped Step local address metadata
  D4 bodyとは独立

H1
  thin normalized handoff meaning

R3
  Business semantic boundary内のhandoff-only relation

C1
  fixture Applicationはoptional reporting context
```

D3 / D4はいずれもproduction採用ではない。

## 5. D4 body candidate

```text
Procedure body
  id
  purpose
  trigger { epistemicStatus, meaning }
  actor   { epistemicStatus, businessLabel }
  ordered step bodies[]
    businessAction
    applicationPatternRefs[]
    validationRefs[]
    capabilityRefs[]
```

D4 bodyはStep local addressを持たない。

```text
D4 body meaning
  != S4-A Step address metadata
```

Current Referenceでminimum必要性が未証明のdirect Data / Relation refsは入れない。

将来sourceにdirect Data / Relation refが現れた場合、D4 projectionは黙ってdropせずfail closedし、explicit-ref designをreopenする。

## 6. ref kindをgeneric bagへ畳まない

```text
Application Pattern
Validation
Capability
```

は異なるauthority kindである。literal identity textが同じでも同一refとして扱わない。

RQ-026のcross-view correctionでは、どのviewpoint / authorityへ戻るべきかを失わないことが重要である。

Generic `semanticRefs[]` / kind-less `[]CanonicalID`はcurrent Evidenceから採用しない。

## 7. D4 + S4-A + H1/R3 composition

```text
D4 Step body
        +
S4-A Procedure-scoped local address
        +
H1/R3 handoff-only relation
        ↓
Normalized Business model pressure
```

合成後も:

- D4 bodyへLocalKeyを混ぜない
- stale Step addressをfail closed
- address + endpoint協調renameを許容
- Handoff condition changeをbodyから独立したBusiness meaning差として保持
- new direct Data / Relation refはaddress/handoff layerで隠さずD4でfail closed

`d4_normalized_composition_pressure_test.go`でtest-only Pressureを置く。

## 8. Derivedであるべきもの

PatternからderiveできるData / Relation / Query / Capability / Datasource / owner Application / datasource policyをBusiness sourceへコピーしない。

Procedureにmandatory single Application ownerを置かない。

Matrix、CRUD / impact view、graph、checker diagnosticsをaccepted Business meaningの第二正本にしない。

## 9. Authoring / Source infrastructureへ残すもの

- file path / artifact locator
- source range / fingerprint
- schemaVersion / artifactRole
- Git revision / ETag
- approval snapshot metadata
- ManagedFiles entry

Pure file moveでBusiness meaningを変えない。

## 10. Coreへ入れないもの

- Runtime Workflow state / retry / timer / queue / scheduler / saga
- Authorization Role / Principal / Permission / Credential
- Provider / deployment Evidence
- UI layout
- generic Core / Business Edge
- generic `semanticRefs[]`
- Observation専用Canonical identityの先行導入
- global Canonical Step ID

## 11. Business Flow / DFDは別pressure

Current Handoff modelへProcess node、ExternalParty / Role participant、Information / Document / Data flow、Store、Payload / FlowType等を入れない。

Current handoff validation PASSをBusiness Flow / DFD Conformanceとはみなさない。

## 12. Gate 15

### 15a multi-Application structural pressure

Synthetic second Applicationで`Business Procedure != single Application child`を確認する。

### 15b social-system realism pressure

Current `order_entry`だけでもmanaged-here sales data、external-observed customer master、human judgment、Authorization、Command、Observation intention、manual/external continuationが同一Procedure文脈に共存する。

### 15c conditional broader realistic pressure

15a + 15bで説明できない具体的不足が出た場合だけ追加する。Application数をrealism KPIにしない。

## 13. ADR作成前Gate

次を満たすまでproduction Canonical contract ADRを作らない。

1. actual Repository environmentで`gofmt / go vet / go test / checker`がgreen
2. Business-only wording / uncertainty changeをsemantic diffとして観測
3. pure file move = meaning no-op
4. logical Procedure / Step address renameをfail closed / coordinated changeとして説明
5. manual / external Business Actionをfake Functionなしで保持
6. Pattern-owned meaning非重複
7. one Procedure -> multiple Applicationsをsingle ownerなしで保持
8. actor uncertaintyをAuthorizationへ混ぜない
9. H0 semantic loss / H1 preservationをactual testsで確認
10. `realization` subtraction / diagnostic independenceをactual testsで確認
11. S4-AでSource Index非依存のhandoff integrityをactual testsで確認
12. global Step IDなしで必要addressabilityを説明
13. H1/R3をhandoff-onlyとして保ちBusiness Flow surfaceを混ぜない
14. Gate 15a / 15bをactual testsで確認し15cは不足時だけ追加
15. Authorization / Observation非対称境界をactual testsで確認
16. BP-03 / BP-05 authority separationをactual testsで確認
17. D3 / D4のどちらがcurrent + next realistic Referenceで最小十分か説明
18. D4 future Data / Relation ref fail-closedをactual testsで確認
19. authority kindをgeneric ref bagへ失わない
20. D4 + S4-A + H1/R3 compositionをactual testsで確認
21. Lane B再監査と矛盾しない

GitHub-hosted Actionsは過去にrunner割当前で終了し、一時workflowはmerge対象から除去済み。Remote Desktopもオンライン端末不在で実行できていない。

したがって**Pre-ADR Status = Not Ready**を維持する。

## 14. 現時点の判断

### productionへ今入れない

- Canonical `BusinessProcedure` schema
- Canonical `BusinessHandoff` schema
- `ir.World.Business`
- full Business Flow / DFD schema
- generic Core / Business Edge
- generic `semanticRefs[]`
- `realization` enum
- global/durable Canonical Step ID
- generic Uncertainty framework
- Workflow Engine

### primary comparison pressure

```text
D3 broader Step body candidate
D4 smaller evidence-backed Step body candidate
S4-A Procedure-scoped Step local address
H1 thin normalized handoff
R3 handoff-only Business relation boundary
C1 optional Reference Application context
```

Actual greenと次のrealistic Referenceから、D3/D4の最小十分性を判断する。
