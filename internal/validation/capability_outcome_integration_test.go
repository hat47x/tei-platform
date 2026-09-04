package validation

import (
	"tei/internal/ir"
	"tei/internal/semantic"
	"testing"
)

func TestValidateIncludesCapabilityOutcomeDiagnosticsOnce(t *testing.T) {
	w := ir.World{Semantics: []semantic.Namespace{{ID: "security", Data: []semantic.Data{{ID: "security.authenticated_subject"}}, Capabilities: []semantic.Capability{{ID: "security.authenticate", Outcomes: []semantic.Outcome{{Name: "authenticated", Payload: "security.authenticated_subject"}}}}}}}
	w.Semantics[0].Capabilities[0].Output = "security.authenticated_subject"

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
