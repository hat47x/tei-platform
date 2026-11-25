package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The unnamed form declares no name, and the named form declares the
// author's. A reader applies "value" to the unnamed one; the parsed
// contract must not claim it. §12 of the rule normal form decision writes the
// name before ':' and the type word after it.
func TestRuleAuthoringReadsANamedOperatorInput(t *testing.T) {
	for _, tc := range []struct {
		declaration string
		wantName    string
	}{
		{"takes number", ""},
		{"takes value: number", "value"},
		{"takes low: number", "low"},
	} {
		t.Run(tc.declaration, func(t *testing.T) {
			src := "namespace sales\noperator sales.op: " + tc.declaration + "\n"
			anchored, err := syntax.ParseRuleAuthoringAnchored(src)
			if err != nil {
				t.Fatalf("%q did not parse: %v", tc.declaration, err)
			}
			if len(anchored.Value.Operators) != 1 {
				t.Fatalf("operators=%+v", anchored.Value.Operators)
			}
			got := anchored.Value.Operators[0]
			if got.InputName != tc.wantName {
				t.Fatalf("inputName=%q, want %q", got.InputName, tc.wantName)
			}
			if got.Input != "number" {
				t.Fatalf("input=%q, want number", got.Input)
			}
		})
	}
}

// The hint that tells an author what to write must not name a word this
// surface refuses. Before this, leaving the type out said "accepted types are
// number, date, text, truth, any, presence", and writing `date` next answered
// `unknown operator input "date"; accepted inputs are number, presence`. An
// author who follows the hint lands on a second error.
//
// date, text, truth and any now have readers (decision sections 8.3 and 8.4),
// so the hint offers them and they are accepted. The invariant this pins is
// the one the hint broke: every word it names is a word the surface reads, and
// a word outside the closed list is still refused.
func TestRuleAuthoringNeverOffersATypeWordItRefuses(t *testing.T) {
	_, missing := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start:\n")
	var parseErr *syntax.ParseError
	if !errors.As(missing, &parseErr) {
		t.Fatalf("a missing type was accepted: %v", missing)
	}
	const prefix = "accepted types are "
	at := strings.Index(parseErr.Msg, prefix)
	if at < 0 {
		t.Fatalf("the hint does not list the accepted words: %q", parseErr.Msg)
	}
	// One list feeds both the hint and the check, so a word cannot be offered
	// here and refused above. Each offered word is written to be sure.
	for _, offered := range strings.Split(parseErr.Msg[at+len(prefix):], ", ") {
		if _, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start: " + offered + "\n"); err != nil {
			t.Fatalf("the hint offers %q but it is refused: %v", offered, err)
		}
	}

	// A word outside the closed list is refused, and named as unknown rather
	// than answered by something else.
	for _, refused := range []string{"keys", "Number", "presences"} {
		_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: takes start: " + refused + "\n")
		var unknown *syntax.ParseError
		if !errors.As(err, &unknown) {
			t.Fatalf("%s was accepted although it is not an input type: %v", refused, err)
		}
		if !strings.Contains(unknown.Msg, `unknown operator input "`+refused+`"`) {
			t.Fatalf("%s was refused for some other reason: %q", refused, unknown.Msg)
		}
	}
}

// A name without a type, a type outside the closed list, and a second input are
// all refused. An operator with an unreadable signature would answer rules
// against a contract nobody can check.
func TestRuleAuthoringRefusesIncompleteOrUnknownNamedInputs(t *testing.T) {
	for _, tc := range []struct {
		name, tail, want string
	}{
		{"name without a type", "takes low:\n", "expected the type of input"},
		{"type outside the closed list", "takes low: keys\n", `unknown operator input "keys"`},
		{"second input", "takes low: number high: number\n", `unexpected "high" after the operator name`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: " + tc.tail)
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("%s was accepted or unlocated: %v", tc.name, err)
			}
			if !strings.Contains(parseErr.Msg, tc.want) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.want)
			}
		})
	}
}
