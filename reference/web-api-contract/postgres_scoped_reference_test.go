package webapicontract

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/query"
	"tei/internal/workspace"
)

type scopedSQLState struct {
	statement string
	args      []sqldriver.NamedValue
	calls     int
	columns   []string
	rows      [][]sqldriver.Value
}

type scopedSQLConnector struct{ state *scopedSQLState }

func (c scopedSQLConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &scopedSQLConn{state: c.state}, nil
}

func (c scopedSQLConnector) Driver() sqldriver.Driver { return scopedSQLDriver{} }

type scopedSQLDriver struct{}

func (scopedSQLDriver) Open(string) (sqldriver.Conn, error) {
	return nil, errors.New("use the test connector")
}

type scopedSQLConn struct{ state *scopedSQLState }

func (*scopedSQLConn) Prepare(string) (sqldriver.Stmt, error) { return nil, sqldriver.ErrSkip }
func (*scopedSQLConn) Close() error                           { return nil }
func (*scopedSQLConn) Begin() (sqldriver.Tx, error)           { return nil, sqldriver.ErrSkip }

func (c *scopedSQLConn) QueryContext(
	_ context.Context,
	statement string,
	args []sqldriver.NamedValue,
) (sqldriver.Rows, error) {
	c.state.calls++
	c.state.statement = statement
	c.state.args = append([]sqldriver.NamedValue(nil), args...)
	return &scopedSQLRows{columns: c.state.columns, rows: c.state.rows}, nil
}

var _ sqldriver.QueryerContext = (*scopedSQLConn)(nil)

type scopedSQLRows struct {
	columns []string
	rows    [][]sqldriver.Value
	index   int
}

func (r *scopedSQLRows) Columns() []string { return r.columns }
func (r *scopedSQLRows) Close() error      { return nil }
func (r *scopedSQLRows) Next(dest []sqldriver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func scopedTestDB(t *testing.T, state *scopedSQLState) *sql.DB {
	t.Helper()
	db := sql.OpenDB(scopedSQLConnector{state: state})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func scopedCanonicalWorld(t *testing.T) driver.Resolver {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return driver.NewResolver(loaded.World)
}

func scopedTestAuthority() PostgresCustomerScope {
	return PostgresCustomerScope{
		PrincipalID:    "subject-order-viewer",
		RowScopeID:     "trusted-customer-scope",
		PolicyRevision: "policy-1",
		CustomerID:     11,
	}
}

func scopedTestRead(t *testing.T, params url.Values) (PublishedBrowse, ReadPlan, ProviderReadSupport) {
	t.Helper()
	pub := scopedPublication(t)
	auth := scopedTestAuthority()
	plan, err := AuthorizeBrowseRead(context.Background(), fixtureAuthorization(t),
		pub, params, TrustedReadContext{
			PrincipalID: auth.PrincipalID, RowScopeID: auth.RowScopeID,
			PolicyRevision: auth.PolicyRevision,
		})
	if err != nil {
		t.Fatal(err)
	}
	return pub, plan, scopedSupport(pub, plan)
}

func TestExplicitScopedPostgresSQLBindsCustomerBeforeClientFilterAndTop(t *testing.T) {
	pub, plan, support := scopedTestRead(t, url.Values{
		"$select":  {"orderId"},
		"$filter":  {"orderId gt 2"},
		"$orderby": {"orderId desc"},
		"$top":     {"1"},
	})
	state := &scopedSQLState{
		columns: []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"},
		rows:    [][]sqldriver.Value{{int64(9), nil, nil, nil, nil}},
	}
	exec := ScopedRecentOrdersPostgres{
		DB: scopedTestDB(t, state), Database: core.MustCanonicalID("sales_core"),
		Scope: scopedTestAuthority(),
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ExecuteScopedCanonicalRead(context.Background(), scopedCanonicalWorld(t),
		pub, plan, support, exec, map[string]any{"from": from})
	if err != nil {
		t.Fatal(err)
	}
	expectedSQL := "SELECT o.order_id, NULL AS customer_id, NULL AS ordered_at, NULL AS status, NULL AS total_amount" +
		" FROM public.orders AS o WHERE o.customer_id = $1 AND o.ordered_at >= $2" +
		" AND o.order_id > $3 ORDER BY o.order_id DESC LIMIT $4"
	if state.calls != 1 || state.statement != expectedSQL {
		t.Fatalf("unexpected scoped SQL: calls=%d sql=%q", state.calls, state.statement)
	}
	if len(state.args) != 4 ||
		state.args[0].Value != int64(11) ||
		state.args[1].Value != from ||
		state.args[2].Value != int64(2) ||
		state.args[3].Value != int64(1) {
		t.Fatalf("unsafe or out-of-order parameter binding: %#v", state.args)
	}
	if !reflect.DeepEqual(result.Columns, []string{"orderId"}) ||
		!reflect.DeepEqual(result.Rows, [][]any{{int64(9)}}) {
		t.Fatalf("SQL raw/sensitive data escaped canonical projection: %+v", result)
	}
}

func TestExplicitScopedPostgresRejectsBeforeQueryContext(t *testing.T) {
	pub, plan, support := scopedTestRead(t, url.Values{"$select": {"orderId"}, "$top": {"2"}})
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		mutate func(*PostgresCustomerScope, *ReadPlan, *map[string]any)
	}{
		{"scope missing", func(scope *PostgresCustomerScope, _ *ReadPlan, _ *map[string]any) {
			scope.CustomerID = 0
		}},
		{"principal mismatch", func(scope *PostgresCustomerScope, _ *ReadPlan, _ *map[string]any) {
			scope.PrincipalID = "other"
		}},
		{"row scope mismatch", func(scope *PostgresCustomerScope, _ *ReadPlan, _ *map[string]any) {
			scope.RowScopeID = "other"
		}},
		{"stale policy", func(scope *PostgresCustomerScope, _ *ReadPlan, _ *map[string]any) {
			scope.PolicyRevision = "stale"
		}},
		{"invalid filter field", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Filter = &Filter{FieldID: "total", Operator: "eq", Value: "9000"}
		}},
		{"invalid filter operator", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Filter = &Filter{FieldID: "orderId", Operator: "ne", Value: "9"}
		}},
		{"SQL injection literal", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Filter = &Filter{FieldID: "orderId", Operator: "eq", Value: "1 OR 1=1"}
		}},
		{"duplicate projection", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Selected = []string{"orderId", "orderId"}
		}},
		{"unknown projection", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Selected = []string{"secretInternal"}
		}},
		{"hidden sort field", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Order = &Ordering{FieldID: "total"}
		}},
		{"oversize limit", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Limit = 101
		}},
		{"negative filter ID", func(_ *PostgresCustomerScope, p *ReadPlan, _ *map[string]any) {
			p.Filter = &Filter{FieldID: "orderId", Operator: "eq", Value: "-1"}
		}},
		{"missing from parameter", func(_ *PostgresCustomerScope, _ *ReadPlan, args *map[string]any) {
			*args = nil
		}},
		{"wrong from type", func(_ *PostgresCustomerScope, _ *ReadPlan, args *map[string]any) {
			*args = map[string]any{"from": "yesterday"}
		}},
		{"unknown parameter", func(_ *PostgresCustomerScope, _ *ReadPlan, args *map[string]any) {
			(*args)["tenant_id"] = "attacker"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &scopedSQLState{}
			scope := scopedTestAuthority()
			p := plan
			params := map[string]any{"from": from}
			tc.mutate(&scope, &p, &params)
			exec := ScopedRecentOrdersPostgres{
				DB: scopedTestDB(t, state), Database: core.MustCanonicalID("sales_core"),
				Scope: scope,
			}
			_, err := exec.ExecuteScopedBrowseQuery(context.Background(), mustScopedQuery(t),
				mustScopedDatasource(t), p, params)
			if err == nil || state.calls != 0 {
				t.Fatalf("unsafe request reached DB: err=%v calls=%d", err, state.calls)
			}
		})
	}
	_ = support
	_ = pub
}

func mustScopedQuery(t *testing.T) query.EmbeddedSQL {
	t.Helper()
	resolved, err := scopedCanonicalWorld(t).ResolvePattern(
		core.MustCanonicalID("order_entry"), core.MustCanonicalID("order_entry.recent_orders"))
	if err != nil || resolved.Query == nil {
		t.Fatalf("query resolution failed: %v", err)
	}
	return *resolved.Query
}

func mustScopedDatasource(t *testing.T) application.DatabaseRequirement {
	t.Helper()
	resolved, err := scopedCanonicalWorld(t).ResolvePattern(
		core.MustCanonicalID("order_entry"), core.MustCanonicalID("order_entry.recent_orders"))
	if err != nil || resolved.Datasource == nil {
		t.Fatalf("datasource resolution failed: %v", err)
	}
	return *resolved.Datasource
}

func TestScopedPostgresRejectsModifiedNativeSQL(t *testing.T) {
	state := &scopedSQLState{}
	queryDef := mustScopedQuery(t)
	queryDef.Source += " /* changed */"
	scope := scopedTestAuthority()
	_, plan, _ := scopedTestRead(t, url.Values{"$select": {"orderId"}})
	exec := ScopedRecentOrdersPostgres{
		DB: scopedTestDB(t, state), Database: core.MustCanonicalID("sales_core"),
		Scope: scope,
	}
	_, err := exec.ExecuteScopedBrowseQuery(context.Background(), queryDef,
		mustScopedDatasource(t), plan, map[string]any{"from": time.Now()})
	if !errors.Is(err, ErrScopedQueryContract) || state.calls != 0 {
		t.Fatalf("changed canonical SQL was silently reinterpreted: %v", err)
	}
	if strings.Contains(state.statement, "changed") {
		t.Fatal("changed source was issued")
	}
}

func TestScopedPostgresPinsCanonicalColumnMappingBeforeSQL(t *testing.T) {
	state := &scopedSQLState{}
	q := mustScopedQuery(t)
	q.ResultColumns = append([]query.ResultColumnMapping(nil), q.ResultColumns...)
	q.ResultColumns[0].Attribute = core.MustCanonicalID("sales.order.total")
	_, plan, _ := scopedTestRead(t, url.Values{"$select": {"orderId"}})
	exec := ScopedRecentOrdersPostgres{
		DB: scopedTestDB(t, state), Database: core.MustCanonicalID("sales_core"),
		Scope: scopedTestAuthority(),
	}
	_, err := exec.ExecuteScopedBrowseQuery(context.Background(), q,
		mustScopedDatasource(t), plan, map[string]any{"from": time.Now()})
	if !errors.Is(err, ErrScopedQueryContract) || state.calls != 0 {
		t.Fatalf("altered Semantic mapping reached SQL: err=%v calls=%d", err, state.calls)
	}
}

func TestScopedPostgresDefaultOrderingHasStableTieBreaker(t *testing.T) {
	state := &scopedSQLState{
		columns: []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"},
	}
	_, plan, _ := scopedTestRead(t, url.Values{"$select": {"orderId"}, "$top": {"2"}})
	exec := ScopedRecentOrdersPostgres{
		DB: scopedTestDB(t, state), Database: core.MustCanonicalID("sales_core"),
		Scope: scopedTestAuthority(),
	}
	_, err := exec.ExecuteScopedBrowseQuery(context.Background(), mustScopedQuery(t),
		mustScopedDatasource(t), plan, map[string]any{
			"from": time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		})
	if err != nil {
		t.Fatal(err)
	}
	if state.calls != 1 ||
		!strings.Contains(state.statement, "ORDER BY o.ordered_at DESC, o.order_id DESC LIMIT $3") {
		t.Fatalf("default Top ordering is not deterministic: %q", state.statement)
	}
}
