package validationeval_test

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The expected values below are written by hand from the contract the design
// fixes for within(value, low, high): the answer is low <= value <= high, the
// first input comes from the rule's subject, and every input after the first
// is assigned by name. They are not read back from the implementation.

// withinContract is the signature the design states, with the input names the
// rules and the examples refer to.
var withinContract = semantic.Operator{
	ID:     "sales.within",
	Symbol: "within",
	Input:  semantic.OperatorInputNumber,
	Inputs: []semantic.OperatorInputDeclaration{
		{Name: "value", Type: semantic.OperatorInputNumber},
		{Name: "low", Type: semantic.OperatorInputNumber},
		{Name: "high", Type: semantic.OperatorInputNumber},
	},
}

// withinImplementation is a plain reading of the same contract, written out
// here so a wrong evaluator is caught even when the built-in is wrong too.
var withinImplementation = validationeval.OperatorImplementation{
	Operator: "sales.within",
	Input:    semantic.OperatorInputNumber,
	Numbers: func(values map[string]*big.Rat) (bool, error) {
		value, low, high := values["value"], values["low"], values["high"]
		if value == nil || low == nil || high == nil {
			return false, nil
		}
		if low.Cmp(high) > 0 {
			return false, fmt.Errorf("within: low (%s) is greater than high (%s); the range specification is wrong", low.RatString(), high.RatString())
		}
		return value.Cmp(low) >= 0 && value.Cmp(high) <= 0, nil
	},
}

// discountInRange assigns only the inputs after the first: the subject
// supplies "value".
var discountInRange = semantic.Validation{
	ID:        "sales.order_line.discount_in_range",
	Symbol:    "discount_in_range",
	Kind:      semantic.ValidationOperator,
	Attribute: "sales.order_line.discount",
	Operator:  "sales.within",
	Inputs: []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "100"},
	},
}

var errTestReaderFailure = errors.New("the reader failed")

func TestMultiInputOperatorDecidesFromDeclaredSignature(t *testing.T) {
	cases := []struct {
		discount string
		want     validationeval.Result
	}{
		// B-05: both bounds are included.
		{"0", validationeval.ResultSatisfied},
		{"100", validationeval.ResultSatisfied},
		{"50", validationeval.ResultSatisfied},
		{"-1", validationeval.ResultNotSatisfied},
		{"101", validationeval.ResultNotSatisfied},
		// A value far below the range is the same answer, not another kind.
		{"-7", validationeval.ResultNotSatisfied},
	}
	for _, tc := range cases {
		source := &operatorSource{value: mustParseRat(t, tc.discount)}
		got, err := validationeval.EvaluateWithOperators(discountInRange, source, validationeval.Operators{
			Declared:        []semantic.Operator{withinContract},
			Implementations: []validationeval.OperatorImplementation{withinImplementation},
		})
		if err != nil {
			t.Fatalf("%s: err=%v", tc.discount, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got=%q want=%q", tc.discount, got, tc.want)
		}
	}
}

// TestMultiInputOperatorInputOrderDoesNotChangeTheAnswer is B-02: the ruling
// matches low and high by name, so writing them in the other order must not
// change the value they receive or the answer.
func TestMultiInputOperatorInputOrderDoesNotChangeTheAnswer(t *testing.T) {
	reversed := discountInRange
	reversed.Inputs = []semantic.ValidationInputAssignment{
		{Name: "high", Value: "100"},
		{Name: "low", Value: "0"},
	}
	want := map[string]validationeval.Result{
		"0":   validationeval.ResultSatisfied,
		"100": validationeval.ResultSatisfied,
		"-1":  validationeval.ResultNotSatisfied,
		"101": validationeval.ResultNotSatisfied,
	}
	for value, expected := range want {
		for _, rule := range []semantic.Validation{discountInRange, reversed} {
			source := &operatorSource{value: mustParseRat(t, value)}
			got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
				Declared:        []semantic.Operator{withinContract},
				Implementations: []validationeval.OperatorImplementation{withinImplementation},
			})
			if err != nil {
				t.Fatalf("value %s: err=%v", value, err)
			}
			if got != expected {
				t.Fatalf("value %s: got=%q want=%q", value, got, expected)
			}
		}
	}
}

// TestMultiInputOperatorLowGreaterThanHighIsAnError is B-06 with a constant
// range: a lower bound above the upper bound is a wrong range specification,
// not an out-of-range value. The bounds are not swapped and the answer is not
// not-satisfied.
func TestMultiInputOperatorLowGreaterThanHighIsAnError(t *testing.T) {
	rule := semantic.Validation{
		ID:        "sales.order_line.discount_in_range_bad",
		Symbol:    "discount_in_range_bad",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.discount",
		Operator:  "sales.within",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "low", Value: "100"},
			{Name: "high", Value: "0"},
		},
	}
	source := &operatorSource{value: mustParseRat(t, "50")}
	got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatalf("a wrong range was answered %q; it is not a decision", got)
	}
	if got != "" {
		t.Fatalf("a wrong range returned both an error and the result %q", got)
	}
}

// TestMultiInputOperatorReadsOnlyWhatItAssigns verifies that a constant
// assignment is parsed rather than read, and that the subject's attribute is
// read once.
func TestMultiInputOperatorReadsOnlyWhatItAssigns(t *testing.T) {
	source := &operatorSource{value: mustParseRat(t, "50")}
	got, err := validationeval.EvaluateWithOperators(discountInRange, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != validationeval.ResultSatisfied {
		t.Fatalf("got=%q want=satisfied", got)
	}
	if len(source.calls) != 1 || source.calls[0] != "Number(sales.order_line.discount)" {
		t.Fatalf("reads=%v, want only the subject attribute", source.calls)
	}
}

// TestMultiInputOperatorAttributeAssignmentIsRead verifies that an input may
// be assigned to another attribute and that the reader is asked for it.
func TestMultiInputOperatorAttributeAssignmentIsRead(t *testing.T) {
	rule := discountInRange
	rule.ID = "sales.order_line.discount_in_range_of_unit_price"
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "sales.order_line.unit_price"},
	}
	source := &readByAttributeSource{values: map[string]*big.Rat{
		"sales.order_line.discount":   mustParseRat(t, "50"),
		"sales.order_line.unit_price": mustParseRat(t, "100"),
	}}
	got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != validationeval.ResultSatisfied {
		t.Fatalf("got=%q want=satisfied", got)
	}
	if len(source.calls) != 2 {
		t.Fatalf("reads=%v, want the subject and the assigned attribute", source.calls)
	}
}

// TestMultiInputOperatorSubjectIsNotAssignedTwice verifies B-04 in part: the
// subject supplies the first input, so a rule that also assigns it is refused
// instead of binding it twice.
func TestMultiInputOperatorSubjectIsNotAssignedTwice(t *testing.T) {
	rule := discountInRange
	rule.ID = "sales.order_line.discount_in_range_twice"
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "value", Value: "sales.order_line.discount"},
		{Name: "low", Value: "0"},
		{Name: "high", Value: "100"},
	}
	source := &operatorSource{value: mustParseRat(t, "50")}
	got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatalf("a twice-bound subject was answered %q", got)
	}
}

// TestMultiInputOperatorUnassignedInputIsRefused verifies B-04 in part: an
// input after the first that the rule does not assign is refused, not read as
// the subject's value, as zero, or as absent.
func TestMultiInputOperatorUnassignedInputIsRefused(t *testing.T) {
	rule := discountInRange
	rule.ID = "sales.order_line.discount_in_range_low_only"
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "low", Value: "0"}}
	source := &operatorSource{value: mustParseRat(t, "50")}
	got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatalf("an unassigned input was answered %q", got)
	}
}

// TestMultiInputOperatorNullValueIsUndecided verifies B-07 in part: a value
// the reader reports as absent is undecided, not not-satisfied, and the
// implementation is not asked.
func TestMultiInputOperatorNullValueIsUndecided(t *testing.T) {
	asked := false
	source := &statefulSource{number: validationeval.NumberReading{State: validationeval.ValueNull}}
	decision, err := validationeval.Decide(discountInRange, source, validationeval.Operators{
		Declared: []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{{
			Operator: "sales.within",
			Input:    semantic.OperatorInputNumber,
			Numbers: func(values map[string]*big.Rat) (bool, error) {
				asked = true
				return false, nil
			},
		}},
	})
	if err != nil {
		t.Fatalf("a null value returned an error: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("a null value was decided as %q", decision.Result)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("undecided=%+v, want the reason that no value is held", decision.Undecided)
	}
	if asked {
		t.Fatal("the implementation was asked about a value the reader reported as absent")
	}
}

// TestMultiInputOperatorReaderErrorIsNotUndecided verifies that a reader
// failure is returned as the error it is and is not turned into "no data".
func TestMultiInputOperatorReaderErrorIsNotUndecided(t *testing.T) {
	source := &statefulSource{err: errTestReaderFailure}
	decision, err := validationeval.Decide(discountInRange, source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatalf("a reader failure was answered %+v", decision)
	}
	if decision.Undecided != nil {
		t.Fatalf("a reader failure was reported as undecided: %+v", decision.Undecided)
	}
}

// TestMultiInputOperatorExamplesMustMatchTheSignature verifies that an
// example, which carries one value, is refused for a multi-input contract
// instead of being read as if it bound the first input alone.
func TestMultiInputOperatorExamplesMustMatchTheSignature(t *testing.T) {
	contract := withinContract
	contract.Cases = []semantic.OperatorCase{{Value: "50", Satisfied: true}}
	source := &operatorSource{value: mustParseRat(t, "50")}
	got, err := validationeval.EvaluateWithOperators(discountInRange, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatalf("an example that states one value was accepted for three inputs and answered %q", got)
	}
}

// TestBuiltinWithinDecidesCorrectly verifies the shipped implementation
// against hand-written expected values, so a change to the built-in is caught
// here and not only through the back end that selects it.
func TestBuiltinWithinDecidesCorrectly(t *testing.T) {
	cases := []struct {
		value string
		want  validationeval.Result
	}{
		{"0", validationeval.ResultSatisfied},
		{"100", validationeval.ResultSatisfied},
		{"-1", validationeval.ResultNotSatisfied},
		{"101", validationeval.ResultNotSatisfied},
	}
	implementation := withinBuiltinImplementation(t)
	// The shipped implementation is named tei.number.within, so the contract
	// it is checked against keeps that identity while stating the same
	// signature the design fixes.
	contract := withinContract
	contract.ID = "tei.number.within"
	rule := discountInRange
	rule.Operator = "tei.number.within"
	for _, tc := range cases {
		source := &operatorSource{value: mustParseRat(t, tc.value)}
		got, err := validationeval.EvaluateWithOperators(rule, source, validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
		if err != nil {
			t.Fatalf("%s: err=%v", tc.value, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got=%q want=%q", tc.value, got, tc.want)
		}
	}
}

// withinBuiltinImplementation returns the built-in tei.number.within
// implementation with the contract it is written for, so a test can use the
// shipped implementation instead of a copy of it.
func withinBuiltinImplementation(t *testing.T) validationeval.OperatorImplementation {
	t.Helper()
	const id = "tei.number.within"
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID == id {
			return validationeval.OperatorImplementation{
				Operator: id,
				Input:    available.Input,
				Numbers:  available.Numbers,
			}
		}
	}
	t.Fatal("builtin tei.number.within not found")
	return validationeval.OperatorImplementation{}
}

// readByAttributeSource answers a number per attribute, so a rule that reads a
// second attribute can be told apart from one that reads the subject only.
type readByAttributeSource struct {
	values map[string]*big.Rat
	calls  []string
}

func (s *readByAttributeSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.calls = append(s.calls, "Number("+string(attribute)+")")
	value, ok := s.values[string(attribute)]
	if !ok {
		return nil, nil
	}
	return value, nil
}

func (s *readByAttributeSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, nil
}

func mustParseRat(t *testing.T, text string) *big.Rat {
	t.Helper()
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatalf("invalid rational: %s", text)
	}
	return value
}

// A per-input example states one value per declared input, and the values reach
// the implementation under the names the declaration gave them. Swapping the
// numbers would answer differently, so a wrong mapping cannot pass.
func TestMultiInputOperatorChecksPerInputExamples(t *testing.T) {
	contract := withinContract
	contract.Cases = []semantic.OperatorCase{
		{Inputs: map[string]string{"value": "3", "low": "0", "high": "5"}, Satisfied: true},
		{Inputs: map[string]string{"value": "6", "low": "0", "high": "5"}, Satisfied: false},
		// The subject is the first input, so its name is "value" here.
	}
	rule := discountInRange
	rule.Operator = contract.ID
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "5"},
	}
	_, err := validationeval.EvaluateWithOperators(rule, &operatorSource{value: mustParseRat(t, "3")}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err != nil {
		t.Fatalf("an implementation that agrees with every example was refused: %v", err)
	}

	// An example that states the wrong values is refused before any row is read.
	contract.Cases = []semantic.OperatorCase{
		{Inputs: map[string]string{"value": "3", "low": "5", "high": "0"}, Satisfied: true},
	}
	_, err = validationeval.EvaluateWithOperators(rule, &operatorSource{value: mustParseRat(t, "3")}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatal("an example the implementation answers differently was accepted")
	}
}

// An example that leaves one declared input unstated is refused: it would
// otherwise be checked as if the missing input held something nobody wrote.
func TestMultiInputOperatorRefusesAnExampleThatOmitsAnInput(t *testing.T) {
	contract := withinContract
	contract.Cases = []semantic.OperatorCase{
		{Inputs: map[string]string{"value": "3", "low": "0"}, Satisfied: true},
	}
	rule := discountInRange
	rule.Operator = contract.ID
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "5"},
	}
	_, err := validationeval.EvaluateWithOperators(rule, &operatorSource{value: mustParseRat(t, "3")}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	})
	if err == nil {
		t.Fatal("an example that omits an input was accepted")
	}
	if !strings.Contains(err.Error(), "is not stated") {
		t.Fatalf("error=%v, want it to name the missing input", err)
	}
}

// A single-input operator may name its one input, and its example then uses that
// name as the column. The value must reach the implementation, not nil.
func TestNamedSingleInputExampleReachesTheImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID:        "sales.non_negative",
		Symbol:    "non_negative",
		Input:     semantic.OperatorInputNumber,
		InputName: "amount",
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "amount", Type: semantic.OperatorInputNumber},
		},
		Cases: []semantic.OperatorCase{
			{Inputs: map[string]string{"amount": "-1"}, Satisfied: false},
			{Inputs: map[string]string{"amount": "0"}, Satisfied: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.non_negative",
		Input:    semantic.OperatorInputNumber,
		Number: func(value *big.Rat) (bool, error) {
			return value.Sign() >= 0, nil
		},
	}
	rule := semantic.Validation{
		ID:        "sales.order_line.amount_non_negative",
		Symbol:    "amount_non_negative",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity",
		Operator:  "sales.non_negative",
	}
	if _, err := validationeval.EvaluateWithOperators(rule, &operatorSource{value: mustParseRat(t, "2")}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}); err != nil {
		t.Fatalf("a named single input example was refused: %v", err)
	}
}
