package validationeval_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
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
func TestRowScopedDecisionRefusesADecidedAnswerWithARowReason(t *testing.T) {
	reason := &validationeval.Undecided{
		Reason: validationeval.UndecidedValueNull, Relation: rowScopedRule.Relation, Attribute: rowScopedRule.Attribute,
	}
	for _, answer := range []validationeval.Tri{validationeval.TriYes, validationeval.TriNo} {
		decision, err := validationeval.DecideRowScoped(rowScopedRule, validationeval.CoverageComplete, true, answer, reason)
		if err == nil {
			t.Fatalf("answer %d with a row reason was accepted: %+v", int(answer), decision)
		}
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

// rowsSource answers for the rows of one relation and refuses the single-record
// question, so a rule answered from one row is caught instead of passing.
type rowsSource struct {
	coverage validationeval.Coverage
	columns  map[core.CanonicalID][]validationeval.NumberReading
	// reads records which attributes were asked for, so a test can show that
	// each input is read once and not once per row.
	reads []core.CanonicalID
}

func (s *rowsSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, fmt.Errorf("this reader answers for the rows of a relation, not for one record")
}

func (s *rowsSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("this reader reads numbers, not keys")
}

func (s *rowsSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	s.reads = append(s.reads, attribute)
	rows, ok := s.columns[attribute]
	if !ok {
		return validationeval.NumberRowsReading{}, fmt.Errorf("attribute %s was not read", attribute)
	}
	return validationeval.NumberRowsReading{Coverage: s.coverage, Rows: rows}, nil
}

// oneRecordSource answers only the single-record question. It stands for every
// reader the rule path already had before row-scoped inputs existed.
type oneRecordSource struct{}

func (oneRecordSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, fmt.Errorf("one record only")
}

func (oneRecordSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("one record only")
}

// The expected answers below are written by hand from decision section 15.2's
// table and from the rule: sales.below_limit holds when every row's value is
// below 100, and it declares `refutes on some rows`.
var rowOperatorRule = semantic.Validation{
	ID:        "sales.order.lines.amount_below_limit",
	Symbol:    "amount_below_limit",
	Kind:      semantic.ValidationOperator,
	Attribute: "sales.order_line.amount",
	Relation:  "sales.order.lines",
	Operator:  "sales.below_limit",
}

func belowLimitContract(refutes bool) semantic.Operator {
	return semantic.Operator{
		ID:     "sales.below_limit",
		Symbol: "below_limit",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: refutes,
	}
}

func belowLimitImplementation() validationeval.OperatorImplementation {
	// The plain reading of the contract, written here so a wrong evaluator is
	// caught even when the shipped implementation is wrong too.
	return validationeval.OperatorImplementation{
		Operator: "sales.below_limit",
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			values, ok := rows["value"]
			if !ok {
				return validationeval.TriMaybe, fmt.Errorf("no value column")
			}
			for _, row := range values {
				switch row.State {
				case validationeval.ValueNotFetched:
					return validationeval.TriMaybe, nil
				case validationeval.ValueNull:
					return validationeval.TriMaybe, nil
				case validationeval.ValuePresent:
					if row.Value.Cmp(big.NewRat(100, 1)) >= 0 {
						return validationeval.TriNo, nil
					}
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func present(value int64) validationeval.NumberReading {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(value, 1)}
}

func decideRowOperator(t *testing.T, source *rowsSource, refutes bool) validationeval.Decision {
	t.Helper()
	decision, err := validationeval.Decide(rowOperatorRule, source, validationeval.Operators{
		Declared:        []semantic.Operator{belowLimitContract(refutes)},
		Implementations: []validationeval.OperatorImplementation{belowLimitImplementation()},
	})
	if err != nil {
		t.Fatalf("refutes=%v: err=%v", refutes, err)
	}
	return decision
}

// A counterexample that was read is a decision when the operator states that one
// witnessed counterexample is enough, whatever the coverage.
func TestRowOperatorCounterexampleWithRefutationIsNotSatisfied(t *testing.T) {
	source := &rowsSource{
		coverage: validationeval.CoveragePartial,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.amount": {present(10), present(200)},
		},
	}
	decision := decideRowOperator(t, source, true)
	if decision.Result != validationeval.ResultNotSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want not-satisfied with no reason", decision)
	}
	// One input is read once, not once per row.
	if len(source.reads) != 1 {
		t.Fatalf("reads=%v, want one read of the one input", source.reads)
	}
}

// The same counterexample without that statement is not enough: unread rows
// might hold anything, so the answer is undecided, not not-satisfied.
func TestRowOperatorCounterexampleWithoutRefutationIsUndecided(t *testing.T) {
	source := &rowsSource{
		coverage: validationeval.CoveragePartial,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.amount": {present(10), present(200)},
		},
	}
	decision := decideRowOperator(t, source, false)
	if decision.Result != "" || decision.Undecided == nil ||
		decision.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("decision=%+v, want undecided relation-partially-read", decision)
	}
}

// With every row read, the statement makes no difference: a counterexample is a
// counterexample, and its absence is an answer.
func TestRowOperatorCompleteReadAnswersBothWays(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows []validationeval.NumberReading
		want validationeval.Result
	}{
		{"counterexample", []validationeval.NumberReading{present(10), present(200)}, validationeval.ResultNotSatisfied},
		{"all below", []validationeval.NumberReading{present(10), present(20)}, validationeval.ResultSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &rowsSource{
				coverage: validationeval.CoverageComplete,
				columns:  map[core.CanonicalID][]validationeval.NumberReading{"sales.order_line.amount": tc.rows},
			}
			decision := decideRowOperator(t, source, true)
			if decision.Result != tc.want || decision.Undecided != nil {
				t.Fatalf("decision=%+v, want %q", decision, tc.want)
			}
		})
	}
}

// A rule whose operator reads every row but names no relation cannot be asked:
// which rows, and how many, would be unknown.
func TestRowOperatorWithoutRelationIsRefused(t *testing.T) {
	rule := rowOperatorRule
	rule.Relation = ""
	_, err := validationeval.Decide(rule, &rowsSource{coverage: validationeval.CoverageComplete}, validationeval.Operators{
		Declared:        []semantic.Operator{belowLimitContract(true)},
		Implementations: []validationeval.OperatorImplementation{belowLimitImplementation()},
	})
	if err == nil || !strings.Contains(err.Error(), "names no relation") {
		t.Fatalf("err=%v, want a refusal naming the missing relation", err)
	}
}

// A reader that cannot read the rows of a relation is refused, not asked row by
// row: a rule about all the rows cannot be answered from one row.
func TestRowOperatorNeedsARowReadingReader(t *testing.T) {
	_, err := validationeval.Decide(rowOperatorRule, oneRecordSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{belowLimitContract(true)},
		Implementations: []validationeval.OperatorImplementation{belowLimitImplementation()},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot read the rows of a relation") {
		t.Fatalf("err=%v, want a refusal naming the reader", err)
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
