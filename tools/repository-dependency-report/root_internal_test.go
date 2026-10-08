package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A first-party package located directly under internal/ is still a direct
// dependency. Silently dropping it would make the graph incomplete.
func TestRootInternalDependencyIsIncluded(t *testing.T) {
	root := fixture(t)
	rootFile := filepath.Join(root, "internal", "root.go")
	if err := os.WriteFile(rootFile, []byte("package internal\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hostFile := filepath.Join(root, "internal", "host", "host.go")
	if err := os.WriteFile(hostFile, []byte("package host\nimport \"example.org/tei/internal\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, edge := range r.Dependencies {
		if edge.From == "internal/host" && edge.To == "internal" {
			found = true
		}
	}
	if !found {
		t.Fatalf("direct root internal import was dropped: %+v", r.Dependencies)
	}
}

func TestMissingRootInternalDependencyFailsClosed(t *testing.T) {
	root := fixture(t)
	hostFile := filepath.Join(root, "internal", "host", "host.go")
	if err := os.WriteFile(hostFile, []byte("package host\nimport \"example.org/tei/internal\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := collect(root); err == nil {
		t.Fatal("missing direct internal/ package must fail closed")
	}
}
