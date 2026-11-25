package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator declares a calendar-day input.
//
// A day is read as a day. The reader reports one reading per attribute, and the
// reading states whether the value was read, whether the record holds no value,
// or which day it holds. The implementation receives the days in the one shape
// section 8.4 fixes — one value per input, carrying the type it was read as —
// and answers a boolean; it never sees a Source, so what a rule reads stays
// decided by the declaration.
//
// A rule whose day input reads every row of a relation is refused rather than
// read row by row: this reader has no reader for a column of days yet, and
// answering from one row would answer a question about the relation from one
// record of it.

// declaresDateInput reports whether some declared input reads a calendar day.
func declaresDateInput(inputs []semantic.OperatorInputDeclaration) bool {
	for _, input := range inputs {
		if input.Type == semantic.OperatorInputDate {
			return true
		}
	}
	return false
}

// decideDateOperator decides a rule whose operator declares at least one
// calendar-day input, from the days of one record.
func decideDateOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	dates, ok := source.(StatefulDateSource)
	if !ok {
		return Decision{}, fmt.Errorf("semantic validation %s reads operator %s, which takes a calendar day, but this reader cannot read a calendar day", rule.ID, rule.Operator)
	}
	for _, input := range inputs {
		if input.OfEachRow {
			return Decision{}, fmt.Errorf(
				"semantic validation %s reads operator %s input %q over every row of relation %s, but this reader reads a calendar day of one record; a column of days has no reader yet",
				rule.ID, rule.Operator, input.Name, rule.Relation)
		}
	}
	// The rule subject supplies the first input, and only it.
	for _, assignment := range rule.Inputs {
		if assignment.Name == inputs[0].Name {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
		}
	}
	// Every input is read before the operator is asked, so an unread part is
	// never turned into a settled answer and a value the caller supplied is
	// never left unread.
	values := make(map[string]semantic.OperatorValue, len(inputs))
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				// A constant day is written YYYY-MM-DD. It is read as a day
				// rather than through the reader, and it belongs to no record,
				// so no attempt is collected for it.
				day, err := semantic.ParseCalendarDate(assignment.Value)
				if err != nil {
					return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is neither an attribute nor a calendar day: %w",
						rule.ID, rule.Operator, input.Name, assignment.Value, err)
				}
				values[input.Name] = semantic.OperatorValue{Date: &day}
				continue
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		reading, err := readDate(rule, dates, attribute)
		if err != nil {
			return Decision{}, err
		}
		switch reading.State {
		case ValuePresent:
			day := reading.Value
			values[input.Name] = semantic.OperatorValue{Date: &day}
		case ValueNull:
			// A record that holds no day for this attribute has no value to
			// ask the operator about. The decision is undecided, and the reason
			// is the record's own.
			return Decision{Undecided: &Undecided{Reason: UndecidedValueNull, Attribute: attribute}}, nil
		case ValueNotFetched:
			return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}}, nil
		default:
			return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	}
	// Every input came from this one record, so no two-attempt check is needed:
	// the reader states the record it read.
	//
	// The implementation is asked about the values in the one shape section 8.4
	// fixes: one value per input, carrying the type it was read as. A value the
	// reader did not read was already turned into an undecided reason above, so
	// every value in hand is a present one.
	if implementation.Values == nil {
		return Decision{}, fmt.Errorf("operator %s takes a calendar day, but its selected implementation does not decide values of any type", rule.Operator)
	}
	satisfied, err := implementation.Values(values)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	if satisfied {
		return Decision{Result: ResultSatisfied}, nil
	}
	return Decision{Result: ResultNotSatisfied}, nil
}
