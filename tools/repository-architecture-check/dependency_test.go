package architecturecheck

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
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
	return productionImportsWithin(root, "internal")
}

// productionImportsWithin inspects actual production Go imports, omitting
// test-only and ignored Go source. It intentionally does not apply a policy.
func productionImportsWithin(root, subtree string) ([]importEdge, error) {
	var edges []importEdge
	err := filepath.WalkDir(filepath.Join(root, subtree), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != filepath.Join(root, subtree) {
				name := d.Name()
				if name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
					return fs.SkipDir
				}
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_") ||
			!strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
	case "core", "subject", "composition":
		return "foundational or composition package must not depend on other TEI internal packages"
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

// coreBoundaryViolation protects the migrated source-of-truth packages from
// dependencies on TEI's host, plugin, tooling and reference implementations.
// It deliberately permits other core packages and third-party imports: an
// isolated build proves resolvable dependencies; this is only a source boundary.
func coreBoundaryViolation(e importEdge, module string) string {
	if e.source != "core" && !strings.HasPrefix(e.source, "core/") {
		return ""
	}
	if !strings.HasPrefix(e.path, module+"/") {
		return ""
	}
	local := strings.TrimPrefix(e.path, module+"/")
	if local == "core" || strings.HasPrefix(local, "core/") {
		return ""
	}
	return "Core must not import TEI implementation, tooling or reference packages outside core/"
}

func TestCoreBoundaryViolationClassification(t *testing.T) {
	for _, tc := range []struct {
		source, imported string
		violation        bool
	}{
		{"core/identity", "fmt", false},
		{"core/generation", "tei/core/effect", false},
		{"core/generation", "tei/core", false},
		{"core/identity", "github.com/thirdparty/example", false},
		{"core/identity", "tei/internal/core", true},
		{"core/generation", "tei/internal/composition", true},
		{"core/generation", "tei/plugins/order-quote", true},
		{"core/subject", "tei/reference/composition-kernel", true},
		{"core/effect", "tei/cmd/tei", true},
		{"core/effect", "tei/tools/diagnostics", true},
		{"core/identity", "tei/corelike", true},
		{"internal/composition", "tei/internal/plugin", false},
	} {
		got := coreBoundaryViolation(importEdge{source: tc.source, path: tc.imported}, "tei")
		if (got != "") != tc.violation {
			t.Errorf("%s -> %s: violation=%q, want %v", tc.source, tc.imported, got, tc.violation)
		}
	}
	changed := importEdge{source: "core/generation", path: "github.com/example/tei-platform/internal/plugin"}
	if coreBoundaryViolation(changed, "github.com/example/tei-platform") == "" {
		t.Fatal("module rename bypassed Core boundary")
	}
	if coreBoundaryViolation(changed, "tei") != "" {
		t.Fatal("unrelated module incorrectly rejected")
	}
}

func TestCoreProductionDependencyBoundaries(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	module, err := readModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := productionImportsWithin(root, "core")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Fatal("no Core production imports inspected")
	}
	for _, e := range edges {
		if reason := coreBoundaryViolation(e, module); reason != "" {
			t.Errorf("%s imports %s: %s", e.file, e.path, reason)
		}
	}
}

func TestCoreProductionScannerFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "core", "generation")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "router.go")
	if err := os.WriteFile(source, []byte(`package generation
import "tei/internal/plugin"
`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "router_test.go"), []byte("not valid Go"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"testdata", "vendor", "_ignored"} {
		ignored := filepath.Join(dir, name)
		if err := os.MkdirAll(ignored, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ignored, "fake.go"), []byte("not valid Go"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := productionImportsWithin(root, "core")
	if err != nil || len(edges) != 1 || edges[0].file != "core/generation/router.go" || coreBoundaryViolation(edges[0], "tei") == "" {
		t.Fatalf("Core import not rejected: edges=%v, err=%v", edges, err)
	}
	if err := os.WriteFile(source, []byte(`package generation
import (
`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := productionImportsWithin(root, "core"); err == nil {
		t.Fatal("malformed Core production source must fail closed")
	}
}

func TestBoundaryViolationClassification(t *testing.T) {
	tests := []struct {
		source, imported string
		wantViolation    bool
	}{
		{"internal/core", "fmt", false},
		{"internal/composition", "context", false},
		{"internal/composition", "tei/internal/core", true},
		{"internal/composition", "tei/internal/plugin", true},
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
		{"internal/core", "tei/internal/core", true},
		{"internal/semantic/capabilityinvoke", "tei/internal/semantic/other", false},
		{"internal/stdlib", "tei/internal/semantic/other", true},
		{"internal/permission", "tei/internal/core/other", true},
		{"internal/plugin", "tei/reference", true},
		{"internal/plugin", "tei/examples", true},
		{"internal/plugin", "tei/cmd", true},
		{"internal/plugin", "tei/toolsmith", false},
		{"internal/plugin", "example.com/tei/tools", false},
	}
	for _, tt := range tests {
		e := importEdge{source: tt.source, path: tt.imported}
		if got := boundaryViolation(e, "tei") != ""; got != tt.wantViolation {
			t.Errorf("boundaryViolation(%s -> %s)=%v, want %v", tt.source, tt.imported, got, tt.wantViolation)
		}
	}
}

func TestProductionDependencyBoundaries(t *testing.T) {
	// Go test starts each package with its source directory as the working
	// directory. Unlike runtime.Caller, this is also reliable with -trimpath.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
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
	// Go excludes testdata, vendor and underscore-prefixed source from package builds.
	if err := os.WriteFile(filepath.Join(dir, "_ignored.go"), []byte("not valid Go"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"testdata", "vendor"} {
		ignored := filepath.Join(dir, name)
		if err := os.MkdirAll(ignored, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ignored, "model.go"), []byte("not valid Go"), 0600); err != nil {
			t.Fatal(err)
		}
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
