package webapicontract

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/environment"
	"tei/internal/query"
)

type postgresScopedTrace struct {
	calls   int
	sql     string
	args    []sqldriver.NamedValue
	rows    [][]sqldriver.Value
	execErr error
}

type postgresScopedConnector struct{ trace *postgresScopedTrace }

func (c postgresScopedConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &postgresScopedConn{trace: c.trace}, nil
}

func (c postgresScopedConnector) Driver() sqldriver.Driver {
	return postgresScopedDriver{}
}

type postgresScopedDriver struct{}

func (postgresScopedDriver) Open(string) (sqldriver.Conn, error) {
	return nil, errors.New("use explicit test connector")
}

type postgresScopedConn struct{ trace *postgresScopedTrace }

func (*postgresScopedConn) Prepare(string) (sqldriver.Stmt, error) {
	return nil, sqldriver.ErrSkip
}

func (*postgresScopedConn) Close() error { return nil }

func (*postgresScopedConn) Begin() (sqldriver.Tx, error) {
	return nil, sqldriver.ErrSkip
}

func (c *postgresScopedConn) QueryContext(
	_ context.Context,
	sqlText string,
	args []sqldriver.NamedValue,
) (sqldriver.Rows, error) {
	c.trace.calls++
	c.trace.sql = sqlText
	c.trace.args = append([]sqldriver.NamedValue(nil), args...)
	if c.trace.execErr != nil {
		return nil, c.trace.execErr
	}
	rows := make([][]sqldriver.Value, len(c.trace.rows))
	for i := range c.trace.rows {
		rows[i] = append([]sqldriver.Value(nil), c.trace.rows[i]...)
	}
	return &postgresScopedRows{rows: rows}, nil
}

var _ sqldriver.QueryerContext = (*postgresScopedConn)(nil)

type postgresScopedRows struct {
	rows  [][]sqldriver.Value
	index int
}

func (*postgresScopedRows) Columns() []string {
	return []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"}
}

func (*postgresScopedRows) Close() error { return nil }

func (r *postgresScopedRows) Next(dest []sqldriver.Value) error {
	if r.index == len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func postgresScopedDB(t *testing.T, trace *postgresScopedTrace) *sql.DB {
	t.Helper()
	db := sql.OpenDB(postgresScopedConnector{trace: trace})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type postgresScopedAuthority struct {
	calls    int
	received SQLTenantScopeRequest
	tenant   string
	forged   *SQLTenantScopeGrant
	err      error
}

func (a *postgresScopedAuthority) ResolveSQLTenantScope(
	_ context.Context,
	request SQLTenantScopeRequest,
) (SQLTenantScopeGrant, error) {
	a.calls++
	a.received = request
	if a.err != nil {
		return SQLTenantScopeGrant{}, a.err
	}
	if a.forged != nil {
		return *a.forged, nil
	}
	return SQLTenantScopeGrant{SQLTenantScopeRequest: request, TenantID: a.tenant}, nil
}

func postgresCanonicalInput(t *testing.T) (query.EmbeddedSQL, application.DatabaseRequirement) {
	t.Helper()
	pattern, err := bridgeWorld(t).ResolvePattern("order_entry", "order_entry.recent_orders")
	if err != nil || pattern.Query == nil || pattern.Datasource == nil {
		t.Fatalf("unable to resolve actual Canonical browse Query: %v", err)
	}
	return *pattern.Query, *pattern.Datasource
}

func postgresReadPlan(t *testing.T) ReadPlan {
	t.Helper()
	plan, err := scopedPublication(t).BindRead(url.Values{
		"$select":  {"orderId,customerId"},
		"$filter":  {"orderId gt 2"},
		"$orderby": {"orderId desc"},
		"$top":     {"1"},
	}, TrustedReadContext{
		PrincipalID:    "subject-order-viewer",
		RowScopeID:     "tenant-a-only",
		PolicyRevision: "policy-17",
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestScopedPostgresBindsTenantInsideSQLBeforeFilterOrderLimit(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	trace := &postgresScopedTrace{
		rows: [][]sqldriver.Value{{int64(9), int64(12), from, "READY", "9000.00"}},
	}
	authority := &postgresScopedAuthority{tenant: "tenant-a"}
	provider := ScopedPostgresOrdersReference{
		DB:        postgresScopedDB(t, trace),
		Authority: authority,
		Database:  "sales_core",
	}
	pub := scopedPublication(t)
	trusted := TrustedReadContext{
		PrincipalID:    "subject-order-viewer",
		RowScopeID:     "tenant-a-only",
		PolicyRevision: "policy-17",
	}
	plan, err := AuthorizeBrowseRead(context.Background(), fixtureAuthorization(t),
		pub, url.Values{
			"$select":  {"orderId,customerId"},
			"$filter":  {"orderId gt 2"},
			"$orderby": {"orderId desc"},
			"$top":     {"1"},
		}, trusted)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t),
		pub, plan, scopedSupport(pub, plan), provider, map[string]any{"from": from})
	if err != nil {
		t.Fatal(err)
	}
	if authority.calls != 1 || authority.received.QueryID != pub.QueryID ||
		authority.received.PrincipalID != trusted.PrincipalID ||
		authority.received.RowScopeID != trusted.RowScopeID ||
		authority.received.PolicyRevision != trusted.PolicyRevision {
		t.Fatalf("incorrect authority-bound scope request: %+v", authority)
	}
	if trace.calls != 1 {
		t.Fatalf("expected one physical SQL call, got %d", trace.calls)
	}
	for _, fragment := range []string{
		"FROM public.tei_web_scoped_orders_reference AS o",
		"WHERE o.tenant_id = $1",
		"AND o.ordered_at >= $2",
		"AND o.order_id > $3",
		"ORDER BY o.order_id DESC",
		"LIMIT $4",
	} {
		if !strings.Contains(trace.sql, fragment) {
			t.Fatalf("physical SQL is missing %q: %s", fragment, trace.sql)
		}
	}
	if strings.Contains(trace.sql, "FROM public.orders AS o") {
		t.Fatal("unscoped native SQL was unexpectedly executed")
	}
	if len(trace.args) != 4 ||
		trace.args[0].Value != "tenant-a" ||
		trace.args[1].Value != from ||
		trace.args[2].Value != int64(2) ||
		trace.args[3].Value != int64(1) {
		t.Fatalf("scope/filter/Top parameters were not bound: %+v", trace.args)
	}
	if !reflect.DeepEqual(out.Columns, []string{"orderId", "customerId"}) ||
		!reflect.DeepEqual(out.Rows, [][]any{{int64(9), int64(12)}}) {
		t.Fatalf("public output includes raw or unselected fields: %+v", out)
	}
}

func TestScopedPostgresRejectsInvalidPlansBeforeDatabaseAccess(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	q, ds := postgresCanonicalInput(t)
	base := postgresReadPlan(t)
	cases := []struct {
		name string
		mutate func(*query.EmbeddedSQL, *application.DatabaseRequirement, *ReadPlan, *map[string]any)
	}{
		{"missing principal", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.PrincipalID = ""
		}},
		{"wrong query ID", func(q *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, _ *map[string]any) {
			q.ID = "other.query"
		}},
		{"wrong provider", func(q *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, _ *map[string]any) {
			q.Dialect = environment.ProviderGeneric
		}},
		{"changed canonical SQL", func(q *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, _ *map[string]any) {
			q.Source += " UNION SELECT * FROM somewhere_else"
		}},
		
		{"wrong datasource target", func(_ *query.EmbeddedSQL, ds *application.DatabaseRequirement, _ *ReadPlan, _ *map[string]any) {
			ds.TargetDatabase = "different_db"
		}},
		{"wrong canonical result mapping", func(q *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, _ *map[string]any) {
			q.ResultColumns = append([]query.ResultColumnMapping(nil), q.ResultColumns...)
			q.ResultColumns[0].Attribute = "sales.order.total"
		}},
		{"unknown selected field", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Selected = []string{"Cost"}
		}},
		{"unsupported filter field", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Filter.FieldID = "customerId"
		}},
		{"unsupported filter operator", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Filter.Operator = "or"
		}},
		{"filter value injection", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Filter.Value = "2; DROP TABLE orders"
		}},
		{"filter overflow", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Filter.Value = "99999999999999999999999999"
		}},
		{"unsupported sort field", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Order.FieldID = "customerId"
		}},
		{"missing from", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, params *map[string]any) {
			*params = map[string]any{}
		}},
		{"invalid from", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, params *map[string]any) {
			*params = map[string]any{"from": "2026-10-01"}
		}},
		{"extra caller-scope parameter", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, _ *ReadPlan, params *map[string]any) {
			*params = map[string]any{"from": from, "tei_scope_tenant": "tenant-b"}
		}},
		{"overlarge top", func(_ *query.EmbeddedSQL, _ *application.DatabaseRequirement, plan *ReadPlan, _ *map[string]any) {
			plan.Limit = 1000000
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			localQ := q
			localDS := ds
			localPlan := base
			localPlan.Selected = append([]string(nil), base.Selected...)
			filter := *base.Filter
			order := *base.Order
			localPlan.Filter = &filter
			localPlan.Order = &order
			parameters := map[string]any{"from": from}
			tc.mutate(&localQ, &localDS, &localPlan, &parameters)
			trace := &postgresScopedTrace{}
			authority := &postgresScopedAuthority{tenant: "tenant-a"}
			provider := ScopedPostgresOrdersReference{
				DB:       postgresScopedDB(t, trace), Authority: authority,
				Database: "sales_core",
			}
			_, err := provider.ExecuteScopedBrowseQuery(context.Background(), localQ,
				localDS, localPlan, parameters)
			if !errors.Is(err, ErrRejected) || trace.calls != 0 || authority.calls != 0 {
				t.Fatalf("invalid plan reached authorization or SQL: %v trace=%+v authority=%+v", err, trace, authority)
			}
		})
	}
}

func TestScopedPostgresRefusesStaleAuthorityOrOutageWithoutSQL(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	q, ds := postgresCanonicalInput(t)
	plan := postgresReadPlan(t)
	tests := []struct {
		name string
		configure func(*postgresScopedAuthority)
		want error
	}{
		{"backend outage", func(a *postgresScopedAuthority) {
			a.err = errors.New("policy database unavailable")
		}, ErrAuthorizationUnavailable},
		{"wrong principal in grant", func(a *postgresScopedAuthority) {
			a.forged = &SQLTenantScopeGrant{
				SQLTenantScopeRequest: SQLTenantScopeRequest{
					PrincipalID:    "other", RowScopeID: plan.ScopeID,
					PolicyRevision: plan.PolicyRevision, QueryID: q.ID,
				},
				TenantID: "tenant-a",
			}
		}, ErrAuthorizationContract},
		{"wrong policy revision", func(a *postgresScopedAuthority) {
			a.forged = &SQLTenantScopeGrant{
				SQLTenantScopeRequest: SQLTenantScopeRequest{
					PrincipalID:    plan.PrincipalID, RowScopeID: plan.ScopeID,
					PolicyRevision: "stale", QueryID: q.ID,
				},
				TenantID: "tenant-a",
			}
		}, ErrAuthorizationContract},
		{"empty tenant", func(a *postgresScopedAuthority) {
			a.tenant = ""
		}, ErrAuthorizationContract},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trace := &postgresScopedTrace{}
			authority := &postgresScopedAuthority{tenant: "tenant-a"}
			tc.configure(authority)
			provider := ScopedPostgresOrdersReference{
				DB:       postgresScopedDB(t, trace), Authority: authority,
				Database: "sales_core",
			}
			_, err := provider.ExecuteScopedBrowseQuery(context.Background(), q, ds, plan,
				map[string]any{"from": from})
			if !errors.Is(err, tc.want) || trace.calls != 0 || authority.calls != 1 {
				t.Fatalf("authority violation reached SQL: result=%v trace=%+v authority=%+v", err, trace, authority)
			}
		})
	}
}

func TestScopedPostgresBindsTenantAsDataEvenWhenItLooksLikeSQL(t *testing.T) {
	q, ds := postgresCanonicalInput(t)
	plan := postgresReadPlan(t)
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	attack := "tenant-a' OR 1=1 --"
	trace := &postgresScopedTrace{}
	provider := ScopedPostgresOrdersReference{
		DB:        postgresScopedDB(t, trace),
		Authority: &postgresScopedAuthority{tenant: attack},
		Database:  "sales_core",
	}
	_, err := provider.ExecuteScopedBrowseQuery(context.Background(), q, ds, plan,
		map[string]any{"from": from})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(trace.sql, attack) || !strings.Contains(trace.sql, "o.tenant_id = $1") ||
		len(trace.args) < 1 || trace.args[0].Value != attack {
		t.Fatalf("tenant parameter escaped SQL binding: query=%s args=%v", trace.sql, trace.args)
	}
}

func TestScopedPostgresPreservesCanonicalDefaultOrder(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	q, ds := postgresCanonicalInput(t)
	plan := postgresReadPlan(t)
	plan.Filter = nil
	plan.Order = nil
	trace := &postgresScopedTrace{}
	provider := ScopedPostgresOrdersReference{
		DB:        postgresScopedDB(t, trace),
		Authority: &postgresScopedAuthority{tenant: "tenant-a"},
		Database:  "sales_core",
	}
	_, err := provider.ExecuteScopedBrowseQuery(context.Background(), q, ds, plan,
		map[string]any{"from": from})
	if err != nil {
		t.Fatal(err)
	}
	if trace.calls != 1 ||
		!strings.Contains(trace.sql, "ORDER BY o.ordered_at DESC, o.order_id DESC") ||
		len(trace.args) != 3 {
		t.Fatalf("scoped realization changed Canonical default ordering: %+v", trace)
	}
}

func TestScopedPostgresRefusesTypedNilTenantAuthority(t *testing.T) {
	q, ds := postgresCanonicalInput(t)
	plan := postgresReadPlan(t)
	trace := &postgresScopedTrace{}
	var absent *postgresScopedAuthority
	provider := ScopedPostgresOrdersReference{
		DB:       postgresScopedDB(t, trace), Authority: absent,
		Database: "sales_core",
	}
	_, err := provider.ExecuteScopedBrowseQuery(context.Background(), q, ds, plan,
		map[string]any{"from": time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)})
	if !errors.Is(err, ErrAuthorizationUnavailable) || trace.calls != 0 {
		t.Fatalf("typed-nil scope authority unexpectedly invoked: %v trace=%+v", err, trace)
	}
}

func TestScopedPostgresBridgePreservesAuthorizationFailureClassification(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	pub := scopedPublication(t)
	plan := postgresReadPlan(t)
	for _, tc := range []struct {
		name string
		modify func(*postgresScopedAuthority)
		want error
	}{
		{
			name: "authorization backend unavailable",
			modify: func(a *postgresScopedAuthority) {
				a.err = errors.New("private policy database details")
			},
			want: ErrAuthorizationUnavailable,
		},
		{
			name: "malformed authority grant",
			modify: func(a *postgresScopedAuthority) {
				a.tenant = ""
			},
			want: ErrAuthorizationContract,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trace := &postgresScopedTrace{}
			authority := &postgresScopedAuthority{tenant: "tenant-a"}
			tc.modify(authority)
			out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t),
				pub, plan, scopedSupport(pub, plan),
				ScopedPostgresOrdersReference{
					DB: postgresScopedDB(t, trace),
					Authority: authority,
					Database: "sales_core",
				},
				map[string]any{"from": from},
			)
			if !errors.Is(err, tc.want) || trace.calls != 0 ||
				len(out.Rows) != 0 || len(out.Columns) != 0 {
				t.Fatalf("authorization failure was lost or escaped: result=%+v err=%v want=%v", out, err, tc.want)
			}
			if err != tc.want {
				t.Fatalf("backend diagnostics leaked through bridge: %v", err)
			}
		})
	}
}
