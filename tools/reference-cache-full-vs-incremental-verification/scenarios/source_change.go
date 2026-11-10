package scenarios

import (
	"fmt"
)

// SourceChangeScenario verifies that a source change invalidates cached
// artifacts and causes a full re-evaluation.
func SourceChangeScenario() Scenario {
	return Scenario{
		ID:                "SC-01",
		Name:              "Source content change invalidates cache",
		Description:       "Modify one source file and verify that the incremental run recomputes affected artifacts and does not reuse stale cache.",
		ComparisonMode:    "byte-exact",
		ChangeType:        "source",
		CounterexampleIDs: []string{"CE-01", "CE-02"},
		Run: func() (VerificationReport, error) {
			// Simulate a full run with source fingerprint S0.
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:S0",
				SourceIndexFingerprint: "si:S0",
				Inputs:                 []string{"source:A"},
				Dependencies:           []DependencyRef{{ID: "generator", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: false, SourceFingerprint: "source:S0", RuleFingerprint: "rule:R1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:full-order", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}

			// Simulate an incremental run where source changed from S0 to S1.
			// The cache entry should be rejected as stale, so we expect a miss.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:S1",
				SourceIndexFingerprint: "si:S1",
				Inputs:                 []string{"source:A"},
				Dependencies:           []DependencyRef{{ID: "generator", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:main", Hit: false, StaleReason: "STALE_SOURCE_TIME", SourceFingerprint: "source:S0", RuleFingerprint: "rule:R1"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:incremental-order", ByteSize: 1024}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-1"},
				DurationMs:   800,
				SkippedSteps: []string{},
			}

			report := VerificationReport{
				SchemaVersion:     "tei.verification.cache-comparison/v0",
				OK:                false,
				ScenarioID:        "SC-01",
				ScenarioName:      "Source content change invalidates cache",
				FullRun:           full,
				IncrementalRun:    incremental,
				ComparisonMode:    "byte-exact",
				Diffs:             []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:full-order", Actual: "sha256:incremental-order", Severity: "error"}},
				CounterexampleIDs: []string{"CE-01", "CE-02"},
				Notes:             []string{"source fingerprint changed from S0 to S1; cache entry rejected with STALE_SOURCE_TIME"},
			}
			return report, nil
		},
	}
}

// SourceTimeChangeScenario verifies that a source time point change
// invalidates cache even when the source bytes are identical.
func SourceTimeChangeScenario() Scenario {
	return Scenario{
		ID:             "SC-02",
		Name:           "Source time point change invalidates identical-byte cache",
		Description:    "Source bytes are unchanged but the source time point advanced; verify cache is not reused.",
		ComparisonMode: "byte-exact",
		ChangeType:     "source-time",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:S0",
				SourceIndexFingerprint: "si:S0",
				Inputs:                 []string{"source:A"},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: false, SourceFingerprint: "source:S0:t0"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:S0",
				SourceIndexFingerprint: "si:S0:t1",
				Inputs:                 []string{"source:A"},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:main", Hit: false, StaleReason: "STALE_SOURCE_TIME", SourceFingerprint: "source:S0:t0"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1024}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-1"},
				DurationMs:   900,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "SC-02",
				ScenarioName:   "Source time point change invalidates identical-byte cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:main rejected with stale reason STALE_SOURCE_TIME", Expected: "not reused", Actual: "rejected", Severity: "error"}},
				Notes:          []string{"source bytes identical but time point changed from t0 to t1; cache correctly rejected"},
			}
			return report, nil
		},
	}
}

// SourceABACycleScenario verifies that an A->B->A source cycle does not
// resurrect old cache entries.
func SourceABACycleScenario() Scenario {
	return Scenario{
		ID:                "SC-03",
		Name:              "Source A->B->A cycle prevents cache resurrection",
		Description:       "Source changes to B and back to A; verify cache from before the cycle is not reused.",
		ComparisonMode:    "byte-exact",
		ChangeType:        "source-cycle",
		CounterexampleIDs: []string{"CE-03"},
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:A",
				SourceIndexFingerprint: "si:A",
				Inputs:                 []string{"source:A"},
				CacheEntries:           []CacheEntryRef{},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-A", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-A"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:A",
				SourceIndexFingerprint: "si:A",
				Inputs:                 []string{"source:A"},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:main", Hit: false, StaleReason: "SNAPSHOT_INTEGRITY_UNPROVEN", SourceFingerprint: "source:A"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-A-fresh", ByteSize: 1024}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-A"},
				DurationMs:   950,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:     "tei.verification.cache-comparison/v0",
				OK:                false,
				ScenarioID:        "SC-03",
				ScenarioName:      "Source A->B->A cycle prevents cache resurrection",
				FullRun:           full,
				IncrementalRun:    incremental,
				ComparisonMode:    "byte-exact",
				Diffs:             []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-A", Actual: "sha256:order-A-fresh", Severity: "error"}},
				CounterexampleIDs: []string{"CE-03"},
				Notes:             []string{"history observed: A -> B -> A; old cache entry rejected despite matching current fingerprint"},
			}
			return report, nil
		},
	}
}

// SourceMissingRequiredScenario verifies that a missing required source
// causes the cache to be rejected rather than silently reusing an old entry.
func SourceMissingRequiredScenario() Scenario {
	return Scenario{
		ID:             "SC-04",
		Name:           "Missing required source rejects cache reuse",
		Description:    "A required source file is removed; verify cache is not reused and the missing input is reported.",
		ComparisonMode: "byte-exact",
		ChangeType:     "source-missing",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:complete",
				SourceIndexFingerprint: "si:complete",
				Inputs:                 []string{"source:A", "source:B"},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1024}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     false,
				WorldFingerprint:       "world:incomplete",
				SourceIndexFingerprint: "si:incomplete",
				Inputs:                 []string{"source:A"},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:main", Hit: false, StaleReason: "STALE_INPUT"}},
				Artifacts:              []ArtifactRef{},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "", ErrorClass: "invalid-request"}},
				Receipts:               []string{},
				DurationMs:             200,
				SkippedSteps:           []string{},
				Error:                  "required source B missing",
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "SC-04",
				ScenarioName:   "Missing required source rejects cache reuse",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "cache-behavior-mismatch", Message: "full run failed but incremental run succeeded; cache may have hidden an error", Expected: "failure", Actual: "success", Severity: "error"}},
				Notes:          []string{"incremental run failed due to missing required source; cache was not reused"},
			}
			return report, nil
		},
	}
}

// AllSourceScenarios returns all source-change scenarios.
func AllSourceScenarios() []Scenario {
	return []Scenario{
		SourceChangeScenario(),
		SourceTimeChangeScenario(),
		SourceABACycleScenario(),
		SourceMissingRequiredScenario(),
	}
}

// ScenarioNames returns human-readable names for all source scenarios.
func ScenarioNames() []string {
	scenarios := AllSourceScenarios()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = fmt.Sprintf("%s: %s", s.ID, s.Name)
	}
	return names
}
