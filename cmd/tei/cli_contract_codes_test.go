package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// commandExecutionCodes returns every ExecutionError code internal/command can
// emit, read from its sources. Reading the source is deliberate: the codes are
// produced on paths that need infrastructure a unit test does not have, and a
// list kept by hand in the test would drift the same way the document did.
func commandExecutionCodes(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "internal", "command", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, path := range paths {
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
			seen[value] = true
			return true
		})
	}
	if len(seen) == 0 {
		t.Fatal("no ExecutionError codes found; the scan stopped matching")
	}
	codes := make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// contractDocRow returns the failure cell of one command's row in the CLI JSON
// output contract.
func contractDocRow(t *testing.T, command string) string {
	t.Helper()
	path := filepath.Join("..", "..", "plan", "design", "CLI_JSON_OUTPUT_CONTRACT_2026-10.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the CLI JSON contract: %v", err)
	}
	body := string(raw)
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "| `"+command+"` |") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 5 {
			t.Fatalf("%s row has %d cells", command, len(cells))
		}
		return cells[4]
	}
	t.Fatalf("no row for %q in the CLI JSON output contract", command)
	return ""
}

// The `view` row must name every code a reference execution can report. Before
// #3419 it named four, while `-execute` passed through whatever
// internal/command produced, so a reader working from the document would not
// know that COMMAND_UNAUTHORIZED or COMMAND_DATASOURCE_READ_ONLY was possible
// (#3807 made that wider, not narrower).
//
// The document is the prose a person reads; this test keeps it complete without
// asking anyone to remember the list twice.
func TestContractDocumentNamesEveryCommandExecutionCode(t *testing.T) {
	codes := commandExecutionCodes(t)
	if len(codes) < 10 {
		t.Fatalf("only %d codes found; the scan probably stopped matching", len(codes))
	}
	row := contractDocRow(t, "view")
	documented := map[string]bool{}
	for _, match := range regexp.MustCompile("`([A-Z][A-Z0-9_]{3,})`").FindAllStringSubmatch(row, -1) {
		documented[match[1]] = true
	}
	var missing []string
	for _, code := range codes {
		if !documented[code] {
			missing = append(missing, code)
		}
	}
	if len(missing) != 0 {
		t.Fatalf("the view row does not name %d code(s) a reference execution can report:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// The document must not claim the effect-unknown override the wrong way round:
// a failure whose effect may be applied is reported as COMMAND_EFFECT_UNKNOWN
// even when the underlying execution error has its own code.
func TestContractDocumentNamesTheEffectUnknownOverride(t *testing.T) {
	row := contractDocRow(t, "view")
	if !strings.Contains(row, "COMMAND_EFFECT_UNKNOWN") {
		t.Fatalf("the view row does not mention COMMAND_EFFECT_UNKNOWN: %s", row)
	}
	if !strings.Contains(row, "優先") {
		t.Fatalf("the view row does not say COMMAND_EFFECT_UNKNOWN takes precedence: %s", row)
	}
}
