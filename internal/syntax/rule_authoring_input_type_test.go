package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// §3 of the rule normal form decision: an input's type fixes how the value is
// read and what is refused. Only `number` exists today, so the check is narrow:
// a numeric input takes a decimal constant or an attribute id, and nothing else.
//
// The expected verdicts below are written from that statement, not from the
// conversion code. A value of the wrong type must be named as a type problem,
// not reported later as an unsupported transport.
func TestRuleAuthoringRefusesAValueOfTheWrongType(t *testing.T) {
	cases := []struct {
		name, value, want string
	}{
		{name: "decimal constant", value: "100"},
		{name: "negative fraction", value: "-0.5"},
		{name: "attribute id", value: "sales.order_line.unit_price"},
		{name: "a word is not a number", value: "ITEM-A", want: "takes a number"},
		{name: "a bare boolean is not a number", value: "true", want: "takes a number"},
		{name: "leading zero is ambiguous", value: "007", want: "takes a number"},
		{name: "a fraction without digits is not a number", value: "1.", want: "takes a number"},
		{name: "a single segment is not an id", value: "price", want: "takes a number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\noperator sales.op: takes low: number\nstring\n" +
				"rule sales.item.probe:\n  sales.item.amount is sales.op low: " + tc.value + "\n"
			if tc.want == "" {
				// The value type is accepted; the assignment is then refused for
				// the separate reason that the meaning cannot carry it yet.
				_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
				if len(issues) != 1 || !strings.Contains(issues[0].Msg, "no place for an operator input value") {
					t.Fatalf("issues=%+v, want the value-transport refusal only", issues)
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
