package architecturecheck

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// importEdge is a direct source import, not a transitive dependency or a proof
// that the imported package can execute with the importer's authority.
type importEdge struct {
	source string
	path   string
}

func productionImports(root string) ([]importEdge, error) {
	var edges []importEdge
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		source := filepath.ToSlash(rel)
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("invalid import path in %s: %w", path, err)
			}
			edges = append(edges, importEdge{source: source, path: value})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].source != edges[j].source {
			return edges[i].source < edges[j].source
		}
		return edges[i].path < edges[j].path
	})
	return edges, nil
}

// boundaryViolation deliberately guards only the small, demonstrated core.
// Plugin -> implementation-specific model coupling belongs to #951 and must
// not be hidden by a premature blanket restriction in this test.
func boundaryViolation(e importEdge) string {
	if strings.HasPrefix(e.path, "tei/cmd/") || strings.HasPrefix(e.path, "tei/tools/") ||
		strings.HasPrefix(e.path, "tei/examples/") || strings.HasPrefix(e.path, "tei/reference/") {
		return "internal packages must not import executable/reference packages"
	}
	if !strings.HasPrefix(e.path, "tei/internal/") {
		return ""
	}
	group := strings.TrimPrefix(e.source, "internal/")
	group = strings.Split(group, "/")[0]
	switch group {
	case "core", "subject":
		return "foundational package must not depend on other TEI internal packages"
	case "permission":
		if e.path != "tei/internal/core" {
			return "permission model may depend only on core"
		}
	case "semantic":
		if e.path != "tei/internal/core" && e.path != "tei/internal/semantic" &&
			!strings.HasPrefix(e.path, "tei/internal/semantic/") {
			return "semantic model must not depend on realization or runtime packages"
		}
	case "stdlib":
		if e.path != "tei/internal/core" && e.path != "tei/internal/semantic" {
			return "standard semantics may depend only on core and semantic model"
		}
	}
	return ""
}

func TestBoundaryViolationClassification(t *testing.T) {
	tests := []struct {
		source, imported string
		wantViolation     bool
	}{
		{"internal/core", "fmt", false},
		{"internal/core", "tei/internal/plugin", true},
		{"internal/subject", "tei/internal/plugin", true},
		{"internal/semantic/capabilityinvoke", "tei/internal/core", false},
		{"internal/semantic/capabilityinvoke", "tei/internal/semantic", false},
		{"internal/semantic", "tei/internal/database", true},
		{"internal/permission", "tei/internal/core", false},
		{"internal/permission", "tei/internal/deployment", true},
		{"internal/stdlib", "tei/internal/semantic", false},
		{"internal/stdlib", "tei/internal/plugin", true},
		{"internal/plugin", "tei/internal/database/migration", false},
		{"internal/plugin", "tei/tools/example", true},
		{"internal/plugin", "tei/cmd/tei", true},
	}
	for _, tt := range tests {
		e := importEdge{source: tt.source, path: tt.imported}
		if got := boundaryViolation(e) != ""; got != tt.wantViolation {
			t.Errorf("boundaryViolation(%s -> %s)=%v, want %v", tt.source, tt.imported, got, tt.wantViolation)
		}
	}
}

func TestProductionDependencyBoundaries(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository guard source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root unavailable: %v", err)
	}
	edges, err := productionImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Fatal("no production imports inspected; dependency check is not meaningful")
	}
	for _, e := range edges {
		if reason := boundaryViolation(e); reason != "" {
			t.Errorf("%s imports %s: %s", e.source, e.path, reason)
		}
	}
}
