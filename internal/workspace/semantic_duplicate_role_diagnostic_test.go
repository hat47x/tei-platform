package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// A duplicate Role must reach a real Workspace consumer as a source-direct
// diagnostic before it can be mistaken for a single accepted Role value.
func TestWorkspaceRejectsDuplicateSemanticRoleWithSourceLocation(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: role_duplicate\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "semantic", "order.tei")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	// CRLF matters: the diagnostic byte offset must point into the authored
	// bytes, not a normalized or pretty-printed interpretation.
	content := []byte("namespace sales\r\ndata Order {\r\n" +
		"  attribute total type std.money role accounting_amount\r\n" +
		"  # duplicated option after a comment\r\n" +
		"  role audit_amount\r\n}\r\n")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := workspace.Load(root); err == nil {
		t.Fatal("Workspace silently accepted duplicate Semantic Attribute Role")
	} else {
		diag, ok := workspace.SourceDiagnosticFromError(root, err)
		if !ok {
			t.Fatalf("Parser rejection was not converted to a source diagnostic: %v", err)
		}
		if diag.Code != "PARSE_ERROR" || diag.Source.ArtifactPath != "semantic/order.tei" {
			t.Fatalf("unexpected source diagnostic: %+v", diag)
		}
		want := strings.LastIndex(string(content), "role ")
		start, end := diag.Source.Range.Start.Offset, diag.Source.Range.End.Offset
		if start != want || start < 0 || end < start || end > len(content) ||
			string(content[start:end]) != "role" {
			t.Fatalf("diagnostic does not identify second Role token: offset=%d range=%+v", want, diag.Source.Range)
		}
		if !strings.Contains(diag.Message, "duplicate role option for attribute total") {
			t.Fatalf("duplicate Role was rejected for a different reason: %q", diag.Message)
		}
	}
}
