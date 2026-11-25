package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// A row-scoped operator states its meaning as one table per example, and a
// Workspace whose tables agree with the selected implementation loads and
// passes the contract check. One table whose expected answer the implementation
// does not give stops the load at that table, and no rule is evaluated against
// data (decision section 8.1).
//
// The document and the expected answers below are written by hand from the
// operator's meaning: every row's value is at most its limit.

const rowScopedCasesWorkspaceManifest = "workspace: row_cases\nformatVersion: 1\nsources:\n  - semantic\n  - applications\nlanguages:\n  - tei.authoring/v1\n" +
	"implementations:\n  - operator: sales.every_line_at_most\n    use: tei.number.every-at-most\n"

const rowScopedCasesDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.every_line_at_most:\n" +
	"  takes value: number of each row\n" +
	"  takes limit: number\n" +
	"  refutes on some rows\n" +
	"```\n" +
	"\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 5 |\n" +
	"\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 4 |\n"

func writeRowScopedCasesWorkspace(t *testing.T, document string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic", "sales"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(rowScopedCasesWorkspaceManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "every-line.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The operator the loaded Workspace declares must carry both examples, each
// with its rows and its answer.
func TestRowScopedCasesLoadAsTwoExamples(t *testing.T) {
	root := writeRowScopedCasesWorkspace(t, rowScopedCasesDocument)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var found *semantic.Operator
	for i := range loaded.World.Semantics {
		for j := range loaded.World.Semantics[i].Operators {
			if loaded.World.Semantics[i].Operators[j].ID == "sales.every_line_at_most" {
				found = &loaded.World.Semantics[i].Operators[j]
			}
		}
	}
	if found == nil {
		t.Fatal("the operator was not loaded")
	}
	if len(found.Cases) != 2 {
		t.Fatalf("cases=%d, want two examples", len(found.Cases))
	}
	for index, want := range []bool{true, false} {
		example := found.Cases[index]
		if example.RowAnswer == nil || *example.RowAnswer != want {
			t.Fatalf("case %d answer=%v, want %v", index, example.RowAnswer, want)
		}
		if len(example.Rows) != 2 {
			t.Fatalf("case %d rows=%v, want two rows", index, example.Rows)
		}
	}

	// The tables agree with the shipped implementation, so the contract check
	// accepts the selection and nothing is refused.
	if err := validationeval.CheckImplementationMatchesContract(*found, selectedEveryAtMost(t)); err != nil {
		t.Fatalf("the shipped implementation does not match its own examples: %v", err)
	}
}

func selectedEveryAtMost(t *testing.T) validationeval.OperatorImplementation {
	t.Helper()
	available := validationeval.BuiltinImplementations()
	for _, candidate := range available {
		if candidate.ID == "tei.number.every-at-most" {
			return validationeval.OperatorImplementation{
				Operator:   "sales.every_line_at_most",
				Input:      candidate.Input,
				NumberRows: candidate.NumberRows,
			}
		}
	}
	t.Fatal("the shipped tei.number.every-at-most is missing")
	return validationeval.OperatorImplementation{}
}

// One example whose answer the implementation does not give stops the load at
// that table: an implementation is not used for an operator whose stated
// meaning it contradicts.
func TestRowScopedCasesThatDisagreeWithTheImplementationStopTheLoad(t *testing.T) {
	// The rows are the same as the satisfied example, but the table states
	// not-satisfied, which every-at-most does not answer.
	document := strings.Replace(rowScopedCasesDocument,
		"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 5 | 5 |\n",
		"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 5 | 5 |\n", 1)
	root := writeRowScopedCasesWorkspace(t, document)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var found *semantic.Operator
	for i := range loaded.World.Semantics {
		for j := range loaded.World.Semantics[i].Operators {
			if loaded.World.Semantics[i].Operators[j].ID == "sales.every_line_at_most" {
				found = &loaded.World.Semantics[i].Operators[j]
			}
		}
	}
	if found == nil || len(found.Cases) == 0 {
		t.Fatal("the examples were not loaded")
	}
	// `tei validate` is the boundary that asks the selected implementation
	// about the contract's examples; Load only carries them.
	if err := validationeval.CheckImplementationMatchesContract(*found, selectedEveryAtMost(t)); err == nil {
		t.Fatal("an example the shipped implementation contradicts was accepted")
	} else if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
		t.Fatalf("the refusal does not name the mismatch: %v", err)
	}
}

// A row-scoped table for an operator that declares no input over each row
// states rows nobody reads: it is refused at the table's answer.
func TestRowScopedCasesNeedAnOperatorThatReadsEveryRow(t *testing.T) {
	document := "# x\n\n" +
		"```tei\nnamespace sales\n\noperator sales.every_line_at_most:\n  takes value: number\n```\n\n" +
		"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| value |\n| --- |\n| 2 |\n| 5 |\n"
	root := writeRowScopedCasesWorkspace(t, document)
	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("a row-scoped table for a one-record operator loaded")
	}
	if !strings.Contains(err.Error(), "reads every row") && !strings.Contains(err.Error(), "declares no input read over each row") {
		t.Fatalf("the refusal does not name the missing row-scoped input: %v", err)
	}
}
