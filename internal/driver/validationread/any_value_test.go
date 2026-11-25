package validationread

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
)

// A value of an `any` input is read as the type the driver handed it back in,
// and it keeps that type on the way to the implementation (section 8.4). The
// driver's own Go type decides, so no guessing is involved: a driver's `int64`
// is a number and its `bool` is a truth.
func TestAnyValueKeepsTheTypeTheDriverHandedBack(t *testing.T) {
	day, err := semantic.ParseCalendarDate("2026-04-01")
	if err != nil {
		t.Fatalf("a calendar date was refused: %v", err)
	}
	for _, tc := range []struct {
		name  string
		value any
		want  semantic.OperatorInput
	}{
		{"a rational number", big.NewRat(7, 1), semantic.OperatorInputNumber},
		{"a calendar date", day, semantic.OperatorInputDate},
		{"a truth", true, semantic.OperatorInputTruth},
		{"a text", "LOT/A", semantic.OperatorInputText},
		{"the bytes of a text column", []byte("LOT/A"), semantic.OperatorInputText},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := anyValue(tc.value)
			if err != nil {
				t.Fatalf("a value was refused: %v", err)
			}
			if got.Type != tc.want {
				t.Fatalf("type=%q, want %q", got.Type, tc.want)
			}
			if !got.Valid() {
				t.Fatalf("the value is not one value of one type: %+v", got)
			}
		})
	}
}

// A text is the one case where the driver's own type does not decide, because a
// driver may hand back the text of a column for any of the types. A text that
// reads as a number or as a truth is refused rather than resolved by picking
// one, which is the same refusal section 8.3 makes for an example cell: a reader
// that resolved it would decide a rule from a type the author did not state.
func TestAnyValueRefusesATextTwoTypesRead(t *testing.T) {
	for _, text := range []string{"7", "true", "-0.01"} {
		_, err := anyValue(text)
		if err == nil {
			t.Fatalf("%q was read as one type", text)
		}
		if !contains(err.Error(), "read as both") {
			t.Fatalf("the refusal does not name the ambiguity: %v", err)
		}
	}
	// The bytes of the same column are refused the same way, so the two
	// spellings a driver may use do not decide the type.
	for _, text := range []string{"7", "true"} {
		_, err := anyValue([]byte(text))
		if err == nil {
			t.Fatalf("%q was read as one type", text)
		}
	}
}

// A text no other type reads is a text, and it keeps its own type even when it
// spells something a number could not.
func TestAnyValueReadsATextNoOtherTypeReads(t *testing.T) {
	got, err := anyValue("LOT/A")
	if err != nil {
		t.Fatalf("a text was refused: %v", err)
	}
	if got.Type != semantic.OperatorInputText || got.Text == nil || *got.Text != "LOT/A" {
		t.Fatalf("value=%+v, want the text LOT/A", got)
	}
}

// A value of a Go type this program does not read is refused rather than turned
// into the nearest type.
func TestAnyValueRefusesAGoTypeItDoesNotRead(t *testing.T) {
	for _, value := range []any{float64(0.5), nil, struct{}{}} {
		if _, err := anyValue(value); err == nil {
			t.Fatalf("a %T was read as a value of some type", value)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
