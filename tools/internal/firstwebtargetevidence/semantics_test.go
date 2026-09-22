package firstwebtargetevidence

import (
	"encoding/json"
	"testing"
)

type validatorCase struct {
	name     string
	id       string
	class    string
	validate func([]byte, Provenance) error
}

func validators() []validatorCase {
	return []validatorCase{
		{"read", "first-web/read-only", ObservationClassReadResult, ValidateReadArtifact},
		{"command-success", "first-web/command-success", ObservationClassBusinessResultPlusObservation, ValidateCommandSuccessArtifact},
		{"authentication-denied", "first-web/authentication-denied", ObservationClassAuthenticationDenied, ValidateAuthenticationDeniedArtifact},
		{"authorization-denied", "first-web/authorization-denied", ObservationClassAuthorizationDenied, ValidateAuthorizationDeniedArtifact},
		{"validation-not-satisfied", "first-web/validation-not-satisfied", ObservationClassValidationNotSatisfied, ValidateValidationNotSatisfiedArtifact},
		{"execution-error", "first-web/execution-error", ObservationClassExecutionError, ValidateExecutionErrorArtifact},
	}
}

func TestExplicitValidatorsAcceptCurrentGateCBoundaries(t *testing.T) {
	p := Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
	for _, tc := range validators() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.class == "" {
				t.Fatal("empty Observation class")
			}
			data := behaviorFixture(t, tc.id, p)
			if err := tc.validate(data, p); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExplicitValidatorsRejectExactProvenanceDrift(t *testing.T) {
	p := Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
	for _, tc := range validators() {
		t.Run(tc.name, func(t *testing.T) {
			data := behaviorFixture(t, tc.id, p)
			var envelope map[string]any
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatal(err)
			}
			envelope["targetBuildRevision"] = "other-target"
			drifted, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.validate(drifted, p); err == nil {
				t.Fatal("target provenance drift was accepted")
			}
		})
	}
}

func TestRejectReferenceCredentialMaterial(t *testing.T) {
	for _, forbidden := range []string{
		"editor-secret", "viewer-secret", "wrong-secret", "locked-secret", "disabled-secret", "credential-ref://legacy/",
	} {
		if err := RejectReferenceCredentialMaterial("artifact.json", []byte("prefix "+forbidden+" suffix")); err == nil {
			t.Fatalf("credential material %q was accepted", forbidden)
		}
	}
	if err := RejectReferenceCredentialMaterial("artifact.json", []byte(`{"result":"safe"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestObservationClassesRemainExact(t *testing.T) {
	got := []string{
		ObservationClassReadResult,
		ObservationClassBusinessResultPlusObservation,
		ObservationClassAuthenticationDenied,
		ObservationClassAuthorizationDenied,
		ObservationClassValidationNotSatisfied,
		ObservationClassExecutionError,
	}
	want := []string{
		"read-result",
		"business-result-plus-separate-observation",
		"business-denial-authentication",
		"business-denial-authorization",
		"validation-not-satisfied",
		"command-invocation-execution-error",
	}
	if len(got) != len(want) {
		t.Fatalf("class count=%d", len(got))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("class[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}

func behaviorFixture(t *testing.T, id string, p Provenance) []byte {
	t.Helper()

	base := map[string]any{
		"handoverBaselineId":         p.HandoverBaselineID,
		"baselineRepositoryRevision": p.BaselineRepositoryRevision,
		"targetBuildRevision":        p.TargetBuildRevision,
		"targetAdapter":              p.TargetAdapter,
		"host":                       p.Host,
	}

	switch id {
	case "first-web/read-only":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.read-only-web-slice",
			"applicationId": "order_entry",
			"runtimeBindings": []any{
				map[string]any{
					"pattern":    "order_entry.recent_orders",
					"datasource": "sales",
					"database":   "sales_core",
					"access":     "read-write",
				},
				map[string]any{
					"pattern":            "order_entry.customer_reference",
					"environment":        "development",
					"instance":           "master_data.dev",
					"provider":           "oracle",
					"executorCapability": "database.query.oracle",
					"contractQuery":      "master_data/master_data.customer/find_by_id",
				},
				map[string]any{
					"pattern":            "order_entry.order_lines",
					"environment":        "development",
					"instance":           "sales_core.dev",
					"provider":           "postgres",
					"executorCapability": "database.query.postgres",
					"contractQuery":      "sales_core/sales_core.order_line/find_by_order",
				},
			},
		}
	case "first-web/command-success":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.submit-order-web-slice",
			"applicationId": "order_entry",
			"stage":         "completed",
			"authentication": map[string]any{
				"capability": "security.authenticate",
				"mode":       "outcome",
				"outcome":    "authenticated",
			},
			"validations": []any{
				map[string]any{"validation": "sales.order_line.quantity_positive", "result": "satisfied"},
				map[string]any{"validation": "sales.order.lines.line_number_unique", "result": "satisfied"},
			},
			"authorization": map[string]any{
				"capability": "security.authorize",
				"mode":       "outcome",
				"outcome":    "allowed",
			},
			"command": map[string]any{
				"capability": "sales.order.submit",
				"mode":       "output",
			},
			"observation": map[string]any{
				"readIdentity": "synthetic.order_state.read",
				"nature":       "synthetic",
				"result":       "submitted",
			},
		}
	case "first-web/authentication-denied":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.submit-order-web-slice",
			"applicationId": "order_entry",
			"stage":         "authentication-denied",
			"authentication": map[string]any{
				"capability": "security.authenticate",
				"mode":       "outcome",
				"outcome":    "denied",
			},
		}
	case "first-web/authorization-denied":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.submit-order-web-slice",
			"applicationId": "order_entry",
			"stage":         "authorization-denied",
			"authentication": map[string]any{
				"capability": "security.authenticate",
				"mode":       "outcome",
				"outcome":    "authenticated",
			},
			"authorization": map[string]any{
				"capability": "security.authorize",
				"mode":       "outcome",
				"outcome":    "denied",
			},
		}
	case "first-web/validation-not-satisfied":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.submit-order-web-slice",
			"applicationId": "order_entry",
			"stage":         "validation-not-satisfied",
			"authentication": map[string]any{
				"capability": "security.authenticate",
				"mode":       "outcome",
				"outcome":    "authenticated",
			},
			"validations": []any{
				map[string]any{"validation": "sales.order_line.quantity_positive", "result": "not-satisfied"},
			},
		}
	case "first-web/execution-error":
		base["result"] = map[string]any{
			"scenarioId":    "order-entry.submit-order-web-slice",
			"applicationId": "order_entry",
			"stage":         "command",
			"authentication": map[string]any{
				"capability": "security.authenticate",
				"mode":       "outcome",
				"outcome":    "authenticated",
			},
			"validations": []any{
				map[string]any{"validation": "sales.order_line.quantity_positive", "result": "satisfied"},
				map[string]any{"validation": "sales.order.lines.line_number_unique", "result": "satisfied"},
			},
			"authorization": map[string]any{
				"capability": "security.authorize",
				"mode":       "outcome",
				"outcome":    "allowed",
			},
		}
		base["executionError"] = "capability sales.order.submitの実行を完了できませんでした: synthetic submit adapter unavailable for order 1003"
	default:
		t.Fatalf("unknown fixture behavior %q", id)
	}

	data, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
