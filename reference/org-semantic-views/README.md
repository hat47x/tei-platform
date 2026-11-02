# Organization semantic view: isolated, read-only reference (2026-11-02)

**Status:** proposal/reference. This folder uses synthetic data only. It is neither an accepted TEI semantic contract nor a production Query/Function/UI/Host implementation.

## Run

```sh
node reference/org-semantic-views/verify.mjs
node --test reference/org-semantic-views/verify-change.mjs
node --test reference/org-semantic-views/verify-handoff.mjs
```

The file is a dependency-free JavaScript module: it computes an in-memory reference and throws on the first unexpected observation. A successful run emits a JSON summary. No DB is created and no files, services, accounts, or business records are changed.

**Evidence recorded during preparation:** `verify.mjs` returned **27/27 PASS** in the execution tool's V8 isolate and **27/27 PASS** when the committed GitHub source was read back and re-evaluated. The new `verify-change.mjs` returned **11/11 PASS** in the same V8 execution tool using a narrowly compatible `node:test`/`node:assert` shim after readback of its exact source. **Separately, the exact `verify-change.mjs` source was reproduced in an isolated Node v22.16.0 environment and `node --test` completed with exit code 0, 11/11 PASS.** Source identity: 5,582 ASCII bytes / FNV-1a 32 `0x6cb1c9d8`, matching the read-back source; SHA-256 `8156207199ada4ac1f01e911548aecfdc3bb13094bf525d20cd7b9f22489a27d`. `verify.mjs` itself has *not* been exercised in the Node CLI (27/27 V8 only). Repository CI, official TEI Go tests, SQL Worker, Context/Pattern binding, browser/Windows, actual Host/Provider, and Owner meaning acceptance remain NOT RUN.

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

**Recorded test evidence:** An isolated local Node **v22.16.0** process executed `node --test verify-handoff.mjs` with **16/16 PASS, 0 failures, exit 0**. The tested file was 5,954 UTF-8 bytes with SHA-256 `afbc6a7c4cf88c32d327739daaae45e065d3aadc5f3f5f171e3f8b57c698c7a9` and Git object hash `3cce96596672a3581214d1e0ffab3dc9d5f29f4b`; GitHub readback returned **the exact same Git blob SHA**. This establishes equality of the tested and committed source, but **not TEI product E2E**.

Cross-owner integration: [#2598](https://github.com/hat47x/tei-platform/issues/2598) owns candidate-to-actual-Request evidence, [#2597](https://github.com/hat47x/tei-platform/issues/2597) owns Function Editing Session/Save Intent and partial-success semantics, [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns the Query→Context→Claim→Pattern→Intent same-version vertical test; [#1174](https://github.com/hat47x/tei-platform/issues/1174) owns actual Effect/Receipt, and [#1124](https://github.com/hat47x/tei-platform/issues/1124) owns Modeler Source edits. A future adapter should consume the existing contracts rather than reimplement this synthetic comparator as a new product engine.

## Owner boundaries and integration

- TEI [#2386](https://github.com/hat47x/tei-platform/issues/2386), [Draft #2389](https://github.com/hat47x/tei-platform/pull/2389) §§39–41 own the organization-domain synthetic meaning/oracle.
- [#2497](https://github.com/hat47x/tei-platform/issues/2497) owns semantic preservation/loss under Projection.
- [#2506](https://github.com/hat47x/tei-platform/issues/2506) owns Function Data Context's Subject/Relation occurrence, Query grain, scope, and selection epoch.
- [#2507](https://github.com/hat47x/tei-platform/issues/2507) owns Function Pattern applicability and view selection.
- [#2593](https://github.com/hat47x/tei-platform/issues/2593) owns Measure×aggregation-axis legality, distinct Persons vs Membership/FTE, and time-dependent aggregates.
- [#2500](https://github.com/hat47x/tei-platform/issues/2500) owns view-change interpretation into typed **candidate** Operation Intents, not execution. Its separate Draft #2517 is an earlier locally tested reference.
- [#2586](https://github.com/hat47x/tei-platform/issues/2586) owns actual **Query→Context→Claim→Pattern→Intent** end-to-end headless observation on the same accepted source/fixture/versions.

**Next meaningful evidence:** re-express this source as existing TEI Data/Relation declarations and external Dataset, then verify equivalent Claims with actual TEI Query/Context/Pattern adapters. Re-use existing Scenario/Provider rather than expanding this reference into a second Query engine or source of truth. If a source-to-view or view-to-action mapping is nonunique, keep it unresolved.

**Standard organization access:** normal departments, membership, and position data are ordinary accessible TEI Data in the same TEI world; special protection for items such as government ID or payment card data is not the default for this module. A visualization candidate or operational Role never grants Host execution privileges.
