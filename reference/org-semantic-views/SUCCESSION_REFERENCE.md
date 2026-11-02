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


## In-flight Case responsibility pressure (ORG-RC; 2026-11-03)

The same standalone successor fixture now has **10 additional Node tests (RC-01 to RC-10)**, distinct from the original 16 organization succession tests. It introduces toy pending purchase Case Q1, with its old D1 responsibility and mutually superseded R1/R2 Case accountability observations. The 26 tests are **synthetic counterexamples**, not actual organization or authorization processing in TEI.

They distinguish Org succession vs Case migration; an explicit single-responsible-org Rule vs a joint Rule; a **positive witness of two conflicting assignments** vs a **zero-result requiring scoped completeness**; application-time freezing vs approval-time resolution; prior publication vs retrospective correction; checked owner vs synthetic final-post-image mismatch; displayed candidate vs SoD/delegation/current Grant; and the completeness of **Case population** vs the completeness of **each Case's responsibility relations**.

Rule acceptance, cohort completeness, Grant, post-image, and receipt in this file are **test-only injected inputs**. This suite must not be used to assert real Provider acquisition, Owner Business Rule adoption, actual Commit fencing, end-to-end TEI Function/Query/SHO/Studio functionality, or production deployment qualification. The existing workflow executes this same `verify-succession.mjs` file; exact-head Node CI status must be confirmed separately before citing any new success.

Related owners: organizational meaning #2386 and [design Draft #2389 §51](https://github.com/hat47x/tei-platform/pull/2389); corpus #2510; shared headless integration #2586; actual Read #2628; Owner-write coverage #1129; effect fence #1174; scenario and oracle #1131/#2514. No second Org/Case schema, central Responsibility Registry, Host, Workflow or Authorization engine is introduced.


## Cross-view response freshness and negative dependency (VF-01–24)

The add-on `verify-consumer-freshness.mjs` **imports the merged shared org fixture** and simulates two Provider routes. It tests distinct obligations for rejecting late responses and retracting an already-painted result after Context/Source/Role/Provider changes; required read evidence; partial/unknown empty results; non-output WHERE/ORDER/EXISTS and changed Rule dependencies; dirty Patch retention when selected occurrence/anchor changes; multi-claim read cohorts; 0→1→0 responsibility history; and non-atomic multi-Provider SaveGroup.

The requalification output is **test-only**. `synthetic-assumed`, `witnessed`, `same-synthetic-read-session`, `candidate-only`, `partial-only` are synthetic labels, not production TEI public interfaces or authoritative Provider/Grant/Effect proof. An ordinary organization's Data remains accessible under normal Data rules; this is not a new organization-specific ACL.

At GitHub Blob `160cc7ef8f141923516bfc446533b6bc92b9d3bc`, a V8-compatible runner with the **actual shared JSON fixture contents** and its imports replaced completed **24/24 synthetic checks PASS**. The exact-head Node CI and real Workbench/Query/Provider/SHO/Host integrated behavior require separate evidence. Design [Draft #2389 §52](https://github.com/hat47x/tei-platform/pull/2389), specialized UI handoff #2631, Context/Claim #2506/#2582, editing #2597/#2601, physical Read #2628, vertical #2586, Intent/Effect #2598/#1174.
