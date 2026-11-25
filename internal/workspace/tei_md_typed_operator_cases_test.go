package workspace

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// A cell of an example table states a value of the type the operator's
// signature gives its input (section 8.3). The signature is the only place that
// type comes from, so the check is made where the declaration is in hand: while
// the Workspace is loaded, before an implementation is selected and before any
// real row is read.
//
// The table reader knows the column names but not their types, because the
// operator may be declared in a document read later. It reads the notation; the
// type check belongs to the Load.

const teiMDTypedCases = "semantic/sales/a-typed-cases.tei.md"

// teiMDTypedOperatorWorkspace declares one operator of each value type and no
// rule that uses it: the shape of the table is the only thing that can be
// wrong, so a finding cannot be about a used rule.
func teiMDTypedOperatorWorkspace(t *testing.T, declarations string) string {
	t.Helper()
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	writeTEIMDTestFile(t, root, "semantic/sales/zz-operators.tei.md",
		"# 型別の演算子\n\n```tei\nnamespace sales\n\n"+declarations+"```\n")
	return root
}

// A cell that states a value of the input's own type is attached to the
// operator, and the four type words are read at authoring time.
func TestTEIMDReadsExamplesOfTheFourValueTypes(t *testing.T) {
	root := teiMDTypedOperatorWorkspace(t,
		"operator sales.due: takes value: date\n"+
			"operator sales.code: takes value: text\n"+
			"operator sales.approved: takes value: truth\n"+
			"operator sales.anything: takes value: any\n")
	writeTEIMDTestFile(t, root, teiMDTypedCases,
		"<!-- tei: cases of sales.due -->\n| value | expect |\n| --- | --- |\n| 2026-04-01 | satisfied |\n\n"+
			"<!-- tei: cases of sales.code -->\n| value | expect |\n| --- | --- |\n| \"A-100\" | satisfied |\n\n"+
			"<!-- tei: cases of sales.approved -->\n| value | expect |\n| --- | --- |\n| true | satisfied |\n\n"+
			"<!-- tei: cases of sales.anything -->\n| value | expect |\n| --- | --- |\n| 100 | satisfied |\n| \"100\" | not-satisfied |\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("examples of the four value types were refused: %v", err)
	}
	want := map[string][]string{
		"sales.due":      {"2026-04-01"},
		"sales.code":     {`"A-100"`},
		"sales.approved": {"true"},
		"sales.anything": {"100", `"100"`},
	}
	for _, operator := range operatorsOf(loaded) {
		cells, expected := want[string(operator.ID)]
		if !expected {
			continue
		}
		if len(operator.Cases) != len(cells) {
			t.Fatalf("operator %s carries %d examples, want %d", operator.ID, len(operator.Cases), len(cells))
		}
		for i, cell := range cells {
			if operator.Cases[i].Value != cell {
				t.Fatalf("operator %s example %d is %q, want %q", operator.ID, i, operator.Cases[i].Value, cell)
			}
		}
		delete(want, string(operator.ID))
	}
	if len(want) != 0 {
		t.Fatalf("these operators were not loaded: %v", want)
	}
}

// A cell written for another type is refused at its own cell, and the refusal
// names the input and its type.
func TestTEIMDRefusesAnExampleCellOfAnotherType(t *testing.T) {
	for _, tc := range []struct {
		name, declaration, table, cell, message string
	}{
		{
			"a word for a date", "operator sales.due: takes value: date",
			"<!-- tei: cases of sales.due -->\n| value | expect |\n| --- | --- |\n| abc | satisfied |\n",
			"abc", "not a decimal number",
		},
		{
			"a word for a boolean", "operator sales.approved: takes value: truth",
			"<!-- tei: cases of sales.approved -->\n| value | expect |\n| --- | --- |\n| yes | satisfied |\n",
			"yes", "not a decimal number",
		},
		{
			"a number for a text", "operator sales.code: takes value: text",
			"<!-- tei: cases of sales.code -->\n| value | expect |\n| --- | --- |\n| 100 | satisfied |\n",
			"100", "between double quotes",
		},
		{
			"an unquoted text for a text", "operator sales.code: takes value: text",
			"<!-- tei: cases of sales.code -->\n| value | expect |\n| --- | --- |\n| A-100 | satisfied |\n",
			"A-100", "between double quotes",
		},
		{
			"a date for a number", "operator sales.amount: takes value: number",
			"<!-- tei: cases of sales.amount -->\n| value | expect |\n| --- | --- |\n| 2026-04-01 | satisfied |\n",
			"2026-04-01", "decimal number",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := teiMDTypedOperatorWorkspace(t, tc.declaration+"\n")
			writeTEIMDTestFile(t, root, teiMDTypedCases, tc.table)
			_, err := Load(root)
			if err == nil {
				t.Fatal("an example cell of another type was attached to the operator")
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("the refusal does not say %q: %v", tc.message, err)
			}
			// The refusal points at the cell, not at the declaration or the
			// mark, so the author is sent to what to change.
			document := readTEIMDTestFile(t, root, teiMDTypedCases)
			if bytes.Count(document, []byte(tc.cell)) != 1 {
				t.Fatalf("%q must occur once in the table", tc.cell)
			}
			var sourceError *SourceError
			if !errors.As(err, &sourceError) {
				t.Fatalf("want a located error, got %v", err)
			}
			if want := bytes.Index(document, []byte(tc.cell)); sourceError.Range.Start.Offset != want {
				t.Fatalf("error offset=%d, want %d (the cell)", sourceError.Range.Start.Offset, want)
			}
		})
	}
}

// Where the cell states a value of another type, the refusal names the
// operator, the input and the type, so the author is not sent to guess which
// declaration is unreadable.
func TestTEIMDNamesTheInputAndTypeOfAMisreadCell(t *testing.T) {
	root := teiMDTypedOperatorWorkspace(t, "operator sales.code: takes value: text\n")
	writeTEIMDTestFile(t, root, teiMDTypedCases,
		"<!-- tei: cases of sales.code -->\n| value | expect |\n| --- | --- |\n| 100 | satisfied |\n")
	_, err := Load(root)
	if err == nil {
		t.Fatal("a number was attached as the text of a text input")
	}
	for _, want := range []string{"sales.code", `input "value"`, "text", "between double quotes"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not say %q: %v", want, err)
		}
	}
}

// A multi-input example states one cell per input, and each is checked against
// its own input's type.
func TestTEIMDChecksEachColumnOfAMultiInputExample(t *testing.T) {
	root := teiMDTypedOperatorWorkspace(t,
		"operator sales.window: takes value: any\n  takes low: number\n  takes high: number\n")
	writeTEIMDTestFile(t, root, teiMDTypedCases,
		"<!-- tei: cases of sales.window -->\n| value | low | high | expect |\n| --- | --- | --- | --- |\n"+
			"| 3 | 0 | 5 | satisfied |\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("a mixed-type example was refused: %v", err)
	}
	for _, operator := range operatorsOf(loaded) {
		if operator.ID != "sales.window" || len(operator.Cases) != 1 {
			continue
		}
		if operator.Cases[0].Inputs["low"] != "0" || operator.Cases[0].Inputs["high"] != "5" {
			t.Fatalf("inputs=%v, want low=0 high=5", operator.Cases[0].Inputs)
		}
		return
	}
	t.Fatal("the mixed-type operator is missing from the loaded World")
}

// The same check is made for the table of rows a row-scoped operator states.
func TestTEIMDChecksTheCellsOfARowExample(t *testing.T) {
	root := teiMDTypedOperatorWorkspace(t,
		"operator sales.named_lines: takes value: text of each row\n  takes limit: number of each row\n")
	writeTEIMDTestFile(t, root, teiMDTypedCases,
		"<!-- tei: cases of sales.named_lines expecting satisfied -->\n| value | limit |\n| --- | --- |\n"+
			"| \"A-100\" | 10 |\n| \"B-200\" | 5 |\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("a row example of the declared types was refused: %v", err)
	}

	root = teiMDTypedOperatorWorkspace(t,
		"operator sales.named_lines: takes value: text of each row\n  takes limit: number of each row\n")
	writeTEIMDTestFile(t, root, teiMDTypedCases,
		"<!-- tei: cases of sales.named_lines expecting satisfied -->\n| value | limit |\n| --- | --- |\n"+
			"| A-100 | 10 |\n")
	_, err := Load(root)
	if err == nil {
		t.Fatal("a row example cell of another type was attached to the operator")
	}
	if !strings.Contains(err.Error(), "between double quotes") {
		t.Fatalf("the refusal does not say what the input takes: %v", err)
	}
}

// asSourceError reports whether err is a located source error.
func asSourceError(err error, target **SourceError) bool {
	if located, ok := err.(*SourceError); ok {
		*target = located
		return true
	}
	return false
}
