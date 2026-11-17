package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// §3 of the rule normal form decision: a value-absent reading is declared per
// input, and a rule may name only a mapping the operator declared. The
// expected verdicts below are written from that statement.
func TestRuleAuthoringReadsTheDeclaredAbsentMapping(t *testing.T) {
	src := `namespace sales

operator sales.op: takes end: number absent as open

rule sales.item.probe:
  sales.item.amount is sales.op end: 1 or open
`
	parsed, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("the declared mapping and its use did not parse: %v", err)
	}
	if len(parsed.Value.Operators) != 1 || parsed.Value.Operators[0].AbsentAs != "open" {
		t.Fatalf("operators=%+v, want the declared mapping", parsed.Value.Operators)
	}
	// The mapping is declared, so the rule is refused only for the separate
	// reason that the value has nowhere to go yet.
	_, issues := syntax.RuleNamespace(parsed.File)
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "no place for an operator input value") {
		t.Fatalf("issues=%+v, want the value-transport refusal only", issues)
	}
}

// A rule that names a mapping the operator never declared states a reading the
// operator did not fix, so it is refused and the diagnostic points at the name.
func TestRuleAuthoringRefusesAnUndeclaredAbsentMapping(t *testing.T) {
	src := `namespace sales

operator sales.op: takes end: number

rule sales.item.probe:
  sales.item.amount is sales.op end: 1 or open
`
	_, issues := syntax.RuleNamespace(mustParseAbsentFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the undeclared-mapping one", issues)
	}
	if !strings.Contains(issues[0].Msg, `no value-absent mapping named "open"`) {
		t.Fatalf("message=%q", issues[0].Msg)
	}
	if issues[0].Span == (syntax.Anchor{}) {
		t.Fatal("the issue carries no span")
	}
}

// `absent as` without a name, or without `as`, is refused rather than read as a
// declaration with no mapping.
func TestRuleAuthoringRefusesIncompleteAbsentMappingDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name, tail, want string
	}{
		{"missing as", "takes end: number absent open\n", `expected "as" after "absent"`},
		{"missing name", "takes end: number absent as\n", "expected the name the absent value is read as"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op: " + tc.tail)
			if err == nil {
				t.Fatalf("%s was accepted", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("message=%q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func mustParseAbsentFile(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
