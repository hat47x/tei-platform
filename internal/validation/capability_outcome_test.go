package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestCapabilityOutcomeValidation(t *testing.T) {
	baseTypes := []semantic.Type{
		{ID: "security.authenticated_subject", Symbol: "AuthenticatedSubject", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
		{ID: "security.authentication_denial", Symbol: "AuthenticationDenial", Kind: semantic.TypeAlias, Underlying: "std.code"},
	}

	tests := []struct {
		name      string
		capability semantic.Capability
		wantCodes []string
	}{
		{
			name: "named outcomes are valid",
			capability: semantic.Capability{
				ID:     "security.authenticate",
				Symbol: "authenticate",
				Outcomes: []semantic.Outcome{
					{Name: "authenticated", Payload: "security.authenticated_subject"},
					{Name: "denied", Payload: "security.authentication_denial"},
				},
			},
		},
		{
			name: "output and outcomes are mutually exclusive",
			capability: semantic.Capability{
				ID:     "security.authenticate",
				Symbol: "authenticate",
				Output: "security.authenticated_subject",
				Outcomes: []semantic.Outcome{
					{Name: "denied", Payload: "security.authentication_denial"},
				},
			},
			wantCodes: []string{"SEMANTIC_CAPABILITY_OUTPUT_OUTCOME_CONFLICT"},
		},
		{
			name: "outcome names are unique within a capability",
			capability: semantic.Capability{
				ID:     "security.authenticate",
				Symbol: "authenticate",
				Outcomes: []semantic.Outcome{
					{Name: "denied", Payload: "security.authentication_denial"},
					{Name: "denied", Payload: "security.authentication_denial"},
				},
			},
			wantCodes: []string{"DUPLICATE_SEMANTIC_CAPABILITY_OUTCOME"},
		},
		{
			name: "outcome payload must resolve",
			capability: semantic.Capability{
				ID:     "security.authenticate",
				Symbol: "authenticate",
				Outcomes: []semantic.Outcome{
					{Name: "denied", Payload: "security.missing"},
				},
			},
			wantCodes: []string{"UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := ir.World{Semantics: []semantic.Namespace{
				{ID: "security", Types: baseTypes, Capabilities: []semantic.Capability{tt.capability}},
			}}
			diagnostics := validation.Validate(world)
			for _, code := range tt.wantCodes {
				if !hasDiagnosticCode(diagnostics, code) {
					t.Fatalf("diagnostics=%#v; want code %s", diagnostics, code)
				}
			}
			if len(tt.wantCodes) == 0 {
				for _, diagnostic := range diagnostics {
					if diagnostic.Severity == validation.Error {
						t.Fatalf("unexpected error diagnostic: %#v", diagnostic)
					}
				}
			}
		})
	}
}

func hasDiagnosticCode(diagnostics []validation.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

var _ core.CanonicalID
