# Subsystem CUD / Procedure cross-view pressure (Reference-only)

This is an **experimental read-only diagnostic**, scoped to Issue #1132 and the parent #1119. It consumes the semantic Data and Application Pattern identities from a caller-supplied `tei world` JSON snapshot, plus **explicit candidate** ownership, Function-level declared I/O, and Procedure-step facts. The latter facts are never accepted Canonical meaning merely because the probe reads them.

Run it with:

```sh
go run ./tools/reference-subsystem-cud -world /path/to/world.json -facts /path/to/facts.json
```

`facts.json` must use `schemaVersion: "subsystem-cud-candidate-v1"` and contain:

- `owners`: `{data, subsystem, source}` tuples.
- `functionIO`: `{pattern, subsystem, data, operation, source}`; operation is exactly one of C/R/U/D.
- `procedureSteps`: `{id, subsystem, realization, pattern?, source}` with realization `system`, `manual`, or `external`.

`source` is a local, non-secret **evidence locator supplied by the caller**, not proof that the declaration is Accepted. The tool never reads database rows, invokes a Capability, rewrites Source, or makes authorization and transactional claims. **A clean report does not prove business ownership correctness or actual I/O effects.** Unknown references, duplicated ownership, unknown operation codes, missing manual/system bindings, empty candidate fact sets, and mismatched schema stay unresolved. The report includes explicit counts for the World identities and supplied candidate facts; these counts do not establish semantic coverage of the application. A declared cross-subsystem C/U/D prompts review; R remains possible without an ownership violation. Delegation is modeled as a distinct owner-side function rather than a self-asserted bypass flag.

This reference intentionally avoids a second Canonical model or an automatic adapter that would promote legacy observations to project truth. Next integration step: use the existing Source/Acceptance mechanisms to derive an exact, provenance-bound and accepted set of ownership and I/O facts from TEI, and combine the report with #1120 typed references and #1124 Studio diagnostics. Until then the result is **candidate inspection** only.

For local checks, `go test ./tools/reference-subsystem-cud` and `go vet ./tools/reference-subsystem-cud` should run without external Go libraries. Repository-wide tests and trusted `tools/repository-health-check.sh` must still be run against the exact PR head before merging.
