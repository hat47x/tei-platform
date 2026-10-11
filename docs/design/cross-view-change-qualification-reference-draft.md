# Cross-view change qualification — reference contract (draft)

Status: **design candidate / non-authoritative / Issue #1143**. This document does not introduce a production schema, a new Canonical kind, or a second acceptance engine.

## Purpose and boundaries

A subject-local Knowledge Qualification or `qualified-for-disclosure` Builder Review Handoff is **not** proof that a design change is compatible across Business, Data and Function views. Cross-view qualification is an evidence-backed, read-only projection of existing model authority and validators. It may block proposal promotion but cannot itself accept or apply a change.

Existing source owners remain authoritative:

- `internal/application/model.go`: `Application.DatabaseByAliasExact`, `DatabaseRequirement.Access`, `Pattern.ContractQuery`. A resolved reference/read contract is not an edit grant.
- `internal/driver/resolver.go`: `Resolver.ResolvePattern` resolves a Pattern's existing references. Optional nil members of `PatternResolution` are kind-dependent, not automatically missing evidence.
- `internal/validation/application_pattern.go` and `internal/validation/validation.go`: `ValidateApplicationPatterns` and `Validate` own Canonical cross-reference diagnostics. `Diagnostic` does **not** contain a complete provenance/freshness envelope.
- `internal/knowledgecontext/builder_review.go`: `BuildBuilderReviewPlan` requires a cross-view review step, but that plan is guidance, not evidence the review ran.
- `internal/knowledgecontext/builder_review_handoff.go`: `ValidateBuilderReviewHandoff` checks subject-local handoff integrity, not whole-change compatibility.
- `tools/reference-shared-acceptance/main.go`: `attach-qualification` and `propose-qualified-reference-pattern` are existing adapter entrances; authoritative mutation remains in explicit `accept`.

The Business Procedure source of truth is `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`. Its current design is a **separate normalized Business semantic plane**, not `ir.World.Business`. Production source loader and authority cutover are not yet adopted. Use test/reference evidence only until that contract is ready.

## Minimal projection candidate (not yet schema)

| Field | Requirement |
| --- | --- |
| target / operation | Explicit Function or other model subject and requested semantic change |
| baseline | Complete World/Conformance base plus applicable Business semantic/source revision |
| affected subject / relation | Typed, directional semantic reference; reverse *traversal* must not imply reverse authority |
| evidence | Source identity, authoritative source-root membership, source bytes or content fingerprint, revision, scope and temporal applicability |
| required check | Named owning checker, applicability, result, evidence reference and unresolved reason |
| outcome | `eligible-for-proposal`, `blocked-incompatible`, `blocked-unresolved`, or `blocked-stale` |
| limits | No authorization, execution permission, accepted decision, or inferred business fact |

A required missing, stale or incompatible check blocks proposal promotion. An optional check must be explicitly marked optional with a reason. Stable output ordering and bounded traversal must prevent cycles from causing nondeterminism. No new global Business Procedure/Step/Handoff IDs.

## First reference scenario

1. Begin with the existing order-entry customer reference Pattern and explicit read-only master-data contract.
2. Show that subject-local Knowledge Qualification can succeed for **reference reuse**.
3. Request `enable-customer-edit`. Do not infer write authority from a readable datasource, Pattern kind, contract-query presence, or observed CRUD.
4. Project `blocked-incompatible` for missing/contradictory explicit write capability; independently project `blocked-unresolved` if an authoritative Business owner/procedure is unavailable. Do not invent a customer-maintenance Handoff.
5. With a missing cross-view result, do not promote the subject-local handoff into a whole-change proposal.
6. Preserve the existing Shared Acceptance review, expected fingerprint and explicit Accept/Reject boundary; no alternate write path.

Accepted Business Handoff evidence currently includes `order-entry.order-handling/BP-09 -> order-entry.follow-up/FU-00` with an explicit condition. It is **not** evidence for customer maintenance. Accepted CRUD traces are read-only: `BP-00 -> R sales.order`, `BP-02 -> R customer.customer`, `BP-03 -> R sales.order_line`. No C/U/D follows by inference.

## Adversarial acceptance matrix

| Input | Expected result |
| --- | --- |
| Read-only reference contract used for read | Applicable to read; no implied edit grant |
| Edit request on read-only contract | Blocked incompatible |
| Edit request without accepted Business responsibility | Blocked unresolved |
| Qualified local handoff without cross-view evidence | Blocked unresolved |
| World changed while Business sources unchanged | Blocked stale |
| Business source added/removed or source-root membership changed | Blocked stale |
| Duplicate managed `Procedure.id` even with identical body | Authority conflict; blocked |
| Procedure step renamed without corresponding explicit Handoff update | Invalid endpoint; blocked |
| Directional Handoff reverse-traversed for impact | No reverse semantic authority inferred |
| Cyclic model references | Finite, deterministic diagnostics |

The freshness envelope must include source bytes, source-set membership, authoritative root membership, logical uniqueness, and World/Conformance base. Git HEAD equality alone is insufficient.

## Implementation sequencing

1. Locate and record the exact Business Procedure test/reference normalization and conformance functions; do not guess a package or assume a production loader.
2. Inspect `decodeQualificationInput`, `attachQualificationEvidence` and `proposeQualifiedReferencePattern` to choose a proposal-stage guard without altering existing Handoff v1.
3. Implement a read-only projection adapter reusing owning validators and source evidence.
4. Add the adversarial reference fixtures above and focused Go tests.
5. Obtain trusted clean exact-head Repository Health evidence before any non-draft merge. This document is **not** a test receipt.

Related: #482, #511, #1026, #1048, #1060, #1143.
