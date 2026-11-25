package validationread_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic/validationeval"
)

// A text is read as a text and a truth as a truth. The reader reports what the
// row holds, and never reads one for the other: the text "7" is not the number
// 7, and the truth true is not the number 1.
//
// The empty text is a value of length zero, which is why it is not the same
// state as a row that holds no value (decision section 8.3).

func TestARowReadsATextAndATruthAsWhatTheyAre(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{
		labelID:  "A-100", // present
		activeID: true,    // present
		// seatID is not held: the query did not return it
	}))
	one, err := detail.Row(0)
	if err != nil {
		t.Fatal(err)
	}

	text, err := one.ReadText(labelID)
	if err != nil || text != (validationeval.TextReading{State: validationeval.ValuePresent, Value: "A-100"}) {
		t.Fatalf("text=%+v err=%v", text, err)
	}
	truth, err := one.ReadTruth(activeID)
	if err != nil || truth != (validationeval.TruthReading{State: validationeval.ValuePresent, Value: true}) {
		t.Fatalf("truth=%+v err=%v", truth, err)
	}
}

// The three states are the same for a text and a truth as for every other type,
// and the empty text is one of them rather than a fourth.
func TestATextAndATruthCarryTheSameThreeStates(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{labelID: "", activeID: false}), // the empty text and a false truth are values
		row(map[core.CanonicalID]any{labelID: nil, activeID: nil}),  // the record holds no value
		row(map[core.CanonicalID]any{}),                             // the query did not return it
	)
	for index, want := range []validationeval.ValueState{
		validationeval.ValuePresent, validationeval.ValueNull, validationeval.ValueNotFetched,
	} {
		reader, err := detail.Row(index)
		if err != nil {
			t.Fatal(err)
		}
		got, err := reader.ReadText(labelID)
		if err != nil || got.State != want {
			t.Fatalf("row %d: text=%+v err=%v, want %s", index, got, err, want)
		}
		gotTruth, err := reader.ReadTruth(activeID)
		if err != nil || gotTruth.State != want {
			t.Fatalf("row %d: truth=%+v err=%v, want %s", index, gotTruth, err, want)
		}
	}
	// The empty text is the value "", and a false truth is the value false.
	first, _ := detail.Row(0)
	if got, _ := first.ReadText(labelID); got.Value != "" {
		t.Fatalf("the empty text was read as %q", got.Value)
	}
	if got, _ := first.ReadTruth(activeID); got.Value {
		t.Fatal("a false truth was read as true")
	}
}

// A text is read only from a text attribute and a truth only from a truth
// attribute. A number is neither, so it is refused rather than read as the
// nearest text.
func TestATextAndATruthAreReadOnlyFromTheirOwnType(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{
		seatID: 3, labelID: "A-100", activeID: true,
	}))
	one, err := detail.Row(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.ReadText(seatID); err == nil {
		t.Fatal("a number was read as a text")
	}
	if _, err := one.ReadTruth(seatID); err == nil {
		t.Fatal("a number was read as a truth")
	}
	// A date is neither a text nor a truth.
	if _, err := one.ReadText(startsOn); err == nil {
		t.Fatal("a date was read as a text")
	}
	if _, err := one.ReadTruth(endsOn); err == nil {
		t.Fatal("a date was read as a truth")
	}
}

// Asking all the rows for the text of one record is refused, as it is for a
// number and a date: a rule about one record reads one row.
func TestAllTheRowsRefuseATextAndATruthOfOneRecord(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{labelID: "A-100", activeID: true}))
	if _, err := detail.ReadText(labelID); err == nil {
		t.Fatal("all the rows were read as one record's text")
	}
	if _, err := detail.ReadTruth(activeID); err == nil {
		t.Fatal("all the rows were read as one record's truth")
	}
}
