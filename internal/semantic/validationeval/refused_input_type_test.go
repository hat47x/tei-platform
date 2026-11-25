package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The evaluator's one operator path reads inputs as numbers. A contract that
// declares any other type is refused before the operator is called, so a rule
// cannot be answered from a value of the wrong kind (#4359).
//
// The refusal used to name `presence` whatever the type was, because the message
// was written for that type and then generalised without changing the wording.
// Measured: a `date` contract said "the presence input has no reader on this
// path yet", which sends the author looking at the wrong declaration.
func TestARefusedInputNamesItsOwnType(t *testing.T) {
	for _, kind := range []semantic.OperatorInput{
		semantic.OperatorInput("date"),
		semantic.OperatorInput("text"),
		semantic.OperatorInput("truth"),
		semantic.OperatorInput("any"),
		semantic.OperatorInputPresence,
	} {
		t.Run(string(kind), func(t *testing.T) {
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
				t.Fatalf("a %s input was accepted by a path that reads numbers", kind)
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
				"date", "text", "truth", "any", semantic.OperatorInputPresence,
			} {
				if other == kind {
					continue
				}
				if strings.Contains(err.Error(), " "+string(other)+" input has no reader") {
					t.Fatalf("the refusal for %s blames the %s input: %v", kind, other, err)
				}
			}
		})
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
