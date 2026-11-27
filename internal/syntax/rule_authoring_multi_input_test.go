package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// TestRuleAuthoringReadsMultiInputOperator verifies that an operator with
// multiple named inputs is parsed and converted correctly.
func TestRuleAuthoringReadsMultiInputOperator(t *testing.T) {
	src := `namespace sales

operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 0 high: 100
`
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("issues: %v", anchored.Issues)
	}

	// The operator carries all three inputs in signature order.
	if len(anchored.Value.Operators) != 1 {
		t.Fatalf("operators=%+v", anchored.Value.Operators)
	}
	op := anchored.Value.Operators[0]
	if op.ID != "sales.within" || op.Symbol != "within" || op.Input != semantic.OperatorInputNumber {
		t.Fatalf("operator=%+v", op)
	}
	if len(op.Inputs) != 3 {
		t.Fatalf("operator inputs=%+v", op.Inputs)
	}
	wantInputs := []semantic.OperatorInputDeclaration{
		{Name: "value", Type: semantic.OperatorInputNumber},
		{Name: "low", Type: semantic.OperatorInputNumber},
		{Name: "high", Type: semantic.OperatorInputNumber},
	}
	for i, want := range wantInputs {
		if op.Inputs[i] != want {
			t.Fatalf("input[%d]=%+v, want %+v", i, op.Inputs[i], want)
		}
	}

	// The rule carries the named assignments in the operator's signature order.
	if len(anchored.Value.Validations) != 1 {
		t.Fatalf("validations=%+v", anchored.Value.Validations)
	}
	v := anchored.Value.Validations[0]
	if v.Kind != semantic.ValidationOperator || v.Operator != "sales.within" ||
		v.Attribute != "sales.order_line.discount" {
		t.Fatalf("validation=%+v", v)
	}
	wantAssignments := []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "100"},
	}
	if !reflect.DeepEqual(v.Inputs, wantAssignments) {
		t.Fatalf("inputs=%+v, want %+v", v.Inputs, wantAssignments)
	}
}

// TestRuleAuthoringRefusesDuplicateInputName verifies that assigning the same
// input name twice in one rule is refused.
func TestRuleAuthoringRefusesDuplicateInputName(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 0 low: 100
`
	_, err := syntax.ParseRuleAuthoringFile(src)
	if err == nil {
		t.Fatal("duplicate input name was accepted")
	}
	var parseErr *syntax.ParseError
	if !contains(err.Error(), "assigned more than once") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(err.Error(), "low") {
		t.Fatalf("error should name the duplicate input: %v", err)
	}
	_ = parseErr
}

// TestRuleAuthoringRefusesMissingRequiredInput verifies that a rule that
// omits a required input is refused.
func TestRuleAuthoringRefusesMissingRequiredInput(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 0
`
	_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want one issue about missing input", issues)
	}
	if !contains(issues[0].Msg, "declares input") && !contains(issues[0].Msg, "assign") {
		t.Fatalf("issue should mention missing input assignment: %v", issues[0].Msg)
	}
	if !contains(issues[0].Msg, "high") {
		t.Fatalf("issue should name the missing input: %v", issues[0].Msg)
	}
}

// TestRuleAuthoringRefusesExtraInputName verifies that a rule that names an
// input the operator does not declare is refused.
func TestRuleAuthoringRefusesExtraInputName(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 0 high: 100 extra: 5
`
	_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want one issue about unknown input", issues)
	}
	if !contains(issues[0].Msg, "declares no input") {
		t.Fatalf("issue should mention undeclared input: %v", issues[0].Msg)
	}
}

// TestRuleAuthoringRefusesWrongTypeForInput verifies that a value whose type
// does not match the declared input type is refused.
func TestRuleAuthoringRefusesWrongTypeForInput(t *testing.T) {
	src := `namespace sales
operator sales.non_negative:
  takes value: number

rule sales.order_line.discount_non_negative:
  sales.order_line.discount is sales.non_negative value: true
`
	_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want one type-mismatch issue", issues)
	}
	// The refusal names the type the declaration states, which is what the
	// author of a wrong-typed value needs: it used to say "takes a number"
	// whatever the declaration said.
	if !contains(issues[0].Msg, "is read as number") {
		t.Fatalf("issue should name the declared type: %v", issues[0].Msg)
	}
}

// TestRuleAuthoringMultiInputOperatorWithDifferentTypes verifies that an
// operator with mixed input types is parsed correctly.
func TestRuleAuthoringMultiInputOperatorWithDifferentTypes(t *testing.T) {
	src := `namespace sales

operator sales.required_if:
  takes target: presence
  takes trigger: presence

rule sales.order_line.reason_required:
  sales.order_line.discount_reason is sales.required_if trigger: sales.order_line.discount
`
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("issues: %v", anchored.Issues)
	}
	if len(anchored.Value.Operators) != 1 {
		t.Fatalf("operators=%+v", anchored.Value.Operators)
	}
	op := anchored.Value.Operators[0]
	if len(op.Inputs) != 2 {
		t.Fatalf("operator inputs=%+v", op.Inputs)
	}
	if op.Inputs[0].Name != "target" || op.Inputs[0].Type != semantic.OperatorInputPresence {
		t.Fatalf("input[0]=%+v", op.Inputs[0])
	}
	if op.Inputs[1].Name != "trigger" || op.Inputs[1].Type != semantic.OperatorInputPresence {
		t.Fatalf("input[1]=%+v", op.Inputs[1])
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Two unnamed inputs both resolve to "value", so the second can never be
// assigned. Before this refusal the declaration validated with an input nobody
// could name, which is worse than refusing it.
func TestRuleAuthoringRefusesTwoUnnamedInputs(t *testing.T) {
	src := `namespace sales
operator sales.two_unnamed:
  takes number
  takes number
`
	_, err := syntax.ParseRuleAuthoringFile(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("two unnamed inputs were accepted or unlocated: %v", err)
	}
	if !strings.Contains(parseErr.Msg, "must name them all") {
		t.Fatalf("message=%q, want it to require names", parseErr.Msg)
	}
	if !strings.Contains(parseErr.Msg, "input 1") {
		t.Fatalf("message=%q, want it to name the first input that has no name", parseErr.Msg)
	}
}

// A named first input and an unnamed second is the same problem: the second is
// unreachable, so the whole declaration is refused rather than half-read.
func TestRuleAuthoringRefusesAnUnnamedSecondInput(t *testing.T) {
	src := `namespace sales
operator sales.mixed:
  takes low: number
  takes number
`
	_, err := syntax.ParseRuleAuthoringFile(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("an unnamed second input was accepted or unlocated: %v", err)
	}
	if !strings.Contains(parseErr.Msg, "must name them all") {
		t.Fatalf("message=%q, want it to require names", parseErr.Msg)
	}
}

// The declaration must cover the second input's modifiers, or a source edit on
// that line falls outside the declaration it belongs to.
func TestTwoInputDeclarationSpanCoversTheSecondInput(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number
  takes low: number absent as missing
`
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("two-input declaration was refused: %v", err)
	}
	end := file.Operators[0].DeclarationSpan.End.Offset
	want := strings.Index(src, "missing") + len("missing")
	if end != want {
		t.Fatalf("declaration end=%d, want %d (the second input's mapping name)", end, want)
	}
}

// The diagnostic must point at the input that has no name, so the author can
// remove exactly that line. A message without a position would send them
// looking through the whole declaration.
func TestUnnamedInputDiagnosticPointsAtThatInput(t *testing.T) {
	src := `namespace sales
operator sales.mixed:
  takes low: number
  takes number
`
	_, err := syntax.ParseRuleAuthoringFile(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("an unnamed second input was accepted or unlocated: %v", err)
	}
	// The second `takes number` is the unnamed one; its type word is where the
	// diagnostic must land.
	want := strings.LastIndex(src, "number")
	if parseErr.Pos.Offset != want {
		t.Fatalf("diagnostic offset=%d, want %d (the unnamed input's type word)", parseErr.Pos.Offset, want)
	}
}

// The second input's own modifier must extend the span: `of each row` on the
// last input is the last thing the declaration says.
func TestTwoInputDeclarationSpanCoversTheSecondInputRangeClause(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number
  takes low: number of each row
`
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("two-input declaration was refused: %v", err)
	}
	end := file.Operators[0].DeclarationSpan.End.Offset
	want := strings.LastIndex(src, "row") + len("row")
	if end != want {
		t.Fatalf("declaration end=%d, want %d (the second input's range clause)", end, want)
	}
}

// A modifier on the FIRST input must not stretch the span past the last input:
// the span ends where the last authored line ends, not at the widest clause.
func TestTwoInputDeclarationSpanEndsAtTheLastInput(t *testing.T) {
	src := `namespace sales
operator sales.within:
  takes value: number absent as missing
  takes low: number
`
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("two-input declaration was refused: %v", err)
	}
	end := file.Operators[0].DeclarationSpan.End.Offset
	want := strings.LastIndex(src, "number") + len("number")
	if end != want {
		t.Fatalf("declaration end=%d, want %d (the last input's own end)", end, want)
	}
}
