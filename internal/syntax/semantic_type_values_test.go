package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticTypeValues(t *testing.T) {
	src := `namespace sales

type Status from std.code values { "open" "closed" }
`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Types) != 1 {
		t.Fatalf("types=%+v", ns.Types)
	}
	values := ns.Types[0].Values
	if len(values) != 2 || values[0] != "open" || values[1] != "closed" {
		t.Fatalf("values=%v", values)
	}
}
