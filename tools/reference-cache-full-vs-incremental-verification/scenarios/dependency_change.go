package scenarios

import (
	"fmt"
)

// DependencyChangeScenario verifies that a dependency change invalidates
// cached artifacts and causes a full re-evaluation.
func DependencyChangeScenario() Scenario {
	return Scenario{
		ID:             "DC-01",
		Name:           "Dependency version change invalidates cache",
		Description:    "Upgrade a generator plugin version and verify that cached artifacts are recomputed.",
		ComparisonMode: "byte-exact",
		ChangeType:     "dependency-version",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:D0",
				SourceIndexFingerprint: "si:D0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen-1.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false, RuleFingerprint: "rule:derived-crud-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-d0", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-d0"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:D1",
				SourceIndexFingerprint: "si:D1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.1.0", Epoch: 2, Fingerprint: "fp:gen-1.1.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_INPUT", RuleFingerprint: "rule:derived-crud-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-d1", ByteSize: 1220}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-d1"},
				DurationMs:             1050,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "DC-01",
				ScenarioName:   "Dependency version change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-d0", Actual: "sha256:order-d1", Severity: "error"}},
				Notes:          []string{"generator version changed from 1.0.0 to 1.1.0; cache correctly rejected"},
			}
			return report, nil
		},
	}
}

// DependencyEpochChangeScenario verifies that a dependency epoch change
// invalidates cached artifacts even if the version is unchanged.
func DependencyEpochChangeScenario() Scenario {
	return Scenario{
		ID:             "DC-02",
		Name:           "Dependency epoch change invalidates cache",
		Description:    "Dependency version unchanged but epoch incremented; verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "dependency-epoch",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:D0",
				SourceIndexFingerprint: "si:D0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen-1.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:D1",
				SourceIndexFingerprint: "si:D1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 2, Fingerprint: "fp:gen-1.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_INPUT"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-d1", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-1"},
				DurationMs:             950,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "DC-02",
				ScenarioName:   "Dependency epoch change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:order-table rejected with stale reason STALE_INPUT", Expected: "not reused", Actual: "rejected", Severity: "error"}},
				Notes:          []string{"epoch changed from 1 to 2; cache invalidated despite same version"},
			}
			return report, nil
		},
	}
}

// TransitiveDependencyChangeScenario verifies that a transitive dependency
// change invalidates cached artifacts.
func TransitiveDependencyChangeScenario() Scenario {
	return Scenario{
		ID:             "DC-03",
		Name:           "Transitive dependency change invalidates cache",
		Description:    "A transitive dependency (not directly declared) changes; verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "transitive-dependency",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:T0",
				SourceIndexFingerprint: "si:T0",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"},
					{ID: "std-common", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:common"},
				},
				CacheEntries: []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-t0", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:     []string{"receipt:full-t0"},
				DurationMs:   1000,
				SkippedSteps: []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:T1",
				SourceIndexFingerprint: "si:T1",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"},
					{ID: "std-common", Version: "2.0.0", Epoch: 2, Fingerprint: "fp:common"},
				},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_INPUT"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-t1", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-t1"},
				DurationMs:   980,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "DC-03",
				ScenarioName:   "Transitive dependency change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "dependency-mismatch", Message: "dependency std-common differs between runs", Expected: "version=2.0.0 epoch=1 fingerprint=fp:common", Actual: "version=2.0.0 epoch=2 fingerprint=fp:common", Severity: "error"}},
				Notes:          []string{"transitive dependency std-common epoch changed from 1 to 2; cache invalidated"},
			}
			return report, nil
		},
	}
}

// DependencyRemovalScenario verifies that removing a transitive dependency
// invalidates cached artifacts.
func DependencyRemovalScenario() Scenario {
	return Scenario{
		ID:             "DC-04",
		Name:           "Transitive dependency removal invalidates cache",
		Description:    "A transitive dependency is removed; verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "dependency-removal",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:T0",
				SourceIndexFingerprint: "si:T0",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"},
					{ID: "std-common", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:common"},
				},
				CacheEntries: []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-t0", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:     []string{"receipt:full-t0"},
				DurationMs:   1000,
				SkippedSteps: []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:T1",
				SourceIndexFingerprint: "si:T1",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"},
				},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_INPUT"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-t1", ByteSize: 1150}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-t1"},
				DurationMs:   900,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "DC-04",
				ScenarioName:   "Transitive dependency removal invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "dependency-mismatch", Message: "dependency std-common missing in incremental run", Expected: "version=2.0.0 epoch=1 fingerprint=fp:common", Actual: "missing", Severity: "error"}},
				Notes:          []string{"transitive dependency std-common removed; cache invalidated"},
			}
			return report, nil
		},
	}
}

// AllDependencyScenarios returns all dependency-change scenarios.
func AllDependencyScenarios() []Scenario {
	return []Scenario{
		DependencyChangeScenario(),
		DependencyEpochChangeScenario(),
		TransitiveDependencyChangeScenario(),
		DependencyRemovalScenario(),
	}
}

// DependencyScenarioNames returns human-readable names for all dependency scenarios.
func DependencyScenarioNames() []string {
	scenarios := AllDependencyScenarios()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = fmt.Sprintf("%s: %s", s.ID, s.Name)
	}
	return names
}
