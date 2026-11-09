package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransformRuntimeEngineSourcesCoverEveryProductionFile(t *testing.T) {
	root := filepath.Join("..", "..")
	paths, err := transformRuntimeEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(paths))
	for _, path := range paths {
		if got[path] {
			t.Fatalf("duplicate Runtime Engine source: %s", path)
		}
		got[path] = true
	}
	for _, path := range []string{
		"internal/core/id.go",
		"internal/core/version.go",
		"internal/plugin/transform/eval.go",
		"internal/plugin/transform/model.go",
		"internal/plugin/transform/source.go",
		"internal/plugin/transform/value.go",
		"internal/plugin/transform/source_budget.go",
		"internal/plugin/transform/json_unicode.go",
	} {
		if !got[path] {
			t.Errorf("Runtime Engine identity does not cover %s", path)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "internal", "plugin", "transform"))
	if err != nil {
		t.Fatal(err)
	}
	expected := 2 // CanonicalID and version validation dependencies.
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			expected++
			if !got["internal/plugin/transform/"+name] {
				t.Errorf("new production Go source is missing: %s", name)
			}
		}
	}
	if len(got) != expected {
		t.Fatalf("Runtime Engine source closure contains %d paths, expected %d", len(got), expected)
	}
	for path := range got {
		if strings.HasSuffix(path, "_test.go") {
			t.Errorf("test file incorrectly included in Runtime Engine identity: %s", path)
		}
	}
}

func TestTransformRuntimeEngineDigestRespondsToChangesAndNewSources(t *testing.T) {
	root := t.TempDir()
	put := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("internal/core/id.go", "id v1")
	put("internal/core/version.go", "version v1")
	put("internal/plugin/transform/model.go", "model v1")
	put("internal/plugin/transform/source_budget.go", "budget v1")
	digest := func() string {
		t.Helper()
		sources, err := transformRuntimeEngineSources(root)
		if err != nil {
			t.Fatal(err)
		}
		got, err := hashSourceFiles(root, sources)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	baseline := digest()
	put("internal/plugin/transform/source_budget.go", "budget v2")
	changedBudget := digest()
	if changedBudget == baseline {
		t.Fatal("resource-budget implementation change did not invalidate engine identity")
	}
	put("internal/plugin/transform/json_unicode.go", "canonicalization v1")
	changedCanonicalization := digest()
	if changedCanonicalization == changedBudget {
		t.Fatal("a new transform implementation did not invalidate engine identity")
	}
	put("internal/plugin/transform/transform_test.go", "test-only change")
	if got := digest(); got != changedCanonicalization {
		t.Fatal("a test-only change unexpectedly altered the engine implementation identity")
	}
	put("unrelated/documentation.txt", "unrelated change")
	if got := digest(); got != changedCanonicalization {
		t.Fatal("unrelated repository content changed engine implementation identity")
	}
}

func TestTransformRuntimeEngineSourcesRejectEmptyImplementation(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "plugin", "transform")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source_test.go"), []byte("test-only"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := transformRuntimeEngineSources(root); err == nil {
		t.Fatal("empty production Source must not yield an engine identity")
	}
}
