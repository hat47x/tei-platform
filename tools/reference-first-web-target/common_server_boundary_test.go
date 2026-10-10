package main

import (
	"go/ast"
	"go/parser"
	"go/token"
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
