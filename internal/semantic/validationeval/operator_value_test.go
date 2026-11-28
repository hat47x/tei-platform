package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/syntax"
)

// The value carrier of section 8.4 carries the type a reader read a value as,
// so the type is what the reader decides, not what the implementation is handed
// to sort out. This file pins the reading of one example in each type and the
// two refusals that keep the types apart.
//
// The expectations are written from section 8.3's table, not read back from the
// reader.

// exampleValue reads one cell in the type its input is declared with. It is the
// reading the reader for an example table applies before an implementation is
// asked, so a table that states a value of the wrong kind is refused at the
// cell rather than at the implementation.
func exampleValue(t *testing.T, input semantic.OperatorInput, cell string) (semantic.OperatorValue, error) {
	t.Helper()
	return semantic.ParseOperatorInputValue(input, cell)
}

// TestAnExampleTableCellIsReadInItsInputType pins the four ways section 8.3
// says a cell is written: a number as digits, a date as YYYY-MM-DD, a text in
// quotes, and a truth as the word true or false.
func TestAnExampleTableCellIsReadInItsInputType(t *testing.T) {
	for _, tc := range []struct {
		input semantic.OperatorInput
		cell  string
		want  semantic.OperatorInput
	}{
		{semantic.OperatorInputNumber, "100", semantic.OperatorInputNumber},
		{semantic.OperatorInputNumber, "-0.01", semantic.OperatorInputNumber},
		{semantic.OperatorInputDate, "2026-04-01", semantic.OperatorInputDate},
		{semantic.OperatorInputText, `"A-100"`, semantic.OperatorInputText},
		{semantic.OperatorInputTruth, "true", semantic.OperatorInputTruth},
		{semantic.OperatorInputTruth, "false", semantic.OperatorInputTruth},
	} {
		t.Run(string(tc.input)+" "+tc.cell, func(t *testing.T) {
			got, err := exampleValue(t, tc.input, tc.cell)
			if err != nil {
				t.Fatalf("the cell was refused: %v", err)
			}
			if got.Kind() != tc.want {
				t.Fatalf("the cell was read as %s, want %s", got.Kind(), tc.want)
			}
			if !got.Valid() {
				t.Fatalf("the value carries more than one type: %+v", got)
			}
		})
	}
}

// TestAnExampleTableCellRefusesAValueOfAnotherType pins that a cell is not read
// as the nearest value of its input's type: a date is not a number, a text is
// not read without its quotes, and a truth is not read from yes or 1.
func TestAnExampleTableCellRefusesAValueOfAnotherType(t *testing.T) {
	for _, tc := range []struct {
		input semantic.OperatorInput
		cell  string
	}{
		{semantic.OperatorInputDate, "2026-4-1"},
		{semantic.OperatorInputDate, "20260401"},
		{semantic.OperatorInputText, "A-100"},
		{semantic.OperatorInputText, `"A-100`},
		{semantic.OperatorInputTruth, "yes"},
		{semantic.OperatorInputTruth, "1"},
		{semantic.OperatorInputTruth, "TRUE"},
		{semantic.OperatorInputNumber, "100%"},
	} {
		t.Run(string(tc.input)+" "+tc.cell, func(t *testing.T) {
			if _, err := exampleValue(t, tc.input, tc.cell); err == nil {
				t.Fatalf("%q was read as a %s", tc.cell, tc.input)
			}
		})
	}
}

// TestAnAnyInputKeepsTheTypeItWasReadAs pins the reason section 8.4 gives for
// one value type: a number and a quoted text are different values, and the
// quoting is what tells them apart. The two are never read as the same value.
func TestAnAnyInputKeepsTheTypeItWasReadAs(t *testing.T) {
	number, err := exampleValue(t, semantic.OperatorInputAny, "100")
	if err != nil {
		t.Fatalf("a number was refused: %v", err)
	}
	text, err := exampleValue(t, semantic.OperatorInputAny, `"100"`)
	if err != nil {
		t.Fatalf("a quoted text was refused: %v", err)
	}
	if number.Kind() != semantic.OperatorInputAny {
		t.Fatalf("a number was read as %s, want any", number.Kind())
	}
	if text.Kind() != semantic.OperatorInputAny {
		t.Fatalf("a quoted text was read as %s, want any", text.Kind())
	}
	if number.Any == nil || text.Any == nil {
		t.Fatalf("the value does not carry its type: %+v %+v", number, text)
	}
	if number.Any.Type == text.Any.Type {
		t.Fatalf("both values were read as %s", number.Any.Type)
	}
	if number.Any.Type != semantic.OperatorInputNumber {
		t.Fatalf("the unquoted value was read as %s, want number", number.Any.Type)
	}
	if text.Any.Type != semantic.OperatorInputText {
		t.Fatalf("the quoted value was read as %s, want text", text.Any.Type)
	}
	if *text.Any.Text != "100" {
		t.Fatalf("the text value is %q, want 100", *text.Any.Text)
	}
}

// TestAnAnyInputRefusesAValueNoTypeReads pins that a value of an `any` input
// is not read by dropping what does not fit. A cell two types read equally is
// refused rather than resolved by position.
func TestAnAnyInputRefusesAValueNoTypeReads(t *testing.T) {
	for _, cell := range []string{"100%", `"`, `"a"b"`, "", "true false"} {
		t.Run(cell, func(t *testing.T) {
			if _, err := exampleValue(t, semantic.OperatorInputAny, cell); err == nil {
				t.Fatalf("%q was read as a value of any type", cell)
			}
		})
	}
}

// An empty text is a value, and a cell that holds no value is not. Section 8.3
// keeps the two apart, so a text input can state both.
func TestAnEmptyTextIsNotTheSameAsNoValue(t *testing.T) {
	value, err := exampleValue(t, semantic.OperatorInputText, `""`)
	if err != nil {
		t.Fatalf("an empty text was refused: %v", err)
	}
	if value.Text == nil || *value.Text != "" {
		t.Fatalf("the empty text was read as %+v", value)
	}
	if _, err := exampleValue(t, semantic.OperatorInputText, ""); err == nil {
		t.Fatal("an empty cell was read as an empty text")
	}
}

// A presence input reads whether a value is there, so an example table cannot
// state a value for it: the cell would say which value, which is not the
// question the input asks.
func TestAPresenceInputTakesNoValueInAnExample(t *testing.T) {
	if _, err := exampleValue(t, semantic.OperatorInputPresence, "7"); err == nil {
		t.Fatal("a presence input was read a value")
	}
}

// A presence input's cell states one of the two words the design gives it, and
// the value it produces answers the presence question rather than carrying a
// value of some type. The words are read here, so an example table is checked
// against the same reading the rule path uses.
func TestAPresenceCellStatesWhetherTheValueIsThere(t *testing.T) {
	for _, tc := range []struct {
		word string
		held bool
	}{
		{semantic.OperatorCasePresencePresent, true},
		{semantic.OperatorCasePresenceAbsent, false},
	} {
		t.Run(tc.word, func(t *testing.T) {
			got, err := exampleValue(t, semantic.OperatorInputPresence, tc.word)
			if err != nil {
				t.Fatalf("the word was refused: %v", err)
			}
			if got.Presence == nil || *got.Presence != tc.held {
				t.Fatalf("the cell was read as %+v, want presence=%v", got, tc.held)
			}
			if got.Kind() != semantic.OperatorInputPresence {
				t.Fatalf("the cell was read as %s, want presence", got.Kind())
			}
		})
	}
	// A word that is neither of the two is refused and the refusal names both.
	_, err := exampleValue(t, semantic.OperatorInputPresence, "yes")
	if err == nil {
		t.Fatal("an unknown presence word was read")
	}
	for _, word := range []string{semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent} {
		if !strings.Contains(err.Error(), word) {
			t.Fatalf("the refusal does not name %q: %v", word, err)
		}
	}
}

// A value that carries two types is refused rather than read as one. An
// implementation must never have to ask which kind it holds.
func TestAValueCarryingTwoTypesIsRefused(t *testing.T) {
	both := semantic.OperatorValue{Number: new(big.Rat), Text: strPtr("7")}
	if both.Valid() {
		t.Fatal("a value carrying a number and a text was accepted")
	}
	empty := semantic.OperatorValue{}
	if empty.Valid() {
		t.Fatal("a value carrying no type was accepted")
	}
}

func strPtr(text string) *string { return &text }

// A value of an `any` input keeps the type it was read as, and a value that
// names one type and holds another is refused.
func TestAnAnyValueCarriesTheTypeItHolds(t *testing.T) {
	text := strPtr("7")
	value := semantic.AnyValue{Type: semantic.OperatorInputText, Text: text}
	if !value.Valid() {
		t.Fatal("a text value was refused")
	}
	wrong := semantic.AnyValue{Type: semantic.OperatorInputText, Number: new(big.Rat)}
	if wrong.Valid() {
		t.Fatal("a value that names a text and holds a number was accepted")
	}
}

// A rule whose operator declares a presence input is still refused, because
// the reader reads values and not whether there is one.
func TestAPresenceInputRuleIsStillRefused(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputPresence}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op",
		Input:    semantic.OperatorInputPresence,
		Number:   func(*big.Rat) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.op",
	}
	_, err := validationeval.Decide(rule, refusingTypeSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a presence input was read as a value")
	}
}

// The four type words are accepted by the surface, so a contract that declares
// one is no longer refused at conversion. The refusal used to name every type
// as `presence`; it now names its own type.
func TestTheFourTypeWordsAreAcceptedByTheSurface(t *testing.T) {
	for _, word := range []string{"number", "date", "text", "truth", "any", "presence"} {
		if !semantic.OperatorInput(word).Valid() {
			t.Fatalf("%s is not a known operator input", word)
		}
	}
	for _, word := range []string{"number", "date", "text", "truth", "any", "presence"} {
		if _, ok := syntax.OperatorInputFor(syntax.OperatorInputWord(word)); !ok {
			t.Fatalf("%s is not an accepted input word", word)
		}
	}
}
