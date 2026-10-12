# Local CPU Reuse Discovery — deterministic R0/R1 Reference

Status: **research-only, synthetic fixtures, not production or TEI Canonical integration** (2026-10-13).
Design owner: [LOCAL_CPU_COMPONENT_REUSE.md](../../plan/research/LOCAL_CPU_COMPONENT_REUSE.md); related TEI #843, #876, #997, #998, #267, #1121, and SEI #543 / #546.

## Purpose

Test a narrow part of reuse discovery **without an LLM, training, embeddings, or network**. It compares caller-provided, pre-extracted feature records from different Applications. It ranks a bounded pair list and keeps `reuse_candidate`, `pattern_candidate`, `needs_semantic_review` and `incompatible` separate. A score derived from labels/IDs is a lexical hint, **not confidence, semantic authority, or an acceptance decision**.

This is **not** a system that extracts trustworthy contract fingerprints from actual TEI Source. A caller cannot claim Verified/Accepted semantics simply by providing an ID and fingerprint. An upstream accepted-source resolver, source provenance verification, authorization, and downstream #267 acceptance/reload/conformance remain to be implemented/integrated. Exact ID/fingerprint equality in these fixtures illustrates what the comparison would do *after trusted projection*, not proof of semantic equivalence across real applications.

## Research observation schema (not Canonical)

One object per observed component/function:
- `id`, `application`, `kind` = `ui` or `logic`
- `source: { locator, digest, revision }` to trace observed input
- `contract?: { id, fingerprint }`: claimed exact contract metadata; **not cryptographically validated**
- `signature: { inputs: [{name,type,nullable}], outputs: [...], effects: [], failures: [] }`
- for UI only: `platform: { target, framework }`
- optional `labels: []` for low-authority lexical hints

The comparator never writes a Source, runs Actions, modifies Canonical, or calls a model. It also never establishes whether two business rules are genuinely the same; a `reuse_candidate` is a *reviewable suggestion*, not an automatic substitution.

Pair rules, in priority order:
1. Different function kinds or incompatible input/output/effect/failure shapes: direct implementation reuse is incompatible.
2. Missing contract ID/fingerprint or different contract IDs: semantic review required, even with matching names.
3. Same ID but different fingerprints: contract drift; **no implicit fallback**.
4. UI with matching contract/shape but different Target/Framework: pattern-level candidate only, not direct renderer/code reuse.
5. Identical declared contract/fingerprint/shape on a compatible Target: possible direct component candidate, requiring external verification and acceptance.

Revision and Source traces are retained for review; this Reference does **not** perform real current-revision checks. Configurable variations and migration adapters require a later design; "incompatible" means **not a direct drop-in** under the tiny Reference rules.

## Run the tests

```sh
node --test reference/local-cpu-reuse/compare-candidates*.test.mjs
node --test reference/local-cpu-reuse/observe-tei-pattern.test.mjs
```

On 2026-10-13 the 14 original cases and 3 additional regression cases were **17/17 PASS in a sandboxed JavaScript evaluator** loaded from the exact GitHub blob contents (not a Node.js repo checkout). The observer's seven cases passed locally with pinned copies of the actual TEI Source contents on Node.js 22.16.0. GitHub exact-PR-head Node execution and full Repository-health remain **NOT_PROVEN**; none of these results is product qualification.

The tests use **invented** Application names, simplified UI and Action signatures and **synthetic/noncryptographic** digests. They check same-contract candidates, deceptively similar names, changed effects/fingerprints, UI framework differences, unknown contract, deterministic output, duplicate IDs and out-of-scope output filtering. These are not two deployed business systems, and the tests do not claim real savings, performance scalability or model accuracy.

### Authorization boundary

`allowedApplications` is a **filter of already-authorized input**. It does not authenticate users or grant project access and cannot stop an upstream caller from fetching prohibited data first. The production integration must enforce access policy **before features are extracted or supplied to this Reference**; excluded data must not enter the comparator process or training corpus in the first place.

## Next grounded experiment

Select two **explicitly authorized** TEI Application Reference sources; construct a read-only feature projection with true Source revision, verified Contract IDs/fingerprints, Application identity and documented unknown fields; compare expected review decisions against deterministic output. Preserve records that are unrelated, surface-similar but semantically incompatible, and cross-framework similarities. Report candidate recall, false reuse suggestions, unknown/abstention, review cost, peak memory and p50/p95 CPU latency separately.

Avoid new TEI-LD teacher/model-training branches: #843 is issue-only subject to #876 and #267 qualification/Acceptance requirements. Do not merge or enable this Reference by claiming its synthetic test proves a real cross-Application component extraction.


## Pinned source observation probe (R0 extension, 2026-10-13)

`observe-tei-pattern.mjs` and `observe-tei-pattern.test.mjs` add a tiny **Node CPU-only** source observation probe. Its seven cases reference existing `reference/workspace/applications/order-entry/patterns/` files: `submit-order.tei`, `order-lines.tei`, and `customer-reference.tei`. Source bytes must match their Git blob SHAs before a structural observation is returned. A changed source, unexpected syntax, or unrecognized extra clauses must not be promoted to a complete Pattern observation. The nested `contract-query` in the customer-reference Pattern is intentionally `partial`.

This is **one real TEI Reference Application**, not two independent cases of business reuse; reading it does not establish Accepted Canonical semantics. Unlike the original synthetic pair comparator, the observer deliberately returns `contract:null`, `acceptance:unknown`, and `source_observation_only`. Its observations must **not** be fed to the candidate comparator as if their Contract ID/fingerprint, effect or IO semantics had already been verified. A TEI-native Accepted Canonical/Contract projection remains a separate integration step.

For review, execute `node --test reference/local-cpu-reuse/observe-tei-pattern.test.mjs` from a complete repository checkout. A local Node run against pinned copies of the same three source contents passed **7/7**, but **exact-PR-head checkout execution has not yet been qualified**. As with the comparator, these tests cannot authorize writes, learning, or standardization.

## Existing Canonical inspect/v0 feature projection (2026-10-13)

[project-inspection-v0.mjs](project-inspection-v0.mjs) is a read-only adapter for the **existing** `internal/operations.InspectSubject` result (`tei.operations.inspect/v0`), not a new TEI parser, CLI registry, or second Canonical model. It was compared with the source-of-truth Go contracts on current `main`: `internal/operations/inspect.go`, `internal/application/model.go`, `internal/semantic/model.go`, and `internal/sourceindex/model.go`.

Supported observation subjects are:
- `application`: explicit `patterns[].id` membership, semantic namespace and validation references, database aliases. `coverage:membership_only`; **not** the full Application contract.
- `application-pattern`: declared Pattern kind, Data/Relation/Query/Capability references, optional datasource and `contractQuery` invocation with exact input bindings.
- `semantic-relation`: target, cardinality, explicit key bindings.
- `semantic-data`: attribute types, relations, candidate keys, and functional dependencies.

Outputs are `tei.research.inspection-features/v0` with `effect:read-only`, `authority:derived_observation_only`, `acceptance:not_asserted` and `contractConformance:not_verified`. World and Source Index fingerprints are **descriptive snapshot references**; they are not proof of Acceptance, executable rights, issuer authenticity, or approval. Do not derive a shared-implementation contract fingerprint from mere matching IDs. `SourceStatus:indexed` identifies a Source Index reference, not a credential, tenant authorization, or disclosure grant.

`linkApplicationPattern` checks **explicit** Application.Pattern membership, and `linkPatternRelation` checks an exact declared Relation ID, both only within the same observed World **and** Source Index fingerprint. All links return `accepted:false`. Unknown Canonical properties and incomplete observations produce `partial` / `unknown` rather than assuming future schema compatibility. The adapter consumes *already-authorized* Inspection results; access control must be enforced before an application exposes them.

**Important limitation:** `InspectSubject` v0 supports `semantic-data`, `semantic-relation`, `application`, `application-pattern`, `query`, but **not a standalone semantic Capability inspection**. A Pattern's `capability` ID alone cannot reveal its typed input/output/outcome/effect or execution authority. Do not feed this observer's Pattern projection into `comparePair` as an accepted action signature. Extending the existing inspection operation, if justified by a real consumer, belongs to #1165 and the Semantic model owner; do not invent a parallel parser or force-complete Source-based fallback.

The first Reference test inputs are synthetic `SubjectInspection` JSON envelopes shaped according to the actual Go structs; **no actual `tei inspect` process/Go World load was executed for these tests**. Execute in a full repository checkout:

```sh
node --test reference/local-cpu-reuse/project-inspection-v0.test.mjs
node --test reference/local-cpu-reuse/project-application-membership.test.mjs
```

The **16 + 7 = 23** cases passed in an isolated JavaScript evaluator using files fetched back from GitHub (not a Node.js checkout, not an `operations.InspectSubject` integration test). Verified behaviors include nested `contractQuery` preservation, model identity, keys/FDs, unknown properties, wrong subject traces, stale snapshots, and explicit Application membership. An actual real-World fixture, provenance/current-revision recheck, authorized cross-application comparison, semantic-effect comparisons, CPU timing and memory metrics remain **NOT_PROVEN**.
