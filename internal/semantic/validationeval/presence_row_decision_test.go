package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input declared `of each row` asks whether every row of the relation
// holds a value. The answer is three-valued, and the read Coverage and the
// operator's refutation statement compose it in one place (row_scope.go).
//
// The expectations come from what the declaration says the input reads, not
// from the code.
func presenceRowOperators() validationeval.Operators {
	rule := semantic.Operator{
		ID: "sales.all_hold", Symbol: "all_hold",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.all_hold",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			for _, row := range rows["value"] {
				switch row.State {
				case validationeval.ValueNull:
					return validationeval.TriNo, nil
				case validationeval.ValueNotFetched:
					return validationeval.TriMaybe, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
	return validationeval.Operators{
		Declared:        []semantic.Operator{rule},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}
}

func presenceRowRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.all_codes_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code", Operator: "sales.all_hold",
		Relation: "sales.order.lines",
	}
}

// Every row holding a value satisfies the rule; a row that holds none refutes it.
func TestAPresenceRowRuleIsAnsweredPerRelation(t *testing.T) {
	operators := presenceRowOperators()
	rule := presenceRowRule()

	decision, err := validationeval.Decide(rule,
		presenceRowsSource{rows: []validationeval.PresenceReading{
			{State: validationeval.ValuePresent},
			{State: validationeval.ValuePresent},
		}},
		operators)
	if err != nil {
		t.Fatalf("a presence row rule was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", decision.Result)
	}

	decision, err = validationeval.Decide(rule,
		presenceRowsSource{rows: []validationeval.PresenceReading{
			{State: validationeval.ValuePresent},
			{State: validationeval.ValueNull},
		}},
		operators)
	if err != nil {
		t.Fatalf("a presence row rule was refused: %v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("result=%q, want not-satisfied", decision.Result)
	}
}

// A row nobody read leaves the rule undecided with that reason, and section 8.6
// settles it before an implementation is asked: the implementation never receives
// `ValueNotFetched`, because "nobody looked" is not an answer it could weigh.
func TestAPresenceRowRuleWithAnUnreadRowIsUndecided(t *testing.T) {
	operators := presenceRowOperators()
	asked := false
	implementation := operators.Implementations[0]
	implementation.PresenceRows = func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
		asked = true
		for _, row := range rows["value"] {
			if row.State == validationeval.ValueNotFetched {
				return validationeval.TriMaybe, nil
			}
		}
		return validationeval.TriYes, nil
	}
	operators.Implementations = []validationeval.OperatorImplementation{implementation}

	decision, err := validationeval.Decide(presenceRowRule(),
		presenceRowsSource{rows: []validationeval.PresenceReading{
			{State: validationeval.ValuePresent},
			{State: validationeval.ValueNotFetched},
		}},
		operators)
	if err != nil {
		t.Fatalf("a presence row rule was refused: %v", err)
	}
	if decision.Undecided == nil {
		t.Fatalf("an unread row was decided as %q", decision.Result)
	}
	if decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("reason=%q, want value-not-fetched", decision.Undecided.Reason)
	}
	if asked {
		t.Fatal("the implementation was asked about a row nobody read")
	}
}

// A reader that cannot read presence over rows is refused by name rather than
// answered from another kind of read.
func TestAPresenceRowRuleRefusesAReaderWithoutThatRead(t *testing.T) {
	if _, err := validationeval.Decide(presenceRowRule(), numbersOnlySource{}, presenceRowOperators()); err == nil {
		t.Fatal("a presence row rule was answered by a reader that reads numbers")
	}
}

// presenceRowsSource reports the presence column it was given and states one read
// attempt. A rule about one record is refused, so the same source cannot be used
// to answer the one-record question by accident.
type presenceRowsSource struct {
	rows []validationeval.PresenceReading
}

func (s presenceRowsSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, nil
}

func (s presenceRowsSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s presenceRowsSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s presenceRowsSource) ReadRelationPresence(relation, attribute core.CanonicalID) (validationeval.PresenceRowsReading, error) {
	return validationeval.PresenceRowsReading{
		Coverage: validationeval.CoverageComplete,
		Rows:     s.rows,
	}, nil
}

// numbersOnlySource reads numbers and states no presence over rows, so a rule
// that asks for that read is refused rather than answered from a number.
type numbersOnlySource struct{}

func (numbersOnlySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, nil
}

func (numbersOnlySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (numbersOnlySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }
