package standard

import (
	"strings"
	"testing"
)

// Window semantics are implemented in the SQLite dialect adapter but are not
// represented by StandardQuery yet. Never report a lossy conversion as valid.
func TestCTKStandardQueryPreservesUnsupportedWindowSQL(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{
			name: "named window after source",
			sql:  "SELECT value FROM observations WINDOW moving AS (PARTITION BY grp ORDER BY seq ROWS BETWEEN 2 PRECEDING AND CURRENT ROW)",
		},
		{
			name: "inline analytical expression",
			sql:  "SELECT AVG(value) OVER (PARTITION BY grp ORDER BY seq ROWS BETWEEN 1 PRECEDING AND CURRENT ROW) AS moving FROM observations",
		},
		{
			name: "locking clause",
			sql:  "SELECT value FROM observations FOR UPDATE",
		},
		{
			name: "extra statement",
			sql:  "SELECT value FROM observations; DELETE FROM observations",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diagnostics := ParseSQLToStandard(tc.sql, "sqlite")
			if got == nil || len(diagnostics) == 0 {
				t.Fatalf("unsupported SQL was accepted without a diagnostic: %#v, %#v", got, diagnostics)
			}
			if len(got.Nodes) != 1 {
				t.Fatalf("unsupported SQL nodes = %#v", got.Nodes)
			}
			opaque, ok := got.Nodes[0].(*Opaque)
			if !ok || opaque.SQL != tc.sql {
				t.Fatalf("lost unsupported original SQL: %#v", got.Nodes)
			}
			foundError := false
			for _, d := range diagnostics {
				if d.Code == "PARSE_ERROR" && d.Severity == SeverityError {
					foundError = true
				}
			}
			if !foundError {
				t.Fatalf("no rejecting diagnostic: %#v", diagnostics)
			}
			out, findings := StandardToSQL(got)
			if out != "" || len(findings) == 0 ||
				findings[0].Code != "OPAQUE_QUERY_UNSUPPORTED" ||
				findings[0].Severity != SeverityUnsupported {
				t.Fatalf("opaque SQL regenerated instead of rejected: sql=%q findings=%#v", out, findings)
			}
		})
	}
}

func TestCTKStandardQueryAcceptsSingleStatementTerminator(t *testing.T) {
	const sql = "SELECT value FROM observations;"
	q, findings := ParseSQLToStandard(sql, "sqlite")
	if len(findings) != 0 {
		t.Fatalf("single final semicolon rejected: %#v", findings)
	}
	out, diagnostics := StandardToSQL(q)
	if len(diagnostics) != 0 || strings.TrimSpace(out) != "SELECT value FROM observations" {
		t.Fatalf("ordinary SELECT was not represented: sql=%q findings=%#v", out, diagnostics)
	}
}

func TestCTKStandardQueryRefusesOpaqueNestedCTE(t *testing.T) {
	query := &StandardQuery{
		CTEs: []CTE{{ID: "src", Query: &StandardQuery{
			Nodes: []Node{&Opaque{NodeID: "opaque", SQL: "SELECT value OVER () FROM t", Reason: "unsupported Window"}},
			Root:  "opaque",
		}}},
		Nodes: []Node{
			&Scan{ID: "scan", Source: "src"},
			&Project{NodeID: "project", Items: []ProjectItem{{Expr: &Column{Name: "value"}}}},
		},
		Root: "project",
	}
	out, diagnostics := StandardToSQL(query)
	if out != "" || len(diagnostics) == 0 || diagnostics[0].Code != "OPAQUE_QUERY_UNSUPPORTED" {
		t.Fatalf("unsupported nested CTE emitted SQL: %q, findings=%#v", out, diagnostics)
	}
}
