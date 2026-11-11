package counterexamples

import (
	"fmt"
)

// SourceInspectionAfterChangeScenario demonstrates a false positive where
// source is inspected before a change, then the change occurs, but the cache
// entry from before the change is incorrectly reused.
func SourceInspectionAfterChangeScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-04",
		Name:           "Source inspection after change causes stale cache reuse",
		Description:    "Source is inspected and cached at time T0, then changed at T1; the incremental run incorrectly uses the T0 cache entry.",
		ComparisonMode: "byte-exact",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:S1",
				SourceIndexFingerprint: "si:S1",
				Inputs:                 []string{"source:A"},
				Dependencies:           []DependencyRef{{ID: "generator", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-s1", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-s1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			// Incremental run uses cache from before source change.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:S1",
				SourceIndexFingerprint: "si:S1",
				Inputs:                 []string{"source:A"},
				Dependencies:           []DependencyRef{{ID: "generator", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: true, SourceFingerprint: "source:A:t0"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-s0", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-s0"},
				DurationMs:             50,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-04",
				ScenarioName:   "Source inspection after change causes stale cache reuse",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-s1", Actual: "sha256:order-s0", Severity: "error"},
					{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:main reused with stale source fingerprint source:A:t0 in incremental run only", Expected: "not reused", Actual: "reused", Severity: "error"},
				},
				Notes: []string{"source changed after inspection but cache entry from before change was reused"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// SourceInspectionAfterChangeName returns the scenario name.
func SourceInspectionAfterChangeName() string {
	return "CE-04: Source inspection after change causes stale cache reuse"
}

// SourceInspectionAfterChangeDiagnostic generates a diagnostic report for CE-04.
func SourceInspectionAfterChangeDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-04
Status: FAIL
Summary: 2 errors, 0 warnings, 2 records
[error] STALE_SOURCE_CACHE_REUSED: cache entry from before source change was reused after change
    expectedSourceTime = t1
    cachedSourceTime = t0
    sourceFingerprint = source:A
[error] SOURCE_TIME_MISMATCH: source time point in cache does not match current source time
    cachedTime = t0
    currentTime = t1
[info] DIAGNOSIS: source inspection after change: source was inspected at T0, changed at T1, but T0 cache was reused
`)
}
