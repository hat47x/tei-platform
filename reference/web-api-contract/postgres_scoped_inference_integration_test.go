package webapicontract

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"tei/internal/core"
)

// 公開していない項目（orderedAt）の値が、公開項目だけを選んだ応答から推測できるかを、
// 実PostgreSQLで確かめる特性化試験。実DBの足場と安全策は postgres_scoped_integration_test.go と同じで、
// 廃棄可能な独立DBでだけ動く（DSN未設定ではSKIPし、実証したことにはならない）。
//
//	TEI_WEB_SCOPED_PG_DSN='postgres://.../tei_web_scoped_e2e?sslmode=disable' \
//	TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
//	  go test ./reference/web-api-contract -run TestScopedPostgresDefaultOrder -v
//
// この試験が固定するのは「現在の参照実装の既知の限界」であり、望ましい動作の証明ではない。
// 既定の並びを公開項目に限る、または並びに使う項目の権限を要求する変更を入れたら、
// この試験は、その変更が意図したものであることを確かめたうえで、更新すること。
func TestScopedPostgresDefaultOrderRevealsUnpublishedOrderedAt(t *testing.T) {
	ctx, db, from := openIsolatedScopedOrders(t)

	type dataset struct {
		name string
		// 公開項目（orderId・customerId）はどのデータセットでも同一で、
		// 非公開項目orderedAtの大小関係だけが異なる。
		orderedAt map[int64]time.Duration
	}
	datasets := []dataset{
		{"A: 9が最新、3が最古", map[int64]time.Duration{3: 1 * time.Hour, 5: 2 * time.Hour, 9: 3 * time.Hour}},
		{"B: 3が最新、9が最古", map[int64]time.Duration{3: 3 * time.Hour, 5: 2 * time.Hour, 9: 1 * time.Hour}},
	}
	// 別顧客の行は、スコープにより応答へ現れないことの対照として入れておく。
	const otherCustomer, ownCustomer = int64(22), int64(11)

	run := func(t *testing.T, options url.Values) [][]any {
		t.Helper()
		pub := scopedPublication(t)
		trusted := TrustedReadContext{PrincipalID: "subject-order-viewer", RowScopeID: "scope-11", PolicyRevision: "e2e-r1"}
		plan, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), pub, options, trusted)
		if err != nil {
			t.Fatal(err)
		}
		provider := ScopedRecentOrdersPostgres{
			DB: db, Database: core.MustCanonicalID("sales_core"),
			Scope: PostgresCustomerScope{
				PrincipalID: trusted.PrincipalID, RowScopeID: trusted.RowScopeID,
				PolicyRevision: trusted.PolicyRevision, CustomerID: ownCustomer,
			},
		}
		got, err := ExecuteScopedCanonicalRead(ctx, scopedCanonicalWorld(t), pub, plan,
			scopedSupport(pub, plan), provider, map[string]any{"from": from})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Columns, []string{"orderId"}) {
			t.Fatalf("the response must expose only the published column: %v", got.Columns)
		}
		return got.Rows
	}

	// 前提: 呼出し元は、公開していないorderedAtを選択にも並べ替えにも指定できない。
	for _, options := range []url.Values{
		{"$select": {"orderedAt"}},
		{"$select": {"orderId"}, "$orderby": {"orderedAt desc"}},
	} {
		trusted := TrustedReadContext{PrincipalID: "subject-order-viewer", RowScopeID: "scope-11", PolicyRevision: "e2e-r1"}
		if _, err := AuthorizeBrowseRead(ctx, fixtureAuthorization(t), scopedPublication(t), options, trusted); !errors.Is(err, ErrRejected) {
			t.Fatalf("the unpublished orderedAt must not be selectable or sortable by the caller (%v): %v", options, err)
		}
	}

	results := make([]map[string][][]any, len(datasets))
	for i, set := range datasets {
		if _, err := db.ExecContext(ctx, "DELETE FROM public.orders"); err != nil {
			t.Fatal("cannot reset the disposable fixture")
		}
		for orderID, offset := range set.orderedAt {
			insertScopedOrder(t, ctx, db, orderID, ownCustomer, from.Add(offset), "OPEN", orderID*1000)
		}
		insertScopedOrder(t, ctx, db, 100, otherCustomer, from.Add(10*time.Hour), "OTHER", 99000)

		results[i] = map[string][][]any{}
		t.Run(set.name, func(t *testing.T) {
			results[i]["default"] = run(t, url.Values{"$select": {"orderId"}})
			results[i]["default-top1"] = run(t, url.Values{"$select": {"orderId"}, "$top": {"1"}})
			results[i]["explicit-orderId"] = run(t, url.Values{"$select": {"orderId"}, "$orderby": {"orderId asc"}})
		})
	}

	// 既定の並びは、非公開のorderedAtの降順に従う。データセットが違えば、公開項目が同じでも並びが違う。
	if want := [][]any{{int64(9)}, {int64(5)}, {int64(3)}}; !reflect.DeepEqual(results[0]["default"], want) {
		t.Fatalf("dataset A default order: %v, want %v", results[0]["default"], want)
	}
	if want := [][]any{{int64(3)}, {int64(5)}, {int64(9)}}; !reflect.DeepEqual(results[1]["default"], want) {
		t.Fatalf("dataset B default order: %v, want %v", results[1]["default"], want)
	}
	if reflect.DeepEqual(results[0]["default"], results[1]["default"]) {
		t.Fatal("the default order no longer follows the unpublished orderedAt: if this is a deliberate fix, update this characterization test")
	}
	// $top=1だけでも、どの行がorderedAtで最新かが分かる。
	if want := [][]any{{int64(9)}}; !reflect.DeepEqual(results[0]["default-top1"], want) {
		t.Fatalf("dataset A newest row: %v, want %v", results[0]["default-top1"], want)
	}
	if want := [][]any{{int64(3)}}; !reflect.DeepEqual(results[1]["default-top1"], want) {
		t.Fatalf("dataset B newest row: %v, want %v", results[1]["default-top1"], want)
	}
	// 対照: 公開項目で明示的に並べれば、非公開項目の違いは応答に現れない。
	want := [][]any{{int64(3)}, {int64(5)}, {int64(9)}}
	for i := range datasets {
		if !reflect.DeepEqual(results[i]["explicit-orderId"], want) {
			t.Fatalf("%s: explicit orderId order: %v, want %v", datasets[i].name, results[i]["explicit-orderId"], want)
		}
	}
}

func insertScopedOrder(t *testing.T, ctx context.Context, db *sql.DB, orderID, customerID int64, orderedAt time.Time, status string, total int64) {
	t.Helper()
	if _, err := db.ExecContext(ctx,
		"INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES ($1, $2, $3, $4, $5)",
		orderID, customerID, orderedAt, status, total); err != nil {
		t.Fatal("disposable fixture insertion failed")
	}
}

// openIsolatedScopedOrders は、廃棄可能な独立DBに接続し、public.ordersを作る。
// 安全策（DSN・DB名の確認・既存テーブルの不在確認）は、既存の実DB試験と同じ。
func openIsolatedScopedOrders(t *testing.T) (context.Context, *sql.DB, time.Time) {
	t.Helper()
	dsn := os.Getenv("TEI_WEB_SCOPED_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_WEB_SCOPED_PG_DSN unset")
	}
	if os.Getenv("TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED") != "tei_web_scoped_e2e" {
		t.Fatal("refusing to create public.orders without the explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("failed to initialize PostgreSQL integration driver")
	}
	t.Cleanup(func() { db.Close() })
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
	var serverVersion string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&serverVersion); err != nil {
		t.Fatal("could not establish actual PostgreSQL server version")
	}
	t.Logf("real PostgreSQL evidence: database=%s server_version=%s", databaseName, serverVersion)
	if _, err := db.ExecContext(ctx, "CREATE TABLE public.orders (order_id BIGINT PRIMARY KEY, customer_id BIGINT NOT NULL, ordered_at TIMESTAMPTZ NOT NULL, status TEXT NOT NULL, total_amount BIGINT NOT NULL)"); err != nil {
		t.Fatal("cannot create disposable orders fixture")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE public.orders"); err != nil {
			t.Errorf("disposable test table cleanup failed: %v", err)
		}
	})
	return ctx, db, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
}
