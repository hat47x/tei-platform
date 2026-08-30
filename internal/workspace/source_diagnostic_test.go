package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestWorkspaceParseErrorBecomesRelativeSourceDiagnostic(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "applications", "order-entry", "application.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("application order_entry { unknown }\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("expected workspace load error")
	}
	diagnostic, ok := workspace.SourceDiagnosticFromError(root, err)
	if !ok {
		t.Fatalf("load error was not converted to source diagnostic: %T %v", err, err)
	}
	if diagnostic.Code != "PARSE_ERROR" {
		t.Fatalf("code=%q", diagnostic.Code)
	}
	if diagnostic.Source.ArtifactPath != "applications/order-entry/application.tei" {
		t.Fatalf("artifact path=%q", diagnostic.Source.ArtifactPath)
	}
	start, end := diagnostic.Source.Range.Start.Offset, diagnostic.Source.Range.End.Offset
	if got := string(content[start:end]); got != "unknown" {
		t.Fatalf("range text=%q range=%+v", got, diagnostic.Source.Range)
	}
	if diagnostic.Message != "expected database or uses" {
		t.Fatalf("message=%q", diagnostic.Message)
	}
}

func TestWorkspaceLexerErrorBecomesSourceDiagnosticWithoutSyntheticSubject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "applications", "order-entry", "application.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("application order_entry { @ }\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("expected workspace load error")
	}
	diagnostic, ok := workspace.SourceDiagnosticFromError(root, err)
	if !ok {
		t.Fatalf("load error was not converted: %T %v", err, err)
	}
	start, end := diagnostic.Source.Range.Start.Offset, diagnostic.Source.Range.End.Offset
	if got := string(content[start:end]); got != "@" {
		t.Fatalf("range text=%q range=%+v", got, diagnostic.Source.Range)
	}
}
