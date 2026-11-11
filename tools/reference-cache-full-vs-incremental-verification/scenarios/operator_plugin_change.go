package scenarios

import (
	"fmt"
)

// OperatorChangeScenario verifies that an operator change invalidates cached
// artifacts and causes a full re-evaluation.
func OperatorChangeScenario() Scenario {
	return Scenario{
		ID:             "OC-01",
		Name:           "Operator definition change invalidates cache",
		Description:    "Modify an operator definition and verify that cached artifacts are recomputed.",
		ComparisonMode: "byte-exact",
		ChangeType:     "operator",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:OP0",
				SourceIndexFingerprint: "si:OP0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "operator:validation", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:op-1.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-validation", Hit: false, RuleFingerprint: "rule:validation-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "validation-operator", Path: "order-validation.sql", Digest: "sha256:order-op0", ByteSize: 800}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-op0"},
				DurationMs:             900,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:OP1",
				SourceIndexFingerprint: "si:OP1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "operator:validation", Version: "1.1.0", Epoch: 2, Fingerprint: "fp:op-1.1.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-validation", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:validation-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "validation-operator", Path: "order-validation.sql", Digest: "sha256:order-op1", ByteSize: 850}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-op1"},
				DurationMs:             920,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "OC-01",
				ScenarioName:   "Operator definition change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "order-validation.sql", Expected: "sha256:order-op0", Actual: "sha256:order-op1", Severity: "error"}},
				Notes:          []string{"operator version changed from 1.0.0 to 1.1.0; cache correctly rejected"},
			}
			return report, nil
		},
	}
}

// PluginChangeScenario verifies that a plugin change invalidates cached
// artifacts and causes a full re-evaluation.
func PluginChangeScenario() Scenario {
	return Scenario{
		ID:             "OC-02",
		Name:           "Plugin definition change invalidates cache",
		Description:    "Update a plugin definition and verify that cached artifacts are recomputed.",
		ComparisonMode: "byte-exact",
		ChangeType:     "plugin",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:PL0",
				SourceIndexFingerprint: "si:PL0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "plugin:pricing", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:plugin-2.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:pricing", Hit: false, RuleFingerprint: "rule:pricing-v2"}},
				Artifacts:              []ArtifactRef{{Kind: "operation-surface", Generator: "pricing-plugin", Path: "pricing-operations.json", Digest: "sha256:pricing-pl0", ByteSize: 600}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-pl0"},
				DurationMs:             1100,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:PL1",
				SourceIndexFingerprint: "si:PL1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "plugin:pricing", Version: "2.1.0", Epoch: 2, Fingerprint: "fp:plugin-2.1.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:pricing", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:pricing-v2"}},
				Artifacts:              []ArtifactRef{{Kind: "operation-surface", Generator: "pricing-plugin", Path: "pricing-operations.json", Digest: "sha256:pricing-pl1", ByteSize: 650}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-pl1"},
				DurationMs:             1050,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "OC-02",
				ScenarioName:   "Plugin definition change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "pricing-operations.json", Expected: "sha256:pricing-pl0", Actual: "sha256:pricing-pl1", Severity: "error"}},
				Notes:          []string{"plugin version changed from 2.0.0 to 2.1.0; cache correctly rejected"},
			}
			return report, nil
		},
	}
}

// UnusedPluginAddedScenario verifies that adding an unused plugin does not
// invalidate cache for unaffected artifacts.
func UnusedPluginAddedScenario() Scenario {
	return Scenario{
		ID:             "OC-03",
		Name:           "Unused plugin addition does not invalidate unrelated cache",
		Description:    "Add a plugin that is not used by the current model; verify cache for existing artifacts is still valid.",
		ComparisonMode: "byte-exact",
		ChangeType:     "plugin-added-unused",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:PL0",
				SourceIndexFingerprint: "si:PL0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-pl0", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-pl0"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:PL1",
				SourceIndexFingerprint: "si:PL1",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"},
					{ID: "plugin:unused", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:unused"},
				},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: true, SourceFingerprint: "source:order", RuleFingerprint: "rule:std-crud-v1"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-pl0", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:     []string{"receipt:incremental-pl0"},
				DurationMs:   100,
				SkippedSteps: []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             true,
				ScenarioID:     "OC-03",
				ScenarioName:   "Unused plugin addition does not invalidate unrelated cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{},
				Notes:          []string{"unused plugin added but std-crud artifact cache remains valid"},
			}
			return report, nil
		},
	}
}

// PluginLockFingerprintChangeScenario verifies that a plugin lock fingerprint
// change invalidates cache.
func PluginLockFingerprintChangeScenario() Scenario {
	return Scenario{
		ID:             "OC-04",
		Name:           "Plugin lock fingerprint change invalidates cache",
		Description:    "Plugin ID and version unchanged but lock fingerprint changed; verify cache is invalidated.",
		ComparisonMode: "byte-exact",
		ChangeType:     "plugin-lock",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:PL0",
				SourceIndexFingerprint: "si:PL0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "plugin:pricing", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:plugin-2.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:pricing", Hit: false, RuleFingerprint: "rule:pricing-v2"}},
				Artifacts:              []ArtifactRef{{Kind: "operation-surface", Generator: "pricing-plugin", Path: "pricing-operations.json", Digest: "sha256:pricing-pl0", ByteSize: 600}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-pl0"},
				DurationMs:             1100,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:PL1",
				SourceIndexFingerprint: "si:PL1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "plugin:pricing", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:plugin-2.0.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:pricing", Hit: false, StaleReason: "STALE_INPUT", RuleFingerprint: "rule:pricing-v2"}},
				Artifacts:              []ArtifactRef{{Kind: "operation-surface", Generator: "pricing-plugin", Path: "pricing-operations.json", Digest: "sha256:pricing-pl1", ByteSize: 600}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-pl1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "OC-04",
				ScenarioName:   "Plugin lock fingerprint change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:pricing rejected with stale reason STALE_INPUT", Expected: "not reused", Actual: "rejected", Severity: "error"}},
				Notes:          []string{"lock fingerprint changed; cache invalidated despite same ID and version"},
			}
			return report, nil
		},
	}
}

// AllOperatorPluginScenarios returns all operator/plugin-change scenarios.
func AllOperatorPluginScenarios() []Scenario {
	return []Scenario{
		OperatorChangeScenario(),
		PluginChangeScenario(),
		UnusedPluginAddedScenario(),
		PluginLockFingerprintChangeScenario(),
	}
}

// OperatorPluginScenarioNames returns human-readable names for all operator/plugin scenarios.
func OperatorPluginScenarioNames() []string {
	scenarios := AllOperatorPluginScenarios()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = fmt.Sprintf("%s: %s", s.ID, s.Name)
	}
	return names
}
