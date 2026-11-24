package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// before #5910 the row-scoped machinery (ReadRelationNumbers, DecideRowScoped,
// the NumberRows field) existed and was tested, but no rule reached it: decide()
// went straight to the per-record path for every operator rule. An operator
// declaring `of each row` was refused at conversion, so nothing could tell the
// difference. This test goes through Decide, so a wiring that is present but
// unreachable is still a failure (#4359, decision section 15.2).
//
// The expected answers below are the section 15.2 table, written out per case.
func TestDecideReachesTheRowScopedOperatorPath(t *testing.T) {
	present := func(value int64) validationeval.NumberReading {
		return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(value, 1)}
	}
	above := func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
		// all_more_than_zero: every row that was read holds more than zero.
		for _, row := range rows["value"] {
			if row.State != validationeval.ValuePresent {
				return validationeval.TriMaybe, nil
			}
			if row.Value.Sign() <= 0 {
				return validationeval.TriNo, nil
			}
		}
		return validationeval.TriYes, nil
	}

	for _, tc := range []struct {
		name     string
		refutes  bool
		coverage validationeval.Coverage
		rows     []validationeval.NumberReading
		want     validationeval.Result
		reason   validationeval.UndecidedReason
	}{
		{
			name: "a counterexample read in a partial relation, refutes stated",
			// The operator says a witnessed counterexample is enough, so the
			// unread rows cannot undo it.
			refutes: true, coverage: validationeval.CoveragePartial,
			rows: []validationeval.NumberReading{present(1), present(-1)},
			want: validationeval.ResultNotSatisfied,
		},
		{
			name: "the same counterexample without refutes stated",
			// Nothing states that the unread rows cannot undo a seen
			// counterexample, so a partial read is undecided.
			coverage: validationeval.CoveragePartial,
			rows:     []validationeval.NumberReading{present(1), present(-1)},
			reason:   validationeval.UndecidedRelationPartiallyRead,
		},
		{
			name: "a counterexample read in the whole relation",
			// Every row was read, so the statement is irrelevant.
			coverage: validationeval.CoverageComplete,
			rows:     []validationeval.NumberReading{present(1), present(-1)},
			want:     validationeval.ResultNotSatisfied,
		},
		{
			name:    "no counterexample in a partial relation",
			refutes: true, coverage: validationeval.CoveragePartial,
			rows:   []validationeval.NumberReading{present(1), present(2)},
			reason: validationeval.UndecidedRelationPartiallyRead,
		},
		{
			name:     "no counterexample in the whole relation",
			coverage: validationeval.CoverageComplete,
			rows:     []validationeval.NumberReading{present(1), present(2)},
			want:     validationeval.ResultSatisfied,
		},
		{
			name: "a row that holds no value is reported before the coverage",
			// Reading more rows cannot fill in a value the row certainly
			// does not hold.
			refutes: true, coverage: validationeval.CoveragePartial,
			rows:   []validationeval.NumberReading{present(1), {State: validationeval.ValueNull}},
			reason: validationeval.UndecidedValueNull,
		},
		{
			name:    "a row that was not read is reported before the coverage",
			refutes: true, coverage: validationeval.CoveragePartial,
			rows:   []validationeval.NumberReading{{State: validationeval.ValueNotFetched}},
			reason: validationeval.UndecidedValueNotFetched,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := semantic.Operator{
				ID: "sales.more_than_zero", Symbol: "more_than_zero",
				Input:             semantic.OperatorInputNumber,
				InputName:         "value",
				RefutesOnSomeRows: tc.refutes,
				Inputs: []semantic.OperatorInputDeclaration{
					{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
				},
			}
			implementation := validationeval.OperatorImplementation{
				Operator:   "sales.more_than_zero",
				Input:      semantic.OperatorInputNumber,
				NumberRows: above,
			}
			rule := semantic.Validation{
				ID:        "sales.order_line.all_more_than_zero",
				Kind:      semantic.ValidationOperator,
				Attribute: "sales.order_line.amount",
				Relation:  "sales.order.lines",
				Operator:  "sales.more_than_zero",
			}
			decision, err := validationeval.Decide(rule, rowSource{
				coverage: tc.coverage, rows: tc.rows,
			}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if err != nil {
				t.Fatalf("the row-scoped rule was not decided: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q (undecided=%v)", decision.Result, tc.want, decision.Undecided)
			}
			if tc.reason != "" {
				if decision.Undecided == nil || decision.Undecided.Reason != tc.reason {
					t.Fatalf("undecided=%v, want reason %q", decision.Undecided, tc.reason)
				}
			} else if decision.Undecided != nil {
				t.Fatalf("undecided=%v, want a decided answer", decision.Undecided)
			}
		})
	}
}

// A per-record implementation must not be asked to decide a row-scoped
// contract: it cannot see that more than one row exists, so it would answer
// from one row and call that the relation's answer.
func TestARowScopedContractRefusesAPerRecordImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.more_than_zero", Symbol: "more_than_zero",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.more_than_zero",
		Input:    semantic.OperatorInputNumber,
		Number:   func(value *big.Rat) (bool, error) { return value.Sign() > 0, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.all_more_than_zero", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.amount", Relation: "sales.order.lines",
		Operator: "sales.more_than_zero",
	}
	_, err := validationeval.Decide(rule, rowSource{coverage: validationeval.CoverageComplete},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatal("a per-record implementation was accepted for a row-scoped contract")
	}
	if !containsText(err.Error(), "reads the rows of a relation") {
		t.Fatalf("the refusal does not say why: %v", err)
	}
}

// A row-scoped contract read from a reader that cannot read rows is refused,
// not answered from one record.
func TestARowScopedContractRefusesAReaderOfOneRecord(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.more_than_zero", Symbol: "more_than_zero",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.more_than_zero", Input: semantic.OperatorInputNumber,
		NumberRows: func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.all_more_than_zero", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.amount", Relation: "sales.order.lines",
		Operator: "sales.more_than_zero",
	}
	_, err := validationeval.Decide(rule, oneRecordOnlySource{},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil || !containsText(err.Error(), "which this reader cannot read") {
		t.Fatalf("a reader of one record was not refused: %v", err)
	}
}

// rowSource is a reader that can read one attribute over every row of a
// relation, and states which read it is.
type rowSource struct {
	coverage validationeval.Coverage
	rows     []validationeval.NumberReading
}

func (s rowSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s rowSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s rowSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s rowSource) ReadRelationNumbers(core.CanonicalID, core.CanonicalID) (validationeval.NumberRowsReading, error) {
	return validationeval.NumberRowsReading{Coverage: s.coverage, Rows: s.rows}, nil
}

// oneRecordOnlySource is a reader that cannot read a relation's rows.
type oneRecordOnlySource struct{}

func (oneRecordOnlySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}, nil
}

func (oneRecordOnlySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (oneRecordOnlySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func containsText(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
