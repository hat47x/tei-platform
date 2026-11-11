package scenarios

// ModelChangeScenario verifies that a model change invalidates cached
// artifacts and causes a full re-evaluation.
func ModelChangeScenario() Scenario {
	return Scenario{
		ID:             "MC-01",
		Name:           "Model definition change invalidates cache",
		Description:    "Change a model element (e.g. add a field) and verify that cached artifacts are recomputed.",
		ComparisonMode: "byte-exact",
		ChangeType:     "model",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:M0",
				SourceIndexFingerprint: "si:M0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false, RuleFingerprint: "rule:derived-crud-v1"}},
				Artifacts: []ArtifactRef{
					{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-M0", ByteSize: 1300},
					{Kind: "operation-surface", Generator: "std-crud", Path: "order-operations.json", Digest: "sha256:ops-M0", ByteSize: 850},
				},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:     []string{"receipt:full-M0"},
				DurationMs:   1200,
				SkippedSteps: []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:M1",
				SourceIndexFingerprint: "si:M1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:derived-crud-v1"},
					{Key: "artifact:order-ops", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:derived-crud-v1"},
				},
				Artifacts: []ArtifactRef{
					{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-M1", ByteSize: 1400},
					{Kind: "operation-surface", Generator: "std-crud", Path: "order-operations.json", Digest: "sha256:ops-M1", ByteSize: 900},
				},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-M1"},
				DurationMs:   1100,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "MC-01",
				ScenarioName:   "Model definition change invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-M0", Actual: "sha256:order-M1", Severity: "error"}},
				Notes:          []string{"model fingerprint changed from M0 to M1; all derived artifacts recomputed"},
			}
			return report, nil
		},
	}
}

// ModelVersionBumpScenario verifies that a model version bump without
// semantic change still invalidates cache if the version is part of the cache key.
func ModelVersionBumpScenario() Scenario {
	return Scenario{
		ID:             "MC-02",
		Name:           "Model version bump invalidates cache",
		Description:    "Model version changes from 1.0.0 to 1.0.1 with no semantic change; verify cache is invalidated.",
		ComparisonMode: "structural",
		ChangeType:     "model-version",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:M1.0.0",
				SourceIndexFingerprint: "si:M1.0.0",
				Inputs:                 []string{"model:order"},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false, RuleFingerprint: "rule:v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:M1.0.1",
				SourceIndexFingerprint: "si:M1.0.1",
				Inputs:                 []string{"model:order"},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: false, StaleReason: "STALE_RULE", RuleFingerprint: "rule:v1"},
				},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:     []string{"receipt:incremental-1"},
				DurationMs:   900,
				SkippedSteps: []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "MC-02",
				ScenarioName:   "Model version bump invalidates cache",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "structural",
				Diffs:          []DiffEntry{{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:order-table rejected with stale reason STALE_RULE", Expected: "not reused", Actual: "rejected", Severity: "error"}},
				Notes:          []string{"version changed from 1.0.0 to 1.0.1; cache invalidated even though content is structurally identical"},
			}
			return report, nil
		},
	}
}

// ModelSemanticEquivalenceScenario verifies that two models with different
// serialization but identical semantics produce equivalent artifacts.
func ModelSemanticEquivalenceScenario() Scenario {
	return Scenario{
		ID:             "MC-03",
		Name:           "Semantically equivalent models produce identical artifacts",
		Description:    "Two models that differ only in non-semantic serialization (e.g. comment whitespace) should produce equivalent artifacts under semantic comparison.",
		ComparisonMode: "semantic",
		ChangeType:     "model-semantic-equivalence",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:M0",
				SourceIndexFingerprint: "si:M0",
				Inputs:                 []string{"model:order"},
				CacheEntries:           []CacheEntryRef{},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-semantic", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:M0",
				SourceIndexFingerprint: "si:M0",
				Inputs:                 []string{"model:order"},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: true, SourceFingerprint: "source:M0", RuleFingerprint: "rule:derived-crud-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-semantic", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-1"},
				DurationMs:             100,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             true,
				ScenarioID:     "MC-03",
				ScenarioName:   "Semantically equivalent models produce identical artifacts",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "semantic",
				Diffs:          []DiffEntry{},
				Notes:          []string{"semantic comparison shows no difference despite different serialization timestamps"},
			}
			return report, nil
		},
	}
}

// AllModelScenarios returns all model-change scenarios.
func AllModelScenarios() []Scenario {
	return []Scenario{
		ModelChangeScenario(),
		ModelVersionBumpScenario(),
		ModelSemanticEquivalenceScenario(),
	}
}
