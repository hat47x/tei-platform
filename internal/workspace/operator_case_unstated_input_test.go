package workspace_test

import (
	"strings"
	"testing"

	"tei/internal/workspace"
)

// #5896 refuses a case column that names no declared input. It does not refuse
// the other half: an example that leaves a declared input unstated. Such an
// example still passes `tei validate` and is refused only when a read decides
// the rule, so a Workspace nobody reads looks valid until someone queries it.
//
// These cases pin both halves at the table, and pin the per-input table that
// must keep loading.
func TestCaseExamplesStateEveryDeclaredInput(t *testing.T) {
	withinDoc := "```tei\nnamespace sales\n\noperator sales.within:\n" +
		"  takes value: number\n  takes low: number\n  takes high: number\n```\n\n"
	for _, tc := range []struct {
		name  string
		table string
		want  string
	}{
		{
			name: "every declared input is stated",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | high | expect |\n| --- | --- | --- | --- |\n" +
				"| 0 | 0 | 100 | satisfied |\n| 101 | 0 | 100 | not-satisfied |\n",
		},
		{
			name: "an example that leaves high unstated is refused",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | expect |\n| --- | --- | --- |\n" +
				"| 0 | 0 | satisfied |\n",
			want: `declares input "high", but this example states no value for it`,
		},
		{
			name: "a column naming an undeclared input is still refused",
			table: "<!-- tei: cases of sales.within -->\n" +
				"| value | low | high | other | expect |\n| --- | --- | --- | --- | --- |\n" +
				"| 0 | 0 | 100 | 5 | satisfied |\n",
			want: `declares no input named "other"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			manifest := "workspace: tei_probe\nformatVersion: 1\nsources:\n  - semantic\n" +
				"languages:\n  - tei.authoring/v1\n"
			writeSource(t, root, "tei.workspace.yaml", []byte(manifest))
			writeSource(t, root, "semantic/sales/within.tei.md", []byte(withinDoc+tc.table))
			_, err := workspace.Load(root)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("a complete example table was refused: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("the table was accepted; want a refusal naming %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal does not say %q: %v", tc.want, err)
			}
		})
	}
}

// A single-input operator's value/expect table states its one input in the
// value cell, so it must keep loading.
func TestSingleInputCaseTableStillLoads(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: tei_probe\nformatVersion: 1\nsources:\n  - semantic\n" +
		"languages:\n  - tei.authoring/v1\n"
	writeSource(t, root, "tei.workspace.yaml", []byte(manifest))
	doc := "```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n```\n\n" +
		"<!-- tei: cases of sales.non_negative -->\n" +
		"| value | expect |\n| --- | --- |\n| 300 | satisfied |\n| -1 | not-satisfied |\n"
	writeSource(t, root, "semantic/sales/non-negative.tei.md", []byte(doc))
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("the single-input table was refused: %v", err)
	}
}
