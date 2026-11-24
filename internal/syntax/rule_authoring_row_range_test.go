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

// Both statements are carried into the meaning (decision section 15.5). A
// refutation statement is carried only when some input reads every row: without
// one there are no rows to witness, and the statement would claim something the
// rule cannot support.
func TestRowRangeAndRefutationAreCarriedIntoTheMeaning(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{
			name: "a row-range input converts",
			body: "operator sales.op:\n  takes start: number of each row\n",
		},
		{
			name: "a row-range input with a refutation converts",
			body: "operator sales.op:\n  takes start: number of each row\n  refutes on some rows\n",
		},
		{
			name: "a refutation without a row-range input is refused",
			body: "operator sales.op:\n  takes start: number\n  refutes on some rows\n",
			want: "declares no input read over all rows",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\n" + tc.body
			namespace, issues := syntax.RuleNamespace(mustParseRangeFile(t, src))
			if tc.want == "" {
				if len(issues) != 0 {
					t.Fatalf("a valid declaration was refused: %+v", issues)
				}
				if len(namespace.Operators) != 1 {
					t.Fatalf("operators=%+v, want the converted one", namespace.Operators)
				}
				return
			}
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want one for the unsupported statement", issues)
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
