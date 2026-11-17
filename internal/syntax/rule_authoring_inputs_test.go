package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// §12 of the rule normal form decision writes a rule that uses an operator with
// named inputs as `<subject> is <operator> <name>: <value> ...`. The subject is
// the first input and is not repeated; the rest are named assignments.
func TestRuleAuthoringReadsNamedOperatorInputs(t *testing.T) {
	src := `namespace sales

operator sales.within: takes low: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 100
`
	parsed, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("the named assignment did not parse: %v", err)
	}
	if len(parsed.File.Rules) != 1 {
		t.Fatalf("rules=%d", len(parsed.File.Rules))
	}
	inputs := parsed.File.Rules[0].Inputs
	if len(inputs) != 1 {
		t.Fatalf("inputs=%+v", inputs)
	}
	if inputs[0].Name != "low" || inputs[0].Value != "100" {
		t.Fatalf("input=%+v, want low: 100", inputs[0])
	}
	if parsed.File.Rules[0].Subject.Text != "sales.order_line.discount" {
		t.Fatalf("subject=%q", parsed.File.Rules[0].Subject.Text)
	}
}

// The unnamed form keeps reading exactly as it did: no assignments at all.
// This pins that accepting the named form did not make every trailing word an
// assignment.
func TestRuleAuthoringOperatorReferenceWithoutInputsHasNoAssignments(t *testing.T) {
	src := "namespace sales\noperator sales.op: takes number\nrule sales.r:\n  sales.a.b is sales.op\n"
	parsed, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.File.Rules[0].Inputs) != 0 {
		t.Fatalf("inputs=%+v, want none", parsed.File.Rules[0].Inputs)
	}
}

// A name without a value is refused: an input with no value would leave the
// rule asking about something the source never gave.
func TestRuleAuthoringRefusesAnInputNameWithoutAValue(t *testing.T) {
	src := "namespace sales\noperator sales.op: takes low: number\nrule sales.r:\n  sales.a.b is sales.op low:\n"
	_, err := syntax.ParseRuleAuthoringFile(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("an input name without a value was accepted or unlocated: %v", err)
	}
	if !strings.Contains(parseErr.Msg, "expected the value of input") {
		t.Fatalf("message=%q", parseErr.Msg)
	}
}
