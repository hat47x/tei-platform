package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestDiagnosticFromLoadErrorReturnsExactParserTokenRange(t *testing.T) {
	root := t.TempDir()
	manifest := []byte("workspace: parse_diagnostic\nsources:\n  - semantic\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	path := "semantic/sales/order.tei"
	source := []byte(`namespace sales

data Order {
    attribute orderId type }
}
`)
	writeSource(t, root, path, source)

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("invalid semantic source unexpectedly loaded")
	}
	diagnostic, ok := workspace.DiagnosticFromLoadError(root, err)
	if !ok {
		t.Fatalf("load error was not projected as source diagnostic: %v", err)
	}
	if diagnostic.Code != "PARSE_ERROR" {
		t.Fatalf("code=%q", diagnostic.Code)
	}
	if diagnostic.ArtifactPath != path {
		t.Fatalf("artifactPath=%q want=%q", diagnostic.ArtifactPath, path)
	}
	if diagnostic.Range.Start.Offset >= diagnostic.Range.End.Offset {
		t.Fatalf("parse error did not preserve token range: %+v", diagnostic.Range)
	}
	if got := string(source[diagnostic.Range.Start.Offset:diagnostic.Range.End.Offset]); got != "}" {
		t.Fatalf("range points to %q want closing brace; range=%+v", got, diagnostic.Range)
	}
	if diagnostic.Location() != "semantic/sales/order.tei:4:28" {
		t.Fatalf("location=%q", diagnostic.Location())
	}
}

func TestDiagnosticFromLoadErrorDoesNotInventSourceForOperationalError(t *testing.T) {
	if diagnostic, ok := workspace.DiagnosticFromLoadError(t.TempDir(), os.ErrNotExist); ok {
		t.Fatalf("operational error unexpectedly became source diagnostic: %+v", diagnostic)
	}
}
