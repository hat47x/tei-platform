package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestValidateAcceptsKnownQueryResultAttribute(t *testing.T) {
	world := queryResultMappingWorld("sales.order.total")

	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "UNKNOWN_QUERY_RESULT_ATTRIBUTE" {
			t.Fatalf("存在するSemantic attributeへのmappingを拒否しています: %#v", diagnostic)
		}
	}
}

func TestValidateReportsUnknownQueryResultAttributeWithQuerySubject(t *testing.T) {
	world := queryResultMappingWorld("sales.order.missing")

	var found []validation.Diagnostic
	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "UNKNOWN_QUERY_RESULT_ATTRIBUTE" {
			found = append(found, diagnostic)
		}
	}
	if len(found) != 1 {
		t.Fatalf("UNKNOWN_QUERY_RESULT_ATTRIBUTE diagnostics=%#v", found)
	}
	got := found[0]
	if got.Subject == nil || got.Subject.Kind != "query" || got.Subject.ID != "order_entry.recent_orders" {
		t.Fatalf("Diagnostic subject=%#v", got.Subject)
	}
}

func queryResultMappingWorld(attribute core.CanonicalID) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "std",
				Types: []semantic.Type{
					{ID: "std.money", Symbol: "money", Kind: semantic.TypePrimitive},
				},
			},
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID:     "sales.order",
						Symbol: "Order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.total", Symbol: "total", TypeRef: "std.money"},
						},
					},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{
				ID: "order_entry.recent_orders",
				ResultColumns: []query.ResultColumnMapping{
					{Column: "total_amount", Attribute: attribute},
				},
			},
		},
	}
}
