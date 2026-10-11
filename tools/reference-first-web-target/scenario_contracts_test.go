package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// These transport-facing types must not become a hidden second business
// implementation or source of authority as E1 packages are separated.
func TestScenarioContractsHaveNoExecutableBusinessCode(t *testing.T) {
	tree, err := parser.ParseFile(token.NewFileSet(), "scenario_contracts.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range tree.Imports {
		name, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if name != "context" {
			t.Fatalf("scenario contracts imported implementation package %q", name)
		}
	}
	for _, declaration := range tree.Decls {
		if _, isImplementation := declaration.(*ast.FuncDecl); isImplementation {
			t.Fatal("scenario contract file acquired an executable implementation")
		}
	}
}

func TestScenarioResultContractsNeverKeepCredentials(t *testing.T) {
	for _, value := range []any{
		readScenarioResult{},
		commandScenarioResult{},
		customerLookupResult{},
	} {
		typeOf := reflect.TypeOf(value)
		for i := 0; i < typeOf.NumField(); i++ {
			field := typeOf.Field(i)
			for _, prohibited := range []string{"credential", "password", "secret"} {
				if strings.Contains(strings.ToLower(field.Name), prohibited) ||
					strings.Contains(strings.ToLower(field.Tag.Get("json")), prohibited) {
					t.Fatalf("%s retained a sensitive field: %s", typeOf, field.Name)
				}
			}
		}
	}
	raw, err := json.Marshal(commandScenarioResult{
		ScenarioID:    "order-entry.submit-order-web-slice",
		ApplicationID: "order_entry",
		Stage:         "authorization-denied",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "credential") ||
		strings.Contains(string(raw), "authentication\":null") ||
		strings.Contains(string(raw), "command\":null") ||
		strings.Contains(string(raw), "observation\":null") {
		t.Fatalf("optional security/command/observation contract drift: %s", raw)
	}
}
