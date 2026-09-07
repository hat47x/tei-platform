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

## Current decisions

- D3 remains subtraction/history evidence, not the preferred minimum candidate.
- D4 alone is insufficient if exact protected-target trace is retained.
- O1 minimum cardinality is zero-or-one; multi-target future cases fail closed and reopen design.
- Production O1 sidecar is not preferred because it requires composite Acceptance machinery for one Step-local relation.
- Same Business source内のsingular narrow O1 field is the current strongest Procedure authoring candidate.
- Handoff remains separate from Procedure body because it is genuinely cross-Procedure meaning.
- Reference Set manifest内Handoffはfixtureには便利だが、artifact locator metadataとmeaningを混在させるためproduction authoring candidateとしては弱い。
- Standalone Handoff relation source is the current strongest H1/R3 authoring candidate.
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

The current fingerprint experiments are pressure vocabulary only, not a production hash/revision API.

## Current unresolved gates

1. actual Repository runner green (`gofmt`, `go vet`, `go test`, checker).
2. production adoption/migration strategy for inline singular O1; test-only shape itself is demonstrated.
3. production adoption/migration strategy for standalone Handoff source; test-only shape itself is demonstrated.
4. production Shared Acceptance / Business semantic revision mechanism spanning Procedure source(s) + Handoff source.
5. second realistic Handoff pressure to confirm `from / to / condition` sufficiency.
6. next TEI-original C/U/D case before mutation/effect contract design.
7. evidence that separate Business plane is insufficient before considering `ir.World` embedding.

## Pre-ADR status

```text
Not Ready
```

This file is a concise comparison baseline. The readiness authority remains `BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`.
