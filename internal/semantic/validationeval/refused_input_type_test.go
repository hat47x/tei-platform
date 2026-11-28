package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The typed words (date, text, truth, any) are read as their own type by the
// value carrier of section 8.4, and presence now has its own reader and its own
// decision function (presence_operator.go). What this file pins is the boundary
// that is still a flag: a reader that can read only numbers refuses the typed
// inputs it is asked for, naming the read it lacks rather than answering from
// the number (#4359).
//
// Section 8.4 names the order: the value carrier and the typed decision field
// come first, and the readers follow one at a time. A later change moves this
// boundary by changing one guard, not by finding out what it used to be.

// TestATypedInputIsRefusedUntilTheReaderReadsIt pins that a reader which reads
// numbers refuses a text or a truth question by naming the read it lacks.
func TestATypedInputIsRefusedUntilTheReaderReadsIt(t *testing.T) {
	for _, kind := range []semantic.OperatorInput{
		semantic.OperatorInputText,
		semantic.OperatorInputTruth,
	} {
		t.Run(string(kind), func(t *testing.T) {
			example := semantic.OperatorCase{Value: `"A-100"`, Satisfied: true}
			if kind == semantic.OperatorInputTruth {
				example = semantic.OperatorCase{Value: "true", Satisfied: true}
			}
			contract := semantic.Operator{
				ID: "sales.op", Symbol: "op", Input: kind,
				Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: kind}},
				Cases:  []semantic.OperatorCase{example},
			}
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op",
				Input:    kind,
				Values: func(map[string]semantic.OperatorValue) (bool, error) {
					return true, nil
				},
			}
			rule := semantic.Validation{
				ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.quantity", Operator: "sales.op",
			}
			_, err := validationeval.Decide(rule, refusingTypeSource{},
				validationeval.Operators{
					Declared:        []semantic.Operator{contract},
					Implementations: []validationeval.OperatorImplementation{implementation},
				})
			if err == nil {
				t.Fatalf("a %s input was decided by a reader that reads numbers", kind)
			}
			if !strings.Contains(err.Error(), "cannot read "+string(kind)) {
				t.Fatalf("the refusal does not name the read it lacks: %v", err)
			}
		})
	}
	// A reader that reads only numbers is asked for a value of any type and
	// says which read it lacks, rather than answering from the number.
	t.Run("any", func(t *testing.T) {
		kind := semantic.OperatorInputAny
		contract := semantic.Operator{
			ID: "sales.op", Symbol: "op", Input: kind,
			Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: kind}},
			Cases:  []semantic.OperatorCase{{Value: "7", Satisfied: true}},
		}
		implementation := validationeval.OperatorImplementation{
			Operator: "sales.op",
			Input:    kind,
			Values: func(map[string]semantic.OperatorValue) (bool, error) {
				return true, nil
			},
		}
		rule := semantic.Validation{
			ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
			Attribute: "sales.order_line.quantity", Operator: "sales.op",
		}
		_, err := validationeval.Decide(rule, refusingTypeSource{},
			validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
		if err == nil {
			t.Fatalf("a %s input was decided by a reader that reads numbers", kind)
		}
		if !strings.Contains(err.Error(), "cannot read a value of "+string(kind)+" type") {
			t.Fatalf("the refusal does not name the read it lacks: %v", err)
		}
	})
	// A reader that reads only numbers is asked whether a record holds a value
	// and says so, rather than answering from the number.
	t.Run("presence", func(t *testing.T) {
		kind := semantic.OperatorInputPresence
		contract := semantic.Operator{
			ID: "sales.op", Symbol: "op", Input: kind,
			Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: kind}},
		}
		implementation := validationeval.OperatorImplementation{
			Operator: "sales.op",
			Input:    kind,
			Presence: func(map[string]bool) (bool, error) { return true, nil },
		}
		rule := semantic.Validation{
			ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
			Attribute: "sales.order_line.quantity", Operator: "sales.op",
		}
		_, err := validationeval.Decide(rule, refusingTypeSource{},
			validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
		if err == nil {
			t.Fatalf("a %s input was decided by a reader that reads numbers", kind)
		}
		// The refusal names the question this reader cannot answer, so the
		// author knows it is the reader and not the declaration.
		if !strings.Contains(err.Error(), "whether an attribute holds a value") {
			t.Fatalf("the refusal does not name the question it cannot answer: %v", err)
		}
	})
}

// A numeric contract is still decided by the operator path, so these refusals
// narrow the inputs they cannot read and do not remove the path.
func TestANumericContractIsStillDecidedByTheOperatorPath(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputNumber}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.non_negative",
		Input:    semantic.OperatorInputNumber,
		Number:   func(value *big.Rat) (bool, error) { return value.Sign() >= 0, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_non_negative", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.non_negative",
	}
	decision, err := validationeval.Decide(rule, refusingTypeSource{},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err != nil {
		t.Fatalf("a numeric operator was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied for the number this reader states", decision.Result)
	}
}

// refusingTypeSource reads numbers and nothing else: every typed read it is
// asked for is refused, which is the reader the boundary is about.
type refusingTypeSource struct{}

func (refusingTypeSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(7, 1)}, nil
}

func (refusingTypeSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (refusingTypeSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (refusingTypeSource) ReadText(core.CanonicalID) (validationeval.TextReading, error) {
	return validationeval.TextReading{}, errNoRead("text")
}

func (refusingTypeSource) ReadTruth(core.CanonicalID) (validationeval.TruthReading, error) {
	return validationeval.TruthReading{}, errNoRead("truth")
}

func (refusingTypeSource) ReadAny(core.CanonicalID) (validationeval.AnyReading, error) {
	return validationeval.AnyReading{}, errNoRead("a value of any type")
}

func (refusingTypeSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{}, errNoRead("whether an attribute holds a value")
}

// errNoRead is the error of a reader that does not hold the read it was asked
// for. It names the read, because a message that names another read sends the
// author looking at the wrong code.
type errNoRead string

func (e errNoRead) Error() string {
	return "this reader reads numbers and cannot read " + string(e)
}
