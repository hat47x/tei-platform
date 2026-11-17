package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Unlike the previous alternate-Relation test, this fixture changes the
// columns, Query input, PostgreSQL table and two-component row identity.
func TestIndependentWarehouseDataFormThroughProductView(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL unavailable")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("isolated database confirmation missing")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != databaseName {
		t.Fatalf("expected disposable database %q, got %q: %v", databaseName, actual, err)
	}
	if _, err := db.Exec("drop table if exists public.stock_receipts"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("create table public.stock_receipts (zone_code varchar not null, lot_code varchar not null, receipt_seq bigint not null, units numeric(12,3) not null, primary key (lot_code, receipt_seq))"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.stock_receipts"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	fixtures := filepath.Join("..", "..", "testdata", "destination-order-change", "independent-model")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("seed warehouse model: %v", err)
	}

	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	copyTreeForDestination(t, filepath.Join(fixtures, "workspace"), root)
	extend := func(path, from, to string) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(path))
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(raw), from) != 1 {
			t.Fatalf("expected one extension point %q in %s", from, path)
		}
		if err := os.WriteFile(file, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	extend("databases/sales-core/database.tei",
		"    uses semantic sales", "    uses semantic sales\n    uses semantic warehouse")
	extend("environments/development/environment.tei",
		"    bind order_entry.sales -> sales_core.dev",
		"    bind order_entry.sales -> sales_core.dev\n    bind warehouse_count.stock -> sales_core.dev")
	binary := buildCLIForLifecycle(t)
	checked := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
	if checked.code != 0 {
		t.Fatalf("warehouse workspace failed validation: %s%s", checked.stdout, checked.stderr)
	}

	data, err := os.ReadFile(filepath.Join(fixtures, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Zones map[string]struct {
			Keys        []string                     `json:"keys"`
			PerRow      map[string]map[string]string `json:"perRow"`
			PerRelation map[string]string            `json:"perRelation"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	if len(expected.Zones) != 3 {
		t.Fatal("expected three independently authored zones")
	}
	for _, zone := range []string{"Z-1", "Z-2", "Z-EMPTY"} {
		t.Run(zone, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "warehouse_count",
				"-pattern", "warehouse_count.receipts", "-environment", "development",
				"-input", "zone_code="+zone, "-format", "json")
			if got.code != 0 || strings.Contains(got.stdout+got.stderr, dsn) {
				t.Fatalf("view failed or leaked DSN: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Relation      string   `json:"relation"`
					KeyAttributes []string `json:"keyAttributes"`
					Coverage      string   `json:"coverage"`
					Rows          []struct {
						Key    string `json:"key"`
						Fields map[string]struct {
							State string `json:"state"`
						} `json:"fields"`
					} `json:"rows"`
					PerRow               map[string]map[string]string `json:"perRow"`
					PerRelation          map[string]string            `json:"perRelation"`
					PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid view JSON: %v (%s)", err, got.stdout)
			}
			if !report.OK || report.Result.Relation != "warehouse.bin.receipts" ||
				report.Result.Coverage != "partial" ||
				!reflect.DeepEqual(report.Result.KeyAttributes, []string{"warehouse.receipt.lot_code", "warehouse.receipt.receipt_seq"}) {
				t.Fatalf("wrong Relation, key or scope: %+v", report.Result)
			}
			keys := []string{}
			for _, row := range report.Result.Rows {
				keys = append(keys, row.Key)
				if row.Fields["warehouse.receipt.units"].State != "present" {
					t.Fatalf("receipt %s has no observed units", row.Key)
				}
			}
			want := expected.Zones[zone]
			if !reflect.DeepEqual(keys, want.Keys) || len(report.Result.PerRow) != len(want.PerRow) {
				t.Fatalf("zone=%s keys=%v want=%v decisions=%v", zone, keys, want.Keys, report.Result.PerRow)
			}
			for key, answer := range want.PerRow {
				if !reflect.DeepEqual(report.Result.PerRow[key], answer) {
					t.Fatalf("zone=%s row=%s decision=%v want=%v", zone, key, report.Result.PerRow[key], answer)
				}
			}
			if !reflect.DeepEqual(report.Result.PerRelation, want.PerRelation) {
				t.Fatalf("zone=%s relation decision=%v want=%v", zone, report.Result.PerRelation, want.PerRelation)
			}
			const unique = "warehouse.bin.receipts.receipt_seq_unique"
			reason := report.Result.PerRelationUndecided[unique]
			if (want.PerRelation[unique] == "undecided" && !strings.Contains(reason, "relation-partially-read")) ||
				(want.PerRelation[unique] == "not-satisfied" && reason != "") {
				t.Fatalf("zone=%s invalid uniqueness reason=%q", zone, reason)
			}
		})
	}

	wrong := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "warehouse_count",
		"-pattern", "warehouse_count.receipts", "-environment", "development",
		"-input", "order_id=1001", "-format", "json")
	if wrong.code == 0 || !strings.Contains(wrong.stdout, "order_id is not an input of the Contract Query") {
		t.Fatalf("stale order-specific input accepted: code=%d stdout=%s", wrong.code, wrong.stdout)
	}
}
