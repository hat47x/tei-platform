package semantic

import (
	"fmt"
	"math/big"
	"strings"
)

// This file carries the value of one input as it was read, and the reading of
// the text an example table writes for it.
//
// Decision section 8.4 decides the shape: an implementation receives ONE value
// type that carries which type was read, not one field per input. `any` is the
// reason — the same input may be a number in one row and a text in another, so
// a caller choosing between a number field and a text field would make a choice
// the reader already made.
//
// Exactly one field is set. The declaration decides which: an input declared
// `number` is only ever read as a number, so an implementation that receives
// this value never has to ask which kind it holds.

// OperatorValue is the value of one input, read according to the type the
// declaration gave. Exactly one field is set, decided by that type; the zero
// value (no field set) is not a readable value and is refused rather than read
// as a number.
type OperatorValue struct {
	// Number is a value read from an input declared `number`.
	Number *big.Rat
	// Date is a value read from an input declared `date`. Section 7 fixes
	// what a reader accepts as one.
	Date *CalendarDate
	// Text is a value read from an input declared `text`. It is written with
	// quotes in an example table (section 8.3), so an empty text and no value
	// at all are different states.
	Text *string
	// Truth is a value read from an input declared `truth`.
	Truth *bool
	// Any is a value read from an input declared `any`. It keeps the type it
	// was read as, so a number and a quoted text are different values.
	Any *AnyValue
	// Presence is what an input declared `presence` reads: whether the
	// attribute holds a value, not the value itself. True means it holds one,
	// false means it holds none.
	//
	// It is a field of this value rather than a use of Truth on purpose: "the
	// attribute holds a value" and "the value is true" are different questions,
	// and answering one from the other is the bug this field prevents (#4359).
	Presence *bool
}

// AnyValue is one value read from an input that may be of any type. The Type
// says which of the remaining fields is set, so the type the reader saw
// survives the trip to an implementation.
type AnyValue struct {
	// Type is the type this value was read as.
	Type OperatorInput
	// Number, Date, Text and Truth hold the value for Type's matching field.
	Number *big.Rat
	Date   *CalendarDate
	Text   *string
	Truth  *bool
}

// OperatorInputs lists the types a value can be read as, in the order the
// reader tries them for an input declared `any`.
var OperatorInputs = []OperatorInput{
	OperatorInputNumber,
	OperatorInputDate,
	OperatorInputText,
	OperatorInputTruth,
}

// Valid reports whether exactly one field is set, and that it is the field the
// type asks for. A value with two fields, or with none, is not something a
// reader produced, so an implementation is never handed one.
func (v OperatorValue) Valid() bool {
	set := 0
	if v.Number != nil {
		set++
	}
	if v.Date != nil {
		set++
	}
	if v.Text != nil {
		set++
	}
	if v.Truth != nil {
		set++
	}
	if v.Any != nil {
		if !v.Any.Valid() {
			return false
		}
		set++
	}
	if v.Presence != nil {
		set++
	}
	return set == 1
}

// Valid reports whether this value holds exactly one field and that the field
// matches its type.
func (v AnyValue) Valid() bool {
	switch v.Type {
	case OperatorInputNumber:
		return v.Number != nil && v.Date == nil && v.Text == nil && v.Truth == nil
	case OperatorInputDate:
		return v.Date != nil && v.Number == nil && v.Text == nil && v.Truth == nil
	case OperatorInputText:
		return v.Text != nil && v.Number == nil && v.Date == nil && v.Truth == nil
	case OperatorInputTruth:
		return v.Truth != nil && v.Number == nil && v.Date == nil && v.Text == nil
	}
	return false
}

// OperatorValueKind is the type an input's value was read as, or the empty
// input when no value was read at all.
func (v OperatorValue) Kind() OperatorInput {
	switch {
	case v.Number != nil:
		return OperatorInputNumber
	case v.Date != nil:
		return OperatorInputDate
	case v.Text != nil:
		return OperatorInputText
	case v.Truth != nil:
		return OperatorInputTruth
	case v.Any != nil:
		return OperatorInputAny
	case v.Presence != nil:
		return OperatorInputPresence
	}
	return ""
}

// ParseOperatorInputValue reads the text an example table writes for one cell,
// in the type the declaration gave. Section 8.3 fixes how each type is
// written:
//
//	number  十進の数字                   100, -0.01
//	date    YYYY-MM-DD                   2026-04-01
//	text    二重引用符で囲んだ文字          "A-100"
//	truth   語 true または false          true
//	any     上のどれか。引用符の有無で    100, true, "100"
//	        他の型と区別する
//
// The text is kept as written where it can be, so no precision or spelling is
// invented. A cell that is not the type it was read as is refused with the
// text it holds, so the author can find the cell.
func ParseOperatorInputValue(input OperatorInput, text string) (OperatorValue, error) {
	switch input {
	case OperatorInputNumber:
		value, err := ParseOperatorCaseValue(text)
		if err != nil {
			return OperatorValue{}, err
		}
		return OperatorValue{Number: value}, nil
	case OperatorInputDate:
		date, err := ParseCalendarDate(text)
		if err != nil {
			return OperatorValue{}, err
		}
		return OperatorValue{Date: &date}, nil
	case OperatorInputText:
		value, err := ParseOperatorTextValue(text)
		if err != nil {
			return OperatorValue{}, err
		}
		return OperatorValue{Text: &value}, nil
	case OperatorInputTruth:
		value, err := ParseOperatorTruthValue(text)
		if err != nil {
			return OperatorValue{}, err
		}
		return OperatorValue{Truth: &value}, nil
	case OperatorInputAny:
		value, err := parseAnyValue(text)
		if err != nil {
			return OperatorValue{}, err
		}
		return OperatorValue{Any: value}, nil
	case OperatorInputPresence:
		// A presence cell states whether the attribute holds a value, in the two
		// words the example table writes. It states no value of another type.
		switch text {
		case OperatorCasePresencePresent:
			held := true
			return OperatorValue{Presence: &held}, nil
		case OperatorCasePresenceAbsent:
			held := false
			return OperatorValue{Presence: &held}, nil
		}
		return OperatorValue{}, fmt.Errorf(
			"%q is not a presence cell; write %s or %s",
			text, OperatorCasePresencePresent, OperatorCasePresenceAbsent)
	}
	return OperatorValue{}, fmt.Errorf("%q is not a value of an input this program reads", text)
}

// parseAnyValue reads one value of an input declared `any`. The quoting
// decides: a quoted cell is a text, and an unquoted cell is the first of the
// remaining types that reads it. A cell two types read equally is refused
// rather than resolved by position (section 8.3).
func parseAnyValue(text string) (*AnyValue, error) {
	if strings.HasPrefix(text, `"`) {
		value, err := ParseOperatorTextValue(text)
		if err != nil {
			return nil, err
		}
		return &AnyValue{Type: OperatorInputText, Text: &value}, nil
	}
	matches := make([]AnyValue, 0, 4)
	for _, input := range OperatorInputs {
		value, err := ParseOperatorInputValue(input, text)
		if err != nil {
			continue
		}
		matches = append(matches, anyOf(value))
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%q is not a number, a date, a quoted text, or true or false; write one of those", text)
	case 1:
		one := matches[0]
		return &one, nil
	}
	return nil, fmt.Errorf("%q is read as more than one type, so the type is not written; a value of an `any` input says its type with quotes", text)
}

// anyOf carries one value read for a `any` input into the type-tagged value an
// implementation receives.
func anyOf(value OperatorValue) AnyValue {
	switch {
	case value.Number != nil:
		return AnyValue{Type: OperatorInputNumber, Number: value.Number}
	case value.Date != nil:
		return AnyValue{Type: OperatorInputDate, Date: value.Date}
	case value.Truth != nil:
		return AnyValue{Type: OperatorInputTruth, Truth: value.Truth}
	}
	return AnyValue{}
}

// ParseOperatorTextValue reads a quoted text. The quotes are part of the
// written form (section 8.3): without them a text would be read as a number or
// a truth, and the same cell would state two things. A text is never read with
// an escape sequence, so `"` cannot appear inside one and an unclosed quote is
// refused rather than closed where the author stopped writing.
func ParseOperatorTextValue(text string) (string, error) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", fmt.Errorf("%q is not a text; write it with quotes, for example \"A-100\"", text)
	}
	inner := text[1 : len(text)-1]
	if strings.Contains(inner, `"`) {
		return "", fmt.Errorf("%q holds a quote inside a text; a text cannot hold one, so write it outside an example", text)
	}
	return inner, nil
}

// ParseOperatorTruthValue reads `true` or `false`. No other word is a truth:
// `yes`, `1` and `TRUE` are refused rather than read as the nearest one.
func ParseOperatorTruthValue(text string) (bool, error) {
	switch text {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a truth; write true or false", text)
}
