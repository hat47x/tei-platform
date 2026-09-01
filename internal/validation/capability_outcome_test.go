package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateCapabilityOutcomesAcceptsNamedBusinessResults(t *testing.T) {
	w := capabilityOutcomeWorld()

	if ds := validateCapabilityOutcomes(w); len(ds) != 0 {
		t.Fatalf("正常なnamed OutcomeにDiagnosticが返りました: %#v", ds)
	}
}

func TestValidateCapabilityOutcomesRejectsOutputAndOutcomeTogether(t *testing.T) {
	w := capabilityOutcomeWorld()
	w.Semantics[1].Capabilities[0].Output = "security.authenticated_subject"

	ds := validateCapabilityOutcomes(w)
	assertCapabilityOutcomeDiagnostic(t, ds, "SEMANTIC_CAPABILITY_OUTPUT_OUTCOME_CONFLICT")
}

func TestValidateCapabilityOutcomesRejectsDuplicateNames(t *testing.T) {
	w := capabilityOutcomeWorld()
	w.Semantics[1].Capabilities[0].Outcomes = append(
		w.Semantics[1].Capabilities[0].Outcomes,
		semantic.Outcome{Name: "denied", Payload: "security.authenticated_subject"},
	)

	ds := validateCapabilityOutcomes(w)
	assertCapabilityOutcomeDiagnostic(t, ds, "DUPLICATE_SEMANTIC_CAPABILITY_OUTCOME")
}

func TestValidateCapabilityOutcomesRequiresPayload(t *testing.T) {
	w := capabilityOutcomeWorld()
	w.Semantics[1].Capabilities[0].Outcomes[0].Payload = ""

	ds := validateCapabilityOutcomes(w)
	assertCapabilityOutcomeDiagnostic(t, ds, "SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD_REQUIRED")
}

func TestValidateCapabilityOutcomesRejectsUnknownPayload(t *testing.T) {
	w := capabilityOutcomeWorld()
	w.Semantics[1].Capabilities[0].Outcomes[0].Payload = "security.missing"

	ds := validateCapabilityOutcomes(w)
	assertCapabilityOutcomeDiagnostic(t, ds, "UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD")
}

func capabilityOutcomeWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{
			ID: "std",
			Types: []semantic.Type{
				{ID: "std.identifier", Kind: semantic.TypePrimitive},
				{ID: "std.code", Kind: semantic.TypePrimitive},
			},
		},
		{
			ID: "security",
			Types: []semantic.Type{
				{ID: "security.authenticated_subject", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
				{ID: "security.authentication_denial", Kind: semantic.TypeAlias, Underlying: "std.code"},
			},
			Capabilities: []semantic.Capability{{
				ID: "security.authenticate",
				Outcomes: []semantic.Outcome{
					{Name: "authenticated", Payload: "security.authenticated_subject"},
					{Name: "denied", Payload: "security.authentication_denial"},
				},
			}},
		},
	}}
}

func assertCapabilityOutcomeDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != code {
			continue
		}
		if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-capability" || diagnostic.Subject.ID != "security.authenticate" {
			t.Fatalf("Diagnostic Subjectが期待値と異なります: %#v", diagnostic.Subject)
		}
		return
	}
	t.Fatalf("Diagnostic %sが見つかりません: %#v", code, diagnostics)
}
