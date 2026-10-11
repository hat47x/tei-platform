# TEI public source selection — value-first assessment (2026-10-11)

**Status: PROVISIONAL / NOT APPROVED FOR PUBLICATION.** This is a file-by-file path inventory and a user-value assessment, not a secret scan or implementation qualification.

- Examined exact private main revision: `3620f0dd3dd677fb1604c99809dde2c729e98b3d`
- Files in recursive Git tree: **1,705 blobs**, `truncated=false`
- Explicit provisional selection: **684 files** (40%); deferred: **1,021 files** (60%)
- Content-size indication: selected ≈4.36 MB vs deferred ≈10.62 MB. This is only the sum of Git tree file sizes, not a clone or download measurement.
- Machine-readable exact path list: [`proposed-user-value-public-paths.txt`](proposed-user-value-public-paths.txt)
- Intended publication flow: [`prepare-candidate.sh`](prepare-candidate.sh) then [`verify-candidate.sh`](verify-candidate.sh), after mandatory source checkout, security review, and link closure. This list has **not been certified by a human reviewer**.

## User-value selection

| Scope | Selected | Decision |
| --- | ---: | --- |
| `cmd/`, `internal/`, `examples/` | 444 | Include first-party executable CLI, internal implementation and examples with tests |
| `product/`, `requirements/` | 7 | Include primary product goals and requirements |
| `docs/` | 44 | Reader onboarding, architecture, conformance, current explanations and most specification pages; no compatibility redirects/audits |
| `plan/` | 47 | Include 32 canonical ADR files and 15 crosscutting planning/evidence entrypoints; defer other working notes |
| `reference/` | 131 | Include CLI validation workspace, selected project/application examples, golden data and documented input fixtures; defer auxiliary/third-party transfer areas |
| `tools/` | 4 | Include only `repository-health-check.sh`, two `reference-golden` files and the process plugin example |
| Root files | 7 | README, MIT LICENSE, Go module and sums, contributor instructions, AI development guidance and ignore rules |
| **Total** | **684** | Proposed output set, subject to all gates |

The reduced set defers: 500 `tools/` files, 314 `plan/` files, 126 `bootstrap/` files, 39 `docs/` files, 34 `reference/` files, 4 `research/` files, 2 `skills/` files, the inactive private workflow and the duplicate `CLAUDE.md` instructions. **Deferred does not mean worthless or legally ineligible.** Many are diagnostics, supporting integration tests, scenario artifacts or historical evidence with real maintainer value; some may be needed for full build, verification, license provenance or accurate published claims.

Examples of lower user-facing value *for an initial distribution*: mass `bootstrap/final-source/` samples, one-time R7B merge/qualification scripts, internal issue handoff/audit timelines, deprecated-location Markdown guides, raw third-party reference transfer artifacts without confirmed licensing. Preserve these inside the Private repository unless a separate internal cleanup is authorized and validated.

Do **not** strip `internal/` or its tests simply because the Go package name says “internal”. These contain the actual TEI implementation. Do **not** publish an untested minimal tree as if it were the complete project.

## Actual decision on the pinned source

**NO-GO** for this provisional selection. The current source tree was read from GitHub and the proposed paths were checked against its exact tree. All 684 selected paths exist, but the selected set is **not closed under document links**.

Concrete first-level examples:

- `README.md` links to `plan/product-roadmap.html` and `plan/issue/000040-semantic-component-composition.md`, which are absent from this provisional list.
- `docs/status.md` links to `tools/reference-shared-acceptance/README.md`, several `plan/issue/` items, and current business-procedure runbooks absent from this list.
- Other selected Markdown, HTML, CLI and test references have **not** been exhaustively checked.

The latest inspected main README **does contain** the required generated-AI development / no systematic human review notice. This specific blocker is therefore resolved **at the pinned revision**, but does not make the export ready.

Further blockers:
1. Trusted **complete** checkout of the selected revision is not available in this chat execution container. The private GitHub data was inspected via the connected repository API, but the actual candidate scripts could not be run on the full TEI source. No actual source archive/candidate is created, and no Git history or visibility is modified.
2. Run the `select-public-docs.py` link/redirect checks and `prepare-candidate.sh` + `verify-candidate.sh` on an actual Git checkout; do not infer their result from GitHub tree scans or synthetic tests.
3. Resolve links by retaining genuinely valuable target docs or editing reader-facing source in a qualified, independent change. Avoid automatically re-adding every historical Issue and audit simply to satisfy references.
4. Test the **exact selected file set**: `go build ./cmd/tei`, `go test ./...`, `go run ./cmd/tei validate -workspace reference/workspace`, and full Repository Health. If omitted tests, fixtures, tools or bootstrap assets are necessary, revisit their classification. A successful `go test` on a reduced Go package set does not cover all unselected tests.
5. Review all selected bytes for secrets, personal/confidential information, third-party licensing, copyright attribution and appropriate disclosure. Reference Model #941 remains an independent hold gate.
6. Agree whether this is an *initial source distribution* or an *active Public development repository*. Developer-facing tests, tooling and exact-head qualification may require a larger profile. Transfer PR/Issue SSOT separately.

## Workflow and review rules

- Treat `proposed-user-value-public-paths.txt` as an **assessment artifact**, not a permissions grant.
- A future export uses a fully qualified source SHA and an explicitly versioned selected manifest.
- Never generate a new Public commit until paths, bytes, all local references and legal/security checks are independently accepted.
- Every new publication iteration recalculates the source tree, changed paths, file digests and the impact of omitted tools on tests.
- Keep the Private version for historical evidence and work-in-progress; do not rewrite existing Private Git refs on the basis of this assessment.

Tracking: #1197 (public documentation selection), #1190 (repository model), #1188 (full surface audit), #1189 (CI), #941 (reference material).
