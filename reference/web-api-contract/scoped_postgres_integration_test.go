package webapicontract

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
)

// Strict fixture authority for the isolated Postgres integration test only.
// These grants do NOT come from TEI's production membership/policy store.
type isolatedTenantScopeAuthority struct {
	grants map[SQLTenantScopeRequest]string
}

func (a isolatedTenantScopeAuthority) ResolveSQLTenantScope(
	ctx context.Context,
	request SQLTenantScopeRequest,
) (SQLTenantScopeGrant, error) {
	if err := ctx.Err(); err != nil {
		return SQLTenantScopeGrant{}, err
	}
	tenant, ok := a.grants[request]
	if !ok || tenant == "" {
		return SQLTenantScopeGrant{}, errors.New("test scope grant unavailable")
	}
	return SQLTenantScopeGrant{
		SQLTenantScopeRequest: request,
		TenantID:              tenant,
	}, nil
}

// TestScopedTenantPostgresAgainstIsolatedDatabase proves that the existing
// tenant-aware reference really applies its SQL predicate BEFORE Filter,
// Order and Top on a PostgreSQL server. The customer-scope integration test
// does not exercise this separate SQLTenantScopeAuthority boundary.
//
// It is disabled by default, and never modifies a pre-existing relation.
// Its evidence MUST NOT be called real production authorization or RLS.
func TestScopedTenantPostgresAgainstIsolatedDatabase(t *testing.T) {
	dsn := os.Getenv("TEI_WEB_SCOPED_PG_DSN")
	if dsn == "" {
		t.Skip("tenant PostgreSQL integration not run: TEI_WEB_SCOPED_PG_DSN unset")
	}
	if os.Getenv("TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED") != "tei_web_scoped_e2e" {
		t.Fatal("refusing DDL without an explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("could not initialize PostgreSQL driver")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("isolated PostgreSQL connection unavailable")
	}
	var actualDB string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&actualDB); err != nil ||
		actualDB != "tei_web_scoped_e2e" {
		t.Fatalf("refusing DDL in unexpected database: %q", actualDB)
	}
	var existing sql.NullString
	if err := db.QueryRowContext(ctx,
		"SELECT to_regclass('public.tei_web_scoped_orders_reference')::text").Scan(&existing); err != nil {
		t.Fatal("cannot establish that the test table is absent")
	}
	if existing.Valid {
		t.Fatal("refusing to modify an existing tenant-scope table")
	}
	var version string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err != nil {
		t.Fatal("cannot determine PostgreSQL server version")
	}
	t.Logf("real tenant SQL evidence: database=%s server_version=%s", actualDB, version)

	// This checked-in, reference-only fixture is not a production migration.
	// It contains exactly a CREATE and INSERT statement. The simple split
	// is intentionally restricted to this trusted fixed fixture.
	data, err := os.ReadFile(filepath.Join("fixtures", "postgres_scoped_orders.sql"))
	if err != nil {
		t.Fatal("cannot read the committed tenant fixture")
	}
	statements := strings.Split(string(data), ";")
	if len(statements) != 3 || strings.TrimSpace(statements[2]) != "" {
		t.Fatal("tenant fixture shape changed; refuse unexpected SQL")
	}
	// Register cleanup immediately after creating the relation so a failed
	// INSERT does not leave behind fixture data.
	if _, err := db.ExecContext(ctx, strings.TrimSpace(statements[0])); err != nil {
		t.Fatal("cannot create the isolated tenant fixture")
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE public.tei_web_scoped_orders_reference"); err != nil {
			t.Errorf("failed to remove isolated tenant fixture: %v", err)
		}
	}()
	if _, err := db.ExecContext(ctx, strings.TrimSpace(statements[1])); err != nil {
		t.Fatal("cannot insert the isolated tenant fixture")
	}

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	tenantA := SQLTenantScopeRequest{
		PrincipalID: "subject-order-editor",
		RowScopeID: "scope-tenant-a",
		PolicyRevision: "e2e-policy-1",
		QueryID: core.MustCanonicalID("order_entry.recent_orders"),
	}
	tenantB := SQLTenantScopeRequest{
		PrincipalID: "subject-order-viewer",
		RowScopeID: "scope-tenant-b",
		PolicyRevision: "e2e-policy-1",
		QueryID: core.MustCanonicalID("order_entry.recent_orders"),
	}
	authority := isolatedTenantScopeAuthority{grants: map[SQLTenantScopeRequest]string{
		tenantA: "tenant-a",
		tenantB: "tenant-b",
	}}
	tests := []struct {
		name string
		scope SQLTenantScopeRequest
		options url.Values
		want [][]any
		wantColumns []string
	}{
		{
			name: "tenant A descending top",
			scope: tenantA,
			options: url.Values{"$select": {"orderId"}, "$orderby": {"orderId desc"}, "$top": {"1"}},
			want: [][]any{{int64(9)}},
			wantColumns: []string{"orderId"},
		},
		{
			name: "tenant A ascending top",
			scope: tenantA,
			options: url.Values{"$select": {"orderId"}, "$orderby": {"orderId asc"}, "$top": {"1"}},
			want: [][]any{{int64(3)}},
			wantColumns: []string{"orderId"},
		},
		{
			name: "tenant A cannot filter tenant B's high ID",
			scope: tenantA,
			options: url.Values{"$select": {"orderId"}, "$filter": {"orderId eq 100"}},
			want: nil,
			wantColumns: []string{"orderId"},
		},
		{
			name: "tenant B descending top",
			scope: tenantB,
			options: url.Values{"$select": {"orderId"}, "$orderby": {"orderId desc"}, "$top": {"1"}},
			want: [][]any{{int64(100)}},
			wantColumns: []string{"orderId"},
		},
		{
			name: "tenant B cannot filter tenant A's ID",
			scope: tenantB,
			options: url.Values{"$select": {"orderId"}, "$filter": {"orderId eq 9"}},
			want: nil,
			wantColumns: []string{"orderId"},
		},
		{
			name: "tenant A projection order and default scope",
			scope: tenantA,
			options: url.Values{"$select": {"orderId,customerId"}, "$orderby": {"orderId asc"}, "$top": {"1"}},
			want: [][]any{{int64(3), int64(11)}},
			wantColumns: []string{"orderId", "customerId"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pub := scopedPublication(t)
			plan, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), pub,
				tc.options, TrustedReadContext{
					PrincipalID: tc.scope.PrincipalID,
					RowScopeID: tc.scope.RowScopeID,
					PolicyRevision: tc.scope.PolicyRevision,
				})
			if err != nil {
				t.Fatal(err)
			}
			provider := ScopedPostgresOrdersReference{
				DB: db,
				Database: core.MustCanonicalID("sales_core"),
				Authority: authority,
			}
			got, err := ExecuteScopedCanonicalRead(ctx, bridgeWorld(t), pub, plan,
				scopedSupport(pub, plan), provider, map[string]any{"from": from})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Rows, tc.want) ||
				!reflect.DeepEqual(got.Columns, tc.wantColumns) {
				t.Fatalf("cross-tenant leakage or incorrect paging: %+v want=%v", got, tc.want)
			}
			if tc.name == "tenant A descending top" {
				raw, err := provider.ExecuteScopedBrowseQuery(ctx,
					mustScopedQuery(t), mustScopedDatasource(t), plan,
					map[string]any{"from": from})
				if err != nil {
					t.Fatal(err)
				}
				if len(raw.Raw.Rows) != 1 || len(raw.Raw.Rows[0]) != 5 {
					t.Fatalf("unexpected physical result: %+v", raw.Raw)
				}
				for i := 1; i < 5; i++ {
					if raw.Raw.Rows[0][i] != nil {
						t.Fatalf("unselected column %d contained physical data: %#v", i, raw.Raw.Rows[0][i])
					}
				}
			}
		})
	}
	// A valid Browse grant cannot be replayed with a different policy,
	// even if the attacker guesses an authorized Tenant ID.
	pub := scopedPublication(t)
	forged := tenantA
	forged.PolicyRevision = "outdated-policy"
	plan, err := pub.BindRead(url.Values{"$select": {"orderId"}},
		TrustedReadContext{
			PrincipalID: forged.PrincipalID,
			RowScopeID: forged.RowScopeID,
			PolicyRevision: forged.PolicyRevision,
		})
	if err != nil {
		t.Fatal(err)
	}
	out, err := ExecuteScopedCanonicalRead(ctx, bridgeWorld(t), pub, plan,
		scopedSupport(pub, plan), ScopedPostgresOrdersReference{
			DB: db, Database: core.MustCanonicalID("sales_core"), Authority: authority,
		}, map[string]any{"from": from})
	if !errors.Is(err, ErrAuthorizationUnavailable) || len(out.Rows) != 0 {
		t.Fatalf("forged/stale tenant scope escaped: result=%+v err=%v", out, err)
	}
}
