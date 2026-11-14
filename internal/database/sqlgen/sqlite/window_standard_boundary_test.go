package sqlite

import (
	"context"
	"fmt"
	"testing"

	sqlgen "tei/internal/database/sqlgen"
	"tei/internal/query/standard"
)

// SQLite can generate and execute a Window query while the current
// StandardQuery model cannot preserve Window semantics. This test protects
// the boundary: a lossy SQL -> Standard -> SQL result is never acceptable.
func TestCTKGeneratedWindowSQLRefusedByUnreadyStandardQuery(t *testing.T) {
	for _, preceding := range []int{2, 1} {
		t.Run(fmt.Sprintf("rows-%d", preceding), func(t *testing.T) {
			opts := sqlgen.SelectOptions{
				Table:   "observations",
				Columns: []string{"obs_id", "AVG(value) OVER moving AS moving"},
				Window:  fmt.Sprintf("moving AS (PARTITION BY grp ORDER BY seq, obs_id ROWS BETWEEN %d PRECEDING AND CURRENT ROW)", preceding),
			}
			sql, err := NewDialect().Select(context.Background(), opts)
			if err != nil {
				t.Fatalf("SQLite generation: %v", err)
			}
			q, diagnostics := standard.ParseSQLToStandard(sql.Statement, "sqlite")
			if q == nil || len(diagnostics) == 0 || len(q.Nodes) != 1 {
				t.Fatalf("unsupported Window was silently accepted: %+v diagnostics=%+v", q, diagnostics)
			}
			opaque, ok := q.Nodes[0].(*standard.Opaque)
			if !ok || opaque.SQL != sql.Statement {
				t.Fatalf("generated Window source lost: %+v", q.Nodes)
			}
			regenerated, findings := standard.StandardToSQL(q)
			if regenerated != "" || len(findings) == 0 ||
				findings[0].Code != "OPAQUE_QUERY_UNSUPPORTED" ||
				findings[0].Severity != standard.SeverityUnsupported {
				t.Fatalf("lossy Window SQL re-emitted: %q diagnostics=%+v", regenerated, findings)
			}
		})
	}
}
