package semantic

import (
	"math/big"
	"strings"
)

// A value is written in a spelling, and read as the type its declaration gives
// it. This file holds the spellings a table or a rule may write and the reader
// that turns one into the value and the kind it spells.
//
// The spellings are the ones decision section 8.3 fixes:
//
//	number   a decimal, as it has always been written
//	date     YYYY-MM-DD
//	text     the text in double quotes
//	truth    the word true or the word false
//
// The spelling states the kind; the declaration states which kinds an input
// accepts. A table therefore keeps its cells as written and the signature
// decides what they mean, so a cell of a kind the input does not accept is
// refused at the signature rather than read as a number.

// IsSpelledValue reports whether text is a value written in one of the
// spellings this program reads. An empty text is not a value: it is the absence
// of one, written as an empty cell.
//
// The decimal is read through ParseOperatorCaseValue, the strict decimal
// reading the number inputs already use: an exponent, a base prefix, a grouped
// digit and a fraction are not decimals, so they are not numbers here either.
// A text that reads as both a decimal and a date does not exist, so a spelling
// names one kind.
func IsSpelledValue(text string) bool {
	if text == "" {
		return false
	}
	if _, err := ParseOperatorCaseValue(text); err == nil {
		return true
	}
	if _, err := ParseCalendarDate(text); err == nil {
		return true
	}
	if len(text) >= 2 && strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
		return true
	}
	return text == "true" || text == "false"
}

// SpelledKind is the kind a spelling states.
type SpelledKind string

const (
	// SpelledKindNumber is the kind of a decimal spelling.
	SpelledKindNumber SpelledKind = "number"
	// SpelledKindDate is the kind of a calendar-day spelling.
	SpelledKindDate SpelledKind = "date"
	// SpelledKindText is the kind of a quoted spelling.
	SpelledKindText SpelledKind = "text"
	// SpelledKindTruth is the kind of a truth-word spelling.
	SpelledKindTruth SpelledKind = "truth"
)

// SpelledValue is one value as a table wrote it: the text, and the kind that
// spelling states. The value is read from the text, so a caller that needs it
// asks for the kind it declared.
type SpelledValue struct {
	// Text is the cell exactly as written, quotes included.
	Text string
	// Kind is the kind the spelling states.
	Kind SpelledKind
}

// ReadSpelledValue reads one cell in the spelling it was written in. The second
// return value is false for a text that spells no value.
//
// The text is kept as written: an implementation receives the value of the kind
// its declaration states, and the quotes of a text are not part of the value.
func ReadSpelledValue(text string) (SpelledValue, bool) {
	switch {
	case isSpelledDecimal(text):
		return SpelledValue{Text: text, Kind: SpelledKindNumber}, true
	case isSpelledDate(text):
		return SpelledValue{Text: text, Kind: SpelledKindDate}, true
	case isSpelledText(text):
		return SpelledValue{Text: text[1 : len(text)-1], Kind: SpelledKindText}, true
	case text == "true" || text == "false":
		return SpelledValue{Text: text, Kind: SpelledKindTruth}, true
	}
	return SpelledValue{}, false
}

// NumberOf reads the value of a number spelling. It is the same strict decimal
// reading every number input uses.
func (v SpelledValue) NumberOf() (*big.Rat, error) {
	if v.Kind != SpelledKindNumber {
		return nil, errSpelledKind(v, SpelledKindNumber)
	}
	return ParseOperatorCaseValue(v.Text)
}

// DateOf reads the value of a calendar-day spelling. A moment in time is not
// one, and a day the calendar does not have is refused.
func (v SpelledValue) DateOf() (CalendarDate, error) {
	if v.Kind != SpelledKindDate {
		return CalendarDate{}, errSpelledKind(v, SpelledKindDate)
	}
	return ParseCalendarDate(v.Text)
}

// TruthOf reads the value of a truth-word spelling.
func (v SpelledValue) TruthOf() (bool, error) {
	if v.Kind != SpelledKindTruth {
		return false, errSpelledKind(v, SpelledKindTruth)
	}
	return v.Text == "true", nil
}

// TextOf returns the text of a quoted spelling, without its quotes.
func (v SpelledValue) TextOf() (string, error) {
	if v.Kind != SpelledKindText {
		return "", errSpelledKind(v, SpelledKindText)
	}
	return v.Text, nil
}

func isSpelledDecimal(text string) bool {
	_, err := ParseOperatorCaseValue(text)
	return err == nil
}

func isSpelledDate(text string) bool {
	_, err := ParseCalendarDate(text)
	return err == nil
}

func isSpelledText(text string) bool {
	return len(text) >= 2 && strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`)
}

func errSpelledKind(value SpelledValue, want SpelledKind) error {
	return &spelledKindError{value: value, want: want}
}

type spelledKindError struct {
	value SpelledValue
	want  SpelledKind
}

func (e *spelledKindError) Error() string {
	return e.value.Text + " is spelled as " + string(e.value.Kind) + ", not as " + string(e.want)
}
