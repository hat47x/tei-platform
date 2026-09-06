# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / Pre-ADR audit / Not Ready / no production Core adoption**

## 1. 目的

Business Procedure Reference pressureからproduction Canonical contractを判断する前に、次を分離する。

1. accepted Business meaningとしてnormalized layerに残す候補
2. 既存Canonical meaningからderiveするもの
3. Authoring / Source / Review infrastructureへ残すもの
4. Projection / Diagnosticだけに置くもの
5. Runtime / Authorization / Provider等へ置くもの
6. full Business Flow等、別Reference pressureとして残すもの

本監査はADRではない。

Production `ir.World`変更、`BusinessProcedure` / `BusinessHandoff` kind採用、Business Flow schema、Workflow Engine導入を決定しない。

## 2. 現在までの反証

### 2.1 identity-only Aは最小十分ではない

同じProcedure IDのままpurpose / trigger / actor / Business Actionが変わると、ID-only shellはBusiness-only accepted semantic changeを失う。

したがってidentityは有用でも、それだけをcomplete contractとはしない。

### 2.2 full Workflow modelは不要

current Referenceはretry / timer / queue / scheduler / saga / runtime stateなしでBusiness Procedure / manual work / handoffを表現できる。

```text
Business Procedure != Runtime Workflow Orchestration
```

を維持する。

### 2.3 single Application ownerは不要

一つのProcedureは複数Application、manual work、external workを跨ぎ得る。

fixture-level `application`はC1としてoptional reporting metadataへ下げ、policy authorityはexplicit Patternを所有するApplicationからexact traceする。

### 2.4 uncertaintyはmeaningだがgeneric frameworkは不要

trigger / actorの`candidate / unresolved`等をlosslessに保持する。

ただしcertainty lattice、approval state machine、全subject共通status vocabularyは作らない。

### 2.5 `realization`はminimum Core必須ではない

`realization`をsubtractしてもBusiness Action、manual work、explicit refs、Business-only changeを保持できるtest-only candidateが成立している。

現状はReference authoring / review labelとして残し、production enumにはしない。

### 2.6 Step body meaningとaddressabilityは分離できる

Step IDをbody meaningから除いてもordered Business Actionは保持できる。

一方handoff / reviewにはexact endpointが必要である。

したがって:

```text
Step body meaning
  !=
Step referential address
  !=
Runtime workflow identity
```

を固定する。

### 2.7 authoring-only handoff H0はaccepted relation changeを失う

handoff condition / endpointだけが変わる場合、Procedure bodyは不変である。

HandoffをAuthoringだけへ残すH0では、そのaccepted Business relation changeをnormalized semantic diffから観測できない。

H1 thin normalized handoff meaningの方がcurrent Referenceをlosslessに説明する。

### 2.8 Data Relation / generic Core EdgeはHandoffの器ではない

現行`semantic.Relation`はData target / cardinality / RelationKeyを持つData Relation専用contractである。

Handoffを流用しない。

またHandoff一件のためにgeneric Core `Edge`を作らない。

### 2.9 Business Handoffはfull Business Flowではない

X-TEA fallbackではBusiness FlowがProcess(Task) node、他participant / storage-like node、DataflowLine等からなる別design pressureを持つことを確認した。

したがって:

```text
Business Handoff != Business Flow / DFD
```

を固定する。

Current H1/R3はhandoff-onlyであり、Business Flow completenessを主張しない。

## 3. 現在のcandidate stack

```text
D3
  thin Business Procedure residual meaning
  - realization review label
  + Step body / address separation

H1
  thin normalized handoff meaning

R3
  Business semantic boundary内のhandoff-only relation

S4-A
  normalized Procedure側のProcedure-scoped Step local address metadata

C1
  fixture-level Applicationはoptional reporting context
```

これは現在最も説明力が高い**pre-ADR candidate pressure**であり、production contract採用ではない。

## 4. D3 body meaning候補

```text
BusinessProcedure
  procedure address / identity candidate
  purpose
  trigger
    epistemicStatus
    meaning
  actor
    epistemicStatus
    businessLabel
  orderedSteps[]
    addressMetadata?     // Procedure scoped
    body
      businessAction
      explicitSemanticRefs
```

### 残す理由

- purpose: Application Patternからderiveできない
- trigger / actor status: candidate / unresolvedを失わない
- order: Business Procedureでは順序自体がmeaning
- Business Action: manual / external / judgment等をFunctionへ還元しない
- explicit refs: accepted Application / Semantic identityとのcross-view traceに必要

### まだ固定しないもの

- production type / field名
- Procedure identity生成規則
- Step local address syntax
- uncertainty enum
- direct Data / Relation / Capability refsの最終許容範囲

## 5. H1 / R3 Handoff候補

```text
Handoff
  from
    Procedure address
    Step local address
  to
    Procedure address
    Step local address
  condition
```

保持したい意味:

- condition change
- source/target endpoint change
- stale endpoint
- duplicate relation rejection

入れないもの:

- source path / fingerprint / revision
- runtime instance / token / state
- retry / queue / timer / scheduler / saga
- transport
- Authorization identity
- Dataflow payload
- external party / store / Process node taxonomy

Natural-language conditionはcurrent Referenceのaccepted meaningとして保持し、executable predicateへ早期変換しない。

## 6. S4-A Step address

H1をnormalized meaningとして検査するなら、Semantic validationがSource Indexへ戻らずendpointを解決できる必要がある。

現在の主候補:

```text
normalized Procedure
  ordered Step
    Procedure-scoped local address
    body meaning
```

local addressは:

- Procedure scoped
- file path independent
- Source fingerprintではない
- global Step IDではない
- runtime identityではない

Step local renameは`BodyMeaningChanged`ではなくreferential-address changeとして観測可能である。

この分類名をproduction enumへはまだ固定しない。

## 7. normalized Business model probe

Test-only model:

```text
NormalizedBusinessModel
  Procedures[]
    D3-like Procedure
    Step { localKey, body }
  Handoffs[]
    from Procedure/localKey
    to Procedure/localKey
    condition
```

Normalize後のreferential validationはこのmodelだけで閉じ、Source Index / artifact locator / fingerprint / Workspaceへ戻らない。

確認Pressure:

- artifact locator moveでmodel不変
- stale local Step address fail closed
- coordinated local address + endpoint rename成立
- missing endpoint reject
- condition change = Business semantic relation change
- Source / Runtime container fieldなし

## 8. Business Flow / DFDを別pressureに残す

`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md` E-014および`BUSINESS_PROCEDURE_HANDOFF_VS_BUSINESS_FLOW_PRESSURE_2026-09.md`を基線とする。

Current Handoff modelへ次を入れない。

```text
BusinessFlow collection
Process node
ExternalParty / Role participant
Information / Document / Data flow
Store / data-source participant
Payload / FlowType
```

`business_flow_scope_pressure_test.go`は、current Handoff validation PASSをBusiness Flow / DFD Conformanceへ読み替えないことをnegative Pressureとして固定する。

Full Business Flow production contractは、realistic ReferenceでProcedure handoffだけではaccepted meaningをlosslessに保てないEvidenceが出た場合だけ別途reopenする。

## 9. Derivedであるべきもの

### Pattern-owned meaning

Application Patternからderiveできる:

```text
Data
Relation
Query
Capability
Datasource
Pattern owner Application
accepted datasource policy
```

をBusiness sourceへコピーしない。

### Application association

Procedureにmandatory single Application ownerを置かず、explicit Pattern refsのowner集合からderiveする。

### Projection / Diagnostic

- Business Action × Application Pattern matrix
- Business Action × Data / Relation matrix
- CRUD / impact view
- graph / flow / UI projection
- checker diagnostics / severity

はaccepted Business meaningの第二正本にしない。

## 10. Authoring / Source infrastructureへ残すもの

- file path / artifact locator
- source range
- source fingerprint
- schemaVersion / artifactRole
- Git revision / ETag
- approval snapshot metadata
- ManagedFiles entry

Pure file moveでBusiness meaningを変えない。

## 11. Coreへ入れないもの

### Runtime Workflow

- branch execution state
- retry / timer / queue / scheduler / saga
- lease / workflow instance
- execution ownership / transport

### Authorization

- Authorization Role
- Principal / Permission / Credential
- Authority policy

### Provider / Deployment Evidence

- Database readiness
- Provider binding
- deployment / Gate B status
- Target rendering state

### UI layout

Table / graph / screen layoutはEditor / Target Projection concernである。

## 12. multi-Application / social-system pressure

Temp Workspaceで:

```text
order_entry interaction
fulfillment_console interaction
manual warehouse coordination
external delivery continuation
```

を同一Procedureへ共存させる中間Pressureを実装済み。

Business ProcedureをApplication画面操作列へ縮めないEvidenceになるが、synthetic second Applicationなので「realistic Reference完成」とはみなさない。

## 13. Watanabe / X-TEAとの整合

設計原点へ戻った結果、current candidateは次を維持する。

- Business Model = business manual / social-system viewpoint
- Business Action != software Function
- Data / Function / Businessを相互批判可能な別viewpointとして扱う
- Matrixはderived review surface
- Business FlowとBusiness Procedureを同一objectへ畳まない
- Historical source shapeをそのままTEIへ移植しない

X-TEAの存在はR3を巨大なBusiness graphへ拡張する根拠ではなく、むしろ**異なる設計面の分離とcross-reference**を守る根拠として使う。

## 14. ADR作成前Gate

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
10. `realization` subtractionをactual testsで確認
11. S4-AでSource Index非依存のhandoff integrityをactual testsで確認
12. global Step IDなしで必要addressabilityを説明
13. H1/R3をhandoff-onlyとして保ち、Business Flow surfaceを混ぜない
14. current Handoff EvidenceをBusiness Flow / DFD Conformanceへ過大主張しない
15. current `order_entry`より一段現実的なmulti-App / manual / external Reference pressure
16. Lane B再監査と矛盾しない
17. D3 + H1 + R3 + S4-A + C1が最小十分である理由を説明

## 15. 現在の到達度

```text
Gate 2-14
  Repositoryへtest-only pressureは概ね実装済み
  actual execution未取得

Gate 15
  synthetic multi-Application intermediate pressureまで
  realistic Reference最終Evidenceではない

Gate 1
  unresolved
```

GitHub-hosted Actionsは過去にrunner割当前で終了し、一時workflowはmerge対象から除去済み。Remote Desktopもオンライン端末不在で実行できていない。

したがって**Pre-ADR Status = Not Ready**を維持する。

## 16. 現時点の判断

### productionへ今入れない

- Canonical `BusinessProcedure` schema
- Canonical `BusinessHandoff` schema
- `ir.World.Business`
- full Business Flow / DFD schema
- generic Core / Business Edge
- `realization` enum
- global/durable Canonical Step ID
- generic Uncertainty framework
- Workflow Engine

### primary candidate pressure

```text
D3 thin Business Procedure meaning
+ H1 thin normalized handoff
+ R3 handoff-only Business relation boundary
+ S4-A Procedure-scoped Step local address
+ C1 optional Reference Application context
```

### 次のLane C作業

1. actual execution Evidenceを得る経路が利用可能になれば最優先で実行
2. realistic Referenceを無理に増やさず、既存corpus / scenarioから再利用可能なmulti-system Business Procedure圧力を探す
3. D3 candidateが別scenarioでも崩れないかPressure
4. full Business FlowはT-GAP-06として分離し、Procedure ADRのためだけに実装しない

この条件が閉じるまでPR #482はDraft、Issue 000042はIn Progressを維持する。
