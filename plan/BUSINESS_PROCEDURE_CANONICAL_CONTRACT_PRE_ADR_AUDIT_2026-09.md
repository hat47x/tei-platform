# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / Pre-ADR audit / Not Ready / no production Core adoption**

## 1. 目的

Business Procedure Reference pressureからproduction Canonical contractを判断する前に、accepted Business meaning、derived meaning、Authoring/Source metadata、Projection/Diagnostic、Runtime/Provider、別Reference pressureを分離する。

本監査はADRではない。Production `ir.World`変更、`BusinessProcedure` / `BusinessHandoff` kind採用、Business Flow schema、generic ref-role model、Workflow Engine導入を決定しない。

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
- ref kind separationだけではref role separationを表せない。
- Authorization policyとBusiness protected targetは別Authorityであり、policyからBusiness targetを逆導出してはならない。
- manufacturing / inventory outlineはphysical-world非対称性を再現するが、O1 generic roleやD4/H1/R3 production拡張をまだ要求しない。
- physical-world Command / Observation separationはtest-onlyで機械化できるが、新しいCanonical Observation identityは不要である。
- second-corpus Data authorityは重要でも、exact TEI Data / Pattern / Database identityなしにsynthetic ownershipを作ってはならない。
- physical factの観測だけではaccepted responsibility Handoffを導出できない。
- Procedure Step順序 / adjacencyやApplication Function sequenceからaccepted responsibility Handoffを導出してはならない。
- normalized H1/R3 relationのauthorityはexplicit Business Handoff sourceに限定する。
- Business Action × Data / Relationはexisting Pattern / Validation / Capability refsからderived projectionできる範囲が広く、Business sourceへData / Relationを複製する理由にはならない。
- current refsからData traceがないことは`no Data dependency`を意味せず、unknown / no-traceとして保持する。
- derived Data traceが存在することもdependency listの完全性を意味しない。
- Business Action × CRUDはcurrent structured meaningから部分的にのみ導出でき、現時点で安全に確定できるoperationは主にRである。
- Database Contract `Writes[]`はwrite authorityを示すが、Business Action単位のC/U/D attributionを所有しない。
- `record` / `command` Pattern kind、Capability input/output、Lifecycle transitionからCRUD letterを推測してはならない。

## 3. Explicit ref authority

Current `order_entry`では、既存authorityがあるだけではdirect refを保持しない。Business Action自身がそのauthorityを直接必要とし、既存Pattern refからlosslessにderiveできない場合だけBusiness sourceへ置く。

### Application Pattern — direct realization

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

Second-corpus physical-world Pressureでも同じ境界を利用し、Command acceptanceやread transport successをphysical completionへ丸めない。

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

O1
  Application Pattern protected-target role overlay pressure
  D4 direct realization refとは独立
```

D3 / D4 / O1はいずれもproduction採用ではない。

## 5. D4 body candidate

```text
Procedure body
  id
  purpose
  trigger { epistemicStatus, meaning }
  actor   { epistemicStatus, businessLabel }
  ordered step bodies[]
    businessAction
    applicationPatternRefs[]   // direct realization only
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

Second-corpus physical stateのrelevanceだけではData / Relation slotを追加しない。Exact TEI-original Data authorityがない状態でownershipを作ることもしない。

### 5.1 Business Action × Data / Relation derived projection

`business_action_data_projection_pressure_test.go`では、Business sourceのaccepted refsとnormalized Worldから次を名称推測なしにderiveする。

```text
Application Pattern
  -> Data
  -> Relation source / target Data
  -> Pattern-owned Capability input/output/outcome Data

Validation
  -> Attribute owner Data
  -> Relation source / target Data

Business explicit Capability
  -> input/output/outcome Data
```

Current `order_entry`では、BP-00〜BP-03 / BP-05〜BP-07に有用なData traceを作れる。

一方、BP-04 / BP-08 / BP-09はcurrent accepted refsからData traceを作れないため、名称・隣接Step・Business wordingから補完せず`NoDataTraceFromCurrentRefs=true`として残す。

このflagは`no Data dependency`を意味しない。またfalseも`all Data dependencies resolved`を意味しない。Derived projectionはreview / impact補助であり第二正本ではない。

BP-06は`security.authorize`からsecurity-side Dataをtraceできるが、O1 protected targetはauthoritative Business sourceにpersistしていないため、隣接BP-07から`sales.order`を補わない。この部分traceはO1 persistence判断のEvidenceにはなるが、O1 source採用を自動決定しない。

以上から、current EvidenceはD4へData / Relation slotを戻すより、existing refsからderived projectionを作る方を支持する。ただし将来Business Action自身がdirect Data / Relation authorityを必要とするaccepted sourceが現れた場合はD4 fail-closed ruleでreopenする。

### 5.2 Business Action × CRUD partial projection

`business_action_crud_projection_pressure_test.go`では、Data traceそのものからCRUD letterを付けず、operation authorityまで辿れるcellだけを埋める。

Current `order_entry`で安全に導出する主要cellは:

```text
BP-00 -> R sales.order
  browse Pattern + Query read boundary

BP-02 -> R customer.customer
  reference Pattern + readable datasource + Database Contract Query read

BP-03 -> R sales.order_line
  header-detail realization binding + Database Contract Query read
```

一方:

```text
BP-01 record
BP-04 manual judgment
BP-05 validation
BP-06 authorization
BP-07 command
BP-08 observation intention
BP-09 manual/external continuation
```

はcurrent refsからCRUD operationを確定しない。

特にDatabase Contractには:

```text
sales_core.order
  read sales.order
  write sales.order.status
```

があるが、`sales.order.submit` Capability / BP-07とこのwriteを結ぶaccepted semantic relationはない。

したがって:

```text
Database write authority
  != BP-07 = U
```

である。

`record`はone-record intentでありR/Uを意味しない。`command`はCapability invocationでありC/U/Dを意味しない。Capability input/outputはpayload contractでありpersistence effectではない。

Business Lifecycle proposalの`pending -> submitted` transitionもBusiness state meaningであり、physical mutation / CRUD Uとは分離する。Lifecycle transitionからUを自動生成しない。

Current answer:

```text
Business Action × CRUD
  partially derivable

R
  some exact rows available

C/U/D
  operation attribution contract insufficient
```

この不足はBusiness Procedure schemaより、Capability / realization / Database effect boundaryのpressureである可能性が高い。CRUD matrixを埋める目的だけでD4へCRUD fieldを追加しない。

## 6. ref kindをgeneric bagへ畳まない

```text
Application Pattern
Validation
Capability
```

は異なるauthority kindである。literal identity textが同じでも同一refとして扱わない。

Generic `semanticRefs[]` / kind-less `[]CanonicalID`はcurrent Evidenceから採用しない。

## 7. O1 — ref role separation

Application Patternという同じauthority kindでも、current Referenceでは少なくとも二つのrole pressureがある。

```text
BP-07 -> order_entry.submit_order
  direct Function realization

BP-06 -> order_entry.submit_order
  protected target / concern
  not realization
```

BP-06は同時に:

```text
supporting semantic authority -> security.authorize
protected Application target  -> order_entry.submit_order
direct Function realization   -> none
```

を保持する必要がある。

`applicationPatterns[]`をprotected targetにも流用すると、target relationをFunction realizationへ誤変換する。

O1ではtest-only overlay:

```text
D4TargetAwareStep
  Body: D4 Step body
  ApplicationPatternTargets[]
```

を比較し、target-only changeをD4 bodyから独立したcross-view meaning差として保持する。

ただしgeneric:

```text
ExplicitSemanticRef { Kind, Role, Target }
```

はまだ採用しない。current strong role pressureはBP-06中心であり、`supports / concerns / protects / checks / realizes` taxonomy全体は未証明である。

## 8. D4 + S4-A + H1/R3 composition

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

Second-corpusではphysical factがresponsibility continuationの前提になり得るが、exact Procedure/Step/conditionがない限りH1/R3 relationを生成しない。

### 8.1 H1/R3 authority direction

`second_corpus_handoff_inference_pressure_test.go`で次をPressureする。

```text
Procedure bodies + ordered steps
explicit Handoff source absent
  -> normalized Handoffs = 0

positive physical Observation
explicit Handoff source absent
  -> normalized Handoffs = 0

explicit Handoff source present
  -> normalized Handoffs = 1
  -> exact endpoint + condition validation
```

したがってauthority directionは:

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

とする。

これはproduction `BusinessHandoff` Canonical kindの必要性を証明しない。H1/R3を採用する場合に最低限守るauthority separationを示すEvidenceである。

## 9. D4 + O1 composition

```text
D4 body
  applicationPatterns[] = direct realization only

O1 overlay
  applicationPatternTargets[] = protected target / concern
```

Pressure:

- BP-06のD4 bodyにApplication Pattern realizationを追加しない
- `security.authorize` Capability refを保持
- `order_entry.submit_order`をtarget overlayでexact resolution
- target-only changeはD4 body不変・overlay meaning変更
- BP-07隣接realizationからtargetをderiveしない
- Authorization policy shapeをBusiness Procedureへ逆流させない

Current O1 / D4+O1 testsのtargetはtest helper引数であり、authoritative Business sourceへpersistした意味ではない。O1をproduction accepted meaningとして残す場合だけ、file-native source / decode / reload / Shared Acceptanceまで同じtarget roleを保持するpressureを追加する。O1を残さない場合はfixture fieldを増やさない。

### 9.1 O1 authority direction

`security.authorize`は`subject / action / resource`を扱う汎用Semantic Authorization contractであり、BP-06や`order_entry.submit_order`というBusiness-side protected-target relationを所有しない。

Legacy Authorization Referenceにはcurrent caseとして:

```text
ORDER_EDITOR
execute
order_entry.submit_order
allow
```

が存在するため、O1 targetとのcross-view corroborationには使える。

ただしauthority directionは次に限定する。

```text
explicit Business protected target
  + security-side Authorization policy
  -> corroboration / contradiction diagnostic

Authorization policy
  -X-> author Business protected target
  -X-> backfill missing O1 target
```

Policy changeはBusiness targetそのものを暗黙変更せず、不一致を`BUSINESS_APPLICATION_PATTERN_TARGET_AUTHORIZATION_POLICY_MISMATCH`等のtest-only cross-view diagnosticとして露出させる。

X-TEA fallbackで確認したTaskAction / TaskFunctionIO分離はBusiness ActionとFunction利用の分離を支持するが、generic protected-target role taxonomyまでは証明しない。この限界をE-015として`BUSINESS_PROCEDURE_O1_AUTHORITY_DIRECTION_PRESSURE_2026-09.md`へ記録した。

## 10. Derived / externalへ残すもの

PatternからderiveできるData / Relation / Query / Capability / Datasource / owner Application / datasource policyをBusiness sourceへコピーしない。

Business Action × Data / Relation projectionもaccepted Business meaningの第二正本にしない。Projection rowはData/Relation traceとEvidence pathを表すだけで、Ownership / Access / WritePolicy / Canonical authorityを所有しない。

Business Action × CRUD projectionも第二正本にしない。Current R cellはApplication / Contract Query / Database Contractのaccepted authorityからderiveし、C/U/D未解決をmatrix側で補完しない。

Procedureにmandatory single Application ownerを置かない。

Matrix、CRUD / impact view、graph、checker diagnosticsをaccepted Business meaningの第二正本にしない。

Source path / fingerprint / revision / ETag / ManagedFiles / approval snapshotはAuthoring/Source infrastructureへ残す。

Second-corpus physical Observationのruntime ReadIdentity / Evidence natureもCanonical Business meaningへ昇格させない。

## 11. Coreへ入れないもの

- Runtime Workflow state / retry / timer / queue / scheduler / saga
- Authorization Role / Principal / Permission / Credential / policy effect
- Provider / deployment Evidence
- UI layout
- generic Core / Business Edge
- generic `semanticRefs[]`
- generic explicit-ref Role taxonomy
- Observation専用Canonical identityの先行導入
- physical-world state engine
- second-corpus synthetic Data ownership / Database / Pattern
- Business Action × Data / Relation derived projectionをCanonical authority化すること
- Business Action / ProcedureへのCRUD letter / generic CRUD field
- Database Contract write -> U自動変換
- Lifecycle transition -> U自動変換
- inferred Handoff from adjacency / Observation / Function sequence
- global Canonical Step ID

## 12. Business Flow / DFDは別pressure

Current Handoff modelへProcess node、ExternalParty / Role participant、Information / Document / Data flow、Store、Payload / FlowType等を入れない。

Current handoff validation PASSをBusiness Flow / DFD Conformanceとはみなさない。

Physical fact observedだけでもBusiness Flow / DFD conformanceやresponsibility Handoff成立とはみなさない。

## 13. Gate 15

### 15a multi-Application structural pressure
Synthetic second Applicationで`Business Procedure != single Application child`を確認する。

### 15b social-system realism pressure
Current `order_entry`だけでもmanaged-here sales data、external-observed customer master、human judgment、Authorization、Command、Observation intention、manual/external continuationが同一Procedure文脈に共存する。

### 15c conditional broader realistic pressure
15a + 15bで説明できない具体的不足が出た場合だけ追加する。Application数をrealism KPIにしない。

### 13.1 manufacturing / inventory second-corpus outline

`BUSINESS_PROCEDURE_SECOND_CORPUS_CONTRACT_PRESSURE_AUDIT_2026-09.md`でprovenance-backed outlineをcandidate stackへ当てた。

結果:

```text
D4
  manual / physical / external Business Action pressureを再現
  direct Data / Relation ref necessityは未証明
  current order_entryではexisting refsからderived Data projectionが有用に成立
  -> Data / Relation slotを戻す圧力は弱まった

O1
  protected-target roleのsecond cross-domain exampleは未証明
  generic Role taxonomyを強める根拠なし
  BP-06 Data projectionはsecurity-side部分traceに留まり、O1 persistence判断のpressureになる

H1 / R3
  responsibility continuation pressureは自然に存在
  Step adjacency / positive physical Observationからは生成しない
  explicit Handoff sourceだけをnormalized relation authorityとするtest-only Pressureへ進展
  production Canonical kind necessityは未証明

Observation boundary
  command accepted != physical actuality
  を別domainから強く再現
  -> test-only mechanical pressureへ進展
  -> no observation / observed=falseをblocking diagnosticで区別
  -> Canonical Observation identityは追加しない

Data authority
  physical state authorityのrelevanceは強い
  exact TEI-original Data / Pattern / Database identityは未証明
  -> synthetic ownership禁止

Business Flow / DFD
  material / information movementのrelevanceはある
  Procedure/Handoffではmeaningを失うexact caseは未証明
```

したがってcurrent candidate stackのproduction contract insufficiencyはまだ見つかっておらず、**Gate 15cは未起動**を維持する。

このoutline-level Evidence、Observation pressure、Handoff inference pressure、Data projection pressure、CRUD projection pressureは「next realistic Reference fixture完成」ではない。actual green前に外部modelのexact sequenceやsynthetic identitiesをfixtureへ昇格しない。

## 14. ADR作成前Gate

### Readiness authority

本節を**production Canonical contract ADR readinessの唯一の判定正本**とする。

- `plan/issue/000042-business-procedure-cross-view-reference.md` のA1〜A90 matrixはEvidence / Pressure inventoryであり、独立したReady判定ではない。
- 同Issue末尾の完了条件はLane Cのoperational checklistであり、本節とは別のreadiness stateを所有しない。
- 両者と本節の表現に差が出た場合、意味を失わないよう同期するが、Ready / Not Ready判定は本節を正とする。

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
14. Gate 15a / 15bをactual testsで確認し15cは具体的contract不足時だけ起動
15. Authorization / Observation非対称境界をactual testsで確認
16. BP-03 / BP-05 authority separationをactual testsで確認
17. D3 / D4のどちらがcurrent + next TEI-original realistic Referenceで最小十分か説明。outline-level corpus breadthはfixture Evidenceと同一視しない
18. D4 future Data / Relation ref fail-closedをactual testsで確認
19. authority kindをgeneric ref bagへ失わない
20. D4 + S4-A + H1/R3 compositionをactual testsで確認
21. O1 target != realizationをactual testsで確認
22. D4 + O1 compositionをactual testsで確認
23. generic ref-role taxonomyが必要か複数realistic pressureから判断。manufacturing / inventory outlineだけではO1反復を証明したとみなさない
24. Legacy Authorization policyをO1 targetのauthor / backfill sourceにせず、corroboration-onlyとしてactual testsで確認
25. O1をproduction accepted meaningとして残す場合、authoritative source / decode / reload / Shared Acceptance境界を閉じる。残さない場合はproduction source fieldを追加しない
26. Lane B再監査と矛盾しない
27. physical-world Command acceptanceだけでcompletionを成立させず、explicit Observationなし / observed=false / positive external Observationをactual testsで区別する
28. second-corpus Data authorityはexact TEI-original Data / Pattern / Database authorityが揃うまでsynthetic化しない
29. physical ObservationからH1/R3 responsibility Handoffを自動生成しない
30. Procedure Step順序 / adjacency / Application Function sequenceからH1/R3を自動生成せず、explicit Handoff sourceだけがnormalized relationをauthorすることをactual testsで確認する
31. Business Action × Data / Relation projectionをactual testsで確認し、Pattern / Validation / Capabilityからexact identity traceを導出できることを示す
32. Data traceがないStepを`no Data dependency`へ丸めず、no-trace / unknownとして保持する
33. derived Data traceが存在しても完全性を主張せず、projectionをOwnership / Access / Canonical authorityの第二正本にしない
34. BP-06 Data projectionがO1未persist時にはsecurity-side部分traceに留まることを確認し、O1 persistence判断とData projectionを相互に誤補完しない
35. Business Action × CRUD partial projectionをactual testsで確認し、BP-00 / BP-02 / BP-03のRをexact authorityからderiveする
36. Data traceとCRUD operation traceを分離し、Relation source/target Dataが見えるだけでR/U等を付与しない
37. `record` / `command` Pattern kindだけからR/U/C/Dを推測しない
38. Database Contract `write sales.order.status`をaccepted action linkなしにBP-07=Uへ帰属させない
39. Capability input/outputやCustom Go realizationをpersistence mutation contractへ読み替えない
40. Business Lifecycle transitionをCRUD Uへ短絡せず、Business state meaningとpersistence mutationを分離する
41. C/U/D不足が複数realistic domainで反復するまでBusiness Procedure / Pattern / Capabilityへgeneric CRUD/effect fieldを追加しない

GitHub-hosted Actionsは過去にrunner割当前で終了し、一時workflowはmerge対象から除去済み。Remote Desktopもオンライン端末不在で実行できていない。

したがって**Pre-ADR Status = Not Ready**を維持する。

## 15. 現時点の判断

### productionへ今入れない

- Canonical `BusinessProcedure` schema
- Canonical `BusinessHandoff` schema
- `ir.World.Business`
- full Business Flow / DFD schema
- generic Core / Business Edge
- generic `semanticRefs[]`
- generic explicit-ref Role enum
- `realization` enum
- Observation Canonical kind / physical-world state engine
- synthetic second-corpus Data ownership model
- Business Action × Data / Relation projectionのCanonical authority化
- Business Procedure / Pattern / Capability generic CRUD/effect field
- Database Contract write -> CRUD U自動変換
- Lifecycle transition -> CRUD U自動変換
- inferred Handoff from adjacency / Observation / Function sequence
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
O1 target-role overlay pressure (authoritative source intentionally deferred)
```

O1 authority directionはtest-onlyで閉じたが、O1をaccepted Business sourceへpersistすべきかは未決である。Manufacturing / inventory outlineはCommand / Observation境界とH1/R3 authority方向をtest-only pressureまで進めた。一方、current `order_entry`のBusiness Action × Data / Relation derived projectionは、D4へData / Relation slotを戻さずともかなりのreview traceを得られることを示した。Business Action × CRUD projectionはさらに、Rの一部だけをexactに導出でき、C/U/D attribution contractがcurrent meaningに不足していることを示した。この不足はBusiness Procedure schemaではなくCapability / realization / Database effect boundary側のpressureである可能性が高い。No-traceはunknown、traceありもcompleteではない。O1 role反復、exact second-corpus Data authority、production `BusinessHandoff` kind necessity、D4/H1/R3 production contract不足はなお未証明である。Actual greenと次のTEI-original realistic Referenceから最小十分性を判断する。