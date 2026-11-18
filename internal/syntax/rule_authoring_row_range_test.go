package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// §12 of the rule normal form decision writes an input that reads every row as
// `takes start: date of each row`, and a declaration whose witnessed
// counterexample is decisive as `refutes on some rows`. Both are read here.
func TestRuleAuthoringReadsTheRowRangeAndTheRefutationStatement(t *testing.T) {
	src := `namespace sales

operator sales.op:
  takes start: number of each row
  refutes on some rows
`
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("the range and refutation did not parse: %v", err)
	}
	if len(file.Operators) != 1 {
		t.Fatalf("operators=%+v", file.Operators)
	}
	operator := file.Operators[0]
	if !operator.OfEachRow {
		t.Fatal("of each row was not recorded")
	}
	if !operator.RefutesOnSomeRows {
		t.Fatal("refutes on some rows was not recorded")
	}
}

// Neither statement has a place in the meaning yet, so a declaration that states
// one is refused by name rather than converted with the statement dropped. One
// problem must not be reported as the other.
func TestRuleAuthoringRefusesUnsupportedRangeAndRefutationByTheirOwnNames(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{
			name: "range",
			body: "operator sales.op:\n  takes start: number of each row\n",
			want: `no place for an input read over all rows`,
		},
		{
			name: "refutation",
			body: "operator sales.op:\n  takes start: number\n  refutes on some rows\n",
			want: "no place for a refutation statement",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\n" + tc.body
			_, issues := syntax.RuleNamespace(mustParseRangeFile(t, src))
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want the unsupported-statement one", issues)
			}
			if !strings.Contains(issues[0].Msg, tc.want) {
				t.Fatalf("message=%q, want it to mention %q", issues[0].Msg, tc.want)
			}
			if issues[0].Span == (syntax.Anchor{}) {
				t.Fatal("the issue carries no span")
			}
		})
	}
}

// The clauses are spelled exactly. A misspelling is refused at the word that is
// wrong, not accepted as a declaration with no range.
func TestRuleAuthoringRefusesMisspelledRangeAndRefutation(t *testing.T) {
	for _, tc := range []struct {
		name, tail, want string
	}{
		{"of without each", "takes start: number of row\n", `expected "each" after "of"`},
		{"of each without row", "takes start: number of each\n", `expected "row" after "of each"`},
		{"refutes without on", "takes start: number\n refutes rows\n", `expected "on" after "refutes"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile("namespace sales\noperator sales.op:\n " + tc.tail)
			if err == nil {
				t.Fatalf("%s was accepted", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("message=%q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func mustParseRangeFile(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
