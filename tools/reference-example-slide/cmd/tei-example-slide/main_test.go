package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyntheticExampleCreatesThreeSlides(t *testing.T) {
	out := t.TempDir()
	input := "../../testdata/partial_inspection.json"
	if err := run(input, "", false, out); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		"projection.json", "slide-001.svg", "slide-002.svg", "slide-003.svg",
	} {
		bytes, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		if len(bytes) == 0 {
			t.Fatalf("empty generated file %s", file)
		}
	}
}

func TestRealWorldDoesNotRewriteStaleCase(t *testing.T) {
	out := t.TempDir()
	input := "../../testdata/partial_inspection.json"
	world := "../../../../reference/world.json"
	err := run(input, world, false, out)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected World/example revision mismatch, got %v", err)
	}
	files, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("stale case produced output files: %d", len(files))
	}
}
