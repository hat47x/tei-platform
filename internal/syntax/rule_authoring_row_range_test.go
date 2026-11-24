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

// The range and the refutation statement are carried into the meaning now, so a
// declaration that states them is converted. What is still refused is a
// combination the product path cannot evaluate, and each is refused by its own
// name, at its own span: one problem must not be reported as another.
func TestRuleAuthoringRefusesWhatCannotBeEvaluatedByTheirOwnNames(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{
			// The row reader reads numbers. An input of another type would be
			// carried as a range nothing can read.
			name: "range on an input with no row reader",
			body: "operator sales.op:\n  takes start: presence of each row\n",
			want: `a row reader reads numbers over all rows`,
		},
		{
			// An operator that reads every row of a relation is asked once for
			// the whole relation. An input read once has no row to be read
			// from, and reading it from the first row would answer the whole
			// relation from one record.
			name: "range on one input only",
			body: "operator sales.op:\n  takes start: number\n  takes end: number of each row\n",
			want: `is not read over all rows, but another input is`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\n" + tc.body
			_, issues := syntax.RuleNamespace(mustParseRangeFile(t, src))
			if len(issues) != 1 {
				t.Fatalf("issues=%+v, want the un-evaluable one", issues)
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

// The range and the refutation statement reach the meaning. `of each row`
// lands on the input declaration it was written on, and `refutes on some rows`
// on the operator, because that is what each of them states about.
func TestRuleAuthoringCarriesTheRangeAndTheRefutationIntoTheMeaning(t *testing.T) {
	src := `namespace sales

operator sales.below_limit:
  takes value: number of each row
  refutes on some rows
`
	value, issues := syntax.RuleNamespace(mustParseRangeFile(t, src))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
	if len(value.Operators) != 1 {
		t.Fatalf("operators=%+v", value.Operators)
	}
	operator := value.Operators[0]
	if len(operator.Inputs) != 1 || !operator.Inputs[0].OfEachRow {
		t.Fatalf("inputs=%+v, want the one input read over all rows", operator.Inputs)
	}
	if !operator.RefutesOnSomeRows {
		t.Fatal("the refutation statement was not carried")
	}
	if !operator.ReadsEveryRow() {
		t.Fatal("the contract does not report that it reads every row")
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
