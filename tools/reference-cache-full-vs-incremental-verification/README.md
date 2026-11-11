# Cache Full vs Incremental Verification Reference

This directory contains a read-only reference implementation for verifying that cached/incremental processing produces the same results as full (non-cached) processing.

## Purpose

The verification framework compares full processing against cached/incremental processing to detect false positives where cache reuse produces results that diverge from a deterministic full run. It produces diagnostics that identify the exact input, dependency, cache entry, and artifact that differ.

## Structure

```
tools/reference-cache-full-vs-incremental-verification/
├── verification.go              # Core types and contracts
├── comparison.go                # Artifact, dependency, input, cache, evidence, and receipt comparison
├── diagnostics.go               # Diagnostic report generation and output
├── scenarios/                   # Test scenarios
│   ├── source_change.go
│   ├── model_change.go
│   ├── dependency_change.go
│   ├── operator_plugin_change.go
│   ├── toolchain_change.go
│   └── receipt_evidence_consistency.go
├── counterexamples/             # Counterexample scenarios for false positives
│   ├── old_artifact_pollution.go
│   ├── incomplete_dependency.go
│   ├── cache_contamination.go
│   ├── source_inspection_after_change.go
│   ├── intermediate_failure.go
│   └── toolchain_reuse.go
└── verification_test.go         # Test runner
```

## Comparison Modes

- **Byte-exact**: Compares raw bytes. Appropriate for artifacts whose content is fully determined by byte-identical inputs and deterministic generation (e.g., compiled binaries, checksum lists).
- **Structural**: Compares normalized JSON structures. Appropriate for artifacts whose serialization order, whitespace, or key casing is not semantically meaningful.
- **Semantic**: Compares after applying domain-specific normalization rules (e.g., timestamp stripping, UUID anonymization, path prefix normalization). Appropriate when the same meaning can be expressed by different bytes.

## Test Scenarios

### Source Changes
- SC-01: Source content change invalidates cache
- SC-02: Source time point change invalidates identical-byte cache
- SC-03: Source A->B->A cycle prevents cache resurrection
- SC-04: Missing required source rejects cache reuse

### Model Changes
- MC-01: Model definition change invalidates cache
- MC-02: Model version bump invalidates cache
- MC-03: Semantically equivalent models produce identical artifacts

### Dependency Changes
- DC-01: Dependency version change invalidates cache
- DC-02: Dependency epoch change invalidates cache
- DC-03: Transitive dependency change invalidates cache
- DC-04: Transitive dependency removal invalidates cache

### Operator/Plugin Changes
- OC-01: Operator definition change invalidates cache
- OC-02: Plugin definition change invalidates cache
- OC-03: Unused plugin addition does not invalidate unrelated cache
- OC-04: Plugin lock fingerprint change invalidates cache

### Toolchain Changes
- TC-01: Toolchain change invalidates cache
- TC-02: Same toolchain version with different build flags invalidates cache
- TC-03: Toolchain reuse across different inputs is correctly rejected

### Receipt/Evidence Consistency
- RC-01: Receipt mismatch is detected
- RC-02: Evidence mismatch is detected
- RC-03: Consistent receipts and evidence pass verification

## Counterexamples

The counterexamples demonstrate false positives that the verification framework must detect:

- CE-01: Old artifact pollution from stale cache
- CE-02: Incomplete dependency graph causes false cache reuse
- CE-03: Cache contamination across tenants
- CE-04: Source inspection after change causes stale cache reuse
- CE-05: Intermediate failure causes incomplete artifact cache reuse
- CE-06: Toolchain artifact reuse across different toolchains

## Running the Tests

```bash
go test ./tools/reference-cache-full-vs-incremental-verification -v
```

## Diagnostic Output

When verification fails, the framework produces a diagnostic report that identifies:
- Which input, dependency, cache entry, and artifact differ
- The expected vs actual values
- The severity of the difference
- A human-readable summary

## Limitations

This is a read-only reference implementation. It does not:
- Modify sources, caches, or artifacts
- Execute actual builds or transformations
- Connect to real caches or artifact stores
- Perform network operations

Its purpose is to define the comparison contracts and diagnostic formats that a production implementation would use.
