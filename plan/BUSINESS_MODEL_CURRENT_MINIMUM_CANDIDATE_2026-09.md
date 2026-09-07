# Business Model Current Minimum Candidate — 2026-09

Status: **Lane C / current comparison baseline / no production adoption**

## Current minimum semantic candidate

```text
BusinessModel
  Procedures[]
    Procedure
      id
      purpose
      trigger
      actor
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

## Placement

Current strongest placement candidate is a **separate normalized Business semantic plane** validated against `ir.World`.

```text
normalized Business model
  <-> cross-view validation <-> ir.World
```

Business meaning does not need to be embedded in `ir.World` merely because it is first-class design meaning.

## Current decisions

- D3 remains subtraction/history evidence, not the preferred minimum candidate.
- D4 alone is insufficient if exact protected-target trace is retained.
- O1 production sidecar is not preferred because it requires composite Acceptance machinery for one Step-local relation.
- O1 minimum cardinality is zero-or-one; multi-target future cases fail closed and reopen design.
- Handoff remains separate from Procedure body because it is genuinely cross-Procedure meaning.
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
2. production authoring representation for singular narrow O1.
3. production placement of H1/R3 set-level relation.
4. production Shared Acceptance / semantic revision mechanism.
5. second realistic Handoff pressure to confirm thin relation sufficiency.
6. next TEI-original C/U/D case before mutation/effect contract design.
7. evidence that separate Business plane is insufficient before considering `ir.World` embedding.

## Pre-ADR status

```text
Not Ready
```

This file is a concise comparison baseline. The readiness authority remains `BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`.
