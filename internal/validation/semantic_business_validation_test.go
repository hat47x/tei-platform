package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateSemanticBusinessValidationsAcceptsReferenceRules(t *testing.T) {
	w := validSemanticBusinessValidationWorld()

	if diagnostics := validateSemanticBusinessValidations(w); len(diagnostics) != 0 {
		t.Fatalf("正常な業務ValidationにDiagnosticが発生しました: %#v", diagnostics)
	}
}

func TestValidateSemanticBusinessValidationReportsUnknownAttributeOnRuleSubject(t *testing.T) {
	w := validSemanticBusinessValidationWorld()
	w.Semantics[1].Validations[0].Attribute = "sales.order_line.missing"

	d := semanticValidationDiagnosticByCode(validateSemanticBusinessValidations(w), "UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE")
	assertSemanticValidationSubject(t, d, "sales.order_line.quantity_positive")
}

func TestValidateSemanticPositiveRequiresNumericAttribute(t *testing.T) {
	w := validSemanticBusinessValidationWorld()
	w.Semantics[1].Validations[0].Attribute = "sales.order_line.item_code"

	d := semanticValidationDiagnosticByCode(validateSemanticBusinessValidations(w), "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE")
	assertSemanticValidationSubject(t, d, "sales.order_line.quantity_positive")
}

func TestValidateSemanticRelationUniqueRequiresKnownManyRelation(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
		edit func(*ir.World)
	}{
		{
			name: "unknown relation",
			code: "UNKNOWN_SEMANTIC_VALIDATION_RELATION",
			edit: func(w *ir.World) {
				w.Semantics[1].Validations[1].Relation = "sales.order.missing"
			},
		},
		{
			name: "one relation",
			code: "SEMANTIC_VALIDATION_RELATION_NOT_MANY",
			edit: func(w *ir.World) {
				w.Semantics[1].Data[0].Relations[0].Cardinality = semantic.CardinalityOne
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := validSemanticBusinessValidationWorld()
			tc.edit(&w)
			d := semanticValidationDiagnosticByCode(validateSemanticBusinessValidations(w), tc.code)
			assertSemanticValidationSubject(t, d, "sales.order.lines.line_number_unique")
		})
	}
}

func TestValidateSemanticRelationUniqueRequiresAttributeFromRelationTarget(t *testing.T) {
	w := validSemanticBusinessValidationWorld()
	w.Semantics[1].Validations[1].Attribute = "sales.order.total"

	d := semanticValidationDiagnosticByCode(validateSemanticBusinessValidations(w), "SEMANTIC_VALIDATION_ATTRIBUTE_OUTSIDE_RELATION_TARGET")
	assertSemanticValidationSubject(t, d, "sales.order.lines.line_number_unique")
}

func TestValidateSemanticBusinessValidationRejectsDuplicateIdentity(t *testing.T) {
	w := validSemanticBusinessValidationWorld()
	w.Semantics[1].Validations = append(w.Semantics[1].Validations, w.Semantics[1].Validations[0])

	d := semanticValidationDiagnosticByCode(validateSemanticBusinessValidations(w), "DUPLICATE_SEMANTIC_VALIDATION")
	assertSemanticValidationSubject(t, d, "sales.order_line.quantity_positive")
}

func validSemanticBusinessValidationWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{
			ID: "std",
			Types: []semantic.Type{
				{ID: "std.integer", Symbol: "integer", Kind: semantic.TypePrimitive},
				{ID: "std.decimal", Symbol: "decimal", Kind: semantic.TypePrimitive},
				{ID: "std.money", Symbol: "money", Kind: semantic.TypePrimitive},
				{ID: "std.code", Symbol: "code", Kind: semantic.TypePrimitive},
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
					Relations: []semantic.Relation{
						{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
					},
				},
				{
					ID:     "sales.order_line",
					Symbol: "OrderLine",
					Attributes: []semantic.Attribute{
						{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
						{ID: "sales.order_line.item_code", Symbol: "itemCode", TypeRef: "std.code"},
						{ID: "sales.order_line.quantity", Symbol: "quantity", TypeRef: "std.decimal"},
					},
				},
			},
			Validations: []semantic.Validation{
				{
					ID:        "sales.order_line.quantity_positive",
					Symbol:    "quantity_positive",
					Kind:      semantic.ValidationPositive,
					Attribute: "sales.order_line.quantity",
				},
				{
					ID:        "sales.order.lines.line_number_unique",
					Symbol:    "line_number_unique",
					Kind:      semantic.ValidationRelationUnique,
					Relation:  "sales.order.lines",
					Attribute: "sales.order_line.line_number",
				},
			},
		},
	}}
}

func semanticValidationDiagnosticByCode(diagnostics []Diagnostic, code string) *Diagnostic {
	for i := range diagnostics {
		if diagnostics[i].Code == code {
			return &diagnostics[i]
		}
	}
	return nil
}

func assertSemanticValidationSubject(t *testing.T, diagnostic *Diagnostic, id core.CanonicalID) {
	t.Helper()
	if diagnostic == nil {
		t.Fatalf("期待するSemantic Validation Diagnosticがありません: %s", id)
	}
	if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-validation" || diagnostic.Subject.ID != string(id) {
		t.Fatalf("Diagnostic Subjectが期待値と異なります: %#v", diagnostic.Subject)
	}
}
