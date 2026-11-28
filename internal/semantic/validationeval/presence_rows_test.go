package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input read over every row asks, of the relation, "does every row
// hold a value?" (decision sections 3 and 15.1). This file pins the three-valued
// answer, the reason a rule stays undecided, and the composition with the read
// Coverage and the operator's `refutes on some rows` statement.
//
// The expectations are written from that table, not read back from the
// evaluator:
//
//   - every row holds a value            -> satisfied (when the rows are all read)
//   - one row holds no value             -> not-satisfied, when `refutes on some rows`
//   - a row nobody read                  -> undecided (value-not-fetched)
//   - a partly read relation, no refutation -> undecided (relation-partially-read)

// presenceRowSource reports one column of presence states and one coverage, so
// a rule can be asked of a relation whose rows are known.
type presenceRowSource struct {
	column   []validationeval.PresenceReading
	coverage validationeval.Coverage
}

func (s presenceRowSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, failureOf("a rule about the rows asked for one record's number")
}

func (s presenceRowSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{Coverage: s.coverage}, nil
}

func (s presenceRowSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	return "one-read", true
}

func (s presenceRowSource) ReadRelationPresence(core.CanonicalID, core.CanonicalID) (validationeval.PresenceRowsReading, error) {
	return validationeval.PresenceRowsReading{Coverage: s.coverage, Rows: s.column}, nil
}

// failureOf is a type that only satisfies error, so a test source can say why a
// read it does not implement was asked for.
type failureOf string

func (f failureOf) Error() string { return string(f) }

// rowPresenceRule builds the contract and the rule for one row-scoped presence
// input, and the implementation that answers it.
func rowPresenceRule(refutes bool) (semantic.Validation, validationeval.Operators) {
	contract := semantic.Operator{
		ID: "sales.every_present", Symbol: "every_present",
		Input:             semantic.OperatorInputPresence,
		Inputs:            []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true}},
		RefutesOnSomeRows: refutes,
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.every_price_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.unit_price", Operator: "sales.every_present",
		Relation: "sales.order.lines",
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.every_present",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			return presenceEveryPresent(rows["value"])
		},
	}
	return rule, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}
}

// presenceEveryPresent is the definition the test implementation follows: a row
// that holds no value refutes, a row nobody read leaves the answer unknown, and
// otherwise every row holds a value.
func presenceEveryPresent(column []validationeval.PresenceReading) (validationeval.Tri, error) {
	unread := false
	for _, row := range column {
		switch row.State {
		case validationeval.ValueNull:
			return validationeval.TriNo, nil
		case validationeval.ValueNotFetched:
			unread = true
		case validationeval.ValuePresent:
		}
	}
	if unread {
		return validationeval.TriMaybe, nil
	}
	return validationeval.TriYes, nil
}

// TestARowScopedPresenceRuleIsDecidedFromTheRows pins the answers and the
// reasons a rule over a relation's presence can have.
func TestARowScopedPresenceRuleIsDecidedFromTheRows(t *testing.T) {
	const (
		ruleID = "sales.order.lines.every_price_stated"
	)
	cases := []struct {
		name         string
		refutes      bool
		coverage     validationeval.Coverage
		column       []validationeval.PresenceReading
		result       validationeval.Result
		undecided    validationeval.UndecidedReason
		undecidedSet bool
	}{
		{
			name:     "every row holds a value",
			refutes:  true,
			coverage: validationeval.CoverageComplete,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent}, {State: validationeval.ValuePresent},
			},
			result: validationeval.ResultSatisfied,
		},
		{
			name:     "one row holds no value, and a witnessed refutation settles it",
			refutes:  true,
			coverage: validationeval.CoverageComplete,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent}, {State: validationeval.ValueNull},
			},
			result: validationeval.ResultNotSatisfied,
		},
		{
			name:     "one row holds no value, and the operator does not refute on some rows while the rows are all read",
			refutes:  false,
			coverage: validationeval.CoverageComplete,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent}, {State: validationeval.ValueNull},
			},
			result: validationeval.ResultNotSatisfied,
		},
		{
			name:     "a row nobody read leaves the rule undecided",
			refutes:  true,
			coverage: validationeval.CoverageComplete,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent}, {State: validationeval.ValueNotFetched},
			},
			undecided:    validationeval.UndecidedValueNotFetched,
			undecidedSet: true,
		},
		{
			name:     "a partly read relation whose rows all hold a value is not satisfied",
			refutes:  false,
			coverage: validationeval.CoveragePartial,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent},
			},
			undecided:    validationeval.UndecidedRelationPartiallyRead,
			undecidedSet: true,
		},
		{
			name:     "a partly read relation with a refutation is refuted",
			refutes:  true,
			coverage: validationeval.CoveragePartial,
			column: []validationeval.PresenceReading{
				{State: validationeval.ValuePresent}, {State: validationeval.ValueNull},
			},
			result: validationeval.ResultNotSatisfied,
		},
		{
			name:     "the empty relation satisfies every row holding a value",
			refutes:  true,
			coverage: validationeval.CoverageComplete,
			column:   nil,
			result:   validationeval.ResultSatisfied,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule, operators := rowPresenceRule(tc.refutes)
			decision, err := validationeval.Decide(rule, presenceRowSource{column: tc.column, coverage: tc.coverage}, operators)
			if err != nil {
				t.Fatalf("the rule was refused: %v", err)
			}
			if tc.undecidedSet {
				if decision.Result != "" {
					t.Fatalf("result=%q, want no result", decision.Result)
				}
				if decision.Undecided == nil || decision.Undecided.Reason != tc.undecided {
					t.Fatalf("undecided=%+v, want reason %q", decision.Undecided, tc.undecided)
				}
				return
			}
			if decision.Result != tc.result {
				t.Fatalf("result=%q, want %q", decision.Result, tc.result)
			}
			if decision.Undecided != nil {
				t.Fatalf("a decided rule carries an undecided reason: %+v", decision.Undecided)
			}
		})
	}
}

// TestARowScopedPresenceRuleRefusesAReaderThatCannotReadIt pins the difference
// between a reader that cannot answer the row question and a row that holds no
// value: the first is an error of the reader, the second is a decided answer.
func TestARowScopedPresenceRuleRefusesAReaderThatCannotReadIt(t *testing.T) {
	rule, operators := rowPresenceRule(true)
	if _, err := validationeval.Decide(rule, numbersOnly{}, operators); err == nil {
		t.Fatal("a rule over a relation's presence was read by a reader that cannot answer it")
	}
}

// TestARowScopedPresenceRuleReadsTheAttributeItNames pins that a named input
// reads the attribute the rule assigned, not the subject's attribute again.
func TestARowScopedPresenceRuleReadsTheAttributeItNames(t *testing.T) {
	const (
		subject = "sales.order_line.unit_price"
		other   = "sales.order_line.item_code"
	)
	reader := &namedPresenceRowSource{column: []validationeval.PresenceReading{
		{State: validationeval.ValuePresent},
	}}
	reader.byAttribute = map[core.CanonicalID][]validationeval.PresenceReading{
		subject: {validationeval.PresenceReading{State: validationeval.ValuePresent}},
		other:   {validationeval.PresenceReading{State: validationeval.ValueNull}},
	}
	contract := semantic.Operator{
		ID: "sales.both_present", Symbol: "both_present",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
			{Name: "other", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.both_stated", Kind: semantic.ValidationOperator,
		Attribute: subject, Operator: "sales.both_present", Relation: "sales.order.lines",
		Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: other}},
	}
	var seen map[string][]validationeval.PresenceReading
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.both_present",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			seen = rows
			return validationeval.TriNo, nil
		},
	}
	if _, err := validationeval.Decide(rule, reader, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}); err != nil {
		t.Fatalf("the two-input rule was refused: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("the implementation received %d inputs, want 2", len(seen))
	}
	if got := seen["value"]; len(got) != 1 || got[0].State != validationeval.ValuePresent {
		t.Fatalf("value=%+v, want one present row from the subject attribute", got)
	}
	if got := seen["other"]; len(got) != 1 || got[0].State != validationeval.ValueNull {
		t.Fatalf("other=%+v, want one null row from the assigned attribute", got)
	}
}

// namedPresenceRowSource answers the row question for the attributes a test
// states, so a rule can bind two inputs to two columns.
type namedPresenceRowSource struct {
	column      []validationeval.PresenceReading
	byAttribute map[core.CanonicalID][]validationeval.PresenceReading
}

func (s *namedPresenceRowSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, failureOf("a rule about the rows asked for one record's number")
}

func (s *namedPresenceRowSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *namedPresenceRowSource) ReadAttempt(core.CanonicalID) (string, bool) {
	return "one-read", true
}

func (s *namedPresenceRowSource) ReadRelationPresence(_ core.CanonicalID, attribute core.CanonicalID) (validationeval.PresenceRowsReading, error) {
	return validationeval.PresenceRowsReading{Coverage: validationeval.CoverageComplete, Rows: s.byAttribute[attribute]}, nil
}
