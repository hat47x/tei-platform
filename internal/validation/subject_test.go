package validation_test

import (
	"testing"

	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/subject"
	"tei/internal/validation"
)

func TestRepresentativeDiagnosticsCarryStructuredSubjects(t *testing.T) {
	tests := []struct {
		name string
		world ir.World
		code string
		want subject.Subject
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
