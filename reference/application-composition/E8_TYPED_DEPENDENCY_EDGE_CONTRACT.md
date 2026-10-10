# E8: Typed dependency edge contract (proposal)

Status: design proposal only; not a validated resolver, source closure, or runtime authority. Parent: #1016 / PR #1051.

## Ownership

- Existing semantic/application validators own reference meaning and scoping.
- A read-only adapter may export *resolved* edges; it must not invent a second resolver.
- Source Index owns definition evidence and artifact fingerprints, not semantic resolution.
- Composition owns traversal, impact reporting, and fail-closed diagnostics, not deployment approval.

## Candidate interface

```go
// Sketch only; package placement and names remain open.
type DependencyEdge struct {
    From subject.Subject
    To subject.Subject
    Relation string
}
type ResolutionIssue struct {
    From subject.Subject
    Field string
    Reason string
}
type DependencySnapshot struct {
    Edges []DependencyEdge
    Issues []ResolutionIssue
    Complete bool
}
```

`Complete` is false if any reference category in the declared analysis scope is unsupported, ambiguous, or unresolved. It must not become true merely because `Issues` is empty: the adapter must declare and exhaustively handle the categories in scope. A successful semantic closure is not equivalent to application compatibility or authorization.

## Resolution requirements

1. Scope an application-local `Pattern.Datasource` through the owning Application, then resolve the unique database alias. Do not construct a global Subject directly from the local name.
2. Scope `Pattern.ContractQuery.Query` through the selected Contract. Same-name queries in distinct contracts are not interchangeable.
3. Resolve Pattern Data/Relation/Query/Capability and semantic Attribute/Relation/Validation/Capability type references with the authoritative existing validation rules.
4. Preserve edge provenance and deterministic ordering. Deduplicate identical resolved edges only after their scope is established; never suppress conflicting resolutions.
5. Traverse cycles with a visited set, but leave cycle validity to semantic rules. An unknown edge kind or ambiguous definition must result in an incomplete snapshot.
6. Join every reached Subject to exactly one `sourceindex.RoleDefinition` artifact and fingerprint. A missing or ambiguous source is unresolved, not an unchanged dependency.
7. Ensure parsed World and Source Index describe the same immutable input bytes. A paired JSON envelope by itself does not prove this.

## Acceptance tests before promotion

- Pattern Capability changes while direct Application references remain unchanged.
- Pattern Data -> Attribute -> Type change.
- Relation target or key change.
- Two Applications reuse an alias name but point to different databases.
- Two Contracts reuse a Query name with different definitions.
- Missing/ambiguous source, unknown edge category, and unresolved reference all fail closed.
- Cyclic graph traversal terminates without inventing semantic acceptance.
- E5/E6/E7 existing output and exit codes remain backward compatible.

This document does not claim these tests have been implemented or executed. The E8 interface should be implemented only after identifying the concrete authoritative validator entry points and confirming that their behavior can be reused without duplicated meaning.

## Interface design review: completeness and evidence

A plain `Complete bool` is not independently auditable. Before implementation, require a declared coverage manifest containing supported reference kinds, excluded kinds, validator version, and input snapshot identity. The caller must reject a claimed complete result when its coverage manifest does not cover the requested analysis scope. `ResolutionIssue` should carry a stable reason code and, when available, a source Subject; free-form text alone is unsuitable for regression assertions. Edge identity must include the fully resolved target and relation, not a local alias or query name.

The existing `sourceindex.Index.Validate(read)` checks artifact hashes and ranges against bytes supplied to its callback, but this alone does not establish that the `ir.World` parser consumed those same bytes. Keep the source-byte identity requirement explicit. The existing `sourceindex.Subject` is an alias of `internal/subject.Subject`, a path-independent `(Kind, ID)` pair; scoped names must be resolved before constructing that pair.

No runtime API is changed by this review. The concrete semantic validator entry points have not yet been located or executed, so the proposed adapter remains conditional.

## Existing E5 resolver audit (2026-10-10)

The reference `e5-world-index.go` already traverses selected direct Pattern Data, Relation, Capability and Query references, database aliases and required contracts, ContractQuery bindings, and validation references. It also constructs lookup/count maps from `ir.World` itself (`patternTargets`, `queryOwners`, `contracts`, `aliasTargets`, `attributeDefinitions`). Therefore E8 must **not** assume E5 only observes Application-level references, nor should it copy these maps into a second implementation. E5 remains a bounded diagnostic; the maps are local checks, not proof that the authoritative semantic validators were invoked. Refactor or adapt only after locating those validators and establishing equivalence tests for existing E5 outputs.

Recommended first regression fixture: one Application Pattern with a resolved Capability whose input type changes while the Pattern and its direct Source remain identical. Verify the new closure reports a changed reachable Type definition, whereas the E5 bounded observation remains explicitly limited. A second fixture should prove two distinct Application-local datasource aliases cannot be conflated. These are proposed tests, not executed tests.
