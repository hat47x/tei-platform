package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanWebTargetAuthorityAcceptsPresentationOnlySource(t *testing.T) {
	dir := t.TempDir()
	writeArchitectureFile(t, filepath.Join(dir, "main.go"), `package demo

const message = "selected order"
`)
	writeArchitectureFile(t, filepath.Join(dir, "README.md"), "Custom UI shares a command runner.
")

	violations, err := scanWebTargetAuthority(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("presentation-only source violations=%v", violations)
	}
}

func TestScanWebTargetAuthorityRejectsTEISQLAndNativeSQL(t *testing.T) {
	dir := t.TempDir()
	writeArchitectureFile(t, filepath.Join(dir, "authoritative.tei"), "application order_entry {}
")
	writeArchitectureFile(t, filepath.Join(dir, "query.sql"), "SELECT order_id FROM orders;
")
	writeArchitectureFile(t, filepath.Join(dir, "embedded.go"), `package demo

const query = "UPDATE orders SET status = 'done'"
`)

	violations, err := scanWebTargetAuthority(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 3 {
		t.Fatalf("authority violations=%v want=3", violations)
	}
	joined := strings.Join(violations, "
")
	for _, want := range []string{
		"authoritative .tei file",
		"native .sql file",
		"native SQL statement text detected",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("authority violations do not contain %q: %s", want, joined)
		}
	}
}

func TestRequireFixedGoEnvironment(t *testing.T) {
	t.Setenv("GOENV", "off")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	if err := requireFixedGoEnvironment(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOFLAGS", "-tags=hidden")
	if err := requireFixedGoEnvironment(); err == nil {
		t.Fatal("non-empty GOFLAGS was accepted")
	}
}

func writeArchitectureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
