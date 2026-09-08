# Business Action × Ownership Boundary Projection Pressure — 2026-09

Status: **Lane C / T-GAP-04 partial answer / derived review view / no Subsystem or SubjectArea production contract adoption**

## 1. 目的

X-TEA / 三要素分析法ではSubsystemが単なるdeployment単位ではなく、少数のTable / Functionをまとめる理解・凝集・疎結合のreview境界として働く。

一方current TEIにはdurable `Subsystem` / `SubjectArea` Canonical nounをまだ持たせていない。

本Pressureは、current TEIにすでに存在する:

```text
Application
Database requirement / access
Database ownership
Business Action -> Application Pattern
Business Action × CRUD partial projection
```

だけを組み合わせ、少なくとも**operation ownership boundary**として有用なreview viewをderiveできるか確認する。

新しいSubsystemを作ることを目的にしない。

## 2. Current authoritative facts

### order_entry Application

```text
application order_entry

sales
  target sales_core
  access read-write

master
  target master_data
  access read
```

### Database ownership

```text
sales_core
  ownership managed-here

master_data
  ownership external-observed
```

### Current accepted CRUD projection

Current Business Action × CRUD pressureでは、exact accepted authorityから次のRだけをderiveできる。

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

その他のStepはcurrent refsからoperation traceがなく、`NoCRUDTraceFromCurrentRefs`として残す。

## 3. Derived ownership boundary view

`business_action_ownership_boundary_projection_pressure_test.go`では、CRUD projectionをoperation gateとして先に解決し、そのaccepted cellへApplication datasource / Database authorityを重ねる。

Current expected rows:

```text
BP-00
  R sales.order
  -> sales_core
  -> ownership managed-here
  -> application access read-write

BP-02
  R customer.customer
  -> master_data
  -> ownership external-observed
  -> application access read

BP-03
  R sales.order_line
  -> sales_core
  -> ownership managed-here
  -> application access read-write
```

重要なのは、ownership viewがCRUDより強いoperation authorityを作らないこと。

```text
Business Action
  -> accepted CRUD cell
     -> Application datasource requirement
        -> Database ownership/access
           -> derived ownership boundary row
```

であり、逆向きに:

```text
Database is read-write
  -> Business Action mutates it
```

とは推測しない。

## 4. Fail-closed composition

初回実装ではreference / header-detail経路がDatabase Contract `Reads[]`消失を独自に再検査しておらず、CRUD projectionより弱いEvidenceでownership rowを作れる余地があった。

これを修正し、ownership projectionは最初に既存Business Action × CRUD projectionを解決する。

したがって:

```text
Contract read authority disappears
  -> CRUD projection fails closed
  -> ownership projection also fails closed
```

となる。

同じoperation semanticsをownership view側で再実装しない。

## 5. Authority direction

Business Procedure sourceへ次をcopyしない。

```text
Database
DatabaseOwnership
Access
Subsystem
SubjectArea
```

`business_action_ownership_authority_direction_pressure_test.go`ではBusiness Procedure fixture / Procedure / Step source typeがこれらを所有しないことをPressureする。

またBusiness wordingを:

```text
maintain customer data in an external subsystem
```

等へ変更しても、Application / Database / CRUD authorityが変わらなければownership projectionは変化しない。

したがって:

```text
Business wording
  != ownership boundary authority

Database / Application source
  -> derived ownership view
```

を維持する。

## 6. Database ownership change pressure

Test-localに`master_data` ownershipを:

```text
external-observed
  -> managed-here
```

へ変更すると、Business Procedure sourceを変更せずBP-02 derived rowだけがその変更へ追随する。

これはownership informationをBusiness Procedureへ複製しなくてもcross-view reviewへ反映できることを示す。

## 7. T-GAP-04を分解する

従来のT-GAP-04:

```text
Can current namespaces / ownership / Application boundaries produce a useful
SubjectArea / Subsystem view without new Canonical nouns?
```

は一問としては広すぎるため、次へ分解する。

### T-GAP-04a — operation ownership boundary

```text
Can current Application / Database ownership/access + evidence-backed CRUD
produce a useful operational boundary view without Subsystem Canonical identity?
```

Current answer: **partial yes for current Reference**。

理由:

- managed-here / external-observedをBusiness Actionから辿れる
- read-only external boundaryを明示できる
- Business sourceへownershipを複製しない
- ownership changeがprojectionへ追随する
- Subsystem nounを必要としない

ただしcurrent operation coverageはRのみなので、X-TEA型のCUD boundary review全体を証明したわけではない。

### T-GAP-04b — business comprehension / SubjectArea boundary

```text
Can current namespace / Application / Database boundaries explain the business
cohesion or subject-area grouping designers need for comprehension?
```

Current answer: **still open**。

Database ownershipは:

```text
who manages / observes this Database
```

を表すが、

```text
what business area belongs together
which Data / Functions should be reviewed as one cohesive design unit
```

を必ずしも表さない。

Applicationもdeployment / execution / dependency meaningを持つため、Business subject areaと同一視しない。

## 8. X-TEA fallback comparison

Historical X-TEA pressureではSubsystemについて:

```text
- comprehension / cohesion review unit
- Table / Function grouping
- external-table C/U/D is a smell
- R may be legitimate across the boundary
```

が重要だった。

Current ownership projectionはこのうち:

```text
external vs managed boundary
read dependency
future direct mutation review hook
```

を部分的に回復する。

しかし次はまだ回復していない。

```text
business cohesion
Function call topology at realistic scale
direct vs effective CUD across published boundaries
subject-area comprehension
```

したがって:

> Database ownership boundary viewが作れたからSubsystem概念は永久に不要

とは結論しない。

逆に:

> X-TEAにSubsystemがあったからTEIにも今すぐCanonical Subsystemが必要

とも結論しない。

## 9. T-GAP-05との関係

T-GAP-05:

```text
At realistic scale, do effective CUD and call dependencies remain understandable
without an explicit Subsystem semantic contract?
```

は引き続きopenである。

Current TEIにはaccepted C/U/D attribution relationもFunction call authorityもないため、direct/effective CUDをcurrent ownership projectionへ追加しない。

将来T-GAP-08でmutation attributionが実証された場合、次を重ねてPressureする。

```text
direct operation
Function call dependency
effective operation
Database ownership
published / cross-boundary contract
```

その時点で初めて、explicit Subsystem semantic boundaryが必要かを再評価する。

## 10. Current production decision

追加しない。

- `Subsystem` Canonical kind
- `SubjectArea` Canonical kind
- Business Procedure ownership field
- Business Procedure datasource/access field
- generic ownership Edge
- Application == Subsystem alias
- Database == Subsystem alias
- namespace == SubjectArea alias

Currentはderived ownership boundary viewをPressureとして保持する。

## 11. Current result

```text
T-GAP-04a operation ownership boundary
  -> partial positive evidence

T-GAP-04b business comprehension / SubjectArea
  -> unresolved

T-GAP-05 effective CUD / call topology at scale
  -> unresolved
```

この結果は、Subsystem / SubjectAreaを早期Canonical化せずにX-TEAの境界reviewの一部を回復できることを示す。

一方、current ReferenceのR-only operation coverageからSubsystem不要を一般化しない。

Actual `gofmt / go vet / go test` runner Evidenceは引き続き未取得であり、本結果はtest-only / pre-contract Pressureである。
