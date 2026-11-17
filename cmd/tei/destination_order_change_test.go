package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The destination of plan/NOW.md, steps 3 and 4: a newcomer adds one rule
// with an example next to it to a copy of the reference workspace, and
// `tei validate` accepts the change. An example the rule answers differently
// is a diagnostic. The inputs are in testdata/destination-order-change; the
// expectations there are written by hand from the meaning of the rule.
//
// Steps 1, 2 and 5 (reading the lines from a real database and answering the
// rules per line) have no entry yet. When they do, this test compares the
// answers with expected-before.json and expected-after.json.

func copyTreeForDestination(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", from, err)
	}
}

func applyDestinationEdit(t *testing.T, root, ruleDocument string) {
	t.Helper()
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the reference workspace already selects a language; edit/README.md step 1 no longer describes the change")
	}
	content = append(content, []byte("languages:\n  - tei.authoring/v1\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "unit-price.tei.md"), []byte(ruleDocument), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor, anchor+"    uses validation sales.order_line.unit_price_positive\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDestinationOrderChangeRuleEditIsAcceptedAndItsExampleIsChecked(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	document, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unchanged copy validates", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, reference, root)
		got := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		if got.code != 0 {
			t.Fatalf("code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
		}
	})

	t.Run("the edit is accepted", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, reference, root)
		applyDestinationEdit(t, root, string(document))
		got := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		if got.code != 0 {
			t.Fatalf("code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
		}
		if strings.Contains(got.stdout+got.stderr, "unit_price_positive") {
			t.Fatalf("the accepted edit is mentioned in a diagnostic: %s%s", got.stdout, got.stderr)
		}
	})

	t.Run("an example the rule answers differently is a diagnostic", func(t *testing.T) {
		const right = "| 0 | not-satisfied |"
		if strings.Count(string(document), right) != 1 {
			t.Fatalf("the document does not hold %q exactly once", right)
		}
		wrong := strings.Replace(string(document), right, "| 0 | satisfied |", 1)
		root := t.TempDir()
		copyTreeForDestination(t, reference, root)
		applyDestinationEdit(t, root, wrong)
		got := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		output := got.stdout + got.stderr
		if got.code == 0 || !strings.Contains(output, "RULE_EXAMPLE_MISMATCH") || !strings.Contains(output, "sales.order_line.unit_price_positive") {
			t.Fatalf("code=%d output=%s", got.code, output)
		}
	})

	t.Run("a rule the application does not name stays unselected", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, reference, root)
		applyDestinationEdit(t, root, string(document))
		application := filepath.Join(root, "applications", "order-entry", "application.tei")
		content, err := os.ReadFile(application)
		if err != nil {
			t.Fatal(err)
		}
		edited := strings.Replace(string(content), "    uses validation sales.order_line.unit_price_positive\n", "    uses validation sales.order_line.unit_price_missing\n", 1)
		if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		got := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		if got.code == 0 {
			t.Fatalf("an application that names an undeclared rule validated: %s%s", got.stdout, got.stderr)
		}
	})
}
