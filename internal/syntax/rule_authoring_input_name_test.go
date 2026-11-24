package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The input name in `takes <name>: number` is how a rule assigns that input.
// Measured before this test: `takes Low: number` and `takes low.high: number`
// were both accepted, although neither is a name the rule surface can write,
// so the declaration looked usable but could never be referred to. The name is
// now refused the same way `absent as <name>` already refuses an unusable
// mapping name.
func TestOperatorInputNameMustBeUsableByARule(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{
			name: "a name with a separator",
			src:  "namespace sales\noperator sales.op: takes low.high: number\n",
		},
		{
			name: "a name with an uppercase letter",
			src:  "namespace sales\noperator sales.op: takes Low: number\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringAnchored(tc.src)
			if err == nil {
				t.Fatalf("input name was accepted:\n%s", tc.src)
			}
			if !strings.Contains(err.Error(), "is not a usable input name") {
				t.Fatalf("the refusal does not name the input name: %v", err)
			}
		})
	}
}

// A usable name is still accepted, and the named form keeps the meaning it
// already had: the contract carries the name and the input type.
func TestOperatorInputNameThatARuleCanWriteIsAccepted(t *testing.T) {
	anchored, err := syntax.ParseRuleAuthoringAnchored(
		"namespace sales\noperator sales.op: takes low: number\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Value.Operators) != 1 {
		t.Fatalf("operators=%+v", anchored.Value.Operators)
	}
	if got := anchored.Value.Operators[0].InputName; got != "low" {
		t.Fatalf("inputName=%q, want %q", got, "low")
	}
}
