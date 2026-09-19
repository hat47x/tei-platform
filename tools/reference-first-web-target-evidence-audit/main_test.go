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
	if summary.TargetDifferenceHandoff.ApplicationID != "order_entry" ||
		summary.TargetDifferenceHandoff.TargetAdapter != "reference-first-web-target" ||
		summary.TargetDifferenceHandoff.Host != "go-net-http" ||
		len(summary.TargetDifferenceHandoff.RuntimeBindings) != 3 ||
		summary.TargetDifferenceHandoff.ExecutionArtifact != "go-run-native-process" ||
		summary.TargetDifferenceHandoff.ABI != "linux/amd64" {
		t.Fatalf("target difference handoff=%+v", summary.TargetDifferenceHandoff)
	}
	if len(summary.TargetDifferenceHandoff.TargetCapabilities) == 0 ||
		len(summary.TargetDifferenceHandoff.DoNotCanonicalize) == 0 ||
		len(summary.TargetDifferenceHandoff.Unresolved) == 0 {
		t.Fatalf("target difference handoff is incomplete: %+v", summary.TargetDifferenceHandoff)
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
		GoVersion:                 "go version go1.25.0 linux/amd64",
		GOOS:                      "linux",
		GOARCH:                    "amd64",
		ServerLaunchMode:          "go-run",
		NetworkBind:               "loopback",
	})
	if err := os.WriteFile(filepath.Join(dir, "source-tests.log"), []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "health.txt"), []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	serverLog := "TEI first Web Target projection: http://127.0.0.1:18080/ application=order_entry baseline=" +
		p.HandoverBaselineID + " R=" + p.BaselineRepositoryRevision + " T=" + p.TargetBuildRevision + "\n"
	if err := os.WriteFile(filepath.Join(dir, "server.log"), []byte(serverLog), 0o600); err != nil {
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

	writeAuditJSON(t, filepath.Join(dir, "submit-validation-not-satisfied.json"), commandEnvelope{
		Provenance: p,
		Result: commandArtifact{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "validation-not-satisfied",
			Authentication: &businessResultArtifact{
				Capability: "security.authenticate",
				Mode:       "outcome",
				Outcome:    "authenticated",
			},
			Validations: []validationResultArtifact{
				{Validation: "sales.order_line.quantity_positive", Result: "not-satisfied"},
			},
		},
	})

	writeAuditJSON(t, filepath.Join(dir, "submit-execution-error.json"), commandErrorEnvelope{
		Provenance: p,
		Result: commandArtifact{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "command",
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
		},
		ExecutionError: "capability sales.order.submitの実行を完了できませんでした: synthetic submit adapter unavailable for order 1003",
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
		[]byte("<html>Scenario state completed Selected order and relations Customer reference Order lines Canonical Validation Command Business Result sales.order.submit Separate Observation synthetic.order_state.read submitted</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-authorization-denied.html"),
		[]byte("<html>authorization-denied Authorization security.authorize denied not_authorized</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-validation-not-satisfied.html"),
		[]byte("<html>validation-not-satisfied Canonical Validation sales.order_line.quantity_positive not-satisfied</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-execution-error.html"),
		[]byte("<html>Scenario state command Authorization allowed Execution error synthetic submit adapter unavailable for order 1003</html>"),
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
