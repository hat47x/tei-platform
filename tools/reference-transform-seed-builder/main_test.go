package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/plugin/transform"
)

func TestSeedBuilderMaterializesDeterministicTransformSource(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source.json")
	firstPath := filepath.Join(root, "first.json")
	secondPath := filepath.Join(root, "second.json")

	raw, err := json.MarshalIndent(transform.TransformSourceRealizerV1(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(sourcePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(sourcePath, firstPath); err != nil {
		t.Fatal(err)
	}
	if err := run(sourcePath, secondPath); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("stage0 seed builder produced different realization bytes")
	}
	if _, err := transform.ParseSourceJSON(first); err != nil {
		t.Fatalf("seed builder output is not a valid transform Source: %v", err)
	}
}

func TestSeedBuilderRejectsInvalidTransformSource(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source.json")
	outputPath := filepath.Join(root, "out.json")
	if err := os.WriteFile(sourcePath, []byte(`{"unknown":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(sourcePath, outputPath); err == nil {
		t.Fatal("expected invalid transform Source to fail")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("seed builder created output after failed validation: %v", err)
	}
}
