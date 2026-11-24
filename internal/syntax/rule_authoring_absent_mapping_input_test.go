package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A value-absent mapping belongs to ONE input: `takes value: number absent as
// open` states how an absent `value` is read, not how an absent `low` is read.
// The check used to ask only whether the operator declared some mapping of that
// name, so a rule could read `low`'s absence as `open` while the operator had
// only spoken about `value` (#4359, decision section 14.2 B-10's neighbours).
func TestAbsentMappingBelongsToItsInput(t *testing.T) {
	const body = "namespace sales\n\noperator sales.op:\n" +
		"  takes value: number absent as open\n" +
		"  takes low: number\n" +
		"  takes high: number\n\n" +
		"rule sales.order_line.r:\n" +
		"  sales.order_line.quantity is sales.op low: 0 or open high: 100\n"
	value, issues := syntax.RuleNamespace(parseAbsentSource(t, body))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the undeclared-for-this-input one", issues)
	}
	if !strings.Contains(issues[0].Msg, `"open"`) || !strings.Contains(issues[0].Msg, `input "low"`) {
		t.Fatalf("the refusal does not name the mapping and the input it was used on: %q", issues[0].Msg)
	}
	if issues[0].Span == (syntax.Anchor{}) {
		t.Fatal("the refusal carries no source position")
	}
	if len(value.Validations) != 0 {
		t.Fatalf("the rule survived: %+v", value.Validations)
	}
}

// The same mapping declared on the input the rule names is accepted: the check
// is about which input the mapping belongs to, not about refusing mappings.
func TestAbsentMappingOnTheNamedInputIsAccepted(t *testing.T) {
	const body = "namespace sales\n\noperator sales.op:\n" +
		"  takes value: number\n" +
		"  takes low: number absent as open\n" +
		"  takes high: number\n\n" +
		"rule sales.order_line.r:\n" +
		"  sales.order_line.quantity is sales.op low: 0 or open high: 100\n"
	value, issues := syntax.RuleNamespace(parseAbsentSource(t, body))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
	if len(value.Validations) != 1 {
		t.Fatalf("validations=%+v, want the rule", value.Validations)
	}
}

// A mapping the operator declares nowhere is still refused, with the same
// wording as before plus the input.
func TestAbsentMappingDeclaredNowhereIsRefused(t *testing.T) {
	const body = "namespace sales\n\noperator sales.op:\n" +
		"  takes value: number\n" +
		"  takes low: number\n" +
		"  takes high: number\n\n" +
		"rule sales.order_line.r:\n" +
		"  sales.order_line.quantity is sales.op low: 0 or closed high: 100\n"
	_, issues := syntax.RuleNamespace(parseAbsentSource(t, body))
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, `no value-absent mapping named "closed" for input "low"`) {
		t.Fatalf("issues=%+v", issues)
	}
}

func parseAbsentSource(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("cannot parse the source: %v", err)
	}
	return file
}
