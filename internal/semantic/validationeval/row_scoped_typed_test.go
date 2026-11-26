package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator reads every row of a relation and declares an input of a
// type other than number is decided from a column of that type (decision section
// 8.4). Before this, the row-scoped path read every input through
// ReadRelationNumbers and refused a typed input by name, so a column of days
// could not be read at all.
//
// The expectations below are written from the meaning: the implementation
// receives the rows in the order they were read, each with the state of the row
// next to its value, and the composition with the coverage and the refutation
// statement is the one place that decides it.

// rowValueSource reads one column of typed values over the rows of a relation.
type rowValueSource struct {
	coverage validationeval.Coverage
	columns  map[core.CanonicalID][]validationeval.RowValue
}

func (s *rowValueSource) ReadRelationValues(relation, attribute core.CanonicalID) (validationeval.ValueRowsReading, error) {
	rows, known := s.columns[attribute]
	if !known {
		return validationeval.ValueRowsReading{}, &unreadRelationError{relation: relation}
	}
	return validationeval.ValueRowsReading{Coverage: s.coverage, Rows: rows}, nil
}

func (s *rowValueSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	return validationeval.NumberRowsReading{}, &unreadRelationError{relation: relation}
}

func (s *rowValueSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s *rowValueSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *rowValueSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	return "read-1", true
}

type unreadRelationError struct{ relation core.CanonicalID }

func (e *unreadRelationError) Error() string {
	return "this reader cannot read relation " + string(e.relation)
}

// anyDayValue carries a day as a value of any type, which is how a driver hands
// a calendar date to an implementation.
func anyDayValue(year, month, day int) semantic.OperatorValue {
	d := semantic.CalendarDate{Year: year, Month: month, Day: day}
	return semantic.OperatorValue{Any: &semantic.AnyValue{Type: semantic.OperatorInputDate, Date: &d}}
}

// dayValue carries a day as the value of an input declared `date`.
func dayValue(year, month, day int) semantic.OperatorValue {
	d := semantic.CalendarDate{Year: year, Month: month, Day: day}
	return semantic.OperatorValue{Date: &d}
}

func presentDayRow(year, month, day int) validationeval.RowValue {
	return validationeval.RowValue{State: validationeval.ValuePresent, Value: dayValue(year, month, day)}
}

// everyDayAtLeast states that every row's day is not before the deadline's day.
// It is a plain reading of that meaning, written here so a wrong evaluator is
// caught even if the shipped one were wrong too.
func everyDayAtLeast(refutes bool) validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.every_day_at_least",
		Input:    semantic.OperatorInputDate,
		ValueRows: func(rows map[string][]validationeval.RowValue) (validationeval.Tri, error) {
			values, ok := rows["value"]
			if !ok {
				return validationeval.TriMaybe, nil
			}
			deadlines, ok := rows["deadline"]
			if !ok {
				return validationeval.TriMaybe, nil
			}
			if len(values) != len(deadlines) {
				return validationeval.TriMaybe, nil
			}
			for index := range values {
				value, deadline := values[index], deadlines[index]
				if value.State != validationeval.ValuePresent || deadline.State != validationeval.ValuePresent {
					continue
				}
				if value.Value.Date == nil || deadline.Value.Date == nil {
					continue
				}
				if value.Value.Date.Compare(*deadline.Value.Date) < 0 {
					return validationeval.TriNo, nil
				}
			}
			for index := range values {
				if values[index].State != validationeval.ValuePresent || deadlines[index].State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func everyDayAtLeastContract(refutes bool) semantic.Operator {
	return semantic.Operator{
		ID: "sales.every_day_at_least", Symbol: "every_day_at_least",
		Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
			{Name: "deadline", Type: semantic.OperatorInputDate, OfEachRow: true},
		},
		RefutesOnSomeRows: refutes,
	}
}

func everyDayAtLeastRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order.lines.shipped_not_before_deadline", Symbol: "shipped_not_before_deadline",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.shipped_at",
		Relation:  "sales.order.lines",
		Operator:  "sales.every_day_at_least",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "deadline", Value: "sales.order_line.deadline_at"},
		},
	}
}

func decideEveryDayAtLeast(t *testing.T, source validationeval.StatefulSource, refutes bool) validationeval.Decision {
	t.Helper()
	decision, err := validationeval.Decide(everyDayAtLeastRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{everyDayAtLeastContract(refutes)},
		Implementations: []validationeval.OperatorImplementation{everyDayAtLeast(refutes)},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	return decision
}

// Every row's day is not before its deadline, so the rule holds.
func TestAColumnOfDaysIsDecidedFromTheDaysThatWereRead(t *testing.T) {
	source := &rowValueSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValue{
			"sales.order_line.shipped_at":  {presentDayRow(2026, 4, 6), presentDayRow(2026, 4, 7)},
			"sales.order_line.deadline_at": {presentDayRow(2026, 4, 5), presentDayRow(2026, 4, 5)},
		},
	}
	decision := decideEveryDayAtLeast(t, source, true)
	if decision.Result != validationeval.ResultSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// A row whose day is before its deadline refutes it, and the refutation stands
// even when another row was not read (decision section 6).
func TestAColumnOfDaysKeepsACounterexampleNextToAnUnreadRow(t *testing.T) {
	source := &rowValueSource{
		coverage: validationeval.CoveragePartial,
		columns: map[core.CanonicalID][]validationeval.RowValue{
			"sales.order_line.shipped_at": {
				presentDayRow(2026, 4, 1),
				{State: validationeval.ValueNotFetched},
			},
			"sales.order_line.deadline_at": {
				presentDayRow(2026, 4, 5),
				presentDayRow(2026, 4, 5),
			},
		},
	}
	decision := decideEveryDayAtLeast(t, source, true)
	if decision.Result != validationeval.ResultNotSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want not-satisfied (a witnessed counterexample)", decision)
	}
}

// A row that holds no day keeps its own reason, and it is reported before the
// coverage, because reading more rows cannot settle it.
func TestAColumnOfDaysReportsTheRowReasonBeforeTheCoverage(t *testing.T) {
	source := &rowValueSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValue{
			"sales.order_line.shipped_at": {
				presentDayRow(2026, 4, 6),
				{State: validationeval.ValueNull},
			},
			"sales.order_line.deadline_at": {
				presentDayRow(2026, 4, 5),
				presentDayRow(2026, 4, 5),
			},
		},
	}
	decision := decideEveryDayAtLeast(t, source, true)
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}

// A partly read relation with no counterexample is undecided unless the contract
// states that a counterexample among the rows read is enough.
func TestAColumnOfDaysStaysUndecidedWhenTheRefutationIsNotStated(t *testing.T) {
	// Both rows were read and neither refutes the contract, so the only thing
	// that stands between the answer and "holds" is that the relation was only
	// partly read.
	source := &rowValueSource{
		coverage: validationeval.CoveragePartial,
		columns: map[core.CanonicalID][]validationeval.RowValue{
			"sales.order_line.shipped_at": {
				presentDayRow(2026, 4, 6),
				presentDayRow(2026, 4, 7),
			},
			"sales.order_line.deadline_at": {
				presentDayRow(2026, 4, 5),
				presentDayRow(2026, 4, 5),
			},
		},
	}
	decision := decideEveryDayAtLeast(t, source, false)
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("decision=%+v, want undecided relation-partially-read", decision)
	}
}

// A constant day is read in the type the declaration gives the input, so the
// deadline may be one day rather than a column.
func TestAConstantDayColumnHoldsThatDayInEveryRow(t *testing.T) {
	rule := everyDayAtLeastRule()
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: "2026-04-05"}}
	source := &rowValueSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValue{
			"sales.order_line.shipped_at": {presentDayRow(2026, 4, 6)},
		},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{everyDayAtLeastContract(true)},
		Implementations: []validationeval.OperatorImplementation{everyDayAtLeast(true)},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// An implementation that decides rows of numbers cannot decide a column of
// days, because the values it would be asked about are not the ones it reads.
func TestANumberRowImplementationCannotDecideAColumnOfDays(t *testing.T) {
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.every_day_at_least",
		Input:    semantic.OperatorInputDate,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	err := validationeval.CheckImplementationMatchesContract(everyDayAtLeastContract(true), implementation)
	if err == nil {
		t.Fatal("an implementation that decides rows of numbers was accepted for a column of days")
	}
}

// An implementation that decides both kinds of rows would leave which column it
// answers about ambiguous, so the selection is refused.
func TestAnImplementationCannotDecideBothKindsOfRows(t *testing.T) {
	implementation := everyDayAtLeast(true)
	implementation.NumberRows = func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
		return validationeval.TriYes, nil
	}
	err := validationeval.CheckImplementationMatchesContract(everyDayAtLeastContract(true), implementation)
	if err == nil {
		t.Fatal("an implementation that decides rows of numbers and rows of days was accepted")
	}
}

// A row example of a typed row-scoped operator is read in the type the
// declaration gives the input, and the implementation is asked about it before
// any real row is read.
func TestATypedRowExampleIsCheckedAgainstTheImplementation(t *testing.T) {
	contract := everyDayAtLeastContract(true)
	contract.RowCases = []semantic.OperatorRowCase{
		{
			Rows: []map[string]string{
				{"value": "2026-04-06", "deadline": "2026-04-05"},
				{"value": "2026-04-07", "deadline": "2026-04-05"},
			},
			Answer: semantic.OperatorRowSatisfied,
		},
		{
			Rows: []map[string]string{
				{"value": "2026-04-06", "deadline": "2026-04-05"},
				{"value": "2026-04-04", "deadline": "2026-04-05"},
			},
			Answer: semantic.OperatorRowNotSatisfied,
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, everyDayAtLeast(true)); err != nil {
		t.Fatalf("the shipped shape disagrees with the examples: %v", err)
	}
	// The same rows with the other answer are refused, and no real data is read.
	wrong := contract
	wrong.RowCases = []semantic.OperatorRowCase{{
		Rows: []map[string]string{
			{"value": "2026-04-06", "deadline": "2026-04-05"},
			{"value": "2026-04-07", "deadline": "2026-04-05"},
		},
		Answer: semantic.OperatorRowNotSatisfied,
	}}
	err := validationeval.CheckImplementationMatchesContract(wrong, everyDayAtLeast(true))
	if err == nil {
		t.Fatal("an example the implementation contradicts was accepted")
	}
	// The empty relation is one example, and it is asked about no rows at all.
	empty := contract
	empty.RowCases = []semantic.OperatorRowCase{{Rows: []map[string]string{}, Answer: semantic.OperatorRowSatisfied}}
	if err := validationeval.CheckImplementationMatchesContract(empty, everyDayAtLeast(true)); err != nil {
		t.Fatalf("the empty relation was refused: %v", err)
	}
}
