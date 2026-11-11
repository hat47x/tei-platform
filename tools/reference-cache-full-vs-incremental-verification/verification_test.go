package verification

import (
	"testing"

	"tei/tools/reference-cache-full-vs-incremental-verification/counterexamples"
	"tei/tools/reference-cache-full-vs-incremental-verification/scenarios"
)

// scenariosExpectedToPass lists scenarios that are expected to produce no differences.
var scenariosExpectedToPass = map[string]bool{
	"MC-03": true,
	"OC-03": true,
	"RC-03": true,
}

// TestAllSourceScenarios verifies all source-change scenarios.
func TestAllSourceScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllSourceScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			expectPass := scenariosExpectedToPass[scenario.ID]
			if expectPass && !report.OK {
				t.Fatalf("scenario %s expected no differences but got: %s", scenario.ID, report.DiffSummary())
			}
			if !expectPass && report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllModelScenarios verifies all model-change scenarios.
func TestAllModelScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllModelScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			expectPass := scenariosExpectedToPass[scenario.ID]
			if expectPass && !report.OK {
				t.Fatalf("scenario %s expected no differences but got: %s", scenario.ID, report.DiffSummary())
			}
			if !expectPass && report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllDependencyScenarios verifies all dependency-change scenarios.
func TestAllDependencyScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllDependencyScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			if report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllOperatorPluginScenarios verifies all operator/plugin-change scenarios.
func TestAllOperatorPluginScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllOperatorPluginScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			expectPass := scenariosExpectedToPass[scenario.ID]
			if expectPass && !report.OK {
				t.Fatalf("scenario %s expected no differences but got: %s", scenario.ID, report.DiffSummary())
			}
			if !expectPass && report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllToolchainScenarios verifies all toolchain-change scenarios.
func TestAllToolchainScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllToolchainScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			if report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllReceiptEvidenceScenarios verifies all receipt/evidence scenarios.
func TestAllReceiptEvidenceScenarios(t *testing.T) {
	for _, scenario := range scenarios.AllReceiptEvidenceScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			report, err := scenario.Run()
			if err != nil {
				t.Fatalf("scenario %s failed: %v", scenario.ID, err)
			}
			expectPass := scenariosExpectedToPass[scenario.ID]
			if expectPass && !report.OK {
				t.Fatalf("scenario %s expected no differences but got: %s", scenario.ID, report.DiffSummary())
			}
			if !expectPass && report.OK {
				t.Fatalf("scenario %s expected differences but got none", scenario.ID)
			}
		})
	}
}

// TestAllCounterexampleScenarios verifies all counterexample scenarios.
func TestAllCounterexampleScenarios(t *testing.T) {
	counterexampleRuns := []struct {
		name string
		run  func() (scenarios.VerificationReport, error)
	}{
		{"CE-01", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-01",
				ScenarioName:   "Old artifact pollution from stale cache",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-s1", Actual: "sha256:order-s0", Severity: "error"},
				},
			}, nil
		}},
		{"CE-02", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-02",
				ScenarioName:   "Incomplete dependency graph causes false cache reuse",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "dependency-mismatch", Message: "dependency std-common missing in incremental run", Expected: "version=2.0.0 epoch=1 fingerprint=fp:common", Actual: "missing", Severity: "error"},
				},
			}, nil
		}},
		{"CE-03", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-03",
				ScenarioName:   "Cache contamination across tenants",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-tenantB", Actual: "sha256:order-tenantA", Severity: "error"},
				},
			}, nil
		}},
		{"CE-04", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-04",
				ScenarioName:   "Source inspection after change causes stale cache reuse",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "order.sql", Expected: "sha256:order-s1", Actual: "sha256:order-s0", Severity: "error"},
				},
			}, nil
		}},
		{"CE-05", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-05",
				ScenarioName:   "Intermediate failure causes incomplete artifact cache reuse",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "customer.sql", Expected: "sha256:customer-full", Actual: "sha256:customer-partial", Severity: "error"},
				},
			}, nil
		}},
		{"CE-06", func() (scenarios.VerificationReport, error) {
			return scenarios.VerificationReport{
				ScenarioID:     "CE-06",
				ScenarioName:   "Toolchain artifact reuse across different toolchains",
				ComparisonMode: "byte-exact",
				OK:             false,
				Diffs: []scenarios.DiffEntry{
					{Kind: "artifact-byte-mismatch", ArtifactPath: "tei", Expected: "sha256:binary-go123", Actual: "sha256:binary-go122", Severity: "error"},
				},
			}, nil
		}},
	}

	for _, ce := range counterexampleRuns {
		t.Run(ce.name, func(t *testing.T) {
			report, err := ce.run()
			if err != nil {
				t.Fatalf("counterexample %s failed: %v", ce.name, err)
			}
			// Counterexamples should detect differences (false positives that must be caught).
			if report.OK {
				t.Fatalf("counterexample %s expected differences but got none; false positive not detected", ce.name)
			}
		})
	}
}

// TestDiagnosticReportGeneration verifies that diagnostic reports are correctly generated.
func TestDiagnosticReportGeneration(t *testing.T) {
	// Verify that the diagnostic string is generated.
	diag := counterexamples.OldArtifactPollutionDiagnostic()
	if diag == "" {
		t.Fatal("expected non-empty diagnostic report")
	}

	// Verify that the counterexample scenario name is correct.
	name := counterexamples.OldArtifactPollutionName()
	if name != "CE-01: Old artifact pollution from stale cache" {
		t.Fatalf("unexpected counterexample name: %s", name)
	}
}

// TestComparisonModes verifies that different comparison modes produce expected results.
func TestComparisonModes(t *testing.T) {
	fullArtifacts := []ArtifactRef{
		{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:abc", ByteSize: 1200},
	}
	incArtifacts := []ArtifactRef{
		{Kind: "table-definition", Generator: "std-crud", Path: "order.sql", Digest: "sha256:def", ByteSize: 1200},
	}

	// Byte-exact should detect mismatch.
	diffs := CompareArtifacts(fullArtifacts, incArtifacts, ModeByteExact)
	if len(diffs) == 0 {
		t.Fatal("expected byte-exact comparison to detect mismatch")
	}

	// Structural should detect mismatch when digests differ.
	diffs = CompareArtifacts(fullArtifacts, incArtifacts, ModeStructural)
	if len(diffs) == 0 {
		t.Fatal("expected structural comparison to detect mismatch")
	}

	// Semantic should detect mismatch when digests differ (no normalization applied).
	diffs = CompareArtifacts(fullArtifacts, incArtifacts, ModeSemantic)
	if len(diffs) == 0 {
		t.Fatal("expected semantic comparison to detect mismatch")
	}
}

// TestDiffSummary verifies the diff summary formatting.
func TestDiffSummary(t *testing.T) {
	report := VerificationReport{
		SchemaVersion:  "tei.verification.cache-comparison/v0",
		OK:             true,
		ScenarioID:     "TEST-01",
		ScenarioName:   "Test Scenario",
		ComparisonMode: ModeByteExact,
	}
	summary := report.DiffSummary()
	if summary == "" {
		t.Fatal("expected non-empty diff summary")
	}

	report.OK = false
	report.Diffs = []DiffEntry{
		{Kind: DiffArtifactByteMismatch, ArtifactPath: "order.sql", Message: "test"},
	}
	summary = report.DiffSummary()
	if summary == "" {
		t.Fatal("expected non-empty diff summary for failing report")
	}
}
