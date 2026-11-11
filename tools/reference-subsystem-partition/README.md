# Subsystem partition: fixed-owner / must-link exact reference

Status: experimental, **read-only, caller-supplied candidate-only**. Issue [#3250](https://github.com/hat47x/tei-platform/issues/3250); design [Draft #3255](https://github.com/hat47x/tei-platform/pull/3255). This is not an Accepted Source adapter or a product optimizer.

## Purpose and exact result

The Go package exposes `Evaluate(Problem)` for the **existing** fixed-owner exact reference, plus the following candidate-generation extension under the new files in this package:

- `CoverageReport` / `BuildCoverageReport` (`coverage.go`): subject-level provenance status including `declared-source`, `parsed-extracted`, `derived-potential`, `inferred-indirect`, `observed-effect`, `unresolved`, and `not-examined`.
- `AccessGraph` / `BuildAccessGraph` (`graph.go`): Function-Data bipartite graph with Read edges and Write (C/U/D) edges split into separate sets. Delegation (Invocation) edges are captured separately and are **not** counted as direct writes.
- `CandidateProblem`, `GenerateCandidates`, `ParetoFront`, `EvaluateCandidates` (`generate.go`, `evaluate.go`): deterministic candidate generation (exhaustive for N<=10), ranked output, multi-objective breakdown (cross-write count/weight, cross-read count, cohesion, max size), and Pareto front. Heuristic mode is labeled `candidate-ranked`.

All new outputs include `authorityLimit` and are **candidate-only**; they do not resolve accepted ownership, grant authorization, or infer runtime effects. Callers must not omit constraints (capacity, cannot-link, shared-update) and then present the result as globally optimal.

## Scope and authority boundaries

The **candidate input is not a qualified Source**. A caller-supplied Owner/Function I/O relation or Source fingerprint is not a business owner acceptance. If a Data ID denotes two different scopes, the caller must first disambiguate it by the existing TEI typed identity contract. This reference intentionally does not implement another Source Index, parser, Graph, Owner policy, acceptance store, optimizer DSL, CLI, transport or runtime.

Use [tools/reference-subsystem-cud](../reference-subsystem-cud/README.md) for the existing read-only cross-view candidate diagnostic, not this evaluator. Use the accepted Source/Identity/Acceptance contracts (#1120, #267) and the latest three-view read-admission work ([Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)) before binding real TEI ownership/Function I/O to this input. The Draft's synthetic or limited Inspector read is not proof of Accepted Subsystem Owner or whole-program direct-write closure.

Any result returned to Studio, CLI, MCP, AI or another consumer must first use a **currently qualified, disclosure-safe Public Slice**. This pure package has no authorization or confidentiality enforcement: it can reveal candidate membership, scores and hidden edge identifiers. Filtering nodes from a private-world optimizer result is not sufficient; see #2516/#1124.

No output represents actual CRUD effects, a Tx receipt, a Source-changing proposal, a migration cutover or formal Owner approval. Do not use it to issue writes. Owners for application/operation/Source, Studio UI, independent expected, disclosure, and runtime execution remain separate.

## Test and qualification

Run at the repository root:

    go test ./tools/reference-subsystem-partition -count=1
    go vet ./tools/reference-subsystem-partition
    gofmt -l tools/reference-subsystem-partition

Tests include the design cases SP-37/38/39/41, a must-link counterexample in which independent lower bound 0 is not feasible, deterministic tie-breaking, input rejection, and an **independent exhaustive assignment oracle** over 1,372 small allow-set/must-link combinations. The exhaustive test does not call the evaluator's component-contraction procedure.

The new candidate-generation tests use a fixed fixture in `testdata/fixture.json` with an independently authored expected JSON `testdata/fixture-expected.json`. The expected values are **hand-written** and are not generated from the implementation output.

Further needed (not implemented by this package): Source closure and actual Accepted owner/Function I/O producer (SP-42), Current Grant and public noninterference (SP-31/43), Boundary Repair/Effector/Tx/Recovery (SP-44), and trusted full repository-health-check on the exact merged revision. Local/CI test PASS is not evidence for those obligations.

## S6-S7 (added for #4413)

- `ComparisonReport` / `BuildComparisonReport` (`compare.go`): deterministic before/after comparison table of candidates (cross-boundary counts/weights, cohesion, max size).
- `ProjectionReport` / `BuildProjectionReport` (`project.go`): per-subsystem projection of Function, write/read edges, and invocation dependencies. Read-only and derived-only.

These are preview outputs and do not change accepted ownership, grant authorization, or expose runtime effects.

**No main integration or Owner Meaning Acceptance is implied by creating this reference.**
