package counterexamples

import (
	"fmt"
)

// CacheContaminationScenario demonstrates a false positive where cache
// entries from a different tenant or context are incorrectly reused.
func CacheContaminationScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-03",
		Name:           "Cache contamination across tenants",
		Description:    "A cache entry from tenant A is incorrectly reused for tenant B due to a shared cache key.",
		ComparisonMode: "byte-exact",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:tenantB",
				SourceIndexFingerprint: "si:tenantB",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-tenantB", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-tenantB"},
				DurationMs:             1000,
				SkippedSteps:           []string{},
			}
			// Incremental run incorrectly reuses tenant A's cache entry.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:tenantB",
				SourceIndexFingerprint: "si:tenantB",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "std-crud", Version: "1.0.0", Epoch: 1, Fingerprint: "fp:std-crud"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:order-table", Hit: true, StaleReason: "CONTEXT_MISMATCH", SourceFingerprint: "source:order:tenantA"}},
				Artifacts:              []ArtifactRef{{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:order-tenantA", ByteSize: 1200}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-tenantA"},
				DurationMs:             50,
				SkippedSteps:           []string{"generate-order-table"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-03",
				ScenarioName:   "Cache contamination across tenants",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-tenantB", Actual: "sha256:order-tenantA", Severity: "error"},
					{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:order-table reused with stale reason CONTEXT_MISMATCH in incremental run only", Expected: "not reused", Actual: "reused", Severity: "error"},
				},
				Notes: []string{"tenant A's cache entry incorrectly reused for tenant B"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// CacheContaminationName returns the scenario name.
func CacheContaminationName() string {
	return "CE-03: Cache contamination across tenants"
}

// CacheContaminationDiagnostic generates a diagnostic report for CE-03.
func CacheContaminationDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-03
Status: FAIL
Summary: 2 errors, 0 warnings, 2 records
[error] CACHE_CONTAMINATION: cache entry from different tenant reused
    expectedTenant = tenantB
    actualTenant = tenantA
    cacheKey = artifact:order-table
[error] CONTEXT_MISMATCH: cache entry context class does not match current request
    expectedContext = tenantB/admin
    actualContext = tenantA/admin
[info] DIAGNOSIS: cache contamination: tenant isolation failure caused incorrect cache reuse
`)
}
