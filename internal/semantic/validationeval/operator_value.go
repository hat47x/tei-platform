package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/semantic"
)

// OperatorValue is the value of one input, read according to the type its
// declaration gave (section 8.4). Exactly one field is set, and the field is
// decided by the declared type, never by what the value happens to look like:
// the evaluator reads the value the way the declaration says and fills that
// field, so an implementation cannot be handed a number for a text input.
//
// The fields are separate rather than one `any` because a value that is
// presented as the wrong type is a contract error, and a type switch over an
// untyped value would read it silently as whatever the driver happened to
// return. An unset field means the input was not read, not that it held a zero
// value: an implementation that reads a field its declaration does not state
// gets an undecided answer rather than a zero.
type OperatorValue struct {
	// Number is the value of a `number` input.
	Number *big.Rat
	// Date is the value of a `date` input.
	Date *semantic.CalendarDate
	// Text is the value of a `text` input.
	Text *string
	// Truth is the value of a `truth` input.
	Truth *bool
	// Any is the value of an `any` input. It keeps the type the value was
	// read as, so a number and a text that spell the same characters stay
	// two different values (section 8.3, section 8.4).
	Any *AnyValue
}

// AnyValue is a value read for an `any` input. It keeps its own type because
// the type is not known from the declaration: `100` and `"100"` are one
// spelling apart in the table and must stay two values.
type AnyValue struct {
	Number *big.Rat
	Date   *semantic.CalendarDate
	Text   *string
	Truth  *bool
}

// Set records the value of the field the given type names. A word this package
// does not read is an error rather than a silently dropped value, because the
// declaration was checked before this point and a type reaching here has a
// field.
func (v *OperatorValue) Set(kind semantic.OperatorInput, value any) error {
	switch kind {
	case semantic.OperatorInputNumber:
		number, ok := value.(*big.Rat)
		if !ok {
			return fmt.Errorf("a number input was handed a %T", value)
		}
		v.Number = number
	case semantic.OperatorInputDate:
		date, ok := value.(*semantic.CalendarDate)
		if !ok {
			return fmt.Errorf("a date input was handed a %T", value)
		}
		v.Date = date
	case semantic.OperatorInputText:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("a text input was handed a %T", value)
		}
		v.Text = &text
	case semantic.OperatorInputTruth:
		truth, ok := value.(bool)
		if !ok {
			return fmt.Errorf("a truth input was handed a %T", value)
		}
		v.Truth = &truth
	case semantic.OperatorInputAny:
		any, ok := value.(AnyValue)
		if !ok {
			return fmt.Errorf("an any input was handed a %T", value)
		}
		v.Any = &any
	default:
		return fmt.Errorf("input type %s has no value field", kind)
	}
	return nil
}

// SetAny records a value read for an `any` input. The word of the value's own
// type decides which field it lands in, so the type travels with the value.
func (v *AnyValue) SetAny(kind semantic.OperatorInput, value any) error {
	switch kind {
	case semantic.OperatorInputNumber:
		number, ok := value.(*big.Rat)
		if !ok {
			return fmt.Errorf("a number was handed a %T", value)
		}
		v.Number = number
	case semantic.OperatorInputDate:
		date, ok := value.(*semantic.CalendarDate)
		if !ok {
			return fmt.Errorf("a date was handed a %T", value)
		}
		v.Date = date
	case semantic.OperatorInputText:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("a text was handed a %T", value)
		}
		v.Text = &text
	case semantic.OperatorInputTruth:
		truth, ok := value.(bool)
		if !ok {
			return fmt.Errorf("a truth was handed a %T", value)
		}
		v.Truth = &truth
	default:
		return fmt.Errorf("input type %s has no value field", kind)
	}
	return nil
}

// Stated reports whether this value carries something an implementation can
// read. A value with no field is an input nobody read, and an implementation
// must answer undecided for it rather than read it as a zero.
func (v OperatorValue) Stated() bool {
	return v.Number != nil || v.Date != nil || v.Text != nil || v.Truth != nil || v.Any != nil
}

// Unread is the value of an input nobody read. It is not the zero value of
// any type: an implementation asked about it answers undecided.
func Unread() OperatorValue { return OperatorValue{} }

// HasField reports whether the given input type has a value field here. It is
// what the authoring surface consults before it offers a type word: a word with
// no field would build a contract nothing can decide.
//
// presence is absent from this list on purpose. A presence input is answered
// from whether the attribute holds a value, not from a value, so it has no
// value field to fill; an implementation is asked about it as present, absent,
// or unknown. It is a reader that exists, which is why the authoring surface
// offers it, and it is not a value, which is why it has no field here.
func HasField(kind semantic.OperatorInput) bool {
	switch kind {
	case semantic.OperatorInputNumber, semantic.OperatorInputDate,
		semantic.OperatorInputText, semantic.OperatorInputTruth, semantic.OperatorInputAny:
		return true
	case semantic.OperatorInputPresence:
		return false
	}
	return false
}

// OperatorInput is re-exported here so a caller building an implementation
// does not have to reach into the semantic package for the type words.
