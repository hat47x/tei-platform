package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

const ruleAuthoringWithOperator = `namespace sales

operator sales.non_negative: takes number

rule sales.order_line.discount_non_negative:
  sales.order_line.discount is sales.non_negative

rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
`

func TestRuleAuthoringReadsAnOperatorDeclarationAndItsUse(t *testing.T) {
	src := ruleAuthoringWithOperator
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("conversion issues: %v", anchored.Issues)
	}

	// `takes number` is the unnamed single-input form, so the parsed contract
	// carries no name. The surface keeps reading it as the input named "value",
	// but nothing is declared, so nothing is stored.
	wantOperators := []semantic.Operator{{ID: "sales.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "", Type: semantic.OperatorInputNumber}}}}
	if len(anchored.Value.Operators) != 1 || !reflect.DeepEqual(anchored.Value.Operators[0], wantOperators[0]) {
		t.Fatalf("operators=%+v, want %+v", anchored.Value.Operators, wantOperators)
	}
	if len(anchored.Value.Validations) != 2 {
		t.Fatalf("validations=%+v", anchored.Value.Validations)
	}
	uses := anchored.Value.Validations[0]
	if uses.Kind != semantic.ValidationOperator || uses.Operator != "sales.non_negative" ||
		uses.Attribute != "sales.order_line.discount" || uses.Relation != "" {
		t.Fatalf("operator rule=%+v", uses)
	}
	// The built-in predicate next to it keeps its existing meaning.
	builtin := anchored.Value.Validations[1]
	if builtin.Kind != semantic.ValidationPositive || builtin.Operator != "" {
		t.Fatalf("built-in rule=%+v", builtin)
	}

	// Spans are located independently in the source.
	declaration := anchored.File.Operators[0]
	rule := anchored.File.Rules[0]
	checks := []struct {
		what string
		span syntax.Anchor
		at   int
		text string
	}{
		{"operator id", declaration.IDSpan, uniqueOffset(t, src, "sales.non_negative:"), "sales.non_negative"},
		{"operator input", declaration.InputSpan, uniqueOffset(t, src, "number"), "number"},
		{"operator reference", rule.Operator.Anchor, uniqueOffset(t, src, "is sales.non_negative") + len("is "), "sales.non_negative"},
	}
	for _, check := range checks {
		if check.span.Start.Offset != check.at || src[check.span.Start.Offset:check.span.End.Offset] != check.text {
			t.Errorf("%s spans [%d,%d), want %q at %d", check.what, check.span.Start.Offset, check.span.End.Offset, check.text, check.at)
		}
	}
	var operatorAnchors int
	for _, anchor := range anchored.Anchors {
		if anchor.Kind == "semantic-operator" && anchor.ID == "sales.non_negative" && anchor.Anchor == declaration.IDSpan {
			operatorAnchors++
		}
	}
	if operatorAnchors != 1 {
		t.Fatalf("operator declaration has %d source anchors, want 1", operatorAnchors)
	}
}

func TestRuleAuthoringAcceptsASourceThatOnlyDeclaresOperators(t *testing.T) {
	file, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.non_negative: takes number\n")
	if err != nil || len(file.Operators) != 1 || len(file.Rules) != 0 {
		t.Fatalf("file=%+v err=%v", file, err)
	}
}

func TestRuleAuthoringOperatorRefusals(t *testing.T) {
	cases := []struct {
		name, src, at, message string
	}{
		{"input word nobody defines",
			"namespace sales\noperator sales.op: takes keys\n", "keys", "unknown operator input"},
		{"missing takes",
			"namespace sales\noperator sales.op: number\n", "number", `unexpected "number" after the operator name`},
		{"missing input",
			"namespace sales\noperator sales.op: takes\n", "takes", "expected what the operator takes"},
		{"second input",
			"namespace sales\noperator sales.op: takes number number2\n", "number2", `unexpected "number2" after the operator name`},
		{"operator declared twice",
			"namespace sales\noperator sales.op: takes number\noperator sales.op: takes number\n", "sales.op: takes number\n\x00", "declared more than once"},
		{"operator outside the namespace",
			"namespace sales\noperator purchase.op: takes number\n", "purchase.op", "must belong to the declared namespace"},
		{"declared operator with a scope",
			"namespace sales\nrule sales.r: sales.a.b is sales.op per sales.a.lines\n", "sales.a.lines", "accepts no scope"},
		{"clauses in the other order",
			"namespace sales\nrule sales.r: positive is sales.a.b\n", "positive", "is a predicate, not an attribute"},
		{"unknown bare predicate is not an operator",
			"namespace sales\nrule sales.r: sales.a.b is non_negative\n", "non_negative", "unknown rule predicate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile(tc.src)
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
			want := 0
			if strings.HasSuffix(tc.at, "\x00") {
				// The second of two identical declarations.
				want = strings.LastIndex(tc.src, strings.TrimSuffix(tc.at, "\x00"))
			} else {
				want = uniqueOffset(t, tc.src, tc.at)
			}
			if parseErr.Pos.Offset != want {
				t.Fatalf("error offset=%d, want %d", parseErr.Pos.Offset, want)
			}
		})
	}
}

// Formatting has no layout for operators yet. It must refuse the source
// instead of returning one that lost the declaration or the reference.
func TestFormatRuleAuthoringRefusesOperatorsInsteadOfDroppingThem(t *testing.T) {
	for _, src := range []string{
		ruleAuthoringWithOperator,
		"namespace sales\nrule sales.r: sales.a.b is sales.op\n",
		// A declaration next to a built-in rule: nothing else would stop the
		// formatter from rendering the rule and losing the declaration.
		"namespace sales\noperator sales.op: takes number\nrule sales.r: sales.a.b is positive\n",
	} {
		if out, err := syntax.FormatRuleAuthoring(src); !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
			t.Fatalf("formatted %q, err=%v", out, err)
		}
	}
}
