# R7B Replay Recipe / Repeatability Contract

- Status: Source-defined / actual replay not executed
- Date: 2026-09-19
- Workstream: Selective Knowledge Routing / Replayable Behavior Evidence
- Branch owner: research/selective-knowledge-routing-r6
- Input boundary: R7A actual handoff entry
- Next step: actual R7A handoff acquisition, then operational replay execution

## 1. Purpose

R7B turns an R7A behavior-unit handoff into an explicit replay recipe without treating the recipe itself as replay success.

The source contract keeps the following states separate.

    actual Gate C Ready
      != actual R7A handoff acquired

    actual R7A handoff acquired
      != replay executed

    one replay pass
      != repeatability established

    behavior repeatability
      != deeper workflow coverage

No current Repository fact is used to fabricate Repository-external Evidence.

## 2. Required input

R7B replay recipe derivation starts only from the already-defined R7B execution intake.

Each behavior must preserve at least:

- behavior identity
- Semantic Operation identity
- Target realization reference
- origin raw Functional artifact filename and exact SHA-256
- actual R7A handoff artifact exact SHA-256
- final Gate C readiness artifact exact SHA-256
- Semantic / Functional summary exact SHA-256
- Handover Baseline B
- baseline Repository revision R
- origin Target build T
- Semantic / Functional run identity
- VisualRun provenance
- Observation class
- ReplayExecutionStatus = not-executed
- RepeatabilityStatus = not-observed
- CoverageClass = behavior-unit

Any input that already claims replay success is rejected.

## 3. First Web Target replay units

The current First Web Target set remains exactly six behavior units.

1. first-web/read-only
2. first-web/command-success
3. first-web/authentication-denied
4. first-web/authorization-denied
5. first-web/validation-not-satisfied
6. first-web/execution-error

R7B derives a Target-local HTTP recipe for each unit from the current Gate C scenario.

The recipe records a named request fixture rather than promoting credentials, DOM selectors, coordinates, route names, or HTTP details into Canonical business identity.

Target-local interaction is operational replay material only.

    Target interaction recipe
      != Canonical identity
      != Semantic Operation authority

## 4. Initial replay target identity

The first replay recipe pins ReplayTargetBuildRevision to the origin Target build revision from the actual handoff.

For the current First Web Target this means the first actual R7B replay must execute the behavior against the same T identity that produced the Gate C Functional Evidence.

A later cross-revision replay is a separate Evidence question and must use a distinct replay target identity instead of silently replacing origin T.

## 5. Replay observation

Each actual replay run must produce an observation with:

- behavior identity
- distinct replay run identity
- explicit replay Target build revision
- observed transport result
- observed semantic result class
- raw replay artifact exact SHA-256
- execution pass/fail

The raw replay artifact remains evidence and is not replaced by a normalized result.

## 6. Repeatability rule

Repeatability requires at least two distinct successful replay runs for the same behavior and the same explicit replay Target build.

All runs must preserve the expected semantic result class and expected transport boundary.

Exact replay output bytes are not required to be identical. Run-specific or non-semantic material may cause exact-byte differences.

Instead:

- every raw replay output is retained and fingerprinted
- semantic result class must remain stable
- transport expectation must remain stable
- replay Target build identity must remain stable

Therefore:

    same semantic result across distinct replay runs
      + exact raw artifact identity for every run
      -> observed-repeatable

but:

    one successful run
      -> not enough for repeatability

and:

    equal raw bytes alone
      -> not enough for semantic repeatability

## 7. Coverage boundary

R7B repeatability is behavior-unit Evidence.

Even all six behavior units becoming repeatable does not by itself prove:

- cumulative multi-step workflow correctness
- long-running state transition correctness
- cross-Target portability
- current Authorization authority
- Business Procedure authority
- whole-application acceptance

CoverageClass stays behavior-unit and DeepWorkflowCovered stays false in this source contract.

## 8. Test-only source

Source pressure is implemented in:

- tools/reference-business-procedure-check/replayable_behavior_r7b_replay_recipe_pressure_test.go

The tests require:

1. exactly six replay recipes
2. exact provenance retention from R7B intake
3. no fabricated replay/repeatability state
4. Target-local interaction remains non-Canonical
5. first replay pins origin Target build
6. one replay pass is insufficient
7. at least two distinct successful runs are required
8. semantic or Target-build drift rejects repeatability
9. every replay output has an exact raw artifact fingerprint
10. differing exact replay bytes may still be repeatable when observed meaning is stable
11. behavior repeatability is not promoted to deeper workflow coverage

## 9. Operational intake acquisition runner

The source-defined R7B intake can be materialized from Repository-external R7A artifacts with:

- `tools/run-first-web-target-r7b-intake.sh`

The runner requires:

- `TEI_R7B_R7A_HANDOFF`
- `TEI_R7B_R7A_RECEIPT`
- `TEI_R7B_OUTPUT_DIR`

Both source artifacts and the output directory must remain outside the Repository.

The runner:

1. requires a clean Repository checkout,
2. fingerprints the exact R7A handoff and acquisition-receipt bytes,
3. source-qualifies `tools/reference-first-web-target-r7b-intake`,
4. derives `r7b-execution-intake.json`,
5. fingerprints the exact intake artifact,
6. writes `r7b-intake-receipt.json` outside the Repository,
7. verifies that the checkout did not change during acquisition.

The intake receipt explicitly records:

```text
replayExecuted = false
repeatabilityObserved = false
```

Therefore:

```text
R7A handoff acquired
  -> R7B execution intake acquired
  != replay executed
  != repeatability observed
```

The runner performs no network replay and starts no First Web Target server.

## 10. Machine-readable replay plan

The execution intake can be converted into a deterministic machine-readable replay plan with:

- `tools/reference-first-web-target-r7b-replay-plan`

The plan pins:

- exact execution-intake SHA-256
- source R7A handoff / acquisition-receipt SHA-256
- origin and replay Target build revision
- one named Target-local request fixture per behavior
- Target-local interaction description
- expected HTTP boundary
- expected semantic Observation class
- origin raw Functional artifact identity
- `RequiredDistinctRuns = 2`
- `ReplayExecutionStatus = not-executed`
- `RepeatabilityStatus = not-observed`
- `CoverageClass = behavior-unit`

The plan intentionally does not carry login IDs, credentials, passwords, selectors, DOM coordinates, click sequences, execution authority, or a replay-success field.

```text
execution intake
  -> replay plan
  != credentials
  != execution authority
  != replay execution
```

`RequestFixtureID` and `TargetLocalInteraction` are Target-local operational material. They do not become Canonical Application identity or Semantic Operation authority.

The first replay plan pins `ReplayTargetBuildRevision` to the origin Gate C Target build `T`. A cross-revision replay requires a different explicit plan / Evidence identity.

## 10. Evidence height

This document, the test-only source, the intake CLI source, and the operational intake runner are Source / Design Evidence until the Repository-external artifacts are actually supplied and the runner is executed.

They do not claim:

- actual R7A handoff artifact acquisition in this Repository session
- actual R7B execution-intake acquisition
- actual replay execution
- actual replay PASS
- actual repeatability
- complete workflow coverage

Actual replay execution remains blocked until the Repository-external R7A handoff and acquisition receipt are supplied, exact-byte verified, and accepted by the R7B intake path.

## 11. Next step

After actual R7A handoff acquisition:

1. run the R7B intake acquisition runner,
2. retain the resulting intake artifact and receipt outside the Repository,
3. derive the machine-readable replay plan from the actual R7B execution intake,
4. retain the exact replay-plan artifact identity outside the Repository,
5. then connect the plan to an operational replay runner that:

- reads the actual R7B execution intake and replay plan,
- verifies its exact SHA-256 and source provenance,
- verifies the six origin raw artifact identities,
- executes at least two distinct replay runs per selected behavior against explicit T,
- stores every raw replay artifact outside the Repository,
- produces a repeatability assessment without converting Target-local replay details into Canonical meaning.
