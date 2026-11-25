package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A single-input operator that reads a calendar day is decided by an
// implementation that provides the Date field (one value, not the multi-value
// Dates field), so the evaluator reads the subject as a date and hands that
// value to the implementation (decision section 8.4).
func TestASingleDateInputIsReadAsADate(t *testing.T) {
	date, err := semantic.ParseCalendarDate("2026-04-01")
	if err != nil {
		t.Fatalf("the test date does not parse: %v", err)
	}
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputDate}},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.shipped_on", Operator: "sales.op",
	}
	seen := make(chan semantic.CalendarDate, 1)
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op",
		Input:    semantic.OperatorInputDate,
		Date: func(value semantic.CalendarDate) (bool, error) {
			seen <- value
			return true, nil
		},
	}
	_, decideErr := validationeval.Decide(rule, typedValueSource{
		date: validationeval.DateReading{State: validationeval.ValuePresent, Value: date},
	}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if decideErr != nil {
		t.Fatalf("the date rule was refused: %v", decideErr)
	}
	select {
	case got := <-seen:
		if got.Compare(date) != 0 {
			t.Fatalf("the implementation received %s, want %s", got, date)
		}
	default:
		t.Fatal("the implementation was not asked about the date input")
	}
	// The reader has to be able to read a date: a reader that reads numbers
	// only is refused rather than given the date as a number.
	_, err = validationeval.Decide(rule, numbersOnly{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a date rule was decided by a reader that reads numbers")
	}
	_ = core.CanonicalID("")
}
