package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditArtifactsPassesSemanticFunctionalButNeverVisualOrGateC(t *testing.T) {
	dir := t.TempDir()
	expected := Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
	writeAuditFixture(t, dir, "run-1", expected)

	summary, err := auditArtifacts(dir, "run-1", expected)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.SourceQualification.Passed {
		t.Fatalf("source qualification=%+v", summary.SourceQualification)
	}
	if !summary.SemanticConformance.Passed {
		t.Fatalf("semantic conformance=%+v", summary.SemanticConformance)
	}
	if !summary.FunctionalEvidence.Passed {
		t.Fatalf("functional evidence=%+v", summary.FunctionalEvidence)
	}
	if summary.VisualEvidence.Status != "pending" || !summary.VisualEvidence.Required {
		t.Fatalf("visual evidence=%+v", summary.VisualEvidence)
	}
	if summary.GateCReady {
		t.Fatal("Semantic/Functional acquisition must not auto-declare Gate C Ready")
	}
}

func TestAuditArtifactsRejectsTargetRevisionMismatch(t *testing.T) {
	dir := t.TempDir()
	expected := Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
	writeAuditFixture(t, dir, "run-1", expected)

	var manifest manifestArtifact
	readAuditJSON(t, filepath.Join(dir, "manifest.json"), &manifest)
	manifest.TargetBuildRevision = "other-target"
	writeAuditJSON(t, filepath.Join(dir, "manifest.json"), manifest)

	if _, err := auditArtifacts(dir, "run-1", expected); err == nil {
		t.Fatal("mismatched targetBuildRevision was accepted")
	}
}

func TestAuditArtifactsRejectsCredentialMaterialInRawArtifact(t *testing.T) {
	dir := t.TempDir()
	expected := Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
	writeAuditFixture(t, dir, "run-1", expected)

	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-success.html"),
		[]byte("<html>editor-secret</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := auditArtifacts(dir, "run-1", expected); err == nil {
		t.Fatal("raw Evidence containing Reference credential material was accepted")
	}
}

func writeAuditFixture(t *testing.T, dir, runID string, p Provenance) {
	t.Helper()
	writeAuditJSON(t, filepath.Join(dir, "run-metadata.json"), runMetadata{
		RunIdentity:               runID,
		TargetBuildRevision:       p.TargetBuildRevision,
		RepositoryPreRunRevision:  p.TargetBuildRevision,
		RepositoryPreRunClean:     true,
		RepositoryPostRunRevision: p.TargetBuildRevision,
		RepositoryPostRunClean:    true,
		SourceQualificationPassed: true,
	})
	if err := os.WriteFile(filepath.Join(dir, "source-tests.log"), []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	writeAuditJSON(t, filepath.Join(dir, "manifest.json"), manifestArtifact{
		Provenance:    p,
		Stage:         "standard-and-custom-ui-shared-boundary",
		ApplicationID: "order_entry",
		Patterns: []manifestPatternArtifact{
			{
				ID:                    "order_entry.recent_orders",
				Kind:                  "browse",
				PresentationCandidate: "collection",
				Datasource: &manifestDatasource{
					Alias:  "sales",
					Target: "sales_core",
					Access: "read-write",
				},
			},
			{
				ID:                    "order_entry.order_record",
				Kind:                  "record",
				PresentationCandidate: "item",
			},
			{
				ID:                    "order_entry.customer_reference",
				Kind:                  "reference",
				PresentationCandidate: "relation-selection",
				Datasource: &manifestDatasource{
					Alias:  "master",
					Target: "master_data",
					Access: "read",
				},
			},
			{
				ID:                    "order_entry.order_lines",
				Kind:                  "header-detail",
				PresentationCandidate: "detail-collection",
			},
			{
				ID:                    "order_entry.submit_order",
				Kind:                  "command",
				PresentationCandidate: "command-invocation",
				Capability:            "sales.order.submit",
			},
		},
	})

	writeAuditJSON(t, filepath.Join(dir, "read.json"), readEnvelope{
		Provenance: p,
		Result: readArtifact{
			ScenarioID:    "order-entry.read-only-web-slice",
			ApplicationID: "order_entry",
			RuntimeBindings: []runtimeBindingArtifact{
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
		},
	})

	writeAuditJSON(t, filepath.Join(dir, "submit-success.json"), commandEnvelope{
		Provenance: p,
		Result: commandArtifact{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "completed",
			Authentication: &businessResultArtifact{
				Capability: "security.authenticate",
				Mode:       "outcome",
				Outcome:    "authenticated",
			},
			Validations: []validationResultArtifact{
				{Validation: "sales.order_line.quantity_positive", Result: "satisfied"},
				{Validation: "sales.order.lines.line_number_unique", Result: "satisfied"},
			},
			Authorization: &businessResultArtifact{
				Capability: "security.authorize",
				Mode:       "outcome",
				Outcome:    "allowed",
			},
			Command: &businessResultArtifact{
				Capability: "sales.order.submit",
				Mode:       "output",
			},
			Observation: &observationArtifact{
				ReadIdentity: "synthetic.order_state.read",
				Nature:       "synthetic",
				Result:       "submitted",
			},
		},
	})

	writeAuditJSON(t, filepath.Join(dir, "submit-authentication-denied.json"), commandEnvelope{
		Provenance: p,
		Result: commandArtifact{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "authentication-denied",
			Authentication: &businessResultArtifact{
				Capability: "security.authenticate",
				Mode:       "outcome",
				Outcome:    "denied",
			},
		},
	})

	writeAuditJSON(t, filepath.Join(dir, "submit-authorization-denied.json"), commandEnvelope{
		Provenance: p,
		Result: commandArtifact{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "authorization-denied",
			Authentication: &businessResultArtifact{
				Capability: "security.authenticate",
				Mode:       "outcome",
				Outcome:    "authenticated",
			},
			Authorization: &businessResultArtifact{
				Capability: "security.authorize",
				Mode:       "outcome",
				Outcome:    "denied",
			},
		},
	})

	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-get.html"),
		[]byte(
			"<html>Custom Submit UI action=\"/custom-submit\" "+
				p.HandoverBaselineID+" "+
				p.BaselineRepositoryRevision+" "+
				p.TargetBuildRevision+"</html>",
		),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-success.html"),
		[]byte("<html>order-entry.submit-order-web-slice sales.order.submit synthetic.order_state.read submitted</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
}

func writeAuditJSON(t *testing.T, path string, value any) {
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

func readAuditJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}
