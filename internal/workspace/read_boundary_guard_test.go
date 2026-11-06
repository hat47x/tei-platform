package workspace

// Architecture guard for the workspace read boundary (#1017 slice 2).
//
// Reading workspace bytes must go through the audited, root-pinned readers
// (workspace.Load's os.Root, ReadSource, the source snapshot). A new Provider
// or loader that reaches the filesystem directly would silently bypass that
// boundary (symlink or alias swaps after validation, unpinned roots). This test
// parses every non-test Go file of THIS package directory with go/ast and fails
// on any use of the entry points in forbiddenReadCalls, or any import in
// forbiddenReadImports, that is not explicitly allowlisted below. An allowlist
// entry names file + function + call and a count, so adding a second call in an
// already-allowed function also fails. Stale entries fail too, so the list
// cannot rot. The audited reader files are pinned by EXACT name
// (auditedReaderFiles): a new file that merely looks like one is scanned, and a
// removed one fails the guard.
//
// What this guard does NOT do (so nobody reads it as more than it is):
//   - It is a static, name-based check of this directory only. Subpackages and
//     other packages (for example internal/operations) are out of scope.
//   - It lists named entry points. A filesystem API that is not in the lists
//     (for example through golang.org/x/sys, cgo, reflection, or a function
//     obtained from another package) is not detected, and neither is a read
//     through a handle that was opened elsewhere.
//   - Allowed sites are reviewed, not proven safe, and a new caller of an
//     allowed helper such as pinnedDirMatches is not detected.
//   - It is not human acceptance of the boundary.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// forbiddenReadCalls are the filesystem entry points that bypass the audited
// reader, keyed by import path then function name. io/fs functions are listed
// because they act on whatever fs.FS they are given: an allowed use must walk
// a pinned Root.FS().
var forbiddenReadCalls = map[string]map[string]bool{
	"os": {
		"ReadFile": true, "Open": true, "OpenFile": true, "Stat": true, "Lstat": true,
		"ReadDir": true, "DirFS": true, "Readlink": true, "OpenRoot": true, "NewFile": true,
	},
	"path/filepath": {
		"Glob": true, "Walk": true, "WalkDir": true, "EvalSymlinks": true,
	},
	"io/ioutil": {
		"ReadFile": true, "ReadDir": true,
	},
	"io/fs": {
		"ReadFile": true, "ReadDir": true, "Stat": true, "WalkDir": true, "Glob": true, "Sub": true,
	},
}

// forbiddenReadImports are packages whose mere import in a non-audited file is
// a violation: they reach the filesystem below the lists above.
var forbiddenReadImports = map[string]bool{
	"syscall": true, "os/exec": true, "unsafe": true,
	"golang.org/x/sys/unix": true, "golang.org/x/sys/windows": true,
}

// auditedReaderFiles are reviewed reader implementations, pinned by exact file
// name. Every use inside them is allowed because they are the audited boundary.
// A listed file that no longer exists fails the guard; a new file is NOT
// exempt just because its name resembles one of these.
var auditedReaderFiles = map[string]string{
	"source_snapshot.go":         "audited source snapshot reader; EvalSymlinks only resolves the absolute path before the pinned open (see its fail-closed comment)",
	"source_read.go":             "ReadSource: Normalize + os.Root + regular-file Stat + bounded read",
	"source_open_root.go":        "rooted open helper (os.OpenRoot) used by the checked reader",
	"source_open_unavailable.go": "fail-closed stub for targets without a trustworthy rooted open",
	"source_read_open_unix.go":   "platform file-open helper for the rooted reader (unix)",
	"source_read_open_other.go":  "platform file-open helper for the rooted reader (other targets)",
}

type readCallSite struct {
	file, function, call string
}

func (s readCallSite) String() string { return s.file + " " + s.function + " " + s.call }

// allowedReadCallSites lists each reviewed non-reader call site individually.
var allowedReadCallSites = map[readCallSite]struct {
	count  int
	reason string
}{
	{"loader.go", "resolveNativeSQLFile", "filepath.EvalSymlinks"}: {2,
		"advisory path containment check for native SQL files (root and candidate); the bytes are not read here"},
	{"loader.go", "resolveNativeSQLFile", "os.Stat"}: {1,
		"advisory regular-file check on the already resolved native SQL path; no content is read"},
	{"loader.go", "pinnedDirMatches", "fs.ReadDir"}: {1,
		"pinned listing: reads a directory of the Load-time Root.FS() (or a temporary rooted handle for direct per-domain loader tests), never the Workspace pathname; the only pattern matched is a fixed file-name pattern"},
	{"loader.go", "scanManagedFiles", "fs.WalkDir"}: {1,
		"pinned listing: walks rootHandle.FS() (the Load-time os.Root), so every traversal stays inside the pinned directory handle"},
	{"loader.go", "listPinnedDomainSources", "fs.WalkDir"}: {1,
		"pinned listing: walks the Load-time Root.FS() (or a temporary rooted handle for direct per-domain loader tests)"},
	{"loader.go", "listPinnedDomainSources", "os.OpenRoot"}: {1,
		"opens a temporary rooted handle only when a direct per-domain loader test has no Load root; Load itself pins the root once"},
}

func isAuditedReaderFile(name string) bool {
	_, ok := auditedReaderFiles[name]
	return ok
}

// scanReadBoundary returns forbidden uses in one parsed Go file, keyed
// file+function+call with occurrence counts. Any selector use counts, not only
// direct calls, so `f := os.Stat` is caught too. Dot imports of the guarded
// packages are reported as a call named "<dot-import>".
func scanReadBoundary(name string, file *ast.File) (map[readCallSite]int, error) {
	names := map[string]string{} // local identifier -> import path
	out := map[readCallSite]int{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		if forbiddenReadImports[path] {
			out[readCallSite{name, "(file)", "import " + path}]++
			continue
		}
		if _, guarded := forbiddenReadCalls[path]; !guarded {
			continue
		}
		local := filepath.Base(path)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		switch local {
		case "_":
			continue
		case ".":
			out[readCallSite{name, "(file)", path + ".<dot-import>"}]++
			continue
		}
		names[local] = path
	}
	if len(names) == 0 {
		return out, nil
	}
	for _, decl := range file.Decls {
		fn := "(package)"
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fn = fd.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Obj != nil { // Obj != nil: a local variable shadowing the import
				return true
			}
			path, ok := names[id.Name]
			if !ok || !forbiddenReadCalls[path][sel.Sel.Name] {
				return true
			}
			out[readCallSite{name, fn, filepath.Base(path) + "." + sel.Sel.Name}]++
			return true
		})
	}
	return out, nil
}

// checkReadBoundary evaluates sources (file name -> content) against the
// allowlist. It returns violations and the observed counts outside the audited
// reader files.
func checkReadBoundary(sources map[string]string) ([]string, map[readCallSite]int) {
	var violations []string
	observed := map[readCallSite]int{}
	names := make([]string, 0, len(sources))
	for n := range sources {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") || isAuditedReaderFile(name) {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, sources[name], 0)
		if err != nil {
			violations = append(violations, fmt.Sprintf("%s: parse error: %v", name, err))
			continue
		}
		found, err := scanReadBoundary(name, f)
		if err != nil {
			violations = append(violations, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		for site, n := range found {
			observed[site] += n
			allowed, ok := allowedReadCallSites[site]
			if !ok {
				violations = append(violations, fmt.Sprintf("unreviewed unpinned read: %s x%d", site, n))
			} else if n > allowed.count {
				violations = append(violations, fmt.Sprintf("more unpinned reads than reviewed: %s x%d (allowed %d)", site, n, allowed.count))
			}
		}
	}
	sort.Strings(violations)
	return violations, observed
}

func TestWorkspaceReadBoundaryAllowlistHasReasons(t *testing.T) {
	for site, a := range allowedReadCallSites {
		if strings.TrimSpace(a.reason) == "" || a.count < 1 {
			t.Errorf("allowlist entry %s needs a reason and a positive count", site)
		}
	}
}

func TestWorkspaceProductionCodeUsesOnlyAuditedReadBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		sources[e.Name()] = string(b)
	}
	if len(sources) == 0 {
		t.Fatal("no production Go files found; the guard would prove nothing")
	}
	for name := range auditedReaderFiles {
		if _, ok := sources[name]; !ok {
			t.Errorf("audited reader file %s no longer exists; update auditedReaderFiles", name)
		}
	}
	violations, observed := checkReadBoundary(sources)
	for _, v := range violations {
		t.Error(v)
	}
	for site, a := range allowedReadCallSites {
		if observed[site] != a.count {
			t.Errorf("stale allowlist entry %s: reviewed %d, found %d; update or remove it", site, a.count, observed[site])
		}
	}
}

func TestWorkspaceReadBoundaryGuardRejectsNewUnpinnedReads(t *testing.T) {
	const head = "package workspace\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\nvar _ = os.Getpid\nvar _ = filepath.Join\n"
	const fsHead = "package workspace\nimport \"io/fs\"\n"
	cases := []struct {
		name    string
		file    string
		src     string
		wantHit string // substring of a violation; empty means clean
	}{
		{"os.ReadFile in new provider", "new_provider.go", head + "func load(p string) { _, _ = os.ReadFile(p) }", "new_provider.go load os.ReadFile"},
		{"os.Open", "x.go", head + "func f(p string) { _, _ = os.Open(p) }", "x.go f os.Open"},
		{"os.OpenFile", "x.go", head + "func f(p string) { _, _ = os.OpenFile(p, 0, 0) }", "os.OpenFile"},
		{"os.Stat", "x.go", head + "func f(p string) { _, _ = os.Stat(p) }", "os.Stat"},
		{"os.Lstat", "x.go", head + "func f(p string) { _, _ = os.Lstat(p) }", "os.Lstat"},
		{"filepath.Glob", "x.go", head + "func f(p string) { _, _ = filepath.Glob(p) }", "filepath.Glob"},
		{"filepath.Walk", "x.go", head + "func f(p string) { _ = filepath.Walk(p, nil) }", "filepath.Walk"},
		{"filepath.WalkDir", "x.go", head + "func f(p string) { _ = filepath.WalkDir(p, nil) }", "filepath.WalkDir"},
		{"filepath.EvalSymlinks", "x.go", head + "func f(p string) { _, _ = filepath.EvalSymlinks(p) }", "filepath.EvalSymlinks"},
		{"function value, not a call", "x.go", head + "var stat = os.Stat", "(package) os.Stat"},
		{"aliased import", "x.go", "package workspace\nimport o \"os\"\nfunc f(p string) { _, _ = o.ReadFile(p) }", "os.ReadFile"},
		{"dot import", "x.go", "package workspace\nimport . \"os\"\nvar _ = Getpid", "dot-import"},
		{"extra call in an allowed function", "loader.go", fsHead + "func pinnedDirMatches(v fs.FS, p string) { _, _ = fs.ReadDir(v, p); _, _ = fs.ReadDir(v, p) }", "more unpinned reads than reviewed"},
		{"allowed call moved to another function", "loader.go", fsHead + "func other(v fs.FS, p string) { _, _ = fs.ReadDir(v, p) }", "loader.go other fs.ReadDir"},
		{"removed fs.Glob is no longer allowed", "loader.go", fsHead + "func pinnedDirMatches(v fs.FS, p string) { _, _ = fs.Glob(v, p) }", "loader.go pinnedDirMatches fs.Glob"},
		{"removed filepath.Glob is no longer allowed", "loader.go", head + "func globSorted(p string) { _, _ = filepath.Glob(p) }", "loader.go globSorted filepath.Glob"},
		{"removed filepath.WalkDir is no longer allowed", "plugin_implementation_loader.go", head + "func loadPluginImplementations(p string) { _ = filepath.WalkDir(p, nil) }", "loadPluginImplementations filepath.WalkDir"},
		{"allowed function name in another file", "new_loader.go", fsHead + "func pinnedDirMatches(v fs.FS, p string) { _, _ = fs.ReadDir(v, p) }", "new_loader.go pinnedDirMatches"},
		{"unguarded calls are fine", "x.go", head + "func f(p string) { _ = filepath.Join(p, \"a\") }", ""},
		{"os.OpenRoot in a new file", "x.go", head + "func f(p string) { _, _ = os.OpenRoot(p) }", "x.go f os.OpenRoot"},
		{"os.ReadDir", "x.go", head + "func f(p string) { _, _ = os.ReadDir(p) }", "os.ReadDir"},
		{"os.DirFS", "x.go", head + "func f(p string) { _ = os.DirFS(p) }", "os.DirFS"},
		{"os.Readlink", "x.go", head + "func f(p string) { _, _ = os.Readlink(p) }", "os.Readlink"},
		{"io/ioutil.ReadFile", "x.go", "package workspace\nimport \"io/ioutil\"\nfunc f(p string) { _, _ = ioutil.ReadFile(p) }", "ioutil.ReadFile"},
		{"io/fs.ReadFile", "x.go", "package workspace\nimport \"io/fs\"\nfunc f(v fs.FS) { _, _ = fs.ReadFile(v, \"a\") }", "fs.ReadFile"},
		{"io/fs.WalkDir in a new file", "x.go", "package workspace\nimport \"io/fs\"\nfunc f(v fs.FS) { _ = fs.WalkDir(v, \".\", nil) }", "x.go f fs.WalkDir"},
		{"syscall import", "x.go", "package workspace\nimport \"syscall\"\nvar _ = syscall.Getpid", "import syscall"},
		{"os/exec import", "x.go", "package workspace\nimport \"os/exec\"\nvar _ = exec.Command", "import os/exec"},
		{"unsafe import", "x.go", "package workspace\nimport \"unsafe\"\nvar _ = unsafe.Sizeof(0)", "import unsafe"},
		{"new file named like an audited reader is scanned", "source_read_helper.go", head + "func f(p string) { _, _ = os.ReadFile(p) }", "source_read_helper.go f os.ReadFile"},
		{"new source_open file is scanned too", "source_open_extra.go", head + "func f(p string) { _, _ = os.Open(p) }", "source_open_extra.go f os.Open"},
		{"exact audited file may use the entry points", "source_read.go", head + "func f(p string) { _, _ = os.Open(p); _, _ = os.Stat(p) }", ""},
		{"local variable named os is not the package", "x.go", "package workspace\nimport \"os\"\nvar _ = os.Getpid\nfunc f() { os := struct{ ReadFile func() }{}; os.ReadFile() }", ""},
		{"test files are not scanned", "x_test.go", head + "func f(p string) { _, _ = os.ReadFile(p) }", ""},
		{"reviewed call stays allowed", "loader.go", fsHead + "func pinnedDirMatches(v fs.FS, p string) { _, _ = fs.ReadDir(v, p) }", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := checkReadBoundary(map[string]string{tc.file: tc.src})
			if tc.wantHit == "" {
				if len(got) != 0 {
					t.Fatalf("expected clean, got %v", got)
				}
				return
			}
			for _, v := range got {
				if strings.Contains(v, tc.wantHit) {
					return
				}
			}
			t.Fatalf("expected a violation containing %q, got %v", tc.wantHit, got)
		})
	}
}
