# R7B Replay Recipe / Repeatability Contract

- Status: Replay-plan Source-defined / actual replay not executed
- Date: 2026-09-20
- Workstream: Selective Knowledge Routing / Replayable Behavior Evidence
- Development flow: direct-to-main solo-development flow; do not create a new remote work branch
- Input boundary: R7A actual handoff -> R7B execution intake -> exact replay plan
- Next step: operational replay runner against the exact Gate C Target build T

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
- `tools/run-first-web-target-r7b-replay-plan.sh`

The plan source requires both the exact `r7b-execution-intake.json` and the exact `r7b-intake-receipt.json`. The receipt must bind to the exact intake bytes, preserve the R7A source identities, record a clean qualified acquisition, and still state that replay / repeatability have not occurred.

The operational replay-plan runner requires:

- `TEI_R7B_EXECUTION_INTAKE`
- `TEI_R7B_INTAKE_RECEIPT`
- `TEI_R7B_REPLAY_PLAN_OUTPUT_DIR`

All source and output artifacts remain outside the Repository. The runner source-qualifies the replay-plan tool, derives `r7b-replay-plan.json`, fingerprints it, and writes `r7b-replay-plan-receipt.json` with `replayExecuted=false` and `repeatabilityObserved=false`.

The plan pins:

- exact execution-intake SHA-256
- exact execution-intake receipt SHA-256
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
  + exact intake receipt
  -> replay plan
  != credentials
  != execution authority
  != replay execution
```

`RequestFixtureID` and `TargetLocalInteraction` are Target-local operational material. They do not become Canonical Application identity or Semantic Operation authority.

The first replay plan pins `ReplayTargetBuildRevision` to the origin Gate C Target build `T`. A cross-revision replay requires a different explicit plan / Evidence identity.

## 11. Origin Gate C Evidence locator

The actual Gate C Evidence used to establish the R7 origin is retained outside the Repository source tree as a GitHub Release artifact.

Current locator:

- Release tag: `gate-c-evidence-T-0ba87a4`
- Release asset: `tei-gate-c-evidence-T-0ba87a4e.tar.gz`
- archive SHA-256: `de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3`
- Target build: `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`
- archive manifest: `MANIFEST.sha256`

The archive contains the origin raw Functional artifacts required by the R7B plan, including:

- `read.json`
- `submit-success.json`
- `submit-authentication-denied.json`
- `submit-authorization-denied.json`
- `submit-validation-not-satisfied.json`
- `submit-execution-error.json`

The archive also contains the Semantic / Functional summary and final Gate C readiness artifact. Credential material is not part of the retained Evidence archive.

An operational R7B replay run must not trust an arbitrary extracted directory merely because the expected filenames exist. Before replay execution it must:

1. verify the archive exact SHA-256 against the recorded locator,
2. extract outside the Repository,
3. verify `MANIFEST.sha256`,
4. verify the six origin raw Functional artifact exact SHA-256 values against the replay plan,
5. reject missing, duplicate, replaced, or mismatched origin artifacts.

The GitHub Release is an Evidence locator and retention mechanism. It does not become Canonical Application state.

## 12. Operational replay runner contract

The first operational replay runner should execute against the exact origin Target build `T`, not the current `main` build.

The current Repository can continue to advance independently. Replay execution therefore uses an isolated detached worktree or equivalent clean checkout pinned to the replay plan's `ReplayTargetBuildRevision`.

Conceptually:

```text
current main
  -> verify current replay-runner source
  -> verify replay plan / replay-plan receipt
  -> verify retained Gate C origin Evidence archive
  -> create isolated clean checkout at exact T
  -> replay run 1 against T
  -> stop Target
  -> replay run 2 against T
  -> stop Target
  -> audit both runs
  -> repeatability assessment
```

The isolated Target checkout must not rewrite or temporarily checkout the developer's current `main` worktree. Any temporary worktree and all replay output belong outside the Repository and must be removed or explicitly retained as operational state after the run.

### 12.1 Required operational inputs

The runner contract should require explicit paths for:

- exact `r7b-replay-plan.json`
- exact `r7b-replay-plan-receipt.json`
- retained Gate C Evidence archive above
- a new Repository-external output directory

The replay-plan receipt must bind to the exact replay-plan bytes and must still state:

```text
replayExecuted = false
repeatabilityObserved = false
```

before execution starts.

### 12.2 Independent runs

`RequiredDistinctRuns = 2` means two distinct execution runs, not two copies of one captured response.

For the First Web Target the minimum operational form is:

- fresh Target process for run 1,
- all six planned behavior units executed,
- Target process stopped,
- fresh Target process for run 2,
- all six planned behavior units executed again.

Each run has its own run identity and raw artifact directory.

A single long-lived server with the same response copied twice is not repeatability Evidence.

### 12.3 Target-local fixture resolution

The replay plan carries named fixture identities rather than credentials or request bodies.

The runner may resolve those fixture identities to the same synthetic Reference inputs used by the Gate C acquisition path. Fixture resolution remains Target-local operational code.

The runner must not:

- serialize credentials into the replay plan,
- save request credentials in replay artifacts,
- reinterpret a fixture name as Canonical business meaning,
- infer current Authorization authority from historical Gate C success.

### 12.4 Observation and semantic validation

Every behavior observation must record at least:

- behavior identity,
- replay run identity,
- replay Target build revision,
- observed HTTP status,
- raw response artifact filename and exact SHA-256,
- expected semantic Observation class,
- semantic validation pass/fail.

Semantic validation should reuse the existing First Web Target Gate C semantic checks rather than introducing a second independent definition of what authentication denial, authorization denial, Validation not-satisfied, Command success, or Execution Error means.

If implementation requires code sharing, factor the existing validation predicates into a reusable helper rather than allowing the Gate C auditor and R7B auditor to drift independently.

### 12.5 Repeatability assessment

For each behavior, repeatability is established only when at least two distinct runs:

- use the same explicit replay Target build,
- satisfy the planned transport boundary,
- satisfy the same semantic Observation class,
- retain an exact SHA-256 for each raw replay artifact.

Exact response bytes may differ. Byte equality is neither required nor sufficient.

The final assessment remains:

```text
CoverageClass = behavior-unit
DeepWorkflowCovered = false
```

even if all six behavior units are repeatable.

### 12.6 Fail-closed conditions

The operational runner must fail closed at least for:

- dirty current source checkout before source qualification,
- replay plan / receipt digest mismatch,
- origin Evidence archive digest mismatch,
- `MANIFEST.sha256` failure,
- origin raw artifact digest mismatch,
- replay Target revision different from explicit T,
- failure to create a clean isolated T checkout,
- duplicate replay run identity,
- fewer than two completed runs,
- missing raw replay artifact,
- replay raw artifact without a valid SHA-256 identity,
- HTTP boundary mismatch,
- semantic classification mismatch,
- credentials detected in retained replay artifacts,
- Repository source changes during the operational run.

### 12.7 Expected output shape

A completed operational acquisition should retain, outside the Repository, a structure conceptually equivalent to:

```text
r7b-replay/
  source/
    replay-plan identity
    origin-evidence identity
  run-01/
    <six raw response artifacts>
    run-observation.json
  run-02/
    <six raw response artifacts>
    run-observation.json
  repeatability-summary.json
  r7b-replay-receipt.json
  source-tests.log
```

The final receipt records the exact source identities, replay Target build, run identities, raw artifact identities, and whether repeatability was actually observed.

It must not claim whole-workflow coverage or current execution authority.

## 13. Implementation / verification gate

The next code increment is the operational replay runner plus the smallest audit source needed to turn the two raw runs into the observation / repeatability summary above.

Under the current Repository development flow, code is committed directly to `main`; no new remote work branch or PR is created.

Because local health checking is the only pre-push verification path, this code increment is eligible to enter `main` only after:

```text
tools/repository-health-check.sh
```

passes in a Go-enabled full checkout. Source-only reasoning is not a substitute for that check.

## 14. Evidence height

This document, the test-only source, the intake CLI source, and the operational intake runner are Source / Design Evidence until the Repository-external artifacts are actually supplied and the runner is executed.

They do not claim:

- actual R7A handoff artifact acquisition in this Repository session
- actual R7B execution-intake acquisition
- actual replay execution
- actual replay PASS
- actual repeatability
- complete workflow coverage

Actual replay execution remains blocked until the Repository-external R7A handoff and acquisition receipt are supplied, exact-byte verified, and accepted by the R7B intake path.

## 15. Next step

The Source path through execution intake and replay-plan materialization is defined. The next implementation milestone is now narrower:

1. implement the operational replay runner against an isolated exact-T checkout,
2. reuse or factor the existing Gate C semantic validators so R7B does not create a second semantic truth,
3. verify the retained Gate C Release archive and six origin raw artifact identities,
4. execute two distinct runs of all six behavior units,
5. retain every raw replay artifact outside the Repository,
6. emit per-run observations and a behavior-unit repeatability summary,
7. run the Repository health check before the code enters `main`,
8. only after actual artifacts exist, update the recorded Evidence height from Source / Design to actual replay / repeatability Evidence.

Until those steps are executed, actual replay PASS and repeatability remain unclaimed.
