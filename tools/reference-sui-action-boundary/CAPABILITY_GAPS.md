# SUI non-generic application pressure-test: remaining capability matrix

Authority: repository-local ISSUE-000052. Execution work: GitHub #993. This is a gap register, not a declaration of implementation completion.

| Priority | Capability | Existing evidence | Remaining proof / acceptance |
| --- | --- | --- | --- |
| P0 | Native UI/interaction host | first-web-target custom HTML form; reference actionboundary | Real SUI React Canvas drop -> Application-scoped Action; preview/cancel no write; browser without required external process |
| P0 | Runtime action authorization | first-web-target authentication/authorization through driver.Resolver; reference Authorizer interface | Use actual subject/resource runtime authority; denial before mutation; cross-binding rejection |
| P0 | Application-owned state commit | reference Store interface with expected revision | Atomic compare-and-write; stale conflict; unknown Card; persistence and DocumentV1 reload; unrelated fields retained |
| P0 | Domain constraint extension | Stage 0 hierarchy M1-M4, semantic evaluator reference | Connect extension to ordinary Application validation; optional parent, parent existence, acyclic; do not add SUI-specific Core enum |
| P0 | Extensible Application intent | fixed browse/record/reference/command/header-detail patterns | Test command/Capability reuse first; show concrete failure before introducing extensible Pattern |
| P1 | Review and domain transaction | SUI merge suggestion and explicit human decision/apply | Distinguish preview, reviewer choice, revalidation, apply, conflict, rollback, undo; no design-time acceptance substitution |
| P1 | Provenance and cognition | SUI native algorithms/providers | AI produces candidates only; retain provenance, confirmation scope, reviewer authority, reproducibility |
| P1 | Offline artifacts | SUI DocumentV1 import/export | Preserve native canonicalization and artifact compatibility; prove roundtrip and failure isolation |
| P2 | Cross-target plugin packaging | plugin and web target references | Browser/WASM portability, dependency resolution, resource budgets, second consumer beyond mock |
| P2 | Product value | Stage 0 read-only evidence | Compare native vs TEI-integrated edits, duplicate semantics, manual synchronization, runtime latency |

## SUI native Action receiver: authority handoff gates (2026-10-11)

The SUI adapter is an **application-owned backend mutation endpoint**, not proof that the generic Go Host is on the browser-to-storage path. The native route is disabled unless `sui_native_action_v1_enabled` is explicitly true. Before enabling, satisfy all of the following at the same exact checked-out revisions:

1. **Trusted Subject** — TEI `CapabilityAuthorizer.Subjects` resolves a live authenticated subject from a verified runtime session, never an Action body field. The SUI route independently resolves the effective caller and tenant using its existing `_authorize_request` and tenant-session preconditions. Do not synthesize or forward browser-supplied `x-actor-ref`/`tenantId` as a privileged delegation. If authenticated caller identity cannot survive the transport boundary, fail closed; do not silently substitute a host service identity as the human caller.
2. **Consistent Scope** — both authorization decisions refer to the same Application `sui`, real Document/resource, requested action `sui.move`, authenticated tenant, and current permission. Neither read-only nor agent/low-trust sessions can gain write permission from a syntactically valid envelope. Verify wrong-tenant, wrong-resource, wrong-Application, changed-session and revoked-permission negatives.
3. **Exactly one authoritative commit** — TEI Go Host may validate Action binding and capability; SUI retains Card/Island computation, DocumentV1 validation, revision CAS, and SQL persistence. Do not independently commit a shadow TEI Document. Expected revision is SUI's strongly quoted SHA-256 ETag when using SUI Application's HTTP profile; TEI generic `Intent` remains opaque.
4. **Acknowledgment and reconciliation** — dispatch exactly once, then read the authoritative SUI Document with no-store and matching revision. A lost acknowledgment may mean the write succeeded: never automatically retry. Local React state must atomically validate previous Document reference, tenant session, ETag, Undo history and dirty status before declaring confirmed; it may not infer success from an asynchronous `setState` call.
5. **Provenance** — `reviewAttribution` is an immutable prior signed fact under Card move. A *new* human-review declaration via PUT must match the authenticated reviewer, but an authorized Card mover need not impersonate the earlier reviewer. Moving the Card cannot modify review attribution, critique inputs, reproposal history, Source, Hold or Edge type. SUI's Action route checks this without adding SUI-native concepts to TEI Core.
6. **Auditing and deployment** — SUI emits an application-owned successful `apply` audit after DB commit; its existing audit transport is best-effort and **not** a durable outbox. A failed audit delivery cannot turn a committed Action into a false HTTP failure. Meet any mandatory durable audit requirement separately. Keep feature gate off until real full-tree tests, trusted delegation and end-to-end browser tests pass.

**Evidence still missing:** actual TEI Go Host -> SUI backend transport with verified caller propagation; SQL CAS under concurrent live requests; real React Undo/dirty reconciliation; trusted BFF session/CSRF E2E; full regression on exact commit SHA. A manually added test is not a PASS.

## Scenario gates

- **S1 hierarchy**: M1/M2 pass; M3 missing parent and M4 cycle fail with explicit evidence; distinguish evaluator reference from production validation.
- **S2 projection**: Native Canvas renders read-only from SUI DocumentV1 without creating a competing document authority.
- **S3 interaction**: preview and cancel produce zero writes; drop produces one authorized action; stale revision, missing Card and invalid payload produce zero writes; reload preserves identity, position, Source/Reviewer/Hold, unknown Edge kinds and other unrelated fields.
- **S4 review**: candidate -> human decision -> revalidate -> apply; stale and unauthorized requests fail without mutation.
- **S5 cognition**: provider output remains a proposal; no implicit document commit; provenance and confirmation preserved.

## Evidence template

For each gate record: repository commit SHA, exact reproduction command, test and browser observations (including failures), fixture, native vs TEI responsibility, changed files, duplicated definitions, manual synchronization steps, performance and target differences. A test added but not executed is **not PASS**.

## Architecture constraints

- Do not move SUI-specific Card, Island, polygon, or KJ algorithms into TEI Core.
- Keep DocumentV1 as the authoritative instance document during incremental adoption.
- Semantic extensions, DSL syntax/parser/validation, UI host, and plugin packaging must be extensible without forcing all native logic into a single DSL.
- Test a second real consumer before promoting a reference adapter to a standard plugin.
- Avoid duplicate GitHub issues; #993 is the implementation entry point for the native interaction boundary.
