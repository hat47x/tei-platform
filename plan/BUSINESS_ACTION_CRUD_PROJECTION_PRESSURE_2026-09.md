# Business Action × CRUD Projection Pressure — 2026-09

Status: **Lane C / X-TEA CRUD matrix pressure / test-only derived projection / no production schema adoption**

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

## 13. Verification status

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

## 14. 現時点の判断

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
