# Organization semantic view: isolated, read-only reference (2026-11-02)

**Status:** proposal/reference. This folder uses synthetic data only. It is neither an accepted TEI semantic contract nor a production Query/Function/UI/Host implementation.

## Run

```sh
node reference/org-semantic-views/verify.mjs
node --test reference/org-semantic-views/verify-change.mjs
node --test reference/org-semantic-views/verify-handoff.mjs
node --test reference/org-semantic-views/verify-projection.mjs
node --test reference/org-semantic-views/verify-context.mjs
node --test reference/org-semantic-views/verify-acquisition.mjs
node --test reference/org-semantic-views/verify-rule-footprint.mjs
node --test reference/org-semantic-views/verify-query-dependencies.mjs
node --test reference/org-semantic-views/verify-temporal.mjs
node --test reference/org-semantic-views/verify-capacity-scenarios.mjs
```


**2026-11-13 allocation-scale counterexample (ORG-CAP-UNIT-01/02 and ORG-CAP-07b):** the existing `basisPoints` field is a historical fixture identifier, **not financial basis points**. Its accepted meaning *within this synthetic test alone* is `fte-hundredths`: 100 raw units = 1.0 FTE. In finance 100 basis points denote 1%, not a whole ratio ([Investor.gov](https://www.investor.gov/introduction-investing/investing-basics/glossary/basis-point)). A generic Quantity/SQL/Studio/SHO consumer must not infer its scale from the field name; scale/unit/basis is a separate mapping decision owned by #1705/#2495/#2593. Tests also reject duplicated Allocation occurrence instead of last-write-wins and demonstrate that the known-time correction affects the synthetic D1/global FTE, but not D2 FTE under this *complete fixture scope*. Same value is **not** evidence of the same actual Provider cohort. No source JSON or publicly accepted Core Unit is changed. Test scope: synthetic read-only, **not** real Unit/Provider/Host validation.

**2026-11-12 capacity/scenario pressure (ORG-CAP-01..05/07):** `verify-capacity-scenarios.mjs` reuses the existing `org-synthetic-r7` files with exact baseline `validAt=2027-06-15`, `knownAt=2027-07-01`. P1/A1=100, P2/A2=60, P2/A3=40, P3/A4=100 basis points yield 3 unique people, 4 Membership occurrences, D1=160, D2=140, total=300 (3.0 FTE under this fixture's unit convention). Seven Node subtests verify uniqueness vs membership count, JOIN double counting, a read-only P1 absence assumption, a P2 allocation-shift assumption, incomplete/unknown inputs, retrospective knownAt change, invalid occurrence/basis and lack of fixture mutation. Literal Expected is in the test file. **This is a synthetic closed-world calculation only; it does not validate actual database rowsets, live employee availability, grant/qualification, Case/Directive, TEI Product Query, Studio/SHO/Harness or Owner acceptance.**

**2026-11-08 temporal pressure (ORG-TM-01..08):** `verify-temporal.mjs` reuses `source-fixture.json` and `source-fixture.mjs`, with sixteen Node subtests for concurrent Memberships, retroactive correction, half-open boundaries, Position overlaps, source-version ambiguity, incomplete period reads, historic known-time cuts, equal-label/different-cohort snapshots, and the separate historical-knowledge claims of an upstream source versus TEI's actual observation (ORG-KH-01..04). The single-/multi-incumbent choices are test-only alternatives: none is a TEI-wide business rule. Independent literal expectations are fixture-local; this does not test Native TEI temporal query, actual Provider/DB, current Tenant/Host Grant, or Owner meaning acceptance. Exact-head CI must be checked separately.

The test files use JavaScript and Node built-ins only. `source-fixture.mjs` loads plain JSON from this directory; no database, external service, or business record is changed.

**Latest observed native Node evidence (2026-11-02, fixed candidate commit `7eaf1afca3a18a42eb7deb090b6a8be9150c8bca`):** [GitHub Actions run #41462231920](https://github.com/hat47x/tei-platform/actions/runs/41462231920), `synthetic-node` job #125333927789 **SUCCESS**. The job's actual log shows pinned Node **v22.16.0**, exact target SHA assertion, clean pre/post, fixture and all test blob logs (new Query test blob `8e08f5f5f301303b9f9435e5a43eb218021b8559`), `verify.mjs` **27/27 PASS**, and native `node --test` **139/139 PASS, 0 FAIL**, for **166 synthetic checks** at that exact commit. The run is not a formal TEI Productの検証基準に基づく適合確認. **A subsequent file edit (including this evidence note) changes HEAD and needs its own run before a current-HEAD PASS claim.**

**Verified native Node evidence (2026-11-02, fixed historical candidate commit `a01a2addc680bc33c5ceee782e8b48e3a77a736f`):** [GitHub Actions run #41456286651](https://github.com/hat47x/tei-platform/actions/runs/41456286651), job `synthetic-node` #125313679792, **SUCCESS**. Job logs show Node **v22.16.0**, exact candidate HEAD assertion, clean pre/post, `verify.mjs` **27/27 PASS**, and `node --test` **114/114 PASS, 0 FAIL** across the six test files. Total **141 *synthetic checks***, not 141 TEI product conformance tests. The GitHub blob logged for the new Rule test was `69f8c6d40b7cea7335bdb417ac1535f177bc63ff`. **The evidence belongs to the exact commit above; any subsequent document/fixture/code commit requires a fresh run before claiming current HEAD is qualified.** Current actual TEI Query/Context/Rule/Studio/Provider/Host, egress and Owner meaning acceptance are NOT RUN. V8 compatible local evaluation also returned **141/141** on the same fixture version.

## One synthetic dataset, distinct observation scenario and five independent consumers

- **`source-fixture.json`** is the single **test-only semantic dataset**: Persons P1/P2/P3, Organizations D1/D2, time/version-qualified Membership A1-A4 (five version records for four stable relations), Position M1, Occupancy O1/O2, hypothetical FTE allocations, and D1 Projects J1/J2.
- **`scenario-fixture.json`** contains the scenario's `sourceRevisionRef`, `validAt=2027-06-15`, `knownAt=2027-07-01`, read-selection/Context evidence and one explicit example of Tree display preference. The scenario is **not** the model. Its `sourceRevisionRef` must match the source model revision; otherwise the loader stops.
- **`source-fixture.mjs`** reads both JSON files and resolves membership versions under **valid and recorded/known time**. Each of the five test files consumes the same dataset, directly or through the helper. Model content and a selected UI/Provider observation are separate objects; UI selection/epoch does not become a Canonical Semantic fact.
- **`verify.mjs`** checks identity, membership occurrences, FTE, join fanout, Occupancy and candidate scope.
- **`verify-change.mjs`** checks read/change dependency, valid/recorded time, correction, incomplete/non-monotone claims.
- **`verify-handoff.mjs`** checks relation occurrence target, selection/Request evidence, Instance and Save Intent boundaries.
- **`verify-projection.mjs`** checks Table/Graph/Tree/Chart from the shared baseline slice and read-only selection coherence.
- **`verify-context.mjs`** checks explicit Function Context mapping, scope, requested vs observed completeness, revision binding, two read-only Consumer projections, and missing/unknown.

This change **eliminates the prior four independent handwritten copies of the synthetic source**; it does **not** create a production TEI Canonical Source or implement the TEI parser, Data Model DSL, normalized IR, Context/Provider, Renderer, Source Acceptance, or shared product Scenario registry. The JSON shapes are provisional Reference data, not a newly approved Core schema. All business meaning remains hypothetical until accepted by the appropriate Owner.

## Fixed synthetic inputs and independent expectations

- Persons P1/P2/P3; departments D1/D2.
- Directly declared Membership occurrences A1=P1→D1, A2=P2→D1, A3=P2→D2, A4=P3→D2. **A2 and A3 share the Person subject but are distinct relation occurrences.**
- Position M1 belongs to D1. O1=P2 occupies M1 on `[2027-04-01,2027-07-01)`; O2=P3 on `[2027-07-01,∞)`.
- Independently declared hypothetical FTE allocations in hundredths: A1=100, A2=60, A3=40, A4=100. FTE is a synthetic **allocation rule**, not a general consequence of Membership.
- Two projects J1/J2 are associated with D1 to create an observable JOIN fanout.

| Claim and grain | Expected | Invalid inference to detect |
| --- | --- | --- |
| Distinct Persons | **3** | Four memberships imply four people |
| Membership occurrences | **4** | Deduplicating by Person ID preserves all relations |
| Per-department distinct Persons | D1 **2**, D2 **2** | 2+2 = global distinct Persons |
| Declared allocation measure | D1 **1.60**, D2 **1.40**, total **3.00 FTE** | FTE is derived from structural membership without explicit rule |
| D1 Membership×Project join rows | **4**, but only **2** original D1 Memberships | SQL result occurrences are new business Subjects |
| Naive joined FTE (a negative oracle) | **3.20** instead of correct D1 **1.60** | Joined sum is a valid aggregation of original allocation |
| M1 occupant as of 2027-06-30 / 07-01 | **P2 / P3** | Role occupancy has no time boundary |
| P3 direct Membership in D1 | **false**, despite P3 holding M1 on July 1 | Occupancy implies a direct Department membership without an accepted bridge rule |
| P2 node vs A3 edge selection | **Person P2** vs **Membership A3** | A relation unlink is Person deletion |
| Aggregate bar / derived edge edit | **unsupported** / **unresolved** | Every view element has a uniquely invertible update |
| Old selection epoch | **stale** | Previous-view row remains actionable after context change |

The checker covers 27 determinate local assertions. These test inputs are *not* database-source data and the expected values are illustrative business assumptions, not an adopted Organization Profile.

## Second reference: temporal and claim-relative changes

The file **verify-change.mjs** is a second **read-only synthetic test**. It does not subscribe to events or implement a reactive cache. Its 11 checks compare explicit snapshots and assess whether a displayed claim could be reused:

| Change pressure | Independent observation | What must not be inferred |
| --- | --- | --- |
| Rename department D2 only | Person/Membership counts and allocation FTE unchanged | Displayed D2 name remains current without re-render |
| Reordering stored rows | Same relational claims | Positional row indexes are semantic identity |
| A2 correction *recorded* July 2; valid ending June 1 | At valid June 15, known July 1: **4 Memberships**, known July 3: **3 Memberships**; D1 count **2→1**, FTE **1.60→1.00**; Person distinct stays **3** | A single "current" date can replace separate valid/recorded times |
| Valid May 31 after correction | Four Memberships remain | Recorded correction invalidates every historical snapshot |
| M1 Occupancy transition on July 1 | P2→P3; P3 has no direct D1 Membership | Position occupancy always creates direct Membership |
| FTE allocation A2 60→40, A3 40→60 | D1 **1.60→1.40**, D2 **1.40→1.60**; counts unchanged | Changing a measure changes entity identity |
| Incomplete slice | One observed relation proves existence for the slice; no full count or universal absence | Missing row on first page means no such relation exists |
| Later matching record | Previously displayed negative claim becomes false | Derived NOT EXISTS results are append-only |
| Old selection epoch | Old A3 selection cannot become a current Action target | A displayed edge remains actionable after selection change |

**Qualifications:** (1) A value staying equal for a particular change does not prove generic dependency independence. Retain a claim only when its relevant dependencies, Scope, Rule, and Source version are understood. (2) This is a closed synthetic fixture, not proof of production completeness or acceptance.

**Ownership:** [#2582](https://github.com/hat47x/tei-platform/issues/2582) owns change impact and refresh; [#2506](https://github.com/hat47x/tei-platform/issues/2506) owns Context scope/epoch; [#2593](https://github.com/hat47x/tei-platform/issues/2593) owns Measure aggregation; [#2597](https://github.com/hat47x/tei-platform/issues/2597)/[Draft #2601](https://github.com/hat47x/tei-platform/pull/2601) own Instance dirty patches and composite saves; [#2500](https://github.com/hat47x/tei-platform/issues/2500) owns Intent; [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns the TEI product end-to-end. Neither synthetic file implements these product modules.

## Third reference: target-preserving read-only handoff (2026-11-02)

`verify-handoff.mjs` adds **16 isolated Node tests** for the transition from a semantically justified organization-view selection to a **candidate** action/request target. **This is a synthetic evidence comparator only**; its `assess()` function does not invoke real TEI Intent, Driver, Authorization, Provider, Receipt or Business completion. Its field list is a test record, **not a new required Core type**.

Evidence used in the checks: Function Instance I1, parent Anchor D2, selection epoch 7, accepted meaning revision org-r7 (hypothetical), Query membership-q1, provider snapshot snap-3, selected occurrence edge-display-3, **Person P2**, **Membership relation A3**. `unlink(A3)` and `delete(P2)` must remain different operations.

| Pressure | Local expected classification |
| --- | --- |
| Matching exact read/selection/target evidence | `candidate` **only; never executable authorization** |
| Changed meaning/epoch/Anchor/Function Instance/snapshot | `unresolved` until current evidence is rechecked |
| Same Person, different Membership A2 vs A3 | Different relation occurrence; `unresolved` for mismatched target |
| Request carries A2 although candidate was A3 | `unresolved`; matching OperationID alone would not prove A3 effect |
| Request operation changes unlink→delete | `unresolved`; a relation unlink is not a Person deletion |
| Required complete read but first page incomplete | `unresolved` for the all-scope claim (individual known target may have narrower requirements) |
| Display-only label rename | Does not by itself invalidate an otherwise identical identity binding |
| Two Function Instances edit the same business relation | Separate dirty-patch ownership and editing contexts |
| Two provider operations behind one Save button | No atomicity from UI grouping alone |
| One confirmed effect and one unknown effect | Business completion remains unproven |

**Historical test evidence for the prior, pre-refactor file only:** An isolated local Node **v22.16.0** process executed `node --test verify-handoff.mjs` with **16/16 PASS, 0 failures, exit 0**. The tested file was 5,954 UTF-8 bytes with SHA-256 `afbc6a7c4cf88c32d327739daaae45e065d3aadc5f3f5f171e3f8b57c698c7a9` and Git object hash `3cce96596672a3581214d1e0ffab3dc9d5f29f4b`; GitHub readback returned **the exact same Git blob SHA**. This establishes equality of the tested and committed source, but **not TEI product E2E**.

Cross-owner integration: [#2598](https://github.com/hat47x/tei-platform/issues/2598) owns candidate-to-actual-Request evidence, [#2597](https://github.com/hat47x/tei-platform/issues/2597) owns Function Editing Session/Save Intent and partial-success semantics, [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns the Query→Context→Claim→Pattern→Intent same-version vertical test; [#1174](https://github.com/hat47x/tei-platform/issues/1174) owns actual Effect/Receipt, and [#1124](https://github.com/hat47x/tei-platform/issues/1124) owns Modeler Source edits. A future adapter should consume the existing contracts rather than reimplement this synthetic comparator as a new product engine.

## Fourth reference: multiple semantic presentations from one source

The file `verify-projection.mjs` reuses a **single in-memory synthetic Organization source revision** (`org-synthetic-r7`) and creates **four read-only presentations**—Table, Graph, department-member Tree/forest and Chart—plus a time-sensitive Position Occupancy lookup. The tests use explicitly authored synthetic assumptions, **not** a TEI model loader, a compiled DSL, or a working UI renderer.

The **same Person P2** appears in two Table rows and two Tree leaves, while Graph has one P2 Person node and two *different* Membership edges A2/A3. The different occurrences never create additional Person identities.

| Presentation | Meaning preserved | Information that is hidden / cannot be inferred |
| --- | --- | --- |
| Table | Membership occurrence ID, Person, department, direct provenance | Display row ID is not the business key |
| Graph | Person/OrgUnit subject IDs, Membership occurrence ID and relation role | A line does not prove a business Rule or action Authority |
| Tree with all occurrences | Parent department and one view occurrence per membership | Counting Person leaves across departments is not distinct Person count |
| Collapsed Tree (single displayed parent) | Explicitly chosen preferred relation (e.g. P2→A2), visible paths and **explicit hidden A3** diagnostic | Without a declared display preference the parent remains **ambiguous** and both edges stay visible; with a preference the projection is **lossy and read-only**, never a complete editable organization tree |
| Chart | Per-dept Membership count / Person distinct / explicit FTE allocation and edge provenance; global Person distinct | An aggregate group is not a directly editable source Relation |
| Occupancy lookup | Position M1 has P2 before July 1 and P3 starting July 1 | Occupying M1 does **not** create a direct Membership(P3,D1) |

**Evidence**: On 2026-11-02 the original 13 assertions and then the expanded 16-case Tree-preference reference passed in an isolated V8 test runner. **The current shared-fixture version also passed 16/16 V8 local checks**, after GitHub readback. Git blob `5c0f88ab3f9214c7b283f2d8ee64f1df7753e9f4` identifies only the earlier **pre-refactor** version, not the current exact file. This is **not an actual Node CLI, TEI Go, Parser, Function Pattern, UI, Web, Host or Provider execution**. For the exact Node run, CI and built product Evidence, use the subsequent integration stage; do not promote this local result.

**Meaning nuance uncovered by the negative Tree case:** first-in-source-row order is not an accepted managerial reporting line, a preferred parent, or an authorization chain. A single-parent Tree layout requires a **declared view preference** (not necessarily a business fact), while the underlying model retains *all* relations. If a preference is missing/invalid, the reference keeps both P2 edges visible and marks the choice unresolved rather than selecting one arbitrarily. It may choose one for visual convenience only if the choice is explicitly supplied, labels the hidden edge, and disables inverse edits.

**One deliberate limit:** Table/Graph/Tree/Chart functions are locally implemented consumers of a hard-coded synthetic fixture. The reference demonstrates *possible* semantics-preserving view data and detection of one lossy projection; it **does not prove that TEI's generic Data/Relation definitions already cause such views to be derived automatically**. This remains the work of [#1127](https://github.com/hat47x/tei-platform/issues/1127)/[#2507](https://github.com/hat47x/tei-platform/issues/2507), with fidelity owned by [#2497](https://github.com/hat47x/tei-platform/issues/2497).

**Connection to current code, not a second implementation plan:** [PR #2587](https://github.com/hat47x/tei-platform/pull/2587) (merged 2026-11-02) contains `internal/operations/surface.go`, whose standard `OperationSurface` lists single-Data `list/read/create/update` operations and declared Capability binding; this does **not yet prove a built-in Relation-occurrence `unlink` operation, or Tree/Graph/Chart generation**. A3 selection is only `select-relation` in this reference. Translating that selection to a business write still requires an explicitly accepted Relation Action/Capability/Host binding under [#2500](https://github.com/hat47x/tei-platform/issues/2500)/[#2598](https://github.com/hat47x/tei-platform/issues/2598), with real effects checked by [#1174](https://github.com/hat47x/tei-platform/issues/1174).

## Fifth reference: explicit Function Data Context versus an available relation

`verify-context.mjs` consumes the **same synthetic source and scenario**. Its selected-Department Function definition explicitly binds the `Membership` relation path, `Relation occurrence` grain, `id/person/dept` fields, parent anchor `D1`, requested result scope and model revision.

The read-only local comparison checks:
- Mapping only D1 yields A1/A2, **not** the available but unmapped D2 relationships A3/A4. An extra Position Occupancy path is not an implied Function read or a direct Membership.
- The authored request `all-for-selected-department` is **not itself observed completeness**. A partial/unknown Provider Result cannot prove full count, total distinct persons or universal absence. One actually observed entity can support a positive existence claim.
- A synthetic completeness assertion contradictory to the closed oracle yields `inconsistent` (this is the closed fixture's independent knowledge, **not** a production Provider verifier).
- An absent/unresolved path, role, scope, grain, model revision or undeclared field is `unsupported`; no UI/DSL method chain or Graph topology implicitly fills it in.
- Two read-only UI-consumer projections (table rows/graph edges) preserve the **same relation IDs, anchor, Source revision, selection epoch, snapshot and result completeness**.
- The read map conveys no new Host grant, commit, authorization, disclosure permission or effect.

**Current evidence:** the GitHub-readback `verify-context.mjs` returned **20/20 PASS** in the restricted local V8 compatibility runner. Exact current-file Node CLI and actual TEI Data/Relation→Function Data Context→Query/Provider→UI/Action **NOT RUN**.

**Relevant current other threads:** [#2619](https://github.com/hat47x/tei-platform/issues/2619) (Everything is Data), [#2624](https://github.com/hat47x/tei-platform/issues/2624) (explicit Function Data Mapping), [#2506](https://github.com/hat47x/tei-platform/issues/2506) (Data Context owner), [#2507](https://github.com/hat47x/tei-platform/issues/2507) (Pattern), [#2586](https://github.com/hat47x/tei-platform/issues/2586) (single-version vertical test). These remain the owners of real TEI product integration. No novel TEI Context runtime, permission gate, SQL engine or Graph renderer is introduced by this reference.

## Sixth reference: declared footprint, synthetic acquisition witness and disclosed UI output

`verify-acquisition.mjs` is an **Organization-specific** pressure case for the five-stage distinction introduced in [#2619](https://github.com/hat47x/tei-platform/issues/2619), [#2622](https://github.com/hat47x/tei-platform/issues/2622), [#2611](https://github.com/hat47x/tei-platform/issues/2611), [PR #2627](https://github.com/hat47x/tei-platform/pull/2627) (merged 2026-11-02) and [PR #2628](https://github.com/hat47x/tei-platform/pull/2628) (open).

| Evidence layer | Example in this finite source | What is still unknown |
| --- | --- | --- |
| **Declared** Function mapping | Membership/D1, fields id/person/dept, Relation-occurrence grain | This is a request, not proof of physical acquisition |
| **Realized** route selection (synthetic) | Scenario chooses `replica` or `primary` with distinct provider/mapping/alias | Which actual Provider/DB session will service a real TEI Action |
| **Acquisition witness** (synthetic) | A log of requested physical fields, paths, route, recorded-time and read row IDs | This deliberately modeled trace does not inspect actual DB I/O, caches, query pushdown, driver internals or OS/network |
| **Result Slice** | D1 A1/A2 on replica known-at July 1; A1 on primary known-at July 3 (retroactive A2 correction) | Shared snapshot/cohort, current completeness, cross-source identity, RLS and effective record scope |
| **Disclosed** Table/Graph value | Both views preserve obtained ID/person/dept and scope | External AI disclosure policy, logging/timing channels, and execution grant are separate concerns |

**The changed domain-specific pressure** is **Person P2 / Membership A2**. The same Semantic Membership type and even the same snapshot token `s7` occur on two synthetic routes, but their recorded-time cutoffs differ. **Source version equality and snapshot token string equality do not prove same read cohort**. A finite independent witness of equal scope/cohort lets a mismatch be reported as a *fixture contradiction*, without claiming that the actual providers are inconsistent. If one route is incomplete or the common witness is absent, the result is `not_established`.

**Overfetch witness:** the simulated Provider can read `provenance` or the unmapped `Occupancy` Relation and return the same Table/Graph rows. The two outputs are intentionally equivalent while the modelled acquisition traces differ. This is a diagnostic that a **physical acquisition witness must be evaluated separately**, not evidence that the chosen Query implementation performed a real overfetch. A Provider may have legitimate auxiliary-key or filtering reads; the meaning/policy of extras needs an explicit explanation. When the synthetic access trace is unknown, the comparator returns `unverified`, even if the visible result looks narrow.

**Routes are scenario data, not semantic Source:** `scenario-fixture.json` now includes two `syntheticProviderRoutes` with distinct stable route IDs, aliases, provider/map identities and known-at cutoffs. Changing an alias label while keeping the route ID cannot silently select another route. Nothing in the `source-fixture.json` semantic Dataset assigns a universal physical Provider to all Person/Membership relationships.

**Latest evidence**: the exact GitHub-readback `verify-acquisition.mjs` code returned **22/22 PASS** in a V8 local test shim, including the route, temporal, incomplete, overfetch, unknown-trace, disclosure and read-only cases. **Current-HEAD native Node CI and real TEI Provider tracing are not yet qualified**. This is an independent fixture, not a new Data Operations/Provider Resolver/Authorization engine. Follow-up product tests belong to [#2622](https://github.com/hat47x/tei-platform/issues/2622)/[#2628](https://github.com/hat47x/tei-platform/pull/2628) (actual access), [#2611](https://github.com/hat47x/tei-platform/issues/2611)/[#2627](https://github.com/hat47x/tei-platform/pull/2627) (route/cohort identity), [#2506](https://github.com/hat47x/tei-platform/issues/2506) (Context), [#2516](https://github.com/hat47x/tei-platform/issues/2516) (external disclosure), and [#1174](https://github.com/hat47x/tei-platform/issues/1174) (effect).

## Seventh reference: Display mapping versus Rule dependency and outward disclosure (2026-11-02)

`verify-rule-footprint.mjs` extends the same synthetic Source/Scenario to distinguish four independently meaningful layers:

1. **Display mapping:** the member-list view of D1 only asks for Membership `id/person/dept`; its two visible rows are A1/P1/D1 and A2/P2/D1. The view contains no allocation basis-points or department capacity.
2. **Rule input footprint:** a **test-only Rule** (`org.fte-within-capacity`, `cap-rule-r1`) needs Membership occurrence identity, the independently declared allocation values, and a DepartmentCapacity fact. The closed source carries a hypothetical capacity of **180 basis points** for D1 and 160 for D2. Current D1 explicit allocations total **160**, so the synthetic comparison reports `satisfied`. Reducing *only* the D1 capacity to **150** makes the synthetic comparison `not_satisfied`, with **the same member-list rows**. This Rule is **not** an accepted personnel/authority policy or TEI business Rule.
3. **Physical input evidence and current cohort:** the Rule requires an explicit read-path witness for Membership, Allocation, and DepartmentCapacity; missing allocation, unknown Scope/trace or incomplete results yield `unresolved`. A display read on replica `s7` and a Rule read on primary `s7` are **not automatically in the same read cohort**; one global snapshot token is insufficient. At the synthetic primary's different known-at cutoff D1 has only A1 (100 basis points); cross-route reuse requires independent evidence.
4. **Consumer disclosure:** outcome disclosure is a separate simulated scenario choice. The local member-list and external-summary Consumers receive the **same `{status:'unavailable'}`** for both worlds (capacity 180 and capacity 150); their outputs do not reveal either capacity or rule verdict. The explicitly configured local review Consumer can see both. **These fixture flags are not real identity, permission, policy, network, timing or side-channel enforcement.** The rest of organization Data remains ordinary usable TEI Data in the intended product design.

**Additional independent negative cases:** changed Rule revision or Source revision, unknown Rule operator/Role, missing capacity, omitted Allocation data, fabricated `complete` on an incomplete synthetic slice, JOIN multiplication of FTE (4 result occurrences vs 2 source Memberships), and synthetic Rule satisfaction misrepresented as a write/Host grant. The candidate status is always labeled `candidate-only` and does not construct a real Action/Request or prove a Business completion.

**Evidence:** the exact file fetched back from GitHub, blob `69f8c6d40b7cea7335bdb417ac1535f177bc63ff`, returned **29/29 PASS in V8** and was part of the **114 native Node tests** in the successful pinned run #41456286651 (commit `a01a2add...`). The seventh file was added to the pinned Node workflow and **the first native exact-head job succeeded at commit `a01a2addc680bc33c5ceee782e8b48e3a77a736f`**; subsequent edits require new exact-head qualification. Do not promote the 141 V8 successes to a real TEI Rule engine, actual Provider read trace or production access/disclosure test.

**Owner handoff:** [#2624](https://github.com/hat47x/tei-platform/issues/2624) owns Display/Rule Dependency footprints and failed prerequisite qualifications; [#2622](https://github.com/hat47x/tei-platform/issues/2622)/[#2628](https://github.com/hat47x/tei-platform/pull/2628) own physical read/Provider; [#2611](https://github.com/hat47x/tei-platform/issues/2611)/[#2627](https://github.com/hat47x/tei-platform/pull/2627) own route/cohort; [#2506](https://github.com/hat47x/tei-platform/issues/2506) owns typed Data Context; [#2516](https://github.com/hat47x/tei-platform/issues/2516) owns Consumer disclosure; [#2507](https://github.com/hat47x/tei-platform/issues/2507) owns Pattern/Action applicability; [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns end-to-end TEI integration; [#1174](https://github.com/hat47x/tei-platform/issues/1174) owns actual Effect/Receipt. No second Rule evaluator, Security gate or Canonical Source is introduced as a product component here.

## Eighth reference: non-output Query dependencies and Claim-relative invalidation

The new `verify-query-dependencies.mjs` uses the **same synthetic Organization Source and Scenario**, but deliberately exercises a new cross-domain boundary informed by [#2635](https://github.com/hat47x/tei-platform/issues/2635) / [PR #2634](https://github.com/hat47x/tei-platform/pull/2634), merged 2026-11-02 (actual Go `query.DeriveFromAnalysis` characterization): **the fields used to generate displayed values are not a complete dependency footprint for deciding which rows exist, their order, Query freshness or which Rule outcomes remain qualified.**

The reference is **not** a SQL query executor or a TEI Query IR adapter. It independently specifies a narrow synthetic Query over known Membership, Allocation and Project facts:

- **Hidden predicate (WHERE)**: a D1 membership listing displays only `id/person/dept`. With `allocation >= 50`, both A1/A2 occur; with `>= 70`, only A1 occurs. A2 Allocation changing from 60 to 40 removes A2 under the same `>=50` predicate while its base Membership ID/person/dept remain unchanged. The three *output-value* fields are identical in all cases; the predicate depends on a non-output allocation field.
- **Hidden order dependency (ORDER BY)**: ordering D1 by Allocation descending changes from A1/A2 to A2/A1 after A2 allocation increases to 120, without changing the displayed set of Relation occurrence IDs. Changing the order is a meaningful Consumer difference even when the unordered result is equal.
- **Existence and non-monotone dependencies**: with J1/J2 Projects in D1, `EXISTS(Project in D1)` includes A1/A2 and `NOT EXISTS(Project in D1)` includes neither. Removing both Project facts *retracts* the EXISTS result and *adds* the NOT EXISTS result even though the projected Membership fields are unchanged. An incomplete Project scope **cannot prove NOT EXISTS**.
- **Unknown/partial/route freshness**: missing non-output Allocation input, partial Membership/Project set, an unknown Query Operator, and a mismatched Source revision are not coerced to known results. Both Replica and Primary use synthetic snapshot label `s7`, but have different known-at cuts and D1 membership sets. Even identical rows observed twice are not automatically the same cache observation.
- **Claim-relative invalidation guidance, not a production cache**: Allocation changes may affect filtered/ordered views and Rule outcomes while leaving a plain Membership display candidate unchanged. Capacity changes affect the Rule candidate but not plain Membership rows. Project changes affect EXISTS. The function reports `recompute-candidate` / `unaffected-candidate` **only with declared complete scope/observed change/cohort inputs**; otherwise it reports `unresolved`. This does not authorize actual data reuse, guarantee Provider no-overread, or validate a real TEI Query Analysis.

The 25 expected comparisons include **hard-coded negative/positive expected IDs** (not copied from the function's outputs), but share a synthetic source and code assumptions. They do **not** demonstrate an independently authored human-accepted verification criteria, hidden holdout or the actual SQLGlot Worker. In particular, PR #2634's current Go characterization shows *non-output expression information exists* in parts of the candidate IR, **not** that Typed Query Control/Existence Dependency edges or safe consumer disclosure already exist.

**Exact-commit verified test evidence:** GitHub readback `verify-query-dependencies.mjs`, blob `8e08f5f5f301303b9f9435e5a43eb218021b8559`, **25/25 PASS in the restricted V8 test shim** and included in the [successful native Node 22.16.0 run #41462231920](https://github.com/hat47x/tei-platform/actions/runs/41462231920) at **commit `7eaf1afca3a18a42eb7deb090b6a8be9150c8bca`**. The job logged **27 script checks + 139 Node tests = 166 synthetic checks**, zero failures. Exact source identity is essential; this README update itself creates a different HEAD requiring a new run before asserting current-head CI success. Formal TEI Query/Context/Rule/Provider/DB/Studio/SHO/Host/Owner approval remains NOT RUN.

**Owner mapping:** #2387/#2635 actual Query IR and condition dependencies; #2506/#2624 Function explicit footprint and Rule prerequisites; #2582 granular freshness/invalidation; #2497 projection semantics; #2514 verification-criteria independence; #2622/#2628 actual Provider reads; #2516 outward disclosure; #2586 one-real-TEI-function vertical; #2600 build-time dependency/cache rules (not to be conflated with runtime data re-query). No second Query engine, Cache, Semantic Registry or generic Host is introduced.

## Owner boundaries and integration

- TEI [#2386](https://github.com/hat47x/tei-platform/issues/2386), [PR #2389](https://github.com/hat47x/tei-platform/pull/2389) (open) §§39–41 own the organization-domain synthetic meaning/oracle.
- [#2497](https://github.com/hat47x/tei-platform/issues/2497) owns semantic preservation/loss under Projection.
- [#2506](https://github.com/hat47x/tei-platform/issues/2506) owns Function Data Context's Subject/Relation occurrence, Query grain, scope, and selection epoch.
- [#2507](https://github.com/hat47x/tei-platform/issues/2507) owns Function Pattern applicability and view selection.
- [#2593](https://github.com/hat47x/tei-platform/issues/2593) owns Measure×aggregation-axis legality, distinct Persons vs Membership/FTE, and time-dependent aggregates.
- [#2500](https://github.com/hat47x/tei-platform/issues/2500) owns view-change interpretation into typed **candidate** Operation Intents, not execution. Its separate PR #2517 (merged 2026-11-02) is an earlier locally tested reference.
- [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns actual **Query→Context→Claim→Pattern→Intent** end-to-end headless observation on the same accepted source/fixture/versions.

**Next meaningful evidence:** re-express this source as existing TEI Data/Relation declarations and external Dataset, then verify equivalent Claims with actual TEI Query/Context/Pattern adapters. Re-use existing Scenario/Provider rather than expanding this reference into a second Query engine or source of truth. If a source-to-view or view-to-action mapping is nonunique, keep it unresolved.

**Standard organization access:** normal departments, membership, and position data are ordinary accessible TEI Data in the same TEI world; special protection for items such as government ID or payment card data is not the default for this module. A visualization candidate or operational Role never grants Host execution privileges.
