package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// #5905 made the rule subject's declared `absent as <mapping>` reach the
// per-record path. The row-scoped path was added later and reads the subject
// through its own loop, which only consults the mapping of the inputs after the
// first: the subject's mapping is never applied there, so a row whose value is
// absent stays undecided while the contract says how to read it.
//
// These expectations are written from that statement, not from the code.

// rowMappingSource answers one attribute over every row.
type rowMappingSource struct {
	coverage validationeval.Coverage
	columns  map[core.CanonicalID][]validationeval.NumberReading
}

func (s *rowMappingSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, nil
}

func (s *rowMappingSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *rowMappingSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	rows, ok := s.columns[attribute]
	if !ok {
		return validationeval.NumberRowsReading{}, nil
	}
	return validationeval.NumberRowsReading{Coverage: s.coverage, Rows: rows}, nil
}

// openIs reports an implementation that reads the absence named open as the
// value it is given.
func openIs(value string) func(string) (*big.Rat, bool) {
	bound, _ := new(big.Rat).SetString(value)
	return func(mapping string) (*big.Rat, bool) {
		if mapping != "open" {
			return nil, false
		}
		return bound, true
	}
}

// subjectAbsentRule states that every row's value is at most the limit, with
// the subject's absent value read as open.
func subjectAbsentRule() (semantic.Validation, semantic.Operator) {
	contract := semantic.Operator{
		ID:    "sales.at_most_with_open",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true, AbsentAs: "open"},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
	rule := semantic.Validation{
		ID:        "sales.order_line.subject_absent",
		Symbol:    "subject_absent",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity",
		Relation:  "sales.order.lines",
		Operator:  "sales.at_most_with_open",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "limit", Value: "sales.order_line.unit_price"},
		},
	}
	return rule, contract
}

// atMostRows answers TriYes when every row of the value column holds a value
// and TriMaybe otherwise, so the answer says whether the mapping replaced the
// absent row or left it absent.
func atMostRows(absent func(string) (*big.Rat, bool)) validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.at_most_with_open",
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			for _, row := range rows["value"] {
				if row.State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
			}
			return validationeval.TriYes, nil
		},
		Absent: absent,
	}
}

// A row whose subject value is absent is read under the declared mapping, so
// the implementation is asked about a value rather than reported as having one
// absent.
func TestRowScopedSubjectUsesItsDeclaredMapping(t *testing.T) {
	rule, contract := subjectAbsentRule()
	quantity, _ := core.ParseCanonicalID("sales.order_line.quantity")
	price, _ := core.ParseCanonicalID("sales.order_line.unit_price")
	source := &rowMappingSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			// The subject column holds one absent row; the limit column holds
			// a value for both rows, so the only reason can come from the
			// subject.
			quantity: {{State: validationeval.ValueNull}, {State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
			price:    {{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}, {State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
		},
	}
	got, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{atMostRows(openIs("1"))},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// The mapping replaced the absent row with a value, and the implementation
	// saw both rows, so its answer is the decision. Before the fix the absent
	// row was reported as a reason instead.
	if got.Result != validationeval.ResultSatisfied || got.Undecided != nil {
		t.Fatalf("got=%+v, want satisfied: the mapping should have made the row present", got)
	}
}

// Without a mapping the absent row keeps its own reason, as before: this must
// not start reading an absence as zero.
func TestRowScopedSubjectWithoutAMappingStaysUndecided(t *testing.T) {
	rule := subjectAbsentRule
	r, contract := rule()
	contract.Inputs[0].AbsentAs = ""
	quantity, _ := core.ParseCanonicalID("sales.order_line.quantity")
	price, _ := core.ParseCanonicalID("sales.order_line.unit_price")
	source := &rowMappingSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			quantity: {{State: validationeval.ValueNull}},
			price:    {{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
		},
	}
	got, err := validationeval.Decide(r, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{atMostRows(openIs("1"))},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("got=%+v, want undecided value-null", got)
	}
}
