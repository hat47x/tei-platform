package semantic

import "testing"

// A cell is read as the value its notation states (section 8.3). The type the
// input declares decides which of those readings is the right one, and `any`
// takes the reading the cell was written as.
func TestOperatorCaseCellIsReadAsWritten(t *testing.T) {
	for _, tc := range []struct {
		text string
		kind OperatorCaseCellKind
	}{
		{"100", OperatorCaseCellNumber},
		{"-0.01", OperatorCaseCellNumber},
		{"2026-04-01", OperatorCaseCellDate},
		{`"A-100"`, OperatorCaseCellText},
		{`""`, OperatorCaseCellText},
		{"true", OperatorCaseCellTruth},
		{"false", OperatorCaseCellTruth},
	} {
		t.Run(tc.text, func(t *testing.T) {
			cell, err := ParseOperatorCaseCell(tc.text)
			if err != nil {
				t.Fatalf("%q was refused: %v", tc.text, err)
			}
			if cell.Kind != tc.kind {
				t.Fatalf("%q is a %s cell, want %s", tc.text, cell.Kind, tc.kind)
			}
		})
	}
}

// A cell that none of the types can state is refused rather than approximated.
func TestOperatorCaseCellRefusals(t *testing.T) {
	for _, text := range []string{"1e5", "1,000", "+1", "007", ".5", "5.", "none", `"A-100`, `"A"100"`, "2026-13-01", "2026-04-1", "TRUE", "yes", ""} {
		t.Run(text, func(t *testing.T) {
			if _, err := ParseOperatorCaseCell(text); err == nil {
				t.Fatalf("%q was accepted as a cell", text)
			}
		})
	}
}

// The number 100 and the text "100" are two values of one `any` input, not one
// written twice.
func TestAnAnyInputKeepsTheTypeItsCellStates(t *testing.T) {
	number, err := ParseOperatorCaseCell("100")
	if err != nil {
		t.Fatal(err)
	}
	text, err := ParseOperatorCaseCell(`"100"`)
	if err != nil {
		t.Fatal(err)
	}
	if SameOperatorCaseCell(number, text) {
		t.Fatal(`the number 100 and the text "100" were read as one value`)
	}
	if !SameOperatorCaseCell(number, number) {
		t.Fatal("the number 100 was not read as itself")
	}
	if !SameOperatorCaseCell(text, text) {
		t.Fatal(`the text "100" was not read as itself`)
	}
	if !SameOperatorCaseValue("0", "0.0") {
		t.Fatal("0 and 0.0 were not read as the same number")
	}
	if SameOperatorCaseValue("0", `"0"`) {
		t.Fatal(`0 and "0" were read as the same value`)
	}
}

// A cell fits only the type it states, and `any` fits every one of them.
func TestACellFitsOnlyItsOwnType(t *testing.T) {
	for _, tc := range []struct {
		text string
		fits []OperatorInput
	}{
		{"100", []OperatorInput{OperatorInputNumber, OperatorInputAny}},
		{"2026-04-01", []OperatorInput{OperatorInputDate, OperatorInputAny}},
		{`"A-100"`, []OperatorInput{OperatorInputText, OperatorInputAny}},
		{"true", []OperatorInput{OperatorInputTruth, OperatorInputAny}},
	} {
		cell, err := ParseOperatorCaseCell(tc.text)
		if err != nil {
			t.Fatal(err)
		}
		for _, kind := range []OperatorInput{OperatorInputNumber, OperatorInputDate, OperatorInputText, OperatorInputTruth, OperatorInputAny, OperatorInputPresence} {
			want := false
			for _, fits := range tc.fits {
				if fits == kind {
					want = true
				}
			}
			if got := cell.Fits(kind); got != want {
				t.Fatalf("%q fits %s = %v, want %v", tc.text, kind, got, want)
			}
		}
	}
}

// A cell of another type is refused with a message that names the input's type
// and what may be written.
func TestCheckOperatorCaseCellFitsNamesTheType(t *testing.T) {
	// A cell that states a value of another type is refused with the message
	// for the type the input declares.
	for _, tc := range []struct {
		text string
		kind OperatorInput
		want string
	}{
		{"100", OperatorInputText, "between double quotes"},
		{`"A-100"`, OperatorInputNumber, "decimal number"},
		{"2026-04-01", OperatorInputNumber, "decimal number"},
		{"true", OperatorInputNumber, "decimal number"},
		{"100", OperatorInputDate, "YYYY-MM-DD"},
		{"true", OperatorInputDate, "YYYY-MM-DD"},
		{"100", OperatorInputTruth, "the word true or the word false"},
		{`"A-100"`, OperatorInputTruth, "the word true or the word false"},
	} {
		err := CheckOperatorCaseCellFits(tc.text, tc.kind)
		if err == nil {
			t.Fatalf("%q was accepted for a %s input", tc.text, tc.kind)
		}
		if !contains(err.Error(), tc.want) {
			t.Fatalf("message=%q, want it to name %q", err.Error(), tc.want)
		}
		if !contains(err.Error(), string(tc.kind)) {
			t.Fatalf("message=%q, want it to name the type %s", err.Error(), tc.kind)
		}
	}
	// A cell that states no value at all is refused with the notation, so the
	// author learns what a cell may hold whatever the input's type is.
	for _, tc := range []struct{ text, want string }{
		{"abc", "not a decimal number"},
		{"yes", "not a decimal number"},
		{`"A-100`, "not a decimal number"},
	} {
		err := CheckOperatorCaseCellFits(tc.text, OperatorInputDate)
		if err == nil || !contains(err.Error(), tc.want) {
			t.Fatalf("%q for a date input: %v, want %q", tc.text, err, tc.want)
		}
	}
	if err := CheckOperatorCaseCellFits("100", OperatorInputNumber); err != nil {
		t.Fatalf("a decimal was refused for a number input: %v", err)
	}
	if err := CheckOperatorCaseCellFits(`"100"`, OperatorInputAny); err != nil {
		t.Fatalf("a quoted cell was refused for an any input: %v", err)
	}
	if err := CheckOperatorCaseCellFits("2026-04-01", OperatorInputAny); err != nil {
		t.Fatalf("a date was refused for an any input: %v", err)
	}
}

func contains(text, want string) bool {
	return len(want) == 0 || len(text) >= len(want) && indexOf(text, want) >= 0
}

func indexOf(text, want string) int {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}
