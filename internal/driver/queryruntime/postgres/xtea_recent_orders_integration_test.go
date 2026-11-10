package postgresquery_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

// TestXTEARecentOrdersReadsActualPostgresThroughEnvironmentBinding is opt-in and read-only.
//
// It never creates, alters or seeds anything: the operator must point
// TEI_XTEA_RECENT_ORDERS_PG_DSN at a disposable PostgreSQL that already holds
// sales."order"(order_id, customer_id, status). The session is forced read-only
// before any statement runs.
//
// Independent Expected: the test reads every row with its own unfiltered SQL and
// applies the status predicate in Go, so neither the product SQL text, its named
// parameter binding nor the Driver mapping is used to compute the expectation.
//
// It proves Pattern -> Query -> Datasource -> Environment instance/provider ->
// actual SQL -> Rowset -> Semantic mapping in one attempt. It is NOT a Function
// Data Context, authorization or product-level read certification.
func TestXTEARecentOrdersReadsActualPostgresThroughEnvironmentBinding(t *testing.T) {
	dsn := os.Getenv("TEI_XTEA_RECENT_ORDERS_PG_DSN")
	if dsn == "" {
		t.Skip("actual PostgreSQL read of order_entry.recent_orders NOT RUN: TEI_XTEA_RECENT_ORDERS_PG_DSN unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize PostgreSQL driver")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, "SET default_transaction_read_only = on"); err != nil {
		t.Fatal("cannot force a read-only PostgreSQL session")
	}
	var readOnly, version string
	if err := db.QueryRowContext(ctx, "SHOW transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("refusing to run: session is not read-only (%q)", readOnly)
	}
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err != nil {
		t.Fatal("cannot determine actual PostgreSQL server version")
	}
	t.Logf("actual PostgreSQL recent_orders read: server_version=%s", version)

	all, err := db.QueryContext(ctx, `SELECT order_id::text, customer_id::text, status::text FROM "sales"."order"`)
	if err != nil {
		t.Fatalf("independent observation of sales.order failed: %v", err)
	}
	type row struct{ order, customer, status string }
	var observed []row
	for all.Next() {
		var r row
		if err := all.Scan(&r.order, &r.customer, &r.status); err != nil {
			t.Fatal(err)
		}
		observed = append(observed, r)
	}
	if err := all.Err(); err != nil {
		t.Fatal(err)
	}
	all.Close()
	if len(observed) == 0 {
		t.Fatal("sales.order is empty: a zero-row result could not be told apart from an unconnected read")
	}

	loaded, err := workspace.Load(filepath.Join("..", "..", "..", "..", "examples", "xtea-lowcode"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := driver.NewResolver(loaded.World)
	executor := postgresquery.Executor{DB: db, Database: "sales_core"}

	statuses := map[string]bool{"__tei_status_absent__": true}
	for _, r := range observed {
		statuses[r.status] = true
	}
	// Independent Expected: the Pattern declares sort status asc and page-size 20.
	// The test sorts in Go with order_id as the tie-break the Query adds for stable paging.
	const pageSize = 20
	sawFullPage, sawSecondPage := false, false
	for status := range statuses {
		for _, operator := range []string{"eq", "ne"} {
			matched := []row{}
			for _, r := range observed {
				if (operator == "eq") == (r.status == status) {
					matched = append(matched, r)
				}
			}
			sort.Slice(matched, func(i, j int) bool {
				if matched[i].status != matched[j].status {
					return matched[i].status < matched[j].status
				}
				return matched[i].order < matched[j].order
			})
			for offset := 0; offset == 0 || offset < len(matched); offset += pageSize {
				end := offset + pageSize
				if end > len(matched) {
					end = len(matched)
				}
				want := []string{}
				if offset < len(matched) {
					for _, r := range matched[offset:end] {
						want = append(want, r.order+"\x00"+r.customer)
					}
				}
				got, err := resolver.ExecuteBrowseQueryInEnvironment(ctx, "development", "order_entry",
					"order_entry.recent_orders", executor, map[string]any{
						"status": status, "status_operator": operator, "page_offset": int64(offset)})
				if err != nil {
					t.Fatalf("status=%q op=%s offset=%d: did not execute on actual PostgreSQL: %v", status, operator, offset, err)
				}
				if got.Instance.ID != "sales_core.dev" || got.Execution.Resolution.Query == nil ||
					got.Execution.Resolution.Query.ID != "order_entry.recent_orders" {
					t.Fatalf("result not attributed to the declared instance/Query: %+v", got.Instance)
				}
				columns := got.Execution.Result.Mapped.Columns
				if len(columns) != 2 || columns[0].Attribute != "sales.order.order_id" || columns[1].Attribute != "sales.order.customer_id" {
					t.Fatalf("mapped columns=%+v", columns)
				}
				actual := []string{}
				for _, r := range got.Execution.Result.Mapped.Rows {
					actual = append(actual, fmt.Sprint(textValue(r[0]))+"\x00"+fmt.Sprint(textValue(r[1])))
				}
				if !reflect.DeepEqual(actual, want) { // order is part of the Expected: no re-sorting
					t.Fatalf("status=%q op=%s offset=%d: physical rows mismatch: actual=%d want=%d", status, operator, offset, len(actual), len(want))
				}
				if len(actual) == pageSize {
					sawFullPage = true
				}
				if offset > 0 {
					sawSecondPage = true
				}
			}
		}
	}
	if !sawFullPage || !sawSecondPage {
		t.Fatalf("data does not exercise paging (fullPage=%v secondPage=%v): seed >20 matching rows", sawFullPage, sawSecondPage)
	}
}

func textValue(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
