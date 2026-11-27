package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// §3 of the rule normal form decision: an input's type fixes how the value is
// read and what is refused. A numeric input takes a decimal constant or an
// attribute id, and nothing else.
//
// The expected verdicts below are written from that statement, not from the
// conversion code. A value of the wrong type must be named as a type problem,
// not misreported as a duplicate subject-input binding.
//
// The refusal names the type the declaration states and says how a value of it
// is written. It used to say "takes a number" whatever the declaration stated,
// which told the author of a text input nothing about what to write instead, so
// the message is checked against the declared type rather than against a word
// this surface assumed.
func TestRuleAuthoringRefusesAValueOfTheWrongType(t *testing.T) {
	cases := []struct {
		name, value, want string
	}{
		{name: "decimal constant", value: "100"},
		{name: "negative fraction", value: "-0.5"},
		{name: "attribute id", value: "sales.order_line.unit_price"},
		// A value another type would read is still refused for a number input,
		// because the declaration states the type and not the spelling. A
		// quoted text is not lexed as a rule value yet, so it is refused by the
		// value's own reader rather than reaching this check.
		{name: "a calendar day is not a number", value: "2026-04-01", want: "is read as number"},
		{name: "attribute id with an empty segment", value: "sales..unit_price", want: "is read as number"},
		{name: "attribute id with an uppercase segment", value: "sales.Item.unit_price", want: "is read as number"},
		{name: "attribute id with a trailing dot", value: "sales.item.", want: "is read as number"},
		{name: "a word is not a number", value: "ITEM-A", want: "is read as number"},
		{name: "a bare boolean is not a number", value: "true", want: "is read as number"},
		{name: "leading zero is ambiguous", value: "007", want: "is read as number"},
		{name: "a fraction without digits is not a number", value: "1.", want: "is read as number"},
		{name: "a single segment is not an id", value: "price", want: "is read as number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\noperator sales.op: takes low: number\nstring\n" +
				"rule sales.item.probe:\n  sales.item.amount is sales.op low: " + tc.value + "\n"
			if tc.want == "" {
				// The value type is accepted; the explicit assignment then
				// rebinds the input already supplied by the subject.
				_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
				if len(issues) != 1 || !strings.Contains(issues[0].Msg, "already supplied by the rule subject") {
					t.Fatalf("issues=%+v, want the subject-input rebinding refusal only", issues)
				}
				return
			}
			_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want the type mismatch one", issues)
			}
			if !strings.Contains(issues[0].Msg, tc.want) {
				t.Fatalf("message=%q, want it to mention %q", issues[0].Msg, tc.want)
			}
			// A malformed reference is a value error at the authored bytes,
			// not a later subject-input rebinding error.
			cleanSource := strings.ReplaceAll(src, "string\n", "")
			valueStart := strings.LastIndex(cleanSource, tc.value)
			if valueStart < 0 || issues[0].Span.Start.Offset != valueStart || issues[0].Span.End.Offset != valueStart+len(tc.value) {
				t.Fatalf("diagnostic span=%+v, want value %q at byte %d", issues[0].Span, tc.value, valueStart)
			}
		})
	}
}

func mustParseRuleFile(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	// The literal marker keeps the test source stable if a keyword is added.
	src = strings.ReplaceAll(src, "string\n", "")
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
