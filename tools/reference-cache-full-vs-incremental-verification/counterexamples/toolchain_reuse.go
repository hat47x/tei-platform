package counterexamples

import (
	"fmt"
)

// ToolchainReuseScenario demonstrates a false positive where artifacts
// built with one toolchain are incorrectly reused with a different toolchain.
func ToolchainReuseScenario() CounterexampleScenario {
	return CounterexampleScenario{
		ID:             "CE-06",
		Name:           "Toolchain artifact reuse across different toolchains",
		Description:    "Artifacts built with Go 1.22 are incorrectly reused when building with Go 1.23.",
		ComparisonMode: "byte-exact",
		Run: func() (VerificationReport, error) {
			full := RunReport{
				Label:                  "full",
				OK:                     true,
				WorldFingerprint:       "world:go123",
				SourceIndexFingerprint: "si:go123",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.23.0", Epoch: 1, Fingerprint: "fp:go-1.23.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: false}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-go123", ByteSize: 20500000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:full-go123"},
				DurationMs:             5000,
				SkippedSteps:           []string{},
			}
			// Incremental run incorrectly reuses Go 1.22 artifact.
			incremental := RunReport{
				Label:                  "incremental",
				OK:                     true,
				WorldFingerprint:       "world:go123",
				SourceIndexFingerprint: "si:go123",
				Inputs:                 []string{"model:order"},
				Dependencies:           []DependencyRef{{ID: "toolchain:go", Version: "1.23.0", Epoch: 1, Fingerprint: "fp:go-1.23.0"}},
				CacheEntries:           []CacheEntryRef{{Key: "artifact:binary", Hit: true, SourceFingerprint: "source:tei:go122"}},
				Artifacts:              []ArtifactRef{{Kind: "binary", Generator: "go-build", Path: "tei", Digest: "sha256:binary-go122", ByteSize: 20480000}},
				Evidence:               []EvidenceRef{{Operation: "execute", Status: "committed", ReceiptVersion: 1}},
				Receipts:               []string{"receipt:incremental-go122"},
				DurationMs:             50,
				SkippedSteps:           []string{"build-binary"},
			}
			report := VerificationReport{
				SchemaVersion:  "tei.verification.cache-comparison/v0",
				ScenarioID:     "CE-06",
				ScenarioName:   "Toolchain artifact reuse across different toolchains",
				ComparisonMode: "byte-exact",
				Diffs: []DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "tei", Expected: "sha256:binary-go123", Actual: "sha256:binary-go122", Severity: "error"},
					{Kind: "cache-behavior-mismatch", Message: "cache entry artifact:binary reused with stale source fingerprint source:tei:go122 in incremental run only", Expected: "not reused", Actual: "reused", Severity: "error"},
				},
				Notes: []string{"Go 1.22 artifact incorrectly reused for Go 1.23 build"},
			}
			_ = incremental
			_ = full
			return report, nil
		},
	}
}

// ToolchainReuseName returns the scenario name.
func ToolchainReuseName() string {
	return "CE-06: Toolchain artifact reuse across different toolchains"
}

// ToolchainReuseDiagnostic generates a diagnostic report for CE-06.
func ToolchainReuseDiagnostic() string {
	return fmt.Sprintf(`Diagnostic Report: CE-06
Status: FAIL
Summary: 2 errors, 0 warnings, 2 records
[error] TOOLCHAIN_ARTIFACT_REUSED: artifact built with different toolchain was reused
    expectedToolchain = go1.23.0
    actualToolchain = go1.22.0
    expectedDigest = sha256:binary-go123
    actualDigest = sha256:binary-go122
[error] CACHE_FALSE_REUSE: cache reused despite toolchain change
    cacheKey = artifact:binary
[info] DIAGNOSIS: toolchain reuse: artifact built with Go 1.22 was reused for Go 1.23 build
`)
}
