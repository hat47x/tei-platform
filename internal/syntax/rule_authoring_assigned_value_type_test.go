package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A rule's assigned value is checked against the type the declaration states,
// not against the type this surface happened to assume. A text input takes a
// quoted text or an attribute id, a calendar-day input takes YYYY-MM-DD or an
// attribute id, and a truth input takes true or false or an attribute id
// (section 3, 8.3).
//
// Measured before this: a rule assigning `deadline: 2026-04-01` to a date input
// was refused with "takes a number", which named a type the declaration never
// stated.
func TestAnAssignedValueIsCheckedAgainstTheDeclaredType(t *testing.T) {
	for _, tc := range []struct {
		name      string
		declares  string
		input     string
		value     string
		wantOK    bool
		wantInMsg string
		spelling  string
	}{
		{
			name: "a text input refuses a decimal", declares: "takes value: text\n  takes other: text",
			input: "other", value: "100", wantInMsg: "is read as text", spelling: `write a text in double quotes such as "A-100"`,
		},
		{
			name: "a text input refuses a bare word", declares: "takes value: text\n  takes other: text",
			input: "other", value: "A-100", wantInMsg: "is read as text", spelling: `write a text in double quotes such as "A-100"`,
		},
		{
			name: "a calendar-day input takes YYYY-MM-DD", declares: "takes value: date\n  takes deadline: date",
			input: "deadline", value: "2026-04-01", wantOK: true,
		},
		{
			name: "a calendar-day input refuses a decimal", declares: "takes value: date\n  takes deadline: date",
			input: "deadline", value: "1", wantInMsg: "is read as date", spelling: "write a calendar day as YYYY-MM-DD",
		},
		{
			name: "a truth input takes true", declares: "takes value: truth\n  takes other: truth",
			input: "other", value: "true", wantOK: true,
		},
		{
			name: "a truth input refuses a decimal", declares: "takes value: truth\n  takes other: truth",
			input: "other", value: "1", wantInMsg: "is read as truth", spelling: "write true or false",
		},
		{
			name: "an any input takes a decimal", declares: "takes value: any\n  takes other: any",
			input: "other", value: "7", wantOK: true,
		},
		{
			name: "an any input refuses a bare word", declares: "takes value: any\n  takes other: any",
			input: "other", value: "A-100", wantInMsg: "is read as any", spelling: "write a decimal number, a calendar day as YYYY-MM-DD, a text in double quotes, or true or false",
		},
		{
			name: "a number input still takes a decimal", declares: "takes value: number\n  takes limit: number",
			input: "limit", value: "200", wantOK: true,
		},
		{
			name: "a number input still refuses a truth", declares: "takes value: number\n  takes limit: number",
			input: "limit", value: "true", wantInMsg: "is read as number", spelling: "write a decimal number such as 100 or -0.01",
		},
		// An attribute id names the item whose value is read, and the reader
		// reads that item as the declared type, so it is accepted for every one.
		{
			name: "a text input takes an attribute id", declares: "takes value: text\n  takes other: text",
			input: "other", value: "sales.order_line.item_code", wantOK: true,
		},
		{
			name: "a calendar-day input takes an attribute id", declares: "takes value: date\n  takes deadline: date",
			input: "deadline", value: "sales.order_line.ordered_at", wantOK: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\noperator sales.op:\n  " + tc.declares + "\n" +
				"rule sales.item_probe:\n  sales.item.amount is sales.op " + tc.input + ": " + tc.value + "\n"
			_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
			if tc.wantOK {
				// The value is accepted and the rule assigns an input the
				// subject does not supply, so nothing is refused.
				if len(issues) != 0 {
					t.Fatalf("issues=%+v, want none", issues)
				}
				return
			}
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want the type mismatch one", issues)
			}
			if !strings.Contains(issues[0].Msg, tc.wantInMsg) {
				t.Fatalf("message=%q, want it to mention %q", issues[0].Msg, tc.wantInMsg)
			}
			// The refusal names how a value of that type is written, so the
			// author is told what to write instead of only what was wrong. The
			// wording is the type's own, so it is checked for the type's
			// spelling rather than for a phrase every type would share.
			if !strings.Contains(issues[0].Msg, tc.spelling) {
				t.Fatalf("message=%q, want it to say how to write a value (%q)", issues[0].Msg, tc.spelling)
			}
		})
	}
}
