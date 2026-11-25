package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/syntax"
)

// A type word is offered only when a value shape exists for it. The shape is
// what lets a contract that states the word be decided: a word with no field
// would build a contract nothing can answer (section 8.4).
func TestRuleAuthoringNeverOffersATypeWordItHasNoValueShapeFor(t *testing.T) {
	for _, word := range []string{"date", "text", "truth", "any"} {
		_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start: " + word + "\n")
		var parseErr *syntax.ParseError
		if !errors.As(err, &parseErr) {
			// The word is accepted, which is only correct if the value shape
			// it names exists here.
			kind, ok := syntax.OperatorInputFor(syntax.OperatorInputWord(word))
			if !ok {
				t.Fatalf("%s was accepted but maps to no input kind", word)
			}
			if !validationeval.HasField(kind) {
				t.Fatalf("%s was accepted but has no value field", word)
			}
			continue
		}
		if !strings.Contains(parseErr.Msg, `unknown operator input "`+word+`"`) {
			t.Fatalf("%s was refused for some other reason: %q", word, parseErr.Msg)
		}
	}

	// Without a type the hint must list only words that are accepted. The
	// words it may offer are checked by writing each of them.
	_, missing := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start:\n")
	var parseErr *syntax.ParseError
	if !errors.As(missing, &parseErr) {
		t.Fatalf("a missing type was accepted: %v", missing)
	}
	if !strings.Contains(parseErr.Msg, "accepted types are number, presence") {
		// One list feeds both the hint and the check, so a word cannot be
		// offered here and refused above.
		t.Fatalf("the hint does not list the accepted words: %q", parseErr.Msg)
	}
	for _, offered := range []string{"date", "text", "truth", "any"} {
		if strings.Contains(parseErr.Msg, offered) {
			// The word is offered, so it must have a value shape.
			kind, ok := syntax.OperatorInputFor(syntax.OperatorInputWord(offered))
			if !ok || !validationeval.HasField(kind) {
				t.Fatalf("the hint offers %q, which has no value field: %q", offered, parseErr.Msg)
			}
		}
	}
}

// A word with no value shape is still refused, so the surface does not offer a
// contract nothing can decide.
func TestRuleAuthoringRefusesAWordWithNoValueShape(t *testing.T) {
	_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start: blobbity\n")
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("an unknown type word was accepted: %v", err)
	}
	if !strings.Contains(parseErr.Msg, `unknown operator input "blobbity"`) {
		t.Fatalf("refused for some other reason: %q", parseErr.Msg)
	}
	for _, word := range []string{"number", "presence", "date", "text", "truth", "any"} {
		if strings.Contains(parseErr.Msg, `"`+word+`"`) {
			t.Fatalf("the refusal does not name the accepted words: %q", parseErr.Msg)
		}
	}
	// The hint the same refusal carries lists every word the surface accepts.
	if !strings.Contains(parseErr.Msg, "accepted inputs are number, presence, date, text, truth, any") {
		t.Fatalf("the refusal does not list the accepted words: %q", parseErr.Msg)
	}
}

// A text value and a number that spell the same characters are two values.
func TestATextValueAndANumberAreDifferentValues(t *testing.T) {
	text := validationeval.OperatorValue{}
	if err := text.Set(semantic.OperatorInputText, "100"); err != nil {
		t.Fatalf("a text value was refused: %v", err)
	}
	number := validationeval.OperatorValue{}
	if err := number.Set(semantic.OperatorInputNumber, text); err == nil {
		// The wrong shape must be refused rather than silently read.
		t.Fatal("a text value was accepted as a number")
	}
	if !text.Stated() || number.Stated() {
		t.Fatalf("text.Stated()=%v number.Stated()=%v, want true, false", text.Stated(), number.Stated())
	}
}

// An input nobody read carries no value, and it is not the zero of a type.
func TestAnUnreadInputCarriesNoValue(t *testing.T) {
	unread := validationeval.Unread()
	if unread.Stated() {
		t.Fatal("an unread input carries a value")
	}
	if unread.Number != nil || unread.Text != nil || unread.Truth != nil || unread.Date != nil {
		t.Fatalf("unread=%+v, want every field nil", unread)
	}
}

// A word with no value field is refused rather than silently dropped.
func TestAShapeWithNoFieldIsRefused(t *testing.T) {
	value := validationeval.OperatorValue{}
	if err := value.Set("blobbity", "x"); err == nil {
		t.Fatal("an input type with no field was accepted")
	}
}
