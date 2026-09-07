# X-TEA Design Evidence E-016 — Explicit CRUD Marks Live on Function/Table I/O

Status: **Lane C research evidence / E-016 / no TEI production contract adoption**

## Evidence grade

```text
S1 + T1
```

- S1: public XeadModeler source behavior
- T1: TEI-local CRUD projection / gap pressure

## Public source provenance

Repository:

```text
xead/XeadModeler
```

Relevant source:

```text
src/xeadModeler/DialogMatrixList.java
src/xeadModeler/DialogImportXEAD.java
src/xeadModeler/DialogDocuments.java
```

## Observed X-TEA structure

`TaskFunctionIO` carries explicit Table I/O operation information including:

```text
TableID
OpC
OpR
OpU
OpD
```

`DialogImportXEAD.java` explicitly copies those attributes when importing the model.

`DialogMatrixList.java` walks `TaskFunctionIO`, groups by Function ID + TableID, and accumulates the explicit `OpC / OpR / OpU / OpD` marks into CRUD cells.

Therefore historical CRUD matrix generation is structurally closer to:

```text
Function
  -> Table I/O
     -> explicit operation C/R/U/D
        -> derived CRUD matrix
```

than to:

```text
Business Procedure prose
  -> keyword inference
  -> CRUD matrix
```

or:

```text
SQL source text
  -> AST/keyword inference
  -> CRUD matrix
```

## Relation to E-003

E-003 established that CRUD matrices are derived projections from structured design information.

E-016 sharpens the provenance of that structured information:

> In XeadModeler, CRUD operation marks are explicit Function/Table I/O information, and the matrix aggregates them.

This matters because TEI has intentionally decomposed the historical Function object into Application Pattern, Semantic Capability, Query, Database Contract and Realization boundaries.

## TEI-local pressure

Current `order_entry` proves that some R cells can be recovered from the decomposed TEI structure:

```text
BP-00 -> browse Pattern -> Query -> readable datasource -> R sales.order
BP-02 -> reference Pattern -> Contract Query -> Contract read -> R customer.customer
BP-03 -> header-detail realization -> Contract Query -> Contract read -> R sales.order_line
```

Current TEI does not have an equally explicit accepted link for C/U/D:

```text
Application Function / Capability
  -> mutation authority
  -> exact Data / Attribute
```

Database Contract `Writes[]` exists, but it neither distinguishes C/U/D nor attributes a write to a Business Action / Function.

## What E-016 supports

E-016 supports:

1. CRUD matrix should remain derived, not authored as a second SOT.
2. Business Procedure wording must not be the source of CRUD letters.
3. If C/U/D attribution becomes necessary, Function/Data-I/O or Function/realization/mutation-authority boundaries are stronger first candidates than Business Procedure annotations.
4. SQL/provider implementation must not be parsed to invent semantic CRUD meaning when an accepted structured operation relation is absent.
5. `no current CRUD trace` must remain unresolved rather than being filled by naming heuristics.

## What E-016 does not prove

E-016 does **not** prove that TEI should introduce:

```text
ApplicationPattern.OpC / OpR / OpU / OpD
Capability.Effect
DatabaseMutationContract
FunctionDataIO canonical kind
```

It also does not prove that CRUD is the right vocabulary for every TEI effect. External API calls, message publication, physical work, document production and other effects may not be usefully reducible to C/R/U/D.

Historical `TaskFunctionIO` shape is evidence of the design pressure, not a schema to clone.

## Current TEI decision

```text
Business Procedure CRUD annotation -> reject
Function/Data-I/O operation relation -> plausible future candidate
Capability realization -> mutation authority binding -> plausible future candidate
Database mutation contract + explicit Function binding -> future pressure
Generic Capability Effect -> defer
```

No candidate is promoted because the current TEI-original Reference corpus still has no qualifying Create / Delete / non-state Update Business Action with independently accepted mutation authority.

Related TEI-local evidence:

```text
BUSINESS_ACTION_CRUD_PROJECTION_PRESSURE_2026-09.md
BUSINESS_ACTION_CRUD_OPERATION_GAP_AUDIT_2026-09.md
BUSINESS_ACTION_CUD_TEI_ORIGINAL_REFERENCE_SEARCH_2026-09.md
BUSINESS_ACTION_CRUD_XTEA_FUNCTION_TABLE_IO_TRANSLATION_2026-09.md
BUSINESS_PROCEDURE_BP01_PREPARE_OPERATION_AMBIGUITY_PRESSURE_2026-09.md
```

## T-GAP-08

```text
T-GAP-08
When a realistic TEI-original C/U/D case appears, what is the minimum accepted
Function/Data-operation attribution relation, and should its authority live in:

- Function/Data-I/O meaning,
- Application + Capability realization,
- Database mutation contract + explicit binding,
- or another narrower domain-specific boundary?
```

T-GAP-08 must be answered with at least one independently motivated realistic C/U/D Reference. Matrix completeness alone is not a reason to create the missing contract.
