package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestStudioBridgeRegistersEveryClientMethod keeps the JS port and the Go
// service in step.
//
// assets/client.mjs addresses the service by name, and a name the service does
// not register fails at the binding — not at compile time. Seven projections
// were missing that way and every Workbench view action failed (#4295). The
// check reads both files, so adding a method to one side and not the other
// fails here instead of in the browser.
func TestStudioBridgeRegistersEveryClientMethod(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	{
		body, err := os.ReadFile(filepath.Join(root, "assets", "client.mjs"))
		if err != nil {
			t.Fatal(err)
		}
		pattern := regexp.MustCompile(`"main\.StudioBridge\.([A-Za-z0-9_]+)"`)
		for _, match := range pattern.FindAllStringSubmatch(string(body), -1) {
			names[match[1]] = true
		}
	}
	if len(names) == 0 {
		t.Fatal("client.mjs names no StudioBridge method")
	}

	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "main.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		star, isStar := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !isStar {
			continue
		}
		ident, isIdent := star.X.(*ast.Ident)
		if !isIdent || ident.Name != "StudioBridge" {
			continue
		}
		registered[fn.Name.Name] = true
	}
	if len(registered) == 0 {
		t.Fatal("main.go declares no StudioBridge method")
	}

	for name := range names {
		if !registered[name] {
			t.Errorf("client.mjs calls main.StudioBridge.%s, which the service does not register", name)
		}
	}
	// A registered method the bootstrap does not call is NOT the failure this
	// check exists for: that failure is a JS name with no Go method behind it.
	// The SHO relay is a Go-facing API (tests, and a future document view) with
	// no Workbench entry yet, so it is named here instead of adding a dead UI
	// binding. Adding a name to this list without saying why is not allowed.
	goFacingOnly := map[string]string{
		"OpenShoDocument":   "relays a draft into the real SHO document CLI; no Workbench entry yet",
		"ResumeShoDocument": "reads a saved document back through the real SHO document CLI; no Workbench entry yet",
	}
	for name := range registered {
		if names[name] {
			continue
		}
		if _, exempt := goFacingOnly[name]; exempt {
			continue
		}
		t.Errorf("main.StudioBridge.%s is registered but assets/client.mjs never names it", name)
	}
}
