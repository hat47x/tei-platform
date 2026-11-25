package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// typedSource answers every read of every type from one table of cells, and
// records which read was asked. It exists so a test can see what the evaluator
// handed an implementation, and so it can refuse a read of a type it does not
// implement.
type typedSource struct {
	cells map[core.CanonicalID]string
	reads []string
}

func (s *typedSource) note(read string) { s.reads = append(s.reads, read) }

func (s *typedSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s *typedSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	s.note("number:" + string(attribute))
	value, err := semantic.ParseOperatorCaseValue(s.cells[attribute])
	if err != nil {
		return validationeval.NumberReading{}, err
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: value}, nil
}

func (s *typedSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	s.note("date:" + string(attribute))
	day, err := semantic.ParseCalendarDate(s.cells[attribute])
	if err != nil {
		return validationeval.DateReading{}, err
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: day}, nil
}

func (s *typedSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	s.note("text:" + string(attribute))
	cell, err := semantic.ParseOperatorCaseCell(s.cells[attribute])
	if err != nil {
		return validationeval.TextReading{}, err
	}
	if !cell.Fits(semantic.OperatorInputText) {
		return validationeval.TextReading{}, errNotText
	}
	text := cell.Text
	return validationeval.TextReading{State: validationeval.ValuePresent, Value: &text}, nil
}

func (s *typedSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	s.note("truth:" + string(attribute))
	cell, err := semantic.ParseOperatorCaseCell(s.cells[attribute])
	if err != nil {
		return validationeval.TruthReading{}, err
	}
	if !cell.Fits(semantic.OperatorInputTruth) {
		return validationeval.TruthReading{}, errNotTruth
	}
	truth := cell.Truth
	return validationeval.TruthReading{State: validationeval.ValuePresent, Value: &truth}, nil
}

func (s *typedSource) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	s.note("any:" + string(attribute))
	cell, err := semantic.ParseOperatorCaseCell(s.cells[attribute])
	if err != nil {
		return validationeval.AnyReading{}, err
	}
	value, err := validationeval.OperatorValueFor(semantic.OperatorInputAny, cell)
	if err != nil {
		return validationeval.AnyReading{}, err
	}
	return validationeval.AnyReading{State: validationeval.ValuePresent, Value: &value}, nil
}

func (s *typedSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

var (
	errNotText  = errString(`" is not text; write it between double quotes`)
	errNotTruth = errString(`" is not the word true or the word false`)
)

type errString string

func (e errString) Error() string { return string(e) }

// noTextSource reads every type but text. A reader that cannot read the
// declared type is refused rather than answered from a value of another type.
type noTextSource struct{ inner *typedSource }

func (s noTextSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	return s.inner.ReadAttempt(attribute)
}

func (s noTextSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return s.inner.ReadNumber(attribute)
}

func (s noTextSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	return s.inner.ReadDate(attribute)
}

func (s noTextSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	return s.inner.ReadTruth(attribute)
}

func (s noTextSource) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	return s.inner.ReadAny(attribute)
}

func (s noTextSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return s.inner.ReadRelationKeys(relation, attribute)
}

// typedContract is a one-record operator whose single input has the given type.
func typedContract(kind semantic.OperatorInput) semantic.Operator {
	return semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: kind,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: kind}},
	}
}

func typedRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.op",
	}
}

// An implementation that decides from typed values is handed the value of the
// type the declaration states, and the read is the read of that type: a text
// input is read through ReadText and never through ReadNumber.
func TestATypedInputIsReadThroughItsOwnReader(t *testing.T) {
	for _, tc := range []struct {
		kind semantic.OperatorInput
		cell string
		want string
	}{
		{semantic.OperatorInputDate, "2026-04-01", "date:sales.order_line.quantity"},
		{semantic.OperatorInputText, `"A-100"`, "text:sales.order_line.quantity"},
		{semantic.OperatorInputTruth, "true", "truth:sales.order_line.quantity"},
		{semantic.OperatorInputAny, `"A-100"`, "any:sales.order_line.quantity"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			source := &typedSource{cells: map[core.CanonicalID]string{"sales.order_line.quantity": tc.cell}}
			var got validationeval.OperatorValue
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op", Input: tc.kind,
				Values: func(values map[string]validationeval.OperatorValue) (bool, error) {
					got = values["value"]
					return true, nil
				},
			}
			decision, err := validationeval.Decide(typedRule(), source, validationeval.Operators{
				Declared:        []semantic.Operator{typedContract(tc.kind)},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if err != nil {
				t.Fatalf("a %s input was refused: %v", tc.kind, err)
			}
			if decision.Result != validationeval.ResultSatisfied {
				t.Fatalf("result=%q, want satisfied", decision.Result)
			}
			if len(source.reads) != 1 || source.reads[0] != tc.want {
				t.Fatalf("reads=%v, want exactly %q", source.reads, tc.want)
			}
			switch tc.kind {
			case semantic.OperatorInputDate:
				if got.Date == nil || got.Date.String() != "2026-04-01" {
					t.Fatalf("date=%v, want 2026-04-01", got.Date)
				}
			case semantic.OperatorInputText:
				if got.Text == nil || *got.Text != "A-100" {
					t.Fatalf("text=%v, want A-100", got.Text)
				}
			case semantic.OperatorInputTruth:
				if got.Truth == nil || !*got.Truth {
					t.Fatalf("truth=%v, want true", got.Truth)
				}
			case semantic.OperatorInputAny:
				if got.Any == nil || got.Any.Text == nil || *got.Any.Text != "A-100" {
					t.Fatalf("any=%+v, want the text A-100", got.Any)
				}
				if got.Number != nil || got.Date != nil || got.Truth != nil {
					t.Fatalf("the value also states another type: %+v", got)
				}
			}
		})
	}
}

// The number 100 and the text "100" are two values of one `any` input, and they
// reach the implementation as two values.
func TestAnAnyInputKeepsTheTwoValuesApart(t *testing.T) {
	var seen []validationeval.OperatorValue
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputAny,
		Values: func(values map[string]validationeval.OperatorValue) (bool, error) {
			seen = append(seen, values["value"])
			return true, nil
		},
	}
	for _, cell := range []string{"100", `"100"`} {
		source := &typedSource{cells: map[core.CanonicalID]string{"sales.order_line.quantity": cell}}
		if _, err := validationeval.Decide(typedRule(), source, validationeval.Operators{
			Declared:        []semantic.Operator{typedContract(semantic.OperatorInputAny)},
			Implementations: []validationeval.OperatorImplementation{implementation},
		}); err != nil {
			t.Fatalf("%s was refused: %v", cell, err)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("the implementation was asked %d times, want 2", len(seen))
	}
	first, second := seen[0], seen[1]
	if first.Any == nil || first.Any.Number == nil || first.Any.Number.Cmp(big.NewRat(100, 1)) != 0 {
		t.Fatalf("the first value is %+v, want the number 100", first.Any)
	}
	if second.Any == nil || second.Any.Text == nil || *second.Any.Text != "100" {
		t.Fatalf("the second value is %+v, want the text \"100\"", second.Any)
	}
	if second.Any.Number != nil {
		t.Fatalf(`the text "100" was also handed as the number %s`, second.Any.Number.RatString())
	}
}

// A reader that cannot read the declared type is refused, and the read of
// another type is never substituted for it.
func TestATypedInputIsRefusedByAReaderWithoutThatReader(t *testing.T) {
	source := &typedSource{cells: map[core.CanonicalID]string{"sales.order_line.quantity": "2026-04-01"}}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputText,
		Values: func(map[string]validationeval.OperatorValue) (bool, error) { return true, nil },
	}
	_, err := validationeval.Decide(typedRule(), noTextSource{inner: source}, validationeval.Operators{
		Declared:        []semantic.Operator{typedContract(semantic.OperatorInputText)},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a text input was decided by a reader that cannot read text")
	}
	if !strings.Contains(err.Error(), "the text input has no reader on this path yet") {
		t.Fatalf("the refusal does not give the reason: %v", err)
	}
	for _, read := range source.reads {
		if !strings.HasPrefix(read, "text:") {
			t.Fatalf("the reader was asked %q for a text input", read)
		}
	}
}

// An implementation that decides from numbers is not selected for a contract
// that declares a typed input: the numbers would answer a question about a date
// or a text, which is the bug #4359 was about.
func TestANumericImplementationIsNotSelectedForATypedContract(t *testing.T) {
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputDate,
		Number: func(*big.Rat) (bool, error) { return true, nil },
	}
	err := validationeval.CheckImplementationMatchesContract(typedContract(semantic.OperatorInputDate), implementation)
	if err == nil {
		t.Fatal("a numeric implementation was selected for a date contract")
	}
	for _, want := range []string{`"value"`, "date", "sales.op"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %v", want, err)
		}
	}
}

// A typed implementation is not selected for a numeric contract either, and a
// numeric contract still decides from its numbers.
func TestATypedImplementationIsNotSelectedForANumericContract(t *testing.T) {
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputNumber,
		Values: func(map[string]validationeval.OperatorValue) (bool, error) { return true, nil },
	}
	if err := validationeval.CheckImplementationMatchesContract(typedContract(semantic.OperatorInputNumber), implementation); err == nil {
		t.Fatal("a typed implementation was selected for a numeric contract")
	}
	numbers := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputNumber,
		Number: func(value *big.Rat) (bool, error) { return value.Sign() >= 0, nil },
	}
	if err := validationeval.CheckImplementationMatchesContract(typedContract(semantic.OperatorInputNumber), numbers); err != nil {
		t.Fatalf("a numeric implementation was refused for a numeric contract: %v", err)
	}
}

// The contract's own examples are asked of the implementation in the form the
// declaration asks for, and an example that states another type is refused
// instead of being read as a number.
func TestATypedContractAsksItsExamplesAsTypedValues(t *testing.T) {
	contract := typedContract(semantic.OperatorInputAny)
	contract.Cases = []semantic.OperatorCase{
		{Value: "100", Satisfied: true},
		{Value: `"100"`, Satisfied: false},
	}
	var asked []validationeval.OperatorValue
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputAny,
		Values: func(values map[string]validationeval.OperatorValue) (bool, error) {
			asked = append(asked, values["value"])
			value := values["value"]
			return value.Any != nil && value.Any.Number != nil, nil
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, implementation); err != nil {
		t.Fatalf("the examples were refused: %v", err)
	}
	if len(asked) != 2 {
		t.Fatalf("the implementation was asked %d examples, want 2", len(asked))
	}
	if asked[0].Any == nil || asked[0].Any.Number == nil {
		t.Fatalf("the first example was not the number 100: %+v", asked[0].Any)
	}
	if asked[1].Any == nil || asked[1].Any.Text == nil || *asked[1].Any.Text != "100" {
		t.Fatalf(`the second example was not the text "100": %+v`, asked[1].Any)
	}
}
