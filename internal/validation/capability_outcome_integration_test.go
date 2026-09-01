package validation

import "testing"

func TestValidateIncludesCapabilityOutcomeDiagnosticsOnce(t *testing.T) {
	w := capabilityOutcomeWorld()
	w.Semantics[1].Capabilities[0].Output = "security.authenticated_subject"

	diagnostics := Validate(w)
	matches := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "SEMANTIC_CAPABILITY_OUTPUT_OUTCOME_CONFLICT" {
			matches++
			if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-capability" || diagnostic.Subject.ID != "security.authenticate" {
				t.Fatalf("共通ValidateでCapability Subjectが失われました: %#v", diagnostic.Subject)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("Outcome conflict Diagnosticの件数=%d want=1: %#v", matches, diagnostics)
	}
}
