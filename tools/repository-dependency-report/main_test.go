package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                         "module example.org/tei\n\ngo 1.23.2\n",
		"internal/core/id.go":            "package core\n",
		"internal/features/a.go":         "package features\nimport \"example.org/tei/internal/core\"\n",
		"internal/features/b.go":         "package features\nimport \"example.org/tei/internal/core\"\n",
		"internal/host/host.go":          "package host\nimport (\"example.org/tei/internal/core\"; \"example.org/tei/internal/features\")\n",
		"internal/host/host_test.go":     "not valid Go, must be ignored",
		"internal/host/testdata/mock.go": "not valid Go, must be ignored",
	}
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSourceDependencyGraph(t *testing.T) {
	root := fixture(t)
	r, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Schema != "tei.repository.direct-import-graph/v1" || r.Module != "example.org/tei" || r.ProductionSourceFiles != 4 {
		t.Fatalf("unexpected inventory: %+v", r)
	}
	if len(r.Packages) != 3 || len(r.Dependencies) != 3 {
		t.Fatalf("unexpected graph: %+v", r)
	}
	for _, p := range r.Packages {
		if p.Path == "internal/core" && (p.DirectDependents != 2 || p.DirectDependencies != 0) {
			t.Fatalf("unexpected core degree: %+v", p)
		}
		if p.Path == "internal/features" && (p.DirectDependents != 1 || p.DirectDependencies != 1) {
			t.Fatalf("unexpected feature degree: %+v", p)
		}
	}
	if got := r.Dependencies[0]; got.From != "internal/features" || got.To != "internal/core" || !reflect.DeepEqual(got.SourceFiles, []string{"internal/features/a.go", "internal/features/b.go"}) {
		t.Fatalf("noncanonical sources: %+v", got)
	}
	r2, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(r2)
	if string(a) != string(b) {
		t.Fatalf("report is nondeterministic: %s != %s", a, b)
	}
}

func TestSourceGraphFailsClosed(t *testing.T) {
	root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "internal", "core", "id.go"), []byte("package core\nimport ("), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := collect(root); err == nil {
		t.Fatal("malformed production source must fail")
	}
	if _, err := collect(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing repository must fail")
	}
}

func TestUnknownLocalImportFailsClosed(t *testing.T) {
	root := fixture(t)
	path := filepath.Join(root, "internal", "host", "host.go")
	if err := os.WriteFile(path, []byte("package host\nimport \"example.org/tei/internal/not-present\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := collect(root); err == nil {
		t.Fatal("missing first-party import target must not appear as a valid package")
	}
}
