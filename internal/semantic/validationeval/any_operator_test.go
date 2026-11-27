package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// anyValueOf builds one value of an `any` input, stating the type it was read
// as, which is what decision section 8.4 asks a value to carry.
func anyValueOf(value any) semantic.OperatorValue {
	switch typed := value.(type) {
	case *big.Rat:
		return semantic.OperatorValue{Any: &semantic.AnyValue{Type: semantic.OperatorInputNumber, Number: typed}}
	case semantic.CalendarDate:
		day := typed
		return semantic.OperatorValue{Any: &semantic.AnyValue{Type: semantic.OperatorInputDate, Date: &day}}
	case string:
		text := typed
		return semantic.OperatorValue{Any: &semantic.AnyValue{Type: semantic.OperatorInputText, Text: &text}}
	case bool:
		truth := typed
		return semantic.OperatorValue{Any: &semantic.AnyValue{Type: semantic.OperatorInputTruth, Truth: &truth}}
	default:
		panic("anyValueOf: unsupported value")
	}
}

// TestTheAnyImplementationComparesWhatTheReaderRead pins decision section 8.4
// for the shipped `tei.any.same-as`: two values are the same when they were read
// as the same type and hold the same value, and a number and a text that spell
// the same characters are different values, so the rule is not satisfied by
// comparing them as one.
func TestTheAnyImplementationComparesWhatTheReaderRead(t *testing.T) {
	number := new(big.Rat)
	number.SetInt64(100)
	day, err := semantic.ParseCalendarDate("2026-04-01")
	if err != nil {
		t.Fatalf("the test day does not parse: %v", err)
	}

	cases := []struct {
		name  string
		value semantic.OperatorValue
		other semantic.OperatorValue
		want  bool
	}{
		{"the same number", anyValueOf(number), anyValueOf(number), true},
		{"a different number", anyValueOf(number), anyValueOf(new(big.Rat).SetInt64(200)), false},
		{"the same day", anyValueOf(day), anyValueOf(day), true},
		{"a different day", anyValueOf(day), anyValueOf(mustDay("2026-04-02")), false},
		{"the same text", anyValueOf("100"), anyValueOf("100"), true},
		{"a different text", anyValueOf("100"), anyValueOf("200"), false},
		{"the same truth", anyValueOf(true), anyValueOf(true), true},
		{"a different truth", anyValueOf(true), anyValueOf(false), false},
		// A number and a text that spell the same characters are two values,
		// which is the distinction section 8.3 makes in an example table and
		// section 8.4 carries into the value an implementation receives.
		{"a number against the text of the same characters", anyValueOf(number), anyValueOf("100"), false},
		{"a day against a number", anyValueOf(day), anyValueOf(number), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			implementation := findBuiltin(t, "tei.any.same-as")
			got, err := implementation.Values(map[string]semantic.OperatorValue{
				"value": tc.value,
				"other": tc.other,
			})
			if err != nil {
				t.Fatalf("the rule was refused: %v", err)
			}
			if got != tc.want {
				t.Fatalf("satisfied=%v, want %v", got, tc.want)
			}
		})
	}
}

// TestTheAnyImplementationRefusesAValueThatStatesNoType pins the other side: an
// input that was not read as a value of any type is refused rather than compared
// as the nearest type.
func TestTheAnyImplementationRefusesAValueThatStatesNoType(t *testing.T) {
	implementation := findBuiltin(t, "tei.any.same-as")
	cases := []struct {
		name   string
		values map[string]semantic.OperatorValue
	}{
		{"no value input", map[string]semantic.OperatorValue{"other": anyValueOf("A-100")}},
		{"no other input", map[string]semantic.OperatorValue{"value": anyValueOf("A-100")}},
		{"a value read as a number", map[string]semantic.OperatorValue{
			"value": {Number: new(big.Rat).SetInt64(7)},
			"other": anyValueOf("7"),
		}},
		{"an empty value", map[string]semantic.OperatorValue{
			"value": {},
			"other": anyValueOf("7"),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := implementation.Values(tc.values)
			if err == nil {
				t.Fatalf("a value that states no type was compared")
			}
		})
	}
}

// findBuiltin returns the shipped implementation under the given name, so a test
// checks what the binary actually carries rather than a copy of it.
func findBuiltin(t *testing.T, id string) validationeval.OperatorImplementation {
	t.Helper()
	for _, named := range validationeval.BuiltinImplementations() {
		if named.ID != id {
			continue
		}
		resolved, err := validationeval.ResolveImplementations(
			[]validationeval.ImplementationSelection{{
				Operator:       core.CanonicalID("sales.op"),
				Implementation: named.ID,
			}},
			[]validationeval.NamedImplementation{named},
			nil,
		)
		if err != nil {
			t.Fatalf("the selection of %s was refused: %v", id, err)
		}
		if len(resolved) != 1 {
			t.Fatalf("the selection of %s resolved to %d implementations", id, len(resolved))
		}
		return resolved[0]
	}
	t.Fatalf("no built-in implementation %s", id)
	return validationeval.OperatorImplementation{}
}

func mustDay(text string) semantic.CalendarDate {
	day, err := semantic.ParseCalendarDate(text)
	if err != nil {
		panic(err)
	}
	return day
}
