package counterexamples

import (
	"fmt"
)

// OldArtifactPollutionScenario demonstrates a false positive where an old
// artifact from a previous run is incorrectly considered valid because
// the cache key matches but the content is stale.
func OldArtifactPollutionScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-01",
		Name:           "Old artifact pollution from stale cache",
		Description:    "An old artifact remains in the cache directory after a source change; the incremental run incorrectly reuses it.",
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
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			// Incremental run incorrectly reuses old artifact from S0.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:S1",
				SourceIndexFingerprint: "si:S1",
				Inputs:                 []string{"source:A"},
				Dependencies:           []DependencyRef{{ID: "generator", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: true, SourceFingerprint: "source:S0"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-s0", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-1"},
				DurationMs:             50,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-01",
				ScenarioName:   "Old artifact pollution from stale cache",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-s1", Actual: "sha256:order-s0", Severity: "error"},
				},
				Notes: []string{"old artifact from S0 incorrectly reused for S1 input"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// OldArtifactPollutionName returns the scenario name.
func OldArtifactPollutionName() string {
	return "CE-01: Old artifact pollution from stale cache"
}

// OldArtifactPollutionDiagnostic generates a diagnostic report for CE-01.
func OldArtifactPollutionDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-01
Status: FAIL
Summary: 1 error, 0 warnings, 1 record
[error] STALE_ARTIFACT_REUSED: incremental run reused old artifact from previous source version
    expectedDigest = sha256:order-s1
    actualDigest = sha256:order-s0
    sourceVersion = S1
    cachedSource = S0
[warning] CACHE_KEY_COLLISION: cache key matches but source fingerprint differs
    cacheKey = artifact:main
[info] DIAGNOSIS: old artifact pollution: source changed from S0 to S1 but stale cache entry was reused
`)
}
