package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestAcceptedCommandsMatchDispatch reads the dispatch expressions in main.go
// and compares them with the list the usage line advertises. A command that is
// dispatched but omitted from the list would be rejected as unknown; a name
// in the list that is never dispatched would be advertised but unusable.
func TestAcceptedCommandsMatchDispatch(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	dispatched := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.BinaryExpr:
			if typed.Op == token.EQL {
				if ident, ok := typed.X.(*ast.Ident); ok && ident.Name == "command" {
					if literal, ok := typed.Y.(*ast.BasicLit); ok && literal.Kind == token.STRING {
						dispatched[strings.Trim(literal.Value, `"`)] = true
					}
				}
			}
		case *ast.CaseClause:
			for _, expr := range typed.List {
				if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					dispatched[strings.Trim(literal.Value, `"`)] = true
				}
			}
		}
		return true
	})
	advertised := map[string]bool{}
	for _, name := range acceptedCommands {
		if advertised[name] {
			t.Fatalf("duplicate accepted command %q", name)
		}
		advertised[name] = true
	}
	for _, name := range helperDispatchedCommands {
		dispatched[name] = true
	}
	for name := range dispatched {
		if !advertised[name] {
			t.Errorf("command %q is dispatched but not advertised", name)
		}
	}
	for name := range advertised {
		if !dispatched[name] {
			t.Errorf("command %q is advertised but never dispatched", name)
		}
	}
}

// TestUnknownCommandRejectionIgnoresFlags checks the actual argv boundary: the
// same unknown name must be rejected the same way with or without a flag, and
// the exit code stays 2.
func TestUnknownCommandRejectionIgnoresFlags(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("CLI execution: %v", err)
			}
			code = exit.ExitCode()
		}
		return code, stdout.String(), stderr.String()
	}
	for _, args := range [][]string{
		{"bogus"},
		{"bogus", "-workspace", "reference/workspace"},
		{"operation", "-workspace", "reference/workspace"},
	} {
		code, stdout, stderr := run(args...)
		if code != 2 {
			t.Fatalf("argv=%q exit=%d want=2", args, code)
		}
		if stdout != "" {
			t.Fatalf("argv=%q wrote stdout=%q", args, stdout)
		}
		if !strings.HasPrefix(stderr, `unknown command "`+args[0]+`"`) {
			t.Fatalf("argv=%q stderr=%q", args, stderr)
		}
	}
}
