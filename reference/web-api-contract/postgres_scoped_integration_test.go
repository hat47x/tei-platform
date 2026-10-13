package webapicontract

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
)

// This is a REAL PostgreSQL integration probe, unlike the database/sql fake.
// It is opt-in and intentionally requires a dedicated, isolated database.
// NEVER point this test at a live application database: it creates a table
// named public.orders. No existing public.orders is modified or dropped.
//
// Setup example (in a disposable Postgres instance):
//   CREATE DATABASE tei_web_scoped_e2e;
//   TEI_WEB_SCOPED_PG_DSN='postgres://.../tei_web_scoped_e2e?sslmode=disable' \
//     TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
//     go test ./reference/web-api-contract -run TestScopedPostgresAgainstIsolatedDatabase -v
func TestScopedPostgresAgainstIsolatedDatabase(t *testing.T) {
	dsn := os.Getenv("TEI_WEB_SCOPED_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_WEB_SCOPED_PG_DSN unset")
	}
	if os.Getenv("TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED") != "tei_web_scoped_e2e" {
		t.Fatal("refusing to create public.orders without the explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("failed to initialize PostgreSQL integration driver")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("cannot connect to isolated PostgreSQL integration database")
	}
	var databaseName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil ||
		databaseName != "tei_web_scoped_e2e" {
		t.Fatalf("database identity mismatch (refusing DDL): %q", databaseName)
	}
	var existing sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.orders')::text").Scan(&existing); err != nil {
		t.Fatal("cannot verify absence of existing public.orders")
	}
	if existing.Valid {
		t.Fatal("public.orders already exists: refusing to touch existing user data")
	}
	_, err = db.ExecContext(ctx, "CREATE TABLE public.orders (order_id BIGINT PRIMARY KEY, customer_id BIGINT NOT NULL, ordered_at TIMESTAMPTZ NOT NULL, status TEXT NOT NULL, total_amount BIGINT NOT NULL)")
	if err != nil {
		t.Fatal("cannot create disposable orders fixture")
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE public.orders"); err != nil {
			t.Errorf("disposable test table cleanup failed: %v", err)
		}
	}()
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	// Tenant 22's extreme/high-ranked value must NOT affect Tenant 11's top
	// or filter results. Alternating input order deliberately defeats a
	// naive pre-scope pagination strategy.
	rows := []struct {
		orderID, customerID, total int64
		orderedAt                  time.Time
		status                     string
	}{
		{100, 22, 99000, from.Add(4 * time.Hour), "OTHER"},
		{3, 11, 3000, from.Add(2 * time.Hour), "OPEN"},
		{1, 22, 100, from.Add(1 * time.Hour), "OTHER"},
		{9, 11, 9000, from.Add(3 * time.Hour), "READY"},
	}
	for _, row := range rows {
		_, err := db.ExecContext(ctx,
			"INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES ($1, $2, $3, $4, $5)",
			row.orderID, row.customerID, row.orderedAt, row.status, row.total)
		if err != nil {
			t.Fatal("disposable fixture insertion failed")
		}
	}
	tests := []struct {
		name       string
		customerID int64
		scopeID    string
		options    url.Values
		want       [][]any
	}{
		{"tenant 11 descending top", 11, "scope-11", url.Values{
			"$select": {"orderId"}, "$orderby": {"orderId desc"}, "$top": {"1"},
		}, [][]any{{int64(9)}}},
		{"tenant 11 ascending top", 11, "scope-11", url.Values{
			"$select": {"orderId"}, "$orderby": {"orderId asc"}, "$top": {"1"},
		}, [][]any{{int64(3)}}},
		{"other tenant's ID invisible", 11, "scope-11", url.Values{
			"$select": {"orderId"}, "$filter": {"orderId eq 100"},
		}, nil},
		{"tenant 22 gets own row only", 22, "scope-22", url.Values{
			"$select": {"orderId"}, "$orderby": {"orderId desc"}, "$top": {"1"},
		}, [][]any{{int64(100)}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pub := scopedPublication(t)
			trusted := TrustedReadContext{
				PrincipalID: "subject-order-viewer",
				RowScopeID: tc.scopeID, PolicyRevision: "e2e-r1",
			}
			plan, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), pub, tc.options, trusted)
			if err != nil {
				t.Fatal(err)
			}
			provider := ScopedRecentOrdersPostgres{
				DB: db, Database: core.MustCanonicalID("sales_core"),
				Scope: PostgresCustomerScope{
					PrincipalID: trusted.PrincipalID,
					RowScopeID: trusted.RowScopeID,
					PolicyRevision: trusted.PolicyRevision,
					CustomerID: tc.customerID,
				},
			}
			got, err := ExecuteScopedCanonicalRead(ctx, scopedCanonicalWorld(t),
				pub, plan, scopedSupport(pub, plan), provider, map[string]any{"from": from})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Rows, tc.want) ||
				!reflect.DeepEqual(got.Columns, []string{"orderId"}) {
				t.Fatalf("cross-tenant or ordering leakage: %+v, want=%v", got, tc.want)
			}
		})
	}
}
