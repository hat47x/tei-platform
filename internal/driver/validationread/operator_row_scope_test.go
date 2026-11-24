package validationread_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The expected answers below are written by hand from decision section 15.2's
// table and from the rule the Workspace wrote: every row's allocation is below
// 100, and the operator may refute on some rows.
//
// The rule reaches this reader the way the product reaches it: the operator
// declares one input read over all rows, so the rule is asked of the relation
// once rather than row by row.

const rowScopedOperatorID core.CanonicalID = "organization.allocation_below"

func rowScopedContract(refutes bool) semantic.Operator {
	return semantic.Operator{
		ID:     rowScopedOperatorID,
		Symbol: "allocation_below",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: refutes,
	}
}

// rowScopedRule is what the conversion produces for
//
//	rule organization.members.allocation_below:
//	  organization.membership.allocation is organization.allocation_below per organization.members
func rowScopedRule() semantic.Validation {
	return semantic.Validation{
		ID:        "organization.members.allocation_below",
		Symbol:    "allocation_below",
		Kind:      semantic.ValidationOperator,
		Attribute: allocation,
		Relation:  membersID,
		Operator:  rowScopedOperatorID,
	}
}

// belowHundred is the plain reading of the contract, written here so a wrong
// evaluator is caught even when the shipped one is wrong too.
func belowHundred() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: rowScopedOperatorID,
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			values, ok := rows["value"]
			if !ok {
				return validationeval.TriMaybe, nil
			}
			for _, one := range values {
				if one.State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if one.Value.Cmp(big.NewRat(100, 1)) >= 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func rowScopedDecide(t *testing.T, detail *validationread.Detail, refutes bool) validationread.Decisions {
	t.Helper()
	decisions, err := detail.Decide(rowScopedRule(), validationeval.Operators{
		Declared:        []semantic.Operator{rowScopedContract(refutes)},
		Implementations: []validationeval.OperatorImplementation{belowHundred()},
	})
	if err != nil {
		t.Fatalf("refutes=%v: %v", refutes, err)
	}
	return decisions
}

// The rule is asked of the relation once: the answer is in Relation, not in
// Rows, whatever the rows hold.
func TestARowScopedOperatorRuleIsAskedOfTheRelationOnce(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{allocation: "10"}),
		row(map[core.CanonicalID]any{allocation: "20"}),
	)
	decisions := rowScopedDecide(t, detail, false)
	if decisions.Rows != nil {
		t.Fatalf("the rule was answered row by row: %+v", decisions.Rows)
	}
	if decisions.Relation == nil {
		t.Fatal("the rule produced no relation answer")
	}
	if decisions.Relation.Result != validationeval.ResultSatisfied {
		t.Fatalf("decision=%+v, want satisfied", *decisions.Relation)
	}
}

// Every row was read and one holds a counterexample: the rule does not hold,
// and the refutation statement is not needed.
func TestARowScopedOperatorRuleSeesACounterexampleInEveryRow(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{allocation: "10"}),
		row(map[core.CanonicalID]any{allocation: "200"}),
	)
	decisions := rowScopedDecide(t, detail, false)
	if decisions.Relation == nil || decisions.Relation.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decisions=%+v, want not-satisfied", decisions)
	}
}

// A partly read relation with a witnessed counterexample is not-satisfied only
// when the operator says a counterexample among the rows read is enough.
func TestARowScopedOperatorRuleOnAPartlyReadRelation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		refutes bool
		rows    []driver.ContractQueryRow
		want    validationeval.Result
		wantWhy validationeval.UndecidedReason
	}{
		{
			name:    "counterexample with a refutation",
			refutes: true,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{allocation: "10"}),
				row(map[core.CanonicalID]any{allocation: "200"}),
			},
			want: validationeval.ResultNotSatisfied,
		},
		{
			name:    "the same counterexample without one",
			refutes: false,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{allocation: "10"}),
				row(map[core.CanonicalID]any{allocation: "200"}),
			},
			wantWhy: validationeval.UndecidedRelationPartiallyRead,
		},
		{
			name:    "no counterexample",
			refutes: true,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{allocation: "10"}),
			},
			wantWhy: validationeval.UndecidedRelationPartiallyRead,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			detail := detailOf(t, validationeval.CoveragePartial, tc.rows...)
			decisions := rowScopedDecide(t, detail, tc.refutes)
			if decisions.Relation == nil {
				t.Fatalf("no relation answer: %+v", decisions)
			}
			decision := *decisions.Relation
			if tc.want != "" {
				if decision.Result != tc.want || decision.Undecided != nil {
					t.Fatalf("decision=%+v, want %q", decision, tc.want)
				}
				return
			}
			if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != tc.wantWhy {
				t.Fatalf("decision=%+v, want undecided %q", decision, tc.wantWhy)
			}
		})
	}
}

// A rule whose operator is not among the declared ones is not silently
// answered row by row: the reader says which declaration is missing.
func TestARowScopedRuleNeedsItsOperatorDeclaration(t *testing.T) {
	rule := rowScopedRule()
	rule.Operator = "organization.not_declared"
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{allocation: "10"}))
	decisions, err := detail.Decide(rule, validationeval.Operators{})
	if err == nil {
		t.Fatalf("a rule with an undeclared operator was answered: %+v", decisions)
	}
	if !strings.Contains(err.Error(), "has no loaded declaration") {
		t.Fatalf("the refusal does not name the missing declaration: %v", err)
	}
	if decisions.Rows != nil {
		t.Fatalf("the rule was answered row by row before the error: %+v", decisions.Rows)
	}
}
