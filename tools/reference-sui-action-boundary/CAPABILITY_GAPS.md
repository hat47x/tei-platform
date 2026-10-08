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
