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
	file   string // workspace-relative .go source for actionable diagnostics
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
		fileRel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("invalid import path in %s: %w", path, err)
			}
			edges = append(edges, importEdge{source: source, file: filepath.ToSlash(fileRel), path: value})
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
		if edges[i].path != edges[j].path {
			return edges[i].path < edges[j].path
		}
		return edges[i].file < edges[j].file
	})
	return edges, nil
}


// readModulePath prevents a future go.mod module rename from silently
// bypassing first-party import checks that were written for "tei".
func readModulePath(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		module := fields[1]
		if strings.HasPrefix(module, "\"") {
			module, err = strconv.Unquote(module)
			if err != nil {
				return "", fmt.Errorf("invalid module directive: %w", err)
			}
		}
		if module == "" {
			return "", fmt.Errorf("empty module directive")
		}
		return module, nil
	}
	return "", fmt.Errorf("go.mod has no module directive")
}

// boundaryViolation deliberately guards only the small, demonstrated core.
// Plugin -> implementation-specific model coupling belongs to #951 and must
// not be hidden by a premature blanket restriction in this test.
func boundaryViolation(e importEdge, module string) string {
	for _, root := range []string{"cmd", "tools", "examples", "reference"} {
		prefix := module + "/" + root
		if e.path == prefix || strings.HasPrefix(e.path, prefix+"/") {
			return "internal packages must not import executable/reference packages"
		}
	}
	if e.path != module+"/internal" && !strings.HasPrefix(e.path, module+"/internal/") {
		return ""
	}
	group := strings.TrimPrefix(e.source, "internal/")
	group = strings.Split(group, "/")[0]
	switch group {
	case "core", "subject":
		return "foundational package must not depend on other TEI internal packages"
	case "permission":
		if e.path != module+"/internal/core" {
			return "permission model may depend only on core"
		}
	case "semantic":
		if e.path != module+"/internal/core" && e.path != module+"/internal/semantic" &&
			!strings.HasPrefix(e.path, module+"/internal/semantic/") {
			return "semantic model must not depend on realization or runtime packages"
		}
	case "stdlib":
		if e.path != module+"/internal/core" && e.path != module+"/internal/semantic" {
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
		{"internal/plugin", "tei/tools", true},
		{"internal/core", "tei/internal", true},
	}
	for _, tt := range tests {
		e := importEdge{source: tt.source, path: tt.imported}
		if got := boundaryViolation(e, "tei") != ""; got != tt.wantViolation {
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
	module, err := readModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := productionImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Fatal("no production imports inspected; dependency check is not meaningful")
	}
	for _, e := range edges {
		if reason := boundaryViolation(e, module); reason != "" {
			t.Errorf("%s imports %s: %s", e.file, e.path, reason)
		}
	}
}

func TestModulePathDetection(t *testing.T) {
	for _, tc := range []struct {
		name, contents, want string
	}{
		{"short", "module tei", "tei"},
		{"qualified", "module github.com/example/tei-platform", "github.com/example/tei-platform"},
		{"quoted", `module "example.com/tei"`, "example.com/tei"},
		{"missing", "go 1.23.2", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(tc.contents), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := readModulePath(root)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected missing module error; got %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("readModulePath() = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestProductionScannerFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "core")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "model.go")
	if err := os.WriteFile(src, []byte(`package core
import "tei/internal/plugin"
`), 0600); err != nil {
		t.Fatal(err)
	}
	// Test-only imports must not become production dependencies.
	if err := os.WriteFile(filepath.Join(dir, "model_test.go"), []byte("this is deliberately not valid Go"), 0600); err != nil {
		t.Fatal(err)
	}
	edges, err := productionImports(root)
	if err != nil || len(edges) != 1 || edges[0].file != "internal/core/model.go" || boundaryViolation(edges[0], "tei") == "" {
		t.Fatalf("production import not detected: edges=%v, err=%v", edges, err)
	}
	if err := os.WriteFile(src, []byte(`package core
import (
`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := productionImports(root); err == nil {
		t.Fatal("malformed production source must fail the check")
	}
}

func TestBoundaryAfterModuleRename(t *testing.T) {
	changed := importEdge{source: "internal/core", path: "github.com/example/tei-platform/internal/plugin"}
	if got := boundaryViolation(changed, "github.com/example/tei-platform"); got == "" {
		t.Fatal("module-qualified import must not bypass boundary guard")
	}
	if got := boundaryViolation(changed, "tei"); got != "" {
		t.Fatalf("third-party import must not be treated as local: %s", got)
	}
}
