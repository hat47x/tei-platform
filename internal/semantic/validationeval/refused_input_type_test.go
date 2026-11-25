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
// the value carrier of section 8.4, so this test pins only the input whose
// question is different: presence asks whether a value is there, not what it
// is, and the numeric read would answer the second question by accident.
//
// The refusal used to name `presence` whatever the type was, because the message
// was written for that type and then generalised without changing the wording.
// Measured: a `date` contract said "the presence input has no reader on this
// path yet", which sends the author looking at the wrong declaration. The
// wording now names its own type, which this pins for the type it still refuses.
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
		// The reason names this type: "the <type> input has no reader".
		// A message written for one type and reused for others names the
		// wrong one, which is what this pins.
		if !strings.Contains(err.Error(), "the "+string(kind)+" input has no reader") {
			t.Fatalf("the refusal does not give %q as the reason: %v", kind, err)
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

// The four typed words are read by the example table, but not yet by the
// reader that decides a rule: it reads an attribute as a number, so a rule that
// declares one is refused at the rule rather than answered from a number it was
// never read as. The value the example asks about is checked separately in
// operator_value_test.go, where a typed implementation receives it.
//
// Section 8.4 names the order: the value carrier and the typed decision field
// come first, and the reader for an attribute of another type follows. This
// test pins the boundary the reader has reached, so a later change can move it
// by changing one guard rather than by finding out what it used to be.
func TestATypedInputIsRefusedUntilTheReaderReadsIt(t *testing.T) {
	for _, kind := range []semantic.OperatorInput{
		semantic.OperatorInputDate,
		semantic.OperatorInputText,
		semantic.OperatorInputTruth,
		semantic.OperatorInputAny,
	} {
		t.Run(string(kind), func(t *testing.T) {
			// The example states a value of the declared type, written the way
			// section 8.3 says it is written.
			example := semantic.OperatorCase{Value: "2026-04-01", Satisfied: true}
			switch kind {
			case semantic.OperatorInputText:
				example = semantic.OperatorCase{Value: `"A-100"`, Satisfied: true}
			case semantic.OperatorInputTruth:
				example = semantic.OperatorCase{Value: "true", Satisfied: true}
			case semantic.OperatorInputAny:
				example = semantic.OperatorCase{Value: "7", Satisfied: true}
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
