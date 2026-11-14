//go:build server

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
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
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
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
		addr, ok := call.Args[0].(*ast.UnaryExpr)
		if !ok || addr.Op != token.AND {
			return true
		}
		value, ok := addr.X.(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ, ok := value.Type.(*ast.Ident)
		if ok && typ.Name == "StudioBridge" {
			registered++
		}
		return true
	})
	if registered != 1 {
		t.Fatalf("server build must explicitly review its registered Wails service: StudioBridge registrations=%d", registered)
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
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || !ast.IsExported(fn.Name.Name) {
			continue
		}
		pointer, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		receiver, ok := pointer.X.(*ast.Ident)
		if !ok || receiver.Name != "StudioBridge" {
			continue
		}
		name := fn.Name.Name
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
