package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A semantic validation rule is small: an id, a kind, and at most one attribute
// and one relation. Each check it takes part in reports its own code, so the
// codes are produced here one rule at a time (#4032).
func TestSemanticValidationCodesAreProduced(t *testing.T) {
	valid := func() ir.World {
		return ir.World{
			Semantics: []semantic.Namespace{
				{
					ID: "sales",
					Types: []semantic.Type{
						{ID: "sales.amount", Symbol: "Amount", Kind: semantic.TypeAlias, Underlying: "std.decimal"},
					},
					Data: []semantic.Data{
						{
							ID: "sales.order",
							Attributes: []semantic.Attribute{
								{ID: "sales.order.amount", Symbol: "amount", TypeRef: "sales.amount"},
							},
							Relations: []semantic.Relation{
								{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
							},
						},
						{
							ID: "sales.order_line",
							Attributes: []semantic.Attribute{
								{ID: "sales.order_line.code", Symbol: "code", TypeRef: "sales.amount"},
							},
						},
					},
					Validations: []semantic.Validation{
						{ID: "sales.order.amount_positive", Symbol: "amountPositive", Kind: semantic.ValidationPositive, Attribute: "sales.order.amount"},
					},
				},
			},
		}
	}

	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{
			name: "namespace mismatch",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].ID = "orders.order.amount_positive"
			},
			want: "SEMANTIC_VALIDATION_NAMESPACE_MISMATCH",
		},
		{
			name: "invalid kind",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].Kind = semantic.ValidationKind("unknown")
			},
			want: "INVALID_SEMANTIC_VALIDATION_KIND",
		},
		{
			name: "attribute required",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].Attribute = ""
			},
			want: "SEMANTIC_VALIDATION_ATTRIBUTE_REQUIRED",
		},
		{
			name: "positive validation with relation",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].Relation = "sales.order.lines"
			},
			want: "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED",
		},
		{
			name: "relation-unique without relation",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].Kind = semantic.ValidationRelationUnique
			},
			want: "SEMANTIC_VALIDATION_RELATION_REQUIRED",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := valid()
			testCase.mutate(&world)
			found := false
			for _, diagnostic := range Validate(world) {
				if diagnostic.Code == testCase.want {
					found = true
					break
				}
			}
			if !found {
				var codes []string
				for _, diagnostic := range Validate(world) {
					codes = append(codes, diagnostic.Code)
				}
				t.Fatalf("expected %s, got %v", testCase.want, codes)
			}
		})
	}
}

// The baseline world must stay accepted, otherwise every case above would be
// reading diagnostics that some other rule produced.
func TestSemanticValidationBaselineWorldIsAccepted(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Types: []semantic.Type{
					{ID: "sales.amount", Symbol: "Amount", Kind: semantic.TypeAlias, Underlying: "std.decimal"},
				},
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.amount", Symbol: "amount", TypeRef: "sales.amount"},
						},
						Relations: []semantic.Relation{
							{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
						},
					},
					{ID: "sales.order_line", Attributes: []semantic.Attribute{
						{ID: "sales.order_line.code", Symbol: "code", TypeRef: "sales.amount"},
					}},
				},
				Validations: []semantic.Validation{
					{ID: "sales.order.amount_positive", Symbol: "amountPositive", Kind: semantic.ValidationPositive, Attribute: "sales.order.amount"},
				},
			},
			{ID: "std", Types: []semantic.Type{{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive}}},
		},
	}
	if diagnostics := Validate(world); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}
}
