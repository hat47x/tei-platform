package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWorkspacePrintsParseErrorAtSourceRange(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: parse_diagnostic\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type }
}
`)

	var out bytes.Buffer
	code, err := validateWorkspace(&out, root)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d output=%q", code, out.String())
	}
	wantPrefix := "semantic/sales/order.tei:4:28\terror\tPARSE_ERROR\t"
	if !strings.HasPrefix(out.String(), wantPrefix) {
		t.Fatalf("output=%q want prefix=%q", out.String(), wantPrefix)
	}
}

func TestValidateWorkspaceProjectsSemanticDiagnosticToDefinition(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: semantic_diagnostic\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`)

	var out bytes.Buffer
	code, err := validateWorkspace(&out, root)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d output=%q", code, out.String())
	}
	want := "applications/order-entry/application.tei:1:13\terror\tINVALID_DATABASE_ACCESS\t"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("output=%q does not contain %q", out.String(), want)
	}
}

func TestValidateWorkspaceKeepsOkForValidSource(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: valid\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")

	var out bytes.Buffer
	code, err := validateWorkspace(&out, root)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit code=%d output=%q", code, out.String())
	}
	if out.String() != "ok\n" {
		t.Fatalf("output=%q", out.String())
	}
}

func writeWorkspaceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
