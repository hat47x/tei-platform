package main

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/workspace"
)

// A rule decided by a declared operator must be answerable through the same
// `tei view -environment` entry the guide uses, and only through it.
//
// Before this, readEnvironmentRows passed an empty validationeval.Operators.
// Every operator rule was then refused as "not declared", so a Workspace that
// declared an operator and selected an implementation still could not be read.
// The two tests below pin both halves: a selection that resolves answers the
// rule, and a Workspace that selected nothing is refused rather than answered
// with fewer rules than the Application names.
//
// No database is needed. Both cases stop before a connection: the first asks
// only for the operator set, and the second fails while the rules are being
// decided.
func operatorWorkspaceForView(t *testing.T, implementations string) workspace.Result {
	t.Helper()
	root := t.TempDir()
	ruleDocument := "# 数量の規則\n\n数量は負の数にしない。0は正しい入力である。\n\n```tei\n" +
		"namespace sales\n\noperator sales.non_negative: takes number\n\n" +
		"rule sales.order_line.quantity_non_negative:\n  sales.order_line.quantity is sales.non_negative\n```\n"
	files := map[string]string{
		"tei.workspace.yaml": "workspace: view_operator\nformatVersion: 1\nsources:\n" +
			"  - semantic\n  - applications\nlanguages:\n  - tei.authoring/v1\n" + implementations,
		"semantic/sales/quantity-rules.tei.md": ruleDocument,
		"applications/check/application.tei": "application check {\n    uses semantic sales\n" +
			"    uses validation sales.order_line.quantity_non_negative\n}\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("the operator Workspace did not load: %v", err)
	}
	return loaded
}

// The selected implementation reaches the evaluator, so the operator rule has
// both a declaration and an implementation and can be decided.
func TestDestinationOperatorsCarryTheSelectedImplementation(t *testing.T) {
	const selected = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n"
	loaded := operatorWorkspaceForView(t, selected)

	operators, err := destinationOperators(driver.NewResolver(loaded.World), loaded.OperatorImplementations)
	if err != nil {
		t.Fatalf("a resolvable selection was refused: %v", err)
	}
	if len(operators.Declared) != 1 || string(operators.Declared[0].ID) != "sales.non_negative" {
		t.Fatalf("the declared operator did not arrive: %+v", operators.Declared)
	}
	if len(operators.Implementations) != 1 {
		t.Fatalf("got %d implementations, want exactly the selected one: %+v",
			len(operators.Implementations), operators.Implementations)
	}
	got := operators.Implementations[0]
	if string(got.Operator) != "sales.non_negative" {
		t.Fatalf("the implementation is bound to %s, not the operator the Workspace selected", got.Operator)
	}
	if got.Number == nil || got.Numbers != nil {
		t.Fatal("the single-input implementation did not arrive as a one-number check")
	}
	// 0 is the value where "not negative" and "not zero" disagree, so the
	// decision below proves which implementation was actually reached.
	holds, err := got.Number(new(big.Rat))
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatal("tei.number.non-negative rejected 0, so a different implementation was selected")
	}
}

// A Workspace that selected nothing still carries the declaration, and
// carries no implementation. That shape is what makes the evaluation refuse
// the rule: it is "declared but no implementation was selected", not "the
// operator does not exist". The refusal itself happens where the rule is
// decided, which needs a Detail; the shape is what this test can pin without
// one, and the CLI run in the PR body shows the refusal it produces.
//
// Answering nothing instead of refusing would be worse: the rule would vanish
// from the read while the Application still names it.
func TestDestinationOperatorsCarryTheDeclarationWithoutAnImplementation(t *testing.T) {
	loaded := operatorWorkspaceForView(t, "")

	operators, err := destinationOperators(driver.NewResolver(loaded.World), loaded.OperatorImplementations)
	if err != nil {
		t.Fatalf("resolving no selections failed: %v", err)
	}
	if len(loaded.OperatorImplementations) != 0 {
		t.Fatalf("test setup: the unselected Workspace already carries %d selections",
			len(loaded.OperatorImplementations))
	}
	if len(operators.Declared) != 1 || string(operators.Declared[0].ID) != "sales.non_negative" {
		t.Fatalf("the declaration was dropped along with the missing implementation: %+v", operators.Declared)
	}
	if len(operators.Implementations) != 0 {
		t.Fatalf("nothing was selected, but %d implementations arrived: %+v",
			len(operators.Implementations), operators.Implementations)
	}
}

// An implementation this tei does not carry must be an error, not silently
// dropped. Dropping it would look identical to a Workspace that selected
// nothing, and the reader could not tell the two apart.
func TestDestinationOperatorsRefuseAnImplementationThisTeiDoesNotCarry(t *testing.T) {
	// A name no builtin carries. It is written here, not read from the
	// implementation table, so this stays a negative case even if the table
	// grows.
	const unknown = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.does-not-exist\n"
	loaded := operatorWorkspaceForView(t, unknown)

	_, err := destinationOperators(driver.NewResolver(loaded.World), loaded.OperatorImplementations)
	if err == nil {
		t.Fatal("an unknown implementation name resolved without an error")
	}
	if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE") {
		t.Fatalf("the refusal did not name the unavailable implementation: %v", err)
	}
	if !strings.Contains(err.Error(), "tei.number.does-not-exist") {
		t.Fatalf("the refusal did not name the implementation that was selected: %v", err)
	}
}
