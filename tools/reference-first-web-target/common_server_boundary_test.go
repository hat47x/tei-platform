package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// Common HTTP infrastructure must not statically reference Application-only
// routes or their HTML templates. Combined Gate C wiring is confined to the
// two explicit combined_* sources and can eventually be replaced at build time.
func TestCommonWebServerDoesNotWireSpecificApplications(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{
		"registerOrderEntryScenarioRoutes": true,
		"registerRepairIntakeScenarioRoutes": true,
		"customSubmitHTML": true,
		"customerLookupHTML": true,
		"newReferenceReadScenarioRunner": true,
		"newReferenceCommandScenarioRunner": true,
		"newReferenceCustomerLookupScenarioRunner": true,
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && forbidden[identifier.Name] {
			t.Errorf("common server.go couples to Application-specific symbol %s", identifier.Name)
		}
		return true
	})
}

// The default combined reference must still have a separate, explicit route
// and template provider. New build-tagged entries may replace those without
// duplicating the business runners or silently reauthoring semantics.
func TestCombinedRouteProvidersAreSeparateSourceFiles(t *testing.T) {
	for _, name := range []string{
		"combined_scenario_templates.go",
		"combined_scenario_routes.go",
	} {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if file.Name.Name != "main" {
			t.Fatalf("combined provider %s has package %s", name, file.Name.Name)
		}
		foundBuildTag := false
		for _, comment := range file.Comments {
			for _, line := range comment.List {
				if line.Text == "//go:build !tei_order_only && !tei_repair_only" {
					foundBuildTag = true
				}
			}
		}
		if !foundBuildTag {
			t.Fatalf("%s lost combined-mode-only build constraint", name)
		}
	}
}

// Even a non-routed link or static HTML snippet is part of the physical
// build's user-visible surface. Shared server sources must not ship a
// misleading Application-specific navigation menu.
func TestSharedIndexTemplateDoesNotAdvertiseAppRoutes(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Errorf("invalid string literal in shared server: %v", err)
			return true
		}
		for _, link := range []string{
			`href="/scenario/read"`,
			`href="/custom-submit"`,
			`href="/customer-lookup"`,
			"POST /scenario/submit",
		} {
			if strings.Contains(value, link) {
				t.Errorf("shared server must not embed Application-only navigation %q", link)
			}
		}
		return true
	})
}
