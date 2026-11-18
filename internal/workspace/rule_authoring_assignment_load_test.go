//go:build !tei_repair_only

package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// A rule subject already supplies its Operator's first input. A Markdown
// source must reject a named assignment that supplies it again at Load.
// The loader is all-or-nothing and must identify that second binding.
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
	if !strings.Contains(err.Error(), "already supplied by the rule subject") {
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
