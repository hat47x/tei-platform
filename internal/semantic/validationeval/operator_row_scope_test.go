package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator declares `of each row` is asked of the rows of the
// relation through the same Decide entry as every other rule (decision section
// 15.4). Halfway is refused: answering such a rule row by row would look
// decided while the declaration says the rows are one collection.
//
// The expectations are written from that statement, not from the code.

// rowsSource answers one attribute over every row.
type rowsSource struct {
	coverage validationeval.Coverage
	calls    map[core.CanonicalID][]validationeval.NumberReading
	asked    []core.CanonicalID
}

func (s *rowsSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, nil
}

func (s *rowsSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *rowsSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	s.asked = append(s.asked, attribute)
	return validationeval.NumberRowsReading{Coverage: s.coverage, Rows: s.calls[attribute]}, nil
}

func atMostImplementation() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.at_most",
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			for _, row := range rows["value"] {
				if row.State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if row.Value.Cmp(big.NewRat(100, 1)) > 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func rowsRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.all_within", Symbol: "all_within",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity",
		Relation:  "sales.order.lines",
		Operator:  "sales.at_most",
	}
}

func rowsContract(refutes bool) semantic.Operator {
	return semantic.Operator{
		ID:    "sales.at_most",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: refutes,
	}
}

func present(values ...string) []validationeval.NumberReading {
	out := make([]validationeval.NumberReading, 0, len(values))
	for _, text := range values {
		value, ok := new(big.Rat).SetString(text)
		if !ok {
			panic("not a number: " + text)
		}
		out = append(out, validationeval.NumberReading{State: validationeval.ValuePresent, Value: value})
	}
	return out
}

func decideRows(t *testing.T, coverage validationeval.Coverage, refutes bool, rows []validationeval.NumberReading) validationeval.Decision {
	t.Helper()
	attribute, err := core.ParseCanonicalID("sales.order_line.quantity")
	if err != nil {
		t.Fatal(err)
	}
	source := &rowsSource{coverage: coverage, calls: map[core.CanonicalID][]validationeval.NumberReading{attribute: rows}}
	got, err := validationeval.Decide(rowsRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{rowsContract(refutes)},
		Implementations: []validationeval.OperatorImplementation{atMostImplementation()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// The rule reads the relation once, never row by row.
	if len(source.asked) != 1 {
		t.Fatalf("the relation was read %d times, want once: %v", len(source.asked), source.asked)
	}
	return got
}

// Every row is at most 100 and every row was read: satisfied.
func TestRowScopedRuleAllRowsSatisfy(t *testing.T) {
	got := decideRows(t, validationeval.CoverageComplete, false, present("2", "5"))
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("got=%+v, want satisfied", got)
	}
}

// A counterexample with the refutation statement settles it as not-satisfied
// even though rows were not all read.
func TestRowScopedRuleRefutesOnAWitnessedRow(t *testing.T) {
	got := decideRows(t, validationeval.CoveragePartial, true, present("2", "500"))
	if got.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("got=%+v, want not-satisfied", got)
	}
}

// The same rows without the statement: the operator may be about the
// collection, so a seen counterexample does not settle the rule.
func TestRowScopedRuleWithoutRefutationStaysUndecided(t *testing.T) {
	got := decideRows(t, validationeval.CoveragePartial, false, present("2", "500"))
	if got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("got=%+v, want undecided relation-partially-read", got)
	}
}

// Every row satisfies the rule but rows were not all read.
func TestRowScopedRulePartiallyReadStaysUndecided(t *testing.T) {
	got := decideRows(t, validationeval.CoveragePartial, false, present("2"))
	if got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("got=%+v, want undecided relation-partially-read", got)
	}
}

// A reader that cannot read the rows of a relation is refused rather than
// answered row by row.
func TestRowScopedRuleNeedsAReaderThatReadsTheRelation(t *testing.T) {
	attribute, err := core.ParseCanonicalID("sales.order_line.quantity")
	if err != nil {
		t.Fatal(err)
	}
	source := &attemptingSource{values: map[string]*big.Rat{string(attribute): big.NewRat(2, 1)}}
	_, err = validationeval.Decide(rowsRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{rowsContract(true)},
		Implementations: []validationeval.OperatorImplementation{atMostImplementation()},
	})
	if err == nil {
		t.Fatal("a reader that cannot read the relation answered a row-scoped rule")
	}
}
