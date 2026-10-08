package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Package inventory must not confuse similarly prefixed external module names
// with this repository's first-party imports.
func TestExternalModulePrefixDoesNotCreateLocalDependency(t *testing.T) {
	root := fixture(t)
	host := filepath.Join(root, "internal", "host", "host.go")
	src := "package host\nimport (\n" +
		"\t\"example.org/tei-extra/internal/core\"\n" +
		"\t\"example.org/tei/internality/core\"\n" +
		"\t\"example.org/tei/internal/core\"\n" +
		")\n"
	if err := os.WriteFile(host, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range r.Dependencies {
		if strings.Contains(edge.To, "internality") || strings.Contains(edge.To, "tei-extra") {
			t.Fatalf("external import classified as first-party: %+v", edge)
		}
	}
	for _, edge := range r.Dependencies {
		if edge.From == "internal/host" && edge.To == "internal/core" {
			return
		}
	}
	t.Fatalf("actual first-party dependency missing: %+v", r.Dependencies)
}

// The inventory must remain portable if the repository module path changes.
func TestQualifiedModuleNameIsReadFromGoMod(t *testing.T) {
	root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module \"example.org/tei\"\n\ngo 1.23.2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Module != "example.org/tei" || len(r.Dependencies) != 3 {
		t.Fatalf("quoted module path changed dependency resolution: %+v", r)
	}
}
