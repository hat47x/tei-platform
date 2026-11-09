# First Web Target — sharedprojection (E1 boundary)

This Go package owns **presentation-only translations** from already-resolved
TEI Driver query results to JSON-safe values. It is not a second Application,
Driver, Provider, or authorization authority.

It is imported by the legacy combined First Web Target through
`shared_semantic_projection.go`, which preserves old in-package type names
and function signatures as aliases and thin wrappers. New Application-specific
Go entries can import this package without importing the combined
`package main` or the other Application's business Runner.

## Public API

- `SemanticRowProjection`, `SemanticValueProjection`, and `RuntimeBindingProjection`
- `ProjectMappedRows`: source column order, Canonical attribute IDs, null values
- `ProjectContractRows`: stable ascending Canonical attribute IDs
- `ProjectEnvironmentBinding`: observed Environment/instance/Provider/Query binding

The functions only map values from already-executed Driver results. They do
not infer Pattern meaning, select Providers, run database commands, authorize
sessions, read the Workspace, or acquire Build Artifact identity.

## Verification and limitations

In a trusted, clean TEI checkout, run:

```sh
go test -count=1 ./tools/reference-first-web-target/sharedprojection
go test -count=1 ./tools/reference-first-web-target
go vet ./tools/reference-first-web-target/sharedprojection ./tools/reference-first-web-target
```

The package test verifies JSON shape, ordering, nil preservation, invalid
row widths, and the actual `go list -deps -json` closure. The latter must not
contain the combined First Web Target, Workspace Loader, legacy authentication,
or legacy authorization adapter.

A Go package boundary is **not** yet evidence that an Application-specific
binary excludes unused code. That requires separate compile-time entry points,
real `go build`/link artifacts and source/linked symbol observations, plus
Business Result/Execution Error/Observation/Authorization regression against
the accepted Handover B/R and Target build T.

Do not merge Draft PR #1122 without the repository's trusted exact-HEAD
qualification and clean pre/post receipts.
