package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// StatefulTextSource is implemented by a reader that can read text values. A
// rule over text is not evaluated with a reader that cannot: the rule is
// refused, it is not decided from another representation (decision section
// 8.4). An empty string is a value; a cell the reader reports as holding no
// value is not.
type StatefulTextSource interface {
	ReadText(attribute core.CanonicalID) (TextReading, error)
}

// TextReading is one string and its state. Value is a string exactly when State
// is ValuePresent, and the zero value otherwise.
type TextReading struct {
	State ValueState
	Value string
}

// StatefulTruthSource is implemented by a reader that can read truth values. A
// rule over truth is not evaluated with a reader that cannot: the rule is
// refused, it is not decided from another representation (decision section 8.4).
type StatefulTruthSource interface {
	ReadTruth(attribute core.CanonicalID) (TruthReading, error)
}

// TruthReading is one truth value and its state. Value is a bool exactly when
// State is ValuePresent, and the zero value otherwise.
type TruthReading struct {
	State ValueState
	Value bool
}

// StatefulAnySource is implemented by a reader that can read a value of any of
// the kinds this program reads, keeping which kind it read (decision section
// 8.4). A rule over `any` is not evaluated with a reader that cannot: the rule
// is refused, it is not decided by reading the value as a number.
type StatefulAnySource interface {
	ReadAny(attribute core.CanonicalID) (AnyReading, error)
}

// AnyReading is one value of any kind and its state. Value is set exactly when
// State is ValuePresent, and it keeps the kind it was read as, so a number and
// a text that spell the same characters stay two values.
type AnyReading struct {
	State ValueState
	Value semantic.OperatorValue
}

// readOperatorInputValue reads one input's value the way the declaration says
// it is read, and returns the value in the field that type owns (decision
// section 8.4: the evaluator reads the value according to the declared type and
// fills that type's field only).
//
// A reader that cannot read the declared type is an error, not a value of
// another type: reading a text as a number would answer a question about a
// different value. Exactly one of the first two results is set when err is nil.
func readOperatorInputValue(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, attribute core.CanonicalID) (*semantic.OperatorValue, *Undecided, error) {
	switch input.Type {
	case semantic.OperatorInputNumber:
		value, stopped, err := readNumberForAttribute(rule, attribute, source)
		if err != nil {
			return nil, nil, err
		}
		if stopped != nil {
			return nil, stopped, nil
		}
		return &semantic.OperatorValue{Number: value}, nil, nil
	case semantic.OperatorInputDate:
		dates, readsDates := source.(StatefulDateSource)
		if !readsDates {
			return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as a date; the %s input has no reader on this path yet", rule.ID, rule.Operator, input.Name, input.Type)
		}
		reading, err := readDate(rule, dates, attribute)
		if err != nil {
			return nil, nil, err
		}
		value, stopped, err := presentDateReading(rule, attribute, reading)
		if err != nil {
			return nil, nil, err
		}
		if stopped != nil {
			return nil, stopped, nil
		}
		return &semantic.OperatorValue{Date: &value}, nil, nil
	case semantic.OperatorInputText:
		texts, readsTexts := source.(StatefulTextSource)
		if !readsTexts {
			return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as text; the %s input has no reader on this path yet", rule.ID, rule.Operator, input.Name, input.Type)
		}
		reading, err := texts.ReadText(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			return &semantic.OperatorValue{Text: &reading.Value}, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, &Undecided{Reason: undecidedReasonFor(reading.State), Attribute: attribute}, nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	case semantic.OperatorInputTruth:
		truths, readsTruths := source.(StatefulTruthSource)
		if !readsTruths {
			return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as truth; the %s input has no reader on this path yet", rule.ID, rule.Operator, input.Name, input.Type)
		}
		reading, err := truths.ReadTruth(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			return &semantic.OperatorValue{Truth: &reading.Value}, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, &Undecided{Reason: undecidedReasonFor(reading.State), Attribute: attribute}, nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	case semantic.OperatorInputAny:
		anys, readsAnys := source.(StatefulAnySource)
		if !readsAnys {
			return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as a value of any type; the %s input has no reader on this path yet", rule.ID, rule.Operator, input.Name, input.Type)
		}
		reading, err := anys.ReadAny(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			if !reading.Value.Valid() {
				return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present with a value no type reads", rule.ID, attribute)
			}
			return &reading.Value, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, &Undecided{Reason: undecidedReasonFor(reading.State), Attribute: attribute}, nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	default:
		return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as %s, which is not a type this program reads a value as", rule.ID, rule.Operator, input.Name, input.Type)
	}
}

// undecidedReasonFor maps a reading's state to the reason a rule cannot be
// decided: a value the record does not hold is not the same as a value the
// reader did not fetch (decision section 5).
func undecidedReasonFor(state ValueState) UndecidedReason {
	if state == ValueNotFetched {
		return UndecidedValueNotFetched
	}
	return UndecidedValueNull
}

// presentDateReading returns the value of a present date reading, or the reason
// there is none. Exactly one of the first two results is set when err is nil.
func presentDateReading(rule semantic.Validation, attribute core.CanonicalID, reading DateReading) (semantic.CalendarDate, *Undecided, error) {
	switch reading.State {
	case ValuePresent:
		return reading.Value, nil, nil
	case ValueNull, ValueNotFetched:
		return semantic.CalendarDate{}, &Undecided{Reason: undecidedReasonFor(reading.State), Attribute: attribute}, nil
	default:
		return semantic.CalendarDate{}, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
}
