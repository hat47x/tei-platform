package workspace

import (
	"reflect"
	"testing"

	"tei/internal/semantic"
)

// A value written in the type its input declares has to reach the product path,
// not only the conversion: the operator and the rule sit in one `.tei.md`, the
// workspace loads, and the constant arrives at the meaning with the spelling the
// author wrote. The surface used to read every assigned value as a number, so a
// text input took no constant at all.
func TestTEIMDTypedInputConstantReachesTheProductPath(t *testing.T) {
	root := teiMDOperatorWorkspace(t)
	// A subject the rule can name: the fixture's text attribute is written in
	// camelCase, which is not a canonical id a rule may reference.
	rewriteTEIMDTestFile(t, root, "semantic/sales/order.tei",
		"    attribute itemName type std.text\n",
		"    attribute itemName type std.text\n    attribute itemCode type std.text\n")
	// The declaration and the rule share one document: a rule that assigns an
	// input is checked against the declarations this source carries, and an
	// operator declared elsewhere is a separate question this test does not
	// ask.
	rewriteTEIMDTestFile(t, root, teiMDOperatorRules,
		"rule sales.order_line.discount_non_negative:\n  sales.order_line.discount is sales.non_negative",
		"operator sales.item_code_prefixed:\n  takes value: text\n  takes prefix: text\n\n"+
			"rule sales.order_line.item_code_prefixed:\n  sales.order_line.item_code is sales.item_code_prefixed prefix: \"A-\"")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("a text constant stopped the product Load: %v", err)
	}
	if codes := diagnosticCodes(loaded); len(codes) != 0 {
		t.Fatalf("the validator rejects a text input with a constant: %v", codes)
	}
	rule := loadedValidation(t, loaded, "sales.order_line.item_code_prefixed")
	want := []semantic.ValidationInputAssignment{{Name: "prefix", Value: `"A-"`}}
	if !reflect.DeepEqual(rule.Inputs, want) {
		t.Fatalf("inputs=%+v, want %+v", rule.Inputs, want)
	}
	// The evaluator reads the carried text in the declared type, so what
	// reached the meaning is what the author wrote.
	if _, err := semantic.ParseOperatorInputValue(semantic.OperatorInputText, rule.Inputs[0].Value); err != nil {
		t.Fatalf("the carried constant is not readable as a text: %v", err)
	}
}
