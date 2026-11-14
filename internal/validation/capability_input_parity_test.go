package validation_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/syntax"
	"tei/internal/validation"
)

type authoringCapabilityAdapter struct{ calls int }

func (a *authoringCapabilityAdapter) Invoke(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
	a.calls++
	return capabilityinvoke.BusinessResult{Capability: id, Mode: capabilityinvoke.ResultOutput, Payload: "account"}, nil
}

// ACD-02/03 use real DSL Source through the parser and the Workspace
// Validator; the runtime refusal is checked independently using the parsed
// Canonical Capability. A diagnostic Subject is not a fabricated byte span.
func TestCapabilitySourceValidationAndPreDispatchAgreeOnMalformedDeclarations(t *testing.T) {
	cases := []struct {
		name      string
		members   []string
		wantCode  string
		wantCalls int
	}{
		{
			name: "ACD-02 duplicate input",
			members: []string{
				"input token security.account",
				"input token security.account",
				"output security.account",
			},
			wantCode: "DUPLICATE_SEMANTIC_CAPABILITY_INPUT",
		},
		{
			name: "ACD-03 output mixed with named outcome",
			members: []string{
				"input token security.account",
				"output security.account",
				"outcome accepted security.account",
			},
			wantCode: "SEMANTIC_CAPABILITY_OUTPUT_OUTCOME_CONFLICT",
		},
		{
			name: "repaired input and single output",
			members: []string{
				"input token security.account",
				"output security.account",
			},
			wantCalls: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "namespace security\n" +
				"data Account {}\n" +
				"capability security.inspect {\n" +
				strings.Join(tc.members, "\n") + "\n}\n"
			namespace, err := syntax.NewParser(source).ParseSemanticFile()
			if err != nil {
				t.Fatalf("syntax should parse this declaration so Workspace checks can run: %v", err)
			}
			if len(namespace.Capabilities) != 1 {
				t.Fatalf("expected one parsed Capability, got %d", len(namespace.Capabilities))
			}
			diagnostics := validation.Validate(ir.World{Semantics: []semantic.Namespace{namespace}})
			count := 0
			for _, d := range diagnostics {
				if d.Code == tc.wantCode && tc.wantCode != "" {
					count++
					if d.Subject == nil || d.Subject.Kind != "semantic-capability" || d.Subject.ID != "security.inspect" {
						t.Fatalf("diagnostic lost its Capability subject: %+v", d)
					}
				}
				if tc.wantCode == "" && d.Severity == validation.Error {
					t.Fatalf("a repaired declaration should not be rejected: %+v", d)
				}
			}
			if tc.wantCode != "" && count != 1 {
				t.Fatalf("expected one %s diagnostic, got %d: %+v", tc.wantCode, count, diagnostics)
			}

			adapter := &authoringCapabilityAdapter{}
			_, err = capabilityinvoke.Invoke(context.Background(), namespace.Capabilities[0], adapter, "token")
			if tc.wantCode != "" {
				if err == nil || adapter.calls != 0 {
					t.Fatalf("invalid declaration reached Adapter: err=%v calls=%d", err, adapter.calls)
				}
			} else if err != nil || adapter.calls != tc.wantCalls {
				t.Fatalf("repaired declaration not dispatched once: err=%v calls=%d", err, adapter.calls)
			}
		})
	}
}

func TestCapabilityCanonicalInputNameRequiredInWorkspaceValidation(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "security",
		Data: []semantic.Data{{ID: "security.account"}},
		Capabilities: []semantic.Capability{{
			ID:     "security.inspect",
			Inputs: []semantic.Parameter{{Name: "", Type: "security.account"}},
			Output: "security.account",
		}},
	}}}
	ds := validation.Validate(world)
	if !hasDiagnosticCode(ds, "SEMANTIC_CAPABILITY_INPUT_NAME_REQUIRED") {
		t.Fatalf("missing required-name diagnostic for canonical bypass: %+v", ds)
	}
}
