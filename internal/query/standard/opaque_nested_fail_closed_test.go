package standard

import (
	"strings"
	"testing"
)

// A nested StandardQuery may retain unsupported native SQL without providing
// an executable equivalent. Emitting an outer SELECT with that subtree omitted
// would falsely claim the query's meaning survived conversion.
func TestNestedOpaqueSQLNeverProducesExecutableSQL(t *testing.T) {
	opaque := func() *StandardQuery {
		return &StandardQuery{
			Nodes: []Node{&Opaque{
				NodeID: "opaque", SQL: "SELECT value OVER () FROM entries",
				Reason: "Window semantics are not supported by StandardQuery",
			}},
			Root: "opaque",
		}
	}
	project := func(e Expr) *StandardQuery {
		return &StandardQuery{
			Nodes: []Node{
				&Scan{ID: "scan", Source: "entries"},
				&Project{NodeID: "projection", Items: []ProjectItem{{Expr: e}}},
			},
			Root: "projection",
		}
	}
	cases := []struct {
		name  string
		query *StandardQuery
	}{
		{
			name: "FROM subquery",
			query: &StandardQuery{
				Nodes: []Node{
					&Scan{ID: "outer", SourceKind: "subquery", SubqueryID: "nested", Alias: "sub"},
					&Subquery{ID: "nested", Query: opaque()},
				},
				Root: "outer",
			},
		},
		{
			name:  "IN expression subquery",
			query: project(&InList{Expr: &Column{Name: "value"}, Subquery: opaque()}),
		},
		{
			name: "EXISTS filter",
			query: &StandardQuery{
				Nodes: []Node{
					&Scan{ID: "scan", Source: "entries"},
					&Filter{NodeID: "filter", Condition: &Exists{Query: opaque()}},
				},
				Root: "filter",
			},
		},
		{
			name:  "projection scalar subquery",
			query: project(&SubqueryExpr{Query: opaque()}),
		},
		{
			name: "CASE inside projection",
			query: project(&Case{Whens: []When{{
				Cond: &Exists{Query: opaque()},
				Then: &Literal{Kind: "number", Value: "1"},
			}}}),
		},
		{
			name: "deeply nested scalar expression",
			query: project(&FunctionCall{Name: "COALESCE", Args: []Expr{
				&Literal{Kind: "number", Value: "1"},
				&Cast{Expr: &SubqueryExpr{Query: opaque()}, Type: "INTEGER"},
			}}),
		},
		{
			name: "unsupported Values expression",
			query: &StandardQuery{
				Nodes: []Node{&Values{NodeID: "values", Rows: [][]Expr{{&Exists{Query: opaque()}}}}},
				Root:  "values",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := StandardToSQL(tc.query)
			if got != "" {
				t.Fatalf("unsupported nested SQL became executable: %q", got)
			}
			if len(diags) != 1 ||
				diags[0].Code != "OPAQUE_QUERY_UNSUPPORTED" ||
				diags[0].Severity != SeverityUnsupported {
				t.Fatalf("nested opaque SQL was not rejected: %#v", diags)
			}
		})
	}
}

func TestSupportedFromSubqueryStillGeneratesSQL(t *testing.T) {
	query := &StandardQuery{
		Nodes: []Node{
			&Scan{ID: "outer", SourceKind: "subquery", SubqueryID: "nested", Alias: "src"},
			&Subquery{
				ID: "nested",
				Query: &StandardQuery{
					Nodes: []Node{&Scan{ID: "inner", Source: "items"}},
					Root:  "inner",
				},
			},
		},
		Root: "outer",
	}
	got, diags := StandardToSQL(query)
	if len(diags) != 0 || !strings.Contains(got, "FROM (SELECT * FROM items) AS src") {
		t.Fatalf("supported nested query was rejected: sql=%q diagnostics=%#v", got, diags)
	}
}
