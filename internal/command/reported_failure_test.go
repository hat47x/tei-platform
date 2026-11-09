package command_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"tei/internal/command"
)

// executionCodeLiterals returns every string literal assigned to a
// `Code:` field in the package's non-test sources.
//
// Reading the source is deliberate. A code that is emitted but never
// classified cannot be caught by calling the function that emits it (it may
// need infrastructure the test does not have), and a hand-maintained list of
// codes in the test would drift the same way the production list did (#3807).
func executionCodeLiterals(t *testing.T) map[string]string {
	t.Helper()
	dirs, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	found := map[string]string{}
	for _, path := range dirs {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Code" {
				return true
			}
			lit, ok := kv.Value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if !strings.Contains(value, "COMMAND") && !strings.HasPrefix(value, "INVALID_COMMAND") &&
				!strings.HasPrefix(value, "MISSING_COMMAND") && !strings.HasPrefix(value, "UNDECLARED_COMMAND") {
				return true
			}
			found[value] = path
			return true
		})
	}
	return found
}

// Every code this package can put in an ExecutionError must reach an entrance
// unchanged. The entrances used to keep their own allow-lists, so 15 of 21 codes
// arrived as a generic COMMAND_FAILED and the two entrances disagreed (#3807).
// This walks the emitted codes instead of a hard-coded list, so adding a code
// without checking it here fails.
func TestEveryEmittedExecutionCodeIsReported(t *testing.T) {
	emitted := executionCodeLiterals(t)
	if len(emitted) < 10 {
		t.Fatalf("only %d codes found; the scanner probably stopped matching", len(emitted))
	}

	var unreported []string
	for code := range emitted {
		got, _, ok := command.ReportedFailure(&command.ExecutionError{Code: code, Message: "x"})
		if !ok {
			unreported = append(unreported, code+" ("+emitted[code]+")")
			continue
		}
		if got != code {
			unreported = append(unreported, code+" reported as "+got)
		}
	}
	if len(unreported) != 0 {
		sort.Strings(unreported)
		t.Fatalf("%d emitted code(s) do not reach the entrance:\n  %s",
			len(unreported), strings.Join(unreported, "\n  "))
	}
}

// The message must not repeat the code: a reader that shows both would print
// "COMMAND_UNAUTHORIZED: capability ... " twice.
func TestReportedFailureMessageHasNoCodePrefix(t *testing.T) {
	const code = "COMMAND_UNAUTHORIZED"
	got, message, ok := command.ReportedFailure(&command.ExecutionError{Code: code, Message: "capability x"})
	if !ok || got != code {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
	if strings.HasPrefix(message, code) {
		t.Fatalf("message repeats the code: %q", message)
	}
}

// A plain error has no machine-readable code, so the entrance keeps its own
// generic answer. This must not become "everything is reportable".
func TestReportedFailureLeavesUnknownErrorsAlone(t *testing.T) {
	if _, _, ok := command.ReportedFailure(errorString("boom")); ok {
		t.Fatal("a plain error must not be reported as a command code")
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }
