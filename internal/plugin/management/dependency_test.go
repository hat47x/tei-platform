package management

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbiddenPrefixes are transports and hosts a management projection must not
// reach: the CLI translates to and from the typed operation, and the Studio
// Bridge delegates, so neither may be imported here (#1158, #4376).
var forbiddenPrefixes = []string{
	"tei/cmd/",
	"tei/tools/",
	"tei/reference/",
	"flag",
	"net",
	"syscall/js",
	"html/template",
	"github.com/wailsapp",
	"golang.org/x/net",
}

// TestManagementDoesNotImportTransportOrReferenceTools reads this package's
// imports and rejects a forbidden prefix as a whole path element, so
// "network/other" is not mistaken for "net".
func TestManagementDoesNotImportTransportOrReferenceTools(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var imports []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, parseErr := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatalf("%s: %v", name, parseErr)
		}
		for _, spec := range f.Imports {
			path, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil {
				t.Fatalf("%s: %v", name, unquoteErr)
			}
			imports = append(imports, path)
		}
	}
	for _, dep := range imports {
		for _, prefix := range forbiddenPrefixes {
			trimmed := strings.TrimSuffix(prefix, "/")
			if dep == trimmed || strings.HasPrefix(dep, trimmed+"/") {
				t.Errorf("internal/plugin/management imports %q, which is a transport, host or reference-tool package", dep)
			}
		}
	}
}

// TestManagementImportsTheReferenceInventoryToolNowhere pins the migration
// rule: the three tools/ reference packages are specifications, not
// dependencies. A product import of them would make the reference tests the
// product behaviour.
func TestManagementImportsTheReferenceInventoryToolNowhere(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	forbiddenPackages := []string{
		"tei/tools/reference-studio-plugin-inventory",
		"tei/tools/reference-studio-plugin-disable",
		"tei/tools/reference-studio-plugin-data-lifecycle",
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "tools" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, forbidden := range forbiddenPackages {
			if strings.Contains(string(body), `"`+forbidden+`"`) {
				t.Errorf("%s imports %s; the reference implementation is a specification, not a product dependency", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
