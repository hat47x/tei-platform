# Business Model Current Minimum Candidate — 2026-09

Status: **Lane C / current comparison baseline / no production adoption**

## Current minimum semantic candidate

```text
BusinessModel
  Procedures[]
    Procedure
      id
      purpose
      trigger { epistemicStatus, meaning }
      actor   { epistemicStatus, businessLabel }
      Steps[]
        localKey
        Body
          businessAction
          applicationPatterns[]
          validations[]
          capabilities[]
        protectedApplicationPattern?

  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

Interpretation:

```text
D4 body
+ singular narrow O1
+ S4-A Procedure-scoped Step local address
+ H1/R3 thin set-level Handoff
```

## Current authoring candidates

### Procedure / O1

Current strongest authoring candidate is the same Business Procedure source with a singular optional field:

```text
protectedApplicationPattern?
```

Test-only inline candidate:

- strict-decodes
- round-trips losslessly
- keeps zero-or-one cardinality
- rejects multi-target / generic role fields
- normalizes to the same D4+O1 meaning as the previous sidecar pressure

Production syntax is not adopted yet.

### Handoff

Current strongest authoring candidate is a standalone set-level Business relation source:

```text
Handoffs[]
  from { procedure, step }
  to   { procedure, step }
  condition
```

It intentionally does not carry Procedure artifact locators.

This keeps:

```text
Business Handoff meaning
  != Reference Set / file assembly metadata
```

and avoids rewriting Handoff source merely because a Procedure file moved.

Production syntax is not adopted yet.

## Placement

Current strongest placement candidate is a **separate normalized Business semantic plane** validated against `ir.World`.

```text
normalized Business model
  <-> cross-view validation <-> ir.World
```

Business meaning does not need to be embedded in `ir.World` merely because it is first-class design meaning.

Current tests pressure Business wording change、protected-target-only change、Handoff condition-only change as Business semantic revision changes while leaving `ir.World` unchanged. Stale Application target still fails closed through cross-view validation.

## Shared Acceptance pressure

Current test-only bundle combines:

```text
inline Procedure source A
inline Procedure source B
...
standalone Handoff source
```

into one normalized Business model.

The bundle pressure establishes that:

- the new source decomposition normalizes to the same current Business meaning
- Procedure wording change changes the Business semantic revision
- protected-target change changes the same Business semantic revision
- Handoff condition change changes the same Business semantic revision
- Procedure source ordering alone does not change the semantic revision
- semantic bundle fields do not carry path / artifact / runtime metadata

Therefore the unresolved question is no longer whether multiple Business source kinds can form one semantic review unit.

The remaining production question is:

```text
how to implement source freshness
+ coordinated / atomic acceptance
+ semantic revision identity
+ reload / cross-view Conformance
```

without turning raw file layout into authority.

The current fingerprint is pressure vocabulary only, not a production hash/revision API.

## Current decisions

- D3 remains subtraction/history evidence, not the preferred minimum candidate.
- D4 alone is insufficient if exact protected-target trace is retained.
- O1 minimum cardinality is zero-or-one; multi-target future cases fail closed and reopen design.
- Production O1 sidecar is not preferred because it requires composite Acceptance machinery for one Step-local relation.
- Same Business source内のsingular narrow O1 field is the current strongest Procedure authoring candidate.
- Handoff remains separate from Procedure body because it is genuinely cross-Procedure meaning.
- Reference Set manifest内Handoffはfixtureには便利だが、artifact locator metadataとmeaningを混在させるためproduction authoring candidateとしては弱い。
- Standalone Handoff relation source is the current strongest H1/R3 authoring candidate.
- Inline Procedure source(s) + standalone Handoff source can form one normalized Business semantic review unit in current test-only pressure.
- Handoff normalized meaning is required for the current accepted Reference Set, but a standalone top-level Canonical `BusinessHandoff` kind is not yet proven.
- Generic ref-role taxonomy remains deferred.
- Generic Business Edge / full Business Flow / Workflow runtime remain out of scope.
- Business Action × Data / CRUD / ownership / Namespace lenses remain derived projections, not authored second SOTs.
- `ir.World.Business` / production `internal/business` package remain deferred.

## Acceptance pressure

The current test-only semantic review unit is the normalized Business model, not raw source layout.

Required behavior:

```text
Business wording change              -> semantic revision changes
protected Application target change  -> semantic revision changes
Handoff condition change             -> semantic revision changes
logical Step/address change           -> semantic revision / coordinated migration
pure artifact locator move            -> semantic meaning unchanged
source ordering only                  -> semantic meaning unchanged
```

A production Shared Acceptance design must additionally handle:

```text
source freshness at accept time
coordinated Step/address + Handoff endpoint changes
all-source reload after accept
cross-view validation against current ir.World
no partial acceptance on broken referential integrity
```

## Current unresolved gates

1. actual Repository runner green (`gofmt`, `go vet`, `go test`, checker).
2. production adoption/migration strategy for inline singular O1; test-only shape itself is demonstrated.
3. production adoption/migration strategy for standalone Handoff source; test-only shape itself is demonstrated.
4. production freshness / atomicity / semantic revision API for the demonstrated multi-source normalized Business review unit.
5. second realistic Handoff pressure to confirm `from / to / condition` sufficiency.
6. next TEI-original C/U/D case before mutation/effect contract design.
7. evidence that separate Business plane is insufficient before considering `ir.World` embedding.

## Pre-ADR status

```text
Not Ready
```

This file is a concise comparison baseline. The readiness authority remains `BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`.
