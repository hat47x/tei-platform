package standard

import (
	"encoding/json"
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

func TestOpaqueNodeRoundTrip(t *testing.T) {
	// Opaque nodes are preserved in the IR but ignored by the flat SQL generator.
	// The generator falls back to the remaining known nodes.
	q := &StandardQuery{
		Nodes: []Node{
			&Scan{ID: "node:0", Source: "dual", SourceKind: "table"},
			&Opaque{NodeID: "node:1", SQL: "SELECT * FROM custom_func()", Reason: "unsupported function"},
		},
		Root: "node:0",
	}
	sql, diags := StandardToSQL(q)
	if sql != "SELECT * FROM dual" {
		t.Fatalf("unexpected generated SQL: %s", sql)
	}
	// Opaque nodes do not currently emit generation diagnostics in the flat path.
	_ = diags
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

// Severity is the closed vocabulary a diagnostic carries. A reader decides
// whether a finding blocks lowering from it, so an unsupported construct has to
// be distinguishable from an error rather than collapse into one value.
func TestSeverityVocabulary(t *testing.T) {
	severities := []Severity{SeverityError, SeverityWarning, SeverityUnsupported}
	want := []string{"error", "warning", "unsupported"}
	if len(severities) != len(want) {
		t.Fatalf("a severity was dropped: %d values, want %d", len(severities), len(want))
	}
	seen := map[string]bool{}
	for i, severity := range severities {
		if string(severity) != want[i] {
			t.Fatalf("severities[%d]=%q want %q", i, severity, want[i])
		}
		if seen[string(severity)] {
			t.Fatalf("severity %q is declared twice", severity)
		}
		seen[string(severity)] = true
	}

	// A diagnostic built with SeverityUnsupported keeps that value through the
	// JSON round trip, which is where a consumer reads it.
	d := Diagnostic{Code: "UNSUPPORTED_CONSTRUCT", Message: "unsupported", Severity: SeverityUnsupported}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Diagnostic
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Severity != SeverityUnsupported {
		t.Fatalf("round-trip severity = %q, want %q", decoded.Severity, SeverityUnsupported)
	}
}
