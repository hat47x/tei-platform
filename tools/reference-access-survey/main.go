// Command reference-access-survey counts where code touches the outside world
// directly, instead of through internal/access.
//
// The permission adoption plan says adapters must eventually reach the guarded
// layer, and that adapters which do not yet do so must be recorded as such. That
// plan needs a number that can be recomputed, not a number typed once into a
// document: a count nobody can re-derive stops being evidence the moment the
// code changes.
//
// The survey reads Go source with the parser rather than with a regular
// expression, so a call inside a comment, a string, or a build-tagged file is
// counted as what it is.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// category groups a call by the resource it touches. The three categories are
// the permission kinds: a survey that mixed them would hide a network call
// behind a file count.
type category string

const (
	categoryFilesystem  category = "filesystem"
	categoryNetwork     category = "network"
	categoryEnvironment category = "environment"
)

// watched names the calls the survey counts. Only direct access is listed: a
// call that already goes through internal/access is the guarded path and is not
// what this survey is looking for.
var watched = map[string]struct {
	category category
	kind     string
}{
	"os.ReadFile":                {categoryFilesystem, "read"},
	"os.WriteFile":               {categoryFilesystem, "write"},
	"os.OpenFile":                {categoryFilesystem, "open"},
	"os.Create":                  {categoryFilesystem, "create"},
	"os.Open":                    {categoryFilesystem, "open-existing"},
	"os.Remove":                  {categoryFilesystem, "remove"},
	"os.RemoveAll":               {categoryFilesystem, "remove-tree"},
	"os.Mkdir":                   {categoryFilesystem, "create-directory"},
	"os.MkdirAll":                {categoryFilesystem, "create-directory-tree"},
	"os.Rename":                  {categoryFilesystem, "rename"},
	"os.ReadDir":                 {categoryFilesystem, "list"},
	"net.Dial":                   {categoryNetwork, "connect"},
	"net.DialTimeout":            {categoryNetwork, "connect"},
	"net.Listen":                 {categoryNetwork, "listen"},
	"http.Get":                   {categoryNetwork, "request"},
	"http.Post":                  {categoryNetwork, "request"},
	"http.NewRequest":            {categoryNetwork, "request"},
	"http.NewRequestWithContext": {categoryNetwork, "request"},
	"os.Getenv":                  {categoryEnvironment, "variable"},
	"os.Environ":                 {categoryEnvironment, "variables"},
	"os.LookupEnv":               {categoryEnvironment, "variable"},
	"os.ExpandEnv":               {categoryEnvironment, "expansion"},
}

// accessPackage is the guarded layer. A file that imports it is on the way to
// enforcement; a file that does not is still direct access.
const accessPackage = "tei/internal/access"

// Call is one direct access found in a file.
type Call struct {
	Call     string `json:"call"`
	Category string `json:"category"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
}

// File is one file and the direct accesses it contains.
type File struct {
	Path      string   `json:"path"`
	Package   string   `json:"package,omitempty"`
	Calls     []Call   `json:"calls"`
	Guarded   bool     `json:"guarded"`
	BuildTags []string `json:"buildTags,omitempty"`
	TestFile  bool     `json:"testFile,omitempty"`
}

// Report is the survey result.
type Report struct {
	Roots       []string       `json:"roots"`
	Files       []File         `json:"files"`
	Totals      Totals         `json:"totals"`
	ByDirectory []DirectoryUse `json:"byDirectory"`
}

// Totals are the numbers a reader can quote.
type Totals struct {
	FilesScanned    int            `json:"filesScanned"`
	FilesWithAccess int            `json:"filesWithAccess"`
	FilesGuarded    int            `json:"filesGuarded"`
	DirectCalls     int            `json:"directCalls"`
	ByCategory      map[string]int `json:"byCategory"`
	ByCall          map[string]int `json:"byCall"`
}

// DirectoryUse is the count per directory, most first.
type DirectoryUse struct {
	Directory string `json:"directory"`
	Files     int    `json:"files"`
	Calls     int    `json:"calls"`
	Guarded   int    `json:"guarded"`
}

// inspect reads one file and reports its direct accesses.
func inspect(path string, content []byte) (File, error) {
	file := File{Path: filepath.ToSlash(path), Calls: []Call{}}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	file.Package = parsed.Name.Name
	file.TestFile = strings.HasSuffix(path, "_test.go")
	file.BuildTags = buildTags(parsed, fset)
	for _, spec := range parsed.Imports {
		if strings.Trim(spec.Path.Value, `"`) == accessPackage {
			file.Guarded = true
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		entry, found := watched[ident.Name+"."+selector.Sel.Name]
		if !found {
			return true
		}
		file.Calls = append(file.Calls, Call{
			Call:     ident.Name + "." + selector.Sel.Name,
			Category: string(entry.category),
			Kind:     entry.kind,
			Line:     fset.Position(selector.Pos()).Line,
		})
		return true
	})
	sort.SliceStable(file.Calls, func(i, j int) bool { return file.Calls[i].Line < file.Calls[j].Line })
	return file, nil
}

// buildTags reads the build constraints of a file. A file that only builds on
// one platform still counts, but recording the tag keeps the number honest.
func buildTags(parsed *ast.File, fset *token.FileSet) []string {
	var tags []string
	for _, group := range parsed.Comments {
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//go:build") {
				continue
			}
			tags = append(tags, strings.TrimSpace(strings.TrimPrefix(comment.Text, "//go:build")))
		}
	}
	return tags
}

// survey walks the roots and builds the report.
func survey(roots []string, includeTests bool) (Report, error) {
	report := Report{Roots: roots, Files: []File{}, Totals: Totals{ByCategory: map[string]int{}, ByCall: map[string]int{}}}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// Vendored and generated trees are not ours to change.
				switch entry.Name() {
				case "vendor", "node_modules", "testdata", ".git":
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if !includeTests && strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			file, err := inspect(path, content)
			if err != nil {
				return err
			}
			report.Totals.FilesScanned++
			if len(file.Calls) == 0 {
				return nil
			}
			report.Files = append(report.Files, file)
			report.Totals.FilesWithAccess++
			if file.Guarded {
				report.Totals.FilesGuarded++
			}
			for _, call := range file.Calls {
				report.Totals.DirectCalls++
				report.Totals.ByCategory[call.Category]++
				report.Totals.ByCall[call.Call]++
			}
			return nil
		})
		if err != nil {
			return Report{}, err
		}
	}
	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Path < report.Files[j].Path })
	report.ByDirectory = directoryUse(report.Files)
	return report, nil
}

// directoryUse groups the per-file counts by directory, because "626 accesses"
// says nothing actionable until a reader knows which directory holds them.
func directoryUse(files []File) []DirectoryUse {
	type counts struct{ files, calls, guarded int }
	byDirectory := map[string]*counts{}
	for _, file := range files {
		directory := filepath.ToSlash(filepath.Dir(file.Path))
		entry, ok := byDirectory[directory]
		if !ok {
			entry = &counts{}
			byDirectory[directory] = entry
		}
		entry.files++
		entry.calls += len(file.Calls)
		if file.Guarded {
			entry.guarded++
		}
	}
	result := make([]DirectoryUse, 0, len(byDirectory))
	for directory, entry := range byDirectory {
		result = append(result, DirectoryUse{Directory: directory, Files: entry.files, Calls: entry.calls, Guarded: entry.guarded})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Calls != result[j].Calls {
			return result[i].Calls > result[j].Calls
		}
		return result[i].Directory < result[j].Directory
	})
	return result
}

// render writes the report. The JSON is the artifact a caller compares between
// runs, so the field order is fixed by the struct, not by map iteration.
func render(report Report) ([]byte, error) {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode survey: %w", err)
	}
	return append(raw, '\n'), nil
}

func main() {
	out := flag.String("out", "", "write the survey here instead of stdout")
	includeTests := flag.Bool("tests", false, "count test files too")
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"internal", "cmd", "tools"}
	}
	report, err := survey(roots, *includeTests)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, err := render(report)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *out == "" {
		_, _ = fmt.Print(string(raw))
		return
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
