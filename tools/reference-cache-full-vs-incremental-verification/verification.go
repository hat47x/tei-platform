// Package verification provides the core types and contracts for comparing
// full (non-cached) processing against cached/incremental processing.
//
// It is a read-only reference implementation that never modifies sources,
// caches, or artifacts. Its purpose is to detect false positives where
// cache reuse produces results that diverge from a deterministic full run,
// and to produce diagnostics that identify the exact input, dependency,
// cache entry, and artifact that differ.
package verification

import (
	"fmt"
)

// ComparisonMode selects how two generated artifacts are compared.
type ComparisonMode string

const (
	// ModeByteExact compares raw bytes. Appropriate for artifacts whose
	// content is fully determined by byte-identical inputs and deterministic
	// generation (e.g. compiled binaries, checksum lists).
	ModeByteExact ComparisonMode = "byte-exact"

	// ModeStructural compares normalized JSON structures. Appropriate for
	// artifacts whose serialization order, whitespace, or key casing is not
	// semantically meaningful.
	ModeStructural ComparisonMode = "structural"

	// ModeSemantic compares after applying domain-specific normalization
	// rules (e.g. timestamp stripping, UUID anonymization, path prefix
	// normalization). Appropriate when the same meaning can be expressed
	// by different bytes.
	ModeSemantic ComparisonMode = "semantic"
)

// ArtifactRef identifies one generated artifact inside a verification run.
type ArtifactRef struct {
	// Kind is the artifact kind (e.g. "table-definition", "ddl").
	Kind string `json:"kind"`
	// Generator is the identity of the generator that produced the artifact.
	Generator string `json:"generator"`
	// Path is the logical path or member name within the artifact set.
	Path string `json:"path"`
	// Digest is the content digest of the artifact.
	Digest string `json:"digest"`
	// ByteSize is the length of the artifact content in bytes.
	ByteSize int `json:"byteSize"`
}

// DependencyRef identifies one dependency observed during a run.
type DependencyRef struct {
	// ID is the stable dependency identifier (e.g. plugin ID, provider ID).
	ID string `json:"id"`
	// Version is the dependency version observed at run time.
	Version string `json:"version"`
	// Epoch is the dependency epoch observed at run time.
	Epoch uint64 `json:"epoch"`
	// Fingerprint is the dependency fingerprint observed at run time.
	Fingerprint string `json:"fingerprint"`
}

// CacheEntryRef identifies one cache entry under test.
type CacheEntryRef struct {
	// Key is the cache key.
	Key string `json:"key"`
	// Hit reports whether the run observed a cache hit.
	Hit bool `json:"hit"`
	// StaleReason explains why the entry was rejected as stale, if any.
	StaleReason string `json:"staleReason,omitempty"`
	// SourceFingerprint is the source fingerprint bound to the cache entry.
	SourceFingerprint string `json:"sourceFingerprint,omitempty"`
	// RuleFingerprint is the rule fingerprint bound to the cache entry.
	RuleFingerprint string `json:"ruleFingerprint,omitempty"`
}

// EvidenceRef identifies one piece of execution evidence.
type EvidenceRef struct {
	// Operation is the operation kind ("execute" or "lookup").
	Operation string `json:"operation"`
	// Status is the observed status.
	Status string `json:"status"`
	// ErrorClass is the observed error class, if any.
	ErrorClass string `json:"errorClass,omitempty"`
	// ReceiptVersion is the receipt resource version, if any.
	ReceiptVersion uint64 `json:"receiptVersion,omitempty"`
}

// RunReport summarizes one execution path (full or incremental).
type RunReport struct {
	// Label distinguishes the run, e.g. "full" or "incremental".
	Label string `json:"label"`
	// OK reports whether the run completed without terminal error.
	OK bool `json:"ok"`
	// Error is the terminal error, if any.
	Error string `json:"error,omitempty"`
	// WorldFingerprint is the fingerprint of the canonical world used.
	WorldFingerprint string `json:"worldFingerprint"`
	// SourceIndexFingerprint is the fingerprint of the source index used.
	SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
	// Inputs is the set of observed input references.
	Inputs []string `json:"inputs"`
	// Dependencies is the set of observed dependency references.
	Dependencies []DependencyRef `json:"dependencies"`
	// CacheEntries is the set of observed cache entries.
	CacheEntries []CacheEntryRef `json:"cacheEntries"`
	// Artifacts is the set of produced artifacts.
	Artifacts []ArtifactRef `json:"artifacts"`
	// Evidence is the set of recorded evidence entries.
	Evidence []EvidenceRef `json:"evidence"`
	// Receipts is the set of observed receipts.
	Receipts []string `json:"receipts"`
	// DurationMs is the elapsed wall-clock time in milliseconds.
	DurationMs int64 `json:"durationMs"`
	// SkippedSteps lists the steps that were skipped due to cache reuse.
	SkippedSteps []string `json:"skippedSteps,omitempty"`
}

// DiffKind categorizes one difference between two runs.
type DiffKind string

const (
	// DiffArtifactByteMismatch means the artifact bytes differ.
	DiffArtifactByteMismatch DiffKind = "artifact-byte-mismatch"
	// DiffArtifactStructuralMismatch means the artifact structures differ after normalization.
	DiffArtifactStructuralMismatch DiffKind = "artifact-structural-mismatch"
	// DiffArtifactSemanticMismatch means the artifact semantics differ after domain normalization.
	DiffArtifactSemanticMismatch DiffKind = "artifact-semantic-mismatch"
	// DiffArtifactMissing means an artifact present in one run is missing in the other.
	DiffArtifactMissing DiffKind = "artifact-missing"
	// DiffDependencyMismatch means the dependency graph differs.
	DiffDependencyMismatch DiffKind = "dependency-mismatch"
	// DiffInputMismatch means the input closure differs.
	DiffInputMismatch DiffKind = "input-mismatch"
	// DiffCacheBehaviorMismatch means cache hit/skip behavior differs.
	DiffCacheBehaviorMismatch DiffKind = "cache-behavior-mismatch"
	// DiffEvidenceMismatch means the recorded evidence differs.
	DiffEvidenceMismatch DiffKind = "evidence-mismatch"
	// DiffReceiptMismatch means the observed receipts differ.
	DiffReceiptMismatch DiffKind = "receipt-mismatch"
)

// DiffEntry records one discovered difference.
type DiffEntry struct {
	// Kind is the category of difference.
	Kind DiffKind `json:"kind"`
	// ArtifactPath identifies the artifact involved, if any.
	ArtifactPath string `json:"artifactPath,omitempty"`
	// Expected is the value from the full (reference) run.
	Expected string `json:"expected,omitempty"`
	// Actual is the value from the incremental run.
	Actual string `json:"actual,omitempty"`
	// Message is a human-readable explanation.
	Message string `json:"message"`
	// Severity is the diagnostic severity.
	Severity string `json:"severity"`
}

// VerificationReport is the top-level output of a comparison.
type VerificationReport struct {
	// SchemaVersion identifies the report format.
	SchemaVersion string `json:"schemaVersion"`
	// OK reports whether the verification passed.
	OK bool `json:"ok"`
	// ScenarioID identifies the test scenario.
	ScenarioID string `json:"scenarioId"`
	// ScenarioName is a human-readable scenario name.
	ScenarioName string `json:"scenarioName"`
	// FullRun summarizes the full (non-cached) execution.
	FullRun RunReport `json:"fullRun"`
	// IncrementalRun summarizes the cached/incremental execution.
	IncrementalRun RunReport `json:"incrementalRun"`
	// ComparisonMode records the mode used for artifact comparison.
	ComparisonMode ComparisonMode `json:"comparisonMode"`
	// Diffs is the ordered list of discovered differences.
	Diffs []DiffEntry `json:"diffs"`
	// CounterexampleIDs lists the counterexample scenarios that were exercised.
	CounterexampleIDs []string `json:"counterexampleIds,omitempty"`
	// Notes contains non-normative observations.
	Notes []string `json:"notes,omitempty"`
}

// IsEmpty reports whether the verification found no differences.
func (r VerificationReport) IsEmpty() bool {
	return len(r.Diffs) == 0
}

// DiffSummary returns a one-line summary suitable for CI output.
func (r VerificationReport) DiffSummary() string {
	if r.OK {
		return fmt.Sprintf("PASS %s: full vs %s produced identical results (%s comparison)",
			r.FullRun.Label, r.IncrementalRun.Label, r.ComparisonMode)
	}
	return fmt.Sprintf("FAIL %s: %d differences detected between full and %s runs",
		r.ScenarioID, len(r.Diffs), r.IncrementalRun.Label)
}

// VerificationScenario defines one executable comparison scenario.
type VerificationScenario struct {
	// ID is a stable scenario identifier.
	ID string `json:"id"`
	// Name is a human-readable scenario name.
	Name string `json:"name"`
	// Description explains what the scenario verifies.
	Description string `json:"description"`
	// ComparisonMode selects the artifact comparison strategy.
	ComparisonMode ComparisonMode `json:"comparisonMode"`
	// ChangeType describes the kind of change injected.
	ChangeType string `json:"changeType"`
	// CounterexampleIDs links to counterexample scenarios.
	CounterexampleIDs []string `json:"counterexampleIds,omitempty"`
	// Run executes the scenario and returns a report.
	Run func() (VerificationReport, error) `json:"-"`
}

// AssertNoDiffs is a test helper that fails t when the report contains diffs.
func AssertNoDiffs(t interface {
	Helper()
	Fatalf(string, ...any)
}, report VerificationReport) {
	t.Helper()
	if !report.OK {
		t.Fatalf("verification failed:\n%s", report.DiffSummary())
	}
}

// AssertDiffCount is a test helper that fails t when the diff count is not n.
func AssertDiffCount(t interface {
	Helper()
	Fatalf(string, ...any)
}, report VerificationReport, n int) {
	t.Helper()
	if len(report.Diffs) != n {
		t.Fatalf("expected %d diffs, got %d:\n%s", n, len(report.Diffs), report.DiffSummary())
	}
}

// AssertDiffKind is a test helper that fails t when a diff with the given kind
// and artifact path is not present.
func AssertDiffKind(t interface {
	Helper()
	Fatalf(string, ...any)
}, report VerificationReport, kind DiffKind, artifactPath string) {
	t.Helper()
	for _, d := range report.Diffs {
		if d.Kind == kind && d.ArtifactPath == artifactPath {
			return
		}
	}
	t.Fatalf("expected diff kind=%q path=%q not found in %d diffs", kind, artifactPath, len(report.Diffs))
}
