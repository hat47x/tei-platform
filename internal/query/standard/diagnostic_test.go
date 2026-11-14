package standard

import (
	"testing"
)

func TestDiagnosticSpan(t *testing.T) {
	// Valid SQL should produce no error diagnostics.
	q, diags := ParseSQLToStandard("SELECT 1 FROM dual", "")
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics for valid SQL, got %d: %+v", len(diags), diags)
	}
	if len(q.Diagnostics) != 0 {
		t.Fatalf("expected no query diagnostics for valid SQL, got %d", len(q.Diagnostics))
	}

	// Invalid SQL should produce a parse-error diagnostic with a non-empty span.
	q, diags = ParseSQLToStandard("SELECT FROM", "")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for invalid SQL")
	}
	found := false
	for _, d := range diags {
		if d.Code == "PARSE_ERROR" {
			found = true
			if d.Span.Start == 0 && d.Span.End == 0 {
				t.Fatalf("expected non-zero span in diagnostic, got %+v", d.Span)
			}
		}
	}
	if !found {
		t.Fatalf("expected PARSE_ERROR diagnostic, got %+v", diags)
	}
}

func TestOpaqueNodeFailsClosedAndPreservesNativeSQL(t *testing.T) {
	// An Opaque node keeps the original unsupported SQL for inspection but
	// must never be dropped to produce a different executable query.
	const nativeSQL = "SELECT * FROM custom_func()"
	opaque := &Opaque{NodeID: "node:1", SQL: nativeSQL, Reason: "unsupported function"}
	q := &StandardQuery{
		Nodes: []Node{
			&Scan{ID: "node:0", Source: "dual", SourceKind: "table"},
			opaque,
		},
		Root: "node:0",
	}
	sql, diags := StandardToSQL(q)
	if sql != "" {
		t.Fatalf("opaque query regenerated as executable SQL: %q", sql)
	}
	if len(diags) != 1 || diags[0].Code != "OPAQUE_QUERY_UNSUPPORTED" || diags[0].Severity != SeverityUnsupported {
		t.Fatalf("opaque query must fail closed with unsupported diagnostic: %+v", diags)
	}
	if opaque.SQL != nativeSQL {
		t.Fatalf("opaque native SQL changed: got %q, want %q", opaque.SQL, nativeSQL)
	}
}

// The two empty-input refusals are reachable, so they are named here: a code
// that stops being emitted, or one that is renamed, fails this case (#4032).
func TestEmptyInputRefusalsNameTheirCodes(t *testing.T) {
	if _, diags := ParseSQLToStandard("   \n\t ", ""); len(diags) == 0 {
		t.Fatal("blank SQL produced no refusal")
	} else if diags[0].Code != "EMPTY_SQL" {
		t.Fatalf("blank SQL code = %q, want EMPTY_SQL", diags[0].Code)
	}

	if _, diags := StandardToSQL(nil); len(diags) == 0 {
		t.Fatal("nil query produced no refusal")
	} else if diags[0].Code != "EMPTY_QUERY" {
		t.Fatalf("nil query code = %q, want EMPTY_QUERY", diags[0].Code)
	}
}
