package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/subject"
	"tei/internal/validation"
)

func TestRepresentativeDiagnosticsCarryStructuredSubjects(t *testing.T) {
	tests := []struct {
		name  string
		world ir.World
		code  string
		want  subject.Subject
	}{
		{
			name: "semantic type",
			world: ir.World{Semantics: []semantic.Namespace{{
				ID: "sales",
				Types: []semantic.Type{{
					ID:         "other.order_id",
					Kind:       semantic.TypeAlias,
					Underlying: "std.text",
				}},
			}}},
			code: "SEMANTIC_TYPE_NAMESPACE_MISMATCH",
			want: subject.Subject{Kind: "semantic-type", ID: "other.order_id"},
		},
		{
			name: "unknown semantic attribute type",
			world: ir.World{Semantics: []semantic.Namespace{{
				ID: "intake",
				Data: []semantic.Data{{
					ID:         "intake.request",
					Attributes: []semantic.Attribute{{ID: "intake.request.summary", TypeRef: "intake.unknown_type"}},
				}},
			}}},
			code: "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE",
			want: subject.Subject{Kind: "semantic-attribute", ID: "intake.request.summary"},
		},
		{
			name: "unknown relation target",
			world: ir.World{Semantics: []semantic.Namespace{{
				ID: "intake",
				Data: []semantic.Data{{
					ID: "intake.request",
					Relations: []semantic.Relation{{
						ID: "intake.request.parent", Target: "intake.missing", Cardinality: semantic.CardinalityOne,
					}},
				}},
			}}},
			code: "UNKNOWN_SEMANTIC_RELATION_TARGET",
			want: subject.Subject{Kind: "semantic-relation", ID: "intake.request.parent"},
		},
		{
			name: "unknown capability input",
			world: ir.World{Semantics: []semantic.Namespace{{
				ID: "intake",
				Capabilities: []semantic.Capability{{
					ID:     "intake.request.submit",
					Inputs: []semantic.Parameter{{Name: "request", Type: "intake.missing"}},
				}},
			}}},
			code: "UNKNOWN_SEMANTIC_CAPABILITY_INPUT",
			want: subject.Subject{Kind: "semantic-capability", ID: "intake.request.submit"},
		},
		{
			name: "unknown capability output",
			world: ir.World{Semantics: []semantic.Namespace{{
				ID: "intake",
				Capabilities: []semantic.Capability{{
					ID: "intake.request.submit", Output: "intake.missing",
				}},
			}}},
			code: "UNKNOWN_SEMANTIC_CAPABILITY_OUTPUT",
			want: subject.Subject{Kind: "semantic-capability", ID: "intake.request.submit"},
		},
		{
			name: "unknown application semantic namespace",
			world: ir.World{Applications: []application.Application{{
				ID: "request_log", SemanticUses: []core.CanonicalID{"intake_missing"},
			}}},
			code: "UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE",
			want: subject.Subject{Kind: "application", ID: "request_log"},
		},
		{
			name: "database",
			world: ir.World{Databases: []database.Database{{
				ID:        "sales_core",
				Ownership: database.Ownership("invalid"),
			}}},
			code: "INVALID_DATABASE_OWNERSHIP",
			want: subject.Subject{Kind: "database", ID: "sales_core"},
		},
		{
			name: "plugin",
			world: ir.World{Plugins: []plugin.Definition{{
				ID:      "tei.test.plugin",
				Version: "1.0.0",
				Runtime: plugin.Runtime{Kind: plugin.RuntimeKind("invalid")},
			}}},
			code: "INVALID_PLUGIN_RUNTIME",
			want: subject.Subject{Kind: "plugin", ID: "tei.test.plugin@1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostic, ok := findDiagnostic(validation.Validate(tt.world), tt.code)
			if !ok {
				t.Fatalf("missing diagnostic %s", tt.code)
			}
			if diagnostic.Subject == nil {
				t.Fatalf("diagnostic %s has no subject", tt.code)
			}
			if *diagnostic.Subject != tt.want {
				t.Fatalf("subject=%+v want=%+v", *diagnostic.Subject, tt.want)
			}
		})
	}
}

func findDiagnostic(diagnostics []validation.Diagnostic, code string) (validation.Diagnostic, bool) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic, true
		}
	}
	return validation.Diagnostic{}, false
}
