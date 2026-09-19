# Business Action × CRUD — 2026-09

Status: **Lane C / test-only derived projection + X-TEA fallback translation / no production schema adoption**

## この文書について

本書は、Business Action × CRUD operation attribution（R/C/U/Dをどこまで機械導出・帰属できるか）を時系列で調査した次の4文書を1本へ統合したものである。

1. `BUSINESS_ACTION_CRUD_PROJECTION_PRESSURE_2026-09.md`（元・current TEI structured meaningからのderived projection pressure、Rは部分導出可・C/U/D未解決という最初の結論）
2. `BUSINESS_ACTION_CRUD_OPERATION_GAP_AUDIT_2026-09.md`（元・C/U/D不足をどの層のaccepted meaning不足かへ分解する監査、M0〜M4比較）
3. `BUSINESS_ACTION_CRUD_XTEA_FUNCTION_TABLE_IO_TRANSLATION_2026-09.md`（元・X-TEA/XeadModelerのFunction/Table I/O構造との比較、X0〜X4比較、predecessor design比較）
4. `BUSINESS_ACTION_CRUD_DIRECT_EFFECTIVE_BOUNDARY_PRESSURE_2026-09.md`（元・direct/effective CUD区別を将来guardrailとして保持）

後続文書は前段の結論を明示的に引用・分解・拡張する一続きの調査であるため統合した。個別ファイルの履歴はGit historyから追跡できる。

外部から旧ファイル名を参照していた文書（`plan/issue/000042-business-procedure-cross-view-reference.md`、`plan/BUSINESS_ACTION_CUD_TEI_ORIGINAL_REFERENCE_SEARCH_2026-09.md`、`plan/DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md`、`plan/XTEA_DESIGN_EVIDENCE_E016_CRUD_FUNCTION_TABLE_IO_2026-09.md`）は本統合と同じPRで本ファイル名へ参照を更新した。

なお、本書の結論は`BUSINESS_ACTION_CUD_TEI_ORIGINAL_REFERENCE_SEARCH_2026-09.md`と`BUSINESS_PROCEDURE_BP01_PREPARE_OPERATION_AMBIGUITY_PRESSURE_2026-09.md`とも組み合わせて扱う（この2文書は本統合の対象外、別文書として存続）。

---

# Part 1 — CRUD Projection Pressure（最初の結論: Rは部分導出可、C/U/D未解決）

## 1. 目的

X-TEA / 三要素分析法では、FunctionとDataの関係をCRUD matrix等として構造化設計から機械的に導出し、Data / Function / Businessの相互検証へ利用する。

TEIでも`Business Action × Data/Relation` traceまではcurrent accepted meaningからかなり導出できるようになった。

次の問いは:

> current TEI structured meaningだけから、Business Action × C/R/U/D operation matrixをどこまで安全に導出できるか。

である。

本PressureはCRUD matrixを第二SOTにしない。C/R/U/Dを名称・SQL本文・Pattern名・Step順から推測せず、operation authorityがあるcellだけを埋める。

## 2. 最初の結論

Current `order_entry` Referenceから安全に確定できるCRUD operationは、現時点では主に**R**である。

```text
BP-00 recent orders browse
  -> R sales.order

BP-02 customer reference
  -> R customer.customer

BP-03 order lines header-detail
  -> R sales.order_line
```

一方:

```text
BP-01 record
BP-04 manual judgment
BP-05 validation
BP-06 authorization
BP-07 submit command
BP-08 observation intention
BP-09 manual/external continuation
```

について、current accepted refsだけからC/U/Dを確定しない。

これは「CRUDがない」ことを意味しない。

```text
NoCRUDTraceFromCurrentRefs = true
  != no persistence operation
```

である。

## 3. Rを導出する三つの経路

### 3.1 browse

`browse` PatternはData + Queryを明示し、Driverは`ExecuteBrowseQuery`というread/query boundaryで実行する。

Current:

```text
BP-00
  -> order_entry.recent_orders browse
     -> data sales.order
     -> query order_entry.recent_orders
     -> readable datasource sales
```

から:

```text
R sales.order
```

をderived projectionへ置く。

Native SQL本文を解析してSELECTかどうかを判定しない。Application Pattern + Query execution contractをauthorityとする。

### 3.2 reference

`reference` Patternは設計基線上明示的なread intentionであり、write-only datasourceはValidationで拒否される。

Current:

```text
BP-02
  -> order_entry.customer_reference
     -> relation sales.order.customer
     -> datasource master
     -> contract master_data.customer
     -> query find_by_id
```

Database Contractは:

```text
read customer.customer
query find_by_id -> customer.customer zero-or-one
```

を持つ。

したがってoperation projectionはRelation source `sales.order`ではなく、実際にContract Queryが読むtarget:

```text
R customer.customer
```

だけを置く。

### 3.3 header-detail

`header-detail` Pattern自体はRelation intentであり、CRUD operationを持たない。

Current `order_entry.order_lines`は別のrealization bindingで:

```text
pattern order_entry.order_lines
  -> datasource sales
  -> database sales_core
  -> contract sales_core.order_line
  -> query find_by_order
```

へ接続される。

Contractは:

```text
read sales.order_line
query find_by_order -> sales.order_line many
```

を持つため:

```text
R sales.order_line
```

を導出できる。

Relation source `sales.order`はquery inputのsource instanceとして意味上関係するが、このoperationで新たにreadされたDataとは限らないため、CRUD cellへRを置かない。

## 4. Data traceとCRUD operation traceを分ける

T-GAP-03 Data projectionでは:

```text
BP-02 Data trace
  sales.order
  customer.customer
  sales.order.customer

BP-03 Data trace
  sales.order
  sales.order_line
  sales.order.lines
```

が得られる。

しかしCRUD projectionでは:

```text
BP-02
  R customer.customer

BP-03
  R sales.order_line
```

だけになる。

したがって:

```text
semantic dependency
  != persistence operation
```

である。

Data trace matrixからCRUD letterを一律に付与してはならない。

## 5. `record`からR/Uを推測しない

`record` Patternは:

> ApplicationがどのSemantic Dataを一件として扱うか

という意図だけを持ち、Query / Datasource / Capabilityを持たない。

Current BP-01:

```text
order_entry.order_record record
  data sales.order
```

から:

```text
R sales.order
U sales.order
```

のどちらも導出しない。

UIの「record/form」イメージからCRUDを補完しない。

## 6. `command`からC/U/Dを推測しない

Current BP-07:

```text
order_entry.submit_order command
  -> capability sales.order.submit
```

Capabilityはinput/outputとして`sales.order`を持つが、Capability contract自体はCreate / Update / Deleteを宣言していない。

Database側には別authorityとして:

```text
contract sales_core.order
  read sales.order
  write sales.order.status
```

が存在する。

しかしcurrent modelには:

```text
Business Action / Capability
  -> this exact Database Contract write
```

というaccepted semantic linkがない。

したがって:

```text
sales_core.order writes sales.order.status
```

という事実を、隣接性・名称・submitという語感から:

```text
BP-07 = U sales.order
```

へ変換しない。

これはcurrent CRUD projectionで最も重要なnegative boundaryである。

## 7. Database Contract `write` != CRUD C/U/D

Database Contractのcurrent operation vocabularyは:

```text
Reads[]
Writes[]
```

である。

`Writes[]`はwrite authorityを示すが、Create / Update / Deleteを区別しない。

特に:

```text
write sales.order.status
```

はAttributeへのwrite authorityを示すが、それだけでBusiness Action単位のUをauthorする契約ではない。

したがってcurrent TEIで機械的に得られるoperation viewは、厳密には:

```text
read evidence
write authority evidence
```

とCRUD C/R/U/Dの間に距離がある。

CRUD projectionはこの距離を隠さない。

## 8. Validation / Authorization / ObservationからCRUDを作らない

### Validation

BP-05はData/Relation dependencyを持つが、Validationはin-memory / semantic constraint evaluationであり、Database read/writeを意味しない。

### Authorization

BP-06の`security.authorize` Capability payloadからsecurity Data traceは得られるが、Persistence CRUD operationではない。

### Observation

BP-08はphysical / external stateを明示readするBusiness intentionだが、current accepted Business sourceにCanonical Observation read targetを持たない。

Runtime `ReadIdentity`をSemantic/CRUD authorityへ昇格させない既存境界を維持する。

## 9. Projection shape

Test-only:

```text
BusinessActionCRUDProjectionRow
  StepID
  Cells[]
    Data
    Operation   // C | R | U | D
    EvidencePaths[]
  NoCRUDTraceFromCurrentRefs
```

ただしcurrent fixtureで生成されるOperationはRのみである。

`C/U/D` enumが存在することを、current ReferenceでC/U/Dが証明済みという意味にはしない。

禁止field:

- Complete
- Authoritative
- Ownership
- WriteAuthority
- SourceOfTruth
- Workflow / BusinessFlow

## 10. Fail-closed

Rを成立させるread authorityが消えた場合、以前のprojectionを黙って維持しない。

例:

```text
master_data.customer
  read customer.customer
```

が失われれば、BP-02をRとして残さずprojectionをfail closedする。

名称やRelation targetだけからread operationを補完しない。

## 11. X-TEA CRUD matrixへの現在回答

X-TEAでCRUD matrixを機械導出できる背景には、Function定義がTable I/Oを構造化して持つという設計がある。

Current TEIでは:

```text
Application Pattern
Database Contract Query
Database Contract read/write
Capability
```

が分離しており、この分離を保ったまま一部のRは安全に導出できる。

しかしC/U/Dについてはcurrent ReferenceだけではBusiness Actionへの帰属を十分に説明できない。

したがって現在回答は:

```text
Business Action × CRUD matrix
  partially derivable

R
  useful exact rows exist

C/U/D
  current accepted meaning is insufficient for exact attribution
```

である。

## 12. Production contractへの含意

現時点で追加しない:

- Business Procedure sourceへのCRUD letter
- Patternへのgeneric CRUD field
- CapabilityへのC/U/D field
- Database Contract `write`のUへの自動変換
- generic Effect / Mutation edge
- SQL本文解析によるCRUD推定
- CRUD matrixのCanonical化

次に必要なのは「CRUDを埋めたいからoperation fieldを足す」ことではない。

Realistic ReferenceでC/U/Dを正確にcross-checkするために、どのaccepted meaningが本当に不足しているかを特定することである。

候補例:

```text
Capability / Business Action effect semantics
Database Contract writeとのexplicit realization link
Lifecycle Transitionとpersistence mutationのrelation
```

はいずれも未採用であり、Evidenceなしに先行導入しない。

## 13. Verification status（Part 1）

Repository上へ:

```text
tools/reference-business-procedure-check/business_action_crud_projection_pressure_test.go
```

を追加した。

actual Repository environmentでの:

```text
gofmt
go vet
go test
checker
```

は未実行のためPASSを主張しない。

## Part 1 現時点の判断

```text
Business Action × Data/Relation trace
  -> useful derived projection

Business Action × CRUD
  -> partial derived projection
  -> Rはcurrent exact pathsから一部成立
  -> C/U/Dはcurrent Referenceでは未解決

No CRUD trace
  != no persistence operation

Database write authority
  != Business Action CRUD attribution
```

この結果はD4へData/Relation/CRUD fieldを増やす根拠ではなく、むしろderived projectionとaccepted meaningを分離する方針を補強する。

---

# Part 2 — Operation Gap Audit（不足をどの層へ帰属させるかの分解、M0〜M4比較）

## 1. 目的

Part 1で、current TEI structured meaningからBusiness Action × CRUDをどこまで機械導出できるかをPressureした。

結果、current `order_entry`では一部のRをexactに導出できる一方、C/U/DはBusiness Actionへ安全に帰属できない。

本監査は、この不足を「CRUD fieldがない」と単純化せず、どの層のaccepted meaningが不足しているのかを分解する。

## 2. Current operation authority inventory

### 2.1 Application Pattern

Current Pattern kindは:

```text
browse
record
reference
command
header-detail
```

である。

Operationとして読める強さは同じではない。

```text
browse
  Query executionによるread intent

reference
  明示的read intent
  write-only datasourceはreject

header-detail
  Pattern単体ではoperationなし
  explicit Contract Query realizationがある場合にreadをderive可能

record
  one-record intentのみ
  read/updateを意味しない

command
  Capability invocation intentのみ
  create/update/deleteを意味しない
```

Pattern kindからgeneric CRUDを生成しない。

### 2.2 Database Contract

Current Database Contractは:

```text
Reads[]
Writes[]
Queries[]
```

を持つ。

Current Reference:

```text
sales_core.order
  read sales.order
  write sales.order.status

sales_core.order_line
  read sales.order_line

master_data.customer
  read customer.customer
```

`Reads[]`はread authorityとして使える。

`Writes[]`はwrite authorityを示すが:

```text
C
U
D
```

を区別しない。

さらに、Database Contract writeとBusiness Action / Capabilityを結ぶaccepted semantic relationはcurrent modelにない。

### 2.3 Semantic Capability

Capabilityは:

```text
input
output / named outcome
```

を持つが、persistence effectを宣言しない。

Current:

```text
sales.order.submit
  input  sales.order
  output sales.order
```

から:

```text
U sales.order
```

を導出しない。

Input/output Dataはpayload contractであり、persistence mutation contractではない。

### 2.4 Capability realization

Current `sales.order.submit` realizationは:

```text
realize sales.order.submit with custom go
```

である。

Custom Go realization identityは実装選択を表すだけで、Database Contract write / mutation targetを宣言しない。

したがってrealization kindからCRUDを推測しない。

## 3. Business Lifecycle proposalとの関係

Business Lifecycle v1設計（評価待ち）は:

```text
sales.order.status
  pending --[sales.order.submit]--> submitted
```

というBusiness transitionを検討している。

これはC/U/D不足への重要なinputだが、Lifecycle transitionそのものをCRUD Uへ変換してはならない。

理由:

1. LifecycleはBusiness state meaningを所有する。
2. physical state code / Persistence Mappingは別realizationである。
3. Command Business ResultとObservation readは分離される。
4. provider上の条件付き更新は別Evidenceとして扱われる。
5. Lifecycle設計自身が`on-entry` / `on-exit` side-effect hookを導入しない。

したがって:

```text
Business transition
  != persistence mutation
```

である。

将来Lifecycleが採用されても、それだけでCRUD Uを自動生成しない。

## 4. Current missing link

BP-07を例にすると、current accepted meaningは:

```text
Business Action BP-07
  -> Application Pattern order_entry.submit_order
     -> Capability sales.order.submit
        -> Custom Go realization
```

Database側には別に:

```text
sales_core.order
  write sales.order.status
```

がある。

しかし次のedgeがない。

```text
sales.order.submit
  -> mutates sales.order.status
```

または:

```text
order_entry.submit_order
  -> realizes Database Contract write sales_core.order/status
```

このedgeがない状態で、名称・Lifecycle候補・隣接Data・Persistence MappingからBP-07=Uを作ると、複数SOT / 推測補完になる。

## 5. Option audit

### M0 — partial CRUD projectionを維持

```text
RだけEvidence-backedに埋める
C/U/Dはunknownのまま
```

Pros:

- current authorityを壊さない
- P11 unresolved remains unresolvedに一致
- 新しいCore contractを先回りしない

Cons:

- X-TEA型CRUD matrixは部分的
- write impact reviewが弱い

Current: **採用中のPressure方針**。

### M1 — Capabilityへgeneric CRUD/effect field

例:

```text
capability sales.order.submit {
  effect update sales.order.status
}
```

Pros:

- Business Function / CapabilityからData mutationを明示できる

Risks:

- persistence-neutral CapabilityへDB風operation vocabularyを混ぜる
- external API / message / physical actionをCRUDへ押し込める
- Lifecycle / Business Result / provider effectとのownershipが曖昧になる

Current: **defer**。

### M2 — Capability / PatternとDatabase Contract writeのexplicit realization link

例:

```text
Capability
  -> Database Contract write target
```

Pros:

- Database write authorityとのcross-view linkを作れる
- operation projectionを構造化できる

Risks:

- Capabilityの実現方式をDatabaseへ過度に固定する
- Custom Go / Plugin / API realizationがDatabaseを使わない場合に不自然
- one Capabilityが複数resourceへeffectを持つ場合の意味が複雑

Current: **candidate pressure only / not adopted**。

### M3 — Lifecycle TransitionとPersistence Mutation Bindingを分離

```text
Business transition
  -> persistence realization binding
```

Pros:

- Business state meaningとphysical mutationを分けられる
- state transitionとconditional updateの対応を明示できる

Risks:

- Lifecycle未採用段階で先行設計になる
- CRUDすべてがLifecycleではない
- Create/Delete/non-state mutationを説明できない

Current: **future input / not adopted**。

### M4 — Business Procedure / Function側へCRUD letterをauthor

例:

```text
BP-07 U sales.order
```

Pros:

- matrixを直接埋められる

Risks:

- Database / Capability meaningの第二SOT
- implementation変更でdrift
- X-TEAの「structured definitionからmatrixをderiveする」方向に逆行

Current: **reject**。

## 6. X-TEAとの比較

X-TEAではFunction definitionがUI / Table I/O等を構造化して持ち、CRUD matrixをそこから機械導出する。

TEIでは意味を:

```text
Business Action
Application Pattern
Semantic Capability
Database Contract
Realization
```

へ分離している。

この分離自体は維持すべきであり、X-TEAのCRUD列を再現するためだけに一つのFunction objectへ戻さない。

必要なのは:

> TEIの分離されたauthority間に、operation attributionに十分なaccepted linkが本当に必要か

をrealistic Referenceで確認すること。

## 7. 次のEvidence pressure

C/U/D contractを新設する前に、次の順で確認する。

1. current `order_entry`でpartial CRUD R projectionをactual testでgreenにする。
2. `sales.order.submit`についてactual execution / Lifecycle / Database writeのどこまでがaccepted meaningとして既に接続されているか確認する。
3. next TEI-original realistic ReferenceでCreate / Delete / non-state Updateの少なくとも一例を探す。
4. 同じgapが複数domainで反復するか確認する。
5. 反復した場合にM1/M2/M3を比較する。

一つのsubmit例だけでgeneric mutation modelを決めない。

## 8. Pre-ADR判断

Current answer:

```text
Business Action × CRUD
  R: partial exact projection possible
  C/U/D: operation attribution contract insufficient
```

この不足はproduction Business Procedure schemaの不足とは限らない。

むしろ:

```text
Capability / realization / Database effect boundary
```

側のpressureである可能性が高い。

したがってLane CからBusiness ProcedureへCRUD fieldを追加しない。

## 9. Productionへ今入れない

- Business Procedure CRUD annotations
- Pattern generic CRUD field
- Capability generic Effect / CRUD field
- Database Contract write -> U自動変換
- Lifecycle transition -> U自動変換
- SQL AST / SQL本文解析によるCRUD推測
- generic mutation edge
- CRUD matrix Canonical authority

## Part 2 Verification status

`business_action_crud_projection_pressure_test.go`をRepositoryへ追加したが、actual Repository runnerでの:

```text
gofmt
go vet
go test
checker
```

は未取得である。

したがって本監査もpre-contract Evidenceであり、production contract採用根拠ではない。

---

# Part 3 — X-TEA Function/Table I/O Translation（歴史的CRUD sourceとの比較、X0〜X4比較）

## 1. 目的

Current TEIではBusiness Action × CRUDのうちRの一部はexact structured authorityからderiveできるが、C/U/DはBusiness Actionへ安全に帰属できない。

本書は、この不足をX-TEA / XeadModelerのFunction/Table I/O構造へ立ち返って確認し、TEIで将来operation attribution contractが必要になった場合にどの層を第一候補とすべきかを絞る。

Historical X-TEA shapeをそのままTEIへ移植しない。

## 2. X-TEA source evidence

XeadModeler sourceでは`TaskFunctionIO`がFunctionで利用するI/O Tableを表し、少なくとも次の属性を保持する。

```text
TableID
OpC
OpR
OpU
OpD
```

`DialogImportXEAD.java`では`TaskFunctionIO` import時に`TableID`と`OpC / OpR / OpU / OpD`を明示的にcopyしている。

`DialogMatrixList.java`では`TaskFunctionIO`を走査し、Function ID + TableIDをkeyとして`OpC / OpR / OpU / OpD`の`+`をCRUD markへ集約する。

したがってCRUD matrixは:

```text
free-form Business Procedure text
```

から推測するのではなく、

```text
Function <-> Table I/O structured relation
  + explicit operation marks
```

から機械導出される。

これは渡辺幸三氏の「matrixをstructured design informationから機械的に得る」という設計姿勢とも一致する。

## 3. TEI current equivalent / gap

TEIはX-TEAのFunction / Table I/Oを一つのobjectへ集めず、意味を複数authorityへ分離している。

```text
Business Action
Application Pattern
Semantic Data / Relation
Semantic Capability
Query
Database Contract
Realization / Provider Binding
```

### R

Current Rの一部は分離されたauthorityを辿って再構成できる。

```text
browse Pattern
  -> Query
  -> readable datasource
  -> Data

reference Pattern
  -> Contract Query
  -> Database Contract read
  -> target Data

header-detail Pattern
  -> realization binding
  -> Contract Query
  -> Database Contract read
  -> detail Data
```

これはX-TEAの`OpR`相当meaningがTEIでは複数層へ分解されているが、exact link chainが揃っているためderived projection可能だと解釈できる。

### C/U/D

Current TEIには:

```text
Database Contract Writes[]
```

は存在するが、

```text
Application Function / Capability
  -> this mutation authority
```

というaccepted linkがない。

さらに`Writes[]`はC/U/D subtypeを区別しない。

したがってX-TEAの`TaskFunctionIO.OpC / OpU / OpD`に相当するoperation attribution informationが、current TEIではまだlosslessに表現されていない。

## 4. Translation conclusion

X-TEA fallbackから得られる結論は:

```text
Business ProcedureへCRUD letterをauthorする
```

ではない。

むしろ:

> C/U/D attributionがrealistic Referenceで必要になった場合、Business ProcedureよりFunction / I/O realization境界を第一候補として再検討する。

である。

Business Procedureは:

```text
what business work happens
who / when / why
manual / external / judgment / handoff
which Function is used
```

を保持し、FunctionがDataへどう作用するかというoperation meaningは別authorityへ残す。

## 5. TEI option refinement

### X0 — Business Procedure CRUD annotation

```text
BP-07 U sales.order
```

X-TEA translation上もrejectする。

理由:

- historical CRUD sourceはFunction/Table I/OでありBusiness Procedure textではない
- Data / Function側authorityとの二重SOTになる
- realization変更でdriftする

### X1 — Application Function / Pattern Data-I/O operation relation

Conceptual example:

```text
Application Function
  -> Data I/O relation
     target semantic Data / Attribute
     operation C/R/U/D
```

X-TEA shapeへ最も近いtranslation。

Pros:

- Function reviewからCRUD matrixを直接derive可能
- Business Procedureを太らせない
- semantic Data identityへ結べる

Risks:

- `Application Pattern`は現在UI-independent intentであり、persistence operationまで所有させると責務が膨張する
- Capability / Query / Database Contractとのduplicate authorityになり得る
- external API / physical action等をCRUDへ押し込む危険

Current: **candidate only / not adopted**。

### X2 — Capability / Application realization -> mutation authority binding

Conceptual example:

```text
Application + Capability
  -> effect realization
     Database Contract mutation target
```

Pros:

- Semantic Capabilityをpersistence-neutralに維持できる
- provider / database dependentなeffectをrealization側へ置ける
- Custom Go / Plugin / external API realizationの違いを保持できる

Risks:

- Business reviewでFunction/Data CRUDを得るためにrealizationまで辿る必要がある
- one Capabilityが複数effectsを持つ場合のaggregation semanticsが必要
- Database以外のeffectとのgeneralizationを先回りしやすい

Current: **strong future candidate / evidence insufficient**。

### X3 — Database mutation contract + explicit Function binding

Current Query Contractに対するmutation counterpartを設け、Function / Capability realizationからtarget identityへbindする案。

Conceptual example:

```text
Database Contract Mutation
  target Data / Attribute
  operation semantics

Capability realization
  -> Database Contract Mutation target
```

Pros:

- read Query Contractと対称に近いexplicit mutation authorityを持てる
- physical SQL本文からoperationを推測しない
- Database ownership / access policyとcross-checkしやすい

Risks:

- current Database Contract `Writes[]`とのrelationを整理する必要
- CRUD C/U/DをDatabase contract vocabularyへ固定してよいか未証明
- external non-DB effectには別modelが必要

Current: **future pressure / not adopted**。

### X4 — generic Capability Effect

```text
capability ... {
  effect ...
}
```

Currentはdeferする。

X-TEAのCRUDを一般化して`effect`へ広げるのは、current C/U/D exampleが0の段階では抽象化が強すぎる。

## 6. Predecessor mutation-boundary comparison

Current TEIの前身設計であるSemantic PlatformのAction execution draftでは、Actionは必要に応じてsubject Dataを持ち、internal Action callを明示し、runtime execution中の`create / update / delete` stepごとにownershipを再検査していた。

概念上は:

```text
Action
  subject Data
  explicit internal call
  create/update/delete step
        ↓
Runtime mutation gate
  owner/subsystem check
        ↓
commit
```

である。

さらにexternal implementationへ無制限なbusiness DB commit権限を渡さず、result / `MutationIntent`を返してRuntime側でvalidate / commitする方向が検討されていた。Detailed MutationIntent schemaは当時もopenだった。

このpredecessor designをcurrent TEIへそのまま戻さない。特にAction Script、Subsystem、MutationIntentをcurrent Canonical kindとして復活させる根拠にはしない。

一方、設計圧力としては重要である。

### 6.1 X2との整合

Predecessor boundaryは:

```text
Semantic/Business Function meaning
  != unrestricted physical DB mutation
```

を明確に分けていた。

External / custom implementationが直接DB authorityになるのではなく、mutationをRuntime-controlled authorityへ戻す方向は、X2の:

```text
Application + Capability realization
  -> explicit mutation authority
```

と整合する。

したがってX2は単なる新規抽象化案ではなく、TEI系譜内でも「実装をauthorityにしない」という既存設計原則の延長として比較できる。

### 6.2 X3との整合

Predecessor `MutationIntent`は詳細schema未確定だったため、current Database Contract / Query Contract分離を踏まえると、mutationをDatabase-owned contractとして明示しFunction realizationからbindするX3も自然な比較候補になる。

ただし:

```text
old MutationIntent
  != current Database Mutation Contract
```

であり、名前やshapeを継承しない。

### 6.3 Direct / effective operationとの関係

Predecessor Action modelではinternal callが明示され、旧Reference/Studioではdirect CUDとsame-Subsystem call propagationによるeffective CUDを別表示していた。

したがってfuture mutation attribution modelは、少なくとも:

```text
direct mutation authority
explicit Function call dependency
effective operation projection
published / ownership boundary
```

を区別できる必要がある。

Flattened CRUD cellだけをCanonical SOTにしない。詳細は Part 4 を参照する。

### 6.4 Current TEIとの差

Current `semantic.Capability`はinput / output / outcomeのみを持ち、call/effect/mutationを所有しない。

Current `Application Pattern command`もCapability identityだけを参照し、persistence effectを所有しない。

そのためpredecessor boundaryをcurrentへ適用するには、まずrealistic C/U/D caseから「何を失っているか」を証明する必要がある。

## 7. Current ranking

Current Evidenceだけなら:

```text
X0 Business Procedure CRUD     -> reject
X1 Function/Data-I/O relation  -> plausible, responsibility risk
X2 realization effect binding  -> plausible, current TEI separation + predecessor mutation gateに整合しやすい
X3 mutation contract + binding -> plausible, Query Contract分離 + predecessor MutationIntent pressureと比較可能
X4 generic Effect              -> defer
```

X2 / X3のどちらがよいかは、Create / Delete / non-state UpdateのTEI-original realistic Referenceがないため決めない。

## 8. Required future pressure

Qualifying C/U/D exampleが自然に現れた場合、少なくとも次を比較する。

1. 同じBusiness Action / Function meaningを保ったままrealizationをDatabase -> external APIへ変えた場合、operation attributionはどこに残るべきか。
2. one Capabilityが複数Data / resourceへeffectを持つ場合、matrix cellをどうderiveするか。
3. Create / DeleteがSemantic Data lifecycle meaningなのかpersistence effectなのか。
4. non-state UpdateとBusiness Lifecycle transitionをどう分離するか。
5. external-observed Databaseに対するwrite prohibitionをoperation relationからcross-checkできるか。
6. operation projectionをBusiness Procedure / Function / Dataの三視点reviewへどう戻すか。
7. direct mutationとexplicit Function-call由来のeffective mutationを分離できるか。
8. published / cross-boundary Function内部mutationをcallerのdirect CUDへ誤帰属しないか。
9. custom / external implementationにunrestricted DB authorityを渡さず、accepted mutation authorityを検査できるか。

## 9. Current production boundary

今は追加しない。

- Business Procedure CRUD field
- Application Pattern generic CRUD field
- Semantic Capability generic Effect
- Database Contract C/U/D enum
- mutation binding
- MutationIntent
- Function/Data-I-O canonical kind
- direct/effective CRUD production fields
- Action Script / generic call DSL

Current partial R projectionを維持し、C/U/Dはunknownを保持する。

## Part 3 結果

Historical shapeをそのまま移植しない翻訳結果は:

```text
X-TEA
  Function/Table I/O owns explicit CRUD marks
  -> matrix derived

TEI predecessor
  Action subject + explicit call + mutation gate
  -> direct/effective CUDを分離
  -> external implementationはunrestricted DB authorityではない

TEI current
  R meaning is distributed but exact-link recoverable
  C/U/D attribution link is missing
  Capabilityにはcall/effect/mutation authorityがない

TEI direction
  keep Business Procedure thin
  investigate Function / realization / mutation-authority boundary when realistic C/U/D evidence appears
  preserve direct/effective distinction as future guardrail
  derive matrix; do not author matrix
```

---

# Part 4 — Direct / Effective Boundary Pressure（direct/effective CUD区別を将来guardrailとして保持）

## 1. 目的

Current TEIではBusiness Action × CRUDのRを一部derived projectionできる一方、C/U/D attributionはaccepted Function -> mutation authority link不足により未解決である。

将来そのlinkを追加する場合、CRUDを単純な一枚のcellへ潰す前に、Action / Function自身のoperationと、Function callを通じて影響するoperationを区別する必要があるかを確認する。

本書は、過去のSemantic Platform / X-TEA vitality workで得たdirect/effective CUD設計を、current TEIへschemaとして移植するのではなく**失ってはいけないreview pressure**として保持する。

## 2. Prior design evidence

Prior Semantic Platform Reference Model workではFunction viewpointを:

```text
Action + View + CRUD/read/call cross-reference
```

として再構成し、Subsystem reviewでは:

```text
owner + direct/effective CUD + exposed Action boundary
```

を用いた。

MTO `sales_order` Reference pressureでは、local Data pageにrelevant Actions / Views / Proceduresを表示しつつ、direct CUDとeffective CUDを別表示した。

意味は:

```text
direct CUD
  = Action自身が行うmutation

effective CUD
  = same-Subsystem internal Action callを通じて到達するmutation
```

である。

Cross-Subsystem published Action boundaryではeffective CUD inheritanceを停止し、caller側へcallee内部mutationを直接所有させない。

## 3. Why this matters to current TEI

Current T-GAP-08では将来のoperation attribution authority候補として:

```text
Function/Data-I/O relation
Application + Capability realization -> mutation authority
Database mutation contract + explicit Function binding
```

を比較している（Part 3参照）。

どの案を採る場合でも、次を一つのCRUD cellへ無差別に畳むと設計review能力を失う。

```text
Function itself mutates Data
Function calls another Function that mutates Data
Function calls a published cross-boundary Function whose internal mutation is encapsulated
```

特にSubsystem / ownership reviewでは:

```text
who directly mutates owned data?
who merely depends on another subsystem's published action?
```

を区別できる必要がある。

## 4. Current TEI does not yet implement this

Current TEI Lane Cには:

- accepted C/U/D attribution relationがない
- qualifying TEI-original C/U/D Business Actionがまだない
- current Business Procedure ReferenceにはFunction call graphをoperation projectionへ合成するcontractがない

したがって今:

```text
DirectCRUD
EffectiveCRUD
```

というproduction field / type / matrix contractを追加しない。

## 5. Future semantic rule candidate

C/U/D attributionが採用される場合のguardrail候補:

```text
Direct operation
  authority = Function / realization自身のaccepted mutation relation

Effective operation
  authority = explicit Function call graph
              + callee direct/effective operation
              + propagation boundary policy
```

Names, Business Procedure adjacency, UI navigation, SQL textからcall/effectを推測しない。

### Same boundary

Same semantic ownership / subsystem boundaryでinternal callがacceptedされる場合:

```text
caller effective CUD
  may include callee direct/effective CUD
```

### Published cross-boundary Action

Cross-subsystem published Action / contract boundaryでは:

```text
caller direct CUD
  does not inherit callee mutation

caller effective dependency
  may record call / contract dependence

callee internal CUD
  remains callee-owned
```

とする方向がpredecessor designとX-TEA Subsystem pressureに整合する。

これはcurrent production ruleではなくfuture comparison criterionである。

## 6. X-TEA relation

E-007ではexternal-table C/U/DをSubsystem smellとして扱うHistorical pressureを記録している。

E-016では`TaskFunctionIO`にexplicit `OpC / OpR / OpU / OpD`があり、CRUD matrixがFunction/Table I/O informationからderiveされることを確認した。

Direct/effective distinctionを加えると、current translation pressureは:

```text
Function direct Data-I/O operation
        +
Function call dependency
        +
Subsystem / published-boundary semantics
        ->
review projection of direct / effective CRUD
```

となる。

Business Procedureはこのoperation graphのSOTにはしない。

## 7. Failure modes this guardrail prevents

### F1 — callerをdirect mutatorと誤表示

A calls B, B updates Data.

Flattened matrix:

```text
A = U Data
```

だけでは、A自身が直接Dataを更新するように見える。

### F2 — cross-subsystem encapsulation loss

Sales Action calls Inventory published Action.
Inventory internals update Inventory Data.

Callerへeffective Uを無条件伝播すると:

```text
Sales appears to mutate Inventory Data
```

となり、published contract boundaryが失われる。

### F3 — ownership diagnostic false positive

Effective CUDをdirect CUDとして扱うと、合法なpublished Action利用をcross-subsystem direct-write violationとして誤診断し得る。

### F4 — impact analysis false negative

逆にcall propagationを完全に捨てると、callee mutation changeがcaller Business Procedure / Function reviewへ与えるimpactを見落とす。

## 8. Future pressure requirements

T-GAP-08をreopenしたとき、operation model候補は少なくとも次をPressureする。

1. direct mutationをexactに識別できる。
2. explicit Function callを通じたeffective effectをderivedできる。
3. same-boundary propagationとcross-boundary encapsulationを分けられる。
4. published Action内部のData mutationをcaller ownershipへ移さない。
5. direct/effective distinctionをBusiness Procedureへ重複authorしない。
6. Function call changeだけでeffective projectionが追随する。
7. callee implementation changeでdirect semantic operationが変わらない場合、不要なBusiness meaning changeを出さない。
8. external API / physical work等のnon-CRUD effectを無理にCUD propagationへ含めない。

## Part 4 現時点の判断（全体の現在の結論）

```text
Direct/effective CUD distinction
  = future guardrailとして保持

Production schema
  = not adopted

Current partial R projection
  = unchanged

Current C/U/D
  = unresolved
```

Qualifying TEI-original C/U/D exampleとexplicit Function-call/effect authorityが揃うまではimplementationへ進めない。
