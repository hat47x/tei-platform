# Organization transition simulation — synthetic pressure only

This Reference asks whether a **read-only what-if organizational proposal** can be kept distinct from the actual accepted world, a published report, a Business Rule acceptance, an operational write target and a Host grant.

## Input and preconditions

The script `verify-transition-simulation.mjs` **imports the existing mainline `source-fixture.mjs`**, pinned by its own `org-synthetic-r7` `sourceRevision` and uses synthetic `validAt=2027-06-15, knownAt=2027-07-01`. It does **not** copy/edit any `.tei` file or save to an actual Provider or DB. Source rows are A1/P1/D1/100, A2/P2/D1/60, A3/P2/D2/40, A4/P3/D2/100 (points, i.e. hundredths of FTE). D1 capacity=180 and D2 capacity=160 under an explicitly **test-only** capacity rule.

A `reallocate` scenario retires A2 and revises existing occurrence A3 from 40 to 100. The result is D1=100, D2=200, total=300, distinct Persons=3, active Membership occurrences=3. This is a deliberate **counterexample**: a proposal preserves total FTE but violates the synthetic D2 capacity by 40, while P2's unrelated Occupancy and Q1 Case responsibility do **not** migrate.

A separate `split` scenario retires A1/A2 and creates new occurrence IDs A1new/P1/D1a and A2new/P2/D1b. It preserves total FTE but does not transfer old D1 capacity, Case responsibility or grants to successors; the successor-capacity rule is **unverified**, not satisfied.

## Synthetic-only verification

`node --test reference/org-semantic-views/verify-transition-simulation.mjs` covers TS-01..TS-20:

- Baseline distinct Person/Occurrence/FTE identity, copy-on-write proposal independence, and same totals with different relation occurrence sets.
- Trial capacity check, insufficiency of unobserved successor capacities or partial field/row readings, and test-only proposal capacity adjustments (never Owner rule approval).
- Bitemporal cut/source/Rule revision mismatch rejection, optimistic-change precondition failures, unknown proposal operator rejection.
- Unchanged Occupancy, unchanged Case responsibility, explicit typed successor comparison bridge required for FTE-only comparison, and no implicit authorization/effect from a proposal.

**Evidence:** isolated Node test logic and a deliberately closed synthetic fixture. A local V8-compatible shim ran 20/20 test bodies before branch creation, but this is **not** native Node or a GitHub Actions verdict. For CI qualification, inspect exact-head run ID, Node version, source Blob hashes and logs. No TEI Parser, actual Function Data Context, Query, Provider/DB, Studio/SHO, EKI node/resource control, Owner semantic acceptance or Host Effect is exercised.

## Planned TEI integration

1. E1: [Native TEI organization workspace PR #2681](https://github.com/hat47x/tei-platform/pull/2681) uses actual `.tei` Parser/Loader, Data/Relation, and read-only derived OperationSurface. Do not infer temporal evaluation, simulation or actual DB effects from that evidence.
2. E2: #2586/#2506/#2628 — bind the exact source/role/valid/known cut/Provider cohort to actual Query→Context and field/row observations, rejecting partial or stale witness. A proposed diff is **not** a new accepted Source revision.
3. E3: #2631/#1124/#1803 — distinguish baseline, preview overlay and R1/R2 published/current reports on Table/Tree/Graph/SHO. Preserve uncommitted Patch independently. Comparing old/new FTE requires a separately accepted Measure/Scope bridge.
4. E4: #2598/#1129/#1174 — if the business Owner accepts a transition, resolve concrete write intents (retire A2, adjust A3; or create new A1new/A2new) against current external Host authorization, transaction constraints and independent post-image reads. This is a **separate** workflow with no generated Grant or implicit commit.

The organization/capacity application remains a *standard TEI application*, not a new Core kind, DSL Host, ACL subsystem, general-purpose authority engine or bespoke simulation runtime. EKI/other resource controllers may later consume read-only simulation projections but must validate their own topology/capacity/grants and effects.

Design: Draft #2389 §55; organization business meaning #2386; independent Profile Corpus #2510. **No adoption, production execution or permissions asserted.**
