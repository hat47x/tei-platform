# Organization split / succession synthetic reference

**State:** proposed read-only reference. Nothing here is an adopted TEI semantic schema, Owner Business Rule, verified acquisition witness or Host Authority grant.

Run `node --test reference/org-semantic-views/verify-succession.mjs`.

## Scope

This fixture is **independent** from the existing `source-fixture.json` (`org-synthetic-r7`). It supplements rather than edits the merged baseline. `verify-succession.mjs` has sixteen OC checks.

- Before 2027-07-01: D1 FTE 1.6, D2 FTE 1.4, three distinct persons, four Membership occurrences.
- At valid 2027-07-01, known 2027-07-01 (publication R1): D1a 1.0, D1b 0.6, D2 1.4, three persons/four occurrences.
- At the **same valid date**, known 2027-07-03 (retrospective correction R2): D1a 1.0, D1b 0, D2 2.0, three persons/three occurrences. The missing D1b bucket is zero only within this closed synthetic fixture.
- FTE sums to 3.0 for all cuts. Same sum does not imply same persons, memberships, organization identity or decision right.

The tests distinguish successor relationships from identity, reused external HR codes, the publication replay from restatement, explicit Owner-accepted *comparison-only* bridges from unspecified bridges, per-Measure/grain comparability, partial acquisition from known completeness, join fanout from source occurrences, old selection IDs from new update targets, and missing allocations from zero.

The synthetic `cohort='witnessed'` flag is **an assumption used in test inputs**, not an actual Provider/DB acquisition trace. No rule or successor edge implies a grant, and no downstream action is performed.

## Evidence and handoff

A local V8 dry-run of the sixteen self-contained tests passed before creating the GitHub file; the actual exact-head native Node workflow result must be independently read before calling it successful. Previous 166-check Node result for merged PR #2596 attested to a **different** fixed commit.

The required next chain is accepted TEI Data/Relation → derived Query (including non-output conditions) → explicit Function Data Context and verified Provider cohort → Measure/Claim and SHО/Studio consumer → typed Intent/Request/Effect. This reference by itself validates none of those integrated steps.

Design Owner: #2386, Draft #2389 §49–50. Corpus Owner: #2510. Other Owners: #2593 Measure grain, #2582 claim requalification, #2586 vertical handoff, #2628 physical read, #2598 Intent/Request, #1174 Effect. No fixed organization Core, secondary Host, mandatory plugin meta-ontology annotations or organization-specific ACL is proposed.
