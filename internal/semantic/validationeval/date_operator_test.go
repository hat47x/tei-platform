package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A calendar-day input is read as a day and decided by a date function. Before
// this, `takes start: date` was refused by the surface, so no contract could
// state it; the design's std.not_overlapping (section 3) needs it.
//
// The expectations below are written from the meaning: the day is the day the
// reader read, and the implementation is asked about the day and not about a
// number.

// dateSource states one day per attribute, so a test can hold a present day, a
// record that holds no day, and an attribute that was not read.
type dayOperatorSource struct {
	readings map[string]validationeval.DateReading
	attempts map[string]string
}

func (s *dayOperatorSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	reading, ok := s.readings[string(attribute)]
	if !ok {
		return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
	}
	return reading, nil
}

func (s *dayOperatorSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s *dayOperatorSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	attempt, stated := s.attempts[string(attribute)]
	return attempt, stated
}

func (s *dayOperatorSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func day(year, month, day int) semantic.CalendarDate {
	return semantic.CalendarDate{Year: year, Month: month, Day: day}
}

// notBefore states that the subject's day is not before the deadline the rule
// names. It is a plain reading of that meaning.
func notBefore() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.not_before",
		Input:    semantic.OperatorInputDate,
		Dates: func(values map[string]semantic.CalendarDate) (bool, error) {
			return values["value"].Compare(values["deadline"]) >= 0, nil
		},
	}
}

func notBeforeContract() semantic.Operator {
	return semantic.Operator{
		ID: "sales.not_before", Symbol: "not_before",
		Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputDate},
			{Name: "deadline", Type: semantic.OperatorInputDate},
		},
	}
}

func notBeforeRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.shipped_not_before_deadline", Symbol: "shipped_not_before_deadline",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.shipped_at",
		Operator:  "sales.not_before",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "deadline", Value: "sales.order_line.deadline_at"},
		},
	}
}

func decideNotBefore(t *testing.T, shipped, deadline validationeval.DateReading) validationeval.Decision {
	t.Helper()
	source := &dayOperatorSource{
		readings: map[string]validationeval.DateReading{
			"sales.order_line.shipped_at":  shipped,
			"sales.order_line.deadline_at": deadline,
		},
		attempts: map[string]string{
			"sales.order_line.shipped_at":  "read-1",
			"sales.order_line.deadline_at": "read-1",
		},
	}
	decision, err := validationeval.Decide(notBeforeRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{notBeforeContract()},
		Implementations: []validationeval.OperatorImplementation{notBefore()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	return decision
}

// The day the reader read is the day the implementation is asked about: 2026-04-01
// is not before 2026-04-05, so the rule holds.
func TestADateInputIsDecidedFromTheDayThatWasRead(t *testing.T) {
	decision := decideNotBefore(t,
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 4, 6)},
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 4, 5)})
	if decision.Result != validationeval.ResultSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// A day before the deadline refutes it.
func TestADateInputIsNotDecidedFromANumber(t *testing.T) {
	decision := decideNotBefore(t,
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 3, 1)},
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 4, 5)})
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
}

// A record that holds no day for the subject has no day to ask about, and the
// decision is undecided with the record's own reason rather than a value this
// implementation invented.
func TestARecordWithoutADayStaysUndecided(t *testing.T) {
	decision := decideNotBefore(t,
		validationeval.DateReading{State: validationeval.ValueNull},
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 4, 5)})
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}

// A day the reader did not read is unknown, not empty, and stays undecided with
// the reader's reason.
func TestADayThatWasNotReadStaysUndecided(t *testing.T) {
	decision := decideNotBefore(t,
		validationeval.DateReading{State: validationeval.ValueNotFetched},
		validationeval.DateReading{State: validationeval.ValuePresent, Value: day(2026, 4, 5)})
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("decision=%+v, want undecided value-not-fetched", decision)
	}
}

// A constant day is written YYYY-MM-DD and read as a day, so the rule may name
// one date without an attribute holding it.
func TestAConstantDayIsReadAsADay(t *testing.T) {
	rule := notBeforeRule()
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: "2026-04-05"}}
	source := &dayOperatorSource{
		readings: map[string]validationeval.DateReading{
			"sales.order_line.shipped_at": {State: validationeval.ValuePresent, Value: day(2026, 4, 6)},
		},
		attempts: map[string]string{"sales.order_line.shipped_at": "read-1"},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{notBeforeContract()},
		Implementations: []validationeval.OperatorImplementation{notBefore()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// A text that is not a calendar day is refused as the input's value, rather
// than read as something else.
func TestAValueThatIsNotADayIsRefused(t *testing.T) {
	rule := notBeforeRule()
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: "2026-04"}}
	source := &dayOperatorSource{
		readings: map[string]validationeval.DateReading{
			"sales.order_line.shipped_at": {State: validationeval.ValuePresent, Value: day(2026, 4, 6)},
		},
		attempts: map[string]string{"sales.order_line.shipped_at": "read-1"},
	}
	_, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{notBeforeContract()},
		Implementations: []validationeval.OperatorImplementation{notBefore()},
	})
	if err == nil {
		t.Fatal("a value that is not a calendar day was accepted as one")
	}
}

// A rule whose day input reads every row of a relation is refused: this reader
// reads the day of one record, and answering from one row would answer a
// question about the relation from one record of it.
func TestADayInputThatReadsEveryRowIsRefused(t *testing.T) {
	contract := notBeforeContract()
	contract.Inputs[0].OfEachRow = true
	rule := notBeforeRule()
	rule.Relation = "sales.order.lines"
	_, err := validationeval.Decide(rule, &dayOperatorSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{notBefore()},
	})
	if err == nil {
		t.Fatal("a day input read over every row was decided from one record")
	}
}

// An implementation that provides a number function and a date function would
// answer from whichever the evaluator filled, so the selection is refused.
func TestAnImplementationCannotDecideNumbersAndDays(t *testing.T) {
	implementation := notBefore()
	implementation.Number = func(value *big.Rat) (bool, error) { return true, nil }
	err := validationeval.CheckImplementationMatchesContract(notBeforeContract(), implementation)
	if err == nil {
		t.Fatal("an implementation that decides numbers and days was accepted")
	}
}

// An implementation that decides days cannot implement an operator that
// declares no calendar-day input, whatever its inputs count.
func TestADayImplementationCannotImplementANumberOperator(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.within", Symbol: "within",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber},
			{Name: "low", Type: semantic.OperatorInputNumber},
			{Name: "high", Type: semantic.OperatorInputNumber},
		},
	}
	implementation := notBefore()
	err := validationeval.CheckImplementationMatchesContract(contract, implementation)
	if err == nil {
		t.Fatal("a day implementation was accepted for a number operator")
	}
}

// A number implementation cannot implement an operator that declares a
// calendar-day input: the days it is asked about would not reach it.
func TestANumberImplementationCannotImplementADayOperator(t *testing.T) {
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.not_before",
		Input:    semantic.OperatorInputDate,
		Number:   func(value *big.Rat) (bool, error) { return true, nil },
	}
	err := validationeval.CheckImplementationMatchesContract(notBeforeContract(), implementation)
	if err == nil {
		t.Fatal("a number implementation was accepted for a day operator")
	}
}
