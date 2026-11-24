package workspace_test

import (
	"strings"
	"testing"

	"tei/internal/workspace"
)

// A case table for a multi-input operator names one column per input, and the
// signature check lives in the Workspace, not the table parser. Measured before
// this: `tei validate` accepted a table naming an undeclared input (`other`) and
// accepted the single-value form for a three-input operator, so the examples
// looked valid until a read decided the rule.
//
// These tests drive the loaded Workspace, because that is where the refusal has
// to happen.
func TestCaseColumnsAreMatchedToTheOperatorSignature(t *testing.T) {
	withinDoc := "```tei\nnamespace sales\n\noperator sales.within:\n" +
		"  takes value: number\n  takes low: number\n  takes high: number\n```\n\n"
	for _, tc := range []struct {
		name  string
		table string
		want  string
	}{
		{
			name: "one column per input is accepted",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | high | expect |\n| --- | --- | --- | --- |\n" +
				"| 0 | 0 | 100 | satisfied |\n",
		},
		{
			name: "a column naming an undeclared input is refused",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | high | other | expect |\n| --- | --- | --- | --- | --- |\n" +
				"| 0 | 0 | 100 | 5 | satisfied |\n",
			want: `declares no input named "other"`,
		},
		{
			name: "an example that leaves a declared input unstated is refused",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | expect |\n| --- | --- | --- |\n" +
				"| 0 | 0 | satisfied |\n",
			want: `declares input "high", but this example states no value for it`,
		},
		{
			name: "the single-value form for a multi-input operator is refused",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | expect |\n| --- | --- |\n| 0 | satisfied |\n",
			want: "declares 3 inputs, but this example states one value only",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			manifest := "workspace: tei_probe\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n"
			writeSource(t, root, "tei.workspace.yaml", []byte(manifest))
			writeSource(t, root, "semantic/sales/within.tei.md", []byte(withinDoc+tc.table))
			_, err := workspace.Load(root)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("a well-formed table was refused: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("the table was accepted; want a refusal naming %s", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal does not say %q: %v", tc.want, err)
			}
		})
	}
}

// A single-input operator keeps the value/expect table it always had.
func TestSingleInputCaseTableIsUnchanged(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: tei_probe\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n"
	writeSource(t, root, "tei.workspace.yaml", []byte(manifest))
	doc := "```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n```\n\n" +
		"<!-- tei: cases of sales.non_negative -->\n" +
		"| value | expect |\n| --- | --- |\n| 300 | satisfied |\n| -1 | not-satisfied |\n"
	writeSource(t, root, "semantic/sales/non-negative.tei.md", []byte(doc))
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("the single-input table was refused: %v", err)
	}
}
