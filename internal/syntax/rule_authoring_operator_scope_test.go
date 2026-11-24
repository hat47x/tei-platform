package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// A rule whose operator reads every row of a relation has to name that
// relation, and before this it could not: the parser refused a scope on any
// operator predicate while the conversion required one for a row-scoped
// operator. The rule could enter the meaning by neither side.
//
// The expectations below are written from decision section 15.4: the scope of
// an operator rule is written `per <relation>`, it is required exactly when the
// operator declares an input read over all rows, and it is refused otherwise.
const rowScopedOperatorSource = `namespace sales

operator sales.below_limit:
  takes value: number of each row
  refutes on some rows
`

func TestARowScopedOperatorRuleMayNameItsRelation(t *testing.T) {
	src := rowScopedOperatorSource + `
rule sales.order.lines.amount_below_limit:
  sales.order_line.amount is sales.below_limit per sales.order.lines
`
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("a row-scoped rule was refused: %+v", anchored.Issues)
	}
	if len(anchored.Value.Validations) != 1 {
		t.Fatalf("validations=%+v", anchored.Value.Validations)
	}
	rule := anchored.Value.Validations[0]
	if rule.Kind != semantic.ValidationOperator || rule.Relation != "sales.order.lines" {
		t.Fatalf("rule=%+v, want an operator rule over sales.order.lines", rule)
	}
	if len(anchored.Value.Operators) != 1 {
		t.Fatalf("operators=%+v", anchored.Value.Operators)
	}
	operator := anchored.Value.Operators[0]
	if !operator.RefutesOnSomeRows {
		t.Fatalf("the refutation statement was dropped: %+v", operator)
	}
	if len(operator.Inputs) != 1 || !operator.Inputs[0].OfEachRow {
		t.Fatalf("the row-scoped input was dropped: %+v", operator.Inputs)
	}
}

// Without the relation, the rule cannot say which rows it is asked of.
func TestARowScopedOperatorRuleMustNameItsRelation(t *testing.T) {
	src := rowScopedOperatorSource + `
rule sales.order.lines.amount_below_limit:
  sales.order_line.amount is sales.below_limit
`
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		// The parser may refuse it, or the conversion may; either is a refusal
		// that names the missing scope.
		if !strings.Contains(err.Error(), "per <relation>") && !strings.Contains(err.Error(), "requires a scope") {
			t.Fatalf("the refusal does not name the missing relation: %v", err)
		}
		return
	}
	if len(anchored.Issues) != 1 {
		t.Fatalf("issues=%+v, want one for the missing scope", anchored.Issues)
	}
	if !strings.Contains(anchored.Issues[0].Msg, "per <relation>") {
		t.Fatalf("message=%q, want it to ask for \"per <relation>\"", anchored.Issues[0].Msg)
	}
}

// A scope on an operator rule whose operator reads one record stays refused:
// accepting it would let a rule choose how it is asked.
func TestAScopeOnARecordScopedOperatorRuleIsStillRefused(t *testing.T) {
	const src = `namespace sales

operator sales.non_negative:
  takes value: number

rule sales.order_line.quantity_non_negative:
  sales.order_line.quantity is sales.non_negative per sales.order.lines
`
	_, err := syntax.ParseRuleAuthoringAnchored(src)
	if err == nil {
		t.Fatal("a scope on a record-scoped operator rule was accepted")
	}
	if !strings.Contains(err.Error(), "accepts no scope") {
		t.Fatalf("the refusal does not name the scope: %v", err)
	}
}

// An operator this source does not declare keeps the old refusal: an
// unresolved reference must not decide how a rule is asked.
func TestAScopeOnAnUndeclaredOperatorRuleIsRefused(t *testing.T) {
	const src = `namespace sales

rule sales.order_line.quantity_bounded:
  sales.order_line.quantity is sales.elsewhere.bounded per sales.order.lines
`
	_, err := syntax.ParseRuleAuthoringAnchored(src)
	if err == nil {
		t.Fatal("a scope on an operator this source does not declare was accepted")
	}
	if !strings.Contains(err.Error(), "accepts no scope") {
		t.Fatalf("the refusal does not name the scope: %v", err)
	}
}
