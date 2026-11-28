package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input read over every row of a relation asks, for each row,
// whether that row holds a value (decision section 8.6). The expectations
// below are written from that question, not read back from the reader.
//
//	rows:            present, absent, present
//	implementation:  answers TriNo when any row holds none   -> not-satisfied
//	                 answers TriYes otherwise                -> satisfied
//
// A row nobody read is not an answer: the implementation is not asked, and the
// rule is undecided for that row's own reason.

// presenceRowSource states one reading per row.
type presenceRowSource struct {
	coverage validationeval.Coverage
	rows     []validationeval.PresenceReading
}

func (s presenceRowSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (presenceRowSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s presenceRowSource) ReadRelationPresence(_, _ core.CanonicalID) (validationeval.PresenceRowsReading, error) {
	return validationeval.PresenceRowsReading{Coverage: s.coverage, Rows: s.rows}, nil
}

func presenceRowContract() (semantic.Operator, semantic.Validation, validationeval.OperatorImplementation, *bool) {
	contract := semantic.Operator{
		ID: "sales.every_price_stated", Symbol: "every_price_stated",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.every_price_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.unit_price", Operator: "sales.every_price_stated",
		Relation: "sales.order.lines",
	}
	asked := false
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.every_price_stated",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			asked = true
			for _, row := range rows["value"] {
				if row.State == validationeval.ValueNull {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
	return contract, rule, implementation, &asked
}

func presenceRows(states ...validationeval.ValueState) []validationeval.PresenceReading {
	rows := make([]validationeval.PresenceReading, 0, len(states))
	for _, state := range states {
		rows = append(rows, validationeval.PresenceReading{State: state})
	}
	return rows
}

// Every row holding a value satisfies the rule; one row holding none refutes it.
func TestAPresenceRowRuleIsDecidedFromTheRowsItRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rows  []validationeval.PresenceReading
		want  validationeval.Result
		asked bool
	}{
		{"every row holds a value", presenceRows(validationeval.ValuePresent, validationeval.ValuePresent), validationeval.ResultSatisfied, true},
		{"one row holds none", presenceRows(validationeval.ValuePresent, validationeval.ValueNull), validationeval.ResultNotSatisfied, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract, rule, implementation, asked := presenceRowContract()
			decision, err := validationeval.Decide(rule, presenceRowSource{coverage: validationeval.CoverageComplete, rows: tc.rows}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if err != nil {
				t.Fatalf("the rule was refused: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q", decision.Result, tc.want)
			}
			if *asked != tc.asked {
				t.Fatalf("implementation asked=%v, want %v", *asked, tc.asked)
			}
		})
	}
}

// A row nobody read is not the answer "holds none": the implementation is not
// asked, and the rule is undecided with that row's own reason.
func TestAPresenceRowRuleWithAnUnreadRowIsUndecided(t *testing.T) {
	contract, rule, implementation, asked := presenceRowContract()
	decision, err := validationeval.Decide(rule, presenceRowSource{
		coverage: validationeval.CoverageComplete,
		rows:     presenceRows(validationeval.ValuePresent, validationeval.ValueNotFetched),
	}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("result=%q, want no result", decision.Result)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("undecided=%+v, want value-not-fetched", decision.Undecided)
	}
	if *asked {
		t.Fatal("the implementation was asked although a row was never read")
	}
}

// A partly read relation with no counterexample is not "satisfied": no rows
// refute the rule, but the rows that were not returned might.
func TestAPresenceRowRuleOverAPartlyReadRelationIsUndecided(t *testing.T) {
	contract, rule, implementation, _ := presenceRowContract()
	decision, err := validationeval.Decide(rule, presenceRowSource{
		coverage: validationeval.CoveragePartial,
		rows:     presenceRows(validationeval.ValuePresent),
	}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("result=%q, want no result", decision.Result)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("undecided=%+v, want relation-partially-read", decision.Undecided)
	}
}

// An implementation that decides values cannot answer a presence column, and
// the shape check says so before any rule is read.
func TestAPresenceColumnNeedsARowPresenceImplementation(t *testing.T) {
	contract, _, _, _ := presenceRowContract()
	valueRows := validationeval.OperatorImplementation{
		Operator: "sales.every_price_stated",
		Input:    semantic.OperatorInputPresence,
		ValueRows: func(map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, valueRows); err == nil {
		t.Fatal("a value-rows implementation was accepted for a presence column")
	}
}

// A reader that cannot read a relation's presence refuses the rule rather than
// answering it row by row.
func TestAPresenceColumnNeedsAReaderThatReadsTheRelation(t *testing.T) {
	contract, rule, implementation, _ := presenceRowContract()
	_, err := validationeval.Decide(rule, refusingTypeSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a reader without a relation-presence read decided the rule")
	}
}
