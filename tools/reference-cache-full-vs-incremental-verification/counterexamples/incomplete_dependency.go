package counterexamples

import (
	"fmt"
)

// IncompleteDependencyScenario demonstrates a false positive where an
// incomplete dependency graph causes cache reuse even though transitive
// dependencies have changed.
func IncompleteDependencyScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-02",
		Name:           "Incomplete dependency graph causes false cache reuse",
		Description:    "The dependency graph only tracks direct dependencies; a transitive dependency change is missed, causing incorrect cache reuse.",
		ComparisonMode: "byte-exact",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:D0",
				SourceIndexFingerprint: "si:D0",
				Inputs:                 []string{"model:order"},
				Dependencies: []DependencyRef{
					{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"},
					{ID: "std-common", Version: "2.0.0", Epoch: 1, Fingerprint: "fp:common"},
				},
				CacheEntries: []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:    []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-d0", ByteSize: 1200}},
				Evidence:     []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:     []string{"receipt:full-d0"},
				DurationMs:   1000,
				SkippedSteps: []string{},
			}
			// Incremental run only sees std-crud dependency, missing std-common change.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:D1",
				SourceIndexFingerprint: "si:D1",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:gen"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: true, SourceFingerprint: "source:order", RuleFingerprint: "rule:std-crud-v1"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-d0", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-d0"},
				DurationMs:             50,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-02",
				ScenarioName:   "Incomplete dependency graph causes false cache reuse",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "dependency-mismatch", Message: "dependency std-common missing in incremental run", Expected: "version=2.0.0 epoch=1 fingerprint=fp:common", Actual: "missing", Severity: "error"},
					{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:order-table reused with stale reason STALE_INPUT in incremental run only", Expected: "not reused", Actual: "reused", Severity: "error"},
				},
				Notes: []string{"transitive dependency std-common changed but not observed in incremental run"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// IncompleteDependencyName returns the scenario name.
func IncompleteDependencyName() string {
	return "CE-02: Incomplete dependency graph causes false cache reuse"
}

// IncompleteDependencyDiagnostic generates a diagnostic report for CE-02.
func IncompleteDependencyDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-02
Status: FAIL
Summary: 2 errors, 0 warnings, 2 records
[error] INCOMPLETE_DEPENDENCY_GRAPH: transitive dependency change not detected due to incomplete dependency tracking
    expectedDependencies = std-crud, std-common
    actualDependencies = std-crud
    changedDependency = std-common
[error] CACHE_FALSE_REUSE: cache reused despite transitive dependency change
    cacheKey = artifact:order-table
[info] DIAGNOSIS: incomplete dependency graph: std-common epoch changed from 1 to 2 but was not observed
`)
}
