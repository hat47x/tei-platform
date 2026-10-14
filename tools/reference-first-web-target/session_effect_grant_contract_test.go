//go:build !tei_repair_only

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

func TestMatchingBusinessGrantIsBoundToExactRequestedAuthority(t *testing.T) {
	grant := legacyauthorization.AuthorizationGrant{
		SubjectID: "subject-order-editor",
		Action:    "execute",
		Resource:  "order_entry.submit_order",
	}
	allowed := capabilityinvoke.BusinessResult{
		Capability: legacyauthorization.AuthorizationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "allowed",
		Payload:    grant,
	}
	if err := requireMatchingBusinessGrant(allowed, grant.SubjectID); err != nil {
		t.Fatalf("correct canonical grant was rejected: %v", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*capabilityinvoke.BusinessResult)
	}{
		{"wrong capability", func(r *capabilityinvoke.BusinessResult) { r.Capability = "other.authorize" }},
		{"wrong mode", func(r *capabilityinvoke.BusinessResult) { r.Mode = capabilityinvoke.ResultOutput }},
		{"denied instead of allowed", func(r *capabilityinvoke.BusinessResult) { r.Outcome = "denied" }},
		{"missing grant", func(r *capabilityinvoke.BusinessResult) { r.Payload = nil }},
		{"malformed payload", func(r *capabilityinvoke.BusinessResult) { r.Payload = map[string]any{"subjectId": grant.SubjectID} }},
		{"changed subject", func(r *capabilityinvoke.BusinessResult) {
			v := grant
			v.SubjectID = "subject-order-viewer"
			r.Payload = v
		}},
		{"changed action", func(r *capabilityinvoke.BusinessResult) { v := grant; v.Action = "read"; r.Payload = v }},
		{"changed resource", func(r *capabilityinvoke.BusinessResult) { v := grant; v.Resource = "other.command"; r.Payload = v }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := allowed
			tc.mutate(&candidate)
			if err := requireMatchingBusinessGrant(candidate, grant.SubjectID); err == nil {
				t.Fatalf("%s authority substitution was accepted", tc.name)
			}
		})
	}
}

func TestSessionEffectFenceRemainsInsideConcreteCommandAdapter(t *testing.T) {
	// A parse-tree regression catches accidental moving of the last check
	// back outside Driver.InvokeCapability. It does not prove transactions
	// or cross-process atomicity and is supplemental to HTTP integration tests.
	file, err := parser.ParseFile(token.NewFileSet(), "command_scenario.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var command *ast.FuncDecl
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if ok && fn.Name.Name == "runAuthenticatedSubjectCommandWithEffectGuard" {
			command = fn
			break
		}
	}
	if command == nil {
		t.Fatal("command runner was removed")
	}
	var adapter *ast.FuncLit
	ast.Inspect(command.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "capabilityAdapterFunc" || len(call.Args) != 1 {
			return true
		}
		adapter, _ = call.Args[0].(*ast.FuncLit)
		return false
	})
	if adapter == nil {
		t.Fatal("command effect does not use a concrete capability Adapter callback")
	}
	guardIndex, mutationIndex := -1, -1
	hasCanonicalRecheck := false
	for index, stmt := range adapter.Body.List {
		if gate, ok := stmt.(*ast.IfStmt); ok {
			if ident, ok := gate.Cond.(*ast.BinaryExpr); ok {
				if left, ok := ident.X.(*ast.Ident); ok && left.Name == "effectGuard" {
					guardIndex = index
					ast.Inspect(gate.Body, func(node ast.Node) bool {
						call, ok := node.(*ast.CallExpr)
						if !ok {
							return true
						}
						selector, ok := call.Fun.(*ast.SelectorExpr)
						if ok && selector.Sel.Name == "invokeBusinessAuthorization" {
							hasCanonicalRecheck = true
						}
						return true
					})
				}
			}
		}
		if mutation, ok := stmt.(*ast.AssignStmt); ok {
			for _, target := range mutation.Lhs {
				indexed, ok := target.(*ast.IndexExpr)
				if !ok {
					continue
				}
				if ident, ok := indexed.X.(*ast.Ident); ok && ident.Name == "state" {
					mutationIndex = index
				}
			}
		}
	}
	if guardIndex < 0 || mutationIndex < 0 || guardIndex >= mutationIndex || !hasCanonicalRecheck {
		t.Fatalf("effect check must precede state mutation inside Adapter; guard=%d mutation=%d reauthorize=%v",
			guardIndex, mutationIndex, hasCanonicalRecheck)
	}
}
