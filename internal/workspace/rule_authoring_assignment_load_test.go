//go:build !tei_repair_only

package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// An operator input assignment cannot be carried into the meaning yet, so a
// Markdown source that writes one is refused at Load. The loader is
// all-or-nothing: the whole source stops rather than contributing the other
// rules it holds, so the failure has to name the assignment.
func TestRuleAuthoringSourceWithOperatorInputAssignmentIsRefusedAtLoad(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml",
		"workspace: probe\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
	writeWorkspaceFile(t, root, "semantic/probe.tei.md", `# probe

`+"```tei\nnamespace sales\n\noperator sales.within: takes low: number\n\nrule sales.item.probe:\n  sales.item.amount is sales.within low: 100\n```\n")

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("a source whose rule assigns an operator input loaded as meaning")
	}
	if !strings.Contains(err.Error(), "no place for an operator input value") {
		t.Fatalf("the refusal does not name the assignment: %v", err)
	}
	if !strings.Contains(err.Error(), "sales.item.probe") {
		t.Fatalf("the refusal does not name the rule: %v", err)
	}
}

// The actual tei.authoring/v1 Workspace reader must reject a repeated input
// before any future semantic normalization could select one of its values.
func TestRuleAuthoringSourceWithRepeatedInputIsRejectedAtLoad(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml",
		"workspace: probe\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
	writeWorkspaceFile(t, root, "semantic/probe.tei.md", `# probe

`+"```tei\nnamespace sales\n\noperator sales.within: takes low: number\n\nrule sales.item.probe:\n  sales.item.amount is sales.within low: 0 low: 100\n```\n")

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("a source whose rule duplicates an operator input loaded as meaning")
	}
	if !strings.Contains(err.Error(), "assigned more than once") {
		t.Fatalf("the refusal does not identify the duplicate input: %v", err)
	}
}

func writeWorkspaceFile(t *testing.T, root, slashPath, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(slashPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
