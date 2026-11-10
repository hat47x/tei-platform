package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
)

func xteaWorkspace() string { return filepath.Join("..", "..", "examples", "xtea-lowcode") }

func runView(t *testing.T, args ...string) (int, map[string]any, string) {
	t.Helper()
	var out bytes.Buffer
	exit, err := runViewCommand(args, &out)
	if err != nil {
		t.Fatalf("runViewCommand: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	return exit, decoded, out.String()
}

func errorCode(report map[string]any) string {
	if e, ok := report["error"].(map[string]any); ok {
		code, _ := e["code"].(string)
		return code
	}
	return ""
}

func browseReadArgs(extra ...string) []string {
	base := []string{"-workspace", xteaWorkspace(), "-application", "order_entry", "-pattern", "order_entry.recent_orders",
		"-execute", "-environment", "development", "-format", "json"}
	return append(base, extra...)
}

func TestBrowseDSNVariableNameIsDerivedFromTheInstanceOnly(t *testing.T) {
	for instance, want := range map[string]string{
		"sales_core.dev":   "TEI_DSN_SALES_CORE_DEV",
		"master-data.prod": "TEI_DSN_MASTER_DATA_PROD",
	} {
		if got := browseDSNVariable(core.CanonicalID(instance)); got != want {
			t.Fatalf("%s -> %s, want %s", instance, got, want)
		}
	}
}

func TestBrowseReadStopsBeforeConnectingWhenTheInstanceHasNoConnection(t *testing.T) {
	t.Setenv("TEI_DSN_SALES_CORE_DEV", "")
	// A test-suite DSN must never stand in for the instance's own connection.
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://leak:leak@127.0.0.1:1/leak")
	exit, report, raw := runView(t, browseReadArgs("-input", "status=OPEN", "-input", "status_operator=eq")...)
	if exit != 1 || errorCode(report) != "DATASOURCE_CONNECTION_UNAVAILABLE" || !strings.Contains(raw, "TEI_DSN_SALES_CORE_DEV") {
		t.Fatalf("exit=%d report=%s", exit, raw)
	}
	if strings.Contains(raw, "leak") {
		t.Fatalf("another variable's value reached the output: %s", raw)
	}
}

func TestBrowseReadArgumentRefusals(t *testing.T) {
	t.Setenv("TEI_DSN_SALES_CORE_DEV", "")
	for _, tc := range []struct {
		name string
		args []string
		code string
		exit int
	}{
		{"execute without environment", []string{"-workspace", xteaWorkspace(), "-application", "order_entry", "-pattern", "order_entry.recent_orders", "-execute", "-format", "json"}, "INVALID_ARGUMENT", 2},
		{"environment without execute", []string{"-workspace", xteaWorkspace(), "-application", "order_entry", "-pattern", "order_entry.recent_orders", "-environment", "development"}, "INVALID_ARGUMENT", 2},
		{"text format", browseReadArgs()[:len(browseReadArgs())-2], "INVALID_ARGUMENT", 2},
		{"undeclared input name", browseReadArgs("-input", "nope=1"), "INVALID_ARGUMENT", 2},
		{"non-integer for integer input", browseReadArgs("-input", "page_offset=x"), "INVALID_ARGUMENT", 2},
		{"unknown environment", append(browseReadArgs()[:7:7], "-environment", "production", "-format", "json"), "VIEW_FAILED", 1},
		{"not a browse pattern", []string{"-workspace", xteaWorkspace(), "-application", "order_entry", "-pattern", "order_entry.order_record", "-execute", "-environment", "development", "-format", "json"}, "VIEW_FAILED", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exit, report, raw := runView(t, tc.args...)
			if exit != tc.exit || errorCode(report) != tc.code {
				t.Fatalf("exit=%d want %d code=%q want %q: %s", exit, tc.exit, errorCode(report), tc.code, raw)
			}
		})
	}
}

func TestBrowseInputsDecodeByDeclaredTypeAndNeverGuess(t *testing.T) {
	inputs := []queryInput{{"status", "std.code"}, {"n", "std.integer"}, {"when", "std.date"}}
	got, _, err := browseInputs(inputs, map[string]string{"status": "OPEN", "n": "20"})
	if err != nil || !reflect.DeepEqual(got, map[string]any{"status": "OPEN", "n": int64(20)}) {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	if _, code, err := browseInputs(inputs, map[string]string{"when": "2026-11-10"}); err == nil || code != "INPUT_TYPE_UNSUPPORTED" {
		t.Fatalf("code=%q err=%v", code, err)
	}
}

// TestBrowseReadReportsDataContextFromActualPostgres is opt-in and read-only. It
// needs TEI_XTEA_RECENT_ORDERS_PG_DSN pointing at a disposable PostgreSQL that
// already holds sales."order"(order_id, customer_id, status). The expected rows
// are computed by this test from an unfiltered SELECT, not from the product SQL.
func TestBrowseReadReportsDataContextFromActualPostgres(t *testing.T) {
	dsn := os.Getenv("TEI_XTEA_RECENT_ORDERS_PG_DSN")
	if dsn == "" {
		t.Skip("actual PostgreSQL view -execute NOT RUN: TEI_XTEA_RECENT_ORDERS_PG_DSN unset")
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
		t.Fatal("cannot force a read-only session")
	}
	rows, err := db.QueryContext(ctx, `SELECT order_id::text, status::text FROM "sales"."order"`)
	if err != nil {
		t.Fatalf("independent observation failed: %v", err)
	}
	type row struct{ order, status string }
	var observed []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.order, &r.status); err != nil {
			t.Fatal(err)
		}
		observed = append(observed, r)
	}
	rows.Close()
	sort.Slice(observed, func(i, j int) bool {
		if observed[i].status != observed[j].status {
			return observed[i].status < observed[j].status
		}
		return observed[i].order < observed[j].order
	})
	t.Setenv("TEI_DSN_SALES_CORE_DEV", dsn)

	identities := func(report map[string]any) []string {
		slice := report["result"].(map[string]any)["dataContext"].(map[string]any)["slice"].(map[string]any)
		out := []string{}
		for _, r := range slice["rows"].([]any) {
			out = append(out, r.(map[string]any)["identity"].(map[string]any)["sales.order.order_id"].(string))
		}
		return out
	}
	slice := func(report map[string]any) map[string]any {
		return report["result"].(map[string]any)["dataContext"].(map[string]any)["slice"].(map[string]any)
	}

	for _, tc := range []struct {
		status, operator string
		offset           int
	}{{"OPEN", "eq", 0}, {"OPEN", "eq", 20}, {"OPEN", "ne", 0}, {"__absent__", "eq", 0}} {
		matched := []string{}
		for _, r := range observed {
			if (tc.operator == "eq") == (r.status == tc.status) {
				matched = append(matched, r.order)
			}
		}
		end := tc.offset + 20
		if end > len(matched) {
			end = len(matched)
		}
		want := []string{}
		if tc.offset < len(matched) {
			want = matched[tc.offset:end]
		}
		exit, report, raw := runView(t, browseReadArgs("-input", "status="+tc.status, "-input", "status_operator="+tc.operator,
			"-input", fmt.Sprintf("page_offset=%d", tc.offset))...)
		if exit != 0 {
			t.Fatalf("%+v: exit=%d %s", tc, exit, raw)
		}
		if got := identities(report); !reflect.DeepEqual(got, want) {
			t.Fatalf("%+v: identities=%v want=%v", tc, got, want)
		}
		wantCoverage := "partial"
		if tc.offset == 0 && len(want) < 20 {
			wantCoverage = "complete"
		}
		if s := slice(report); s["coverage"] != wantCoverage || int(s["rowCount"].(float64)) != len(want) {
			t.Fatalf("%+v: slice=%v want coverage %s", tc, s, wantCoverage)
		}
		for _, secret := range []string{dsn, "password", "postgres://"} {
			if strings.Contains(raw, secret) {
				t.Fatalf("connection detail leaked into the output: %q", secret)
			}
		}
	}

	exit, report, raw := runView(t, browseReadArgs("-input", "status=OPEN", "-input", "status_operator=ge")...)
	if exit != 1 || errorCode(report) != "BROWSE_READ_FAILED" {
		t.Fatalf("undeclared operator must fail, not read: exit=%d %s", exit, raw)
	}
}
