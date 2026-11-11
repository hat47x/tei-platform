package counterexamples

import (
	"fmt"
)

// IntermediateFailureScenario demonstrates a false positive where an
// intermediate failure during processing causes incomplete artifacts to be
// cached, and the incremental run reuses the incomplete artifacts.
func IntermediateFailureScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-05",
		Name:           "Intermediate failure causes incomplete artifact cache reuse",
		Description:    "A run fails partway through, leaving incomplete artifacts in the cache; the next run reuses the incomplete artifacts.",
		ComparisonMode: "byte-exact",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:F0",
				SourceIndexFingerprint: "si:F0",
				Inputs:                 []string{"model:order", "model:customer"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts: []ArtifactRef{
					{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-full", ByteSize: 1200},
					{Kind: "table-definition", Generator: "std-crud", Path: "customer.sql", Digest: "sha256:customer-full", ByteSize: 1000},
				},
				Evidence: []EvidenceRef{
					{Operation: "execute", Status: "committed", ReceiptVersion: 1},
					{Operation: "execute", Status: "committed", ReceiptVersion: 2},
				},
				Receipts:     []string{"receipt:full-1", "receipt:full-2"},
				DurationMs:   2000,
				SkippedSteps: []string{},
			}
			// Incremental run reuses incomplete artifact from failed run.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     false,
				WorldFingerprint:       "world:F0",
				SourceIndexFingerprint: "si:F0",
				Inputs:                 []string{"model:order", "model:customer"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries: []CacheEntryRef{
					{Key: "artifact:order-table", Hit: true, SourceFingerprint: "source:order", RuleFingerprint: "rule:std-crud-v1"},
					{Key: "artifact:customer-table", Hit: true, SourceFingerprint: "source:customer", RuleFingerprint: "rule:std-crud-v1", StaleReason: "PARTIAL_FAILURE"},
				},
				Artifacts: []ArtifactRef{
					{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-full", ByteSize: 1200},
					{Kind: "table-definition", Generator: "std-crud", Path: "customer.sql", Digest: "sha256:customer-partial", ByteSize: 400},
				},
				Evidence: []EvidenceRef{
					{Operation: "execute", Status: "committed", ReceiptVersion: 1},
					{Operation: "execute", Status: "", ErrorClass: "execution-error"},
				},
				Receipts:     []string{"receipt:full-1"},
				DurationMs:   300,
				SkippedSteps: []string{"generate-customer-table"},
				Error:        "intermediate failure: customer artifact incomplete",
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-05",
				ScenarioName:   "Intermediate failure causes incomplete artifact cache reuse",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "customer.sql", Expected: "sha256:customer-full", Actual: "sha256:customer-partial", Severity: "error"},
					{Kind: "artifact-byte-mismatch", ArtifactPath: "customer.sql", Expected: "1000 bytes", Actual: "400 bytes", Severity: "warning"},
					{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:customer-table reused with stale reason PARTIAL_FAILURE in incremental run only", Expected: "not reused", Actual: "reused", Severity: "error"},
				},
				Notes: []string{"incremental run reused incomplete customer artifact from failed run"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// IntermediateFailureName returns the scenario name.
func IntermediateFailureName() string {
	return "CE-05: Intermediate failure causes incomplete artifact cache reuse"
}

// IntermediateFailureDiagnostic generates a diagnostic report for CE-05.
func IntermediateFailureDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-05
Status: FAIL
Summary: 3 errors, 1 warning, 3 records
[error] INCOMPLETE_ARTIFACT_REUSED: incremental run reused incomplete artifact from failed intermediate run
    artifactPath = customer.sql
    expectedDigest = sha256:customer-full
    actualDigest = sha256:customer-partial
    expectedSize = 1000
    actualSize = 400
[error] INTERMEDIATE_FAILURE_ARTIFACT: artifact from failed run was cached and reused
    failedRunStatus = execution-error
    reusedArtifact = customer.sql
[error] CACHE_FALSE_REUSE: cache reused despite intermediate failure
    cacheKey = artifact:customer-table
[warning] ARTIFACT_SIZE_MISMATCH: artifact size differs between runs
    expected = 1000 bytes
    actual = 400 bytes
[info] DIAGNOSIS: intermediate failure: incomplete artifact from failed run was reused in subsequent run
`)
}
