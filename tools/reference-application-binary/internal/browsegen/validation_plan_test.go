package browsegen

import (
	"errors"
	"testing"

	"tei/internal/semantic"
)

// V-T5: a rule kind without an evaluation is refused, not generated as a no-op.
func TestUnsupportedRuleKindIsRefused(t *testing.T) {
	_, err := realisableRules([]semantic.Validation{
		{ID: "sales.a", Kind: semantic.ValidationPositive, Attribute: "sales.x.y"},
		{ID: "sales.b", Kind: semantic.ValidationOperator, Attribute: "sales.x.z", Operator: "std.op"},
	})
	if !errors.Is(err, ErrValidationKindUnsupported) {
		t.Fatalf("want VALIDATION_KIND_UNSUPPORTED, got %v", err)
	}
	rules, err := realisableRules([]semantic.Validation{
		{ID: "sales.b", Kind: semantic.ValidationRelationUnique, Attribute: "a", Relation: "r"},
		{ID: "sales.a", Kind: semantic.ValidationPositive, Attribute: "x"},
	})
	if err != nil || len(rules) != 2 || rules[0].ID != "sales.b" {
		t.Fatalf("declared order must be kept: %v %v", rules, err)
	}
}
