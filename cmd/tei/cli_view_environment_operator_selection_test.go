//go:build !tei_api_only

package main

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// A rule decided by a declared operator must be answerable through the same
// `tei view -environment` entry the guide uses.
//
// Two things are checked here, both decided before PostgreSQL is opened, so
// neither needs a database: a selection this binary cannot honour is refused
// with a message that says what it could have chosen, and the selected
// implementation arrives bound to the operator the Workspace selected.
//
// The Workspace below is the reference one, copied and given an operator rule
// against an attribute it already declares. Using the real Workspace matters:
// a hand-built one would have to re-declare a Pattern and a Contract Query
// binding, and then this test would be about the fixture rather than about the
// reader.
func operatorViewWorkspace(t *testing.T, implementations string) workspace.Result {
	t.Helper()
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)

	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the reference Workspace already selects a language, so this copy would not exercise the edit")
	}
	content = append(content, []byte("languages:\n  - tei.authoring/v1\n"+implementations)...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// The rule reads an attribute the reference Workspace already declares, so
	// no semantic model has to be invented for this check.
	rule := "# 数量の規則\n\n数量は負の数にしない。0は正しい入力である。\n\n```tei\n" +
		"namespace sales\n\noperator sales.non_negative: takes number\n\n" +
		"rule sales.order_line.quantity_non_negative:\n  sales.order_line.quantity is sales.non_negative\n```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "quantity-non-negative.tei.md"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order.lines.line_number_unique\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor,
		anchor+"    uses validation sales.order_line.quantity_non_negative\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("the operator Workspace did not load: %v", err)
	}
	return loaded
}

// The reader refuses a selection it cannot honour before it opens PostgreSQL.
// A missing DSN must not mask it: if the connection were attempted first, the
// reader would report a database problem for a Workspace mistake.
func TestEnvironmentRefusesAnUnavailableOperatorImplementationBeforeConnection(t *testing.T) {
	const unavailable = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.does-not-exist\n"
	loaded := operatorViewWorkspace(t, unavailable)
	if len(loaded.OperatorImplementations) != 1 {
		t.Fatalf("test setup: the Workspace selection did not load: %+v", loaded.OperatorImplementations)
	}

	t.Setenv("TEI_VIEW_PG_DSN", "")
	_, err := readEnvironmentRows(
		loaded.World, "order_entry", "order_entry.order_lines", "development",
		map[string]string{"order_id": "1001"},
		scenarioImplementationSelections(loaded.OperatorImplementations),
	)
	if err == nil {
		t.Fatal("an unavailable implementation was accepted")
	}
	if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE") {
		t.Fatalf("the refusal did not name the unavailable implementation: %v", err)
	}
	// The names this binary does carry must appear, so the reader is told what
	// could have been selected instead of being left to guess.
	if !strings.Contains(err.Error(), "tei.number.non-negative") {
		t.Fatalf("the refusal did not list what is available: %v", err)
	}
	if strings.Contains(err.Error(), "TEI_VIEW_PG_DSN") {
		t.Fatalf("the database connection was considered before the selection was refused: %v", err)
	}
}

// The selected implementation arrives bound to the operator the Workspace
// selected. 0 is the value where "not negative" and "not zero" disagree, so
// deciding 0 proves which implementation was reached and not merely that some
// implementation was.
func TestEnvironmentResolvesTheSelectedOperatorImplementation(t *testing.T) {
	const selected = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n"
	loaded := operatorViewWorkspace(t, selected)

	available := validationeval.BuiltinImplementations()
	implementations, err := validationeval.ResolveImplementations(
		scenarioImplementationSelections(loaded.OperatorImplementations), available, nil,
	)
	if err != nil {
		t.Fatalf("a resolvable selection was refused: %v", err)
	}
	if len(implementations) != 1 {
		t.Fatalf("got %d implementations, want exactly the selected one: %+v", len(implementations), implementations)
	}
	got := implementations[0]
	if string(got.Operator) != "sales.non_negative" {
		t.Fatalf("the implementation is bound to %s, not the operator the Workspace selected", got.Operator)
	}
	if got.Number == nil || got.Numbers != nil {
		t.Fatal("the single-input implementation did not arrive as a one-number check")
	}
	holds, err := got.Number(new(big.Rat))
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatal("tei.number.non-negative rejected 0, so a different implementation was selected")
	}

	// The declaration must arrive too. Without it the evaluator reports the
	// operator as "not declared" even though the rule's Source declares it.
	declared := driver.NewResolver(loaded.World).Operators()
	if len(declared) != 1 || string(declared[0].ID) != "sales.non_negative" {
		t.Fatalf("the declared operator did not arrive: %+v", declared)
	}
}

// A Workspace that selected nothing carries the declaration and no
// implementation. That shape is what makes an operator rule undecidable rather
// than silently dropped: the reader must not answer fewer rules than the
// Application names.
func TestEnvironmentKeepsTheDeclarationWhenNothingIsSelected(t *testing.T) {
	loaded := operatorViewWorkspace(t, "")
	if len(loaded.OperatorImplementations) != 0 {
		t.Fatalf("test setup: the unselected Workspace already carries %d selections",
			len(loaded.OperatorImplementations))
	}
	implementations, err := validationeval.ResolveImplementations(nil, validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("resolving no selections failed: %v", err)
	}
	if len(implementations) != 0 {
		t.Fatalf("nothing was selected, but %d implementations arrived: %+v", len(implementations), implementations)
	}
	declared := driver.NewResolver(loaded.World).Operators()
	if len(declared) != 1 {
		t.Fatalf("the declaration was dropped along with the missing implementation: %+v", declared)
	}
}
