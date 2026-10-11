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

## Exact-input accounting and non-inference (E0 hardening)

- The CLI attaches `inputFingerprints.worldJSON` and `inputFingerprints.factsJSON` (SHA-256 of **exact supplied bytes**) to its JSON report. This identifies an input snapshot for downstream review, but does **not** verify that the World was regenerated from current accepted Source or that candidate facts have been accepted.
- If a declared Function I/O points to a missing or ambiguous World Pattern, the tool reports `IO_PATTERN_UNRESOLVED` and **does not** infer `CROSS_SUBSYSTEM_WRITE_INTENT` from that entry. Data identity and owner must also be exact before write-boundary review; unknown values are not replaced with an arbitrary match.
- The source JSON has real `semantics[].data[].id` and `applications[].patterns[].id` arrays. This tool intentionally reads only those known fields; it does not derive owner, actual effect, or Function I/O from a Pattern name, `kind`, or `capability`.
- The typed Data/Relation/Pattern edge provider is owned by #1120 (Draft PR #1139, `modelref/derivedbridge`). When it is qualified, #1132 should consume those read-only edges by exact scoped identity rather than make another parallel graph representation. Accepted Subsystem ownership and explicitly scoped declared I/O still require a separate verified source/acceptance contract.
- A complete Source→World→typed edge→CUD diagnostic→Studio navigation path has **not** been run; the local isolated unit tests only qualify this tool's candidate logic.

## Stale-world rejection (E0 snapshot binding)

`facts.json` additionally **must** include `worldJSONFingerprint`, e.g. `"sha256:b21ed42b17ca10937dcf7020d27fce1546a389327375f2ceb958b0a09937d51c"` for the exact bytes of `testdata/world.json`. To prepare a candidate for your own snapshot, compute `sha256sum /path/to/world.json` and prefix the 64 hexadecimal characters with `sha256:`. This binding is deliberately byte-exact: reformatting JSON requires updating the candidate evidence.

- **Matching supplied World bytes**: the CLI runs the existing read-only candidate checks, recording both input digests in the report.
- **Missing fingerprint**: `WORLD_SNAPSHOT_UNBOUND` (`unresolved`, nonzero exit).
- **Changed World bytes**: `WORLD_SNAPSHOT_STALE` (`unresolved`, nonzero exit), even if the edited JSON still describes the same identities.
- **Malformed World/facts JSON**: input decoding fails before inspection. The CLI does not accept an unrelated pre-decoded `World` beside the digest bytes; `InspectSnapshot` decodes and inspects exactly the supplied bytes in one function. A pure `Inspect(w,f)` remains available for candidate-only unit tests but is **not** a snapshot-integrity check.

**This is consistency, not authenticity.** The same caller controls both `facts.json` and `world.json`, so a newly computed matching digest cannot establish that either comes from an accepted TEI Workspace. It is not a signature, Source Index conformance, Shared Acceptance authorization, or proof of Function-level/actual effects. #1120 should provide the shared typed Source/Graph evidence; #1132 still needs an accepted ownership/declaration producer, exact Workspace revision binding, and a real integration test before qualification.
