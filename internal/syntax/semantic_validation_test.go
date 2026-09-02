package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

func TestParseSemanticPositiveValidation(t *testing.T) {
	src := `namespace sales
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("positive ValidationのParseに失敗しました: %v", err)
	}
	if len(ns.Validations) != 1 {
		t.Fatalf("validations=%#v", ns.Validations)
	}
	got := ns.Validations[0]
	if got.ID != "sales.order_line.quantity_positive" || got.Kind != semantic.ValidationPositive {
		t.Fatalf("positive Validationが期待値と異なります: %#v", got)
	}
	if got.Attribute != "sales.order_line.quantity" || got.Relation != "" {
		t.Fatalf("positive Validationの参照が期待値と異なります: %#v", got)
	}
}

func TestParseSemanticRelationUniqueValidation(t *testing.T) {
	src := `namespace sales
validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("relation-unique ValidationのParseに失敗しました: %v", err)
	}
	if len(ns.Validations) != 1 {
		t.Fatalf("validations=%#v", ns.Validations)
	}
	got := ns.Validations[0]
	if got.ID != "sales.order.lines.line_number_unique" || got.Kind != semantic.ValidationRelationUnique {
		t.Fatalf("relation-unique Validationが期待値と異なります: %#v", got)
	}
	if got.Relation != "sales.order.lines" || got.Attribute != "sales.order_line.line_number" {
		t.Fatalf("relation-unique Validationの参照が期待値と異なります: %#v", got)
	}
}

func TestParseSemanticValidationRejectsUnsupportedKind(t *testing.T) {
	src := `namespace sales
validation sales.order_line.quantity_rule expression {
    attribute sales.order_line.quantity
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("未検証の汎用Validation kindを受理しました")
	}
}

func TestParseSemanticPositiveValidationRejectsRelation(t *testing.T) {
	src := `namespace sales
validation sales.order_line.quantity_positive positive {
    relation sales.order.lines
    attribute sales.order_line.quantity
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("positive ValidationがRelationを受理しました")
	}
}

func TestParseSemanticRelationUniqueValidationRequiresRelation(t *testing.T) {
	src := `namespace sales
validation sales.order.lines.line_number_unique relation-unique {
    attribute sales.order_line.line_number
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("relation-unique ValidationがRelationなしで成立しました")
	}
}

func TestParseSemanticValidationRequiresAttribute(t *testing.T) {
	src := `namespace sales
validation sales.order_line.quantity_positive positive {
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("Attributeを持たないSemantic Validationを受理しました")
	}
}

func TestParseSemanticValidationRejectsDuplicateFields(t *testing.T) {
	for name, src := range map[string]string{
		"attribute": `namespace sales
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
    attribute sales.order_line.unit_price
}`,
		"relation": `namespace sales
validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    relation sales.order.customer
    attribute sales.order_line.line_number
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
				t.Fatalf("%s fieldの重複を受理しました", name)
			}
		})
	}
}

func TestParseSemanticValidationRejectsForeignNamespaceIdentity(t *testing.T) {
	src := `namespace sales
validation customer.quantity_positive positive {
    attribute sales.order_line.quantity
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("Semantic Namespace外のValidation identityを受理しました")
	}
}
