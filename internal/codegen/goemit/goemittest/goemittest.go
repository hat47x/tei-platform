// Package goemittest builds and runs code emitted by goemit in a module of its
// own, outside the TEI repository. Tests use it to show that the generated Go
// needs nothing but the standard library, and to run it against fixed input.
//
// The harness it writes reads the value of the attribute that Emit's Trace
// binds to the generated check function, so the binding a test observes comes
// from goemit, not from the test. Doc comments of the generated code are
// prose for people and are never parsed here.
package goemittest

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
)

// ModulePath is the module the generated code is built in.
const ModulePath = "example.com/generated"

// GoExecutable returns the go command, or skips the test when there is none.
func GoExecutable(t testing.TB) string {
	t.Helper()
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(runtime.GOROOT(), "bin", name)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	t.Skip("the go command is not available")
	return ""
}

// Env is the environment of a go command run in the standalone module. It
// replaces any -modfile or workspace setting of the calling test, and keeps
// the build offline.
func Env() []string {
	return append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"GOWORK=off",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
	)
}

// Files maps each emitted path to its content.
func Files(out goemit.Output) map[string][]byte {
	files := make(map[string][]byte, len(out.Files))
	for _, file := range out.Files {
		files[file.Path] = file.Content
	}
	return files
}

// WriteModule writes files and a go.mod for ModulePath into a new temporary
// directory and returns it. The directory is outside the repository, so
// nothing in it can resolve a tei/ import.
func WriteModule(t testing.TB, files map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	all := map[string][]byte{"go.mod": []byte("module " + ModulePath + "\n\ngo 1.23\n")}
	for name, content := range files {
		all[name] = content
	}
	for name, content := range all {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// RunGo runs the go command in dir with Env and fails the test on error. It
// returns the combined output.
func RunGo(t testing.TB, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(GoExecutable(t), args...)
	cmd.Dir = dir
	cmd.Env = Env()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s in the standalone module: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

// TEIImports returns every import path under tei/ found in the Go files.
func TEIImports(files map[string][]byte) ([]string, error) {
	var found []string
	for name, content := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, content, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if path == "tei" || strings.HasPrefix(path, "tei/") {
				found = append(found, name+": "+path)
			}
		}
	}
	sort.Strings(found)
	return found, nil
}

// Binding is what a generated check function receives: the value of
// Attribute, through its one parameter Parameter.
type Binding struct {
	Check     string
	Parameter string
	Attribute string
}

// Bindings returns one binding per emitted rule, taken from the structured
// Trace that Emit returns, never from comment prose. Each one is checked
// against the generated file: the file must declare a function named Check
// that takes one *big.Rat named Parameter and returns Result.
func Bindings(out goemit.Output) ([]Binding, error) {
	if len(out.Trace) == 0 {
		return nil, errors.New("no emitted rule")
	}
	var bindings []Binding
	for _, trace := range out.Trace {
		src := out.File(trace.File)
		if src == nil {
			return nil, fmt.Errorf("trace names file %s, which was not emitted", trace.File)
		}
		file, err := parser.ParseFile(token.NewFileSet(), trace.File, src, 0)
		if err != nil {
			return nil, err
		}
		found := false
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == trace.Check && isCheckSignature(fn.Type) &&
				fn.Type.Params.List[0].Names[0].Name == trace.Parameter {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%s declares no check function %s(%s *big.Rat) Result", trace.File, trace.Check, trace.Parameter)
		}
		bindings = append(bindings, Binding{Check: trace.Check, Parameter: trace.Parameter, Attribute: string(trace.Attribute)})
	}
	return bindings, nil
}

func isCheckSignature(fn *ast.FuncType) bool {
	if fn.Params == nil || len(fn.Params.List) != 1 || len(fn.Params.List[0].Names) != 1 ||
		fn.Results == nil || len(fn.Results.List) != 1 {
		return false
	}
	star, ok := fn.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Rat" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	result, okResult := fn.Results.List[0].Type.(*ast.Ident)
	return ok && pkg.Name == "big" && okResult && result.Name == "Result"
}

// Cell is one value of a record. A Null cell holds no value.
type Cell struct {
	Text string
	Null bool
}

// Case is one record given to the harness. An attribute missing from Row was
// not fetched. Like a null cell, it reaches the check function as nil: the
// generated code cannot tell the two apart, and both are undecided.
type Case struct {
	Name string
	Row  map[string]Cell
}

// harnessMain is the same for every test. It reads the bound attribute of
// each case once, parses it with the generated ParseDecimal and prints
// "<case>:<result>:<reads>:<attribute read>". Text that ParseDecimal refuses
// prints "error".
const harnessMain = `package main

import (
	"fmt"
	"math/big"
)

type cell struct {
	text string
	null bool
}

type record struct {
	row       map[string]cell
	reads     int
	requested string
}

func (r *record) number(attribute string) (*big.Rat, error) {
	r.reads++
	r.requested = attribute
	c, fetched := r.row[attribute]
	if !fetched || c.null {
		return nil, nil
	}
	return ParseDecimal(c.text)
}

func main() {
	for _, c := range cases {
		r := &record{row: c.row}
		value, err := r.number(boundAttribute)
		result := "error"
		if err == nil {
			result = string(boundCheck(value))
		}
		fmt.Printf("%s:%s:%d:%s\n", c.name, result, r.reads, r.requested)
	}
}
`

// HarnessFiles returns main.go, cases.go and binding.go of package main. They
// are compiled together with the generated files of package main.
func HarnessFiles(binding Binding, cases []Case) map[string][]byte {
	var b strings.Builder
	b.WriteString("package main\n\nvar cases = []struct {\n\tname string\n\trow  map[string]cell\n}{\n")
	for _, c := range cases {
		fmt.Fprintf(&b, "\t{name: %q, row: map[string]cell{", c.Name)
		keys := make([]string, 0, len(c.Row))
		for key := range c.Row {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "%q: {text: %q, null: %t}, ", key, c.Row[key].Text, c.Row[key].Null)
		}
		b.WriteString("}},\n")
	}
	b.WriteString("}\n")
	bound := fmt.Sprintf("package main\n\n// The attribute and function below come from the Trace of goemit.Emit.\nvar boundAttribute = %q\n\nvar boundCheck = %s\n",
		binding.Attribute, binding.Check)
	return map[string][]byte{
		"main.go":    []byte(harnessMain),
		"cases.go":   []byte(b.String()),
		"binding.go": []byte(bound),
	}
}

// SingleBinding returns the one binding of the one rule file in out.
func SingleBinding(t testing.TB, out goemit.Output) Binding {
	t.Helper()
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out.Trace) != 1 {
		t.Fatalf("want one emitted rule, got %d", len(out.Trace))
	}
	bindings, err := Bindings(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 {
		t.Fatalf("want one check function, got %+v", bindings)
	}
	return bindings[0]
}

// Program returns the generated files of out together with the harness for
// its single rule. out must be emitted for package main.
func Program(t testing.TB, out goemit.Output, cases []Case) map[string][]byte {
	t.Helper()
	files := Files(out)
	for name, content := range HarnessFiles(SingleBinding(t, out), cases) {
		files[name] = content
	}
	return files
}

// Run builds the program for out and cases in a standalone module, runs it
// and returns its output.
func Run(t testing.TB, out goemit.Output, cases []Case) string {
	t.Helper()
	dir := WriteModule(t, Program(t, out, cases))
	return string(RunGo(t, dir, "run", "."))
}
