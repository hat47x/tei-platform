package syntax_test

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Section 8.3 of the rule normal form decision fixes how a value is written for
// each declared input type. A rule that assigns an input is checked against the
// type the declaration gives THAT input, not against the one type the surface
// used to assume for every input: `low: 100` is a number, `prefix: "A-"` is a
// text, and `flag: true` is a truth.
//
// The verdicts below are read from that table, not from the conversion code. An
// attribute id is accepted for every type, because it names the item whose
// value is read and the declaration decides that value's type, not the rule.
func TestRuleAuthoringAcceptsAValueWrittenInTheDeclaredType(t *testing.T) {
	cases := []struct{ name, inputType, value string }{
		{"number takes a decimal", "number", "100"},
		{"number takes a negative fraction", "number", "-0.5"},
		{"date takes YYYY-MM-DD", "date", "2026-04-01"},
		{"text takes a quoted constant", "text", `"A-100"`},
		{"text takes an empty quoted constant", "text", `""`},
		{"truth takes true", "truth", "true"},
		{"truth takes false", "truth", "false"},
		{"any takes a decimal", "any", "100"},
		{"any takes a calendar date", "any", "2026-04-01"},
		{"any takes a quoted text", "any", `"A-100"`},
		{"any takes a truth", "any", "true"},
		{"presence takes present", "presence", "present"},
		{"presence takes absent", "presence", "absent"},
		{"number takes an attribute id", "number", "sales.order_line.unit_price"},
		{"date takes an attribute id", "date", "sales.order_line.shipped_on"},
		{"text takes an attribute id", "text", "sales.order_line.item_name"},
		{"truth takes an attribute id", "truth", "sales.order_line.paid"},
		{"any takes an attribute id", "any", "sales.order_line.unit_price"},
		{"presence takes an attribute id", "presence", "sales.order_line.discount_reason"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := syntax.RuleNamespace(mustParseRuleFile(t, ruleAssigningOneTypedInput(tc.inputType, tc.value)))
			if len(issues) != 0 {
				t.Fatalf("issues=%+v, want a %s input to take %q", issues, tc.inputType, tc.value)
			}
		})
	}
}

// The other side of the same table: a value written in another type's spelling
// is refused, and the refusal says which type the input takes and how a value
// of that type is written. A refusal that only named the type would leave the
// author guessing what to write instead.
func TestRuleAuthoringRefusesAValueNotWrittenInTheDeclaredType(t *testing.T) {
	cases := []struct{ name, inputType, value, want string }{
		{
			name: "a text without its quotes", inputType: "text", value: "A-100",
			want: `input "bound" takes a text, but "A-100" is not written as a text; ` +
				`write it between double quotes, for example "A-100", or name an attribute`,
		},
		{
			name: "a truth takes no other word", inputType: "truth", value: "yes",
			want: `input "bound" takes a truth, but "yes" is not written as a truth; ` +
				`write true or false, or name an attribute`,
		},
		{
			name: "a truth takes no digit", inputType: "truth", value: "1",
			want: `input "bound" takes a truth, but "1" is not written as a truth; ` +
				`write true or false, or name an attribute`,
		},
		{
			name: "a day the calendar does not have", inputType: "date", value: "2026-13-45",
			want: `input "bound" takes a calendar date, but "2026-13-45" is not written as a calendar date; ` +
				`write a calendar date as YYYY-MM-DD, for example 2026-04-01, or name an attribute`,
		},
		{
			name: "a quoted decimal is not a number", inputType: "number", value: `"100"`,
			want: `input "bound" takes a number, but "\"100\"" is not written as a number; ` +
				`write a decimal number, for example 100, or name an attribute`,
		},
		{
			name: "an any input takes no bare word", inputType: "any", value: "ITEM-A",
			want: `input "bound" takes a value of any type, but "ITEM-A" is not written as a value of any type; ` +
				`write a decimal number, a calendar date, a quoted text, or a truth, or name an attribute`,
		},
		{
			name: "a presence takes no other word", inputType: "presence", value: "maybe",
			want: `input "bound" takes a presence word, but "maybe" is not written as a presence word; ` +
				`write present or absent, or name an attribute`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := ruleAssigningOneTypedInput(tc.inputType, tc.value)
			_, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want the one type refusal", issues)
			}
			if issues[0].Msg != tc.want {
				t.Fatalf("message=%q, want %q", issues[0].Msg, tc.want)
			}
			// The refusal points at the bytes the author wrote for the value,
			// so the diagnostic resolves back to the source it came from.
			start := strings.Index(src, tc.value+"\n")
			if start < 0 || issues[0].Span.Start.Offset != start ||
				issues[0].Span.End.Offset != start+len(tc.value) {
				t.Fatalf("span=%+v, want value %q at byte %d", issues[0].Span, tc.value, start)
			}
		})
	}
}

// A quoted constant keeps its quotes all the way to the meaning, which is the
// spelling the evaluator reads a text with. A surface that dropped them would
// hand the evaluator a word and get an unreadable value back.
func TestRuleAuthoringCarriesAConstantAsTheAuthorWroteIt(t *testing.T) {
	for _, tc := range []struct{ inputType, value string }{
		{"text", `"A-100"`},
		{"number", "100"},
		{"date", "2026-04-01"},
		{"truth", "true"},
		{"any", `"A-100"`},
	} {
		t.Run(tc.inputType+" "+tc.value, func(t *testing.T) {
			ns, issues := syntax.RuleNamespace(mustParseRuleFile(t, ruleAssigningOneTypedInput(tc.inputType, tc.value)))
			if len(issues) != 0 {
				t.Fatalf("issues=%+v", issues)
			}
			want := []semantic.ValidationInputAssignment{{Name: "bound", Value: tc.value}}
			if !reflect.DeepEqual(ns.Validations[0].Inputs, want) {
				t.Fatalf("inputs=%+v, want %+v", ns.Validations[0].Inputs, want)
			}
			// The evaluator reads the carried text in the declared type, so
			// the value that reached the meaning is the one it can read.
			if _, err := semantic.ParseOperatorInputValue(semantic.OperatorInput(tc.inputType), tc.value); err != nil {
				t.Fatalf("the carried constant is not readable as %s: %v", tc.inputType, err)
			}
		})
	}
}

// The number rules that already worked keep working, byte for byte: the same
// two assignments convert to the same assignments, whether the bound is a
// decimal or an attribute id.
func TestRuleAuthoringKeepsTheExistingNumberRules(t *testing.T) {
	src := "namespace sales\n\noperator sales.within:\n  takes value: number\n  takes low: number\n  takes high: number\n\n" +
		"rule sales.order_line.discount_in_range:\n  sales.order_line.discount is sales.within low: 0 high: 200\n"
	ns, issues := syntax.RuleNamespace(mustParseRuleFile(t, src))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want the number rule to convert as before", issues)
	}
	if len(ns.Validations) != 1 {
		t.Fatalf("validations=%+v, want one", ns.Validations)
	}
	want := []semantic.ValidationInputAssignment{{Name: "low", Value: "0"}, {Name: "high", Value: "200"}}
	if !reflect.DeepEqual(ns.Validations[0].Inputs, want) {
		t.Fatalf("inputs=%+v, want %+v", ns.Validations[0].Inputs, want)
	}

	bound := strings.Replace(src, "low: 0 high: 200", "low: 0 high: sales.order_line.unit_price", 1)
	ns, issues = syntax.RuleNamespace(mustParseRuleFile(t, bound))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want an attribute id to bind a number input as before", issues)
	}
	want = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "sales.order_line.unit_price"},
	}
	if !reflect.DeepEqual(ns.Validations[0].Inputs, want) {
		t.Fatalf("inputs=%+v, want %+v", ns.Validations[0].Inputs, want)
	}
}

// ruleAssigningOneTypedInput writes a source whose operator declares one input
// of the given type for the rule subject and a second of the same type for the
// rule to assign. The subject supplies the first input, so an accepted
// assignment is the only thing left to report: a source that also rebound the
// subject's input would be refused for a reason that has nothing to do with
// the value's type.
func ruleAssigningOneTypedInput(inputType, value string) string {
	return "namespace sales\n\noperator sales.probe:\n  takes value: " + inputType + "\n  takes bound: " + inputType + "\n\n" +
		"rule sales.item.probe:\n  sales.item.amount is sales.probe bound: " + value + "\n"
}
