package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The expected values below are written by hand from decision section 15.2's
// table: a counterexample that was read is a decision whatever the coverage,
// but only when the operator states `refutes on some rows`; a partly read
// relation with no counterexample is undecided; and a reason inside the rows
// is reported before the coverage.

var rowScopedRule = semantic.Validation{
	ID:        "sales.order.lines.amount_below_limit",
	Symbol:    "amount_below_limit",
	Kind:      semantic.ValidationOperator,
	Attribute: "sales.order_line.amount",
	Relation:  "sales.order.lines",
	Operator:  "sales.below_limit",
}

func rowScopedDecision(t *testing.T, coverage validationeval.Coverage, refutes bool, answer validationeval.Tri, reason *validationeval.Undecided) validationeval.Decision {
	t.Helper()
	rule := rowScopedRule
	// decideRowScoped is the package's one place for this composition; the
	// tests reach it through the exported entry so a wrong composition is
	// caught even when the rule's own evaluator changes.
	decision, err := validationeval.DecideRowScoped(rule, coverage, refutes, answer, reason)
	if err != nil {
		t.Fatalf("coverage=%q refutes=%v answer=%d: err=%v", coverage, refutes, answer, err)
	}
	return decision
}

func TestRowScopedDecisionTable(t *testing.T) {
	nullReason := &validationeval.Undecided{
		Reason: validationeval.UndecidedValueNull, Relation: rowScopedRule.Relation, Attribute: rowScopedRule.Attribute,
	}
	cases := []struct {
		name     string
		coverage validationeval.Coverage
		refutes  bool
		answer   validationeval.Tri
		reason   *validationeval.Undecided
		want     validationeval.Result
		wantWhy  validationeval.UndecidedReason
	}{
		// (a) a counterexample in a partly read relation, with the statement
		// that a witnessed counterexample is enough.
		{name: "counterexample, partial, refutes", coverage: validationeval.CoveragePartial, refutes: true, answer: validationeval.TriNo, want: validationeval.ResultNotSatisfied},
		// (b) the same counterexample without that statement is not enough.
		{name: "counterexample, partial, no refutation", coverage: validationeval.CoveragePartial, refutes: false, answer: validationeval.TriNo, wantWhy: validationeval.UndecidedRelationPartiallyRead},
		// (c) every row was read, so the statement does not matter.
		{name: "counterexample, complete, refutes", coverage: validationeval.CoverageComplete, refutes: true, answer: validationeval.TriNo, want: validationeval.ResultNotSatisfied},
		{name: "counterexample, complete, no refutation", coverage: validationeval.CoverageComplete, refutes: false, answer: validationeval.TriNo, want: validationeval.ResultNotSatisfied},
		// (d) no counterexample in a partly read relation: unread rows may
		// still hold one.
		{name: "no counterexample, partial", coverage: validationeval.CoveragePartial, refutes: true, answer: validationeval.TriYes, wantWhy: validationeval.UndecidedRelationPartiallyRead},
		{name: "no counterexample, complete", coverage: validationeval.CoverageComplete, refutes: false, answer: validationeval.TriYes, want: validationeval.ResultSatisfied},
		// (e) a reason inside the rows is reported before the coverage. An
		// implementation that already answered TriYes or TriNo around such a
		// row is refused instead: it cannot have decided around a value it
		// never saw.
		{name: "null inside rows, partial, maybe", coverage: validationeval.CoveragePartial, answer: validationeval.TriMaybe, reason: nullReason, wantWhy: validationeval.UndecidedValueNull},
		{name: "null inside rows, complete, maybe", coverage: validationeval.CoverageComplete, answer: validationeval.TriMaybe, reason: nullReason, wantWhy: validationeval.UndecidedValueNull},
		// (f) the operator could not tell from what it read.
		{name: "maybe", coverage: validationeval.CoverageComplete, answer: validationeval.TriMaybe, wantWhy: validationeval.UndecidedValueNotFetched},
		{name: "maybe, partial", coverage: validationeval.CoveragePartial, answer: validationeval.TriMaybe, wantWhy: validationeval.UndecidedValueNotFetched},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := rowScopedDecision(t, tc.coverage, tc.refutes, tc.answer, tc.reason)
			if tc.want != "" {
				if decision.Result != tc.want {
					t.Fatalf("got result %q, want %q (decision=%+v)", decision.Result, tc.want, decision)
				}
				if decision.Undecided != nil {
					t.Fatalf("a decided rule carries an undecided reason: %+v", decision.Undecided)
				}
				return
			}
			if decision.Result != "" {
				t.Fatalf("got result %q, want no result", decision.Result)
			}
			if decision.Undecided == nil || decision.Undecided.Reason != tc.wantWhy {
				t.Fatalf("undecided=%+v, want the reason %q", decision.Undecided, tc.wantWhy)
			}
		})
	}
}

// A decided answer reported together with a reason inside the rows is a
// contradiction: the implementation would have decided around a value that was
// absent or was never read.
// A reason inside the rows is settled for TriNo but not for TriYes: a
// counterexample that was seen decides the rule whatever else is unknown
// (section 6), while "it holds everywhere" cannot be claimed over a row that
// holds no value or was not read.
func TestRowScopedDecisionRowReasonWithADecidedAnswer(t *testing.T) {
	reason := &validationeval.Undecided{
		Reason: validationeval.UndecidedValueNull, Relation: rowScopedRule.Relation, Attribute: rowScopedRule.Attribute,
	}
	// TriNo: the counterexample stands, so an absent row elsewhere does not
	// turn it into an error or into undecided.
	decision, err := validationeval.DecideRowScoped(rowScopedRule, validationeval.CoverageComplete, true, validationeval.TriNo, reason)
	if err != nil {
		t.Fatalf("a witnessed counterexample with an absent row was refused: %v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
	// TriYes: the absent row might break the rule, so the answer is undecided
	// with that row's reason rather than satisfied.
	decision, err = validationeval.DecideRowScoped(rowScopedRule, validationeval.CoverageComplete, true, validationeval.TriYes, reason)
	if err != nil {
		t.Fatalf("answer TriYes with a row reason returned an error: %v", err)
	}
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}

// A coverage that is neither complete nor partial is a reader error, not an
// answer: the evaluator must not guess which half it means.
func TestRowScopedDecisionRefusesUnknownCoverage(t *testing.T) {
	for _, coverage := range []validationeval.Coverage{"", "unknown", "all"} {
		decision, err := validationeval.DecideRowScoped(rowScopedRule, coverage, true, validationeval.TriNo, nil)
		if err == nil {
			t.Fatalf("coverage %q was accepted: %+v", coverage, decision)
		}
		if decision.Result != "" || decision.Undecided != nil {
			t.Fatalf("coverage %q produced a decision: %+v", coverage, decision)
		}
	}
}

// A row-scoped answer that is not one of the three is a programming error in
// the implementation, not a not-satisfied rule.
func TestRowScopedDecisionRefusesAnUnknownAnswer(t *testing.T) {
	decision, err := validationeval.DecideRowScoped(rowScopedRule, validationeval.CoverageComplete, true, validationeval.Tri(99), nil)
	if err == nil {
		t.Fatalf("an unknown answer was accepted: %+v", decision)
	}
}

// A row-scoped implementation is selected and called through the same
// resolution the flat and single-input implementations use, and the call is
// recorded once.
func TestRowScopedImplementationIsResolvedAndCalled(t *testing.T) {
	var calls []validationeval.ImplementationCall
	available := []validationeval.NamedImplementation{{
		ID:    "example.rows",
		Input: semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			// The expected answer is written here: one row holds 10 and the
			// other 200, against a limit of 100, so a counterexample was read.
			values := rows["value"]
			if len(values) != 2 {
				t.Fatalf("rows=%v", values)
			}
			return validationeval.TriNo, nil
		},
	}}
	resolved, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.below_limit", Implementation: "example.rows"}},
		available,
		func(call validationeval.ImplementationCall) { calls = append(calls, call) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].NumberRows == nil {
		t.Fatalf("resolved=%+v", resolved)
	}
	rows := map[string][]validationeval.NumberReading{
		"value": {
			{State: validationeval.ValuePresent, Value: big.NewRat(10, 1)},
			{State: validationeval.ValuePresent, Value: big.NewRat(200, 1)},
		},
	}
	got, err := resolved[0].NumberRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got != validationeval.TriNo {
		t.Fatalf("got %d, want TriNo", int(got))
	}
	if len(calls) != 1 || calls[0].Implementation != "example.rows" {
		t.Fatalf("calls=%+v, want one call to example.rows", calls)
	}
}

// Two selected implementations of one operator are still refused; adding a
// third kind of decision function must not change that.
func TestRowScopedResolutionKeepsTheOneImplementationRule(t *testing.T) {
	available := []validationeval.NamedImplementation{
		{ID: "example.a", Input: semantic.OperatorInputNumber, NumberRows: func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		}},
		{ID: "example.b", Input: semantic.OperatorInputNumber, Number: func(*big.Rat) (bool, error) { return true, nil }},
	}
	_, err := validationeval.ResolveImplementations([]validationeval.ImplementationSelection{
		{Operator: "sales.below_limit", Implementation: "example.a"},
		{Operator: "sales.below_limit", Implementation: "example.b"},
	}, available, nil)
	if err == nil {
		t.Fatal("two selections for one operator were accepted")
	}
}
