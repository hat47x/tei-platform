package scenarios

import (
	"fmt"
)

// ToolchainChangeScenario verifies that a toolchain change invalidates cached
// artifacts and causes a full re-evaluation.
func ToolchainChangeScenario() Scenario {
	return Scenario{
		ID:             "TC-01",
		Name:           "Toolchain change invalidates cache",
		Description:    "Change the Go version or build flags and verify that cached artifacts are recomputed.",
		ComparisonMode: "byte-exact",
		ChangeType:     "toolchain",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:TC0",
				SourceIndexFingerprint: "si:TC0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.22.0", Epoch: 1, Fingerprint: "fp:go-1.22.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: false, RuleFingerprint: "rule:build-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-tc0", ByteSize: 20480000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-tc0"},
				DurationMs:             5000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:TC1",
				SourceIndexFingerprint: "si:TC1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.23.0", Epoch: 2, Fingerprint: "fp:go-1.23.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:build-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-tc1", ByteSize: 20500000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-tc1"},
				DurationMs:             5200,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "TC-01",
				ScenarioName:   "Toolchain change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "tei", Expected: "sha256:binary-tc0", Actual: "sha256:binary-tc1", Severity: "error"}},
				Notes:          []string{"toolchain changed from Go 1.22.0 to 1.23.0; binary artifact recomputed"},
			}
			return report, nil
		},
	}
}

// ToolchainSameVersionDifferentBuildScenario verifies that a toolchain with
// the same version but different build flags invalidates cache.
func ToolchainSameVersionDifferentBuildScenario() Scenario {
	return Scenario{
		ID:             "TC-02",
		Name:           "Same toolchain version with different build flags invalidates cache",
		Description:    "Toolchain version unchanged but build flags changed (e.g. -race added); verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "toolchain-flags",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:TC0",
				SourceIndexFingerprint: "si:TC0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.22.0", Epoch: 1, Fingerprint: "fp:go-1.22.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: false, RuleFingerprint: "rule:build-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-tc0", ByteSize: 20480000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-tc0"},
				DurationMs:             5000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:TC1",
				SourceIndexFingerprint: "si:TC1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.22.0", Epoch: 1, Fingerprint: "fp:go-1.22.0-race"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:build-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-tc1", ByteSize: 22000000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-tc1"},
				DurationMs:             5500,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "TC-02",
				ScenarioName:   "Same toolchain version with different build flags invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "tei", Expected: "sha256:binary-tc0", Actual: "sha256:binary-tc1", Severity: "error"}},
				Notes:          []string{"build flags changed; cache invalidated despite same Go version"},
			}
			return report, nil
		},
	}
}

// ToolchainReuseAcrossDifferentInputsScenario verifies that a toolchain
// cannot be reused across different inputs even if the toolchain itself
// is unchanged.
func ToolchainReuseAcrossDifferentInputsScenario() Scenario {
	return Scenario{
		ID:             "TC-03",
		Name:           "Toolchain reuse across different inputs is correctly rejected",
		Description:    "Same toolchain but different source input; verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "toolchain-reuse-different-input",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:TC0",
				SourceIndexFingerprint: "si:TC0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.22.0", Epoch: 1, Fingerprint: "fp:go-1.22.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-binary", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "order-service", Digest: "sha256:order-tc0", ByteSize: 10240000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-tc0"},
				DurationMs:             4000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:TC1",
				SourceIndexFingerprint: "si:TC1",
				Inputs:                 []string{"model:invoice"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.22.0", Epoch: 1, Fingerprint: "fp:go-1.22.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-binary", Hit: false, StaleReason: "STALE_INPUT"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "invoice-service", Digest: "sha256:invoice-tc1", ByteSize: 9800000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-tc1"},
				DurationMs:             3900,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "TC-03",
				ScenarioName:   "Toolchain reuse across different inputs is correctly rejected",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-missing", ArtifactPath: "order-service", Expected: "present (sha256:order-tc0, 10240000 bytes)", Actual: "missing", Severity: "error"}},
				Notes:          []string{"different input model; toolchain cache not reused"},
			}
			return report, nil
		},
	}
}

// AllToolchainScenarios returns all toolchain-change scenarios.
func AllToolchainScenarios() []Scenario {
	return []Scenario{
		ToolchainChangeScenario(),
		ToolchainSameVersionDifferentBuildScenario(),
		ToolchainReuseAcrossDifferentInputsScenario(),
	}
}

// ToolchainScenarioNames returns human-readable names for all toolchain scenarios.
func ToolchainScenarioNames() []string {
	scenarios := AllToolchainScenarios()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = fmt.Sprintf("%s: %s", s.ID, s.Name)
	}
	return names
}
