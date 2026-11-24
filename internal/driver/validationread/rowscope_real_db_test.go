package validationread_test

import (
	"math/big"
	"testing"

	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator declares `of each row` is asked of the rows of the
// relation through Detail.Decide, over rows a real database produced. The
// answers below are decided from the seed rows, not from the run.
//
//	D1: allocation '1', '0.5', '0', null
//	D2: allocation '1', '1'
//
// With `refutes on some rows` a witnessed counterexample settles the rule
// whatever the coverage; without it, a seen counterexample in a partly read
// relation leaves the rule undecided (decision section 15.2).

// allocationAbove reads the allocation column of every row and answers
// TriYes when every row is at most one.
func allocationAbove(limit *big.Rat) validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "organization.above_limit",
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			for _, row := range rows["value"] {
				if row.State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if row.Value.Cmp(limit) > 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

// rowsRule states the rule over the rows of the membership relation.
func rowsRule(refutes bool) (semantic.Validation, semantic.Operator) {
	contract := semantic.Operator{
		ID:    "organization.above_limit",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: refutes,
	}
	rule := semantic.Validation{
		ID:        "organization.membership.allocation_below",
		Symbol:    "allocation_below",
		Kind:      semantic.ValidationOperator,
		Attribute: allocation,
		Relation:  membersID,
		Operator:  "organization.above_limit",
	}
	return rule, contract
}

// relationAnswer asks one row-scoped rule through the product dispatch.
func relationAnswer(t *testing.T, detail *validationread.Detail, refutes bool, limit *big.Rat) string {
	t.Helper()
	rule, contract := rowsRule(refutes)
	decisions, err := detail.Decide(rule, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{allocationAbove(limit)},
	})
	if err != nil {
		t.Fatalf("%s: %v", rule.ID, err)
	}
	// A row-scoped rule has one answer for the relation, never one per row.
	if decisions.Relation == nil || decisions.Rows != nil {
		t.Fatalf("%s: the answer is not the relation's: %+v", rule.ID, decisions)
	}
	if decisions.Relation.Undecided != nil {
		return "undecided: " + string(decisions.Relation.Undecided.Reason)
	}
	return string(decisions.Relation.Result)
}

func TestARowScopedRuleIsDecidedFromRealRows(t *testing.T) {
	executor := membershipExecutor(t)

	// D1 holds '1', '0.5', '0' and null. With the limit 1 every read row
	// satisfies it, but the null means the operator cannot tell, and reading
	// more rows cannot settle a value that is absent.
	d1 := readMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
	if got := relationAnswer(t, d1, true, big.NewRat(1, 1)); got != "undecided: value-null" {
		t.Fatalf("D1 with limit 1: got %q, want the null to keep its own reason", got)
	}

	// Limit 0: the rows '1' and '0.5' are above it, so a counterexample was
	// read. With the refutation statement that is decisive even though one
	// row holds no value and the relation is complete.
	if got := relationAnswer(t, d1, true, big.NewRat(0, 1)); got != "not-satisfied" {
		t.Fatalf("D1 with limit 0: got %q, want not-satisfied", got)
	}

	// The same rows without the statement are not settled by a seen
	// counterexample when the relation was partly read.
	page := readMembers(t, executor, "members_first_page", "D1", validationeval.CoveragePartial)
	if got := relationAnswer(t, page, false, big.NewRat(0, 1)); got != "undecided: relation-partially-read" {
		t.Fatalf("first page without the statement: got %q, want undecided relation-partially-read", got)
	}
	// With the statement the same page's counterexample is decisive.
	if got := relationAnswer(t, page, true, big.NewRat(0, 1)); got != "not-satisfied" {
		t.Fatalf("first page with the statement: got %q, want not-satisfied", got)
	}

	// D2 holds '1', '1': every row satisfies the limit 1 and every row was
	// read, so the rule holds.
	d2 := readMembers(t, executor, "members", "D2", validationeval.CoverageComplete)
	if got := relationAnswer(t, d2, false, big.NewRat(1, 1)); got != "satisfied" {
		t.Fatalf("D2 with limit 1: got %q, want satisfied", got)
	}

	// The narrow query never returned the allocation column, so every row is
	// not-fetched: the rule is undecided with that reason rather than read as
	// zero.
	narrow := readMembers(t, executor, "members_narrow", "D1", validationeval.CoverageComplete)
	if got := relationAnswer(t, narrow, true, big.NewRat(0, 1)); got != "undecided: value-not-fetched" {
		t.Fatalf("narrow query: got %q, want undecided value-not-fetched", got)
	}
}
