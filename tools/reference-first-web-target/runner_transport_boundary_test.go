package main

import (
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// A source-level boundary for the combined Reference, prior to compiling
// distinct Application entry points: business Runner code may depend on
// Driver, but must not import net/http or html/template directly.
func TestBusinessRunnerSourcesDoNotImportHTTPTransport(t *testing.T) {
	for _, filename := range []string{
		"command_scenario.go",
		"customer_lookup_scenario.go",
		"read_scenario.go",
		"reference_contract_query_fixture.go",
		"shared_semantic_projection.go",
	} {
		t.Run(filename, func(t *testing.T) {
			node, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("invalid Go source: %v", err)
			}
			if node.Name.Name != "main" {
				t.Fatalf("unexpected package %q", node.Name.Name)
			}
			for _, spec := range node.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if path == "net/http" || path == "html/template" || path == "encoding/json" {
					t.Fatalf("business Runner source imports HTTP/serialization transport %q", path)
				}
			}
		})
	}
}
