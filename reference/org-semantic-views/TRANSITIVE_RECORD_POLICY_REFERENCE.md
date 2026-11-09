# Organization-linked record policy — synthetic-only pressure

This is a **negative/positive synthetic contract fixture**, not an authentication/authorization provider, runtime row-level security (RLS), TEI-specific Host, tenant isolation implementation, nor an adopted Organization Policy DSL. Run `node --test reference/org-semantic-views/verify-transitive-record-policy.mjs`.

## Intended product behavior

A user can opt in to **one centrally visible default policy for specified Data/record classes**, binding an approved organization axis and a typed record-to-OrgUnit relation path. The Organization Data itself stays ordinary Data, unless a particular field or record has an independent protection requirement. Selected exceptions may be made by specific resource, Function, Rule or purpose — but an exception to *policy applicability* must **not be interpreted as an allow from the Host**.

Potential outcomes must be separated:
- **Applicable / not applicable:** Does this configured Organization rule apply to the record and consumer?
- **Scope result:** In-scope, out-of-scope, or unverified (partial relation/read/unknown path, stale source/policy epoch).
- **Candidate behavior:** Display eligible, explicit denial, or read-only (no update candidate). A candidate is not current Host permission.
- **Actual authority / enforcement / disclosure:** Existing Host/Provider must enforce every permitted route, including Query, pagination/count, joins, exports, batch, AI/SHO, cache and update. Even a matching synthetic Grant in this fixture is **not real authorization or Effect**.

A record excluded from an Organization-based filter may still be blocked by other applicable rules, the Provider or the Host. An exclusion is **not** equivalent to `public`, `allow`, or `bypass`. A deliberate public-access override would require an independent, explicitly accepted policy and independent Host authorization.

## Graph/identity/temporal boundaries

The toy graph uses tenants T1/T2, a command hierarchy H→D1→L1 and D1→L2, and a second H→S→L2 path, plus a separate D1→B **budget** link. Multi-path reachability is not a license to duplicate the target record; a strict single-parent hierarchy may reject the same graph. Cycles are diagnosed; depth cutoff, unknown nodes, partial Graph reads and unknown relation sets do not prove absence.

A later `knownAt=2027-07-03` correction changes validity of D1→L2 at `validAt=2027-06-15`. This is a synthetic bitemporal cut in JavaScript, **not** an implemented `.tei` temporal evaluator. The same L2 remains accessible along H→S in the relevant cut for the H anchor, demonstrating that **all paths, selected axis and root scope matter**. Cross-tenant record links are rejected as unsafe anomalies; the chosen test does not authorize exceptional cross-tenant access.

For multi-org records, the `any` vs `all` rule is an **explicit test-input choice**. No global TEI default is adopted. A positive-path witness can justify a restricted local eligibility candidate under a stated `any` rule despite incomplete collection, while a negative/universal claim requires the relevant collection closure. A policy-defined strict single-parent tree is different from the structurally permitted DAG; the selected business axis determines whether multi-parent graph edges are valid.

## Cases (VP-01..33)

- Structural: typed command vs budget edges, reachable Nodes vs Path occurrences, multiple parents, cycle, depth cutoffs, tenant barriers.
- Applicability and exceptions: default on for **only explicitly targeted** order records, metadata unaffected, per-record read-only, exclude-from-this-policy, explicit deny. Multiple applicable rules cannot silently be overridden by one opt-out.
- Time and freshness: `validAt` vs `knownAt`, source-graph revision, policy revision, original/current Host grant epoch and principal, plus a separate binding of Host grant scope-root/axis/issued-policy-version. A client/AI cannot widen D1 to H, switch command→budget, or reuse an old policy revision merely because the same principal has a synthetically matching action.
- Negative/partial: absent/unknown record→Org links; partial hierarchy; `any` vs `all` with unknown extent; no grant inferred from Organization Membership/Directory Group or structural reachability.
- Consumer/effect: no automatic external disclosure, export, current Host Grant or Effect; frontend post-filtering cannot attest to Provider-enforced page/count visibility.

**Local evidence at creation:** 33/33 tests passed through V8-compatible shim with Node test imports replaced. The exact-head CI's native Node result is a **separate fact** to check after the PR is created. Even if Node CI passes, the scope is closed synthetic Reference E0 only: no actual Query/Function Data Context/Provider RLS, real host/grant, sensitive user data, Studio/SHO, actual AI recipient or Owner acceptance.

## Existing Owner boundaries / required proof

- **Organization meaning and policy applicability**: #2386 and §58 of [org design Draft #2389](https://github.com/hat47x/tei-platform/pull/2389); Profile negative corpus #2510.
- **Configurable Plugin + AI fitting**: #951, #2619, and §57; actual identity/Directory/SCIM authority does not follow from role naming.
- **Semantic and physical paths**: #1126 Data/Relation identity; #2506 Function explicit Mapping; #2387 Query negative dependencies; #2622 Data Ops; #2628 actual Provider access/witness.
- **Consumer and runtime**: #2582 change requalification; #2516 Disclosure to SHО/AI; #2631 Studio projection; #2598 typed Intent, #1174 actual Effect, #1119 current product gates.

**Before any production claim**:
1. Domain Owner explicitly accepts scope/purpose, inheritance axis, multiple-parent and cross-tenant treatment, rule priority, per-record exclusion semantics, and read-only vs hidden behavior.
2. Existing Host/Provider's **real** request and current grant checks enforce the chosen record scope before pagination/filtering/count/export; verify both positive and denied cases with actual DB/API, not HTML-only post-filtering.
3. Verify backdoors across joined, aggregated, cached, exported, batch, search, AI/SHO and write paths, including disclosure of presence/row count and unknown values. No silent fallback if enforcement cannot be proven.
4. Test actual org revision, delayed responses, Role Pin, Read Cohort, active Identity/Grant, and stale cache behavior. Keep publication and historical as-of queries separate from current permission.

No new Core Organization kind, global ACL engine, duplicate Host, policy registry, or mandatory annotation burden is proposed. A shared `Rule/Policy binding` may be expressed with existing Data/Relation/Plugin contracts, assisted by Studio/AI, while the existing Host or specialized Provider retains authoritative enforcement.

## Visible-state requalification (VE-01..20)

The companion `verify-visibility-epoch.mjs` is a 20-case read-only state characterization, not an actual Studio/Browser/Host test. Its independent toy Context binds source identity **including locator**, accepted meaning, policy revision, hierarchy axis/root, valid/known cut, principal/tenant, current grant epoch, read cohort, Query non-output dependency footprint, selection epoch and session epoch. A response is accepted to visible state **only after** current-context and current permit checks; a successful prior Provider read is insufficient if either has changed while a request was in flight.

- VE-01/02: accepted list candidates and **revoked visible/cache** after policy revision. A previously lawfully downloaded/exported copy cannot be retroactively erased; future access and controlled application surfaces are different from external retention.
- VE-03..08: delayed reply, revocation, selection A→B→A, scope root D1→H, hierarchy axis change and policy-version change require requalification even when row IDs remain the same.
- VE-09..13: unavailable/partial results are not empty complete; list/read is not export; a `count` route must be explicitly qualified, and filtering a previously paged/unrestricted set in the browser cannot attest policy-enforced total counts.
- VE-14..15: preserve an unsaved edit as a **quarantined** draft when its read/operation grant lapses, not as a reusable submit intent; purely cosmetic style changes can retain still-qualified data.
- VE-16..20: identical bytes at a different source locator, Query hidden predicate revision, Provider cohort, tenant mismatch and previously exported copies are distinct invalidation/disclosure conditions.

A `synthetic-provider-observed` or toy permit object in the tests is **test data**, not evidence from a real Provider, IdP, Host, Studio or external Consumer. These tests make no claim about revocation of screenshots, documents already received by a third party, browser backups, physical caches or secrets already disclosed.

**Next real proof** (existing Owners #2582/#2506/#2516/#2631/#2598/#1174): one actual TEI read→Context→UI/consumer on current Source/Policy/Role/Grant, then a revoked/scope-changed read with an old response delivered late, and read/write/export/count attempts. Observe actual request and response bytes, cache/DOM/a11y, diagnostic and external-channel disclosures, while maintaining non-sensitive delayed edits in a quarantined state. Missing real execution must be recorded as `NOT RUN`, not inferred from Node results.

At creation-time, VE-01..20 ran 20/20 in a local V8-compatible test shim; exact-head native Node CI is a separate verdict.