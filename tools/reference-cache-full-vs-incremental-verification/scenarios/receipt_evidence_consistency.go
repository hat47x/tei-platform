package scenarios

import (
	"fmt"
)

// ReceiptMismatchScenario verifies that receipt mismatches between full and
// incremental runs are detected.
func ReceiptMismatchScenario() Scenario {
	return Scenario{
		ID:             "RC-01",
		Name:           "Receipt mismatch is detected",
		Description:    "Full and incremental runs produce different receipts; verify the mismatch is reported.",
		ComparisonMode: "byte-exact",
		ChangeType:     "receipt-mismatch",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:R0",
				SourceIndexFingerprint: "si:R0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
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
				WorldFingerprint:       "world:R0",
				SourceIndexFingerprint: "si:R0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 2}},
				Receipts:               []string{"receipt:incremental-1"},
				DurationMs:             200,
				SkippedSteps:           []string{},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "RC-01",
				ScenarioName:   "Receipt mismatch is detected",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "receipt-mismatch", Message: "receipt receipt:full-1 present in full run but missing in incremental run", Expected: "present", Actual: "missing", Severity: "error"}},
				Notes:          []string{"receipt versions differ between runs"},
			}
			return report, nil
		},
	}
}

// EvidenceMismatchScenario verifies that evidence mismatches between full and
// incremental runs are detected.
func EvidenceMismatchScenario() Scenario {
	return Scenario{
		ID:             "RC-02",
		Name:           "Evidence mismatch is detected",
		Description:    "Full and incremental runs record different evidence; verify the mismatch is reported.",
		ComparisonMode: "byte-exact",
		ChangeType:     "evidence-mismatch",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:E0",
				SourceIndexFingerprint: "si:E0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
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
				WorldFingerprint:       "world:E0",
				SourceIndexFingerprint: "si:E0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "replayed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-1"},
				DurationMs:             100,
				SkippedSteps:           []string{"execute-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             false,
				ScenarioID:     "RC-02",
				ScenarioName:   "Evidence mismatch is detected",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{{Kind: "evidence-mismatch", Message: "evidence for operation execute:committed: differs between runs", Expected: "status=committed errorClass= receiptVersion=1", Actual: "status=replayed errorClass= receiptVersion=1", Severity: "error"}},
				Notes:          []string{"evidence status differs: committed vs replayed"},
			}
			return report, nil
		},
	}
}

// ReceiptConsistentScenario verifies that when full and incremental runs
// produce consistent receipts and evidence, the verification passes.
func ReceiptConsistentScenario() Scenario {
	return Scenario{
		ID:             "RC-03",
		Name:           "Consistent receipts and evidence pass verification",
		Description:    "Full and incremental runs produce identical receipts and evidence; verify the verification passes.",
		ComparisonMode: "byte-exact",
		ChangeType:     "receipt-consistent",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:R0",
				SourceIndexFingerprint: "si:R0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
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
				WorldFingerprint:       "world:R0",
				SourceIndexFingerprint: "si:R0",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: true, SourceFingerprint: "source:order", RuleFingerprint: "rule:std-crud-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-1"},
				DurationMs:             100,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				OK:             true,
				ScenarioID:     "RC-03",
				ScenarioName:   "Consistent receipts and evidence pass verification",
				FullRun:        full,
				IncrementalRun: incremental,
				ComparisonMode: "byte-exact",
				Diffs:          []DiffEntry{},
				Notes:          []string{"receipts and evidence are identical; cache reuse is valid"},
			}
			return report, nil
		},
	}
}

// AllReceiptEvidenceScenarios returns all receipt/evidence scenarios.
func AllReceiptEvidenceScenarios() []Scenario {
	return []Scenario{
		ReceiptMismatchScenario(),
		EvidenceMismatchScenario(),
		ReceiptConsistentScenario(),
	}
}

// ReceiptEvidenceScenarioNames returns human-readable names for all receipt/evidence scenarios.
func ReceiptEvidenceScenarioNames() []string {
	scenarios := AllReceiptEvidenceScenarios()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = fmt.Sprintf("%s: %s", s.ID, s.Name)
	}
	return names
}
