package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestBusinessValidationRejectsRelationAttributeOutsideTarget(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{
				{
					ID: "sales.order",
					Relations: []semantic.Relation{{ID: "sales.order.lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany}},
				},
				{
					ID:         "sales.order_line",
					Attributes: []semantic.Attribute{{ID: "sales.order_line.line_number", TypeRef: "std.integer"}},
				},
				{
					ID:         "sales.other",
					Attributes: []semantic.Attribute{{ID: "sales.other.line_number", TypeRef: "std.integer"}},
				},
			},
			Validations: []semantic.Validation{{
				ID:        "sales.order.lines_line_number_unique",
				Kind:      semantic.ValidationRelationAttributeUnique,
				Relation:  "sales.order.lines",
				Attribute: "sales.other.line_number",
			}},
		}},
	}

	diagnostics := validateBusinessValidationReferences(world)
	if !hasDiagnosticCode(diagnostics, "SEMANTIC_VALIDATION_ATTRIBUTE_RELATION_MISMATCH") {
		t.Fatalf("Relation target外の属性を拒否していません: %#v", diagnostics)
	}
}

func TestApplicationValidationRequiresExistingRuleFromUsedSemantic(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID:         "sales.order_line",
				Attributes: []semantic.Attribute{{ID: "sales.order_line.quantity", TypeRef: "std.decimal"}},
			}},
			Validations: []semantic.Validation{{
				ID:        "sales.order_line.quantity_positive",
				Kind:      semantic.ValidationAttributePositive,
				Attribute: "sales.order_line.quantity",
			}},
		}},
		Applications: []application.Application{
			{ID: "missing", SemanticUses: []core.CanonicalID{"sales"}, Validations: []core.CanonicalID{"sales.unknown"}},
			{ID: "not_using_sales", Validations: []core.CanonicalID{"sales.order_line.quantity_positive"}},
		},
	}

	diagnostics := validateBusinessValidationReferences(world)
	if !hasDiagnosticCode(diagnostics, "UNKNOWN_APPLICATION_VALIDATION") {
		t.Fatalf("未知Validation参照を拒否していません: %#v", diagnostics)
	}
	if !hasDiagnosticCode(diagnostics, "APPLICATION_VALIDATION_SEMANTIC_NOT_USED") {
		t.Fatalf("未利用SemanticのValidation参照を拒否していません: %#v", diagnostics)
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
