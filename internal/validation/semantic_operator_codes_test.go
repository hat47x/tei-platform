package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A declared operator is data in the World. Each check it takes part in
// reports its own code, produced here one change at a time from a World the
// validator accepts.
func TestSemanticOperatorCodesAreProduced(t *testing.T) {
	valid := func() ir.World {
		return ir.World{
			Semantics: []semantic.Namespace{
				{
					ID: "sales",
					Data: []semantic.Data{{
						ID: "sales.order_line",
						Attributes: []semantic.Attribute{
							{ID: "sales.order_line.discount", Symbol: "discount", TypeRef: "std.decimal"},
							{ID: "sales.order_line.item_name", Symbol: "itemName", TypeRef: "std.text"},
						},
						Relations: []semantic.Relation{
							{ID: "sales.order_line.parts", Symbol: "parts", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
						},
					}},
					Operators: []semantic.Operator{
						{ID: "sales.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputNumber},
						{ID: "sales.all_positive", Symbol: "all_positive", Input: semantic.OperatorInputNumber,
							Inputs: []semantic.OperatorInputDeclaration{
								{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
							}},
					},
					Validations: []semantic.Validation{
						{ID: "sales.order_line.discount_non_negative", Symbol: "discount_non_negative",
							Kind: semantic.ValidationOperator, Attribute: "sales.order_line.discount", Operator: "sales.non_negative"},
						{ID: "sales.order_line.discount_positive", Symbol: "discount_positive",
							Kind: semantic.ValidationPositive, Attribute: "sales.order_line.discount"},
						{ID: "sales.order_line.parts_all_positive", Symbol: "parts_all_positive",
							Kind: semantic.ValidationOperator, Attribute: "sales.order_line.discount",
							Operator: "sales.all_positive", Relation: "sales.order_line.parts"},
					},
				},
				{ID: "std", Types: []semantic.Type{
					{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive},
					{ID: "std.text", Symbol: "Text", Kind: semantic.TypePrimitive},
				}},
			},
		}
	}
	if diagnostics := Validate(valid()); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}

	cases := []struct {
		name    string
		mutate  func(w *ir.World)
		want    string
		subject string
	}{
		{"rule names an operator nobody declared", func(w *ir.World) {
			w.Semantics[0].Validations[0].Operator = "sales.non_negativ"
		}, "UNKNOWN_SEMANTIC_VALIDATION_OPERATOR", "sales.order_line.discount_non_negative"},
		{"rule names no operator", func(w *ir.World) {
			w.Semantics[0].Validations[0].Operator = ""
		}, "UNKNOWN_SEMANTIC_VALIDATION_OPERATOR", "sales.order_line.discount_non_negative"},
		{"declaration removed while a rule still uses it", func(w *ir.World) {
			w.Semantics[0].Operators = nil
		}, "UNKNOWN_SEMANTIC_VALIDATION_OPERATOR", "sales.order_line.discount_non_negative"},
		{"number operator on a text attribute", func(w *ir.World) {
			w.Semantics[0].Validations[0].Attribute = "sales.order_line.item_name"
		}, "SEMANTIC_VALIDATION_OPERATOR_REQUIRES_NUMERIC_ATTRIBUTE", "sales.order_line.discount_non_negative"},
		{"operator rule with a relation", func(w *ir.World) {
			w.Semantics[0].Validations[0].Relation = "sales.order_line.parts"
		}, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "sales.order_line.discount_non_negative"},
		// The same relation, on a rule whose operator reads every row, is
		// required rather than refused: the declaration decides how the rule
		// is asked, not the kind (decision section 15.4).
		{"row-scoped operator rule without a relation", func(w *ir.World) {
			w.Semantics[0].Validations[2].Relation = ""
		}, "SEMANTIC_VALIDATION_OPERATOR_RELATION_REQUIRED", "sales.order_line.parts_all_positive"},
		// A relation the rule names must be one a Data declares. Before this
		// the read refused it and validate did not, so a mistyped relation was
		// found only when a screen happened to be opened.
		{"row-scoped operator rule naming a relation no Data declares", func(w *ir.World) {
			w.Semantics[0].Validations[2].Relation = "sales.order_line.parts_typo"
		}, "SEMANTIC_VALIDATION_OPERATOR_RELATION_UNKNOWN", "sales.order_line.parts_all_positive"},
		// The rule is read over the rows of the relation it names, so the
		// subject attribute has to be a column of those rows. A relation to
		// another Data gives that Data's rows, and the read refuses it; validate
		// must refuse it first, as relation-unique already does.
		{"row-scoped operator rule reading an attribute of another Data", func(w *ir.World) {
			w.Semantics[0].Data = append(w.Semantics[0].Data, semantic.Data{
				ID: "sales.customer",
				Attributes: []semantic.Attribute{
					{ID: "sales.customer.customer_id", Symbol: "customerId", TypeRef: "std.integer"},
				},
			})
			w.Semantics[0].Data[0].Relations = append(w.Semantics[0].Data[0].Relations, semantic.Relation{
				ID: "sales.order_line.customer", Symbol: "customer",
				Target: "sales.customer", Cardinality: semantic.CardinalityOne,
			})
			w.Semantics[0].Validations[2].Relation = "sales.order_line.customer"
		}, "SEMANTIC_VALIDATION_OPERATOR_ATTRIBUTE_OUTSIDE_RELATION_TARGET", "sales.order_line.parts_all_positive"},
		{"built-in kind carrying an operator", func(w *ir.World) {
			w.Semantics[0].Validations[1].Operator = "sales.non_negative"
		}, "SEMANTIC_VALIDATION_OPERATOR_NOT_ALLOWED", "sales.order_line.discount_positive"},
		{"operator declared twice", func(w *ir.World) {
			w.Semantics[0].Operators = append(w.Semantics[0].Operators, w.Semantics[0].Operators[0])
		}, "DUPLICATE_SEMANTIC_OPERATOR", "sales.non_negative"},
		{"operator outside its namespace", func(w *ir.World) {
			w.Semantics[0].Operators[0].ID = "purchase.non_negative"
			w.Semantics[0].Validations[0].Operator = "purchase.non_negative"
		}, "SEMANTIC_OPERATOR_NAMESPACE_MISMATCH", "purchase.non_negative"},
		{"operator with an input nobody defines", func(w *ir.World) {
			w.Semantics[0].Operators[0].Input = "keys"
		}, "INVALID_SEMANTIC_OPERATOR_INPUT", "sales.non_negative"},
		{"operator sharing an identity with an attribute", func(w *ir.World) {
			w.Semantics[0].Operators[0].ID = "sales.order_line.discount"
			w.Semantics[0].Validations[0].Operator = "sales.order_line.discount"
		}, "DUPLICATE_SEMANTIC_ELEMENT", "sales.order_line.discount"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := valid()
			tc.mutate(&world)
			var found bool
			var codes []string
			for _, diagnostic := range Validate(world) {
				codes = append(codes, diagnostic.Code)
				if diagnostic.Code == tc.want && diagnostic.Subject != nil && diagnostic.Subject.ID == tc.subject {
					found = true
				}
			}
			if !found {
				t.Fatalf("want %s on %s, got %v", tc.want, tc.subject, codes)
			}
		})
	}
}
