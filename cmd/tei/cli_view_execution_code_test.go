package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// viewEnvelopeCode runs one view invocation and returns the exit code and the
// error.code a machine-readable caller would branch on.
func viewEnvelopeCode(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out bytes.Buffer
	code, err := runViewCommand(args, &out)
	if err != nil {
		t.Fatalf("runViewCommand returned an execution error: %v", err)
	}
	var report struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, out.String())
	}
	if report.Error == nil {
		return code, "", ""
	}
	return code, report.Error.Code, report.Error.Message
}

// A rejected command must name its cause in error.code. These two used to
// collapse into COMMAND_FAILED, so the only machine-readable identifier was the
// prose prefix of error.message: a caller had to match strings to tell "fix the
// input" from "fix the authority" (#3797).
func TestViewExecuteNamesTheCommandFailureCause(t *testing.T) {
	// The existing product-slice fixture declares the command pattern and its
	// capability, so the two failures below reach the Invoker instead of
	// stopping at pattern resolution.
	root := productSliceWorkspace(t)
	common := []string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-execute",
		"-format", "json",
	}

	t.Run("missing required input", func(t *testing.T) {
		code, failure, message := viewEnvelopeCode(t, common...)
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (a refusal, not an input error)", code)
		}
		if failure != "INVALID_COMMAND_INPUT" {
			t.Fatalf("error.code=%q, want INVALID_COMMAND_INPUT (message=%q)", failure, message)
		}
		if strings.HasPrefix(message, "INVALID_COMMAND_INPUT:") {
			t.Fatalf("the code is repeated in the message now that it has its own field: %q", message)
		}
	})

	t.Run("capability not allowed", func(t *testing.T) {
		args := append(append([]string{}, common...),
			"-input", "order=ORD-1", "-allow-capability", "sales.other")
		code, failure, message := viewEnvelopeCode(t, args...)
		if code != 1 {
			t.Fatalf("exit=%d, want 1", code)
		}
		if failure != "COMMAND_UNAUTHORIZED" {
			t.Fatalf("error.code=%q, want COMMAND_UNAUTHORIZED (message=%q)", failure, message)
		}
		if strings.HasPrefix(message, "COMMAND_UNAUTHORIZED:") {
			t.Fatalf("the code is repeated in the message now that it has its own field: %q", message)
		}
	})
}

// The mapping list is only a guard if it keeps up with internal/command. When a
// new ExecutionError code is added there and not here, the new code silently
// falls back to COMMAND_FAILED again — exactly the defect #3797 reported. This
// reads the literal codes out of internal/command/invoke.go so the two cannot
// drift apart.
func TestEveryExecutionCodeHasAName(t *testing.T) {
	// The codes live across internal/command (invoke.go for the execution
	// boundary, lifecycle.go for replay decisions), so scan the whole package
	// rather than one file.
	packageDir := filepath.Join(testRepositoryRoot(t), "internal", "command")
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		t.Fatalf("read %s: %v", packageDir, err)
	}
	found := map[string]bool{}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source := filepath.Join(packageDir, name)
		file, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", source, err)
		}
		scanned++
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil || !strings.HasPrefix(value, "COMMAND_") && value != "INVALID_COMMAND_INPUT" {
				return true
			}
			found[value] = true
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("no non-test files were scanned in internal/command")
	}
	if len(found) == 0 {
		t.Fatal("no ExecutionError codes were found; the scan is not looking at the right file")
	}
	// Every code internal/command produces must have an explicit decision: it
	// either keeps its own error.code or is deliberately reported as
	// COMMAND_FAILED. A code in neither list is the defect #3797 reported — it
	// collapses into the generic code without anyone choosing that.
	decided := map[string]bool{}
	for _, code := range commandExecutionCodes {
		decided[code] = true
	}
	for _, code := range commandGenericCodes {
		if decided[code] {
			t.Errorf("%q is listed as both named and generic; pick one", code)
		}
		decided[code] = true
	}
	for code := range found {
		if !decided[code] {
			t.Errorf("internal/command produces %q but neither commandExecutionCodes nor commandGenericCodes in "+
				"cmd/tei/cli_view.go lists it, so it silently collapses into COMMAND_FAILED (#3797)", code)
		}
	}
	// The reverse direction matters too: a name listed for a code the package
	// no longer produces is stale guidance.
	for code := range decided {
		if !found[code] {
			t.Errorf("%q is listed in cmd/tei/cli_view.go but internal/command no longer produces it", code)
		}
	}
}
