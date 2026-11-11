package standard

import (
	"testing"
)

func TestOpaqueFallbackOnParseError(t *testing.T) {
	// Unsupported SQL should still produce a query with an Opaque node
	// containing the original SQL, plus a parse-error diagnostic.
	src := "SELECT FROM t"
	q, diags := ParseSQLToStandard(src, "")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for unsupported SQL")
	}
	if q == nil {
		t.Fatal("expected non-nil query even on parse error")
	}

	// Expect at least one Opaque node with the original SQL preserved.
	var foundOpaque bool
	for _, n := range q.Nodes {
		if o, ok := n.(*Opaque); ok {
			foundOpaque = true
			if o.SQL != src {
				t.Fatalf("opaque SQL mismatch: got %q, want %q", o.SQL, src)
			}
			if o.Reason == "" {
				t.Fatal("expected non-empty reason in opaque node")
			}
		}
	}
	if !foundOpaque {
		t.Fatalf("expected Opaque node in query, got %+v", q.Nodes)
	}
}
