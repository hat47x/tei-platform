//go:build server

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// This is an inventory guard for the *existing loopback-only* Wails Server
// reference, not authorization for public exposure. Wails server mode binds
// exported Go service methods; the JavaScript client's method list is not an
// access-control list. Every added exported method requires a deliberate
// classification before a shared Web host can be considered.
func TestServerBridgeExportedMethodInventoryRequiresReview(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	registered := 0
	unreviewed := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewService" {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "application" {
			return true
		}
		if len(call.Args) == 1 {
			if addr, ok := call.Args[0].(*ast.UnaryExpr); ok && addr.Op == token.AND {
				if value, ok := addr.X.(*ast.CompositeLit); ok {
					if typ, ok := value.Type.(*ast.Ident); ok && typ.Name == "StudioBridge" {
						registered++
						return true
					}
				}
			}
		}
		unreviewed++
		return true
	})
	if registered != 1 || unreviewed != 0 {
		t.Fatalf("Wails Server service registration changed; review all exported surfaces: StudioBridge=%d unreviewed=%d", registered, unreviewed)
	}

	// The labels are audit classifications, NOT grants. In particular,
	// "reference command" is still an exported method even when its current
	// implementation only writes to an in-memory reference store.
	expected := map[string]string{
		"ListWorkspaces":                   "read projection",
		"CatalogWorkspace":                 "read projection",
		"ValidateWorkspace":                "read projection",
		"InspectSubject":                   "read projection",
		"ObserveXead":                      "bounded source read",
		"ViewPattern":                      "read projection",
		"ViewApplication":                  "read projection",
		"ViewPackage":                      "read projection",
		"PlanCommand":                      "command plan",
		"ExecuteCommand":                   "reference command",
		"ExecuteCommandWithPeriods":        "reference command",
		"ExecuteCommandWithPolicy":         "reference command",
		"ExecuteCommandWithReferenceStore": "reference command with memory effect",
		"OpenShoDocument":                  "external child process / file path",
		"ResumeShoDocument":                "external child process / file path",
	}
	found := map[string]bool{}
	// Reflection inspects the effective Go method set, including methods
	// promoted from embedded types and methods declared in another file.
	// It is the candidate set, not proof of every Wails HTTP route.
	typ := reflect.TypeOf(&StudioBridge{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		found[name] = true
		if _, classified := expected[name]; !classified {
			t.Errorf("new Wails Server exported method %q has no exposure classification; review before adding it to the shared-host surface", name)
		}
	}
	if len(found) == 0 {
		t.Fatal("no StudioBridge exported methods found: inventory check would pass vacuously")
	}
	for name, classification := range expected {
		if !found[name] {
			t.Errorf("reviewed StudioBridge method %q (%s) disappeared; update the inventory deliberately", name, classification)
		}
	}
	var elevated []string
	for name := range found {
		if classification := expected[name]; classification != "" && classification != "read projection" && classification != "bounded source read" {
			elevated = append(elevated, name+"="+classification)
		}
	}
	sort.Strings(elevated)
	if len(elevated) == 0 {
		t.Fatal("expected privileged/non-read methods are absent; re-evaluate this inventory test when the server bridge is split")
	}
	t.Logf("NOT PUBLIC-QUALIFIED: loopback Wails Server currently registers privileged/non-read surfaces: %s", strings.Join(elevated, ", "))
}
