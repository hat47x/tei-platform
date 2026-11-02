# Organization semantic view: isolated, read-only reference (2026-11-02)

**Status:** proposal/reference. This folder uses synthetic data only. It is neither an accepted TEI semantic contract nor a production Query/Function/UI/Host implementation.

## Run

```sh
node reference/org-semantic-views/verify.mjs
```

The file is a dependency-free JavaScript module: it computes an in-memory reference and throws on the first unexpected observation. A successful run emits a JSON summary. No DB is created and no files, services, accounts, or business records are changed.

**Evidence recorded during preparation:** the generator's checks returned **27/27 PASS** within the execution tool's V8 isolate, and the exact file fetched back from the GitHub branch was evaluated again with **27/27 PASS**. **Node CLI, repository CI, official TEI Go tests, SQL Worker, Context/Pattern binding, browser/Windows, actual Host/Provider, and Owner meaning acceptance are NOT RUN.** A future test should capture the actual Node version/exit code and the file/commit SHA before claiming Node execution.

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
