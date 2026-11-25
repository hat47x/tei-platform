package semantic_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
)

// A value is written in a spelling, and the spelling states the kind it is read
// as. Decision section 8.3 fixes the spellings: a decimal is a number,
// YYYY-MM-DD is a calendar day, a text in double quotes is a text, and the
// words true and false are the truth words.
//
// The expectations below are the rows of that table, written out by hand.

func TestAValueIsReadInTheSpellingItWasWrittenIn(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want semantic.SpelledValue
	}{
		{
			name: "a decimal is a number",
			text: "100",
			want: semantic.SpelledValue{Text: "100", Kind: semantic.SpelledKindNumber},
		},
		{
			name: "a negative decimal is a number",
			text: "-0.01",
			want: semantic.SpelledValue{Text: "-0.01", Kind: semantic.SpelledKindNumber},
		},
		{
			name: "a calendar day",
			text: "2026-04-01",
			want: semantic.SpelledValue{Text: "2026-04-01", Kind: semantic.SpelledKindDate},
		},
		{
			name: "a text in double quotes",
			text: `"A-100"`,
			want: semantic.SpelledValue{Text: "A-100", Kind: semantic.SpelledKindText},
		},
		{
			name: "an empty text is a text of length zero",
			text: `""`,
			want: semantic.SpelledValue{Text: "", Kind: semantic.SpelledKindText},
		},
		{
			name: "the truth word true",
			text: "true",
			want: semantic.SpelledValue{Text: "true", Kind: semantic.SpelledKindTruth},
		},
		{
			name: "the truth word false",
			text: "false",
			want: semantic.SpelledValue{Text: "false", Kind: semantic.SpelledKindTruth},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := semantic.ReadSpelledValue(tc.text)
			if !ok {
				t.Fatalf("%q was not read as a spelled value", tc.text)
			}
			if got != tc.want {
				t.Fatalf("%q read as %+v, want %+v", tc.text, got, tc.want)
			}
		})
	}
}

// A text that spells no value is refused, and an empty text is the absence of a
// value rather than a value of length zero.
func TestATextThatSpellsNoValueIsRefused(t *testing.T) {
	for _, text := range []string{"", "1e3", "0x10", "1/2", "1_000", "2026-04", "2026-13-01", `"unclosed`, "yes", "TRUE"} {
		if _, ok := semantic.ReadSpelledValue(text); ok {
			t.Fatalf("%q was read as a spelled value", text)
		}
		if semantic.IsSpelledValue(text) {
			t.Fatalf("%q was reported as a spelled value", text)
		}
	}
}

// The value of each spelling is read with the reader that kind uses, and asking
// for another kind's value is refused rather than approximated.
func TestEachSpellingIsReadAsItsOwnKind(t *testing.T) {
	number, _ := semantic.ReadSpelledValue("-0.01")
	value, err := number.NumberOf()
	if err != nil || value.Cmp(big.NewRat(-1, 100)) != 0 {
		t.Fatalf("number=%v err=%v, want -0.01", value, err)
	}
	if _, err := number.DateOf(); err == nil {
		t.Fatal("a number was read as a calendar day")
	}

	date, _ := semantic.ReadSpelledValue("2026-04-01")
	day, err := date.DateOf()
	if err != nil || (day != semantic.CalendarDate{Year: 2026, Month: 4, Day: 1}) {
		t.Fatalf("day=%+v err=%v, want 2026-04-01", day, err)
	}
	if _, err := date.NumberOf(); err == nil {
		t.Fatal("a calendar day was read as a number")
	}

	text, _ := semantic.ReadSpelledValue(`"A-100"`)
	inner, err := text.TextOf()
	if err != nil || inner != "A-100" {
		t.Fatalf("text=%q err=%v, want A-100 (without its quotes)", inner, err)
	}

	truth, _ := semantic.ReadSpelledValue("true")
	yes, err := truth.TruthOf()
	if err != nil || !yes {
		t.Fatalf("truth=%v err=%v, want true", yes, err)
	}
}

// A number and a text that spell alike are two values, not one written twice:
// the quotes are part of the spelling, and a text keeps them out of its value.
func TestANumberAndATextOfTheSameSpellingAreTwoValues(t *testing.T) {
	number, _ := semantic.ReadSpelledValue("100")
	text, _ := semantic.ReadSpelledValue(`"100"`)
	if number.Kind == text.Kind {
		t.Fatalf("%q and %q were read as the same kind %q", number.Text, text.Text, number.Kind)
	}
	if inner, err := text.TextOf(); err != nil || inner != "100" {
		t.Fatalf("text=%q err=%v, want 100 without its quotes", inner, err)
	}
	if value, err := number.NumberOf(); err != nil || value.Cmp(big.NewRat(100, 1)) != 0 {
		t.Fatalf("number=%v err=%v, want 100", value, err)
	}
}
