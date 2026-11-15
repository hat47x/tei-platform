package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// The rule surface is not wired into workspace.Load. A Workspace that puts a
// rule declaration into a semantic Source must therefore stop with a
// source-located diagnostic, not load a half-understood model and not silently
// ignore the file (#4359 S6 compatibility, #4367).
func TestWorkspaceNamesARuleDeclarationAsPlacement(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: rule_placement\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "semantic", "sales", "validation.tei")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("namespace sales\n\ndata Order {\n    attribute order_id type std.identifier\n}\n\nrule sales.q: sales.order_id is positive\n")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("Workspace accepted a rule declaration through the semantic loader")
	}
	diag, ok := workspace.SourceDiagnosticFromError(root, err)
	if !ok {
		t.Fatalf("rejection was not converted to a source diagnostic: %v", err)
	}
	if diag.Source.ArtifactPath != "semantic/sales/validation.tei" {
		t.Fatalf("artifact path=%q", diag.Source.ArtifactPath)
	}
	if !strings.Contains(diag.Message, "own file") {
		t.Fatalf("message=%q", diag.Message)
	}
	start, end := diag.Source.Range.Start.Offset, diag.Source.Range.End.Offset
	if start < 0 || start >= end || end > len(content) {
		t.Fatalf("range=%d:%d", start, end)
	}
	if got := string(content[start:end]); got != "rule" {
		t.Fatalf("range text=%q", got)
	}
}
