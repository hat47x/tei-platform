package validation_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestQueryResultMappingValidationRequiresExistingSemanticAttribute(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.order_id", TypeRef: "std.identifier"},
						},
					},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{
				ID: "order_entry.recent_orders",
				ResultColumns: []query.ResultColumnMapping{
					{Column: "order_id", Attribute: "sales.order.order_id"},
					{Column: "unknown", Attribute: "sales.order.unknown"},
				},
			},
		},
	}

	diagnostics := validation.Validate(world)
	if !containsQueryResultMappingDiagnostic(diagnostics, "UNKNOWN_QUERY_RESULT_ATTRIBUTE") {
		t.Fatalf("存在しないSemantic attributeを検出できていません: %#v", diagnostics)
	}
}

func TestQueryResultMappingValidationKeepsQueryIdentityAsSubject(t *testing.T) {
	world := ir.World{
		Queries: []query.EmbeddedSQL{
			{
				ID: "order_entry.recent_orders",
				ResultColumns: []query.ResultColumnMapping{
					{Column: "order_id", Attribute: "sales.order.order_id"},
				},
			},
		},
	}

	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code != "UNKNOWN_QUERY_RESULT_ATTRIBUTE" {
			continue
		}
		if diagnostic.Subject == nil || diagnostic.Subject.Kind != "query" || diagnostic.Subject.ID != "order_entry.recent_orders" {
			t.Fatalf("Query result mappingのDiagnostic Subject=%#v", diagnostic.Subject)
		}
		return
	}
	t.Fatal("Query result mappingのDiagnosticがありません")
}

func TestQueryResultMappingValidationRejectsDuplicateCanonicalMappings(t *testing.T) {
	world := ir.World{
		Queries: []query.EmbeddedSQL{
			{
				ID: "order_entry.recent_orders",
				ResultColumns: []query.ResultColumnMapping{
					{Column: "order_id", Attribute: "sales.order.order_id"},
					{Column: "order_id", Attribute: "sales.order.order_id"},
				},
			},
		},
	}

	diagnostics := validation.Validate(world)
	if !containsQueryResultMappingDiagnostic(diagnostics, "DUPLICATE_QUERY_RESULT_COLUMN") {
		t.Fatalf("重複raw列mappingを検出できていません: %#v", diagnostics)
	}
	if !containsQueryResultMappingDiagnostic(diagnostics, "DUPLICATE_QUERY_RESULT_ATTRIBUTE") {
		t.Fatalf("重複Semantic attribute mappingを検出できていません: %#v", diagnostics)
	}
}

func containsQueryResultMappingDiagnostic(diagnostics []validation.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
