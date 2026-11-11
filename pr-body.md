## Summary

Implements the S1-S5 read-only candidate-generation prototype plus S6-S7 comparison and projection previews for logical subsystem partition (#3250 child). Adds a fixed fixture and hand-written independent expected JSON to validate deterministic multi-candidate generation with multi-objective breakdown.

## Base

`5cfc0f2d142c07132f7e9cf7405099386ca80ec1` (origin/main, 2026-11-11 21:13 JST)

## Scope (new files only)

- `tools/reference-subsystem-partition/coverage.go` + `coverage_test.go`
- `tools/reference-subsystem-partition/graph.go` + `graph_test.go`
- `tools/reference-subsystem-partition/constraints.go`
- `tools/reference-subsystem-partition/generate.go` + `generate_test.go`
- `tools/reference-subsystem-partition/evaluate.go`
- `tools/reference-subsystem-partition/compare.go` + `compare_test.go`
- `tools/reference-subsystem-partition/project.go`
- `tools/reference-subsystem-partition/provenance.go`
- `tools/reference-subsystem-partition/README.md` (extended)
- `tools/reference-subsystem-partition/testdata/fixture.json`
- `tools/reference-subsystem-partition/testdata/fixture-expected.json`

Unchanged: `partition.go`, `partition_test.go`, `workspace_intent_test.go`, `surface_writer_nonclosure_test.go`. PR #4388 / #4403 files untouched.

## Verification (run in worktree)

    gofmt -l tools/reference-subsystem-partition
    go vet ./tools/reference-subsystem-partition
    go test ./tools/reference-subsystem-partition -count=1

Result: all PASS (package ok).

## Design notes / remaining risks

- Candidate generation is exhaustive for N<=10 (`exhaustive-proven`). Larger N uses a heuristic labeled `candidate-ranked`. Status labels intentionally do NOT claim `exact` under capacity/cannot-link/shared-update.
- `GenerateCandidates` does NOT modify `partition.Evaluate` or its tests. The exact fixed-owner evaluator remains the authority for its supported constraint set.
- Read edges are separate from Write edges. Delegated Invocation edges are not counted as direct writes.
- Provenance labels include `declared-source`, `parsed-extracted`, `derived-potential`, `inferred-indirect`, `observed-effect`, `unresolved`, `not-examined`. Unknown is not treated as zero.
- `fixture-expected.json` is hand-written. Implementation output is not used to regenerate the expected file.
- Public Slice / disclosure-safe projection is not implemented here; see #2516/#1124.

## S6-S7

Added comparison report and per-subsystem projection report (read-only previews). Not a binding acceptance or ownership grant.
