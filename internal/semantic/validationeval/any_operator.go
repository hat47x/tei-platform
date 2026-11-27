package validationeval

import (
	"fmt"

	"tei/internal/semantic"
)

// tei.any.same-as decides "the value holds exactly the other value", for inputs
// declared `any`. A value of an `any` input keeps the type it was read as
// (decision section 8.4), so this implementation compares the two values as the
// values they were read as: a number and a text that spell the same characters
// are different values and are never the same.
//
// The reader reports each value's type, so this implementation never chooses a
// type to compare in: it refuses a value that states no type, because answering
// from a value nobody read would answer a question about a different value.
func anySameAs(values map[string]semantic.OperatorValue) (bool, error) {
	value, ok := values["value"]
	if !ok {
		return false, fmt.Errorf("same-as reads an input named \"value\"; the rule supplied no value input")
	}
	other, ok := values["other"]
	if !ok {
		return false, fmt.Errorf("same-as reads an input named \"other\"; the rule supplied no other input")
	}
	// An `any` value carries the type it was read as beside the value, so the
	// comparison reads the kind from there rather than choosing one.
	if value.Any == nil || !value.Any.Valid() {
		return false, fmt.Errorf("same-as reads a value of any type; its input was not read as one")
	}
	if other.Any == nil || !other.Any.Valid() {
		return false, fmt.Errorf("same-as reads a value of any type; the other input was not read as one")
	}
	if value.Any.Type != other.Any.Type {
		return false, nil
	}
	switch value.Any.Type {
	case semantic.OperatorInputNumber:
		return value.Any.Number.Cmp(other.Any.Number) == 0, nil
	case semantic.OperatorInputDate:
		return value.Any.Date.Compare(*other.Any.Date) == 0, nil
	case semantic.OperatorInputText:
		return *value.Any.Text == *other.Any.Text, nil
	case semantic.OperatorInputTruth:
		return *value.Any.Truth == *other.Any.Truth, nil
	default:
		return false, fmt.Errorf("same-as cannot compare a value read as %s", value.Any.Type)
	}
}
