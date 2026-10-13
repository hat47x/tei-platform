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
node --test reference/local-cpu-reuse/project-inspection-source-trace.test.mjs
```

The **16 + 7 + 3 = 26** cases passed in an isolated JavaScript evaluator using files fetched back from GitHub (not a Node.js checkout, not an `operations.InspectSubject` integration test). The added 3 negative/positive cases check that Source Index references point to a unique listed artifact path; this is a **structural consistency check**, not an authenticity or source-permission proof. Verified behaviors include nested `contractQuery` preservation, model identity, keys/FDs, unknown properties, wrong subject traces, stale snapshots, and explicit Application membership. An actual real-World fixture, provenance/current-revision recheck, authorized cross-application comparison, semantic-effect comparisons, CPU timing and memory metrics remain **NOT_PROVEN**.

## Opt-in real TEI CLI inspection path (2026-10-13)

[inspect-cli-bridge.mjs](inspect-cli-bridge.mjs) accepts the **actual CLI envelope** `tei.cli.inspect/v0`: the Go CLI's `ok: true` and `result` containing the `tei.operations.inspect/v0` operation output. The bridge requires an exact subject kind+ID match, rejects error envelopes, malformed or unknown schemas, and rejects changing World or Source Index fingerprints across successive reads. It never converts a CLI failure into an empty feature observation.

[run-real-inspection.mjs](run-real-inspection.mjs) is an **opt-in executable entry point** for the trusted local `tei` binary. It requests exactly five explicitly named subjects from the existing **single** `order_entry` Reference Workspace and emits one aggregate *derived observation* JSON document on stdout. It invokes the binary via Node `execFile` with an argument array (no shell); subprocess failures produce a short error code without copying potentially sensitive stdout/stderr into logs. A 1 MiB buffer and 30-second timeout apply per subject. These are Reference safety bounds, not general product performance requirements.

From the repository root in a trusted checkout, a developer can run:

```sh
go build -o /tmp/tei-local-cpu-probe ./cmd/tei
node reference/local-cpu-reuse/run-real-inspection.mjs \
  --tei-bin /tmp/tei-local-cpu-probe \
  --workspace reference/workspace
```

(The binary path is an illustrative Unix-like temporary path; use an appropriate location for another OS.) This procedure requires access to the Workspace and explicit authorization before any inspection. Do **not** aim this research command at customer workspaces without ensuring their source-disclosure policy is satisfied. The runner does not authenticate, enforce tenant permissions, or validate a trusted binary signature. It does not perform any write or provider invocation, collect training data, or transmit observations over a network.

The five selectors are: `application:order_entry`, `application-pattern:order_entry.order_lines`, `semantic-relation:sales.order.lines`, `semantic-data:sales.order`, and `application-pattern:order_entry.customer_reference`. The runner reports the two declared Application→Pattern and Pattern→Relation links **only as observations** with `accepted:false`; it does not claim that a renderer, Action, or Contract is implementation-compatible. It refuses inconsistent sequential snapshots rather than combining them.

Test commands in a full checkout:

```sh
node --test reference/local-cpu-reuse/inspect-cli-bridge.test.mjs
node --test reference/local-cpu-reuse/run-real-inspection.test.mjs
```

Evidence obtained so far: **12/12 CLI-envelope / batch tests** and **4/4 runner orchestration tests** passed in an isolated JavaScript evaluator with *synthetic* Go-shaped CLI JSON; the runner tests inject a fake read function. The **actual** `go build`, `tei inspect` subprocess execution, Node ESM execution from the Git checkout, and full Repository-health remain **NOT_PROVEN**. This is a runnable verification route, not a claim that the target E2E has been observed.

The next measured step is to execute this exact command at an identified checked-out revision, record the binary's source commit and `tei cli inspect` schema, inspect all five results and their source fingerprints, and keep them separate from an Approval / Acceptance evidence packet. Only after that should a second explicitly authorized Application be considered for cross-Application candidate retrieval.

## E2E execution availability check (2026-10-13)

The working runtime has Go, Node.js and Git, but **does not have a checked-out TEI repository**. An attempt to resolve the remote Git origin failed with `Could not resolve host: github.com`. GitHub connector access permits reading/editing individual files, not a trusted full checkout for building the actual TEI binary. Therefore the previously documented end-to-end command **has not been executed**; do not record a passing real-World inspection, CPU benchmark or Repository-health status.

To make this an actionable gate for a future authorized checkout: first record the checkout commit SHA and `git status --porcelain`, then run the two CLI bridge test files under Node, build the Go CLI, execute the five-subject runner, and attach only sanitized subject IDs, observed source/world fingerprint equality, error codes and elapsed-time measurements. Avoid uploading Canonical source bodies or full CLI JSON containing business data. Recheck the checkout commit SHA after the run and discard evidence if it changed. If any of the five exact subjects is absent, report a bounded `SUBJECT_NOT_FOUND` result rather than fabricating a replacement ID.

**No Go integration, mainline rebase or PR merge is claimed.**

## Evidence output hardening (2026-10-13)

The opt-in CLI runner now writes a **metadata-only evidence summary** by default. Canonical bodies, field bindings, Source artifact paths, and raw World/Source Index fingerprint values are not serialized to stdout. Its in-memory API still returns detailed observations for an explicitly authorized caller; those details must not be logged or exported without separate permission. A new synthetic regression test checks that representative business attribute names and raw fingerprints are absent. This does not constitute a security audit or guarantee that Subject IDs themselves are non-sensitive. Real CLI E2E remains unexecuted.

## Default evidence privacy tightening (2026-10-13)

The default runner stdout omits **Subject IDs**, raw diagnostic strings, Canonical bodies, Source artifact paths, and fingerprint values. It emits only ordinal index, known Subject kind, completeness, diagnostic count, source status and bounded link status. Identifiers are still available through the explicit in-memory API for an authorized caller. The runner's five synthetic orchestration tests passed in an isolated JS evaluator after this change; no real Go CLI E2E was executed. Because kind, order, count and link status can themselves reveal structural information, use only on an authorized workspace and do not assume the summary is anonymous.

## Pairwise candidate search: fail-closed and bounded CPU work (2026-10-13)

`compare-candidates.mjs` is still a **research-only, caller-authorized, read-only** comparator. Its accepted input schema is deliberately closed at the structural comparison boundary: unknown top-level record, Source, Signature, Contract, typed-field, and UI Platform properties raise `CandidateInputError` rather than being silently discarded and misrepresented as equivalent behavior. Known `labels` are retrieval hints only. Effects and failures are treated as **unordered but non-duplicated** sets; duplicate entries are invalid (they are not silently collapsed). A `logic` record carrying unmodeled `platform` metadata is rejected until that constraint has an explicit comparison rule.

`findReuseCandidates(records, {allowedApplications, maxRecords, maxPairs})` has defaults **256 visible records** and **8,192 cross-Application pairs**. Callers can explicitly raise limits to a maximum of **4,096 records** and **65,536 pairs**. It rejects an excessive search *before constructing candidate results*, using an exact count of cross-Application combinations. Same-Application-only pairs do not consume the pair budget. Identical component IDs across Applications are rejected, including direct `comparePair` calls. Sorting uses ordinal string comparison for deterministic results across environments. These bounds are protective defaults for this tiny Reference, **not a production performance requirement imposed on plugin developers**.

The `allowedApplications` filter does **not** authorize data retrieval: permission checking and upstream disclosure control must happen *before* creating or passing records. The caller-provided contract IDs/fingerprints remain **unverified**, and `reuse_candidate` never implies substitution approval.

`compare-candidates.safety.test.mjs` adds 11 tests for duplicate effects/failures, unknown semantic constraints, direct duplicate IDs, invalid limits, record/pair exhaustion, cross-Application pair counting, stable unordered signatures, and excluded Application budget behavior. Together with the existing 14 baseline and 3 regression tests, **28/28 PASS** on Node.js 22.16.0 in a local staging directory. The staged comparator and new test had Git blob SHA matches with the GitHub branch versions. This is a **targeted Node test**, not a complete exact-PR-head checkout, Go build, real two-Application experiment, full CI, or Acceptance qualification.

## Verified redaction boundary for the opt-in CLI summary (2026-10-13)

The CLI runner now uses the dedicated [inspection-evidence-summary.mjs](inspection-evidence-summary.mjs) module. Before emitting a redacted JSON summary it **rechecks** the expected five Reference selectors in their fixed order, the `inspection-batch/v0` and `inspection-features/v0` schema/effect/authority tags, a shared World and Source Index fingerprint across all observations, unverified contract conformance, non-Accepted status, and whitelisted completeness, source-status and link-status values. Only after these checks can `consistentSnapshot:true` appear in a summary. Unexpected subjects, missing rows, snapshot drift, unsupported status strings or claimed Acceptance fail closed with the bounded `INVALID_INSPECTION_EVIDENCE` code. The CLI output never includes business subject IDs, Canonical fields, raw diagnostics, World/Source Index fingerprints or arbitrary link extensions.

This is **structural consistency and output disclosure control**, not cryptographic source proof, CLI binary authenticity, tenant authorization, or Accepted business semantics. The caller must authorize the Workspace *before* the underlying CLI reads it. A malicious producer can still fabricate a self-consistent observation; the summary does not certify the provenance. No cross-Application source scan is added.

`node --test reference/local-cpu-reuse/inspection-evidence-summary.test.mjs`: **7/7 PASS** in a local Node.js 22.16.0 staging directory; both source and test blobs matched the GitHub branch blob SHAs (`6b9ec3b9dfb87655f37c324d57cd3bee65fe873a`, `4b4871cfbad9506e1ecdffe97be8cbe34c7e08a5`). This is a targeted pure-module test, **not** a complete exact-head checkout or actual TEI subprocess run. Existing CLI bridge/runner tests were not re-executed against a full repository checkout in this environment. The local Go toolchain is **Go 1.23.2**, while this branch's `go.mod` requires **Go 1.26.9**; the GitHub repository was not network-clonable here. Real `go build`, Workspace E2E, Source revision truth, and trusted exact-head qualification therefore remain **NOT_PROVEN**. The observed `cmd/tei/main.go` and `cmd/tei/cli_inspect.go` flag and JSON-envelope paths match the runner's requested `inspect -workspace ... -subject-kind ... -subject-id ...` arguments.

## Real Node subprocess transport smoke test (synthetic TEI CLI, 2026-10-13)

The opt-in `run-real-inspection.mjs` entry point is now tested with **actual child processes** (`node:child_process.execFile`) and a **synthetic** `tei.cli.inspect/v0` executable fixture, rather than relying only on injected in-process Reader mocks. The new committed files are `fixtures/synthetic-inspect-cli.cjs`, `inspect-cli-process-success.test.mjs` and `inspect-cli-process-failures.test.mjs`.

On POSIX, the tests copy the fixture to a temporary executable, use an explicitly named workspace path containing a semicolon to exercise shell-free argument transport, and clean up the temporary files. The expected five `inspect -workspace ... -subject-kind ... -subject-id ...` invocations are checked in order; a deliberately altered third World fingerprint must abort the run with no partial stdout. Other cases check nonzero child exit, unparseable stdout, unexpected stderr and output exceeding the 1 MiB `execFile` buffer, with bounded error codes and no private raw child text forwarded. The tests are skipped on Windows because their temporary Node executable fixture assumes POSIX executable permissions; this **does not** mean TEI Desktop/Windows support was tested or rejected.

Run from a trusted checkout:

```sh
node --test reference/local-cpu-reuse/inspect-cli-process-success.test.mjs reference/local-cpu-reuse/inspect-cli-process-failures.test.mjs reference/local-cpu-reuse/inspection-evidence-summary.test.mjs
```

**Local evidence:** Node.js 22.16.0 on Linux, **13/13 PASS** (six newly committed subprocess cases plus seven previously committed summary cases). The staged source modules `run-real-inspection.mjs`, `inspect-cli-bridge.mjs`, `project-inspection-v0.mjs`, `inspection-evidence-summary.mjs`, both new tests and the fixture were verified against their exact GitHub blob SHAs. This was a **targeted staging directory test**, not a full checkout test or CI run. No TEI Go binary was built or executed. Therefore real Workspace Load, `go build`, genuine `tei inspect` end-to-end behavior, two authorized Applications, actual source authenticity, and Acceptance remain **NOT_PROVEN**. The synthetic fixture is not a substitute for the TEI executable and must never be used for Canonical decision-making or production inspection.

## Opt-in genuine Go CLI → Node feature projection integration gate (2026-10-14)

[real-go-inspection.e2e.test.mjs](real-go-inspection.e2e.test.mjs) extends the earlier fake-process transport test with a *genuine Go CLI binary* built from the checkout being tested. It is intentionally **opt-in**; normal Node test discovery reports a skip rather than a synthetic PASS. It never substitutes the previous fake `tei.cli.inspect/v0` emitter for this gate.

From a trusted checkout with the Go version required by `go.mod`, Node 22+ and permission to read the checked-in `reference/workspace`, run:

```sh
TEI_REAL_CLI_E2E=1 node --test reference/local-cpu-reuse/real-go-inspection.e2e.test.mjs
```

The test creates a disposable temporary binary with `go build -o <temp> ./cmd/tei`, with `GOTOOLCHAIN=local` (no transparent Go toolchain upgrade), and then executes the existing `run-real-inspection.mjs` against exactly the five fixed subjects in `reference/workspace`. It checks that the Go→CLI JSON→Node derived observation→redacted summary chain yields the expected subject kinds, shared snapshot, explicit declared membership/relation, `not_asserted` Acceptance and no raw Subject IDs, Canonical bodies, World/Source Index fingerprints, Contract Query or Source paths in stdout. It also calls the *real* binary for a missing Semantic Data subject and checks the structured `SUBJECT_NOT_FOUND` failure envelope. Only a **successful opt-in run** may be cited as real CLI E2E evidence. Its build may download Go module dependencies if not cached; the test itself does not send inspection results over the network.

This does **not** prove trusted source identity, tenant/project authorization, real two-Application comparison, full Contract semantics, performance, or #267 Acceptance. It is one explicitly chosen **single Reference Workspace**, with read-only inspection of multiple subjects. The normal fake executable tests remain useful transport regressions, but are a different evidence tier.

As of this addition the local execution environment has **Go 1.23.2** while the current `go.mod` requires **1.26.9**. Consequently the opt-in real-Go test cannot be qualified locally: new test code is committed for execution in a suitable trusted checkout, **not claimed PASS**. Its default skipped path can be syntax-smoke-tested independently. This is a targeted gate, not a request to run the full repository health suite or to enable TEI-LD training.

## Two actual Reference Applications: shared customer Query, different Application context (2026-10-14)

`order_entry.customer_reference` and `repair_intake.customer_reference` are distinct actual TEI Patterns in the **same checked-in Reference Workspace**. Both declare Database Contract `master_data.customer`, Query `find_by_id`, input binding `customer.customer.customer_id -> id`, target Database `master_data`, read-only access and compatible major `2`. Their Pattern Relations and Application-local datasource aliases differ (`sales.order.customer` / `repair_intake.intake.customer`, and `master` / `customer_master`). Their names and common contract do **not** prove UI-implementation or business-rule equivalence.

The small read-only [compare-two-application-references.mjs](compare-two-application-references.mjs) consumes exactly four caller-authorized `tei.cli.inspect/v0` reports (Application and customer-reference Pattern from each). It uses the existing typed inspection projection to verify exact subjects, same World/Source Index snapshot, indexed Source, completeness and declared Application→Pattern membership. It additionally checks the original typed Go Application database routing/required-major metadata and exact Reference Pattern contract-query bindings. Any unknown or unsupported semantics, non-read datasource, missing/ambiguous contract route, incomplete Source, or snapshot drift **abstains or fails closed**, never implies direct reuse. This is an intentionally fixed public Reference case, not a general cross-tenant query API or second TEI parser.

Its output `shared_query_invocation_declared` means the two **declarations** target an identical Query invocation under the observed World. The result **always** reports `reuseDecision:needs_semantic_review`, `acceptance:not_asserted` and `contractConformance:not_verified`. It does not feed raw CLI metadata into the synthetic `comparePair` as a verified contract fingerprint. The in-memory comparison contains publicly checked-in identifiers and Source snapshot hashes, so it must **not** be automatically printed or exported for customer workspaces.

Targeted tests:

```sh
node --test reference/local-cpu-reuse/compare-two-application-references.test.mjs
TEI_REAL_CLI_E2E=1 node --test reference/local-cpu-reuse/real-go-inspection.e2e.test.mjs
```

The first suite uses synthetic `inspect/v0` JSON to stress changed Query, compatible major, binding, datasource, unknown policy, membership and snapshot drift. The opt-in E2E builds the genuine `tei` Go CLI and inspects the **four actual Subjects**, then checks shared Query metadata and distinct Application context. Neither exercise establishes an Accepted Contract implementation, renderer interoperability, real project-to-project authorization, transitive license rights or the cost/benefit of component extraction. Both Applications are in the **same Reference Workspace**, not independent production customers. Real two-project Source trust, review time, CPU cost and maintainability gain remain future research gates.

## Read-only Query-invocation adapter boundary proposal (2026-10-14)

After the **real Go** two-Application observation qualified in the previous PR, `proposeReferenceQueryAdapter(reports)` extends the narrow fixed-Reference comparison into an **in-memory design candidate**, not an executable adapter. It reuses `compareTwoReferenceApplications` and therefore preserves the original exact Subject, read-only Source Index, snapshot, Application membership, route, Contract requirement, Query and input-binding checks. There is no new DSL parser, new authorization system, generated runtime code, read query execution, or modification of TEI Go Canonical types.

Its `tei.research.query-invocation-adapter-proposal/v0` response deliberately separates:

| Layer | Fields | Meaning |
| --- | --- | --- |
| Shared *declaration candidate* | `databaseTarget`, `contract`, `compatibleMajor`, `query`, `bindings` | The two public Reference Applications explicitly declared the same read-Query invocation. No Contract implementation or Query output signature was verified. |
| Application wiring | `application`, `pattern`, `relation`, `datasourceAlias` for each Application | Relation and alias remain owned by each Application. A future adapter must receive these through Application-specific binding; it cannot infer or overwrite them. |
| Unresolved gates | `contract_semantics`, `query_result_signature`, `caller_authorization`, `runtime_binding`, `acceptance` | Explicit omissions, **not** automatically satisfied by identifier or signature coincidence. |

Status is `candidate_requires_semantic_review` only when the two *observed declarations* match. Contract/query/input drift leads to `not_shareable_from_declarations` and `commonInvocation:null`; malformed, incomplete, unindexed or out-of-snapshot Canonical observations still fail closed. Even in the matching case `executionPermitted:false`, `implementationSelected:false`, `accepted:false`, `acceptance:not_asserted` and all semantic/authorization verifier fields remain false. It does **not** call the existing `comparePair` with a fabricated Contract fingerprint or assign semantic authority.

The intended eventual shape is **Application binding → parameterized shared Query-call logic → already-authorized Database Contract runtime**. This is an architectural decomposition only; runtime provider APIs, mandatory security requirements, query result shape, UI rendering, identity mapping and caching remain for concrete requirements and qualification. In particular, the Application-local `datasourceAlias` must never become a cross-Application routing instruction, and a compatible-major declaration must not be treated as verified runtime compatibility.

Both fixed cases live in the checked-in `reference/workspace`. The returned in-memory object contains domain identifiers and an observation snapshot; it is *not* a general JSON export format. Disclosure and authorization remain the caller's duty before `tei inspect`, including for future customer Workspaces.

Focused tests add nine new cases to the existing 15-case suite, for declaration/context split, differing Query/major/input, unresolved alias, unexpected write access, snapshot/unknown semantic policy rejection, and detached output copies. The opt-in [real-go-inspection.e2e.test.mjs](real-go-inspection.e2e.test.mjs) also asserts the boundary against the actual Go CLI's four Reference observations:

```sh
node --test reference/local-cpu-reuse/compare-two-application-references.test.mjs
TEI_REAL_CLI_E2E=1 node --test reference/local-cpu-reuse/real-go-inspection.e2e.test.mjs
```

Execution status belongs to the *corresponding exact-head CI result*; source changes alone do not count as a pass. No #843 training, #876 automatic qualification, or #267 Acceptance is initiated.

## M0 Database Contract Query typed-signature observation (2026-10-14)

The checked-in Go-normalized `reference/workspace` includes Contract `master_data.customer` **v2.4** with `find_by_id(input id: customer.customer_id) → customer.customer`, result cardinality **`zero-or-one`**. Both `order_entry` and `repair_intake` require compatible major `2`, bind `customer.customer.customer_id -> id`, and use read-only, distinct Application-local datasource aliases. The typed input, output and result cardinality come from the **owning Database Contract**: do not infer them from identical Pattern names or customer-facing UI.

The research-only [qualify-query-signature.mjs](qualify-query-signature.mjs) consumes the four authorized `tei.cli.inspect/v0` reports and the normalized `tei world -workspace reference/workspace` JSON. It first reuses the existing four-subject application comparison (exact identities, Source/World Index fingerprints within inspect, declared read-only route, membership and matching Query invocation). It checks both Application and Pattern Canonical JSON **against the independently read normalized World** before looking up exactly one Database, Contract and Query. It checks Contract major, complete typed Query input binding, resolves the bound Semantic attribute to its declared type, and verifies the result Semantic Data identity and the declared result cardinality. It rejects unknown fields at the Database/Contract/Query-signature boundary, ambiguous identities, mismatches and incomplete declarations rather than guessing.

Its `declared_signature_matched` outcome means **all of these declarations agree** for this fixed Reference input, and records input `customer.customer_id`, result Semantic Data `customer.customer` and `zero-or-one` cardinality. It does **not** mean the external-observed `master_data` provider is live, returns that shape, implements the expected Query, or allows a real caller to access customer data. The independent `world` and `inspect` reads are not an atomic transaction: matching the two Application subtrees reduces observable drift but **cannot prove unchanged Database Contract data, Source authenticity or a consistent revision**. Therefore `verified.providerImplementation`, `resultRuntimeShape`, `authorization`, `sourceAuthenticity` and `atomicWorkspaceSnapshot` remain false; `executionPermitted:false`, `implementationSelected:false` and `accepted:false`.

The `tei world` command returns an entire normalized World, potentially containing sensitive material. This fixed public Reference E2E parses it **in memory only** and does not log or export it. Never use the research adapter as a broad tenant data export or bypass authorized Workspace disclosure controls. Neither the Reference result nor a compatible-major declaration establishes an accepted common runtime plugin.

Focused qualification commands:

```sh
node --test reference/local-cpu-reuse/qualify-query-signature.test.mjs
TEI_REAL_CLI_E2E=1 node --test reference/local-cpu-reuse/real-go-inspection.e2e.test.mjs
```

The test suite covers declared v2.4 input, type and cardinality, Contract version drift, missing/extra inputs, invalid result shape, duplicate identities, unsupported policy and mismatches between the World and exact CLI observations. CI evidence must be recorded for the *exact PR head*, not inferred from previous successful versions. No Go Core, Canonical model, production runtime, or TEI-LD training is changed.
