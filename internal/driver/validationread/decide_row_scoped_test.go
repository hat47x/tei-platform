package validationread_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator reads every row of a relation is asked once, of the
// Detail, and has one answer. Which of the two it is comes from the
// declaration and not from the rows: the same operator without `of each row`
// is asked once per row.
//
// The expected answers are written by hand from decision section 15.2's table
// and from the shipped implementation, which answers that every value of the
// column is non-negative:
//
//	allocation: 1, 1/2, -1
//
// The third row is the counterexample. With every row read it refutes the rule
// whatever the declaration says; with only some rows read it refutes it only
// when the operator states `refutes on some rows`.
const rowScopedOperatorID core.CanonicalID = "organization.share_non_negative"

func rowScopedRule(refutesOnSomeRows bool) semantic.Validation {
	return semantic.Validation{
		ID:        "organization.membership.share_non_negative",
		Symbol:    "share_non_negative",
		Kind:      semantic.ValidationOperator,
		Attribute: allocation,
		Relation:  membersID,
		Operator:  rowScopedOperatorID,
	}
}

func rowScopedOperators(t *testing.T, refutesOnSomeRows bool) validationeval.Operators {
	t.Helper()
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{
			Operator:       rowScopedOperatorID,
			Implementation: "tei.number.non-negative",
		}},
		validationeval.BuiltinImplementations(), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return validationeval.Operators{
		Declared: []semantic.Operator{{
			ID: rowScopedOperatorID, Symbol: "share_non_negative",
			Input: semantic.OperatorInputNumber,
			Inputs: []semantic.OperatorInputDeclaration{
				{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			},
			RefutesOnSomeRows: refutesOnSomeRows,
		}},
		Implementations: implementations,
	}
}

func counterexampleRows() []driver.ContractQueryRow {
	return []driver.ContractQueryRow{
		row(map[core.CanonicalID]any{allocation: "1"}),
		row(map[core.CanonicalID]any{allocation: "0.5"}),
		row(map[core.CanonicalID]any{allocation: "-1"}),
	}
}
func TestDecideAsksARowScopedOperatorOnceOfTheDetail(t *testing.T) {
	for _, tc := range []struct {
		name     string
		coverage validationeval.Coverage
		refutes  bool
		rows     []driver.ContractQueryRow
		want     string
		wantWhy  validationeval.UndecidedReason
	}{
		// Every row was read, so the witnessed counterexample decides it and
		// the refutation statement does not matter.
		{"counterexample, complete", validationeval.CoverageComplete, false, counterexampleRows(), "not-satisfied", ""},
		{"counterexample, complete, refutes", validationeval.CoverageComplete, true, counterexampleRows(), "not-satisfied", ""},
		// The same counterexample in a partly read relation: only the
		// declaration that a witnessed counterexample is enough decides it.
		{"counterexample, partial, refutes", validationeval.CoveragePartial, true, counterexampleRows(), "not-satisfied", ""},
		{"counterexample, partial, no refutation", validationeval.CoveragePartial, false, counterexampleRows(), "", validationeval.UndecidedRelationPartiallyRead},
		// No counterexample in a partly read relation: an unread row may still
		// hold one, so the rule is not answered satisfied.
		{"no counterexample, partial", validationeval.CoveragePartial, true, counterexampleRows()[:2], "", validationeval.UndecidedRelationPartiallyRead},
		{"no counterexample, complete", validationeval.CoverageComplete, false, counterexampleRows()[:2], "satisfied", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			detail := detailOf(t, tc.coverage, tc.rows...)
			decisions, err := detail.Decide(rowScopedRule(tc.refutes), rowScopedOperators(t, tc.refutes))
			if err != nil {
				t.Fatalf("the row-scoped rule was refused: %v", err)
			}
			// One answer for the relation, and none per row: the rule is about
			// all the rows together.
			if len(decisions.Rows) != 0 {
				t.Fatalf("the rule was answered row by row: %+v", decisions.Rows)
			}
			if decisions.Relation == nil {
				t.Fatal("the rule has no answer for the relation")
			}
			if tc.want != "" {
				if decisions.Relation.Result != validationeval.Result(tc.want) {
					t.Fatalf("result=%q, want %q (%+v)", decisions.Relation.Result, tc.want, decisions.Relation)
				}
				if decisions.Relation.Undecided != nil {
					t.Fatalf("a decided rule carries an undecided reason: %+v", decisions.Relation.Undecided)
				}
				return
			}
			if decisions.Relation.Result != "" {
				t.Fatalf("result=%q, want no result", decisions.Relation.Result)
			}
			if decisions.Relation.Undecided == nil || decisions.Relation.Undecided.Reason != tc.wantWhy {
				t.Fatalf("undecided=%+v, want the reason %q", decisions.Relation.Undecided, tc.wantWhy)
			}
			if decisions.Relation.Undecided.Relation != membersID {
				t.Fatalf("the reason names relation %q, want %q", decisions.Relation.Undecided.Relation, membersID)
			}
		})
	}
}

// A row that holds no value is reported as itself, before the coverage: reading
// more rows cannot settle a value that is absent in a row that was read.
func TestDecideReportsARowReasonBeforeTheCoverage(t *testing.T) {
	detail := detailOf(t, validationeval.CoveragePartial,
		row(map[core.CanonicalID]any{allocation: "1"}),
		row(map[core.CanonicalID]any{allocation: nil}),
	)
	decisions, err := detail.Decide(rowScopedRule(true), rowScopedOperators(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if decisions.Relation == nil || decisions.Relation.Undecided == nil {
		t.Fatalf("decision=%+v, want an undecided answer", decisions.Relation)
	}
	if decisions.Relation.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("reason=%q, want %q", decisions.Relation.Undecided.Reason, validationeval.UndecidedValueNull)
	}
	if decisions.Relation.Undecided.Attribute != allocation {
		t.Fatalf("attribute=%q, want %q", decisions.Relation.Undecided.Attribute, allocation)
	}
}

// The dispatch is the declaration's, not the rows'. The same operator without
// `of each row` is asked once per row, so its answers stay in Rows.
func TestDecideAsksAPerRecordOperatorOncePerRow(t *testing.T) {
	operators := rowScopedOperators(t, false)
	operators.Declared[0].Inputs[0].OfEachRow = false
	detail := detailOf(t, validationeval.CoverageComplete, counterexampleRows()...)
	decisions, err := detail.Decide(rowScopedRule(false), operators)
	if err != nil {
		t.Fatal(err)
	}
	if decisions.Relation != nil {
		t.Fatalf("a rule about one record was answered for the relation: %+v", decisions.Relation)
	}
	want := []validationeval.Result{
		validationeval.ResultSatisfied, validationeval.ResultSatisfied, validationeval.ResultNotSatisfied,
	}
	if len(decisions.Rows) != len(want) {
		t.Fatalf("rows=%+v, want %d answers", decisions.Rows, len(want))
	}
	for index, expected := range want {
		if decisions.Rows[index].Result != expected {
			t.Fatalf("row %d result=%q, want %q", index, decisions.Rows[index].Result, expected)
		}
	}
}

// A reader that can answer for one row only cannot answer a rule about all the
// rows. Asking it anyway is refused, not answered row by row.
func TestDecideRefusesARowScopedRuleWhenNoImplementationReadsRows(t *testing.T) {
	operators := rowScopedOperators(t, true)
	operators.Implementations = nil
	detail := detailOf(t, validationeval.CoverageComplete, counterexampleRows()...)
	if _, err := detail.Decide(rowScopedRule(true), operators); err == nil {
		t.Fatal("a rule that reads every row was decided with no implementation that reads rows")
	}
}
