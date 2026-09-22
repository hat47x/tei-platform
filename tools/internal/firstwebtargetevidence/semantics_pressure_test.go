package firstwebtargetevidence

import (
	"encoding/json"
	"testing"
)

type pressureValidator struct {
	name     string
	kind     string
	validate func([]byte, Provenance) error
}

func pressureValidators() []pressureValidator {
	return []pressureValidator{
		{"read", "read", ValidateReadArtifact},
		{"success", "success", ValidateCommandSuccessArtifact},
		{"authentication-denied", "authentication-denied", ValidateAuthenticationDeniedArtifact},
		{"authorization-denied", "authorization-denied", ValidateAuthorizationDeniedArtifact},
		{"validation-not-satisfied", "validation-not-satisfied", ValidateValidationNotSatisfiedArtifact},
		{"execution-error", "execution-error", ValidateExecutionErrorArtifact},
	}
}

func pressureProvenance() Provenance {
	return Provenance{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	}
}

func TestPressureRejectsEveryProvenanceFieldEmptyOrDrift(t *testing.T) {
	p := pressureProvenance()
	mutations := []struct {
		name  string
		field string
		value string
	}{
		{"baseline-id-empty", "handoverBaselineId", ""},
		{"baseline-id-drift", "handoverBaselineId", "other-B"},
		{"baseline-revision-empty", "baselineRepositoryRevision", ""},
		{"baseline-revision-drift", "baselineRepositoryRevision", "other-R"},
		{"target-revision-empty", "targetBuildRevision", ""},
		{"target-revision-drift", "targetBuildRevision", "other-T"},
		{"adapter-empty", "targetAdapter", ""},
		{"adapter-drift", "targetAdapter", "other-adapter"},
		{"host-empty", "host", ""},
		{"host-drift", "host", "other-host"},
	}
	for _, validator := range pressureValidators() {
		for _, mutation := range mutations {
			t.Run(validator.name+"/"+mutation.name, func(t *testing.T) {
				doc := pressureFixture(t, validator.kind, p)
				doc[mutation.field] = mutation.value
				if err := validator.validate(marshalPressure(t, doc), p); err == nil {
					t.Fatalf("accepted %s provenance mutation", mutation.name)
				}
			})
		}
	}
}

func TestPressureReadBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"scenario", func(doc map[string]any) { result(doc)["scenarioId"] = "other" }},
		{"application", func(doc map[string]any) { result(doc)["applicationId"] = "other" }},
		{"browse-binding", func(doc map[string]any) { binding(doc, 0)["datasource"] = "other" }},
		{"browse-inferred-environment", func(doc map[string]any) { binding(doc, 0)["environment"] = "development" }},
		{"customer-binding", func(doc map[string]any) { binding(doc, 1)["provider"] = "postgres" }},
		{"order-lines-binding", func(doc map[string]any) { binding(doc, 2)["contractQuery"] = "other/query" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "read", p)
			tc.mutate(doc)
			if err := ValidateReadArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted read boundary drift: %s", tc.name)
			}
		})
	}
}

func TestPressureCommandSuccessBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"stage", func(doc map[string]any) { result(doc)["stage"] = "other" }},
		{"authentication-missing", func(doc map[string]any) { delete(result(doc), "authentication") }},
		{"authentication-outcome", func(doc map[string]any) { obj(result(doc), "authentication")["outcome"] = "denied" }},
		{"validation-count", func(doc map[string]any) { result(doc)["validations"] = []any{validation("a", "satisfied")} }},
		{"validation-result", func(doc map[string]any) { sliceObj(result(doc), "validations", 0)["result"] = "not-satisfied" }},
		{"authorization-missing", func(doc map[string]any) { delete(result(doc), "authorization") }},
		{"authorization-outcome", func(doc map[string]any) { obj(result(doc), "authorization")["outcome"] = "denied" }},
		{"command-missing", func(doc map[string]any) { delete(result(doc), "command") }},
		{"command-capability", func(doc map[string]any) { obj(result(doc), "command")["capability"] = "other" }},
		{"command-mode", func(doc map[string]any) { obj(result(doc), "command")["mode"] = "other" }},
		{"observation-missing", func(doc map[string]any) { delete(result(doc), "observation") }},
		{"observation-identity", func(doc map[string]any) { obj(result(doc), "observation")["readIdentity"] = "other" }},
		{"observation-nature", func(doc map[string]any) { obj(result(doc), "observation")["nature"] = "other" }},
		{"observation-result", func(doc map[string]any) { obj(result(doc), "observation")["result"] = "other" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "success", p)
			tc.mutate(doc)
			if err := ValidateCommandSuccessArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted success boundary drift: %s", tc.name)
			}
		})
	}
}

func TestPressureAuthenticationDeniedBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"stage", func(doc map[string]any) { result(doc)["stage"] = "other" }},
		{"authentication-missing", func(doc map[string]any) { delete(result(doc), "authentication") }},
		{"authentication-outcome", func(doc map[string]any) { obj(result(doc), "authentication")["outcome"] = "authenticated" }},
		{"command-present", func(doc map[string]any) { result(doc)["command"] = map[string]any{"capability": "sales.order.submit"} }},
		{"observation-present", func(doc map[string]any) { result(doc)["observation"] = map[string]any{"result": "x"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "authentication-denied", p)
			tc.mutate(doc)
			if err := ValidateAuthenticationDeniedArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted authentication-denied drift: %s", tc.name)
			}
		})
	}
}

func TestPressureAuthorizationDeniedBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"stage", func(doc map[string]any) { result(doc)["stage"] = "other" }},
		{"authorization-missing", func(doc map[string]any) { delete(result(doc), "authorization") }},
		{"authorization-outcome", func(doc map[string]any) { obj(result(doc), "authorization")["outcome"] = "allowed" }},
		{"command-present", func(doc map[string]any) { result(doc)["command"] = map[string]any{"capability": "sales.order.submit"} }},
		{"observation-present", func(doc map[string]any) { result(doc)["observation"] = map[string]any{"result": "x"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "authorization-denied", p)
			tc.mutate(doc)
			if err := ValidateAuthorizationDeniedArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted authorization-denied drift: %s", tc.name)
			}
		})
	}
}

func TestPressureValidationNotSatisfiedBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"stage", func(doc map[string]any) { result(doc)["stage"] = "other" }},
		{"validation-count", func(doc map[string]any) { result(doc)["validations"] = []any{} }},
		{"validation-identity", func(doc map[string]any) { sliceObj(result(doc), "validations", 0)["validation"] = "other" }},
		{"validation-result", func(doc map[string]any) { sliceObj(result(doc), "validations", 0)["result"] = "satisfied" }},
		{"authorization-present", func(doc map[string]any) { result(doc)["authorization"] = map[string]any{"outcome": "allowed"} }},
		{"command-present", func(doc map[string]any) { result(doc)["command"] = map[string]any{"capability": "sales.order.submit"} }},
		{"observation-present", func(doc map[string]any) { result(doc)["observation"] = map[string]any{"result": "x"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "validation-not-satisfied", p)
			tc.mutate(doc)
			if err := ValidateValidationNotSatisfiedArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted validation-not-satisfied drift: %s", tc.name)
			}
		})
	}
}

func TestPressureExecutionErrorBoundaryDrift(t *testing.T) {
	p := pressureProvenance()
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"stage", func(doc map[string]any) { result(doc)["stage"] = "other" }},
		{"authorization-missing", func(doc map[string]any) { delete(result(doc), "authorization") }},
		{"authorization-outcome", func(doc map[string]any) { obj(result(doc), "authorization")["outcome"] = "denied" }},
		{"command-present", func(doc map[string]any) { result(doc)["command"] = map[string]any{"capability": "sales.order.submit"} }},
		{"observation-present", func(doc map[string]any) { result(doc)["observation"] = map[string]any{"result": "x"} }},
		{"error-text", func(doc map[string]any) { doc["executionError"] = "different error" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := pressureFixture(t, "execution-error", p)
			tc.mutate(doc)
			if err := ValidateExecutionErrorArtifact(marshalPressure(t, doc), p); err == nil {
				t.Fatalf("accepted execution-error drift: %s", tc.name)
			}
		})
	}
}

func TestPressureCredentialMaterialRejectSet(t *testing.T) {
	for _, value := range []string{
		"editor-secret",
		"viewer-secret",
		"wrong-secret",
		"locked-secret",
		"disabled-secret",
		"credential-ref://legacy/",
	} {
		if err := RejectReferenceCredentialMaterial("artifact.json", []byte("prefix "+value+" suffix")); err == nil {
			t.Fatalf("accepted credential material %q", value)
		}
	}
}

func pressureFixture(t *testing.T, kind string, p Provenance) map[string]any {
	t.Helper()
	doc := map[string]any{
		"handoverBaselineId":         p.HandoverBaselineID,
		"baselineRepositoryRevision": p.BaselineRepositoryRevision,
		"targetBuildRevision":        p.TargetBuildRevision,
		"targetAdapter":              p.TargetAdapter,
		"host":                       p.Host,
	}
	switch kind {
	case "read":
		doc["result"] = map[string]any{
			"scenarioId":    "order-entry.read-only-web-slice",
			"applicationId": "order_entry",
			"runtimeBindings": []any{
				map[string]any{"pattern": "order_entry.recent_orders", "datasource": "sales", "database": "sales_core", "access": "read-write"},
				map[string]any{"pattern": "order_entry.customer_reference", "environment": "development", "instance": "master_data.dev", "provider": "oracle", "executorCapability": "database.query.oracle", "contractQuery": "master_data/master_data.customer/find_by_id"},
				map[string]any{"pattern": "order_entry.order_lines", "environment": "development", "instance": "sales_core.dev", "provider": "postgres", "executorCapability": "database.query.postgres", "contractQuery": "sales_core/sales_core.order_line/find_by_order"},
			},
		}
	case "success":
		doc["result"] = map[string]any{
			"scenarioId":     "order-entry.submit-order-web-slice",
			"applicationId":  "order_entry",
			"stage":          "completed",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"},
			"validations":    []any{validation("sales.order_line.quantity_positive", "satisfied"), validation("sales.order.lines.line_number_unique", "satisfied")},
			"authorization":  map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "allowed"},
			"command":        map[string]any{"capability": "sales.order.submit", "mode": "output"},
			"observation":    map[string]any{"readIdentity": "synthetic.order_state.read", "nature": "synthetic", "result": "submitted"},
		}
	case "authentication-denied":
		doc["result"] = map[string]any{
			"scenarioId":     "order-entry.submit-order-web-slice",
			"applicationId":  "order_entry",
			"stage":          "authentication-denied",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "denied"},
		}
	case "authorization-denied":
		doc["result"] = map[string]any{
			"scenarioId":     "order-entry.submit-order-web-slice",
			"applicationId":  "order_entry",
			"stage":          "authorization-denied",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"},
			"authorization":  map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "denied"},
		}
	case "validation-not-satisfied":
		doc["result"] = map[string]any{
			"scenarioId":     "order-entry.submit-order-web-slice",
			"applicationId":  "order_entry",
			"stage":          "validation-not-satisfied",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"},
			"validations":    []any{validation("sales.order_line.quantity_positive", "not-satisfied")},
		}
	case "execution-error":
		doc["result"] = map[string]any{
			"scenarioId":     "order-entry.submit-order-web-slice",
			"applicationId":  "order_entry",
			"stage":          "command",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"},
			"validations":    []any{validation("sales.order_line.quantity_positive", "satisfied"), validation("sales.order.lines.line_number_unique", "satisfied")},
			"authorization":  map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "allowed"},
		}
		doc["executionError"] = "capability sales.order.submit execution failed: synthetic submit adapter unavailable for order 1003"
	default:
		t.Fatalf("unknown fixture kind %q", kind)
	}
	return doc
}

func result(doc map[string]any) map[string]any {
	return doc["result"].(map[string]any)
}

func obj(parent map[string]any, key string) map[string]any {
	return parent[key].(map[string]any)
}

func sliceObj(parent map[string]any, key string, index int) map[string]any {
	return parent[key].([]any)[index].(map[string]any)
}

func binding(doc map[string]any, index int) map[string]any {
	return result(doc)["runtimeBindings"].([]any)[index].(map[string]any)
}

func validation(name, result string) map[string]any {
	return map[string]any{"validation": name, "result": result}
}

func marshalPressure(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
