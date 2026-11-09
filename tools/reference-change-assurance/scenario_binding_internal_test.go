package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadScenarioLocalAcceptsSlashSeparatedPath(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "fixtures", "recent_orders.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if _, err := readScenarioLocal(root, "fixtures/recent_orders.json"); err != nil {
		t.Fatalf("slash-separated local path was rejected: %v", err)
	}
	for _, filename := range []string{"./fixtures/recent_orders.json", "../recent_orders.json"} {
		if _, err := readScenarioLocal(root, filename); err == nil {
			t.Fatalf("unnormalized or escaping path %q was accepted", filename)
		}
	}
}
