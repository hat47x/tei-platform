package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditReadinessCanDeclareGateCReadyOnlyAfterAllEvidenceClassesPass(t *testing.T) {
	dir := t.TempDir()
	semanticPath, architecturePath, visualPath := writeReadinessFixture(t, dir)

	summary, err := auditReadiness(semanticPath, architecturePath, visualPath)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.GateCReady {
		t.Fatalf("Gate C readiness=%+v", summary)
	}
	if summary.HandoverBaselineID != "B" ||
		summary.BaselineRepositoryRevision != "baseline-r" ||
		summary.TargetBuildRevision != "target-t" {
		t.Fatalf("provenance=%+v", summary)
	}
	if summary.TargetDifferenceHandoff.ExecutionArtifact != "go-run-native-process" ||
		summary.TargetDifferenceHandoff.ABI != "linux/amd64" ||
		len(summary.TargetDifferenceHandoff.RuntimeBindings) != 3 {
		t.Fatalf("target handoff=%+v", summary.TargetDifferenceHandoff)
	}
	for _, check := range summary.Checks {
		if !check.Passed {
			t.Fatalf("readiness check failed: %+v", check)
		}
	}
}

func TestAuditReadinessRejectsVisualFailure(t *testing.T) {
	dir := t.TempDir()
	semanticPath, architecturePath, visualPath := writeReadinessFixture(t, dir)

	var visual visualAudit
	readReadinessJSON(t, visualPath, &visual)
	visual.VisualFidelityPassed = false
	writeReadinessJSON(t, visualPath, visual)

	if _, err := auditReadiness(semanticPath, architecturePath, visualPath); err == nil {
		t.Fatal("Visual failure was accepted as Gate C readiness")
	}
}

func TestAuditReadinessRejectsTargetRevisionMismatch(t *testing.T) {
	dir := t.TempDir()
	semanticPath, architecturePath, visualPath := writeReadinessFixture(t, dir)

	var architecture architectureEvidence
	readReadinessJSON(t, architecturePath, &architecture)
	architecture.TargetBuildRevision = "other-target"
	writeReadinessJSON(t, architecturePath, architecture)

	if _, err := auditReadiness(semanticPath, architecturePath, visualPath); err == nil {
		t.Fatal("mismatched Target revision was accepted")
	}
}

func TestAuditReadinessRejectsUnknownUnresolvedItem(t *testing.T) {
	dir := t.TempDir()
	semanticPath, architecturePath, visualPath := writeReadinessFixture(t, dir)

	var semantic semanticFunctionalSummary
	readReadinessJSON(t, semanticPath, &semantic)
	semantic.Unresolved = append(semantic.Unresolved, "new-unknown-gate-c-gap")
	writeReadinessJSON(t, semanticPath, semantic)

	if _, err := auditReadiness(semanticPath, architecturePath, visualPath); err == nil {
		t.Fatal("unclassified unresolved item was accepted")
	}
}

func TestAuditReadinessRejectsIncompleteTargetDifferenceHandoff(t *testing.T) {
	dir := t.TempDir()
	semanticPath, architecturePath, visualPath := writeReadinessFixture(t, dir)

	var semantic semanticFunctionalSummary
	readReadinessJSON(t, semanticPath, &semantic)
	semantic.TargetDifferenceHandoff.ExecutionArtifact = "unresolved"
	writeReadinessJSON(t, semanticPath, semantic)

	if _, err := auditReadiness(semanticPath, architecturePath, visualPath); err == nil {
		t.Fatal("incomplete Target difference handoff was accepted")
	}
}

func TestClassifyUnresolvedCarriesKnownGapsWithoutUpgradingEvidence(t *testing.T) {
	dispositions, err := classifyUnresolved(
		[]string{
			"visual-fidelity-evidence-not-acquired",
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-external-command-observation-not-acquired",
		},
		[]string{
			"target-capability-taxonomy-remains-candidate-until-second-target",
		},
		[]string{
			"general-target-neutral-presentation-taxonomy-not-yet-defined",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(dispositions) != 5 {
		t.Fatalf("dispositions=%+v", dispositions)
	}
	for _, disposition := range dispositions {
		if disposition.Blocking {
			t.Fatalf("known carry-forward item became blocking: %+v", disposition)
		}
		if disposition.ID == "actual-external-command-observation-not-acquired" &&
			disposition.Disposition != "carry-forward" {
			t.Fatalf("external Evidence gap was silently resolved: %+v", disposition)
		}
	}
}

func writeReadinessFixture(t *testing.T, dir string) (string, string, string) {
	t.Helper()

	semanticPath := filepath.Join(dir, "semantic-functional-summary.json")
	writeReadinessJSON(t, semanticPath, semanticFunctionalSummary{
		EvidenceNature:             "first-web-target-semantic-functional-evidence",
		RunIdentity:                "semantic-run-1",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SourceQualification:        passState{Passed: true},
		SemanticConformance:        passState{Passed: true},
		FunctionalEvidence:         passState{Passed: true},
		VisualEvidence: pendingVisualState{
			Status:   "pending",
			Required: true,
		},
		TargetDifferenceHandoff: targetDifferenceHandoff{
			ApplicationID:      "order_entry",
			TargetAdapter:      "reference-first-web-target",
			Host:               "go-net-http",
			TargetCapabilities: []string{"http-json-manifest", "http-command-scenario"},
			RuntimeBindings: []runtimeBinding{
				{
					Pattern:    "order_entry.recent_orders",
					Datasource: "sales",
					Database:   "sales_core",
					Access:     "read-write",
				},
				{
					Pattern:            "order_entry.customer_reference",
					Environment:        "development",
					Instance:           "master_data.dev",
					Provider:           "oracle",
					ExecutorCapability: "database.query.oracle",
					ContractQuery:      "master_data/master_data.customer/find_by_id",
				},
				{
					Pattern:            "order_entry.order_lines",
					Environment:        "development",
					Instance:           "sales_core.dev",
					Provider:           "postgres",
					ExecutorCapability: "database.query.postgres",
					ContractQuery:      "sales_core/sales_core.order_line/find_by_order",
				},
			},
			ExecutionArtifact: "go-run-native-process",
			ABI:               "linux/amd64",
			ExecutionTrust:    []string{"reference-runtime-data-is-synthetic"},
			WebSpecific:       []string{"/scenario/submit", "HTTP status mapping"},
			DoNotCanonicalize: []string{"HTTP route", "HTTP status"},
			Unresolved: []string{
				"browse-environment-instance-binding-not-exposed-by-current-driver-api",
				"actual-external-command-observation-not-acquired",
				"target-capability-taxonomy-remains-candidate-until-second-target",
			},
		},
		Unresolved: []string{
			"visual-fidelity-evidence-not-acquired",
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-external-command-observation-not-acquired",
		},
		GateCReady: false,
	})

	architecturePath := filepath.Join(dir, "architecture-conformance.json")
	writeReadinessJSON(t, architecturePath, architectureEvidence{
		EvidenceNature:             "first-web-target-architecture-conformance",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		WorkingTreeClean:           true,
		ReproducibleRevision:       true,
		PostRunRepository: postRunRepository{
			Checked:          true,
			RevisionStable:   true,
			WorkingTreeClean: true,
		},
		CanonicalWorldInvariant:    architectureCheck{Passed: true},
		WebTargetAuthorityBoundary: architectureCheck{Passed: true},
		SharedCommandBoundary:      architectureCheck{Passed: true},
		Unresolved: []string{
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-external-command-observation-not-acquired",
			"general-target-neutral-presentation-taxonomy-not-yet-defined",
		},
	})

	visualPath := filepath.Join(dir, "visual-audit.json")
	writeReadinessJSON(t, visualPath, visualAudit{
		EvidenceNature:                      "first-web-target-visual-evidence-audit",
		VisualRunIdentity:                   "visual-run-1",
		SourceSemanticFunctionalRunIdentity: "semantic-run-1",
		HandoverBaselineID:                  "B",
		BaselineRepositoryRevision:          "baseline-r",
		TargetBuildRevision:                 "target-t",
		TargetAdapter:                       "reference-first-web-target",
		Host:                                "go-net-http",
		AllRequiredCapturesPresent:          true,
		AllCriteriaResolved:                 true,
		VisualFidelityPassed:                true,
		GateCReady:                          false,
	})

	return semanticPath, architecturePath, visualPath
}

func writeReadinessJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readReadinessJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}
