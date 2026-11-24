package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The first input of an operator rule comes from the rule subject's attribute.
// Its declaration may state `absent as <name>`, but the subject has no place to
// write `or <name>` — the subject is the attribute, not an assignment. The
// evaluator therefore has to consult the declaration for the subject itself.
//
// It did not: the subject was read without looking at the declaration, so a
// contract that stated how to read an absent subject produced an undecided
// rule instead. The mapping was accepted at authoring time and never used
// (#4359).
func TestAnAbsentSubjectIsReadUnderTheDeclaredMapping(t *testing.T) {
	got, err := validationeval.Decide(subjectMappingRule, subjectStateSource{state: validationeval.ValueNull},
		subjectMappingOperators(nonNegativeReadingOpen()))
	if err != nil {
		t.Fatalf("the mapping could not be read: %v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q undecided=%v, want satisfied: the contract states how an absent subject is read",
			got.Result, got.Undecided)
	}
}

// A subject the reader did not fetch is not an absent value: nothing is known
// about it, so no mapping applies and the rule stays undecided. Reading it as
// the mapping would decide a rule from a value that was never asked for.
func TestAnUnreadSubjectIsNotReadUnderTheMapping(t *testing.T) {
	got, err := validationeval.Decide(subjectMappingRule, subjectStateSource{state: validationeval.ValueNotFetched},
		subjectMappingOperators(nonNegativeReadingOpen()))
	if err != nil {
		t.Fatalf("an unread subject must not be an error: %v", err)
	}
	if got.Result != "" || got.Undecided == nil {
		t.Fatalf("result=%q undecided=%v, want undecided", got.Result, got.Undecided)
	}
	if got.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("reason=%q, want %q", got.Undecided.Reason, validationeval.UndecidedValueNotFetched)
	}
}

// An implementation that cannot read the mapping leaves the rule undecided
// rather than reading the absence as zero: what the mapping names is the
// implementation's to state, and silence is not zero.
func TestAnAbsentSubjectWithoutAReadableMappingStaysUndecided(t *testing.T) {
	implementation := nonNegativeReadingOpen()
	implementation.Absent = nil
	got, err := validationeval.Decide(subjectMappingRule, subjectStateSource{state: validationeval.ValueNull},
		subjectMappingOperators(implementation))
	if err != nil {
		t.Fatalf("an unreadable mapping must not be an error: %v", err)
	}
	if got.Result != "" || got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("result=%q undecided=%v, want the value-null reason", got.Result, got.Undecided)
	}
}

// A subject that holds a value is unaffected: the mapping is about absence.
func TestAPresentSubjectIsDecidedWithoutTheMapping(t *testing.T) {
	got, err := validationeval.Decide(subjectMappingRule, subjectValueSource{value: big.NewRat(2, 1)},
		subjectMappingOperators(nonNegativeReadingOpen()))
	if err != nil {
		t.Fatalf("a present subject must not be an error: %v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied for a present positive value", got.Result)
	}
}

var subjectMappingContract = semantic.Operator{
	ID:        "sales.non_negative",
	Symbol:    "non_negative",
	Input:     semantic.OperatorInputNumber,
	InputName: "value",
	AbsentAs:  "open",
	Inputs: []semantic.OperatorInputDeclaration{
		{Name: "value", Type: semantic.OperatorInputNumber, AbsentAs: "open"},
	},
}

var subjectMappingRule = semantic.Validation{
	ID:        "sales.order_line.quantity_non_negative",
	Kind:      semantic.ValidationOperator,
	Attribute: "sales.order_line.quantity",
	Operator:  "sales.non_negative",
}

func subjectMappingOperators(implementation validationeval.OperatorImplementation) validationeval.Operators {
	return validationeval.Operators{
		Declared:        []semantic.Operator{subjectMappingContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}
}

// nonNegativeReadingOpen reads the mapping `open` as zero, which is satisfied.
func nonNegativeReadingOpen() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.non_negative",
		Input:    semantic.OperatorInputNumber,
		Number:   func(value *big.Rat) (bool, error) { return value.Sign() >= 0, nil },
		Absent: func(mapping string) (*big.Rat, bool) {
			if mapping == "open" {
				return new(big.Rat), true
			}
			return nil, false
		},
	}
}

// subjectStateSource reports one state for every attribute and states one
// attempt, which is what a reader that cannot separate attributes looks like.
type subjectStateSource struct {
	state validationeval.ValueState
}

func (s subjectStateSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: s.state}, nil
}

func (s subjectStateSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s subjectStateSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

// subjectValueSource reports one present value for every attribute.
type subjectValueSource struct {
	value *big.Rat
}

func (s subjectValueSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: s.value}, nil
}

func (s subjectValueSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s subjectValueSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }
