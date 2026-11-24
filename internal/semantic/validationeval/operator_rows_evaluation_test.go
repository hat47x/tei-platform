package validationeval_test

import (
	"errors"
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator reads every row of a relation reaches the same Decide
// path every other kind uses. These tests read rows from a synthetic reader, so
// what they pin is the reading and the composition, not a database: the reader
// records every attribute it was asked for, so two inputs of one rule that were
// read from one attempt are visible as one.

// rowsSource is a stateful reader that can read a column of one attribute over
// the rows of a relation. Every call is recorded, and the attempt is one for
// the whole reader: one row set is one read.
type rowsSource struct {
	columns  map[string][]validationeval.NumberReading
	coverage validationeval.Coverage
	calls    []string
}

func (s *rowsSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, errors.New("this reader reads columns, not one value")
}

func (s *rowsSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, errors.New("this reader reads numbers, not keys")
}

func (s *rowsSource) ReadAttempt(core.CanonicalID) (string, bool) { return "rows-read-1", true }

func (s *rowsSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	if string(relation) != "sales.order.lines" {
		return validationeval.NumberRowsReading{}, errors.New("unknown relation " + string(relation))
	}
	column, held := s.columns[string(attribute)]
	if !held {
		return validationeval.NumberRowsReading{}, errors.New("unknown attribute " + string(attribute))
	}
	s.calls = append(s.calls, string(attribute))
	return validationeval.NumberRowsReading{Coverage: s.coverage, Rows: column}, nil
}

// valueReader is a stateful reader that cannot read a column. A rule about all
// the rows must be refused by it rather than answered from one row.
type valueReader struct{}

func (valueReader) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}, nil
}

func (valueReader) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func rowsRule(inputs ...semantic.ValidationInputAssignment) semantic.Validation {
	return semantic.Validation{
		ID:        "sales.order.lines.amount_below_limit",
		Symbol:    "amount_below_limit",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.amount",
		Relation:  "sales.order.lines",
		Operator:  "sales.below_limit",
		Inputs:    inputs,
	}
}

func rowsOperators(declared semantic.Operator, implementation validationeval.OperatorImplementation) validationeval.Operators {
	return validationeval.Operators{Declared: []semantic.Operator{declared}, Implementations: []validationeval.OperatorImplementation{implementation}}
}

// belowLimit decides "every amount is at most the limit". The limit is the
// second input and is read over the same rows, so the rule is decided from one
// reading of one relation.
func belowLimitContract() semantic.Operator {
	return semantic.Operator{
		ID: "sales.below_limit", Symbol: "below_limit", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
	}
}

func belowLimitImplementation() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.below_limit", Input: semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			amounts, limits := rows["value"], rows["limit"]
			if len(amounts) != len(limits) {
				return validationeval.TriMaybe, errors.New("the two inputs were read over a different number of rows")
			}
			for index := range amounts {
				if amounts[index].State != validationeval.ValuePresent || limits[index].State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if amounts[index].Value.Cmp(limits[index].Value) > 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func column(values ...string) []validationeval.NumberReading {
	out := make([]validationeval.NumberReading, 0, len(values))
	for _, text := range values {
		value, ok := new(big.Rat).SetString(text)
		if !ok {
			panic(text)
		}
		out = append(out, validationeval.NumberReading{State: validationeval.ValuePresent, Value: value})
	}
	return out
}

// Every row was read and every amount is at most its limit, so the rule holds.
func TestRowScopedRuleIsDecidedFromOneReadingOfTheRelation(t *testing.T) {
	source := &rowsSource{coverage: validationeval.CoverageComplete, columns: map[string][]validationeval.NumberReading{
		"sales.order_line.amount": column("10", "20"),
		"sales.order_line.limit":  column("100", "100"),
	}}
	rule := rowsRule(semantic.ValidationInputAssignment{Name: "limit", Value: "sales.order_line.limit"})
	got, err := validationeval.Decide(rule, source, rowsOperators(belowLimitContract(), belowLimitImplementation()))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("got=%+v, want satisfied", got)
	}
	// Both inputs were read, and both from this one reader.
	if len(source.calls) != 2 {
		t.Fatalf("calls=%v, want one reading per input", source.calls)
	}
}

// A counterexample in a partly read relation is decided only when the operator
// states that a witnessed counterexample is enough.
func TestRowScopedRefutationFollowsTheDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		refutes bool
		want    validationeval.Result
		wantWhy validationeval.UndecidedReason
	}{
		{"with the statement", true, validationeval.ResultNotSatisfied, ""},
		{"without the statement", false, "", validationeval.UndecidedRelationPartiallyRead},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &rowsSource{coverage: validationeval.CoveragePartial, columns: map[string][]validationeval.NumberReading{
				"sales.order_line.amount": column("10", "200"),
				"sales.order_line.limit":  column("100", "100"),
			}}
			contract := belowLimitContract()
			contract.RefutesOnSomeRows = tc.refutes
			rule := rowsRule(semantic.ValidationInputAssignment{Name: "limit", Value: "sales.order_line.limit"})
			got, err := validationeval.Decide(rule, source, rowsOperators(contract, belowLimitImplementation()))
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if tc.want != "" {
				if got.Result != tc.want || got.Undecided != nil {
					t.Fatalf("got=%+v, want %q", got, tc.want)
				}
				return
			}
			if got.Result != "" || got.Undecided == nil || got.Undecided.Reason != tc.wantWhy {
				t.Fatalf("got=%+v, want the reason %q", got, tc.wantWhy)
			}
		})
	}
}

// A reader that cannot read the rows of a relation refuses the rule. Answering
// it from one row would answer a rule about all the rows from one record.
func TestRowScopedRuleIsRefusedByAReaderWithoutColumns(t *testing.T) {
	contract := belowLimitContract()
	contract.Inputs = contract.Inputs[:1]
	rule := rowsRule()
	if _, err := validationeval.Decide(rule, valueReader{}, rowsOperators(contract, belowLimitImplementation())); err == nil {
		t.Fatal("a rule about all the rows was decided by a reader that reads one value")
	}
}

// An input read over the rows names the attribute its column comes from. A
// constant has no column, so it is refused instead of being read as one value
// among the rows.
func TestRowScopedInputRefusesAConstant(t *testing.T) {
	source := &rowsSource{coverage: validationeval.CoverageComplete, columns: map[string][]validationeval.NumberReading{
		"sales.order_line.amount": column("10", "20"),
	}}
	contract := belowLimitContract()
	rule := rowsRule(semantic.ValidationInputAssignment{Name: "limit", Value: "100"})
	if _, err := validationeval.Decide(rule, source, rowsOperators(contract, belowLimitImplementation())); err == nil {
		t.Fatal("a constant was read as a column")
	}
}

// An implementation that answers certainly around a row it never saw is
// refused: it cannot have decided around a value that was absent.
func TestRowScopedAnswerAroundAnAbsentRowIsRefused(t *testing.T) {
	source := &rowsSource{coverage: validationeval.CoverageComplete, columns: map[string][]validationeval.NumberReading{
		"sales.order_line.amount": {{State: validationeval.ValueNull}},
		"sales.order_line.limit":  column("100"),
	}}
	implementation := belowLimitImplementation()
	implementation.NumberRows = func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
		return validationeval.TriYes, nil
	}
	rule := rowsRule(semantic.ValidationInputAssignment{Name: "limit", Value: "sales.order_line.limit"})
	if _, err := validationeval.Decide(rule, source, rowsOperators(belowLimitContract(), implementation)); err == nil {
		t.Fatal("a decided answer was accepted beside a row that holds no value")
	}
}

// A contract that reads every row states no example this shape can check: an
// example carries one value, and a rule decided from the rows together is not
// one value. Selecting an implementation for such a contract is refused rather
// than asked about an example it never saw.
func TestRowScopedContractWithExamplesIsRefused(t *testing.T) {
	contract := belowLimitContract()
	contract.Cases = []semantic.OperatorCase{{Value: "10", Satisfied: true}}
	rule := rowsRule(semantic.ValidationInputAssignment{Name: "limit", Value: "sales.order_line.limit"})
	operators := rowsOperators(contract, belowLimitImplementation())
	if _, err := validationeval.Decide(rule, &rowsSource{coverage: validationeval.CoverageComplete, columns: map[string][]validationeval.NumberReading{
		"sales.order_line.amount": column("10"),
		"sales.order_line.limit":  column("100"),
	}}, operators); err == nil {
		t.Fatal("a contract that reads every row was decided from its examples")
	}
}
