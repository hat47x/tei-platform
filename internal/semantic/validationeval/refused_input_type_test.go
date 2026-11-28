package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The evaluator's one operator path reads inputs as values. A contract that
// declares a type this program cannot read is refused before the operator is
// called, so a rule cannot be answered from a value of the wrong kind (#4359).
//
// The four typed words (date, text, truth, any) are read as their own type by
// the value carrier of section 8.4, and presence is read as whether a value is
// there. This test pins the refusal that remains: a numeric implementation
// answers a value, which is a different question than presence asks, so the
// rule is refused and the refusal names both the type and the input.
//
// The refusal used to say the presence input had no reader at all. It now says
// what is actually wrong: the selected implementation decides a value.
func TestARefusedInputNamesItsOwnType(t *testing.T) {
	t.Run("presence", func(t *testing.T) {
		kind := semantic.OperatorInputPresence
		contract := semantic.Operator{
			ID: "sales.op", Symbol: "op", Input: kind,
			Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: kind}},
		}
		implementation := validationeval.OperatorImplementation{
			Operator: "sales.op",
			Input:    kind,
			Number:   func(*big.Rat) (bool, error) { return true, nil },
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
			t.Fatalf("a %s input was accepted by a path that reads values", kind)
		}
		// The message names this type and this input, so the author does
		// not have to guess which declaration is unreadable.
		if !strings.Contains(err.Error(), string(kind)) {
			t.Fatalf("the refusal does not name the type %q: %v", kind, err)
		}
		if !strings.Contains(err.Error(), `"value"`) {
			t.Fatalf("the refusal does not name the input: %v", err)
		}
		// The reason says what is wrong with the implementation rather than
		// blaming a missing reader: presence has a reader now, and the
		// implementation is the part that cannot answer it.
		if !strings.Contains(err.Error(), "its selected implementation decides a value") {
			t.Fatalf("the refusal does not give the reason: %v", err)
		}
		// And it does not blame a different type.
		for _, other := range []semantic.OperatorInput{
			"date", "text", "truth", "any",
		} {
			if strings.Contains(err.Error(), " "+string(other)+" input has no reader") {
				t.Fatalf("the refusal for %s blames the %s input: %v", kind, other, err)
			}
		}
	})
}

// The typed words are read by the example table, and the reader that decides a
// rule reads a text and a truth as their own type. The two that have no reader
// on this path yet are refused at the rule rather than answered from a number
// they were never read as, and a reader that cannot read the type it is asked
// for is refused with a message that says which read it lacks.
//
// Section 8.4 names the order: the value carrier and the typed decision field
// come first, and the readers follow one at a time. This test pins the boundary
// the reader has reached, so a later change moves it by changing one guard
// rather than by finding out what it used to be.
//
// A calendar day is read as a day on this path now, so it is not in this list.
// Neither is presence: an example that states a value for a presence input is
// refused by the example check, which answers before the reader path is
// reached, and a presence input without an example is pinned by
// TestARefusedInputNamesItsOwnType. What stays here are the words that still
// have no reader on this path.
func TestATypedInputIsRefusedUntilTheReaderReadsIt(t *testing.T) {
	// A reader that reads only numbers is asked for a text and a truth and
	// says which read it lacks, rather than answering from the number.
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
	for _, kind := range []semantic.OperatorInput{
		semantic.OperatorInputAny,
	} {
		t.Run(string(kind), func(t *testing.T) {
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
	}
	// A type with no reader on this path is refused by name. It is not answered
	// from the empty value: a subject nobody read is not the zero of its type.
	for _, kind := range []semantic.OperatorInput{} {
		t.Run(string(kind), func(t *testing.T) {
			example := semantic.OperatorCase{Value: "7", Satisfied: true}
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
			if !strings.Contains(err.Error(), "the "+string(kind)+" input has no reader") {
				t.Fatalf("the refusal does not give %q as the reason: %v", kind, err)
			}
		})
	}
}

// An implementation that reads values of any type is asked about the values as
// the example states them, so a text is a text and a number is a number
// (section 8.3, 8.4). The value it receives carries the type, and an
// implementation that answers differently from the example is refused.
func TestATypedImplementationIsAskedAboutTheExamplesItIsGiven(t *testing.T) {
	for _, tc := range []struct {
		input semantic.OperatorInput
		cell  string
	}{
		{semantic.OperatorInputDate, "2026-04-01"},
		{semantic.OperatorInputText, `"A-100"`},
		{semantic.OperatorInputTruth, "true"},
		{semantic.OperatorInputAny, "7"},
	} {
		t.Run(string(tc.input)+" "+tc.cell, func(t *testing.T) {
			contract := semantic.Operator{
				ID: "sales.op", Symbol: "op", Input: tc.input,
				Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: tc.input}},
				Cases:  []semantic.OperatorCase{{Value: tc.cell, Satisfied: true}},
			}
			seen := make(chan semantic.OperatorValue, 1)
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op",
				Input:    tc.input,
				Values: func(values map[string]semantic.OperatorValue) (bool, error) {
					seen <- values["value"]
					return true, nil
				},
			}
			// The check is made before any rule is read, so this calls the
			// same function validate does: a Workspace whose selection the
			// binary cannot honour is refused before real data is read.
			err := validationeval.CheckImplementationMatchesContract(contract, implementation)
			if err != nil {
				t.Fatalf("the implementation did not match the contract: %v", err)
			}
			select {
			case got := <-seen:
				if got.Kind() != tc.input {
					t.Fatalf("the implementation received %s, want %s", got.Kind(), tc.input)
				}
			default:
				t.Fatalf("the implementation was not asked about the %s input", tc.input)
			}
		})
	}
}

// An implementation that provides both a typed value and a numeric one is
// refused: which value it answers from would be ambiguous (section 8.4).
func TestAnImplementationProvidesOneKindOfValue(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputNumber}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op",
		Input:    semantic.OperatorInputNumber,
		Number:   func(*big.Rat) (bool, error) { return true, nil },
		Values: func(map[string]semantic.OperatorValue) (bool, error) {
			return true, nil
		},
	}
	err := validationeval.CheckImplementationMatchesContract(contract, implementation)
	if err == nil {
		t.Fatal("an implementation provided a typed value and a numeric one")
	}
	if !strings.Contains(err.Error(), "more than one kind of decision function") {
		t.Fatalf("the refusal does not name what is wrong: %v", err)
	}
}

// A numeric contract is still decided: this refuses the types it cannot read,
// not operator rules.
func TestANumericContractIsStillDecidedByTheOperatorPath(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber},
		},
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
		t.Fatalf("a numeric contract was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied for 7", decision.Result)
	}
}

// refusingTypeSource reports one present number for every attribute.
type refusingTypeSource struct{}

func (refusingTypeSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(7, 1)}, nil
}

func (refusingTypeSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (refusingTypeSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }
