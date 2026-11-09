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

	"tei/internal/core"
)

// Tenant用の参照（ScopedPostgresOrdersReference）でも、公開していない項目（orderedAt）の
// 大小関係が、公開項目だけの応答の並びから推測できるかを、実PostgreSQLで確かめる特性化試験。
// 顧客用の参照の同種の試験は postgres_scoped_inference_integration_test.go にある。
// 実DBの足場と安全策は postgres_scoped_integration_test.go と同じで、
// 廃棄可能な独立DBでだけ動く（DSN未設定ではSKIPし、実証したことにはならない）。
//
//	TEI_WEB_SCOPED_PG_DSN='postgres://.../tei_web_scoped_e2e?sslmode=disable' \
//	TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
//	  go test ./reference/web-api-contract -run TestScopedTenantPostgres -v
//
// この試験が固定するのは「現在の参照実装の既知の限界」であり、望ましい動作の証明ではない。
func TestScopedTenantPostgresDefaultOrderRevealsUnpublishedOrderedAt(t *testing.T) {
	ctx, db, from := openIsolatedTenantFixture(t)

	tenantA := SQLTenantScopeRequest{
		PrincipalID: "subject-order-viewer", RowScopeID: "scope-tenant-a",
		PolicyRevision: "e2e-policy-1", QueryID: core.MustCanonicalID("order_entry.recent_orders"),
	}
	authority := isolatedTenantScopeAuthority{grants: map[SQLTenantScopeRequest]string{tenantA: "tenant-a"}}

	run := func(t *testing.T, options url.Values) [][]any {
		t.Helper()
		pub := scopedPublication(t)
		plan, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), pub, options, TrustedReadContext{
			PrincipalID: tenantA.PrincipalID, RowScopeID: tenantA.RowScopeID, PolicyRevision: tenantA.PolicyRevision,
		})
		if err != nil {
			t.Fatal(err)
		}
		provider := ScopedPostgresOrdersReference{DB: db, Database: core.MustCanonicalID("sales_core"), Authority: authority}
		got, err := ExecuteScopedCanonicalRead(ctx, bridgeWorld(t), pub, plan,
			scopedSupport(pub, plan), provider, map[string]any{"from": from})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Columns, []string{"orderId"}) {
			t.Fatalf("the response must expose only the published column: %v", got.Columns)
		}
		return got.Rows
	}

	// 前提: 呼出し元は、orderedAtを選択・並べ替えに指定できず、`from`や別Tenantも要求で渡せない。
	trusted := TrustedReadContext{PrincipalID: tenantA.PrincipalID, RowScopeID: tenantA.RowScopeID, PolicyRevision: tenantA.PolicyRevision}
	for _, options := range []url.Values{
		{"$select": {"orderedAt"}},
		{"$select": {"orderId"}, "$orderby": {"orderedAt desc"}},
		{"$select": {"orderId"}, "$filter": {"orderedAt ge 2026-10-02"}},
		{"$select": {"orderId"}, "from": {"2026-10-02T00:00:00Z"}},
		{"$select": {"orderId"}, "$from": {"2026-10-02T00:00:00Z"}},
		{"$select": {"orderId"}, "tenant_id": {"tenant-b"}},
	} {
		if _, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), scopedPublication(t), options, trusted); !errors.Is(err, ErrRejected) {
			t.Fatalf("the request must be rejected (%v): %v", options, err)
		}
	}

	// 公開項目（tenant-aの注文3・9）は同一で、非公開のorderedAtの大小関係だけを変える。
	// tenant-bの行は、スコープにより応答へ現れないことの対照。
	setOrderedAt := func(t *testing.T, at3, at9 time.Time) {
		t.Helper()
		for id, at := range map[int64]time.Time{3: at3, 9: at9} {
			if _, err := db.ExecContext(ctx,
				"UPDATE public.tei_web_scoped_orders_reference SET ordered_at = $1 WHERE tenant_id = 'tenant-a' AND order_id = $2", at, id); err != nil {
				t.Fatal("cannot update the disposable fixture")
			}
		}
	}
	setOrderedAt(t, from.Add(1*time.Hour), from.Add(3*time.Hour)) // A: 9が新しい
	defaultA := run(t, url.Values{"$select": {"orderId"}})
	top1A := run(t, url.Values{"$select": {"orderId"}, "$top": {"1"}})
	explicitA := run(t, url.Values{"$select": {"orderId"}, "$orderby": {"orderId asc"}})
	setOrderedAt(t, from.Add(3*time.Hour), from.Add(1*time.Hour)) // B: 3が新しい
	defaultB := run(t, url.Values{"$select": {"orderId"}})
	top1B := run(t, url.Values{"$select": {"orderId"}, "$top": {"1"}})
	explicitB := run(t, url.Values{"$select": {"orderId"}, "$orderby": {"orderId asc"}})

	if want := [][]any{{int64(9)}, {int64(3)}}; !reflect.DeepEqual(defaultA, want) {
		t.Fatalf("dataset A default order: %v, want %v", defaultA, want)
	}
	if want := [][]any{{int64(3)}, {int64(9)}}; !reflect.DeepEqual(defaultB, want) {
		t.Fatalf("dataset B default order: %v, want %v", defaultB, want)
	}
	if reflect.DeepEqual(defaultA, defaultB) {
		t.Fatal("the default order no longer follows the unpublished orderedAt: if this is a deliberate fix, update this characterization test")
	}
	if want := [][]any{{int64(9)}}; !reflect.DeepEqual(top1A, want) {
		t.Fatalf("dataset A newest row: %v, want %v", top1A, want)
	}
	if want := [][]any{{int64(3)}}; !reflect.DeepEqual(top1B, want) {
		t.Fatalf("dataset B newest row: %v, want %v", top1B, want)
	}
	want := [][]any{{int64(3)}, {int64(9)}}
	if !reflect.DeepEqual(explicitA, want) || !reflect.DeepEqual(explicitB, want) {
		t.Fatalf("explicit orderId order must hide orderedAt: A=%v B=%v want %v", explicitA, explicitB, want)
	}
}

// openIsolatedTenantFixture は、廃棄可能な独立DBにtenant用のfixtureを作る。
// 安全策（DSN・DB名の確認・既存テーブルの不在確認）は、既存のtenant実DB試験と同じ。
func openIsolatedTenantFixture(t *testing.T) (context.Context, *sql.DB, time.Time) {
	t.Helper()
	dsn := os.Getenv("TEI_WEB_SCOPED_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_WEB_SCOPED_PG_DSN unset")
	}
	if os.Getenv("TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED") != "tei_web_scoped_e2e" {
		t.Fatal("refusing DDL without an explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("could not initialize PostgreSQL driver")
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("isolated PostgreSQL connection unavailable")
	}
	var actualDB string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&actualDB); err != nil || actualDB != "tei_web_scoped_e2e" {
		t.Fatalf("refusing DDL in unexpected database: %q", actualDB)
	}
	var existing sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.tei_web_scoped_orders_reference')::text").Scan(&existing); err != nil {
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

	data, err := os.ReadFile(filepath.Join("fixtures", "postgres_scoped_orders.sql"))
	if err != nil {
		t.Fatal("cannot read the committed tenant fixture")
	}
	createSQL, insertTail, ok := strings.Cut(string(data), ";\n\nINSERT INTO")
	if !ok || !strings.HasSuffix(strings.TrimSpace(insertTail), ";") ||
		!strings.Contains(createSQL, "CREATE TABLE public.tei_web_scoped_orders_reference") {
		t.Fatal("tenant fixture shape changed; refuse unexpected SQL")
	}
	insertSQL := "INSERT INTO " + strings.TrimSuffix(strings.TrimSpace(insertTail), ";")
	if _, err := db.ExecContext(ctx, strings.TrimSpace(createSQL)); err != nil {
		t.Fatal("cannot create the isolated tenant fixture")
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE public.tei_web_scoped_orders_reference"); err != nil {
			t.Errorf("failed to remove isolated tenant fixture: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, insertSQL); err != nil {
		t.Fatal("cannot insert the isolated tenant fixture")
	}
	return ctx, db, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
}
