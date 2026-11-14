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

func TestOpaqueNodeRetainsSourceButRefusesLossyRegeneration(t *testing.T) {
	// The Opaque IR node retains the original SQL for review, even when
	// other nodes could produce a syntactically valid but incomplete SELECT.
	// Refusing regeneration is required: the source cannot be reconstructed
	// faithfully by silently dropping custom_func().
	const originalSQL = "SELECT * FROM custom_func()"
	opaque := &Opaque{NodeID: "node:1", SQL: originalSQL, Reason: "unsupported function"}
	q := &StandardQuery{
		Nodes: []Node{
			&Scan{ID: "node:0", Source: "dual", SourceKind: "table"},
			opaque,
		},
		Root: "node:0",
	}
	if opaque.SQL != originalSQL {
		t.Fatalf("opaque native SQL changed: got %q, want %q", opaque.SQL, originalSQL)
	}
	sql, diags := StandardToSQL(q)
	if sql != "" {
		t.Fatalf("opaque node silently disappeared from regenerated SQL: %q", sql)
	}
	if len(diags) != 1 || diags[0].Code != "OPAQUE_QUERY_UNSUPPORTED" ||
		diags[0].Severity != SeverityUnsupported {
		t.Fatalf("expected fail-closed opaque diagnostic, got %#v", diags)
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
